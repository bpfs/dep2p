// dep2p

package dep2p

import (
	"bytes"
	"context"
	"encoding/gob"
	"sync"
	
	"github.com/bpfs/dep2p/kbucket"
	"github.com/bpfs/dep2p/streams"
	"github.com/bpfs/dep2p/utils"

	"github.com/libp2p/go-libp2p/config"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
)

type Options struct {
	Libp2pOpts       []config.Option // Option 是一个 libp2p 配置选项，可以提供给 libp2p 构造函数（`libp2p.New`）。
	DhtOpts          []Option        // Option 是一个 dht 配置选项，可以提供给 dht 构造函数(`dht.New`)
	BootstrapsPeers  []string        // 引导节点地址
	RendezvousString string          // 组名
	AddsBook         []string        // 地址簿
}

// Option 类型是一个函数，它接受一个 DeP2P 指针，并返回一个错误。它用于配置 DeP2P 结构体的选项。
type OptionDeP2P func(*DeP2P) error

type DeP2P struct {
	mu             sync.RWMutex
	ctx            context.Context // ctx 是 dep2p 实例的上下文，用于在整个实例中传递取消信号和其他元数据。
	host           host.Host       // libp2p的主机对象。
	peerDHT        *DeP2PDHT       // 分布式哈希表。
	startUp        bool            // 是否启动标志。
	options        Options         // 存储libp2p 选项
	nodeInfo       *NodeInfo       // 节点信息
	connSupervisor *ConnSupervisor // 连接器
}

// New 创建新的 DeP2P 实例
func NewDeP2P(ctx context.Context, opts ...OptionDeP2P) (*DeP2P, error) {
	// 初始化 DeP2P 结构体
	deP2P := &DeP2P{
		ctx:      ctx,           // 上下文
		startUp:  false,         // 启动状态
		nodeInfo: GetNodeInfo(), // 获取节点信息（主机信息、硬盘信息、cpu信息、网络信息）
	}

	// 调用每个选项函数，并配置 DeP2PHost
	for _, opt := range opts {
		if err := opt(deP2P); err != nil {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			return nil, err
		}
	}

	// 使用配置后的 DeP2PHost 创建 libp2p 节点
	h, err := libp2p.New(deP2P.options.Libp2pOpts...)
	if err != nil {
		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
		return nil, err
	}

	for _, addr := range h.Addrs() {
		logrus.Println("正在监听: ", addr)
	}

	logrus.Info("主机已创建: ", h.ID())

	deP2P.host = h

	// New 使用指定的主机和选项创建一个新的 DHT。 请注意，连接到 DHT 对等端并不一定意味着它也在 DHT 路由表中。
	// 如果路由表有超过“minRTRefreshThreshold”对等点，我们仅当我们成功从它获得查询响应或向我们发送查询时才将对等点视为路由表候选者。
	dhtDeP2P, err := New(ctx, h, deP2P.options.DhtOpts...)
	if err != nil {
		logrus.Errorf("[%s]DHT失败: %v", utils.WhereAmI(), err)
		return nil, err

	}

	logrus.Info("引导DHT")
	// Bootstrap 告诉 DHT 进入满足 DeP2PRouter 接口的引导状态。
	if err := dhtDeP2P.Bootstrap(ctx); err != nil {
		logrus.Errorf("[%s]引导DHT失败: %v", utils.WhereAmI(), err)
		return nil, err
	}

	deP2P.peerDHT = dhtDeP2P

	// 解析引导节点地址
	peerAddrInfos, err := utils.ParseAddrInfo(deP2P.options.AddsBook)
	if err != nil {
		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
		return nil, err
	}

	// 创建新的连接监管器实例
	deP2P.connSupervisor = newConnSupervisor(deP2P.Host(), deP2P.ctx, deP2P.nodeInfo, deP2P.peerDHT, peerAddrInfos)

	// 创建启动信号
	readySignalC := make(chan struct{})
	// 连接已有节点
	deP2P.connSupervisor.startSupervising(readySignalC)
	// 主机启动发现节点
	if err := deP2P.networkDiscoveryNode(); err != nil {
		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
		return nil, err
	}
	// 连接握手协议
	streams.RegisterStreamHandler(deP2P.host, HandshakeProtocol, streams.HandlerWithRW(deP2P.handshakeHandle))
	// 设置deP2P启动状态为true
	deP2P.startUp = true

	return deP2P, nil
}

// 接收方处理握手请求
// func (bp *DeP2P) handshakeHandle(req *streams.RequestMessage, res *streams.ResponseMessage) error {
// 	res.Code = 200 // 响应代码
// 	res.Msg = "成功" // 响应消息

// 	handshakeres := &Handshake{
// 		ModeId:   int(bp.DHT().Mode()),
// 		NodeInfo: bp.nodeInfo,
// 	}
// 	var buf bytes.Buffer
// 	enc := gob.NewEncoder(&buf)
// 	if err := enc.Encode(handshakeres); err != nil {
// 		return err
// 	}

// 	// Mode 允许自省 DHT 的操作模式
// 	res.Data = buf.Bytes()

// 	res.Message.Sender = bp.Host().ID().String() // 发送方ID

// 	handshake := new(Handshake)

// 	slicePayloadBuffer := bytes.NewBuffer(req.Payload)
// 	gobDecoder := gob.NewDecoder(slicePayloadBuffer)
// 	if err := gobDecoder.Decode(handshake); err != nil {
// 		return err
// 	}

// 	table, exists := bp.connSupervisor.routingTables[handshake.ModeId]
// 	pidDecode, err := peer.Decode(req.Message.Sender)
// 	if err != nil {
// 		return err

// 	}

// 	if exists {

// 		// 键存在，可以使用 value
// 		// 更新Kbucket路由表
// 		renewKbucketRoutingTable(table, handshake.ModeId, pidDecode)

// 	} else {
// 		// 键不存在
// 		table, err := NewTable(bp.connSupervisor.host)
// 		if err != nil {
// 			logrus.Error("新建k桶失败", err)
// 		}
// 		// 更新Kbucket路由表
// 		renewKbucketRoutingTable(table, handshake.ModeId, pidDecode)
// 		//
// 		bp.connSupervisor.routingTables[handshake.ModeId] = table

// 	}

//		return nil
//	}
func (bp *DeP2P) handshakeHandle(req *streams.RequestMessage, res *streams.ResponseMessage) (int32, string) {
	// 检查请求消息是否为空
	if req == nil || req.Message == nil {
		logrus.Errorf("[%s]:请求消息为空", utils.WhereAmI())
		return 400, "请求消息为空"
	}

	// 检查DHT实例是否已初始化
	if bp.DHT() == nil {
		logrus.Errorf("[%s]:DHT未初始化", utils.WhereAmI())
		return 500, "DHT未初始化"
	}

	// 检查节点信息是否已初始化
	if bp.nodeInfo == nil {
		logrus.Errorf("[%s]:节点信息未初始化", utils.WhereAmI())
		return 500, "节点信息未初始化"
	}

	// 假设默认响应码为成功
	res.Code = 200
	res.Msg = "成功"

	// 构建握手响应
	handshakeres := &Handshake{
		ModeId:   int(bp.DHT().Mode()),
		NodeInfo: bp.nodeInfo,
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(handshakeres); err != nil {
		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
		// 编码失败，返回错误响应码和消息
		return 500, "握手响应编码失败"
	}

	// 设置响应数据
	res.Data = buf.Bytes()

	// 检查Host实例是否已初始化
	if bp.Host() == nil {
		logrus.Errorf("[%s]:Host未初始化", utils.WhereAmI())
		return 500, "Host未初始化"
	}
	if res.Message == nil {
		res.Message = &streams.Message{}
	}
	res.Message.Sender = bp.Host().ID().String() // 设置发送方ID

	handshake := new(Handshake)
	slicePayloadBuffer := bytes.NewBuffer(req.Payload)
	gobDecoder := gob.NewDecoder(slicePayloadBuffer)
	if err := gobDecoder.Decode(handshake); err != nil {
		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
		// 请求解码失败
		return 400, "握手请求解码失败"
	}

	// 检查connSupervisor是否已初始化
	if bp.connSupervisor == nil || bp.connSupervisor.routingTables == nil {
		logrus.Errorf("[%s]:连接监督器未初始化", utils.WhereAmI())
		return 500, "连接监督器未初始化"
	}

	// 处理握手逻辑
	table, exists := bp.connSupervisor.routingTables[handshake.ModeId]
	pidDecode, err := peer.Decode(req.Message.Sender)
	if err != nil {
		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
		// Peer ID解码失败
		return 400, "Peer ID解码失败"
	}

	if exists {
		// 更新Kbucket路由表
		renewKbucketRoutingTable(table, handshake.ModeId, pidDecode)
	} else {
		// 新建Kbucket路由表
		table, err := NewTable(bp.connSupervisor.host)
		if err != nil {
			logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
			// 新建路由表失败
			return 500, "新建路由表失败"
		}
		renewKbucketRoutingTable(table, handshake.ModeId, pidDecode)
		bp.connSupervisor.routingTables[handshake.ModeId] = table
	}

	// 如果执行到这里，说明处理成功
	return 200, "握手成功"
}

// newConnSupervisor 创建一个新的 ConnSupervisor 实例
func newConnSupervisor(host host.Host, ctx context.Context, nodeInfo *NodeInfo, peerDHT *DeP2PDHT, peerAddrInfos []peer.AddrInfo) *ConnSupervisor {
	return &ConnSupervisor{
		host:          host,
		peerDHT:       peerDHT,
		nodeInfo:      nodeInfo,
		routingTables: make(map[int]*kbucket.RoutingTable),
		ctx:           ctx,
		peerAddrInfos: peerAddrInfos,
		startUp:       false,
		IsConnected:   false,
		actuators:     make(map[peer.ID]*tryToDialActuator),
	}
}

// Context 返回上下文
func (bp *DeP2P) Context() context.Context {
	return bp.ctx
}

// Host 返回 Host
func (bp *DeP2P) Host() host.Host {
	return bp.host
}

// RoutingTables 返回 RoutingTables
func (bp *DeP2P) RoutingTables() map[int]*kbucket.RoutingTable {
	return bp.connSupervisor.routingTables
}

// RoutingTable 根据 dht 类型获取指定类型 RoutingTable
func (bp *DeP2P) RoutingTable(mode int) *kbucket.RoutingTable {
	bp.mu.RLock()
	routingTable, exists := bp.connSupervisor.routingTables[mode]
	bp.mu.RUnlock()

	if exists {
		// 键存在，可以使用 value
		return routingTable
	}

	bp.mu.Lock()
	defer bp.mu.Unlock()

	// 仔细检查它是否在此期间创建
	routingTable, exists = bp.connSupervisor.routingTables[mode]
	if exists {
		return routingTable
	}

	// 键不存在
	table, err := NewTable(bp.host)
	if err != nil {
		logrus.Errorf("[%s]: %v", utils.WhereAmI(), err)
		return nil
	}
	bp.connSupervisor.routingTables[mode] = table
	return table
}

// NodeInfo 返回 节点信息
func (bp *DeP2P) NodeInfo() *NodeInfo {
	return bp.nodeInfo
}

// DHT 返回分布式哈希表
func (bp *DeP2P) DHT() *DeP2PDHT {
	return bp.peerDHT
}

// DHT 返回分布式哈希表
func (bp *DeP2P) ConnectPeer() []peer.AddrInfo {
	return bp.connSupervisor.ConnectedAddrInfo()
}

// IsRunning 当 Libp2p 已启动时返回true；否则返回false。
func (bp *DeP2P) IsRunning() bool {
	// bp.lock.RLock()
	// defer bp.lock.RUnlock()
	return bp.startUp
}

// Start 方法用于启动 Libp2p 主机。
func (bp *DeP2P) Start() error {
	return nil
}

// Stop 方法用于停止 Libp2p 主机。
func (bp *DeP2P) Stop() error {
	return bp.host.Close()
}

// Options 方法获取DeP2P 配置项
func (bp *DeP2P) Options() Options {
	return bp.options
}

// WithRendezvousString 设置本地网络监听地址
func WithRendezvousString(rendezvousString string) OptionDeP2P {

	return func(bp *DeP2P) error {
		bp.options.RendezvousString = rendezvousString
		return nil
	}
}

// WithLibp2pOpts 设置本地网络监听地址
func WithLibp2pOpts(Option []config.Option) OptionDeP2P {

	return func(bp *DeP2P) error {
		bp.options.Libp2pOpts = Option

		return nil
	}
}

// WithDhtOpts 设置本地网络监听地址
func WithDhtOpts(option []Option) OptionDeP2P {
	// baseOpts := []Option{}

	// baseOpts = append(baseOpts, Mode(m))

	return func(bp *DeP2P) error {
		bp.options.DhtOpts = option
		return nil
	}
}

// WithBootstrapsPeers 设置引导节点
func WithBootstrapsPeers(bootstrapsPeers []string) OptionDeP2P {

	return func(bp *DeP2P) error {
		bp.options.BootstrapsPeers = bootstrapsPeers
		return nil
	}
}
