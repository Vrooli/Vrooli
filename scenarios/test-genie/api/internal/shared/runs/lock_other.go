//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || windows)

package runs

import (
	"errors"
	"os"
)

func lockFile(_ *os.File) (func(), error) {
	return nil, errors.New("run-store locking is unsupported on this operating system")
}
