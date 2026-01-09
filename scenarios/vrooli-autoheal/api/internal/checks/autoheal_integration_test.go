package checks

import (
	"context"
	"testing"

	"vrooli-autoheal/internal/platform"
)

// Integration tests that verify end-to-end auto-heal scenarios.
// These tests simulate the full flow: check fails -> auto-heal selects action -> action executes -> verify result.

// testConfigProvider is a mock ConfigProvider for integration tests.
type testConfigProvider struct {
	enabledChecks   map[string]bool
	autoHealEnabled map[string]bool
}

func newTestConfigProvider() *testConfigProvider {
	return &testConfigProvider{
		enabledChecks:   make(map[string]bool),
		autoHealEnabled: make(map[string]bool),
	}
}

func (c *testConfigProvider) IsCheckEnabled(checkID string) bool {
	enabled, exists := c.enabledChecks[checkID]
	return !exists || enabled // Default to enabled if not explicitly set
}

func (c *testConfigProvider) IsAutoHealEnabled(checkID string) bool {
	enabled, exists := c.autoHealEnabled[checkID]
	return exists && enabled
}

func (c *testConfigProvider) enableAutoHeal(checkID string) {
	c.autoHealEnabled[checkID] = true
	c.enabledChecks[checkID] = true
}

func (c *testConfigProvider) disableAutoHeal(checkID string) {
	c.autoHealEnabled[checkID] = false
	c.enabledChecks[checkID] = true
}

// mockAutoHealCheck simulates a check that can be auto-healed.
type mockAutoHealCheck struct {
	id              string
	status          Status
	autoHealEnabled bool
	recoveryActions []RecoveryAction
	executeResult   ActionResult
	runCount        int
}

func (c *mockAutoHealCheck) ID() string                 { return c.id }
func (c *mockAutoHealCheck) Title() string              { return "Mock Auto-Heal Check" }
func (c *mockAutoHealCheck) Description() string        { return "A mock check for integration testing" }
func (c *mockAutoHealCheck) Importance() string         { return "Required for testing auto-heal scenarios" }
func (c *mockAutoHealCheck) Category() Category         { return CategoryInfrastructure }
func (c *mockAutoHealCheck) IntervalSeconds() int       { return 60 }
func (c *mockAutoHealCheck) Platforms() []platform.Type { return nil }

func (c *mockAutoHealCheck) Run(ctx context.Context) Result {
	c.runCount++
	return Result{
		CheckID: c.id,
		Status:  c.status,
		Message: "Mock result",
		Details: map[string]interface{}{
			"runCount": c.runCount,
		},
	}
}

func (c *mockAutoHealCheck) RecoveryActions(lastResult *Result) []RecoveryAction {
	return c.recoveryActions
}

func (c *mockAutoHealCheck) ExecuteAction(ctx context.Context, actionID string) ActionResult {
	c.executeResult.ActionID = actionID
	c.executeResult.CheckID = c.id
	return c.executeResult
}

// TestAutoHealIntegration_CriticalCheckTriggersHeal verifies that a critical check
// triggers auto-heal when enabled and a safe action is available.
func TestAutoHealIntegration_CriticalCheckTriggersHeal(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := NewRegistry(caps)

	// Create config provider
	config := newTestConfigProvider()
	registry.SetConfigProvider(config)

	// Create a mock check that will report critical status
	check := &mockAutoHealCheck{
		id:              "critical-check",
		status:          StatusCritical,
		autoHealEnabled: true,
		recoveryActions: []RecoveryAction{
			{ID: "start", Name: "Start", Dangerous: false, Available: true},
			{ID: "restart", Name: "Restart", Dangerous: true, Available: true},
		},
		executeResult: ActionResult{
			Success: true,
			Message: "Action executed successfully",
		},
	}

	registry.Register(check)
	config.enableAutoHeal(check.id)

	// Run all checks
	ctx := context.Background()
	results := registry.RunAll(ctx, true)

	// Verify check ran and returned critical
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	var criticalResult *Result
	for i := range results {
		if results[i].CheckID == "critical-check" {
			criticalResult = &results[i]
			break
		}
	}

	if criticalResult == nil {
		t.Fatal("expected critical-check result")
	}

	if criticalResult.Status != StatusCritical {
		t.Errorf("expected Critical status, got %s", criticalResult.Status)
	}

	// Run auto-heal
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Verify auto-heal attempted action
	var healResult *AutoHealResult
	for i := range autoHealResults {
		if autoHealResults[i].CheckID == "critical-check" {
			healResult = &autoHealResults[i]
			break
		}
	}

	if healResult == nil {
		t.Fatal("expected auto-heal result for critical-check")
	}

	if !healResult.Attempted {
		t.Error("expected auto-heal to be attempted")
	}

	// Should have picked the safe (non-dangerous) action
	if healResult.ActionResult.ActionID != "start" {
		t.Errorf("expected 'start' action (safe), got %s", healResult.ActionResult.ActionID)
	}

	if !healResult.ActionResult.Success {
		t.Error("expected action to succeed")
	}
}

// TestAutoHealIntegration_DisabledDoesNotTrigger verifies that auto-heal
// does not trigger when disabled for a check.
func TestAutoHealIntegration_DisabledDoesNotTrigger(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := NewRegistry(caps)

	config := newTestConfigProvider()
	registry.SetConfigProvider(config)

	check := &mockAutoHealCheck{
		id:              "disabled-heal-check",
		status:          StatusCritical,
		autoHealEnabled: false,
		recoveryActions: []RecoveryAction{
			{ID: "start", Name: "Start", Dangerous: false, Available: true},
		},
		executeResult: ActionResult{Success: true},
	}

	registry.Register(check)
	// Explicitly disable auto-heal
	config.disableAutoHeal(check.id)

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Verify no auto-heal was attempted
	for _, ahr := range autoHealResults {
		if ahr.CheckID == "disabled-heal-check" && ahr.Attempted {
			t.Error("auto-heal should not be attempted when disabled")
		}
	}
}

// TestAutoHealIntegration_NoSafeActionSkipsHeal verifies that auto-heal
// skips checks that only have dangerous actions.
func TestAutoHealIntegration_NoSafeActionSkipsHeal(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := NewRegistry(caps)

	config := newTestConfigProvider()
	registry.SetConfigProvider(config)

	check := &mockAutoHealCheck{
		id:              "dangerous-only-check",
		status:          StatusCritical,
		autoHealEnabled: true,
		recoveryActions: []RecoveryAction{
			{ID: "restart", Name: "Restart", Dangerous: true, Available: true},
			{ID: "stop", Name: "Stop", Dangerous: true, Available: true},
		},
		executeResult: ActionResult{Success: true},
	}

	registry.Register(check)
	config.enableAutoHeal(check.id)

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Verify no auto-heal was attempted (all actions are dangerous)
	for _, ahr := range autoHealResults {
		if ahr.CheckID == "dangerous-only-check" && ahr.Attempted {
			t.Error("auto-heal should not execute dangerous actions automatically")
		}
	}
}

// TestAutoHealIntegration_OKStatusSkipsHeal verifies that auto-heal
// is not triggered for healthy checks.
func TestAutoHealIntegration_OKStatusSkipsHeal(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := NewRegistry(caps)

	config := newTestConfigProvider()
	registry.SetConfigProvider(config)

	check := &mockAutoHealCheck{
		id:              "healthy-check",
		status:          StatusOK,
		autoHealEnabled: true,
		recoveryActions: []RecoveryAction{
			{ID: "start", Name: "Start", Dangerous: false, Available: true},
		},
		executeResult: ActionResult{Success: true},
	}

	registry.Register(check)
	config.enableAutoHeal(check.id)

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Verify no auto-heal was attempted for healthy check
	for _, ahr := range autoHealResults {
		if ahr.CheckID == "healthy-check" && ahr.Attempted {
			t.Error("auto-heal should not trigger for healthy checks")
		}
	}
}

// TestAutoHealIntegration_WarningStatusSkipsHeal verifies that auto-heal
// only triggers for critical checks, not warnings.
func TestAutoHealIntegration_WarningStatusSkipsHeal(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := NewRegistry(caps)

	config := newTestConfigProvider()
	registry.SetConfigProvider(config)

	check := &mockAutoHealCheck{
		id:              "warning-check",
		status:          StatusWarning,
		autoHealEnabled: true,
		recoveryActions: []RecoveryAction{
			{ID: "start", Name: "Start", Dangerous: false, Available: true},
		},
		executeResult: ActionResult{Success: true},
	}

	registry.Register(check)
	config.enableAutoHeal(check.id)

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Verify no auto-heal was attempted for warning check
	for _, ahr := range autoHealResults {
		if ahr.CheckID == "warning-check" && ahr.Attempted {
			t.Error("auto-heal should not trigger for warning checks")
		}
	}
}

// TestAutoHealIntegration_UnavailableActionSkipped verifies that auto-heal
// skips actions that are not currently available.
func TestAutoHealIntegration_UnavailableActionSkipped(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := NewRegistry(caps)

	config := newTestConfigProvider()
	registry.SetConfigProvider(config)

	check := &mockAutoHealCheck{
		id:              "unavailable-action-check",
		status:          StatusCritical,
		autoHealEnabled: true,
		recoveryActions: []RecoveryAction{
			{ID: "start", Name: "Start", Dangerous: false, Available: false}, // Not available
			{ID: "status", Name: "Status", Dangerous: false, Available: true},
		},
		executeResult: ActionResult{Success: true},
	}

	registry.Register(check)
	config.enableAutoHeal(check.id)

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Verify auto-heal picked the available action
	for _, ahr := range autoHealResults {
		if ahr.CheckID == "unavailable-action-check" {
			if ahr.Attempted && ahr.ActionResult.ActionID == "start" {
				t.Error("should not attempt unavailable action")
			}
			if ahr.Attempted && ahr.ActionResult.ActionID == "status" {
				// This is expected - picked the available action
				t.Log("correctly picked available action 'status'")
			}
		}
	}
}

// TestAutoHealIntegration_FailedActionRecorded verifies that failed
// auto-heal actions are properly recorded.
func TestAutoHealIntegration_FailedActionRecorded(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := NewRegistry(caps)

	config := newTestConfigProvider()
	registry.SetConfigProvider(config)

	check := &mockAutoHealCheck{
		id:              "failing-action-check",
		status:          StatusCritical,
		autoHealEnabled: true,
		recoveryActions: []RecoveryAction{
			{ID: "start", Name: "Start", Dangerous: false, Available: true},
		},
		executeResult: ActionResult{
			Success: false,
			Error:   "Failed to start service",
			Message: "Service start failed",
		},
	}

	registry.Register(check)
	config.enableAutoHeal(check.id)

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Verify failed action is recorded
	var healResult *AutoHealResult
	for i := range autoHealResults {
		if autoHealResults[i].CheckID == "failing-action-check" {
			healResult = &autoHealResults[i]
			break
		}
	}

	if healResult == nil {
		t.Fatal("expected auto-heal result")
	}

	if !healResult.Attempted {
		t.Error("expected auto-heal to be attempted")
	}

	if healResult.ActionResult.Success {
		t.Error("expected action to fail")
	}

	if healResult.ActionResult.Error == "" {
		t.Error("expected error message in result")
	}
}

// TestAutoHealIntegration_MultipleChecksParallel verifies that auto-heal
// handles multiple critical checks correctly.
func TestAutoHealIntegration_MultipleChecksParallel(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := NewRegistry(caps)

	config := newTestConfigProvider()
	registry.SetConfigProvider(config)

	// Register multiple critical checks
	for i := 1; i <= 3; i++ {
		check := &mockAutoHealCheck{
			id:              "critical-check-" + string(rune('0'+i)),
			status:          StatusCritical,
			autoHealEnabled: true,
			recoveryActions: []RecoveryAction{
				{ID: "start", Name: "Start", Dangerous: false, Available: true},
			},
			executeResult: ActionResult{Success: true},
		}
		registry.Register(check)
		config.enableAutoHeal(check.id)
	}

	// Add one healthy check
	healthyCheck := &mockAutoHealCheck{
		id:              "healthy-check",
		status:          StatusOK,
		autoHealEnabled: true,
		recoveryActions: []RecoveryAction{
			{ID: "start", Name: "Start", Dangerous: false, Available: true},
		},
		executeResult: ActionResult{Success: true},
	}
	registry.Register(healthyCheck)
	config.enableAutoHeal(healthyCheck.id)

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Count auto-heal attempts
	attempted := 0
	for _, ahr := range autoHealResults {
		if ahr.Attempted {
			attempted++
		}
	}

	// Should have attempted auto-heal for the 3 critical checks
	if attempted != 3 {
		t.Errorf("expected 3 auto-heal attempts, got %d", attempted)
	}
}

// TestAutoHealIntegration_SelectsFirstSafeAction verifies that auto-heal
// selects the first available safe action from the list.
func TestAutoHealIntegration_SelectsFirstSafeAction(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := NewRegistry(caps)

	config := newTestConfigProvider()
	registry.SetConfigProvider(config)

	check := &mockAutoHealCheck{
		id:              "ordered-actions-check",
		status:          StatusCritical,
		autoHealEnabled: true,
		recoveryActions: []RecoveryAction{
			// First action is dangerous (should be skipped)
			{ID: "restart", Name: "Restart", Dangerous: true, Available: true},
			// Second action is unavailable (should be skipped)
			{ID: "fix", Name: "Fix", Dangerous: false, Available: false},
			// Third action is safe and available (should be selected)
			{ID: "start", Name: "Start", Dangerous: false, Available: true},
			// Fourth action is also safe (should not be selected - first wins)
			{ID: "logs", Name: "Logs", Dangerous: false, Available: true},
		},
		executeResult: ActionResult{Success: true},
	}

	registry.Register(check)
	config.enableAutoHeal(check.id)

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Verify correct action was selected
	for _, ahr := range autoHealResults {
		if ahr.CheckID == "ordered-actions-check" && ahr.Attempted {
			if ahr.ActionResult.ActionID != "start" {
				t.Errorf("expected 'start' action to be selected (first safe available), got %s", ahr.ActionResult.ActionID)
			}
		}
	}
}
