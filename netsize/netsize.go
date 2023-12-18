package netsize

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bpfs/dep2p/kbucket"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	ks "github.com/whyrusleeping/go-keyspace"
)

// invalidEstimate 表示我们当前没有缓存有效的估计。
const invalidEstimate int32 = -1

var (
	ErrNotEnoughData   = fmt.Errorf("not enough data")
	ErrWrongNumOfPeers = fmt.Errorf("expected bucket size number of peers")
)

var (
	MaxMeasurementAge        = 2 * time.Hour
	MinMeasurementsThreshold = 5
	MaxMeasurementsThreshold = 150
	keyspaceMaxInt, _        = new(big.Int).SetString(strings.Repeat("1", 256), 2)
	keyspaceMaxFloat         = new(big.Float).SetInt(keyspaceMaxInt)
)

type Estimator struct {
	localID    kbucket.ID
	rt         *kbucket.RoutingTable
	bucketSize int

	measurementsLk sync.RWMutex
	measurements   map[int][]measurement

	netSizeCache int32
}

func NewEstimator(localID peer.ID, rt *kbucket.RoutingTable, bucketSize int) *Estimator {
	// 初始化地图以保存测量观测值
	measurements := map[int][]measurement{}
	for i := 0; i < bucketSize; i++ {
		measurements[i] = []measurement{}
	}

	return &Estimator{
		localID:      kbucket.ConvertPeerID(localID),
		rt:           rt,
		bucketSize:   bucketSize,
		measurements: measurements,
		netSizeCache: invalidEstimate,
	}
}

// NormedDistance 计算给定键的标准化 XOR 距离（从 0 到 1）。
func NormedDistance(p peer.ID, k ks.Key) float64 {
	pKey := ks.XORKeySpace.Key([]byte(p))
	ksDistance := new(big.Float).SetInt(pKey.Distance(k))
	normedDist, _ := new(big.Float).Quo(ksDistance, keyspaceMaxFloat).Float64()
	return normedDist
}

type measurement struct {
	distance  float64
	weight    float64
	timestamp time.Time
}

// Track 跟踪给定密钥的对等点列表，以合并到下一个网络大小估计中。
// key 预计 **不** 位于 kademlia 键空间中，并且对等点预计是与给定键最接近的对等点的排序列表（最接近的第一个）。
// 该函数期望对等点的长度与路由表桶大小相同。 它还剥离旧数据并限制数据点的数量（有利于新数据）。
func (e *Estimator) Track(key string, peers []peer.ID) error {
	e.measurementsLk.Lock()
	defer e.measurementsLk.Unlock()

	// 完整性检查
	if len(peers) != e.bucketSize {
		return ErrWrongNumOfPeers
	}

	logrus.Debug("Tracking peers for key", "key", key)

	now := time.Now()

	// 使缓存无效
	atomic.StoreInt32(&e.netSizeCache, invalidEstimate)

	// 计算节点距离的权重。
	weight := e.calcWeight(key, peers)

	// 将给定密钥映射到 Kademlia 密钥空间（对其进行哈希处理）
	ksKey := ks.XORKeySpace.Key([]byte(key))

	// 测量数据点的最大时间戳
	maxAgeTs := now.Add(-MaxMeasurementAge)

	for i, p := range peers {
		// 构建测量结构
		m := measurement{
			distance:  NormedDistance(p, ksKey),
			weight:    weight,
			timestamp: now,
		}

		measurements := append(e.measurements[i], m)

		// 找到仍在允许时间窗口内的测量的最小索引
		// 所有索引较低的测量值都应该被丢弃，因为它们太旧了
		n := len(measurements)
		idx := sort.Search(n, func(j int) bool {
			return measurements[j].timestamp.After(maxAgeTs)
		})

		// 如果测量值超出允许的时间范围，则将其删除。
		// idx == n - 在允许的时间窗口内没有测量 -> 重置切片
		// idx == 0 - 正常情况下我们只有有效的条目
		// idx != 0 - 混合了有效和过时的条目
		if idx != 0 {
			x := make([]measurement, n-idx)
			copy(x, measurements[idx:])
			measurements = x
		}

		// 如果数据点的数量超过最大阈值，则剥离最旧的测量数据点。
		if len(measurements) > MaxMeasurementsThreshold {
			measurements = measurements[len(measurements)-MaxMeasurementsThreshold:]
		}

		e.measurements[i] = measurements
	}

	return nil
}

// NetworkSize 指示估计器计算当前网络大小估计。
func (e *Estimator) NetworkSize() (int32, error) {

	// 无锁返回缓存计算（快速路径）
	if estimate := atomic.LoadInt32(&e.netSizeCache); estimate != invalidEstimate {
		logrus.Debug("Cached network size estimation", "estimate", estimate)
		return estimate, nil
	}

	e.measurementsLk.Lock()
	defer e.measurementsLk.Unlock()

	// 检查第二次。 这是必要的，因为我们可能不得不等待另一个 goroutine 进行计算。
	// 那么计算刚刚由另一个 goroutine 完成，我们不需要重做。
	if estimate := e.netSizeCache; estimate != invalidEstimate {
		logrus.Debug("Cached network size estimation", "estimate", estimate)
		return estimate, nil
	}

	// 删除过时的数据点
	e.garbageCollect()

	// 初始化切片以进行线性拟合
	xs := make([]float64, e.bucketSize)
	ys := make([]float64, e.bucketSize)
	yerrs := make([]float64, e.bucketSize)

	for i := 0; i < e.bucketSize; i++ {
		observationCount := len(e.measurements[i])

		// 如果我们没有足够的数据来合理计算网络规模，请尽早返回
		if observationCount < MinMeasurementsThreshold {
			return 0, ErrNotEnoughData
		}

		// 计算平均距离
		sumDistances := 0.0
		sumWeights := 0.0
		for _, m := range e.measurements[i] {
			sumDistances += m.weight * m.distance
			sumWeights += m.weight
		}
		distanceAvg := sumDistances / sumWeights

		// 计算标准差
		sumWeightedDiffs := 0.0
		for _, m := range e.measurements[i] {
			diff := m.distance - distanceAvg
			sumWeightedDiffs += m.weight * diff * diff
		}
		variance := sumWeightedDiffs / (float64(observationCount-1) / float64(observationCount) * sumWeights)
		distanceStd := math.Sqrt(variance)

		// 跟踪计算
		xs[i] = float64(i + 1)
		ys[i] = distanceAvg
		yerrs[i] = distanceStd
	}

	// 计算线性回归（假设线穿过原点）
	var x2Sum, xySum float64
	for i, xi := range xs {
		yi := ys[i]
		xySum += yerrs[i] * xi * yi
		x2Sum += yerrs[i] * xi * xi
	}
	slope := xySum / x2Sum

	// 计算最终网络大小
	netSize := int32(1/slope - 1)

	// 缓存网络大小估计
	atomic.StoreInt32(&e.netSizeCache, netSize)

	logrus.Debug("New network size estimation", "estimate", netSize)
	return netSize, nil
}

// 如果数据点落入非满桶， calcWeight 的权重会呈指数级减少。
// 它根据 CPL 和存储桶级别来衡量距离估计。
// Bucket Level: 20 -> 1/2^0 -> weight: 1
// Bucket Level: 17 -> 1/2^3 -> weight: 1/8
// Bucket Level: 10 -> 1/2^10 -> weight: 1/1024
//
// 路由表可能没有完整的存储桶，但我们在这里跟踪理论上适合该存储桶的对等点列表。
// 假设存储桶 3 中只有 13 个对等点，尽管有 20 个的空间。
// 现在，Track 函数获取一个对等点列表 (len 20)，其中所有对等点都落入存储桶 3。
// 这组对等点的权重应该为 1，而不是 1/2^7。
// 我实际上认为这不可能发生，因为在调用 Track 函数之前，对等点就已添加到路由表中。 但有时它们似乎不被添加。
func (e *Estimator) calcWeight(key string, peers []peer.ID) float64 {

	cpl := kbucket.CommonPrefixLen(kbucket.ConvertKey(key), e.localID)
	bucketLevel := e.rt.NPeersForCpl(uint(cpl))

	if bucketLevel < e.bucketSize {
		// 路由表没有完整的存储桶。 检查该存储桶中有多少个对等点
		peerLevel := 0
		for _, p := range peers {
			if cpl == kbucket.CommonPrefixLen(kbucket.ConvertPeerID(p), e.localID) {
				peerLevel += 1
			}
		}

		if peerLevel > bucketLevel {
			return math.Pow(2, float64(peerLevel-e.bucketSize))
		}
	}

	return math.Pow(2, float64(bucketLevel-e.bucketSize))
}

// garbageCollect 从列表中删除超出测量时间窗口的所有测量值。
func (e *Estimator) garbageCollect() {
	logrus.Debug("Running garbage collection")

	// 测量数据点的最大时间戳
	maxAgeTs := time.Now().Add(-MaxMeasurementAge)

	for i := 0; i < e.bucketSize; i++ {

		// 找到仍在允许时间窗口内的测量的最小索引
		// 所有索引较低的测量值都应该被丢弃，因为它们太旧了
		n := len(e.measurements[i])
		idx := sort.Search(n, func(j int) bool {
			return e.measurements[i][j].timestamp.After(maxAgeTs)
		})

		// 如果测量值超出允许的时间范围，则将其删除。
		// idx == n - 在允许的时间窗口内没有测量 -> 重置切片
		// idx == 0 - 正常情况下我们只有有效的条目
		// idx != 0 - 混合了有效和过时的条目
		if idx == n {
			e.measurements[i] = []measurement{}
		} else if idx != 0 {
			e.measurements[i] = e.measurements[i][idx:]
		}
	}
}
