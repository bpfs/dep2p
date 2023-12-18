package dep2p

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"

	"github.com/multiformats/go-multiaddr"
)

// DefaultBootstrapPeers 是 libp2p 提供的一组公共 DHT 引导节点。
var DefaultBootstrapPeers []multiaddr.Multiaddr

// Minimum 路由表中的对等点数量。 如果我们跌破这个值并且看到一个新的对等点，我们就会触发一轮引导程序。
var minRTRefreshThreshold = 10

const (
	periodicBootstrapInterval = 2 * time.Minute
	maxNBoostrappers          = 2
)

func init() {
	for _, s := range []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/118.31.66.14/tcp/4001/p2p/QmQ3m69P9rwZerNSSHk8RvZTPiXg5EKyppfrrQHDbKEGeD",
	} {
		ma, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			panic(err)
		}
		DefaultBootstrapPeers = append(DefaultBootstrapPeers, ma)
	}
}

// GetDefaultBootstrapPeerAddrInfos 返回默认引导对等点的peer.AddrInfos，因此我们可以通过将它们传递给 BootstrapPeers(...) 选项来使用它们来初始化 DHT。
func GetDefaultBootstrapPeerAddrInfos() []peer.AddrInfo {
	ds := make([]peer.AddrInfo, 0, len(DefaultBootstrapPeers))

	for i := range DefaultBootstrapPeers {
		info, err := peer.AddrInfoFromP2pAddr(DefaultBootstrapPeers[i])
		if err != nil {
			logrus.Error("无法将引导程序地址转换为对等地址信息、地址", DefaultBootstrapPeers[i].String(), err, "err")
			continue
		}
		ds = append(ds, *info)
	}
	return ds
}

// Bootstrap 告诉 DHT 进入满足 DeP2PRouter 接口的引导状态。
func (dht *DeP2PDHT) Bootstrap(ctx context.Context) error {
	dht.fixRTIfNeeded()
	dht.rtRefreshManager.RefreshNoWait()
	return nil
}

// RefreshRoutingTable 告诉 DHT 刷新它的路由表。
//
// 返回的通道将阻塞直到刷新完成，然后产生错误并关闭。 该通道已缓冲并且可以安全地忽略。
func (dht *DeP2PDHT) RefreshRoutingTable() <-chan error {
	return dht.rtRefreshManager.Refresh(false)
}

// ForceRefresh 行为类似于 RefreshRoutingTable，但强制 DHT 刷新路由表中的所有存储桶，无论上次刷新时间如何。
//
// 返回的通道将阻塞直到刷新完成，然后产生错误并关闭。 该通道已缓冲并且可以安全地忽略。
func (dht *DeP2PDHT) ForceRefresh() <-chan error {
	return dht.rtRefreshManager.Refresh(true)
}
