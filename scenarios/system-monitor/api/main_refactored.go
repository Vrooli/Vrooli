// +build refactored

package main

// This file provides the refactored entry point.
// To use the refactored version, build with:
// go build -tags refactored

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	// Set module path
	os.Setenv("GO111MODULE", "on")
	
	// Execute the refactored server
	cmd := exec.Command("go", "run", "./cmd/server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run refactored server: %v", err)
	}
}