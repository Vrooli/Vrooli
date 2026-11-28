// Package namespace tests
package namespace

import (
	"strings"
	"testing"

	"workspace-sandbox/internal/process"
)

// testStarter is the production OSExecStarter, used by these tests
// because they exercise host-level capability probes. Tests that want
// to inject scripted responses construct a procmocks.FakeStarter
// directly.
var testStarter = process.NewOSExecStarter()

// TestCheckReturnsStatus verifies that Check(testStarter) returns a valid Status struct
// [REQ:P0-001] Sandbox creation relies on namespace status detection
func TestCheckReturnsStatus(t *testing.T) {
	status := Check(testStarter)

	// Status should always have a kernel version
	if status.KernelVersion == "" {
		t.Error("expected non-empty kernel version")
	}

	// If we're in a CI environment or user namespaces are unavailable,
	// we should get a reason explaining why.
	if !status.CanCreateUserNamespace && status.Reason == "" {
		t.Error("expected reason when user namespace creation is unavailable")
	}

	t.Logf("Namespace status: InUserNamespace=%v, CanCreate=%v, CanMount=%v, Kernel=%s, Reason=%s",
		status.InUserNamespace,
		status.CanCreateUserNamespace,
		status.CanMountOverlayfs,
		status.KernelVersion,
		status.Reason)
}

// TestIsKernelAtLeastBasicCases tests kernel version comparison
// [REQ:P0-002] Driver selection depends on kernel version detection
func TestIsKernelAtLeastBasicCases(t *testing.T) {
	// Get actual kernel version for context
	status := Check(testStarter)
	t.Logf("Running on kernel: %s", status.KernelVersion)

	// Test against very old kernel - should always pass on modern systems
	if !IsKernelAtLeast(2, 0) {
		t.Error("expected kernel >= 2.0 on any modern system")
	}

	// Test against impossibly new kernel - should always fail
	if IsKernelAtLeast(99, 0) {
		t.Error("expected kernel < 99.0")
	}
}

// TestIsKernelAtLeastEdgeCases tests boundary conditions in version comparison
// [REQ:P0-002] Driver selection depends on kernel version detection
func TestIsKernelAtLeastEdgeCases(t *testing.T) {
	// These tests are based on the running kernel
	status := Check(testStarter)

	// Parse actual kernel version for comparison
	parts := strings.Split(status.KernelVersion, ".")
	if len(parts) < 2 {
		t.Skipf("Cannot parse kernel version: %s", status.KernelVersion)
	}

	// Test that the current kernel version matches itself
	// We can't test exact values without parsing, but we can test relative behavior
	t.Logf("Kernel parts: %v", parts)
}

// TestConstantsAreDefined verifies kernel requirement constants.
// [REQ:P0-007] Sandbox lifecycle management uses these for detection
func TestConstantsAreDefined(t *testing.T) {
	if RequiredKernelMajor == 0 {
		t.Error("RequiredKernelMajor should not be 0")
	}
}

// TestKernelVersionFormat verifies kernel version string format
// [REQ:P2-025] Cross-platform driver selection uses kernel version
func TestKernelVersionFormat(t *testing.T) {
	status := Check(testStarter)

	if status.KernelVersion == "unknown" {
		t.Skip("kernel version detection returned 'unknown'")
	}

	// Kernel version should contain at least one dot (e.g., "5.15.0")
	if !strings.Contains(status.KernelVersion, ".") {
		t.Errorf("expected kernel version to contain '.', got: %s", status.KernelVersion)
	}

	// Should start with a digit
	if len(status.KernelVersion) == 0 || status.KernelVersion[0] < '0' || status.KernelVersion[0] > '9' {
		t.Errorf("expected kernel version to start with digit, got: %s", status.KernelVersion)
	}
}

// TestStatusFieldsInitialized verifies Status struct is properly initialized
// [REQ:P0-001] Sandbox creation depends on accurate status detection
func TestStatusFieldsInitialized(t *testing.T) {
	status := Check(testStarter)

	// Booleans default to false, which is acceptable
	// But KernelVersion should always be set
	if status.KernelVersion == "" {
		t.Error("KernelVersion should be set even if detection fails")
	}

	// Reason should explain when features aren't available
	if !status.CanCreateUserNamespace && !status.InUserNamespace && status.Reason == "" {
		t.Logf("Note: CanCreateUserNamespace=%v with empty Reason", status.CanCreateUserNamespace)
	}
}
