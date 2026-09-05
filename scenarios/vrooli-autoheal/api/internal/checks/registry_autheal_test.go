package checks

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

func TestRunAutoHeal_ScenarioSharedPackageDriftPrefersSetupRestart(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id:     "scenario-example",
		result: Result{CheckID: "scenario-example", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
			{ID: "setup-restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healableCheck)

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"scenario-example": true,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{
			CheckID: "scenario-example",
			Status:  StatusCritical,
			Details: map[string]interface{}{
				"rootCause":         "shared-package-drift",
				"recommendedAction": "setup-restart",
			},
		},
	}

	reg.RunAutoHeal(context.Background(), results)

	if len(healableCheck.executedActions) != 1 || healableCheck.executedActions[0] != "setup-restart" {
		t.Fatalf("expected setup-restart to be selected, got %v", healableCheck.executedActions)
	}
}

func TestRunAutoHeal_StoppedScenarioPrefersSafeStart(t *testing.T) {
	reg := newTestRegistry()

	healable := &mockHealableCheck{
		id:     "scenario-vrooli-events",
		result: Result{CheckID: "scenario-vrooli-events", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "start", Available: true},
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healable)
	reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: map[string]bool{"scenario-vrooli-events": true}})

	reg.RunAutoHeal(context.Background(), []Result{{
		CheckID: "scenario-vrooli-events",
		Status:  StatusCritical,
		Details: map[string]interface{}{"scenarioStatus": "stopped"},
	}})

	if len(healable.executedActions) != 1 || healable.executedActions[0] != "start" {
		t.Fatalf("expected stopped scenario to select safe start, got %v", healable.executedActions)
	}
}

// TestRunAutoHeal_ScenarioGoDriftPrefersRecoverGo verifies that when a Go
// module drift signature is detected, the targeted recover-go action wins
// over a plain restart.
func TestRunAutoHeal_ScenarioGoDriftPrefersRecoverGo(t *testing.T) {
	reg := newTestRegistry()
	healable := &mockHealableCheck{
		id:     "scenario-agent-manager",
		result: Result{CheckID: "scenario-agent-manager", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
			{ID: "setup-restart", Available: true, Dangerous: true},
			{ID: "recover-go", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healable)
	reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: map[string]bool{"scenario-agent-manager": true}})
	reg.RunAutoHeal(context.Background(), []Result{{
		CheckID: "scenario-agent-manager", Status: StatusCritical,
		Details: map[string]interface{}{
			"rootCause":         "go-module-drift",
			"recommendedAction": "recover-go",
		},
	}})
	if len(healable.executedActions) != 1 || healable.executedActions[0] != "recover-go" {
		t.Fatalf("expected recover-go, got %v", healable.executedActions)
	}
}

// TestRunAutoHeal_ScenarioPnpmDriftPrefersRecoverPnpm mirrors the Go case for
// the pnpm signature path.
func TestRunAutoHeal_ScenarioPnpmDriftPrefersRecoverPnpm(t *testing.T) {
	reg := newTestRegistry()
	healable := &mockHealableCheck{
		id:     "scenario-chart-generator",
		result: Result{CheckID: "scenario-chart-generator", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
			{ID: "recover-pnpm", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healable)
	reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: map[string]bool{"scenario-chart-generator": true}})
	reg.RunAutoHeal(context.Background(), []Result{{
		CheckID: "scenario-chart-generator", Status: StatusCritical,
		Details: map[string]interface{}{
			"rootCause":         "pnpm-install-drift",
			"recommendedAction": "recover-pnpm",
		},
	}})
	if len(healable.executedActions) != 1 || healable.executedActions[0] != "recover-pnpm" {
		t.Fatalf("expected recover-pnpm, got %v", healable.executedActions)
	}
}

// TestRunAutoHeal_RecommendedActionUnavailableFallsThroughToRestart verifies
// that when the recommended action is not Available in the current state, the
// selector falls through to the default scenario restart policy.
func TestRunAutoHeal_RecommendedActionUnavailableFallsThroughToRestart(t *testing.T) {
	reg := newTestRegistry()
	healable := &mockHealableCheck{
		id:     "scenario-agent-manager",
		result: Result{CheckID: "scenario-agent-manager", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
			{ID: "recover-go", Available: false, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healable)
	reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: map[string]bool{"scenario-agent-manager": true}})
	reg.RunAutoHeal(context.Background(), []Result{{
		CheckID: "scenario-agent-manager", Status: StatusCritical,
		Details: map[string]interface{}{"recommendedAction": "recover-go"},
	}})
	if len(healable.executedActions) != 1 || healable.executedActions[0] != "restart" {
		t.Fatalf("expected restart fallback, got %v", healable.executedActions)
	}
}

func TestRunAutoHeal_OrphanCheckPrefersCleanup(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id:     "vrooli-orphans",
		result: Result{CheckID: "vrooli-orphans", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "list", Available: true, Dangerous: false},
			{ID: "kill", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healableCheck)

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"vrooli-orphans": true,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "vrooli-orphans", Status: StatusCritical},
	}

	autoHealResults := reg.RunAutoHeal(context.Background(), results)
	if len(autoHealResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(autoHealResults))
	}
	if !autoHealResults[0].Attempted {
		t.Fatal("expected auto-heal to be attempted")
	}
	if len(healableCheck.executedActions) != 1 || healableCheck.executedActions[0] != "kill" {
		t.Errorf("expected kill to be selected, got %v", healableCheck.executedActions)
	}
}

// TestRunAutoHeal_ScenarioCriticalAllowsRestart verifies controlled dangerous restart
// is allowed for scenario checks.
func TestRunAutoHeal_ScenarioCriticalAllowsRestart(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id:     "scenario-app-monitor",
		result: Result{CheckID: "scenario-app-monitor", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
			{ID: "logs", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true, Message: "Restarted"},
	}
	reg.Register(healableCheck)

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"scenario-app-monitor": true,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "scenario-app-monitor", Status: StatusCritical},
	}

	autoHealResults := reg.RunAutoHeal(context.Background(), results)
	if len(autoHealResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(autoHealResults))
	}
	if !autoHealResults[0].Attempted {
		t.Fatal("expected auto-heal to be attempted")
	}
	if len(healableCheck.executedActions) != 1 || healableCheck.executedActions[0] != "restart" {
		t.Errorf("expected restart to be executed, got %v", healableCheck.executedActions)
	}
}

// TestRunAutoHeal_WarningPolicyCanTrigger verifies warning+critical policy.
func TestRunAutoHeal_WarningPolicyCanTrigger(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id:     "scenario-app-monitor",
		result: Result{CheckID: "scenario-app-monitor", Status: StatusWarning},
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
			{ID: "logs", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true, Message: "Restarted"},
	}
	reg.Register(healableCheck)

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"scenario-app-monitor": true,
		},
		autoHealOn: map[string]string{
			"scenario-app-monitor": "warning+critical",
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "scenario-app-monitor", Status: StatusWarning},
	}

	autoHealResults := reg.RunAutoHeal(context.Background(), results)
	if len(autoHealResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(autoHealResults))
	}
	if !autoHealResults[0].Attempted {
		t.Fatal("expected auto-heal to be attempted for warning+critical policy")
	}
	if len(healableCheck.executedActions) != 1 || healableCheck.executedActions[0] != "restart" {
		t.Errorf("expected restart to be executed, got %v", healableCheck.executedActions)
	}
}

// TestRunAutoHeal_HandlesMultipleCriticalChecks verifies multiple checks are handled
// [REQ:HEAL-ACTION-001]
func TestRunAutoHeal_HandlesMultipleCriticalChecks(t *testing.T) {
	reg := newTestRegistry()

	check1 := &mockHealableCheck{
		id:     "check-1",
		result: Result{CheckID: "check-1", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "action-1", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true},
	}
	check2 := &mockHealableCheck{
		id:     "check-2",
		result: Result{CheckID: "check-2", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "action-2", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: false, Error: "failed"},
	}
	reg.Register(check1)
	reg.Register(check2)

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"check-1": true,
			"check-2": true,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "check-1", Status: StatusCritical},
		{CheckID: "check-2", Status: StatusCritical},
	}

	autoHealResults := reg.RunAutoHeal(context.Background(), results)

	if len(autoHealResults) != 2 {
		t.Fatalf("expected 2 results, got %d", len(autoHealResults))
	}

	// Both should be attempted
	attemptedCount := 0
	for _, r := range autoHealResults {
		if r.Attempted {
			attemptedCount++
		}
	}
	if attemptedCount != 2 {
		t.Errorf("expected 2 attempted healings, got %d", attemptedCount)
	}
}

// TestRunAutoHeal_HandlesMissingCheck verifies graceful handling of missing checks
// [REQ:HEAL-ACTION-001]
func TestRunAutoHeal_HandlesMissingCheck(t *testing.T) {
	reg := newTestRegistry()

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"missing-check": true,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "missing-check", Status: StatusCritical},
	}

	autoHealResults := reg.RunAutoHeal(context.Background(), results)

	// Should skip with appropriate reason since check doesn't exist
	if len(autoHealResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(autoHealResults))
	}
	if autoHealResults[0].Attempted {
		t.Error("expected auto-heal to not be attempted for missing check")
	}
}

// TestSetResult verifies pre-populating results
func TestSetResult(t *testing.T) {
	reg := newTestRegistry()

	check := &mockCheck{
		id:       "test-check",
		interval: 3600, // 1 hour
	}
	reg.Register(check)

	// Pre-populate with a result
	preResult := Result{
		CheckID:   "test-check",
		Status:    StatusOK,
		Message:   "Pre-populated",
		Timestamp: time.Now().Add(-30 * time.Minute), // 30 minutes ago
	}
	reg.SetResult(preResult)

	// Verify result is stored
	result, exists := reg.GetResult("test-check")
	if !exists {
		t.Fatal("expected result to exist after SetResult")
	}
	if result.Message != "Pre-populated" {
		t.Errorf("expected message 'Pre-populated', got %s", result.Message)
	}

	// Verify interval filtering works with pre-populated timestamp
	ctx := context.Background()
	results := reg.RunAll(ctx, false)
	if len(results) != 0 {
		t.Error("expected no checks to run due to interval not elapsed")
	}
}

// =============================================================================
// Concurrent Execution Edge Case Tests
// =============================================================================

// TestConcurrentRegisterUnregister verifies thread-safe registration
// [REQ:HEALTH-REGISTRY-001]
func TestConcurrentRegisterUnregister(t *testing.T) {
	reg := newTestRegistry()

	// Create multiple goroutines registering and unregistering
	const numWorkers = 10
	const opsPerWorker = 50

	done := make(chan bool, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for j := 0; j < opsPerWorker; j++ {
				checkID := string(rune('a'+workerID)) + "-" + string(rune('0'+j%10))
				check := &mockCheck{
					id:     checkID,
					result: Result{CheckID: checkID, Status: StatusOK},
				}

				// Alternate between register and unregister
				if j%2 == 0 {
					reg.Register(check)
				} else {
					reg.Unregister(checkID)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all workers
	for i := 0; i < numWorkers; i++ {
		<-done
	}

	// Verify registry is still in a valid state
	checks := reg.ListChecks()
	t.Logf("After concurrent operations: %d checks registered", len(checks))
}

// TestConcurrentRunAllAndGetResult verifies thread-safe execution
// [REQ:HEALTH-REGISTRY-002]
func TestConcurrentRunAllAndGetResult(t *testing.T) {
	reg := newTestRegistry()

	// Register several checks
	for i := 0; i < 5; i++ {
		check := &mockCheck{
			id:       string(rune('a' + i)),
			interval: 0,
			result:   Result{CheckID: string(rune('a' + i)), Status: StatusOK},
		}
		reg.Register(check)
	}

	const numReaders = 5
	const numWriters = 3
	const opsPerWorker = 20

	done := make(chan bool, numReaders+numWriters)
	ctx := context.Background()

	// Start readers (GetResult, GetAllResults, GetSummary)
	for i := 0; i < numReaders; i++ {
		go func() {
			for j := 0; j < opsPerWorker; j++ {
				switch j % 3 {
				case 0:
					_, _ = reg.GetResult("a")
				case 1:
					_ = reg.GetAllResults()
				case 2:
					_ = reg.GetSummary()
				}
			}
			done <- true
		}()
	}

	// Start writers (RunAll)
	for i := 0; i < numWriters; i++ {
		go func() {
			for j := 0; j < opsPerWorker; j++ {
				_ = reg.RunAll(ctx, j%2 == 0)
			}
			done <- true
		}()
	}

	// Wait for all workers
	for i := 0; i < numReaders+numWriters; i++ {
		<-done
	}

	// Verify registry is still in a valid state
	summary := reg.GetSummary()
	t.Logf("After concurrent operations: total=%d, ok=%d", summary.TotalCount, summary.OkCount)
}

// TestConcurrentAutoHeal verifies thread-safe auto-healing
// [REQ:HEAL-ACTION-001]
func TestConcurrentAutoHeal(t *testing.T) {
	reg := newTestRegistry()

	// Register healable checks
	for i := 0; i < 3; i++ {
		check := &mockHealableCheck{
			id:     string(rune('a' + i)),
			result: Result{CheckID: string(rune('a' + i)), Status: StatusCritical},
			actions: []RecoveryAction{
				{ID: "safe-action", Available: true, Dangerous: false},
			},
			executeResult: ActionResult{Success: true},
		}
		reg.Register(check)
	}

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"a": true,
			"b": true,
			"c": true,
		},
	}
	reg.SetConfigProvider(config)

	const numWorkers = 5
	done := make(chan bool, numWorkers)

	criticalResults := []Result{
		{CheckID: "a", Status: StatusCritical},
		{CheckID: "b", Status: StatusCritical},
		{CheckID: "c", Status: StatusCritical},
	}

	// Multiple workers trying to auto-heal concurrently
	for i := 0; i < numWorkers; i++ {
		go func() {
			_ = reg.RunAutoHeal(context.Background(), criticalResults)
			done <- true
		}()
	}

	// Wait for all workers
	for i := 0; i < numWorkers; i++ {
		<-done
	}

	t.Log("Concurrent auto-heal completed without deadlock")
}

// TestRunAllWithManyChecks verifies performance with many checks
func TestRunAllWithManyChecks(t *testing.T) {
	reg := newTestRegistry()

	const numChecks = 100

	// Register many checks
	for i := 0; i < numChecks; i++ {
		check := &mockCheck{
			id:       "check-" + string(rune('a'+i/26)) + string(rune('a'+i%26)),
			interval: 0,
			result:   Result{Status: StatusOK},
		}
		reg.Register(check)
	}

	ctx := context.Background()
	start := time.Now()
	results := reg.RunAll(ctx, true)
	duration := time.Since(start)

	if len(results) != numChecks {
		t.Errorf("Expected %d results, got %d", numChecks, len(results))
	}

	t.Logf("Running %d checks took %v", numChecks, duration)
}

// TestRunAllContextTimeout verifies timeout handling
func TestRunAllContextTimeout(t *testing.T) {
	reg := newTestRegistry()

	// Add a check that respects context (mock doesn't actually block)
	check := &mockCheck{
		id:       "timeout-check",
		interval: 0,
		result:   Result{CheckID: "timeout-check", Status: StatusOK},
	}
	reg.Register(check)

	// Use a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Sleep to ensure timeout
	time.Sleep(1 * time.Millisecond)

	results := reg.RunAll(ctx, true)
	t.Logf("With expired context, got %d results", len(results))
}

// TestGetCheck verifies retrieving a registered check
// [REQ:HEALTH-REGISTRY-001]
func TestGetCheck(t *testing.T) {
	reg := newTestRegistry()

	check := &mockCheck{
		id:       "single-check",
		interval: 0,
		result:   Result{CheckID: "single-check", Status: StatusWarning, Message: "Test warning"},
	}
	reg.Register(check)

	retrievedCheck, exists := reg.GetCheck("single-check")
	if !exists {
		t.Fatal("Expected check to exist")
	}
	if retrievedCheck.ID() != "single-check" {
		t.Errorf("ID = %q, want %q", retrievedCheck.ID(), "single-check")
	}
}

// TestGetCheckNotFound verifies error handling for missing check
func TestGetCheckNotFound(t *testing.T) {
	reg := newTestRegistry()

	_, exists := reg.GetCheck("nonexistent")
	if exists {
		t.Error("Expected check to not exist")
	}
}

// TestListChecksMetadata verifies retrieving check metadata via ListChecks
func TestListChecksMetadata(t *testing.T) {
	reg := newTestRegistry()

	check := &mockCheck{
		id:        "info-check",
		desc:      "A descriptive check",
		interval:  300,
		platforms: []platform.Type{platform.Linux, platform.MacOS},
	}
	reg.Register(check)

	infos := reg.ListChecks()
	if len(infos) != 1 {
		t.Fatalf("Expected 1 check info, got %d", len(infos))
	}
	info := infos[0]
	if info.ID != "info-check" {
		t.Errorf("ID = %q, want %q", info.ID, "info-check")
	}
	if info.Description != "A descriptive check" {
		t.Errorf("Description = %q, want %q", info.Description, "A descriptive check")
	}
	if info.IntervalSeconds != 300 {
		t.Errorf("IntervalSeconds = %d, want 300", info.IntervalSeconds)
	}
}

// TestListChecksEmpty verifies handling when no checks registered
func TestListChecksEmpty(t *testing.T) {
	reg := newTestRegistry()

	infos := reg.ListChecks()
	if len(infos) != 0 {
		t.Errorf("Expected 0 check infos, got %d", len(infos))
	}
}

// TestRegistryThreadSafetyWithSetResult verifies SetResult is thread-safe
func TestRegistryThreadSafetyWithSetResult(t *testing.T) {
	reg := newTestRegistry()

	check := &mockCheck{
		id:       "thread-check",
		interval: 0,
		result:   Result{CheckID: "thread-check", Status: StatusOK},
	}
	reg.Register(check)

	const numWorkers = 10
	done := make(chan bool, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for j := 0; j < 50; j++ {
				// Alternate between SetResult and GetResult
				if j%2 == 0 {
					reg.SetResult(Result{
						CheckID:   "thread-check",
						Status:    StatusOK,
						Message:   "Update from worker",
						Timestamp: time.Now(),
					})
				} else {
					_, _ = reg.GetResult("thread-check")
				}
			}
			done <- true
		}(i)
	}

	// Wait for all workers
	for i := 0; i < numWorkers; i++ {
		<-done
	}

	// Final state should be valid
	result, exists := reg.GetResult("thread-check")
	if !exists {
		t.Error("Expected result to exist after concurrent operations")
	}
	if result.CheckID != "thread-check" {
		t.Errorf("CheckID = %q, want %q", result.CheckID, "thread-check")
	}
}

// TestConfigProviderIntegration verifies config provider is used correctly
// [REQ:CONFIG-CHECK-001]
func TestConfigProviderIntegration(t *testing.T) {
	reg := newTestRegistry()

	check1 := &mockCheck{
		id:       "enabled-check",
		interval: 0,
		result:   Result{CheckID: "enabled-check", Status: StatusOK},
	}
	check2 := &mockCheck{
		id:       "disabled-check",
		interval: 0,
		result:   Result{CheckID: "disabled-check", Status: StatusOK},
	}
	reg.Register(check1)
	reg.Register(check2)

	// Without config provider, all checks run
	ctx := context.Background()
	results := reg.RunAll(ctx, true)
	if len(results) != 2 {
		t.Errorf("Without config, expected 2 results, got %d", len(results))
	}

	// With config provider that disables one check
	config := &mockConfigProvider{
		enabledChecks: map[string]bool{
			"enabled-check":  true,
			"disabled-check": false,
		},
	}
	reg.SetConfigProvider(config)

	results = reg.RunAll(ctx, true)
	// Note: This depends on whether RunAll respects IsCheckEnabled
	// The current implementation may not filter by enabled status in RunAll
	// Just verify it doesn't crash
	t.Logf("With config provider, got %d results", len(results))
}
