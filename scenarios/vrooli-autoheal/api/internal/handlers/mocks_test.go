package handlers

import (
	"context"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/systemevents"
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
	transitions      *persistence.TransitionsResponse
	transitionsErr   error
	systemEvents     *systemevents.Response
	systemEventsErr  error
	incidentsErr     error
	incident         *incidents.Incident
	recordedArtifact *incidents.RemediationArtifact
	recordedOutcome  *incidents.Outcome
	savedInventories int
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

func (m *mockStore) GetTransitions(ctx context.Context, windowHours, limit int) (*persistence.TransitionsResponse, error) {
	if m.transitionsErr != nil {
		return nil, m.transitionsErr
	}
	if m.transitions != nil {
		return m.transitions, nil
	}
	return &persistence.TransitionsResponse{
		Transitions: []persistence.Transition{},
		WindowHours: windowHours,
		Total:       0,
	}, nil
}

func (m *mockStore) UpsertSystemEvents(ctx context.Context, events []systemevents.Event) (int, int, error) {
	return len(events), 0, nil
}

func (m *mockStore) UpsertSystemEventSource(ctx context.Context, source systemevents.SourceStatus) error {
	return nil
}

func (m *mockStore) ListSystemEvents(ctx context.Context, filters systemevents.Filters) (*systemevents.Response, error) {
	if m.systemEventsErr != nil {
		return nil, m.systemEventsErr
	}
	if m.systemEvents != nil {
		return m.systemEvents, nil
	}
	return &systemevents.Response{Events: []systemevents.Event{}, Sources: []systemevents.SourceStatus{}, Filters: systemevents.FiltersEcho{Limit: filters.Limit}}, nil
}

func (m *mockStore) GetSystemEventSources(ctx context.Context) ([]systemevents.SourceStatus, error) {
	return []systemevents.SourceStatus{}, nil
}

func (m *mockStore) CleanupOldSystemEvents(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (m *mockStore) SaveHostInventorySnapshot(ctx context.Context, inv hostinventory.HostInventory) (*hostinventory.SnapshotRecord, []hostinventory.Change, error) {
	m.savedInventories++
	return &hostinventory.SnapshotRecord{ID: "test", Inventory: inv}, nil, nil
}

func (m *mockStore) GetLatestHostInventorySnapshot(ctx context.Context) (*hostinventory.SnapshotRecord, error) {
	return nil, nil
}

func (m *mockStore) GetRecentHostInventoryChanges(ctx context.Context, limit int) ([]hostinventory.Change, error) {
	return []hostinventory.Change{}, nil
}

func (m *mockStore) UpsertIncident(ctx context.Context, input incidents.UpsertInput) (*incidents.Incident, error) {
	return &incidents.Incident{ID: "inc_test", Fingerprint: input.Fingerprint, Type: input.Type, Severity: input.Severity, Status: incidents.StatusOpen, EventCount: 1, ObservationCount: 1}, nil
}

func (m *mockStore) ListIncidents(ctx context.Context, filters incidents.ListFilters) (*incidents.ListResponse, error) {
	if m.incidentsErr != nil {
		return nil, m.incidentsErr
	}
	return &incidents.ListResponse{Incidents: []incidents.Incident{}, Filters: filters}, nil
}

func (m *mockStore) GetIncident(ctx context.Context, id string) (*incidents.Incident, error) {
	if m.incident != nil {
		incident := *m.incident
		return &incident, nil
	}
	return nil, nil
}

func (m *mockStore) ListIncidentObservations(ctx context.Context, incidentID string, limit int) ([]incidents.Observation, error) {
	return []incidents.Observation{}, nil
}

func (m *mockStore) UpdateIncidentStatus(ctx context.Context, incidentID string, status incidents.Status, note string) (*incidents.Incident, error) {
	return &incidents.Incident{ID: incidentID, Status: status}, nil
}

func (m *mockStore) RecordIncidentRemediationArtifact(ctx context.Context, incidentID string, artifact incidents.RemediationArtifact) (*incidents.Incident, error) {
	m.recordedArtifact = &artifact
	if m.incident == nil {
		return &incidents.Incident{ID: incidentID, RemediationArtifacts: []incidents.RemediationArtifact{artifact}}, nil
	}
	incident := *m.incident
	incident.RemediationArtifacts = append(incident.RemediationArtifacts, artifact)
	return &incident, nil
}

func (m *mockStore) RecordIncidentRemediationOutcome(ctx context.Context, incidentID string, outcome incidents.Outcome) (*incidents.Incident, error) {
	m.recordedOutcome = &outcome
	if m.incident == nil {
		return &incidents.Incident{ID: incidentID, Outcome: &outcome}, nil
	}
	incident := *m.incident
	incident.Outcome = &outcome
	return &incident, nil
}

// Action log mock methods [REQ:HEAL-ACTION-001].
func (m *mockStore) SaveActionLog(ctx context.Context, checkID, actionID string, success, timedOut bool, message, output, errMsg string, durationMs int64) error {
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

	h := NewWithInterface(registry, store, caps)
	h.hostCollector = fakeHostCollector{}
	return h
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

	h := NewWithInterface(registry, store, caps)
	h.hostCollector = fakeHostCollector{}
	return h
}

type fakeHostCollector struct{}

func (fakeHostCollector) Collect(ctx context.Context) (hostinventory.HostInventory, error) {
	return hostinventory.HostInventory{
		CollectedAt: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		Platform:    "linux",
		OS:          "linux",
		Arch:        "amd64",
		BootID:      "boot-test",
		Kernel: hostinventory.KernelInfo{
			Release:           "test-kernel",
			ModuleTreePresent: true,
		},
		ProbeStatus: map[string]hostinventory.ProbeState{"test": hostinventory.ProbeOK},
		Fingerprint: "test-fingerprint",
	}, nil
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
