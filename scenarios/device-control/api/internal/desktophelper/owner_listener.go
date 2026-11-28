package desktophelper

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// ListenOwner owns a private socket next to protected owner configuration.
// Only a stale refused socket may be replaced under the exclusive process lock.
func ListenOwner(bootstrapPath string) (*net.UnixListener, func(), error) {
	dir := filepath.Dir(bootstrapPath)
	if err := privateDirectory(dir); err != nil {
		return nil, nil, err
	}
	path := filepath.Join(dir, "desktop-owner.sock")
	unlock, err := lockPrivateFile(path + ".lock")
	if err != nil {
		return nil, nil, err
	}
	fail := func(err error) (*net.UnixListener, func(), error) { unlock(); return nil, nil, err }
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fail(ErrBootstrap)
		}
		conn, err := net.DialTimeout("unix", path, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return fail(ErrBootstrap)
		}
		if !errors.Is(err, syscall.ECONNREFUSED) {
			return fail(err)
		}
		if err = os.Remove(path); err != nil {
			return fail(err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fail(err)
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return fail(err)
	}
	if err = os.Chmod(path, 0600); err != nil {
		listener.Close()
		return fail(err)
	}
	return listener, func() { listener.Close(); unlock() }, nil
}
