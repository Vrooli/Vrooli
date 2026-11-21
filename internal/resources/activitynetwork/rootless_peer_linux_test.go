//go:build linux

package activitynetwork

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

func startRootlessPeerUIDProbe(socketPath, resultPath string) error {
	if !filepath.IsAbs(socketPath) || !filepath.IsAbs(resultPath) {
		return fmt.Errorf("peer identity test paths must be absolute")
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	go func() {
		defer listener.Close()
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		uconn, ok := conn.(*net.UnixConn)
		if !ok {
			return
		}
		raw, err := uconn.SyscallConn()
		if err != nil {
			return
		}
		var cred *syscall.Ucred
		var sockErr error
		if err := raw.Control(func(fd uintptr) {
			cred, sockErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		}); err != nil || sockErr != nil {
			return
		}
		if err := os.WriteFile(resultPath, []byte(fmt.Sprintf("%d %d", cred.Pid, cred.Uid)), 0600); err != nil {
			return
		}
	}()
	return nil
}
