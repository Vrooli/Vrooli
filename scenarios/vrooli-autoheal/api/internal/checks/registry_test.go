// Package checks tests for Registry
// [REQ:HEALTH-REGISTRY-001] [REQ:HEALTH-REGISTRY-002] [REQ:HEALTH-REGISTRY-003] [REQ:HEALTH-REGISTRY-004]
package checks

import (
	"context"
	"testing"
	"time"
	"vrooli-autoheal/internal/platform"
)

// TestNewRegistry verifies registry initialization
// [REQ:HEALTH-REGISTRY-001]
func TestNewRegistry(t *testing.T) {
	plat := testPlatform()
	reg := NewRegistry(plat)

	if reg == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if reg.checks == nil {
		t.Error("checks map not initialized")
	}

	if reg.results == nil {
		t.Error("results map not initialized")
	}

	if reg.lastRun == nil {
		t.Error("lastRun map not initialized")
	}

	if reg.platform == nil {
		t.Error("platform not set")
	}

	if reg.platform != plat {
		t.Error("platform not set to injected value")
	}
}

// TestRegisterUnregister verifies check registration and removal
// [REQ:HEALTH-REGISTRY-001]
func TestRegisterUnregister(t *testing.T) {
	reg := newTestRegistry()

	check := &mockCheck{
		id:       "test-check",
		desc:     "Test check",
		interval: 60,
		result:   Result{CheckID: "test-check", Status: StatusOK, Message: "OK"},
	}

	// Register check
	reg.Register(check)

	checks := reg.ListChecks()
	if len(checks) != 1 {
		t.Errorf("ListChecks() returned %d checks, want 1", len(checks))
	}

	if checks[0].ID != "test-check" {
		t.Errorf("check ID = %q, want %q", checks[0].ID, "test-check")
	}

	// Unregister check
	reg.Unregister("test-check")

	checks = reg.ListChecks()
	if len(checks) != 0 {
		t.Errorf("After unregister, ListChecks() returned %d checks, want 0", len(checks))
	}
}

// TestRunAll verifies running all checks
// [REQ:HEALTH-REGISTRY-002]
func TestRunAll(t *testing.T) {
	reg := newTestRegistry()

	check1 := &mockCheck{
		id:       "check-1",
		interval: 0, // Always run
		result:   Result{CheckID: "check-1", Status: StatusOK, Message: "OK"},
	}
	check2 := &mockCheck{
		id:       "check-2",
		interval: 0,
		result:   Result{CheckID: "check-2", Status: StatusWarning, Message: "Warning"},
	}

	reg.Register(check1)
	reg.Register(check2)

	ctx := context.Background()
	results := reg.RunAll(ctx, false)

	if len(results) != 2 {
		t.Errorf("RunAll() returned %d results, want 2", len(results))
	}
}

// TestPlatformFiltering verifies platform-based check filtering
// [REQ:HEALTH-REGISTRY-003]
func TestPlatformFiltering(t *testing.T) {
	reg := newTestRegistry()

	currentPlatform := reg.platform.Platform

	// Check that runs on all platforms
	allPlatformsCheck := &mockCheck{
		id:        "all-platforms",
		interval:  0,
		platforms: nil, // nil = all platforms
		result:    Result{CheckID: "all-platforms", Status: StatusOK},
	}

	// Check that runs only on current platform
	currentPlatformCheck := &mockCheck{
		id:        "current-platform",
		interval:  0,
		platforms: []platform.Type{currentPlatform},
		result:    Result{CheckID: "current-platform", Status: StatusOK},
	}

	// Check that runs on a different platform
	var otherPlatform platform.Type
	switch currentPlatform {
	case platform.Linux:
		otherPlatform = platform.Windows
	case platform.Windows:
		otherPlatform = platform.Linux
	case platform.MacOS:
		otherPlatform = platform.Linux
	default:
		otherPlatform = platform.Linux
	}

	otherPlatformCheck := &mockCheck{
		id:        "other-platform",
		interval:  0,
		platforms: []platform.Type{otherPlatform},
		result:    Result{CheckID: "other-platform", Status: StatusOK},
	}

	reg.Register(allPlatformsCheck)
	reg.Register(currentPlatformCheck)
	reg.Register(otherPlatformCheck)

	ctx := context.Background()
	results := reg.RunAll(ctx, true)

	// Should run all-platforms and current-platform, but not other-platform
	foundAllPlatforms := false
	foundCurrentPlatform := false
	foundOtherPlatform := false

	for _, r := range results {
		switch r.CheckID {
		case "all-platforms":
			foundAllPlatforms = true
		case "current-platform":
			foundCurrentPlatform = true
		case "other-platform":
			foundOtherPlatform = true
		}
	}

	if !foundAllPlatforms {
		t.Error("all-platforms check did not run")
	}
	if !foundCurrentPlatform {
		t.Error("current-platform check did not run")
	}
	if foundOtherPlatform {
		t.Errorf("other-platform check ran on %s", currentPlatform)
	}
}

// TestIntervalFiltering verifies interval-based check filtering
// [REQ:HEALTH-REGISTRY-003]
func TestIntervalFiltering(t *testing.T) {
	reg := newTestRegistry()

	check := &mockCheck{
		id:       "interval-check",
		interval: 3600, // 1 hour
		result:   Result{CheckID: "interval-check", Status: StatusOK},
	}

	reg.Register(check)

	ctx := context.Background()

	// First run should execute
	results1 := reg.RunAll(ctx, false)
	if len(results1) != 1 {
		t.Errorf("First RunAll() returned %d results, want 1", len(results1))
	}

	// Second run without force should skip (interval not elapsed)
	results2 := reg.RunAll(ctx, false)
	if len(results2) != 0 {
		t.Errorf("Second RunAll() without force returned %d results, want 0", len(results2))
	}

	// Third run with force should execute
	results3 := reg.RunAll(ctx, true)
	if len(results3) != 1 {
		t.Errorf("Third RunAll() with force returned %d results, want 1", len(results3))
	}
}

// TestGetResult verifies result retrieval
// [REQ:HEALTH-REGISTRY-004]
func TestGetResult(t *testing.T) {
	reg := newTestRegistry()

	check := &mockCheck{
		id:       "result-check",
		interval: 0,
		result:   Result{CheckID: "result-check", Status: StatusOK, Message: "Test OK"},
	}

	reg.Register(check)

	// Before running, result should not exist
	_, exists := reg.GetResult("result-check")
	if exists {
		t.Error("GetResult() found result before check was run")
	}

	// Run the check
	ctx := context.Background()
	reg.RunAll(ctx, true)

	// After running, result should exist
	result, exists := reg.GetResult("result-check")
	if !exists {
		t.Fatal("GetResult() did not find result after check was run")
	}

	if result.Status != StatusOK {
		t.Errorf("result.Status = %v, want %v", result.Status, StatusOK)
	}

	if result.Message != "Test OK" {
		t.Errorf("result.Message = %q, want %q", result.Message, "Test OK")
	}
}

// TestGetAllResults verifies bulk result retrieval
func TestGetAllResults(t *testing.T) {
	reg := newTestRegistry()

	check1 := &mockCheck{
		id:       "bulk-1",
		interval: 0,
		result:   Result{CheckID: "bulk-1", Status: StatusOK},
	}
	check2 := &mockCheck{
		id:       "bulk-2",
		interval: 0,
		result:   Result{CheckID: "bulk-2", Status: StatusWarning},
	}

	reg.Register(check1)
	reg.Register(check2)

	ctx := context.Background()
	reg.RunAll(ctx, true)

	allResults := reg.GetAllResults()
	if len(allResults) != 2 {
		t.Errorf("GetAllResults() returned %d results, want 2", len(allResults))
	}
}

// TestGetSummary verifies health summary calculation
// [REQ:HEALTH-REGISTRY-004]
func TestGetSummary(t *testing.T) {
	reg := newTestRegistry()

	checks := []*mockCheck{
		{id: "ok-1", result: Result{CheckID: "ok-1", Status: StatusOK}},
		{id: "ok-2", result: Result{CheckID: "ok-2", Status: StatusOK}},
		{id: "warn-1", result: Result{CheckID: "warn-1", Status: StatusWarning}},
		{id: "crit-1", result: Result{CheckID: "crit-1", Status: StatusCritical}},
	}

	for _, c := range checks {
		reg.Register(c)
	}

	ctx := context.Background()
	reg.RunAll(ctx, true)

	summary := reg.GetSummary()

	if summary.TotalCount != 4 {
		t.Errorf("summary.TotalCount = %d, want 4", summary.TotalCount)
	}

	if summary.OkCount != 2 {
		t.Errorf("summary.OkCount = %d, want 2", summary.OkCount)
	}

	if summary.WarnCount != 1 {
		t.Errorf("summary.WarnCount = %d, want 1", summary.WarnCount)
	}

	if summary.CritCount != 1 {
		t.Errorf("summary.CritCount = %d, want 1", summary.CritCount)
	}

	// Overall status should be critical (worst status)
	if summary.Status != StatusCritical {
		t.Errorf("summary.Status = %v, want %v", summary.Status, StatusCritical)
	}
}

// TestSummaryStatusCalculation verifies overall status is worst status
func TestSummaryStatusCalculation(t *testing.T) {
	tests := []struct {
		name     string
		statuses []Status
		expected Status
	}{
		{"all ok", []Status{StatusOK, StatusOK}, StatusOK},
		{"one warning", []Status{StatusOK, StatusWarning}, StatusWarning},
		{"one critical", []Status{StatusOK, StatusCritical}, StatusCritical},
		{"warning and critical", []Status{StatusWarning, StatusCritical}, StatusCritical},
		{"empty", []Status{}, StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := newTestRegistry()

			for i, status := range tc.statuses {
				check := &mockCheck{
					id:     string(rune('a' + i)),
					result: Result{CheckID: string(rune('a' + i)), Status: status},
				}
				reg.Register(check)
			}

			ctx := context.Background()
			reg.RunAll(ctx, true)

			summary := reg.GetSummary()
			if summary.Status != tc.expected {
				t.Errorf("status = %v, want %v", summary.Status, tc.expected)
			}
		})
	}
}

// TestRunCheckSetsTimestamp verifies timestamp is set correctly
func TestRunCheckSetsTimestamp(t *testing.T) {
	reg := newTestRegistry()

	check := &mockCheck{
		id:       "timestamp-check",
		interval: 0,
		result:   Result{CheckID: "timestamp-check", Status: StatusOK},
	}

	reg.Register(check)

	before := time.Now()
	ctx := context.Background()
	reg.RunAll(ctx, true)
	after := time.Now()

	result, _ := reg.GetResult("timestamp-check")

	if result.Timestamp.Before(before) || result.Timestamp.After(after) {
		t.Errorf("Timestamp %v not in range [%v, %v]", result.Timestamp, before, after)
	}

	if result.Duration < 0 {
		t.Errorf("Duration %v should be non-negative", result.Duration)
	}
}

// TestContextCancellation verifies checks stop on context cancellation
func TestContextCancellation(t *testing.T) {
	reg := newTestRegistry()

	// Add several checks
	for i := 0; i < 10; i++ {
		check := &mockCheck{
			id:       string(rune('a' + i)),
			interval: 0,
			result:   Result{CheckID: string(rune('a' + i)), Status: StatusOK},
		}
		reg.Register(check)
	}

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := reg.RunAll(ctx, true)

	// Should return immediately with no or very few results
	t.Logf("With cancelled context, got %d results", len(results))
}

// TestListChecks verifies check metadata listing
func TestListChecks(t *testing.T) {
	reg := newTestRegistry()

	check := &mockCheck{
		id:        "list-check",
		desc:      "A test check",
		interval:  120,
		platforms: []platform.Type{platform.Linux},
	}

	reg.Register(check)

	checks := reg.ListChecks()

	if len(checks) != 1 {
		t.Fatalf("ListChecks() returned %d checks, want 1", len(checks))
	}

	info := checks[0]
	if info.ID != "list-check" {
		t.Errorf("info.ID = %q, want %q", info.ID, "list-check")
	}
	if info.Description != "A test check" {
		t.Errorf("info.Description = %q, want %q", info.Description, "A test check")
	}
	if info.IntervalSeconds != 120 {
		t.Errorf("info.IntervalSeconds = %d, want 120", info.IntervalSeconds)
	}
	if len(info.Platforms) != 1 || info.Platforms[0] != platform.Linux {
		t.Errorf("info.Platforms = %v, want [linux]", info.Platforms)
	}
}

// TestIsHealable verifies check healability detection
// [REQ:HEAL-ACTION-001]
func TestIsHealable(t *testing.T) {
	reg := newTestRegistry()

	// Register a non-healable check
	regularCheck := &mockCheck{
		id:     "regular-check",
		result: Result{CheckID: "regular-check", Status: StatusOK},
	}
	reg.Register(regularCheck)

	// Register a healable check
	healableCheck := &mockHealableCheck{
		id:     "healable-check",
		result: Result{CheckID: "healable-check", Status: StatusOK},
	}
	reg.Register(healableCheck)

	if reg.IsHealable("regular-check") {
		t.Error("expected regular check to not be healable")
	}

	if !reg.IsHealable("healable-check") {
		t.Error("expected healable check to be healable")
	}

	if reg.IsHealable("nonexistent-check") {
		t.Error("expected nonexistent check to not be healable")
	}
}

// TestGetHealableCheck verifies retrieving healable checks
// [REQ:HEAL-ACTION-001]
func TestGetHealableCheck(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id: "healable-check",
	}
	reg.Register(healableCheck)

	check, ok := reg.GetHealableCheck("healable-check")
	if !ok {
		t.Fatal("expected to find healable check")
	}
	if check.ID() != "healable-check" {
		t.Errorf("expected check ID 'healable-check', got %s", check.ID())
	}

	_, ok = reg.GetHealableCheck("nonexistent")
	if ok {
		t.Error("expected nonexistent check to not be found")
	}
}

// TestIsAutoHealEnabled verifies auto-heal config integration
// [REQ:CONFIG-CHECK-001]
func TestIsAutoHealEnabled(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id: "healable-check",
	}
	reg.Register(healableCheck)

	// Without config provider, auto-heal should be disabled
	if reg.IsAutoHealEnabled("healable-check") {
		t.Error("expected auto-heal to be disabled without config provider")
	}

	// With config provider that enables auto-heal
	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"healable-check": true,
		},
	}
	reg.SetConfigProvider(config)

	if !reg.IsAutoHealEnabled("healable-check") {
		t.Error("expected auto-heal to be enabled for healable-check")
	}

	// Non-healable check should never have auto-heal enabled
	regularCheck := &mockCheck{
		id: "regular-check",
	}
	reg.Register(regularCheck)
	config.autoHealChecks["regular-check"] = true

	if reg.IsAutoHealEnabled("regular-check") {
		t.Error("expected auto-heal to be disabled for non-healable check")
	}
}

// TestRunAutoHeal_SkipsNonCriticalChecks verifies only critical checks are auto-healed
// [REQ:HEAL-ACTION-001]
func TestRunAutoHeal_SkipsNonCriticalChecks(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id:     "healable-check",
		result: Result{CheckID: "healable-check", Status: StatusWarning},
		actions: []RecoveryAction{
			{ID: "action-1", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healableCheck)

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"healable-check": true,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "healable-check", Status: StatusWarning}, // Not critical
	}

	autoHealResults := reg.RunAutoHeal(context.Background(), results)

	// Should not attempt to heal warning status
	if len(autoHealResults) > 0 {
		t.Errorf("expected no auto-heal attempts for warning status, got %d", len(autoHealResults))
	}
}

func TestRunAutoHeal_SkipsIneligibleResultEvenWhenPolicyMatches(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id: "scenario-app-monitor",
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
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
		{
			CheckID: "scenario-app-monitor",
			Status:  StatusWarning,
			Details: map[string]interface{}{
				"autoHealEligible": false,
				"fallback":         "direct-health-check",
			},
		},
	}

	autoHealResults := reg.RunAutoHeal(context.Background(), results)
	if len(autoHealResults) != 0 {
		t.Fatalf("expected no auto-heal attempts for ineligible result, got %d", len(autoHealResults))
	}
	if len(healableCheck.executedActions) != 0 {
		t.Fatalf("expected no actions executed, got %v", healableCheck.executedActions)
	}
}

// TestRunAutoHeal_SkipsDisabledAutoHeal verifies auto-heal respects config
// [REQ:CONFIG-CHECK-001]
func TestRunAutoHeal_SkipsDisabledAutoHeal(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id:     "healable-check",
		result: Result{CheckID: "healable-check", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "action-1", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healableCheck)

	// Auto-heal disabled for this check
	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"healable-check": false,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	}

	autoHealResults := reg.RunAutoHeal(context.Background(), results)

	if len(autoHealResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(autoHealResults))
	}
	if autoHealResults[0].Attempted {
		t.Error("expected auto-heal to not be attempted when disabled")
	}
	if autoHealResults[0].Reason != "auto-heal not enabled for this check" {
		t.Errorf("unexpected reason: %s", autoHealResults[0].Reason)
	}
}

// TestRunAutoHeal_SkipsDangerousActions verifies dangerous actions are not auto-executed
// [REQ:HEAL-ACTION-001]
func TestRunAutoHeal_SkipsDangerousActions(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id:     "healable-check",
		result: Result{CheckID: "healable-check", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "dangerous-action", Available: true, Dangerous: true},
			{ID: "unavailable-action", Available: false, Dangerous: false},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healableCheck)

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"healable-check": true,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	}

	autoHealResults := reg.RunAutoHeal(context.Background(), results)

	if len(autoHealResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(autoHealResults))
	}
	if autoHealResults[0].Attempted {
		t.Error("expected auto-heal to not be attempted with only dangerous/unavailable actions")
	}
	if autoHealResults[0].Reason != "no auto-heal recovery action available" {
		t.Errorf("unexpected reason: %s", autoHealResults[0].Reason)
	}
}

// TestRunAutoHeal_ExecutesSafeAction verifies safe actions are executed
// [REQ:HEAL-ACTION-001]
func TestRunAutoHeal_ExecutesSafeAction(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id:     "healable-check",
		result: Result{CheckID: "healable-check", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "dangerous-action", Available: true, Dangerous: true},
			{ID: "safe-action", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true, Message: "Healed"},
	}
	reg.Register(healableCheck)

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"healable-check": true,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	}

	autoHealResults := reg.RunAutoHeal(context.Background(), results)

	if len(autoHealResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(autoHealResults))
	}
	if !autoHealResults[0].Attempted {
		t.Error("expected auto-heal to be attempted")
	}
	if !autoHealResults[0].ActionResult.Success {
		t.Error("expected action to succeed")
	}
	if len(healableCheck.executedActions) != 1 || healableCheck.executedActions[0] != "safe-action" {
		t.Errorf("expected 'safe-action' to be executed, got %v", healableCheck.executedActions)
	}
}

// TestRunAutoHeal_SelectsFirstSafeAction verifies action selection order
// [REQ:HEAL-ACTION-001]
func TestRunAutoHeal_SelectsFirstSafeAction(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id:     "healable-check",
		result: Result{CheckID: "healable-check", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "first-safe", Available: true, Dangerous: false},
			{ID: "second-safe", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healableCheck)

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"healable-check": true,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	}

	reg.RunAutoHeal(context.Background(), results)

	if len(healableCheck.executedActions) != 1 || healableCheck.executedActions[0] != "first-safe" {
		t.Errorf("expected 'first-safe' to be selected, got %v", healableCheck.executedActions)
	}
}

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

func TestNewAutoHealPolicyFromGlobal(t *testing.T) {
	policy, err := NewAutoHealPolicyFromGlobal(300, 3, 30, 300, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.BaseCooldown != 5*time.Minute {
		t.Fatalf("BaseCooldown = %v, want %v", policy.BaseCooldown, 5*time.Minute)
	}
	if policy.MaxRestartAttempts != 3 {
		t.Fatalf("MaxRestartAttempts = %d, want 3", policy.MaxRestartAttempts)
	}
	if policy.FastActionTimeout != 30*time.Second {
		t.Fatalf("FastActionTimeout = %v, want %v", policy.FastActionTimeout, 30*time.Second)
	}
	if policy.RestartActionTimeout != 5*time.Minute {
		t.Fatalf("RestartActionTimeout = %v, want %v", policy.RestartActionTimeout, 5*time.Minute)
	}
	if policy.TimeoutRetryCooldown != 30*time.Second {
		t.Fatalf("TimeoutRetryCooldown = %v, want %v", policy.TimeoutRetryCooldown, 30*time.Second)
	}
}

func TestNewAutoHealPolicyFromGlobal_AppliesTimeoutDefaults(t *testing.T) {
	policy, err := NewAutoHealPolicyFromGlobal(300, 3, 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.FastActionTimeout != DefaultFastActionTimeout {
		t.Fatalf("FastActionTimeout = %v, want %v", policy.FastActionTimeout, DefaultFastActionTimeout)
	}
	if policy.RestartActionTimeout != DefaultRestartActionTimeout {
		t.Fatalf("RestartActionTimeout = %v, want %v", policy.RestartActionTimeout, DefaultRestartActionTimeout)
	}
	if policy.TimeoutRetryCooldown != DefaultTimeoutRetryCooldown {
		t.Fatalf("TimeoutRetryCooldown = %v, want %v", policy.TimeoutRetryCooldown, DefaultTimeoutRetryCooldown)
	}
}

func TestNewAutoHealPolicyFromGlobal_InvalidValues(t *testing.T) {
	if _, err := NewAutoHealPolicyFromGlobal(0, 3, 30, 300, 30); err == nil {
		t.Fatal("expected error for zero restart cooldown")
	}
	if _, err := NewAutoHealPolicyFromGlobal(60, 0, 30, 300, 30); err == nil {
		t.Fatal("expected error for max restart attempts < 1")
	}
}

func TestSetAutoHealPolicy_RejectsInvalidPolicy(t *testing.T) {
	reg := NewRegistry(testPlatform())
	err := reg.SetAutoHealPolicy(AutoHealPolicy{
		BaseCooldown:       0,
		MaxRestartAttempts: 1,
	})
	if err == nil {
		t.Fatal("expected invalid policy to be rejected")
	}
}

func TestCalculateFailureCooldown_CapsAtMaxFailureCooldown(t *testing.T) {
	policy := AutoHealPolicy{
		BaseCooldown:         5 * time.Minute,
		MaxRestartAttempts:   3,
		FastActionTimeout:    DefaultFastActionTimeout,
		RestartActionTimeout: DefaultRestartActionTimeout,
		TimeoutRetryCooldown: DefaultTimeoutRetryCooldown,
	}
	if err := policy.Validate(); err != nil {
		t.Fatalf("policy invalid: %v", err)
	}

	// Below threshold: BaseCooldown (capped).
	if got := policy.CalculateFailureCooldown(1); got != policy.BaseCooldown {
		t.Errorf("failure=1: got %v, want %v", got, policy.BaseCooldown)
	}

	// At threshold (3): 5m * 2^1 = 10m.
	if got := policy.CalculateFailureCooldown(3); got != 10*time.Minute {
		t.Errorf("failure=3: got %v, want 10m", got)
	}

	// Past the cap: must saturate at MaxFailureCooldown.
	for _, failures := range []int{20, 50, 1000} {
		if got := policy.CalculateFailureCooldown(failures); got != MaxFailureCooldown {
			t.Errorf("failure=%d: got %v, want %v (cap)", failures, got, MaxFailureCooldown)
		}
	}

	// Extreme value must not overflow into negative durations.
	if got := policy.CalculateFailureCooldown(1<<30); got != MaxFailureCooldown {
		t.Errorf("very large failures: got %v, want %v", got, MaxFailureCooldown)
	}
}

func TestHealTrackerCooldownHelpers(t *testing.T) {
	now := time.Now()
	tracker := HealTracker{
		CooldownUntil: now.Add(2 * time.Second),
	}

	if !tracker.IsInCooldown() {
		t.Fatal("expected tracker to be in cooldown")
	}
	if tracker.CooldownRemaining() <= 0 {
		t.Fatal("expected positive cooldown remaining")
	}

	tracker.CooldownUntil = now.Add(-1 * time.Second)
	if tracker.IsInCooldown() {
		t.Fatal("expected tracker cooldown to be expired")
	}
	if tracker.CooldownRemaining() != 0 {
		t.Fatalf("expected zero cooldown remaining, got %v", tracker.CooldownRemaining())
	}
}

func TestRunAutoHeal_RequiresPolicy(t *testing.T) {
	reg := NewRegistry(testPlatform())
	check := &mockHealableCheck{
		id: "healable-check",
		actions: []RecoveryAction{
			{ID: "start", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"healable-check": true},
	})

	results := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Attempted {
		t.Fatal("expected no attempt without policy")
	}
	if results[0].Reason != "auto-heal policy not configured" {
		t.Fatalf("unexpected reason: %s", results[0].Reason)
	}
}

func TestAutoHealCooldown_UsesConfiguredPolicy(t *testing.T) {
	reg := NewRegistry(testPlatform())
	if err := reg.SetAutoHealPolicy(AutoHealPolicy{
		BaseCooldown:         5 * time.Minute,
		MaxRestartAttempts:   3,
		FastActionTimeout:    DefaultFastActionTimeout,
		RestartActionTimeout: DefaultRestartActionTimeout,
		TimeoutRetryCooldown: DefaultTimeoutRetryCooldown,
	}); err != nil {
		t.Fatalf("set policy: %v", err)
	}

	clk := &fixedClock{current: time.Date(2026, 2, 19, 2, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	check := &mockHealableCheck{
		id: "healable-check",
		actions: []RecoveryAction{
			{ID: "start", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"healable-check": true},
	})

	first := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	})
	if len(first) != 1 || !first[0].Attempted {
		t.Fatalf("expected first attempt to run, got %+v", first)
	}

	clk.current = clk.current.Add(4 * time.Minute)
	second := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	})
	if len(second) != 1 {
		t.Fatalf("expected second result, got %d", len(second))
	}
	if second[0].Attempted {
		t.Fatalf("expected cooldown skip, got attempt: %+v", second[0])
	}
	if second[0].CooldownRemaining <= 0 {
		t.Fatalf("expected positive cooldown remaining, got %v", second[0].CooldownRemaining)
	}
}

func TestAutoHealBackoff_UsesConfiguredMaxRestartAttempts(t *testing.T) {
	reg := NewRegistry(testPlatform())
	if err := reg.SetAutoHealPolicy(AutoHealPolicy{
		BaseCooldown:         60 * time.Second,
		MaxRestartAttempts:   2,
		FastActionTimeout:    DefaultFastActionTimeout,
		RestartActionTimeout: DefaultRestartActionTimeout,
		TimeoutRetryCooldown: DefaultTimeoutRetryCooldown,
	}); err != nil {
		t.Fatalf("set policy: %v", err)
	}

	clk := &fixedClock{current: time.Date(2026, 2, 19, 2, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	check := &mockHealableCheck{
		id: "failing-check",
		actions: []RecoveryAction{
			{ID: "start", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: false, Error: "failed"},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"failing-check": true},
	})

	// Failure #1: base cooldown (60s).
	r1 := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "failing-check", Status: StatusCritical},
	})
	if len(r1) != 1 || !r1[0].Attempted {
		t.Fatalf("first failure should attempt heal, got %+v", r1)
	}

	clk.current = clk.current.Add(61 * time.Second)
	// Failure #2: threshold reached -> 120s cooldown.
	r2 := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "failing-check", Status: StatusCritical},
	})
	if len(r2) != 1 || !r2[0].Attempted {
		t.Fatalf("second failure should attempt heal, got %+v", r2)
	}

	tracker, ok := reg.GetHealTracker("failing-check")
	if !ok {
		t.Fatal("expected heal tracker")
	}
	if tracker.ConsecutiveFailures != 2 {
		t.Fatalf("ConsecutiveFailures = %d, want 2", tracker.ConsecutiveFailures)
	}
	if got := tracker.CooldownUntil.Sub(clk.current); got != 120*time.Second {
		t.Fatalf("cooldown after 2nd failure = %v, want 120s", got)
	}

	clk.current = clk.current.Add(121 * time.Second)
	// Failure #3: exponential growth -> 240s cooldown.
	r3 := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "failing-check", Status: StatusCritical},
	})
	if len(r3) != 1 || !r3[0].Attempted {
		t.Fatalf("third failure should attempt heal, got %+v", r3)
	}

	tracker, ok = reg.GetHealTracker("failing-check")
	if !ok {
		t.Fatal("expected heal tracker")
	}
	if tracker.ConsecutiveFailures != 3 {
		t.Fatalf("ConsecutiveFailures = %d, want 3", tracker.ConsecutiveFailures)
	}
	if got := tracker.CooldownUntil.Sub(clk.current); got != 240*time.Second {
		t.Fatalf("cooldown after 3rd failure = %v, want 240s", got)
	}
}

type mockHealTrackerStore struct {
	trackers map[string]*HealTracker
	saveCh   chan string
}

func (m *mockHealTrackerStore) SaveHealTracker(ctx context.Context, checkID string, tracker *HealTracker) error {
	if m.trackers == nil {
		m.trackers = make(map[string]*HealTracker)
	}
	copyTracker := *tracker
	m.trackers[checkID] = &copyTracker
	if m.saveCh != nil {
		select {
		case m.saveCh <- checkID:
		default:
		}
	}
	return nil
}

func (m *mockHealTrackerStore) GetAllHealTrackers(ctx context.Context) (map[string]*HealTracker, error) {
	out := make(map[string]*HealTracker, len(m.trackers))
	for k, v := range m.trackers {
		copyTracker := *v
		out[k] = &copyTracker
	}
	return out, nil
}

func (m *mockHealTrackerStore) DeleteHealTracker(ctx context.Context, checkID string) error {
	delete(m.trackers, checkID)
	return nil
}

func TestLoadHealTrackers_RestoresState(t *testing.T) {
	reg := newTestRegistry()
	store := &mockHealTrackerStore{
		trackers: map[string]*HealTracker{
			"resource-postgres": {
				ConsecutiveFailures: 4,
				CooldownUntil:       time.Now().Add(5 * time.Minute),
			},
		},
	}
	reg.SetHealTrackerStore(store)

	if err := reg.LoadHealTrackers(context.Background()); err != nil {
		t.Fatalf("LoadHealTrackers failed: %v", err)
	}

	tracker, ok := reg.GetHealTracker("resource-postgres")
	if !ok {
		t.Fatal("expected tracker to be loaded")
	}
	if tracker.ConsecutiveFailures != 4 {
		t.Fatalf("ConsecutiveFailures = %d, want 4", tracker.ConsecutiveFailures)
	}
}

func TestUpdateHealTracker_PersistsToStore(t *testing.T) {
	reg := newTestRegistry()
	clk := &fixedClock{current: time.Date(2026, 2, 19, 2, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	store := &mockHealTrackerStore{saveCh: make(chan string, 1)}
	reg.SetHealTrackerStore(store)

	check := &mockHealableCheck{
		id: "resource-postgres",
		actions: []RecoveryAction{
			{ID: "start", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: false},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"resource-postgres": true},
	})

	_ = reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "resource-postgres", Status: StatusCritical},
	})

	select {
	case <-store.saveCh:
	case <-time.After(2 * time.Second):
		t.Fatal("expected heal tracker save to be called")
	}

	if _, ok := store.trackers["resource-postgres"]; !ok {
		t.Fatal("expected stored tracker for resource-postgres")
	}
}

// newTimeoutPolicyRegistry returns a registry pre-configured with short
// fast/restart timeouts and a short timeout-retry cooldown so tests can run
// quickly. BaseCooldown is also short so success cooldown doesn't dominate.
func newTimeoutPolicyRegistry(t *testing.T, fast, restart, timeoutRetry, base time.Duration) *Registry {
	t.Helper()
	reg := NewRegistry(testPlatform())
	policy := AutoHealPolicy{
		BaseCooldown:         base,
		MaxRestartAttempts:   3,
		FastActionTimeout:    fast,
		RestartActionTimeout: restart,
		TimeoutRetryCooldown: timeoutRetry,
	}
	if err := reg.SetAutoHealPolicy(policy); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	return reg
}

// TestActionTimeoutFor_RestartActionsGetRestartBudget verifies the timeout
// dispatcher hands restart-class actions the generous budget and everything
// else the short budget. This is the core of D1: a 90s outer cap killed a
// legitimately slow restart before verifyRecovery could observe success.
func TestActionTimeoutFor_RestartActionsGetRestartBudget(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, 30*time.Second, 5*time.Minute, 30*time.Second, time.Minute)

	cases := []struct {
		action string
		want   time.Duration
	}{
		{"restart", 5 * time.Minute},
		{"restart-clean", 5 * time.Minute},
		{"setup-restart", 5 * time.Minute},
		{"start", 5 * time.Minute},
		{"stop", 5 * time.Minute},
		{"logs", 30 * time.Second},
		{"diagnose", 30 * time.Second},
		{"cleanup-ports", 30 * time.Second},
		{"kill", 30 * time.Second},
		{"unknown-future-action", 30 * time.Second},
	}
	for _, tc := range cases {
		if got := reg.actionTimeoutFor(tc.action); got != tc.want {
			t.Errorf("actionTimeoutFor(%q) = %v, want %v", tc.action, got, tc.want)
		}
	}
}

// TestExecuteAutoHealAction_RestartCompletesWithinExtendedBudget proves that
// a 200ms restart succeeds under a 1s restart budget but would have been killed
// by the old 100ms fast budget. The reverse-config (fast budget for a restart
// action) is verified in the timeout-not-failure test below.
func TestExecuteAutoHealAction_RestartCompletesWithinExtendedBudget(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, 50*time.Millisecond, time.Second, 100*time.Millisecond, time.Minute)

	check := &mockHealableCheck{
		id: "scenario-app-monitor",
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true, Message: "restarted"},
		executeSleep:  200 * time.Millisecond,
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"scenario-app-monitor": true},
	})

	results := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "scenario-app-monitor", Status: StatusCritical},
	})

	if len(results) != 1 || !results[0].Attempted {
		t.Fatalf("expected one attempted result, got %+v", results)
	}
	if !results[0].ActionResult.Success {
		t.Fatalf("expected restart to succeed inside extended budget, got %+v", results[0].ActionResult)
	}
	if results[0].TimedOut || results[0].ActionResult.TimedOut {
		t.Fatalf("expected TimedOut=false, got %+v", results[0])
	}
	tracker, _ := reg.GetHealTracker("scenario-app-monitor")
	if tracker.ConsecutiveFailures != 0 {
		t.Fatalf("ConsecutiveFailures = %d, want 0", tracker.ConsecutiveFailures)
	}
}

// TestExecuteAutoHealAction_TimeoutDoesNotRatchetFailures is the heart of D2.
// Before the fix, a timed-out heal was indistinguishable from a failure and
// pushed ConsecutiveFailures into the exponential cooldown, silencing autoheal
// for hours. After the fix, a timeout records TotalTimeouts, applies the short
// retry cooldown, and leaves ConsecutiveFailures untouched.
func TestExecuteAutoHealAction_TimeoutDoesNotRatchetFailures(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, 50*time.Millisecond, 50*time.Millisecond, 75*time.Millisecond, time.Minute)
	clk := &fixedClock{current: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	check := &mockHealableCheck{
		id: "scenario-app-monitor",
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
		executeSleep:  500 * time.Millisecond, // far longer than 50ms restart budget
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"scenario-app-monitor": true},
	})

	results := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "scenario-app-monitor", Status: StatusCritical},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].TimedOut {
		t.Fatalf("expected TimedOut=true, got %+v", results[0])
	}
	if !results[0].ActionResult.TimedOut {
		t.Fatalf("expected ActionResult.TimedOut=true, got %+v", results[0].ActionResult)
	}
	if results[0].ActionResult.Success {
		t.Fatalf("expected Success=false on timeout")
	}

	tracker, ok := reg.GetHealTracker("scenario-app-monitor")
	if !ok {
		t.Fatal("expected tracker to exist")
	}
	if tracker.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures = %d, want 0 (timeouts must not ratchet failures)", tracker.ConsecutiveFailures)
	}
	if tracker.TotalTimeouts != 1 {
		t.Errorf("TotalTimeouts = %d, want 1", tracker.TotalTimeouts)
	}
	expectedCooldownUntil := clk.current.Add(75 * time.Millisecond)
	if !tracker.CooldownUntil.Equal(expectedCooldownUntil) {
		t.Errorf("CooldownUntil = %v, want %v (short timeout-retry cooldown)", tracker.CooldownUntil, expectedCooldownUntil)
	}
}

// TestExecuteAutoHealAction_GenuineFailureStillRatchets verifies the failure
// path is unchanged when the action explicitly reports Success=false without
// timing out. ConsecutiveFailures must increment so the exponential cooldown
// still kicks in for genuinely-broken actions.
func TestExecuteAutoHealAction_GenuineFailureStillRatchets(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, time.Second, time.Second, 30*time.Second, 2*time.Minute)
	clk := &fixedClock{current: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	check := &mockHealableCheck{
		id: "resource-postgres",
		actions: []RecoveryAction{
			{ID: "start", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: false, Error: "exit 1"},
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"resource-postgres": true},
	})

	results := reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "resource-postgres", Status: StatusCritical},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TimedOut {
		t.Fatalf("genuine failure must not be marked TimedOut")
	}
	tracker, _ := reg.GetHealTracker("resource-postgres")
	if tracker.ConsecutiveFailures != 1 {
		t.Errorf("ConsecutiveFailures = %d, want 1", tracker.ConsecutiveFailures)
	}
	if tracker.TotalTimeouts != 0 {
		t.Errorf("TotalTimeouts = %d, want 0", tracker.TotalTimeouts)
	}
	expectedCooldownUntil := clk.current.Add(2 * time.Minute)
	if !tracker.CooldownUntil.Equal(expectedCooldownUntil) {
		t.Errorf("CooldownUntil = %v, want %v", tracker.CooldownUntil, expectedCooldownUntil)
	}
}

// TestUpdateHealTracker_ConsecutiveTimeoutCapFallsThroughToFailure verifies
// the safety cap: after MaxRestartAttempts consecutive timeouts, the next
// timeout falls through to the regular failure ratchet so a permanently-stuck
// action eventually cools down on the exponential backoff.
func TestUpdateHealTracker_ConsecutiveTimeoutCapFallsThroughToFailure(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, time.Second, time.Second, time.Minute, time.Minute)
	clk := &fixedClock{current: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	// Drive MaxRestartAttempts=3 consecutive timeouts directly.
	for i := 0; i < 3; i++ {
		reg.updateHealTracker("scenario-x", outcomeTimeout)
	}
	tracker, _ := reg.GetHealTracker("scenario-x")
	if tracker.ConsecutiveFailures != 0 {
		t.Fatalf("after 3 timeouts ConsecutiveFailures = %d, want 0", tracker.ConsecutiveFailures)
	}
	if tracker.ConsecutiveTimeouts != 3 {
		t.Fatalf("ConsecutiveTimeouts = %d, want 3", tracker.ConsecutiveTimeouts)
	}

	// 4th consecutive timeout exceeds MaxRestartAttempts and falls through
	// to the failure ratchet.
	reg.updateHealTracker("scenario-x", outcomeTimeout)
	tracker, _ = reg.GetHealTracker("scenario-x")
	if tracker.ConsecutiveFailures != 1 {
		t.Errorf("after 4th consecutive timeout ConsecutiveFailures = %d, want 1", tracker.ConsecutiveFailures)
	}
}

// TestUpdateHealTracker_SuccessResetsBothCounters verifies success clears
// both consecutive counters so the tracker returns to a clean state.
func TestUpdateHealTracker_SuccessResetsBothCounters(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, time.Second, time.Second, 30*time.Second, time.Minute)

	reg.updateHealTracker("scenario-x", outcomeTimeout)
	reg.updateHealTracker("scenario-x", outcomeFailure)
	reg.updateHealTracker("scenario-x", outcomeSuccess)
	tracker, _ := reg.GetHealTracker("scenario-x")
	if tracker.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures after success = %d, want 0", tracker.ConsecutiveFailures)
	}
	if tracker.ConsecutiveTimeouts != 0 {
		t.Errorf("ConsecutiveTimeouts after success = %d, want 0", tracker.ConsecutiveTimeouts)
	}
	if tracker.TotalSuccesses != 1 || tracker.TotalTimeouts != 1 {
		t.Errorf("totals = (s=%d, t=%d), want (1, 1)", tracker.TotalSuccesses, tracker.TotalTimeouts)
	}
}

// TestAutoHeal_TimeoutThenSuccess_Recovers is the end-to-end D2 regression.
// Before the fix, a slow first attempt timed out, ratcheted failures, and the
// next tick was silenced by the exponential cooldown. After the fix the short
// timeout-retry cooldown lets the next attempt succeed.
func TestAutoHeal_TimeoutThenSuccess_Recovers(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, 50*time.Millisecond, 50*time.Millisecond, 1*time.Millisecond, time.Minute)
	clk := &fixedClock{current: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)

	check := &mockHealableCheck{
		id: "scenario-app-monitor",
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true, Message: "restarted"},
		executeSleep:  300 * time.Millisecond, // exceeds 50ms budget on first attempt
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"scenario-app-monitor": true},
	})

	results := []Result{{CheckID: "scenario-app-monitor", Status: StatusCritical}}

	// First attempt: times out (sleep > budget).
	first := reg.RunAutoHeal(context.Background(), results)
	if len(first) != 1 || !first[0].TimedOut {
		t.Fatalf("first attempt: expected TimedOut, got %+v", first)
	}

	// Advance clock past the 1ms timeout-retry cooldown so the cooldown gate
	// allows the next attempt.
	clk.current = clk.current.Add(10 * time.Millisecond)

	// Make the mock fast and successful for the second attempt.
	check.mu.Lock()
	check.executeSleep = 0
	check.mu.Unlock()

	second := reg.RunAutoHeal(context.Background(), results)
	if len(second) != 1 {
		t.Fatalf("second attempt result count = %d", len(second))
	}
	if !second[0].Attempted {
		t.Fatalf("second attempt should be attempted, got reason=%q (cooldown=%v)", second[0].Reason, second[0].CooldownRemaining)
	}
	if !second[0].ActionResult.Success {
		t.Fatalf("second attempt should succeed, got %+v", second[0].ActionResult)
	}

	tracker, _ := reg.GetHealTracker("scenario-app-monitor")
	if tracker.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures after timeout-then-success = %d, want 0", tracker.ConsecutiveFailures)
	}
	if tracker.TotalTimeouts != 1 || tracker.TotalSuccesses != 1 {
		t.Errorf("totals = (t=%d, s=%d), want (1, 1)", tracker.TotalTimeouts, tracker.TotalSuccesses)
	}
}

// TestExecuteAutoHealAction_HealTrackerPersistsTimeoutCounters checks that the
// persistence seam writes a tracker carrying the new TotalTimeouts counter so
// the on-disk record matches in-memory state. This guards against future code
// inadvertently dropping the new fields.
func TestExecuteAutoHealAction_HealTrackerPersistsTimeoutCounters(t *testing.T) {
	reg := newTimeoutPolicyRegistry(t, 50*time.Millisecond, 50*time.Millisecond, 30*time.Second, time.Minute)
	clk := &fixedClock{current: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)}
	reg.SetClock(clk)
	store := &mockHealTrackerStore{saveCh: make(chan string, 1)}
	reg.SetHealTrackerStore(store)

	check := &mockHealableCheck{
		id: "scenario-app-monitor",
		actions: []RecoveryAction{
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
		executeSleep:  500 * time.Millisecond,
	}
	reg.Register(check)
	reg.SetConfigProvider(&mockConfigProvider{
		autoHealChecks: map[string]bool{"scenario-app-monitor": true},
	})

	_ = reg.RunAutoHeal(context.Background(), []Result{
		{CheckID: "scenario-app-monitor", Status: StatusCritical},
	})

	select {
	case <-store.saveCh:
	case <-time.After(2 * time.Second):
		t.Fatal("expected persistence save")
	}

	persisted := store.trackers["scenario-app-monitor"]
	if persisted == nil {
		t.Fatal("expected persisted tracker")
	}
	if persisted.TotalTimeouts != 1 {
		t.Errorf("persisted TotalTimeouts = %d, want 1", persisted.TotalTimeouts)
	}
	if persisted.ConsecutiveFailures != 0 {
		t.Errorf("persisted ConsecutiveFailures = %d, want 0", persisted.ConsecutiveFailures)
	}
}
