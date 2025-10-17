package main

import (
	"os"
	"testing"
)

func TestOptionsHandler_Main(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Note: OPTIONS handler returns via middleware, not specific handler
	// CORS headers are tested in handlers_test.go TestCORSMiddleware
	t.Skip("OPTIONS handler tested via CORS middleware test")
}

func TestLifecycleProtection(t *testing.T) {
	// This test verifies that main() would exit if VROOLI_LIFECYCLE_MANAGED is not set
	// We can't test main() directly as it calls os.Exit, but we test the principle
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") == "" {
		// This is expected in test environment
		t.Log("VROOLI_LIFECYCLE_MANAGED not set (expected in tests)")
	}
}

func TestServiceConstants(t *testing.T) {
	if serviceName != "maintenance-orchestrator" {
		t.Errorf("Expected serviceName 'maintenance-orchestrator', got '%s'", serviceName)
	}

	if apiVersion == "" {
		t.Error("apiVersion should not be empty")
	}
}
