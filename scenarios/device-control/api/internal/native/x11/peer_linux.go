//go:build linux

package x11

import (
	"golang.org/x/sys/unix"
	"net"
)

func serverPeer(conn net.Conn) (Peer, error) {
	socket, ok := conn.(*net.UnixConn)
	if !ok {
		return Peer{}, ErrUnavailable
	}
	raw, err := socket.SyscallConn()
	if err != nil {
		return Peer{}, err
	}
	var peer *unix.Ucred
	var peerErr error
	err = raw.Control(func(fd uintptr) { peer, peerErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED) })
	if err != nil || peerErr != nil || peer == nil || peer.Pid <= 0 {
		return Peer{}, ErrUnavailable
	}
	return Peer{PID: int(peer.Pid), UID: peer.Uid}, nil
}
