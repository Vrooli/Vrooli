package resources

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
)

func publishComposeCandidate(stage, destination string) error {
	return unix.Renameat2(unix.AT_FDCWD, stage, unix.AT_FDCWD, destination, unix.RENAME_NOREPLACE)
}
func composeCandidatePlatformSupported() bool { return true }

// Open each path component relative to a held directory descriptor. A source
// replacement cannot redirect this read through a symlink or block on a FIFO.
func openCandidateInput(root, relative string) (*os.File, error) {
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(filepath.Clean(filepath.FromSlash(relative)), string(filepath.Separator))
	for i, part := range parts {
		flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC | unix.O_NONBLOCK
		if i < len(parts)-1 {
			flags |= unix.O_DIRECTORY
		}
		next, openErr := unix.Openat(fd, part, flags, 0)
		unix.Close(fd)
		if openErr != nil {
			return nil, openErr
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), relative), nil
}
