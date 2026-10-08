//go:build e2e

package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/boottest"
)

// Verify this API entry point serves its own healthy response and terminates.
// boottest owns process setup, isolated storage, deadlines, and diagnostics.
func TestE2E_BinaryBootsAndServesHealth(t *testing.T) {
	bootstrap, err := filepath.Abs(filepath.Join("..", "bootstrap", "bootstrap.sh"))
	if err != nil {
		t.Fatalf("resolve test bootstrap script: %v", err)
	}
	boottest.Run(t, boottest.Config{Service: "vrooli-bridge-api", StartupTimeout: 30 * time.Second, Env: []string{"BRIDGE_BOOTSTRAP_SCRIPT=" + bootstrap}})
}
