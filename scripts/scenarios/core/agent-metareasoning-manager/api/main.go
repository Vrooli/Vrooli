// Package main - Compatibility wrapper for simplified API
// This file ensures compatibility with existing service.json build commands
// that expect to build from the root directory

package main

import (
	"os"
	"os/exec"
)

func main() {
	// Simply execute the simplified version
	cmd := exec.Command("go", "run", "cmd/server/main_simplified.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}