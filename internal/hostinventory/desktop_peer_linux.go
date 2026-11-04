//go:build linux

package hostinventory

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

type desktopPeer struct {
	PID int
	UID uint32
}

func desktopPeerForDisplay(ctx context.Context, display uint16) (desktopPeer, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	path := "/tmp/.X11-unix/X" + strconv.Itoa(int(display))
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", path)
	if err != nil {
		return desktopPeer{}, err
	}
	defer conn.Close()
	return desktopPeerForUnixConn(conn)
}

func desktopPeerForUnixConn(conn net.Conn) (desktopPeer, error) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return desktopPeer{}, fmt.Errorf("desktop peer connection is not Unix")
	}
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return desktopPeer{}, err
	}
	var peer desktopPeer
	var controlErr error
	if err := raw.Control(func(fd uintptr) {
		cred, err := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if err != nil {
			controlErr = err
			return
		}
		if cred == nil || cred.Pid <= 0 {
			controlErr = fmt.Errorf("desktop peer credentials are invalid")
			return
		}
		peer = desktopPeer{PID: int(cred.Pid), UID: cred.Uid}
	}); err != nil {
		return desktopPeer{}, err
	}
	if controlErr != nil {
		return desktopPeer{}, controlErr
	}
	return peer, nil
}
