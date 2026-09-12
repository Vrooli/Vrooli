//go:build e2e

package main

import (
	"runtime"
	"testing"

	"github.com/vrooli/api-core/boottest"
)

// Verify this API entry point serves its own healthy response and terminates.
// boottest owns process setup, isolated storage, deadlines, and diagnostics.
func TestE2E_BinaryBootsAndServesHealth(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Fatal("e2e binary boot test relies on SIGTERM; not portable to Windows")
	}
	boottest.Run(t, boottest.Config{Service: "experience-manager-api"})
}
