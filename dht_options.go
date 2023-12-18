// kad-dht

package dep2p

import (
	dhtcfg "github.com/bpfs/dep2p/internal/config"
)

// ModeOpt 描述 dht 应以何种模式运行
type ModeOpt = dhtcfg.ModeOpt

const (
	// ModeAuto 利用通过事件总线发送的 EvtLocalReachabilityChanged 事件根据网络条件在客户端和服务器模式之间动态切换 DHT
	ModeAuto ModeOpt = iota
	// ModeClient DHT 仅作为客户端运行，它无法响应传入的查询
	ModeClient
	// ModeServer 将 DHT 作为服务器运行，它既可以发送查询，也可以响应查询
	ModeServer
	// ModeAutoServer 操作方式与 ModeAuto 相同，但在可达性未知时充当服务器
	ModeAutoServer
)

type Option = dhtcfg.Option

// Mode 配置 DHT 运行的模式（客户端、服务器、自动）。
//
// 默认为自动模式。
func Mode(m ModeOpt) Option {
	return func(c *dhtcfg.Config) error {
		c.Mode = m
		return nil
	}
}
