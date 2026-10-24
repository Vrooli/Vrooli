//go:build !windows

package main

import (
	"os/exec"
	"strings"
	"syscall"
)

func execProcess(argv0 string, argv []string, env []string) error {
	if !strings.ContainsRune(argv0, '/') {
		resolved, err := exec.LookPath(argv0)
		if err != nil {
			return err
		}
		argv0 = resolved
	}
	return syscall.Exec(argv0, argv, env)
}
