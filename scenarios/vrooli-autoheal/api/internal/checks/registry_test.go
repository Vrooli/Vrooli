// Package checks tests for Registry
// [REQ:HEALTH-REGISTRY-001] [REQ:HEALTH-REGISTRY-002] [REQ:HEALTH-REGISTRY-003] [REQ:HEALTH-REGISTRY-004]
package checks

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// TestSeedStartupJitterSpreadsFirstRuns verifies per-check startup jitter:
// each cold-start check's lastRun lands in [now-interval, now) so its first
// run is staggered across (now, now+interval], and checks already seeded from
// persistence are left untouched.
func TestSeedStartupJitterSpreadsFirstRuns(t *testing.T) {
	reg := NewRegistry(testPlatform())
	const interval = 300
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		reg.Register(&mockCheck{id: id, interval: interval})
	}
	// Simulate a check restored from persistence: it already has a lastRun and
	// must NOT be re-jittered.
	restored := &mockCheck{id: "restored", interval: interval}
	reg.Register(restored)
	restoredAt := time.Date(2026, 6, 24, 11, 0, 0, 0, time.UTC)
	reg.SetResult(Result{CheckID: "restored", Timestamp: restoredAt})

	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	reg.SeedStartupJitter(now, rand.New(rand.NewSource(1)))

	intervalDur := time.Duration(interval) * time.Second
	offsets := map[time.Duration]struct{}{}
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		lastRun, ok := reg.lastRun[id]
		if !ok {
			t.Fatalf("check %q has no seeded lastRun", id)
		}
		offset := now.Sub(lastRun)
		if offset < 0 || offset >= intervalDur {
			t.Errorf("check %q offset = %s, want within [0, %s)", id, offset, intervalDur)
		}
		offsets[offset] = struct{}{}
	}
	if len(offsets) < 2 {
		t.Errorf("jitter produced %d distinct offsets, want spread (>=2)", len(offsets))
	}

	// The restored check keeps its persisted lastRun (not jittered).
	if got := reg.lastRun["restored"]; !got.Equal(restoredAt) {
		t.Errorf("restored check lastRun = %s, want untouched %s", got, restoredAt)
	}
}

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

// TestPlatformFiltering verifies platform-based checks are represented honestly
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

	// Should run all-platforms and current-platform, and represent other-platform as N/A.
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
	if !foundOtherPlatform {
		t.Errorf("other-platform check was omitted on %s", currentPlatform)
	}
	for _, result := range results {
		if result.CheckID == "other-platform" && result.Status != StatusNotApplicable {
			t.Errorf("other-platform status = %s, want %s", result.Status, StatusNotApplicable)
		}
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
