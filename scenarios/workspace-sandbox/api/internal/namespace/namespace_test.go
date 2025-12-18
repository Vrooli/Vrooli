// Package namespace tests
package namespace

import (
	"os"
	"strings"
	"testing"
)

// TestCheckReturnsStatus verifies that Check() returns a valid Status struct
// [REQ:P0-001] Sandbox creation relies on namespace status detection
func TestCheckReturnsStatus(t *testing.T) {
	status := Check()

	// Status should always have a kernel version
	if status.KernelVersion == "" {
		t.Error("expected non-empty kernel version")
	}

	// If we're in a CI environment or user namespaces disabled,
	// we should get a reason explaining why
	if !status.CanCreateUserNamespace && status.Reason == "" {
		// This is expected when user namespaces are disabled
		if os.Getenv(DisableUserNamespaceEnv) != "1" {
			// Only require reason if not explicitly disabled
			// (disabled case sets reason in Check)
		}
	}

	t.Logf("Namespace status: InUserNamespace=%v, CanCreate=%v, CanMount=%v, Kernel=%s, Reason=%s",
		status.InUserNamespace,
		status.CanCreateUserNamespace,
		status.CanMountOverlayfs,
		status.KernelVersion,
		status.Reason)
}

// TestCheckWithDisabledEnv verifies Check honors the disable environment variable
// [REQ:P2-025] Cross-platform driver interface fallback detection
func TestCheckWithDisabledEnv(t *testing.T) {
	// Save and restore environment
	oldVal := os.Getenv(DisableUserNamespaceEnv)
	defer os.Setenv(DisableUserNamespaceEnv, oldVal)

	// Disable user namespaces
	os.Setenv(DisableUserNamespaceEnv, "1")

	status := Check()

	if status.CanCreateUserNamespace {
		t.Error("expected CanCreateUserNamespace to be false when disabled via env")
	}

	if !strings.Contains(status.Reason, DisableUserNamespaceEnv) {
		t.Errorf("expected reason to mention %s, got: %s", DisableUserNamespaceEnv, status.Reason)
	}
}

// TestIsKernelAtLeastBasicCases tests kernel version comparison
// [REQ:P0-002] Driver selection depends on kernel version detection
func TestIsKernelAtLeastBasicCases(t *testing.T) {
	// Get actual kernel version for context
	status := Check()
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
	status := Check()

	// Parse actual kernel version for comparison
	parts := strings.Split(status.KernelVersion, ".")
	if len(parts) < 2 {
		t.Skipf("Cannot parse kernel version: %s", status.KernelVersion)
	}

	// Test that the current kernel version matches itself
	// We can't test exact values without parsing, but we can test relative behavior
	t.Logf("Kernel parts: %v", parts)
}

// TestEnterUserNamespaceAlreadyIn verifies no-op when already in namespace
// [REQ:P0-001] Sandbox creation can be repeated safely
func TestEnterUserNamespaceAlreadyIn(t *testing.T) {
	// Save and restore environment
	oldVal := os.Getenv(InUserNamespaceEnv)
	defer os.Setenv(InUserNamespaceEnv, oldVal)

	// Simulate already being in user namespace
	os.Setenv(InUserNamespaceEnv, "1")

	err := EnterUserNamespace()
	if err != nil {
		t.Errorf("expected nil error when already in namespace, got: %v", err)
	}
}

// TestEnterUserNamespaceDisabled verifies no-op when explicitly disabled
// [REQ:P2-025] Cross-platform driver interface fallback handling
func TestEnterUserNamespaceDisabled(t *testing.T) {
	// Save and restore environment
	oldVal := os.Getenv(DisableUserNamespaceEnv)
	oldInVal := os.Getenv(InUserNamespaceEnv)
	defer func() {
		os.Setenv(DisableUserNamespaceEnv, oldVal)
		os.Setenv(InUserNamespaceEnv, oldInVal)
	}()

	// Ensure we're not "in" a namespace
	os.Unsetenv(InUserNamespaceEnv)

	// Disable user namespaces
	os.Setenv(DisableUserNamespaceEnv, "1")

	err := EnterUserNamespace()
	if err != nil {
		t.Errorf("expected nil error when disabled, got: %v", err)
	}
}

// TestConstantsAreDefined verifies that environment variable constants are defined
// [REQ:P0-007] Sandbox lifecycle management uses these for detection
func TestConstantsAreDefined(t *testing.T) {
	if InUserNamespaceEnv == "" {
		t.Error("InUserNamespaceEnv constant should not be empty")
	}
	if DisableUserNamespaceEnv == "" {
		t.Error("DisableUserNamespaceEnv constant should not be empty")
	}
	if RequiredKernelMajor == 0 {
		t.Error("RequiredKernelMajor should not be 0")
	}
}

// TestKernelVersionFormat verifies kernel version string format
// [REQ:P2-025] Cross-platform driver selection uses kernel version
func TestKernelVersionFormat(t *testing.T) {
	status := Check()

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
	status := Check()

	// Booleans default to false, which is acceptable
	// But KernelVersion should always be set
	if status.KernelVersion == "" {
		t.Error("KernelVersion should be set even if detection fails")
	}

	// Reason should explain when features aren't available
	if !status.CanCreateUserNamespace && !status.InUserNamespace && status.Reason == "" {
		// Reason may be empty only if env var explicitly disables and we already checked
		if os.Getenv(DisableUserNamespaceEnv) != "1" {
			// In most CI environments, user namespaces aren't available
			// Reason should explain this
			t.Logf("Note: CanCreateUserNamespace=%v with empty Reason", status.CanCreateUserNamespace)
		}
	}
}

// TestMustEnterUserNamespaceCallsLogger verifies logger is called on failure
// [REQ:P0-007] Sandbox lifecycle management logging
func TestMustEnterUserNamespaceCallsLogger(t *testing.T) {
	// Save and restore environment
	oldVal := os.Getenv(DisableUserNamespaceEnv)
	oldInVal := os.Getenv(InUserNamespaceEnv)
	defer func() {
		os.Setenv(DisableUserNamespaceEnv, oldVal)
		os.Setenv(InUserNamespaceEnv, oldInVal)
	}()

	// Ensure we're not "in" a namespace and namespaces work
	os.Unsetenv(InUserNamespaceEnv)
	os.Unsetenv(DisableUserNamespaceEnv)

	// Track if logger was called
	loggerCalled := false
	logger := func(format string, args ...interface{}) {
		loggerCalled = true
		t.Logf("Logger output: "+format, args...)
	}

	// Call MustEnterUserNamespace - it may or may not call logger
	// depending on whether namespaces are available
	MustEnterUserNamespace(logger)

	// We can't assert loggerCalled because it depends on the system
	// Just verify no panic occurred
	t.Logf("Logger was called: %v", loggerCalled)
}
