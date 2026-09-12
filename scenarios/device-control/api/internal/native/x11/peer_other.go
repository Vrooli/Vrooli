//go:build !linux

package x11

import "net"

func serverPeer(net.Conn) (Peer, error) { return Peer{}, ErrUnavailable }
