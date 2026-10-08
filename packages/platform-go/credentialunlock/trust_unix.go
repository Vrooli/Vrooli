//go:build !windows

package credentialunlock

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// trustedSocket proves the socket and its directory belong to this user and
// that nobody else can replace it. Without this, another local account could
// plant a socket at the path and harvest nothing — but could answer with a
// passphrase of its choosing; refusing keeps the answer's origin certain.
func trustedSocket(socket string) error {
	self := uint32(os.Getuid()) // #nosec G115 -- a POSIX uid fits uint32.
	info, err := os.Lstat(socket)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%s is not a socket", socket)
	}
	if owner, ok := info.Sys().(*syscall.Stat_t); !ok || owner.Uid != self {
		return fmt.Errorf("%s is not owned by this user", socket)
	}
	dir, err := os.Stat(filepath.Dir(socket))
	if err != nil {
		return err
	}
	if owner, ok := dir.Sys().(*syscall.Stat_t); !ok || owner.Uid != self {
		return fmt.Errorf("%s is not owned by this user", filepath.Dir(socket))
	}
	if dir.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("%s is writable by other users", filepath.Dir(socket))
	}
	return nil
}
