// main.go - Entry point wrapper that delegates to cmd/server
// This file exists to satisfy v2.0 contract requirements for api/main.go
// The actual implementation is in api/cmd/server/main.go

package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Get the directory of the current executable
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exeDir := filepath.Dir(exePath)

	// Delegate to the actual server binary
	serverPath := filepath.Join(exeDir, "network-tools-api")
	cmd := exec.Command(serverPath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
