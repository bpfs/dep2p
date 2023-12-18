// kad-dht

package dep2p

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/bpfs/dep2p/kbucket"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
)

// KeyKadID 包含字符串和二进制形式的 Kademlia 密钥。
type KeyKadID struct {
	Key string
	Kad kbucket.ID
}

// NewKeyKadID 从字符串 Kademlia ID 创建 KeyKadID。
func NewKeyKadID(k string) *KeyKadID {
	return &KeyKadID{
		Key: k,
		Kad: kbucket.ConvertKey(k),
	}
}

// PeerKadID 包含一个 libp2p Peer ID 和一个二进制 Kademlia ID。
type PeerKadID struct {
	Peer peer.ID
	Kad  kbucket.ID
}

// NewPeerKadID 从 libp2p Peer ID 创建 PeerKadID。
func NewPeerKadID(p peer.ID) *PeerKadID {
	return &PeerKadID{
		Peer: p,
		Kad:  kbucket.ConvertPeerID(p),
	}
}

// NewPeerKadIDSlice 从传递的 libp2p Peer ID 片段创建 PeerKadID 片段。
func NewPeerKadIDSlice(p []peer.ID) []*PeerKadID {
	r := make([]*PeerKadID, len(p))
	for i := range p {
		r[i] = NewPeerKadID(p[i])
	}
	return r
}

// OptPeerKadID 返回一个指向 PeerKadID 的指针，如果传递的 Peer ID 是它的默认值，则返回 nil。
func OptPeerKadID(p peer.ID) *PeerKadID {
	if p == "" {
		return nil
	}
	return NewPeerKadID(p)
}

// NewLookupEvent 创建一个 LookupEvent，自动将节点 dep2p Peer ID 转换为 PeerKadID，并将字符串 Kademlia 键转换为 KeyKadID。
func NewLookupEvent(
	node peer.ID,
	id uuid.UUID,
	key string,
	request *LookupUpdateEvent,
	response *LookupUpdateEvent,
	terminate *LookupTerminateEvent,
) *LookupEvent {
	return &LookupEvent{
		Node:      NewPeerKadID(node),
		ID:        id,
		Key:       NewKeyKadID(key),
		Request:   request,
		Response:  response,
		Terminate: terminate,
	}
}

// LookupEvent 为 DHT 查找期间发生的每个显着事件发出。
// LookupEvent 支持 JSON 编组，因为它的所有字段都以递归方式支持。
type LookupEvent struct {
	// Node 是执行查找的节点的 ID。
	Node *PeerKadID
	// ID 是查找实例的唯一标识符。
	ID uuid.UUID
	// Key 是用作查找目标的 Kademlia 密钥。
	Key *KeyKadID
	// Request, 如果不为零，则描述与传出查询请求关联的状态更新事件。
	Request *LookupUpdateEvent
	// Response, 如果不为零，则描述与传出查询响应关联的状态更新事件。
	Response *LookupUpdateEvent
	// Terminate, 如果不为零，则描述终止事件。
	Terminate *LookupTerminateEvent
}

// NewLookupUpdateEvent 创建一个新的查找更新事件，自动将传递的对等 ID 转换为对等 Kad ID。
func NewLookupUpdateEvent(
	cause peer.ID,
	source peer.ID,
	heard []peer.ID,
	waiting []peer.ID,
	queried []peer.ID,
	unreachable []peer.ID,
) *LookupUpdateEvent {
	return &LookupUpdateEvent{
		Cause:       OptPeerKadID(cause),
		Source:      OptPeerKadID(source),
		Heard:       NewPeerKadIDSlice(heard),
		Waiting:     NewPeerKadIDSlice(waiting),
		Queried:     NewPeerKadIDSlice(queried),
		Unreachable: NewPeerKadIDSlice(unreachable),
	}
}

// LookupUpdateEvent 描述查找状态更新事件。
type LookupUpdateEvent struct {
	// Cause 是其响应（或缺乏响应）导致更新事件的对等方。
	// 如果 Cause 为零，则这是查找中的第一个更新事件，由播种引起。
	Cause *PeerKadID
	// Source 是向我们通报此更新中的对等 ID 的对等点（如下）。
	Source *PeerKadID
	// Heard 是一组对等体，其在查找对等体集中的状态被设置为“已听到”。
	Heard []*PeerKadID
	// Waiting 是一组对等体，其在查找对等体集中的状态被设置为“等待”。
	Waiting []*PeerKadID
	// Queried 是一组对等体，其在查找对等体集中的状态被设置为“已查询”。
	Queried []*PeerKadID
	// Unreachable 是一组对等体，其在查找对等体集中的状态被设置为“无法访问”。
	Unreachable []*PeerKadID
}

// LookupTerminateEvent 描述查找终止事件。
type LookupTerminateEvent struct {
	// Reason 是查找终止的原因。
	Reason LookupTerminationReason
}

// NewLookupTerminateEvent 创建一个具有给定原因的新查找终止事件。
func NewLookupTerminateEvent(reason LookupTerminationReason) *LookupTerminateEvent {
	return &LookupTerminateEvent{Reason: reason}
}

// LookupTerminationReason 捕获终止查找的原因。
type LookupTerminationReason int

// MarshalJSON 返回传递的查找终止原因的 JSON 编码。
func (r LookupTerminationReason) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r LookupTerminationReason) String() string {
	switch r {
	case LookupStopped:
		return "stopped"
	case LookupCancelled:
		return "cancelled"
	case LookupStarvation:
		return "starvation"
	case LookupCompleted:
		return "completed"
	}
	panic("unreachable")
}

const (
	// LookupStopped 表示查找被用户的 stopFn 中止。
	LookupStopped LookupTerminationReason = iota
	// LookupCancelled 表示查找被上下文中止。
	LookupCancelled
	// LookupStarvation 表示查找由于缺少未查询的对等点而终止。
	LookupStarvation
	// LookupCompleted 表示查找成功终止，达到 Kademlia 结束条件。
	LookupCompleted
)

type routingLookupKey struct{}

// TODO: LookupEventChannel 复制了 eventChanel 的实现。
// 应重构两者以使用公共事件通道实现。
// 常见的实现需要重新考虑 RegisterForEvents 的签名，因为如果不创建额外的“适配器”通道，则返回类型化通道不能成为多态。 当 Go 引入泛型时，这个问题会更容易处理。
type lookupEventChannel struct {
	mu  sync.Mutex
	ctx context.Context
	ch  chan<- *LookupEvent
}

// waitThenClose 当通道注册时，在 goroutine 中生成。 当上下文被取消时，这会安全地清理通道。
func (e *lookupEventChannel) waitThenClose() {
	<-e.ctx.Done()
	e.mu.Lock()
	close(e.ch)
	// 1. 信号表明我们已经完成了。
	// 2. 释放内存（以防我们最终坚持一段时间）。
	e.ch = nil
	e.mu.Unlock()
}

// send 在事件通道上发送事件，如果传递的上下文或内部上下文过期则中止。
func (e *lookupEventChannel) send(ctx context.Context, ev *LookupEvent) {
	e.mu.Lock()
	// Closed.
	if e.ch == nil {
		e.mu.Unlock()
		return
	}
	// 如果传递的上下文不相关，则等待两者。
	select {
	case e.ch <- ev:
	case <-e.ctx.Done():
	case <-ctx.Done():
	}
	e.mu.Unlock()
}

// RegisterForLookupEvents 使用给定的上下文注册查找事件通道。
// 返回的上下文可以传递给 DHT 查询以接收返回通道上的查找事件。
//
// 当调用者不再对查询事件感兴趣时，必须取消传递的上下文。
func RegisterForLookupEvents(ctx context.Context) (context.Context, <-chan *LookupEvent) {
	ch := make(chan *LookupEvent, LookupEventBufferSize)
	ech := &lookupEventChannel{ch: ch, ctx: ctx}
	go ech.waitThenClose()
	return context.WithValue(ctx, routingLookupKey{}, ech), ch
}

// LookupEventBufferSize 是要缓冲的事件数。
var LookupEventBufferSize = 16

// PublishLookupEvent 将查询事件发布到与给定上下文关联的查询事件通道（如果有）。
func PublishLookupEvent(ctx context.Context, ev *LookupEvent) {
	ich := ctx.Value(routingLookupKey{})
	if ich == nil {
		return
	}

	// 我们 *want* 在这里恐慌。
	ech := ich.(*lookupEventChannel)
	ech.send(ctx, ev)
}
