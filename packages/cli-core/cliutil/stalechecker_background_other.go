//go:build !unix

package cliutil

import (
	"errors"
	"os"
	"os/exec"
)

// Background refresh needs an inheritable advisory lock, which this platform
// does not provide here. A latency-bound command then simply runs the current
// binary; interactive commands still refresh synchronously.
func acquireRefreshLock(string) (*os.File, bool, error) {
	return nil, false, errors.New("background CLI refresh is unsupported on this platform")
}

func detachProcess(*exec.Cmd) {}
