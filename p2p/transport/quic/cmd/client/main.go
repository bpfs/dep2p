package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	ic "github.com/bpfs/dep2p/core/crypto"
	"github.com/bpfs/dep2p/core/peer"
	dep2pquic "github.com/bpfs/dep2p/p2p/transport/quic"
	"github.com/bpfs/dep2p/p2p/transport/quicreuse"
	"github.com/sirupsen/logrus"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/quic-go/quic-go"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s <multiaddr> <peer id>", os.Args[0])
		return
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		logrus.Fatalf(err.Error())
	}
}

func run(raddr string, p string) error {
	peerID, err := peer.Decode(p)
	if err != nil {
		return err
	}
	addr, err := ma.NewMultiaddr(raddr)
	if err != nil {
		return err
	}
	priv, _, err := ic.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		return err
	}

	reuse, err := quicreuse.NewConnManager(quic.StatelessResetKey{}, quic.TokenGeneratorKey{})
	if err != nil {
		return err
	}
	t, err := dep2pquic.NewTransport(priv, reuse, nil, nil, nil)
	if err != nil {
		return err
	}

	logrus.Printf("Dialing %s\n", addr.String())
	conn, err := t.Dial(context.Background(), addr, peerID)
	if err != nil {
		return err
	}
	defer conn.Close()
	str, err := conn.OpenStream(context.Background())
	if err != nil {
		return err
	}
	defer str.Close()
	const msg = "Hello world!"
	logrus.Printf("Sending: %s\n", msg)
	if _, err := str.Write([]byte(msg)); err != nil {
		return err
	}
	if err := str.CloseWrite(); err != nil {
		return err
	}
	data, err := io.ReadAll(str)
	if err != nil {
		return err
	}
	logrus.Printf("Received: %s\n", data)
	return nil
}