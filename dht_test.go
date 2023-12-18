// kad-dht

package dep2p

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"github.com/multiformats/go-multistream"
	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	u "github.com/ipfs/boxo/util"
	"github.com/ipfs/go-cid"
	detectrace "github.com/ipfs/go-detect-race"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
)

var testCaseCids []cid.Cid

func init() {
	// 初始化测试用例的 Cid 列表
	for i := 0; i < 100; i++ {
		v := fmt.Sprintf("%d -- value", i)

		var newCid cid.Cid
		switch i % 3 {
		case 0:
			// 使用 cid.NewCidV0 创建一个 Cid
			mhv := u.Hash([]byte(v))
			newCid = cid.NewCidV0(mhv)
		case 1:
			// 使用 cid.NewCidV1 创建一个 Cid
			mhv := u.Hash([]byte(v))
			newCid = cid.NewCidV1(cid.DagCBOR, mhv)
		case 2:
			// 使用 cid.NewCidV1 创建一个使用原始编码的 Cid
			rawMh := make([]byte, 12)
			binary.PutUvarint(rawMh, cid.Raw)
			binary.PutUvarint(rawMh[1:], 10)
			copy(rawMh[2:], []byte(v)[:10])
			_, mhv, err := multihash.MHFromBytes(rawMh)
			if err != nil {
				panic(err)
			}
			newCid = cid.NewCidV1(cid.Raw, mhv)
		}
		testCaseCids = append(testCaseCids, newCid)
	}
}

// var testPrefix = ProtocolPrefix("/test") // testPrefix是一个变量，表示协议的前缀为 "/test"。
// setupDHT 函数用于设置 DHT，并返回一个 DeP2PDHT 对象。
// 它接受一个上下文对象 ctx、一个测试对象 t、一个布尔值 client，以及一系列选项参数 options。
func setupDHT(ctx context.Context, t *testing.T, client bool, options ...Option) *DeP2PDHT {
	// 设置基本选项
	baseOpts := []Option{
		// testPrefix, // 使用测试前缀
		// NamespacedValidator("v", blankValidator{}), // 使用空白验证器
		// DisableAutoRefresh(),                       // 禁用自动刷新
	}

	// 根据 client 参数设置不同的模式选项
	if client {
		baseOpts = append(baseOpts, Mode(ModeClient)) // 客户端模式
	} else {
		baseOpts = append(baseOpts, Mode(ModeServer)) // 服务器模式
	}

	// 创建基于 Swarm 的主机对象
	host, err := bhost.NewHost(swarmt.GenSwarm(t, swarmt.OptDisableReuseport), new(bhost.HostOpts))
	require.NoError(t, err)
	host.Start()
	t.Cleanup(func() { host.Close() })

	// 创建 DeP2PDHT 对象
	d, err := New(ctx, host, append(baseOpts, options...)...)
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })

	// 返回创建的 DeP2PDHT 对象
	return d
}

// setupDHTS 函数用于设置多个 DHT，并返回一个包含这些 DHT 对象的切片。
// 它接受一个测试对象 t、一个上下文对象 ctx、一个整数 n，表示要设置的 DHT 的数量，以及一系列选项参数 options。
func setupDHTS(t *testing.T, ctx context.Context, n int, options ...Option) []*DeP2PDHT {
	// 创建用于存储地址、DHT 对象和对等节点的切片
	addrs := make([]ma.Multiaddr, n)
	dhts := make([]*DeP2PDHT, n)
	peers := make([]peer.ID, n)

	// 创建用于检查地址和对等节点是否重复的映射
	sanityAddrsMap := make(map[string]struct{})
	sanityPeersMap := make(map[string]struct{})

	// 设置 n 个 DHT
	for i := 0; i < n; i++ {
		// 调用 setupDHT 函数设置单个 DHT
		dhts[i] = setupDHT(ctx, t, false, options...)
		peers[i] = dhts[i].PeerID()
		addrs[i] = dhts[i].host.Addrs()[0]

		// 检查地址是否重复
		if _, lol := sanityAddrsMap[addrs[i].String()]; lol {
			t.Fatal("While setting up DHTs address got duplicated.")
		} else {
			sanityAddrsMap[addrs[i].String()] = struct{}{}
		}

		// 检查对等节点是否重复
		if _, lol := sanityPeersMap[peers[i].String()]; lol {
			t.Fatal("While setting up DHTs peerid got duplicated.")
		} else {
			sanityPeersMap[peers[i].String()] = struct{}{}
		}
	}

	// 返回设置的 DHT 对象切片
	return dhts
}

// connectNoSync 函数用于在两个 DHT 节点之间建立连接，但不进行同步。
// 它接受一个测试对象 t、一个上下文对象 ctx，以及两个 DeP2PDHT 对象 a 和 b。
func connectNoSync(t *testing.T, ctx context.Context, a, b *DeP2PDHT) {
	t.Helper()

	// 获取节点 b 的 ID 和地址
	idB := b.self
	addrB := b.peerstore.Addrs(idB)

	// 检查是否设置了本地地址
	if len(addrB) == 0 {
		t.Fatal("peers setup incorrectly: no local address")
	}

	// 使用节点 a 的主机对象连接到节点 b
	if err := a.host.Connect(ctx, peer.AddrInfo{ID: idB, Addrs: addrB}); err != nil {
		t.Fatal(err)
	}
}

// wait 函数用于等待两个 DHT 节点之间的连接。
// 它接受一个测试对象 t、一个上下文对象 ctx，以及两个 DeP2PDHT 对象 a 和 b。
func wait(t *testing.T, ctx context.Context, a, b *DeP2PDHT) {
	t.Helper()

	// 循环直到收到连接通知。
	// 在高负载情况下，这可能不会像我们希望的那样立即发生。
	for a.routingTable.Find(b.self) == "" {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(time.Millisecond * 5):
		}
	}
} // connect 函数用于在两个 DHT 节点之间建立连接，并等待连接完成。
// 它接受一个测试对象 t、一个上下文对象 ctx，以及两个 DeP2PDHT 对象 a 和 b。
func connect(t *testing.T, ctx context.Context, a, b *DeP2PDHT) {
	t.Helper()

	// 调用 connectNoSync 函数建立连接
	connectNoSync(t, ctx, a, b)

	// 等待连接完成
	wait(t, ctx, a, b)
	wait(t, ctx, b, a)
}

// bootstrap 函数用于引导一组 DHT 节点。
// 它接受一个测试对象 t、一个上下文对象 ctx，以及一个 DeP2PDHT 对象数组 dhts。
func bootstrap(t *testing.T, ctx context.Context, dhts []*DeP2PDHT) {
	// 创建一个新的上下文对象，并在函数结束时取消该上下文
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logrus.Debugf("refreshing DHTs routing tables...")

	// 随机选择一个起始节点，以减少偏差
	start := rand.Intn(len(dhts))

	// 循环刷新每个 DHT 节点的路由表
	for i := range dhts {
		// 获取当前要刷新的 DHT 节点
		dht := dhts[(start+i)%len(dhts)]

		select {
		case err := <-dht.RefreshRoutingTable():
			// 如果刷新过程中出现错误，则打印错误日志
			if err != nil {
				t.Error(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// TestRefreshMultiple 函数用于测试刷新多个 DHT 节点的路由表。
// 它创建了一个上下文对象和一组 DHT 节点，然后在其中的一个节点上多次调用 RefreshRoutingTable 函数。
// 最后，它检查每次调用是否都会返回，并在超时时打印错误日志。
func TestRefreshMultiple(t *testing.T) {
	// 创建一个具有超时时间的上下文对象
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	// 设置5个 DHT 节点
	dhts := setupDHTS(t, ctx, 5)
	defer func() {
		for _, dht := range dhts {
			dht.Close()
			defer dht.host.Close()
		}
	}()

	// 将除第一个节点外的其他节点与第一个节点建立连接
	for _, dht := range dhts[1:] {
		connect(t, ctx, dhts[0], dht)
	}

	// 在第一个节点上多次调用 RefreshRoutingTable 函数
	a := dhts[0].RefreshRoutingTable()
	time.Sleep(time.Nanosecond)
	b := dhts[0].RefreshRoutingTable()
	time.Sleep(time.Nanosecond)
	c := dhts[0].RefreshRoutingTable()

	// 确保每次调用最终都会返回
	select {
	case <-a:
	case <-ctx.Done():
		t.Fatal("第一个通道没有发出信号")
	}
	select {
	case <-b:
	case <-ctx.Done():
		t.Fatal("第二个通道没有发出信号")
	}
	select {
	case <-c:
	case <-ctx.Done():
		t.Fatal("第三个通道没有发出信号")
	}
}

// TestContextShutDown 函数用于测试关闭 DHT 时的上下文状态。
// 在函数中，首先创建一个上下文对象，并在函数结束时取消上下文。
// 然后，创建一个 DHT 节点。
// 接下来，检查上下文是否处于活动状态，期望上下文未完成。
// 关闭 DHT 节点。
// 最后，再次检查上下文状态，期望上下文已完成。
func TestContextShutDown(t *testing.T) {
	t.Skip("This test is flaky, see https://github.com/libp2p/go-libp2p-kad-dht/issues/724.")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建一个 DHT 节点
	dht := setupDHT(ctx, t, false)

	// 上下文应该处于活动状态
	select {
	case <-dht.Context().Done():
		t.Fatal("上下文不应该已完成")
	default:
	}

	// 关闭 DHT 节点
	require.NoError(t, dht.Close())

	// 现在上下文应该已完成
	select {
	case <-dht.Context().Done():
	default:
		t.Fatal("上下文应该已完成")
	}
}

// 如果 minPeers 或 avgPeers 为 0，则不进行测试。
func waitForWellFormedTables(t *testing.T, dhts []*DeP2PDHT, minPeers, avgPeers int, timeout time.Duration) {
	// 测试“良好形成的”路由表（每个路由表中有 >= minPeers 个对等节点）
	t.Helper()

	timeoutA := time.After(timeout)
	for {
		select {
		case <-timeoutA:
			t.Errorf("在 %s 后未能达到良好形成的路由表", timeout)
			return
		case <-time.After(5 * time.Millisecond):
			if checkForWellFormedTablesOnce(t, dhts, minPeers, avgPeers) {
				// 成功
				return
			}
		}
	}
}

// checkForWellFormedTablesOnce 函数用于检查路由表是否良好形成。
// 它接受一个测试对象 `t`，一个 `DeP2PDHT` 节点数组 `dhts`，最小对等节点数 `minPeers` 和平均对等节点数 `avgPeers`。
// 函数首先计算总对等节点数 `totalPeers`，然后遍历每个 `dht` 节点。
// 对于每个节点，获取其路由表的大小 `rtlen`，并将其加到 `totalPeers` 中。
// 如果 `minPeers` 大于 0 且 `rtlen` 小于 `minPeers`，则表示路由表不满足最小对等节点数的要求，返回 false。
// 计算实际的平均对等节点数 `actualAvgPeers`，如果 `avgPeers` 大于 0 且 `actualAvgPeers` 小于 `avgPeers`，则返回 false。
// 如果路由表满足要求，返回 true。

func checkForWellFormedTablesOnce(t *testing.T, dhts []*DeP2PDHT, minPeers, avgPeers int) bool {
	t.Helper()
	totalPeers := 0
	for _, dht := range dhts {
		rtlen := dht.routingTable.Size()
		totalPeers += rtlen
		if minPeers > 0 && rtlen < minPeers {
			//t.Logf("路由表 %s 只有 %d 个对等节点（应该有 >%d）", dht.self, rtlen, minPeers)
			return false
		}
	}
	actualAvgPeers := totalPeers / len(dhts)
	t.Logf("平均路由表大小：%d", actualAvgPeers)
	if avgPeers > 0 && actualAvgPeers < avgPeers {
		t.Logf("平均路由表大小：%d < %d", actualAvgPeers, avgPeers)
		return false
	}
	return true
}

// printRoutingTables 函数用于打印路由表。
// 它接受一个 `DeP2PDHT` 节点数组 `dhts`。
// 函数遍历每个 `dht` 节点，并打印其路由表。

func printRoutingTables(dhts []*DeP2PDHT) {
	// 现在应该填满了路由表，让我们来检查它们。
	fmt.Printf("检查 %d 个节点的路由表\n", len(dhts))
	for _, dht := range dhts {
		fmt.Printf("检查 %s 的路由表\n", dht.self)
		dht.routingTable.Print()
		fmt.Println("")
	}
}

// TestRefresh 函数用于测试刷新（refresh）操作。
// 如果测试运行时间较短，则跳过测试。
// 创建一个包含 30 个 DHT 节点的环境，并在测试结束时关闭这些节点。
// 打印连接了多少个 DHT 节点。
// 将这些 DHT 节点按顺序连接成环。
// 等待 100 毫秒。
// 多次进行引导操作，直到路由表形成良好。
// 如果一次引导操作后，检查路由表是否满足最小对等节点数为 7 且平均对等节点数为 10，则跳出循环。
// 每次引导操作之间暂停 50 微秒。
// 如果启用了调试模式，则打印路由表。
func TestRefresh(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nDHTs := 30
	dhts := setupDHTS(t, ctx, nDHTs)
	defer func() {
		for i := 0; i < nDHTs; i++ {
			dhts[i].Close()
			defer dhts[i].host.Close()
		}
	}()

	t.Logf("连接 %d 个 DHT 节点成环", nDHTs)
	for i := 0; i < nDHTs; i++ {
		connect(t, ctx, dhts[i], dhts[(i+1)%len(dhts)])
	}

	<-time.After(100 * time.Millisecond)
	t.Logf("引导它们，使它们相互发现 %d 次", nDHTs)

	for {
		bootstrap(t, ctx, dhts)

		if checkForWellFormedTablesOnce(t, dhts, 7, 10) {
			break
		}

		time.Sleep(time.Microsecond * 50)
	}

	if u.Debug {
		// 现在应该填满了路由表，让我们来检查它们。
		printRoutingTables(dhts)
	}
}

// TestRefreshBelowMinRTThreshold 函数用于测试当路由表中的对等节点数低于最小对等节点数阈值时的刷新（refresh）操作。
// 创建一个包含 3 个 DHT 节点的环境，并在测试结束时关闭这些节点。
// 将节点 A 设置为自动引导模式，并等待它与节点 B 和节点 C 建立连接。
// 执行一次路由表刷新操作，并等待一轮完成，确保节点 A 与节点 B 和节点 C 都连接成功。
// 创建两个新的 DHT 节点 D 和 E，并将它们相互连接。
// 将节点 D 连接到节点 A，触发节点 A 的最小对等节点数阈值，从而进行引导操作。
// 由于默认的引导扫描间隔为 30 分钟 - 1 小时，我们可以确定如果进行了引导操作，那是因为最小对等节点数阈值被触发了。
// 等待引导操作完成，并确保节点 A 的路由表中有 4 个对等节点，包括节点 E。
// 等待 100 毫秒。
// 断言节点 A 的路由表中存在节点 E。
func TestRefreshBelowMinRTThreshold(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建一个新的主机
	host, err := bhost.NewHost(swarmt.GenSwarm(t, swarmt.OptDisableReuseport), new(bhost.HostOpts))
	require.NoError(t, err)
	host.Start()

	// 在节点 A 上启用自动引导
	dhtA, err := New(
		ctx,
		host,
		// testPrefix,
		Mode(ModeServer),
		// NamespacedValidator("v", blankValidator{}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// 创建节点 B 和节点 C
	dhtB := setupDHT(ctx, t, false)
	dhtC := setupDHT(ctx, t, false)

	defer func() {
		// 关闭节点 A、B 和节点 C
		dhtA.Close()
		dhtA.host.Close()

		dhtB.Close()
		dhtB.host.Close()

		dhtC.Close()
		dhtC.host.Close()
	}()

	// 将节点 A 与节点 B 和节点 C 连接
	connect(t, ctx, dhtA, dhtB)
	connect(t, ctx, dhtB, dhtC)

	// 对节点 A 执行一次路由表刷新操作，并等待一轮完成，确保节点 A 与节点 B 和节点 C 都连接成功
	dhtA.RefreshRoutingTable()
	waitForWellFormedTables(t, []*DeP2PDHT{dhtA}, 2, 2, 20*time.Second)

	// 创建两个新的节点 D 和节点 E
	dhtD := setupDHT(ctx, t, false)
	dhtE := setupDHT(ctx, t, false)

	// 将节点 D 和节点 E 相互连接
	connect(t, ctx, dhtD, dhtE)
	defer func() {
		// 关闭节点 D 和节点 E
		dhtD.Close()
		dhtD.host.Close()

		dhtE.Close()
		dhtE.host.Close()
	}()

	// 将节点 D 连接到节点 A，触发节点 A 的最小对等节点数阈值，从而进行引导操作
	connect(t, ctx, dhtA, dhtD)

	// 由于上述引导操作，节点 A 也会发现节点 E
	waitForWellFormedTables(t, []*DeP2PDHT{dhtA}, 4, 4, 10*time.Second)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, dhtE.self, dhtA.routingTable.Find(dhtE.self), "A 的路由表应包含节点 E！")
}

// TestPeriodicRefresh 用于测试定期刷新操作。
func TestPeriodicRefresh(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	if runtime.GOOS == "windows" {
		t.Skip("由于 #760，跳过测试")
	}
	if detectrace.WithRace() {
		t.Skip("由于竞争检测器的最大 goroutine 数量，跳过测试")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nDHTs := 30
	dhts := setupDHTS(t, ctx, nDHTs)
	defer func() {
		for i := 0; i < nDHTs; i++ {
			dhts[i].Close()
			defer dhts[i].host.Close()
		}
	}()

	t.Logf("DHTs 未连接。%d", nDHTs)
	for _, dht := range dhts {
		rtlen := dht.routingTable.Size()
		if rtlen > 0 {
			t.Errorf("对于 %s，路由表应该有 0 个对等节点，但实际有 %d 个", dht.self, rtlen)
		}
	}

	for i := 0; i < nDHTs; i++ {
		connect(t, ctx, dhts[i], dhts[(i+1)%len(dhts)])
	}

	t.Logf("DHTs 现在连接到其他 1-2 个节点。%d", nDHTs)
	for _, dht := range dhts {
		rtlen := dht.routingTable.Size()
		if rtlen > 2 {
			t.Errorf("对于 %s，路由表最多应该有 2 个对等节点，但实际有 %d 个", dht.self, rtlen)
		}
	}

	if u.Debug {
		printRoutingTables(dhts)
	}

	t.Logf("引导它们以便它们互相发现。%d", nDHTs)
	var wg sync.WaitGroup
	for _, dht := range dhts {
		wg.Add(1)
		go func(d *DeP2PDHT) {
			<-d.RefreshRoutingTable()
			wg.Done()
		}(dht)
	}

	wg.Wait()
	// 这是异步的，我们不知道它何时完成一个周期，因此一直检查
	// 直到路由表看起来更好，或者超过长时间的超时时间，表示失败。
	waitForWellFormedTables(t, dhts, 7, 10, 20*time.Second)

	if u.Debug {
		printRoutingTables(dhts)
	}
}

// TestProvidesMany 用于测试提供者功能。
func TestProvidesMany(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("由于 #760，跳过测试")
	}
	if detectrace.WithRace() {
		t.Skip("由于竞争检测器的最大 goroutine 数量，跳过测试")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nDHTs := 40
	dhts := setupDHTS(t, ctx, nDHTs)
	defer func() {
		for i := 0; i < nDHTs; i++ {
			dhts[i].Close()
			defer dhts[i].host.Close()
		}
	}()

	t.Logf("连接 %d 个 DHT 节点形成环", nDHTs)
	for i := 0; i < nDHTs; i++ {
		connect(t, ctx, dhts[i], dhts[(i+1)%len(dhts)])
	}

	<-time.After(100 * time.Millisecond)
	t.Logf("引导它们以便它们互相发现。%d", nDHTs)
	ctxT, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	bootstrap(t, ctxT, dhts)

	if u.Debug {
		// 路由表现在应该已经填满了。让我们检查一下。
		t.Logf("检查 %d 的路由表", nDHTs)
		for _, dht := range dhts {
			fmt.Printf("检查 %s 的路由表\n", dht.self)
			dht.routingTable.Print()
			fmt.Println("")
		}
	}

	providers := make(map[cid.Cid]peer.ID)

	d := 0
	for _, c := range testCaseCids {
		d = (d + 1) % len(dhts)
		dht := dhts[d]
		providers[c] = dht.self

		t.Logf("为 %s 声明提供者", c)
		if err := dht.Provide(ctx, c, true); err != nil {
			t.Fatal(err)
		}
	}

	// 这个超时是为了什么？之前是 60ms。
	time.Sleep(time.Millisecond * 6)

	errchan := make(chan error)

	ctxT, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	getProvider := func(dht *DeP2PDHT, k cid.Cid) {
		defer wg.Done()

		expected := providers[k]

		provchan := dht.FindProvidersAsync(ctxT, k, 1)
		select {
		case prov := <-provchan:
			actual := prov.ID
			if actual == "" {
				errchan <- fmt.Errorf("获取到了空的提供者 (%s at %s)", k, dht.self)
			} else if actual != expected {
				errchan <- fmt.Errorf("获取到了错误的提供者 (%s != %s) (%s at %s)",
					expected, actual, k, dht.self)
			}
		case <-ctxT.Done():
			errchan <- fmt.Errorf("未获取到提供者 (%s at %s)", k, dht.self)
		}
	}

	for _, c := range testCaseCids {
		// 每个节点都应该能够找到它...
		for _, dht := range dhts {
			logrus.Debugf("获取 %s 的提供者：%s", c, dht.self)
			wg.Add(1)
			go getProvider(dht, c)
		}
	}

	// 我们需要这个是因为要打印错误
	go func() {
		wg.Wait()
		close(errchan)
	}()

	for err := range errchan {
		t.Error(err)
	}
}

// TestProvidesAsync 用于测试异步提供者功能。
func TestProvidesAsync(t *testing.T) {
	// t.Skip("跳过测试以调试其他问题")
	if testing.Short() {
		t.SkipNow()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dhts := setupDHTS(t, ctx, 4)
	defer func() {
		for i := 0; i < 4; i++ {
			dhts[i].Close()
			defer dhts[i].host.Close()
		}
	}()

	connect(t, ctx, dhts[0], dhts[1])
	connect(t, ctx, dhts[1], dhts[2])
	connect(t, ctx, dhts[1], dhts[3])

	err := dhts[3].Provide(ctx, testCaseCids[0], true)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 60)

	ctxT, cancel := context.WithTimeout(ctx, time.Millisecond*300)
	defer cancel()
	provs := dhts[0].FindProvidersAsync(ctxT, testCaseCids[0], 5)
	select {
	case p, ok := <-provs:
		if !ok {
			t.Fatal("提供者通道已关闭...")
		}
		if p.ID == "" {
			t.Fatal("获取到了空的提供者！")
		}
		if p.ID != dhts[3].self {
			t.Fatalf("获取到了提供者，但不是正确的提供者。%s", p)
		}
	case <-ctxT.Done():
		t.Fatal("未获取到提供者")
	}
}

// TestConnectCollision 用于测试连接冲突的情况。
func TestConnectCollision(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	runTimes := 10

	for rtime := 0; rtime < runTimes; rtime++ {
		logrus.Info("Running Time: ", rtime)

		ctx, cancel := context.WithCancel(context.Background())

		// 设置 DHT A 和 DHT B
		dhtA := setupDHT(ctx, t, false)
		dhtB := setupDHT(ctx, t, false)

		addrA := dhtA.peerstore.Addrs(dhtA.self)[0]
		addrB := dhtB.peerstore.Addrs(dhtB.self)[0]

		peerA := dhtA.self
		peerB := dhtB.self

		errs := make(chan error)
		go func() {
			// 将 DHT B 的地址添加到 DHT A 的 peerstore 中
			dhtA.peerstore.AddAddr(peerB, addrB, peerstore.TempAddrTTL)
			pi := peer.AddrInfo{ID: peerB}
			err := dhtA.host.Connect(ctx, pi)
			errs <- err
		}()
		go func() {
			// 将 DHT A 的地址添加到 DHT B 的 peerstore 中
			dhtB.peerstore.AddAddr(peerA, addrA, peerstore.TempAddrTTL)
			pi := peer.AddrInfo{ID: peerA}
			err := dhtB.host.Connect(ctx, pi)
			errs <- err
		}()

		timeout := time.After(5 * time.Second)
		select {
		case e := <-errs:
			if e != nil {
				t.Fatal(e)
			}
		case <-timeout:
			t.Fatal("Timeout received!")
		}
		select {
		case e := <-errs:
			if e != nil {
				t.Fatal(e)
			}
		case <-timeout:
			t.Fatal("Timeout received!")
		}

		// 关闭 DHT A 和 DHT B
		dhtA.Close()
		dhtB.Close()
		dhtA.host.Close()
		dhtB.host.Close()
		cancel()
	}
}

// TestFindClosestPeers 函数的用途是测试查找最接近对等节点的功能。
func TestFindClosestPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nDHTs := 30
	dhts := setupDHTS(t, ctx, nDHTs)
	defer func() {
		for i := 0; i < nDHTs; i++ {
			dhts[i].Close()
			defer dhts[i].host.Close()
		}
	}()

	t.Logf("连接 %d 个 DHT 节点成环", nDHTs)
	for i := 0; i < nDHTs; i++ {
		connect(t, ctx, dhts[i], dhts[(i+1)%len(dhts)])
	}

	querier := dhts[1]
	peers, err := querier.GetClosestPeers(ctx, "foo")
	if err != nil {
		t.Fatal(err)
	}

	if len(peers) < querier.beta {
		t.Fatalf("获取到错误数量的对等节点（获取到 %d 个，期望至少 %d 个）", len(peers), querier.beta)
	}
}

// TestFixLowPeers 函数的用途是测试修复低对等节点的功能。
func TestFixLowPeers(t *testing.T) {
	ctx := context.Background()

	dhts := setupDHTS(t, ctx, minRTRefreshThreshold+5)

	defer func() {
		for _, d := range dhts {
			d.Close()
			d.Host().Close()
		}
	}()

	mainD := dhts[0]

	// 将主节点与其他节点连接起来
	for _, d := range dhts[1:] {
		mainD.peerstore.AddAddrs(d.self, d.peerstore.Addrs(d.self), peerstore.TempAddrTTL)
		require.NoError(t, mainD.Host().Connect(ctx, peer.AddrInfo{ID: d.self}))
	}

	waitForWellFormedTables(t, []*DeP2PDHT{mainD}, minRTRefreshThreshold, minRTRefreshThreshold+4, 5*time.Second)

	// 对所有节点运行一次路由表刷新
	for _, d := range dhts {
		err := <-d.RefreshRoutingTable()
		require.NoError(t, err)
	}

	// 现在从路由表中删除对等节点，以触发达到阈值的情况
	for _, d := range dhts[3:] {
		mainD.routingTable.RemovePeer(d.self)
	}

	// 由于修复低对等节点的功能，我们仍然会得到足够的对等节点
	waitForWellFormedTables(t, []*DeP2PDHT{mainD}, minRTRefreshThreshold, minRTRefreshThreshold, 5*time.Second)
}

// TestPing 该方法的用途是是测试Ping操作是否成功。
func TestPing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 在测试之前设置DHT节点
	ds := setupDHTS(t, ctx, 2)

	// 将ds[1]的地址添加到ds[0]的Peerstore中
	ds[0].Host().Peerstore().AddAddrs(ds[1].PeerID(), ds[1].Host().Addrs(), peerstore.AddressTTL)

	// 执行Ping操作并断言没有错误发生
	assert.NoError(t, ds[0].Ping(context.Background(), ds[1].PeerID()))
}

// TestClientModeAtInit 该方法的用途是测试在初始化时设置客户端模式是否正确。
func TestClientModeAtInit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置pinger和client两个DHT节点
	pinger := setupDHT(ctx, t, false)
	client := setupDHT(ctx, t, true)

	// 将client的地址添加到pinger的Peerstore中
	pinger.Host().Peerstore().AddAddrs(client.PeerID(), client.Host().Addrs(), peerstore.AddressTTL)

	// 执行Ping操作并断言错误类型为multistream.ErrNotSupported[protocol.ID]
	err := pinger.Ping(context.Background(), client.PeerID())
	assert.True(t, errors.Is(err, multistream.ErrNotSupported[protocol.ID]{}))
}

// TestModeChange 是一个测试函数，用于测试模式切换功能。
func TestModeChange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置 clientOnly 和 clientToServer 的 DHT 实例
	clientOnly := setupDHT(ctx, t, true)
	clientToServer := setupDHT(ctx, t, true)

	// 将 clientToServer 的地址添加到 clientOnly 的 Peerstore 中
	clientOnly.Host().Peerstore().AddAddrs(clientToServer.PeerID(), clientToServer.Host().Addrs(), peerstore.AddressTTL)

	// 使用 clientOnly 发送 Ping 请求到 clientToServer
	err := clientOnly.Ping(ctx, clientToServer.PeerID())
	assert.True(t, errors.Is(err, multistream.ErrNotSupported[protocol.ID]{}))

	// 将 clientToServer 的模式设置为 modeServer
	err = clientToServer.setMode(modeServer)
	assert.Nil(t, err)

	// 使用 clientOnly 再次发送 Ping 请求到 clientToServer
	err = clientOnly.Ping(ctx, clientToServer.PeerID())
	assert.Nil(t, err)

	// 将 clientToServer 的模式设置为 modeClient
	err = clientToServer.setMode(modeClient)
	assert.Nil(t, err)

	// 使用 clientOnly 再次发送 Ping 请求到 clientToServer
	err = clientOnly.Ping(ctx, clientToServer.PeerID())
	assert.NotNil(t, err)
}

// TestDynamicModeSwitching 是一个测试函数，用于测试动态模式切换功能。
func TestDynamicModeSwitching(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置 prober 和 node 的 DHT 实例
	prober := setupDHT(ctx, t, true)               // 我们的测试工具
	node := setupDHT(ctx, t, true, Mode(ModeAuto)) // 待测试的节点

	// 将 node 的地址添加到 prober 的 Peerstore 中
	prober.Host().Peerstore().AddAddrs(node.PeerID(), node.Host().Addrs(), peerstore.AddressTTL)

	// 使用 prober 的 Host 进行拨号连接到 node
	if _, err := prober.Host().Network().DialPeer(ctx, node.PeerID()); err != nil {
		t.Fatal(err)
	}

	// 创建一个事件发射器，用于监听本地可达性变化事件
	emitter, err := node.host.EventBus().Emitter(new(event.EvtLocalReachabilityChanged))
	if err != nil {
		t.Fatal(err)
	}

	// 断言 DHT 客户端模式的状态
	assertDHTClient := func() {
		err = prober.Ping(ctx, node.PeerID())
		assert.True(t, errors.Is(err, multistream.ErrNotSupported[protocol.ID]{}))
		if l := len(prober.RoutingTable().ListPeers()); l != 0 {
			t.Errorf("期望路由表长度为 0，实际为 %d", l)
		}
	}

	// 断言 DHT 服务器模式的状态
	assertDHTServer := func() {
		err = prober.Ping(ctx, node.PeerID())
		assert.Nil(t, err)
		// 由于 prober 在节点更新协议时会调用 fixLowPeers，所以节点应该出现在 prober 的路由表中
		if l := len(prober.RoutingTable().ListPeers()); l != 1 {
			t.Errorf("期望路由表长度为 1，实际为 %d", l)
		}
	}

	// 发送本地可达性变化事件，将可达性设置为私网
	err = emitter.Emit(event.EvtLocalReachabilityChanged{Reachability: network.ReachabilityPrivate})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)

	assertDHTClient()

	// 发送本地可达性变化事件，将可达性设置为公网
	err = emitter.Emit(event.EvtLocalReachabilityChanged{Reachability: network.ReachabilityPublic})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)

	assertDHTServer()

	// 发送本地可达性变化事件，将可达性设置为未知
	err = emitter.Emit(event.EvtLocalReachabilityChanged{Reachability: network.ReachabilityUnknown})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)

	assertDHTClient()
}
