//go:build linux

package lightingbinding

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

// Listen creates only a current-user-owned private socket, with no TCP route.
func Listen(directory string, identity Identity) (net.Listener, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	if identity.Scenario != "lighting-control" || identity.Variant != "live" || identity.PID != os.Getpid() || !filepath.IsAbs(directory) {
		return nil, errors.New("native lighting requires its own exact default scenario identity and absolute directory")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 || !ok || stat.Uid != uint32(os.Geteuid()) {
		return nil, errors.New("native lighting socket directory must be owned exclusively by the current user")
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: SocketPath(directory, identity), Net: "unix"})
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(listener.Addr().String(), 0600); err != nil {
		_ = listener.Close()
		return nil, err
	}
	return listener, nil
}
