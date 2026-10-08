//go:build linux

package cleanup

import (
	"errors"
	"io"
	"os"
	"syscall"
)

func writeChaosChunk(dst io.Writer, buf []byte, offset int64) (int, error) {
	file, ok := dst.(*os.File)
	if !ok {
		return dst.Write(buf)
	}
	if err := syscall.Fallocate(int(file.Fd()), 0, offset, int64(len(buf))); err == nil {
		return len(buf), nil
	} else if !errors.Is(err, syscall.EOPNOTSUPP) && !errors.Is(err, syscall.ENOSYS) {
		return 0, err
	}
	return dst.Write(buf)
}
