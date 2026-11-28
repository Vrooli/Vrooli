//go:build !linux

package platform

import "net"

func peerProcessID(net.Conn) (int, error) { return 0, ErrPeerProcessIdentityUnavailable }
