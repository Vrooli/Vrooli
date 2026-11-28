//go:build linux

package hostwireless

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"golang.org/x/sys/unix"
)

// Create/open the public artifact parent through pinned no-follow descriptors.
// In particular an elevated publisher cannot be redirected through an operator
// symlink into creating files elsewhere on the host.
func openSignedBundleParent(path string) (*os.Root, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("canonical absolute artifact parent required")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		if fd >= 0 {
			unix.Close(fd)
		}
	}()
	uid, gid, invoking := hostreqkit.InvokingUserIDs()
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		next, e := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if errors.Is(e, unix.ENOENT) {
			if e = unix.Mkdirat(fd, part, 0700); e != nil && !errors.Is(e, unix.EEXIST) {
				return nil, e
			}
			next, e = unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
			if e == nil && invoking && os.Geteuid() == 0 {
				e = unix.Fchown(next, uid, gid)
			}
		}
		if e != nil {
			if next >= 0 {
				unix.Close(next)
			}
			return nil, e
		}
		unix.Close(fd)
		fd = next
	}
	f := os.NewFile(uintptr(fd), path)
	fd = -1
	defer f.Close()
	// /proc/self/fd resolves only the pinned admitted descriptor, never a caller
	// supplied symlink. Root operations thereafter remain descriptor relative.
	return os.OpenRoot("/proc/self/fd/" + strconv.FormatUint(uint64(f.Fd()), 10))
}

func chownSignedBundleFile(f *os.File, canonical string) error {
	fd, err := unix.Dup(int(f.Fd()))
	if err != nil {
		return err
	}
	named := os.NewFile(uintptr(fd), canonical)
	defer named.Close()
	return config.ChownFileToInvokingUser(named)
}
