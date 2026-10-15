//go:build linux

package finiteinstallation

import (
	"github.com/vrooli/api-core/effortauthority"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func readProtected(root, name string, owner uint32) ([]byte, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || filepath.Base(name) != name {
		return nil, effortauthority.ErrRefused
	}
	fd, e := unix.Open("/", unix.O_DIRECTORY|unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, effortauthority.ErrRefused
	}
	defer func() { unix.Close(fd) }()
	parts := strings.Split(strings.TrimPrefix(root, "/"), "/")
	for _, part := range parts {
		next, e := unix.Openat(fd, part, unix.O_DIRECTORY|unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if e != nil {
			return nil, effortauthority.ErrRefused
		}
		unix.Close(fd)
		fd = next
		var st unix.Stat_t
		if unix.Fstat(fd, &st) != nil || st.Uid != owner || st.Mode&0022 != 0 {
			return nil, effortauthority.ErrRefused
		}
	}
	return readFromTrustedAnchor(fd, name, owner)
}

// Private helper requires an already validated anchor; tests supply only a disposable anchor.
func readFromTrustedAnchor(fd int, name string, owner uint32) ([]byte, error) {
	if filepath.Base(name) != name {
		return nil, effortauthority.ErrRefused
	}
	file, e := unix.Openat(fd, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, effortauthority.ErrRefused
	}
	f := os.NewFile(uintptr(file), name)
	defer f.Close()
	var st unix.Stat_t
	if unix.Fstat(file, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != owner || (st.Mode&0777 != 0644 && st.Mode&0777 != 0444) || st.Nlink != 1 || st.Size <= 0 || st.Size > 262144 {
		return nil, effortauthority.ErrRefused
	}
	raw, e := io.ReadAll(io.LimitReader(f, 262145))
	if e != nil || len(raw) > 262144 {
		return nil, effortauthority.ErrRefused
	}
	return raw, nil
}
