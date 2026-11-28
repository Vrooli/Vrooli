//go:build linux

package platform

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

func replaceProcessImage(file *os.File, argv, env []string) error {
	var magic [4]byte
	if _, err := file.ReadAt(magic[:], 0); err != nil {
		return err
	}
	if string(magic[:]) != "\x7fELF" {
		return fmt.Errorf("platform: verified image execution requires a native ELF binary")
	}
	return syscall.Exec("/proc/self/fd/"+strconv.FormatUint(uint64(file.Fd()), 10), argv, env)
}
