package bootstrap

import (
	"context"
	"sync"
	"time"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// mockStore implements ResultLoader and ResultSaver for testing.
type mockStore struct {
	mu          sync.Mutex
	results     []checks.Result
	actionLogs  []mockActionLog
	saveErrors  map[string]error
	loadResults []checks.Result
	loadError   error
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

// mockConfigProvider implements checks.ConfigProvider for testing.
type mockConfigProvider struct {
	mu             sync.Mutex
	enabledChecks  map[string]bool
	autoHealChecks map[string]bool
	autoHealOn     map[string]string
}

func newMockConfigProvider() *mockConfigProvider {
	return &mockConfigProvider{
		enabledChecks:  make(map[string]bool),
		autoHealChecks: make(map[string]bool),
		autoHealOn:     make(map[string]string),
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

func (m *mockConfigProvider) GetAutoHealOn(checkID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.autoHealOn[checkID]; ok && v != "" {
		return v
	}
	return "critical"
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

// mockHealableCheck implements checks.HealableCheck for testing.
type mockHealableCheck struct {
	id              string
	status          checks.Status
	recoveryActions []checks.RecoveryAction
	executeResult   checks.ActionResult
	runCount        int
	executedActions []string
	mu              sync.Mutex
}

func (c *mockHealableCheck) ID() string    { return c.id }
func (c *mockHealableCheck) Title() string { return "Mock Healable Check " + c.id }
func (c *mockHealableCheck) Description() string {
	return "A mock healable check for integration testing"
}
func (c *mockHealableCheck) Importance() string         { return "Required for full-stack testing" }
func (c *mockHealableCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *mockHealableCheck) IntervalSeconds() int       { return 60 }
func (c *mockHealableCheck) Platforms() []platform.Type { return nil }

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

// mockCheckFactory is a test implementation of CheckFactory.
type mockCheckFactory struct {
	infraChecks  []checks.Check
	systemChecks []checks.Check
	vrooliChecks []checks.Check
	callCounts   map[string]int
}

func newMockCheckFactory() *mockCheckFactory {
	return &mockCheckFactory{
		callCounts: make(map[string]int),
	}
}

func (f *mockCheckFactory) CreateInfrastructureChecks(caps *platform.Capabilities) []checks.Check {
	f.callCounts["infrastructure"]++
	return f.infraChecks
}

func (f *mockCheckFactory) CreateSystemChecks() []checks.Check {
	f.callCounts["system"]++
	return f.systemChecks
}

func (f *mockCheckFactory) CreateVrooliChecks(caps *platform.Capabilities) []checks.Check {
	f.callCounts["vrooli"]++
	return f.vrooliChecks
}
