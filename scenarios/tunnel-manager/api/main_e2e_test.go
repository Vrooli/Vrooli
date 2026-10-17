//go:build e2e

package main

import (
	"testing"

	"github.com/vrooli/api-core/boottest"
)

// Verify this API entry point serves its own healthy response and terminates.
// boottest owns process setup, isolated storage, deadlines, and diagnostics.
func TestE2E_BinaryBootsAndServesHealth(t *testing.T) {
	boottest.Run(t, boottest.Config{Service: "tunnel-manager-api"})
}
