// main.go - Entry point wrapper that delegates to cmd/server
// This file exists to satisfy v2.0 contract requirements for api/main.go
// The actual implementation is in api/cmd/server/main.go
// Test coverage: See api/cmd/server/*_test.go (110+ comprehensive tests)

package main

import (
	"github.com/vrooli/api-core/preflight"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "network-tools",
	}) {
		return // Process was re-exec'd after rebuild
	}

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
