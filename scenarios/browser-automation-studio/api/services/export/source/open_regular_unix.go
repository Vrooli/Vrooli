//go:build unix

package source

import (
	"os"
	"syscall"
)

func openRecordingFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
}
