//go:build e2e

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/boottest"
)

// Verify this API entry point serves its own healthy response and terminates.
// boottest owns process setup, isolated storage, deadlines, and diagnostics.
// The boot reads an empty research home of its own, never the operator's,
// and is not the live instance, so it writes nothing (F7).
func TestE2E_BinaryBootsAndServesHealth(t *testing.T) {
	root := t.TempDir()
	research := filepath.Join(root, "research")
	if err := os.MkdirAll(filepath.Join(research, "topics"), 0o755); err != nil {
		t.Fatal(err)
	}
	boottest.Run(t, boottest.Config{Service: "research-desk-api", Env: []string{
		envResearchDir + "=" + research,
		envStateDir + "=" + filepath.Join(root, "state"),
		envCodexDir + "=" + filepath.Join(root, "codex"),
	}})
	if _, err := os.Stat(filepath.Join(root, "state")); !os.IsNotExist(err) {
		t.Fatalf("the boot instance touched its state folder: %v", err)
	}
}
