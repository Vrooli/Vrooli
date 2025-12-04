// Package bootstrap tests for full-stack integration
// [REQ:HEALTH-REGISTRY-001] [REQ:HEAL-ACTION-001] [REQ:PERSIST-STORE-001]
// Full-stack integration test: tick → persist → autoheal cycle
package bootstrap

import (
	"context"
	"sync"
	"testing"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// =============================================================================
// Mock Infrastructure for Full-Stack Testing
// =============================================================================

// mockStore implements ResultLoader and ResultSaver for testing
type mockStore struct {
	mu           sync.Mutex
	results      []checks.Result
	actionLogs   []mockActionLog
	saveErrors   map[string]error
	loadResults  []checks.Result
	loadError    error
}

type mockActionLog struct {
	CheckID    string
	ActionID   string
	Success    bool
	Message    string
	Output     string
	Error      string
	DurationMs int64
	Timestamp  time.Time
}

func newMockStore() *mockStore {
	return &mockStore{
		results:    make([]checks.Result, 0),
		actionLogs: make([]mockActionLog, 0),
		saveErrors: make(map[string]error),
	}
}

func (m *mockStore) SaveResult(ctx context.Context, result checks.Result) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err, ok := m.saveErrors[result.CheckID]; ok {
		return err
	}

	m.results = append(m.results, result)
	return nil
}

func (m *mockStore) GetLatestResultPerCheck(ctx context.Context) ([]checks.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.loadError != nil {
		return nil, m.loadError
	}
	return m.loadResults, nil
}

func (m *mockStore) SaveActionLog(checkID, actionID string, success bool, message, output, errMsg string, durationMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.actionLogs = append(m.actionLogs, mockActionLog{
		CheckID:    checkID,
		ActionID:   actionID,
		Success:    success,
		Message:    message,
		Output:     output,
		Error:      errMsg,
		DurationMs: durationMs,
		Timestamp:  time.Now(),
	})
}

func (m *mockStore) GetResults() []checks.Result {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.results
}

func (m *mockStore) GetActionLogs() []mockActionLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.actionLogs
}

// mockConfigProvider implements checks.ConfigProvider for testing
type mockConfigProvider struct {
	mu              sync.Mutex
	enabledChecks   map[string]bool
	autoHealChecks  map[string]bool
}

func newMockConfigProvider() *mockConfigProvider {
	return &mockConfigProvider{
		enabledChecks:  make(map[string]bool),
		autoHealChecks: make(map[string]bool),
	}
}

func (m *mockConfigProvider) IsCheckEnabled(checkID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	enabled, exists := m.enabledChecks[checkID]
	return !exists || enabled
}

func (m *mockConfigProvider) IsAutoHealEnabled(checkID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.autoHealChecks[checkID]
}

func (m *mockConfigProvider) EnableAutoHeal(checkID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabledChecks[checkID] = true
	m.autoHealChecks[checkID] = true
}

func (m *mockConfigProvider) DisableAutoHeal(checkID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabledChecks[checkID] = true
	m.autoHealChecks[checkID] = false
}

// mockHealableCheck implements checks.HealableCheck for testing
type mockHealableCheck struct {
	id              string
	status          checks.Status
	recoveryActions []checks.RecoveryAction
	executeResult   checks.ActionResult
	runCount        int
	executedActions []string
	mu              sync.Mutex
}

func (c *mockHealableCheck) ID() string                         { return c.id }
func (c *mockHealableCheck) Title() string                      { return "Mock Healable Check " + c.id }
func (c *mockHealableCheck) Description() string                { return "A mock healable check for integration testing" }
func (c *mockHealableCheck) Importance() string                 { return "Required for full-stack testing" }
func (c *mockHealableCheck) Category() checks.Category          { return checks.CategoryInfrastructure }
func (c *mockHealableCheck) IntervalSeconds() int               { return 60 }
func (c *mockHealableCheck) Platforms() []platform.Type         { return nil }

func (c *mockHealableCheck) Run(ctx context.Context) checks.Result {
	c.mu.Lock()
	c.runCount++
	runCount := c.runCount
	status := c.status
	c.mu.Unlock()

	return checks.Result{
		CheckID:   c.id,
		Status:    status,
		Message:   "Mock result from run " + string(rune('0'+runCount%10)),
		Timestamp: time.Now(),
		Duration:  10 * time.Millisecond,
		Details: map[string]interface{}{
			"runCount": runCount,
			"mockData": true,
		},
	}
}

func (c *mockHealableCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	return c.recoveryActions
}

func (c *mockHealableCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	c.mu.Lock()
	c.executedActions = append(c.executedActions, actionID)
	result := c.executeResult
	c.mu.Unlock()

	result.ActionID = actionID
	result.CheckID = c.id
	result.Timestamp = time.Now()
	result.Duration = 50 * time.Millisecond
	return result
}

func (c *mockHealableCheck) SetStatus(status checks.Status) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = status
}

func (c *mockHealableCheck) GetExecutedActions() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.executedActions
}

// =============================================================================
// Full-Stack Integration Tests
// =============================================================================

// TestFullStack_TickPersistAutohealCycle tests the complete cycle:
// 1. Register checks
// 2. Run tick (all checks execute)
// 3. Persist results
// 4. Auto-heal critical checks
// 5. Verify all stages
func TestFullStack_TickPersistAutohealCycle(t *testing.T) {
	// Setup
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	store := newMockStore()
	config := newMockConfigProvider()
	registry.SetConfigProvider(config)

	// Create checks: 1 healthy, 1 warning, 1 critical (healable)
	healthyCheck := &mockHealableCheck{
		id:     "healthy-check",
		status: checks.StatusOK,
		recoveryActions: []checks.RecoveryAction{
			{ID: "restart", Name: "Restart", Available: true, Dangerous: false},
		},
		executeResult: checks.ActionResult{Success: true},
	}

	warningCheck := &mockHealableCheck{
		id:     "warning-check",
		status: checks.StatusWarning,
		recoveryActions: []checks.RecoveryAction{
			{ID: "fix", Name: "Fix", Available: true, Dangerous: false},
		},
		executeResult: checks.ActionResult{Success: true},
	}

	criticalCheck := &mockHealableCheck{
		id:     "critical-check",
		status: checks.StatusCritical,
		recoveryActions: []checks.RecoveryAction{
			{ID: "start", Name: "Start", Available: true, Dangerous: false},
			{ID: "restart", Name: "Restart", Available: true, Dangerous: true},
		},
		executeResult: checks.ActionResult{Success: true, Message: "Service started"},
	}

	// Register all checks
	registry.Register(healthyCheck)
	registry.Register(warningCheck)
	registry.Register(criticalCheck)

	// Enable auto-heal for critical check
	config.EnableAutoHeal("critical-check")

	ctx := context.Background()

	// === STAGE 1: Run Tick (execute all checks) ===
	t.Run("tick_executes_all_checks", func(t *testing.T) {
		results := registry.RunAll(ctx, true)

		if len(results) != 3 {
			t.Errorf("expected 3 results, got %d", len(results))
		}

		// Verify each check ran
		foundStatuses := make(map[string]checks.Status)
		for _, r := range results {
			foundStatuses[r.CheckID] = r.Status
		}

		if foundStatuses["healthy-check"] != checks.StatusOK {
			t.Error("healthy check should be OK")
		}
		if foundStatuses["warning-check"] != checks.StatusWarning {
			t.Error("warning check should be Warning")
		}
		if foundStatuses["critical-check"] != checks.StatusCritical {
			t.Error("critical check should be Critical")
		}

		// === STAGE 2: Persist Results ===
		for _, result := range results {
			if err := store.SaveResult(ctx, result); err != nil {
				t.Errorf("failed to save result for %s: %v", result.CheckID, err)
			}
		}

		storedResults := store.GetResults()
		if len(storedResults) != 3 {
			t.Errorf("expected 3 stored results, got %d", len(storedResults))
		}
	})

	// === STAGE 3: Auto-Heal Critical Checks ===
	t.Run("autoheal_critical_checks", func(t *testing.T) {
		// Get current results
		currentResults := registry.GetAllResults()

		// Run auto-heal
		autoHealResults := registry.RunAutoHeal(ctx, currentResults)

		// Verify auto-heal behavior
		var criticalHealResult *checks.AutoHealResult
		for i := range autoHealResults {
			if autoHealResults[i].CheckID == "critical-check" {
				criticalHealResult = &autoHealResults[i]
				break
			}
		}

		if criticalHealResult == nil {
			t.Fatal("expected auto-heal result for critical-check")
		}

		if !criticalHealResult.Attempted {
			t.Error("auto-heal should have been attempted for critical check")
		}

		// Should have selected the safe action "start"
		if criticalHealResult.ActionResult.ActionID != "start" {
			t.Errorf("expected 'start' action, got %q", criticalHealResult.ActionResult.ActionID)
		}

		if !criticalHealResult.ActionResult.Success {
			t.Error("action should have succeeded")
		}

		// Log the action
		store.SaveActionLog(
			criticalHealResult.CheckID,
			criticalHealResult.ActionResult.ActionID,
			criticalHealResult.ActionResult.Success,
			criticalHealResult.ActionResult.Message,
			criticalHealResult.ActionResult.Output,
			criticalHealResult.ActionResult.Error,
			criticalHealResult.ActionResult.Duration.Milliseconds(),
		)

		actionLogs := store.GetActionLogs()
		if len(actionLogs) != 1 {
			t.Errorf("expected 1 action log, got %d", len(actionLogs))
		}
	})

	// === STAGE 4: Verify Full Cycle State ===
	t.Run("verify_final_state", func(t *testing.T) {
		// Verify executed actions on critical check
		executedActions := criticalCheck.GetExecutedActions()
		if len(executedActions) != 1 {
			t.Errorf("expected 1 executed action, got %d", len(executedActions))
		}
		if len(executedActions) > 0 && executedActions[0] != "start" {
			t.Errorf("expected 'start' action executed, got %q", executedActions[0])
		}

		// Healthy and warning checks should NOT have any actions executed
		healthyActions := healthyCheck.GetExecutedActions()
		if len(healthyActions) != 0 {
			t.Errorf("healthy check should have 0 executed actions, got %d", len(healthyActions))
		}

		warningActions := warningCheck.GetExecutedActions()
		if len(warningActions) != 0 {
			t.Errorf("warning check should have 0 executed actions, got %d", len(warningActions))
		}
	})
}

// TestFullStack_PopulateFromPersistence tests loading persisted state on startup
func TestFullStack_PopulateFromPersistence(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	store := newMockStore()

	// Pre-populate store with some results
	store.loadResults = []checks.Result{
		{
			CheckID:   "check-a",
			Status:    checks.StatusOK,
			Message:   "Previously stored OK",
			Timestamp: time.Now().Add(-5 * time.Minute),
		},
		{
			CheckID:   "check-b",
			Status:    checks.StatusWarning,
			Message:   "Previously stored Warning",
			Timestamp: time.Now().Add(-3 * time.Minute),
		},
	}

	// Register checks (they must exist to receive results)
	checkA := &mockHealableCheck{id: "check-a", status: checks.StatusOK}
	checkB := &mockHealableCheck{id: "check-b", status: checks.StatusWarning}
	registry.Register(checkA)
	registry.Register(checkB)

	ctx := context.Background()

	// Populate from persistence
	err := PopulateRecentResults(ctx, registry, store)
	if err != nil {
		t.Fatalf("PopulateRecentResults failed: %v", err)
	}

	// Verify results were loaded
	resultA, existsA := registry.GetResult("check-a")
	if !existsA {
		t.Error("check-a result should exist after population")
	}
	if resultA.Status != checks.StatusOK {
		t.Errorf("check-a status = %v, want OK", resultA.Status)
	}

	resultB, existsB := registry.GetResult("check-b")
	if !existsB {
		t.Error("check-b result should exist after population")
	}
	if resultB.Status != checks.StatusWarning {
		t.Errorf("check-b status = %v, want Warning", resultB.Status)
	}
}

// TestFullStack_MultipleTickCycles tests multiple consecutive tick cycles
func TestFullStack_MultipleTickCycles(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	store := newMockStore()
	config := newMockConfigProvider()
	registry.SetConfigProvider(config)

	// Create a check that changes status
	dynamicCheck := &mockHealableCheck{
		id:     "dynamic-check",
		status: checks.StatusOK,
		recoveryActions: []checks.RecoveryAction{
			{ID: "fix", Name: "Fix", Available: true, Dangerous: false},
		},
		executeResult: checks.ActionResult{Success: true},
	}
	registry.Register(dynamicCheck)
	config.EnableAutoHeal("dynamic-check")

	ctx := context.Background()

	// Cycle 1: Healthy
	t.Run("cycle1_healthy", func(t *testing.T) {
		results := registry.RunAll(ctx, true)
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Status != checks.StatusOK {
			t.Errorf("expected OK status, got %v", results[0].Status)
		}

		for _, r := range results {
			store.SaveResult(ctx, r)
		}

		// No auto-heal needed
		autoHealResults := registry.RunAutoHeal(ctx, results)
		for _, ahr := range autoHealResults {
			if ahr.Attempted {
				t.Error("no auto-heal should be attempted for OK status")
			}
		}
	})

	// Cycle 2: Goes critical
	t.Run("cycle2_critical", func(t *testing.T) {
		dynamicCheck.SetStatus(checks.StatusCritical)

		results := registry.RunAll(ctx, true)
		if results[0].Status != checks.StatusCritical {
			t.Errorf("expected Critical status, got %v", results[0].Status)
		}

		for _, r := range results {
			store.SaveResult(ctx, r)
		}

		// Auto-heal should trigger
		autoHealResults := registry.RunAutoHeal(ctx, results)
		var healResult *checks.AutoHealResult
		for i := range autoHealResults {
			if autoHealResults[i].CheckID == "dynamic-check" {
				healResult = &autoHealResults[i]
				break
			}
		}

		if healResult == nil || !healResult.Attempted {
			t.Error("auto-heal should have been attempted")
		}
	})

	// Cycle 3: Recovers to OK
	t.Run("cycle3_recovered", func(t *testing.T) {
		dynamicCheck.SetStatus(checks.StatusOK)

		results := registry.RunAll(ctx, true)
		if results[0].Status != checks.StatusOK {
			t.Errorf("expected OK status, got %v", results[0].Status)
		}

		// Persist result
		for _, r := range results {
			store.SaveResult(ctx, r)
		}

		// No auto-heal needed
		autoHealResults := registry.RunAutoHeal(ctx, results)
		for _, ahr := range autoHealResults {
			if ahr.Attempted {
				t.Error("no auto-heal should be attempted for OK status")
			}
		}
	})

	// Verify total stored results (3 cycles × 1 check = 3 results)
	allResults := store.GetResults()
	if len(allResults) != 3 {
		t.Errorf("expected 3 stored results total, got %d", len(allResults))
	}
}

// TestFullStack_DangerousActionsNotAutoExecuted tests that dangerous actions are skipped
func TestFullStack_DangerousActionsNotAutoExecuted(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	config := newMockConfigProvider()
	registry.SetConfigProvider(config)

	// Create a check with ONLY dangerous actions
	dangerousCheck := &mockHealableCheck{
		id:     "dangerous-only",
		status: checks.StatusCritical,
		recoveryActions: []checks.RecoveryAction{
			{ID: "force-restart", Name: "Force Restart", Available: true, Dangerous: true},
			{ID: "delete-data", Name: "Delete Data", Available: true, Dangerous: true},
		},
		executeResult: checks.ActionResult{Success: true},
	}
	registry.Register(dangerousCheck)
	config.EnableAutoHeal("dangerous-only")

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Find result for our check
	var healResult *checks.AutoHealResult
	for i := range autoHealResults {
		if autoHealResults[i].CheckID == "dangerous-only" {
			healResult = &autoHealResults[i]
			break
		}
	}

	if healResult == nil {
		t.Fatal("expected auto-heal result for dangerous-only check")
	}

	if healResult.Attempted {
		t.Error("dangerous actions should NOT be auto-executed")
	}

	if healResult.Reason != "no safe recovery action available" {
		t.Errorf("expected 'no safe recovery action available', got %q", healResult.Reason)
	}

	// Verify no actions were actually executed
	executedActions := dangerousCheck.GetExecutedActions()
	if len(executedActions) != 0 {
		t.Errorf("expected 0 executed actions, got %d", len(executedActions))
	}
}

// TestFullStack_DisabledAutoHealSkipped tests that disabled auto-heal is respected
func TestFullStack_DisabledAutoHealSkipped(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	config := newMockConfigProvider()
	registry.SetConfigProvider(config)

	// Create a critical check but disable auto-heal
	disabledCheck := &mockHealableCheck{
		id:     "disabled-heal",
		status: checks.StatusCritical,
		recoveryActions: []checks.RecoveryAction{
			{ID: "fix", Name: "Fix", Available: true, Dangerous: false},
		},
		executeResult: checks.ActionResult{Success: true},
	}
	registry.Register(disabledCheck)
	config.DisableAutoHeal("disabled-heal") // Explicitly disabled

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Find result
	var healResult *checks.AutoHealResult
	for i := range autoHealResults {
		if autoHealResults[i].CheckID == "disabled-heal" {
			healResult = &autoHealResults[i]
			break
		}
	}

	if healResult == nil {
		t.Fatal("expected auto-heal result for disabled-heal check")
	}

	if healResult.Attempted {
		t.Error("auto-heal should NOT be attempted when disabled")
	}

	if healResult.Reason != "auto-heal not enabled for this check" {
		t.Errorf("expected reason 'auto-heal not enabled for this check', got %q", healResult.Reason)
	}
}

// TestFullStack_FailedActionLogged tests that failed recovery actions are properly logged
func TestFullStack_FailedActionLogged(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	store := newMockStore()
	config := newMockConfigProvider()
	registry.SetConfigProvider(config)

	// Create a check whose action will fail
	failingCheck := &mockHealableCheck{
		id:     "failing-action",
		status: checks.StatusCritical,
		recoveryActions: []checks.RecoveryAction{
			{ID: "fix", Name: "Fix", Available: true, Dangerous: false},
		},
		executeResult: checks.ActionResult{
			Success: false,
			Message: "Failed to fix",
			Error:   "Service not found",
		},
	}
	registry.Register(failingCheck)
	config.EnableAutoHeal("failing-action")

	ctx := context.Background()
	results := registry.RunAll(ctx, true)
	autoHealResults := registry.RunAutoHeal(ctx, results)

	// Find result
	var healResult *checks.AutoHealResult
	for i := range autoHealResults {
		if autoHealResults[i].CheckID == "failing-action" {
			healResult = &autoHealResults[i]
			break
		}
	}

	if healResult == nil {
		t.Fatal("expected auto-heal result")
	}

	if !healResult.Attempted {
		t.Error("auto-heal should have been attempted")
	}

	if healResult.ActionResult.Success {
		t.Error("action should have failed")
	}

	if healResult.ActionResult.Error != "Service not found" {
		t.Errorf("expected error 'Service not found', got %q", healResult.ActionResult.Error)
	}

	// Log the failed action
	store.SaveActionLog(
		healResult.CheckID,
		healResult.ActionResult.ActionID,
		healResult.ActionResult.Success,
		healResult.ActionResult.Message,
		healResult.ActionResult.Output,
		healResult.ActionResult.Error,
		healResult.ActionResult.Duration.Milliseconds(),
	)

	// Verify failure was logged
	logs := store.GetActionLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 action log, got %d", len(logs))
	}
	if logs[0].Success {
		t.Error("logged action should show failure")
	}
	if logs[0].Error != "Service not found" {
		t.Errorf("expected error in log, got %q", logs[0].Error)
	}
}

// TestFullStack_ConcurrentTicksAndAutoHeal tests concurrent access patterns
func TestFullStack_ConcurrentTicksAndAutoHeal(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	store := newMockStore()
	config := newMockConfigProvider()
	registry.SetConfigProvider(config)

	// Register multiple checks
	for i := 0; i < 5; i++ {
		check := &mockHealableCheck{
			id:     "concurrent-" + string(rune('a'+i)),
			status: checks.StatusCritical,
			recoveryActions: []checks.RecoveryAction{
				{ID: "fix", Name: "Fix", Available: true, Dangerous: false},
			},
			executeResult: checks.ActionResult{Success: true},
		}
		registry.Register(check)
		config.EnableAutoHeal(check.id)
	}

	ctx := context.Background()
	var wg sync.WaitGroup
	const concurrentWorkers = 5
	const ticksPerWorker = 3

	// Run concurrent ticks and auto-heals
	for w := 0; w < concurrentWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < ticksPerWorker; i++ {
				results := registry.RunAll(ctx, true)
				for _, r := range results {
					store.SaveResult(ctx, r)
				}
				registry.RunAutoHeal(ctx, results)
			}
		}()
	}

	wg.Wait()

	// Verify no panics or race conditions
	allResults := store.GetResults()
	t.Logf("Total results after concurrent test: %d", len(allResults))

	// Should have some results (exact count depends on timing)
	if len(allResults) == 0 {
		t.Error("expected some results after concurrent operations")
	}
}
