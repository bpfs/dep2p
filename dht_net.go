// kad-dht

package dep2p

import (
	"io"
	"time"

	"github.com/bpfs/dep2p/internal/net"
	"github.com/bpfs/dep2p/metrics"

	pb "github.com/bpfs/dep2p/pb"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-msgio"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var dhtStreamIdleTimeout = 1 * time.Minute

// handleNewStream 实现network.StreamHandler
func (dht *DeP2PDHT) handleNewStream(s network.Stream) {
	if dht.handleNewMessage(s) {
		// 如果我们没有错误地退出，请优雅地关闭。
		_ = s.Close()
	} else {
		// 否则，发送错误。
		_ = s.Reset()
	}
}

// Returns true 表示有序完成写入（因此我们可以关闭流）。
func (dht *DeP2PDHT) handleNewMessage(s network.Stream) bool {
	ctx := dht.ctx
	r := msgio.NewVarintReaderSize(s, network.MessageSizeMax)

	mPeer := s.Conn().RemotePeer()

	timer := time.AfterFunc(dhtStreamIdleTimeout, func() { _ = s.Reset() })
	defer timer.Stop()

	for {
		if dht.getMode() != modeServer {
			logrus.Debugln("ignoring incoming dht message while not in server mode")
			return false
		}

		var req pb.Message
		msgbytes, err := r.ReadMsg()
		msgLen := len(msgbytes)
		if err != nil {
			r.ReleaseMsg(msgbytes)
			if err == io.EOF {
				return true
			}

			if msgLen > 0 {
				_ = stats.RecordWithTags(ctx,
					[]tag.Mutator{tag.Upsert(metrics.KeyMessageType, "UNKNOWN")},
					metrics.ReceivedMessages.M(1),
					metrics.ReceivedMessageErrors.M(1),
					metrics.ReceivedBytes.M(int64(msgLen)),
				)
			}
			return false
		}
		err = req.Unmarshal(msgbytes)
		r.ReleaseMsg(msgbytes)
		if err != nil {
			_ = stats.RecordWithTags(ctx,
				[]tag.Mutator{tag.Upsert(metrics.KeyMessageType, "UNKNOWN")},
				metrics.ReceivedMessages.M(1),
				metrics.ReceivedMessageErrors.M(1),
				metrics.ReceivedBytes.M(int64(msgLen)),
			)
			return false
		}

		timer.Reset(dhtStreamIdleTimeout)

		startTime := time.Now()
		ctx, _ := tag.New(ctx,
			tag.Upsert(metrics.KeyMessageType, req.GetType().String()),
		)

		stats.Record(ctx,
			metrics.ReceivedMessages.M(1),
			metrics.ReceivedBytes.M(int64(msgLen)),
		)

		handler := dht.handlerForMsgType(req.GetType())
		if handler == nil {
			stats.Record(ctx, metrics.ReceivedMessageErrors.M(1))
			return false
		}

		resp, err := handler(ctx, mPeer, &req)
		if err != nil {
			stats.Record(ctx, metrics.ReceivedMessageErrors.M(1))
			return false
		}

		if resp == nil {
			continue
		}

		// 发送响应消息
		err = net.WriteMsg(s, resp)
		if err != nil {
			stats.Record(ctx, metrics.ReceivedMessageErrors.M(1))
			return false
		}

		elapsedTime := time.Since(startTime)

		latencyMillis := float64(elapsedTime) / float64(time.Millisecond)
		stats.Record(ctx, metrics.InboundRequestLatency.M(latencyMillis))
	}
}
