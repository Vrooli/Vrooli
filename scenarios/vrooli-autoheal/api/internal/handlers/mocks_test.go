package handlers

import (
	"context"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/persistence"
	"vrooli-autoheal/internal/platform"
)

// mockStore implements StoreInterface for testing.
type mockStore struct {
	pingErr          error
	saveErr          error
	recentResults    []checks.Result
	recentErr        error
	timelineEvents   []persistence.TimelineEvent
	timelineErr      error
	uptimeStats      *persistence.UptimeStats
	uptimeErr        error
	uptimeHistory    *persistence.UptimeHistory
	uptimeHistoryErr error
	checkTrends      *persistence.CheckTrendsResponse
	checkTrendsErr   error
	incidents        *persistence.IncidentsResponse
	incidentsErr     error
}

func (m *mockStore) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockStore) SaveResult(ctx context.Context, result checks.Result) error {
	return m.saveErr
}

func (m *mockStore) GetRecentResults(ctx context.Context, checkID string, limit int) ([]checks.Result, error) {
	return m.recentResults, m.recentErr
}

func (m *mockStore) GetTimelineEvents(ctx context.Context, limit int) ([]persistence.TimelineEvent, error) {
	if m.timelineErr != nil {
		return nil, m.timelineErr
	}
	return m.timelineEvents, nil
}

func (m *mockStore) GetUptimeStats(ctx context.Context, windowHours int) (*persistence.UptimeStats, error) {
	if m.uptimeErr != nil {
		return nil, m.uptimeErr
	}
	if m.uptimeStats != nil {
		return m.uptimeStats, nil
	}
	return &persistence.UptimeStats{
		TotalEvents:      100,
		OkEvents:         90,
		WarningEvents:    8,
		CriticalEvents:   2,
		UptimePercentage: 90.0,
		WindowHours:      24,
	}, nil
}

func (m *mockStore) GetUptimeHistory(ctx context.Context, windowHours, bucketCount int) (*persistence.UptimeHistory, error) {
	if m.uptimeHistoryErr != nil {
		return nil, m.uptimeHistoryErr
	}
	if m.uptimeHistory != nil {
		return m.uptimeHistory, nil
	}
	return &persistence.UptimeHistory{
		Buckets:     []persistence.UptimeHistoryBucket{},
		Overall:     persistence.UptimeStats{UptimePercentage: 90.0, TotalEvents: 100},
		WindowHours: windowHours,
		BucketCount: bucketCount,
	}, nil
}

func (m *mockStore) GetCheckTrends(ctx context.Context, windowHours int) (*persistence.CheckTrendsResponse, error) {
	if m.checkTrendsErr != nil {
		return nil, m.checkTrendsErr
	}
	if m.checkTrends != nil {
		return m.checkTrends, nil
	}
	return &persistence.CheckTrendsResponse{
		Trends:      []persistence.CheckTrend{},
		WindowHours: windowHours,
		TotalChecks: 0,
	}, nil
}

func (m *mockStore) GetIncidents(ctx context.Context, windowHours, limit int) (*persistence.IncidentsResponse, error) {
	if m.incidentsErr != nil {
		return nil, m.incidentsErr
	}
	if m.incidents != nil {
		return m.incidents, nil
	}
	return &persistence.IncidentsResponse{
		Incidents:   []persistence.Incident{},
		WindowHours: windowHours,
		Total:       0,
	}, nil
}

// Action log mock methods [REQ:HEAL-ACTION-001].
func (m *mockStore) SaveActionLog(ctx context.Context, checkID, actionID string, success bool, message, output, errMsg string, durationMs int64) error {
	return nil
}

func (m *mockStore) GetActionLogs(ctx context.Context, limit int) (*persistence.ActionLogsResponse, error) {
	return &persistence.ActionLogsResponse{
		Logs:  []persistence.ActionLog{},
		Total: 0,
	}, nil
}

func (m *mockStore) GetActionLogsForCheck(ctx context.Context, checkID string, limit int) (*persistence.ActionLogsResponse, error) {
	return &persistence.ActionLogsResponse{
		Logs:  []persistence.ActionLog{},
		Total: 0,
	}, nil
}

// mockCheck implements checks.Check for testing.
type mockCheck struct {
	id       string
	status   checks.Status
	message  string
	platform []platform.Type
}

func (c *mockCheck) ID() string                 { return c.id }
func (c *mockCheck) Title() string              { return "Mock Check" }
func (c *mockCheck) Description() string        { return "Mock check for testing" }
func (c *mockCheck) Importance() string         { return "Test importance" }
func (c *mockCheck) IntervalSeconds() int       { return 60 }
func (c *mockCheck) Platforms() []platform.Type { return c.platform }
func (c *mockCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *mockCheck) Run(ctx context.Context) checks.Result {
	return checks.Result{
		CheckID: c.id,
		Status:  c.status,
		Message: c.message,
	}
}

func setupTestHandlers(store StoreInterface) *Handlers {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
		HasDocker:       true,
	}

	registry := checks.NewRegistry(caps)
	_ = registry.SetAutoHealPolicy(checks.AutoHealPolicy{
		BaseCooldown:       5 * time.Minute,
		MaxRestartAttempts: 3,
	})

	// Register a mock check.
	registry.Register(&mockCheck{
		id:      "test-check",
		status:  checks.StatusOK,
		message: "Test OK",
	})

	return NewWithInterface(registry, store, caps)
}

// mockHealableCheck implements checks.HealableCheck for testing actions.
type mockHealableCheck struct {
	mockCheck
	recoveryActions []checks.RecoveryAction
	executeResult   checks.ActionResult
}

func (c *mockHealableCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	return c.recoveryActions
}

func (c *mockHealableCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	c.executeResult.ActionID = actionID
	return c.executeResult
}

func setupTestHandlersWithHealable(store StoreInterface) *Handlers {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
		HasDocker:       true,
	}

	registry := checks.NewRegistry(caps)
	_ = registry.SetAutoHealPolicy(checks.AutoHealPolicy{
		BaseCooldown:       5 * time.Minute,
		MaxRestartAttempts: 3,
	})

	// Register a healable mock check.
	registry.Register(&mockHealableCheck{
		mockCheck: mockCheck{
			id:      "healable-check",
			status:  checks.StatusOK,
			message: "Test OK",
		},
		recoveryActions: []checks.RecoveryAction{
			{ID: "restart", Name: "Restart", Description: "Restart service", Available: true},
			{ID: "logs", Name: "View Logs", Description: "View logs", Available: true},
		},
		executeResult: checks.ActionResult{
			CheckID: "healable-check",
			Success: true,
			Message: "Action completed",
		},
	})

	return NewWithInterface(registry, store, caps)
}

// mockHealableCheckCritical implements a critical healable check for auto-heal testing.
type mockHealableCheckCritical struct {
	mockCheck
	recoveryActions []checks.RecoveryAction
	executeResult   checks.ActionResult
	executeCalled   bool
}

func (c *mockHealableCheckCritical) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	return c.recoveryActions
}

func (c *mockHealableCheckCritical) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	c.executeCalled = true
	c.executeResult.ActionID = actionID
	return c.executeResult
}

// mockConfigProvider implements checks.ConfigProvider for testing.
type mockConfigProvider struct {
	enabledChecks  map[string]bool
	autoHealChecks map[string]bool
	autoHealOn     map[string]string
}

func (m *mockConfigProvider) IsCheckEnabled(checkID string) bool {
	if enabled, ok := m.enabledChecks[checkID]; ok {
		return enabled
	}
	return true // Default enabled.
}

func (m *mockConfigProvider) IsAutoHealEnabled(checkID string) bool {
	if enabled, ok := m.autoHealChecks[checkID]; ok {
		return enabled
	}
	return false // Default disabled.
}

func (m *mockConfigProvider) GetAutoHealOn(checkID string) string {
	if m.autoHealOn != nil {
		if v, ok := m.autoHealOn[checkID]; ok && v != "" {
			return v
		}
	}
	return "critical"
}

func setupTestHandlersWithAutoHeal(store StoreInterface) (*Handlers, *mockHealableCheckCritical) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
		HasDocker:       true,
	}

	registry := checks.NewRegistry(caps)
	_ = registry.SetAutoHealPolicy(checks.AutoHealPolicy{
		BaseCooldown:       5 * time.Minute,
		MaxRestartAttempts: 3,
	})

	// Create a critical check that will trigger auto-heal.
	criticalCheck := &mockHealableCheckCritical{
		mockCheck: mockCheck{
			id:      "critical-check",
			status:  checks.StatusCritical,
			message: "Service down",
		},
		recoveryActions: []checks.RecoveryAction{
			{ID: "start", Name: "Start", Description: "Start service", Available: true, Dangerous: false},
			{ID: "restart", Name: "Restart", Description: "Restart service", Available: true, Dangerous: true},
		},
		executeResult: checks.ActionResult{
			CheckID: "critical-check",
			Success: true,
			Message: "Service started",
		},
	}

	registry.Register(criticalCheck)

	// Enable auto-heal for this check.
	configProvider := &mockConfigProvider{
		enabledChecks:  map[string]bool{"critical-check": true},
		autoHealChecks: map[string]bool{"critical-check": true},
	}
	registry.SetConfigProvider(configProvider)

	return NewWithInterface(registry, store, caps), criticalCheck
}
