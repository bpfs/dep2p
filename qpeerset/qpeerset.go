package qpeerset

import (
	"math/big"
	"sort"

	"github.com/libp2p/go-libp2p/core/peer"
	ks "github.com/whyrusleeping/go-keyspace"
)

// PeerState 描述单个查找的生命周期中对等 ID 的状态。
type PeerState int

const (
	// PeerHeard 应用于尚未被查询的对等点。
	PeerHeard PeerState = iota
	// PeerWaiting 应用于当前正在查询的对等点。
	PeerWaiting
	// PeerQueried 应用于已被查询并成功检索到响应的对等点。
	PeerQueried
	// PeerUnreachable 应用于已被查询但未成功检索到响应的对等点。
	PeerUnreachable
)

// QueryPeerset 维护 Kademlia 异步查找的状态。
// 查找状态是一组对等体，每个对等体都标有一个对等体状态。
type QueryPeerset struct {
	key ks.Key

	all []queryPeerState

	sorted bool
}

type queryPeerState struct {
	id         peer.ID
	distance   *big.Int
	state      PeerState
	referredBy peer.ID
}

type sortedQueryPeerset QueryPeerset

func (sqp *sortedQueryPeerset) Len() int {
	return len(sqp.all)
}

func (sqp *sortedQueryPeerset) Swap(i, j int) {
	sqp.all[i], sqp.all[j] = sqp.all[j], sqp.all[i]
}

func (sqp *sortedQueryPeerset) Less(i, j int) bool {
	di, dj := sqp.all[i].distance, sqp.all[j].distance
	return di.Cmp(dj) == -1
}

// NewQueryPeerset 创建一个新的空对等点集。
// key 是该对等集所要查找的目标键。
func NewQueryPeerset(key string) *QueryPeerset {
	return &QueryPeerset{
		key:    ks.XORKeySpace.Key([]byte(key)),
		all:    []queryPeerState{},
		sorted: false,
	}
}

func (qp *QueryPeerset) find(p peer.ID) int {
	for i := range qp.all {
		if qp.all[i].id == p {
			return i
		}
	}
	return -1
}

func (qp *QueryPeerset) distanceToKey(p peer.ID) *big.Int {
	return ks.XORKeySpace.Key([]byte(p)).Distance(qp.key)
}

// TryAdd 将对等点 p 添加到对等点集中。
// 如果对等点已经存在，则不采取任何操作。
// 否则，添加对等点并将状态设置为 PeerHeard。
// 如果对等点尚未存在，则 TryAdd 返回 true。
func (qp *QueryPeerset) TryAdd(p, referredBy peer.ID) bool {
	if qp.find(p) >= 0 {
		return false
	} else {
		qp.all = append(qp.all,
			queryPeerState{id: p, distance: qp.distanceToKey(p), state: PeerHeard, referredBy: referredBy})
		qp.sorted = false
		return true
	}
}

func (qp *QueryPeerset) sort() {
	if qp.sorted {
		return
	}
	sort.Sort((*sortedQueryPeerset)(qp))
	qp.sorted = true
}

// SetState 将对等点 p 的状态设置为 s。
// 如果 p 不在对等集中，SetState 会发生恐慌。
func (qp *QueryPeerset) SetState(p peer.ID, s PeerState) {
	qp.all[qp.find(p)].state = s
}

// GetState 返回对等点 p 的状态。
// 如果 p 不在对等集中，则 GetState 会出现恐慌。
func (qp *QueryPeerset) GetState(p peer.ID) PeerState {
	return qp.all[qp.find(p)].state
}

// GetReferrer 返回将我们引向对等点 p 的对等点。
// 如果 p 不在对等集中，则 GetReferrer 会出现恐慌。
func (qp *QueryPeerset) GetReferrer(p peer.ID) peer.ID {
	return qp.all[qp.find(p)].referredBy
}

// GetClosestNInStates 返回最接近处于给定状态之一的关键对等点。
// 如果满足条件的对等点较少，则返回 n 个或更少的对等点。
// 返回的对等点按照与键的距离升序排列。
func (qp *QueryPeerset) GetClosestNInStates(n int, states ...PeerState) (result []peer.ID) {
	qp.sort()
	m := make(map[PeerState]struct{}, len(states))
	for i := range states {
		m[states[i]] = struct{}{}
	}

	for _, p := range qp.all {
		if _, ok := m[p.state]; ok {
			result = append(result, p.id)
		}
	}
	if len(result) >= n {
		return result[:n]
	}
	return result
}

// GetClosestInStates 返回处于给定状态之一的对等点。
// 返回的对等点按照与键的距离升序排列。
func (qp *QueryPeerset) GetClosestInStates(states ...PeerState) (result []peer.ID) {
	return qp.GetClosestNInStates(len(qp.all), states...)
}

// NumHeard 返回处于 PeerHeard 状态的对等点数量。
func (qp *QueryPeerset) NumHeard() int {
	return len(qp.GetClosestInStates(PeerHeard))
}

// NumWaiting 返回处于 PeerWaiting 状态的对等点数量。
func (qp *QueryPeerset) NumWaiting() int {
	return len(qp.GetClosestInStates(PeerWaiting))
}
