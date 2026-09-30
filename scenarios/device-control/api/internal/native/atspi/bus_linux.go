//go:build linux

package atspi

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/godbus/dbus/v5"
	"golang.org/x/sys/unix"
)

// Dial connects only to an explicitly provisioned local bus path. The guard
// binds its kernel peer to trusted bootstrap/session policy before any auth
// bytes are sent. The returned private connection follows ctx's lifetime.
func Dial(ctx context.Context, path string, guard func(context.Context, uint32, uint32) error) (*dbus.Conn, error) {
	if guard == nil || !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 107 {
		return nil, ErrRefused
	}
	handshake, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	wire, err := (&net.Dialer{}).DialContext(handshake, "unix", path)
	if err != nil {
		return nil, ErrRefused
	}
	ok := false
	defer func() {
		if !ok {
			wire.Close()
		}
	}()
	stop := context.AfterFunc(handshake, func() { wire.Close() })
	defer stop()
	deadline, _ := handshake.Deadline()
	if wire.SetDeadline(deadline) != nil {
		return nil, ErrRefused
	}
	socket, valid := wire.(*net.UnixConn)
	if !valid {
		return nil, ErrRefused
	}
	raw, err := socket.SyscallConn()
	if err != nil {
		return nil, ErrRefused
	}
	var peer *unix.Ucred
	var peerErr error
	err = raw.Control(func(fd uintptr) { peer, peerErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED) })
	if err != nil || peerErr != nil || peer == nil || peer.Pid <= 0 || peer.Uid != uint32(os.Getuid()) || guard(handshake, uint32(peer.Pid), peer.Uid) != nil {
		return nil, ErrRefused
	}
	conn, err := dbus.NewConn(wire, dbus.WithContext(ctx))
	if err != nil {
		return nil, ErrRefused
	}
	if conn.Auth([]dbus.Auth{dbus.AuthExternal(strconv.Itoa(os.Getuid()))}) != nil || conn.Hello() != nil || handshake.Err() != nil || wire.SetDeadline(time.Time{}) != nil {
		conn.Close()
		return nil, ErrRefused
	}
	if !stop() {
		conn.Close()
		return nil, ErrRefused
	}
	ok = true
	return conn, nil
}
