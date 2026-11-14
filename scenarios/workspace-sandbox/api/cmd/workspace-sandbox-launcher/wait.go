package main

import (
	"os"
	"os/exec"
)

func runAndWait(argv0 string, argv []string, env []string) error {
	args := []string(nil)
	if len(argv) > 1 {
		args = argv[1:]
	}
	cmd := exec.Command(argv0, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
