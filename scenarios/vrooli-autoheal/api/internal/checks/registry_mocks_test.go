package checks

import (
	"context"
	"sync"
	"time"
	"vrooli-autoheal/internal/platform"
)

// mockCheck is a test implementation of Check interface.
type mockCheck struct {
	id        string
	desc      string
	interval  int
	platforms []platform.Type
	result    Result
}

func (m *mockCheck) ID() string                 { return m.id }
func (m *mockCheck) Title() string              { return "Mock Check" }
func (m *mockCheck) Description() string        { return m.desc }
func (m *mockCheck) Importance() string         { return "Test importance" }
func (m *mockCheck) IntervalSeconds() int       { return m.interval }
func (m *mockCheck) Platforms() []platform.Type { return m.platforms }
func (m *mockCheck) Category() Category         { return CategoryInfrastructure }
func (m *mockCheck) Run(ctx context.Context) Result {
	return m.result
}

// mockHealableCheck implements both Check and HealableCheck for testing.
type mockHealableCheck struct {
	id              string
	result          Result
	actions         []RecoveryAction
	executeResult   ActionResult
	executedActions []string
	mu              sync.Mutex
}

func (m *mockHealableCheck) ID() string                 { return m.id }
func (m *mockHealableCheck) Title() string              { return "Healable Check" }
func (m *mockHealableCheck) Description() string        { return "Test healable check" }
func (m *mockHealableCheck) Importance() string         { return "Test importance" }
func (m *mockHealableCheck) IntervalSeconds() int       { return 60 }
func (m *mockHealableCheck) Platforms() []platform.Type { return nil }
func (m *mockHealableCheck) Category() Category         { return CategoryInfrastructure }
func (m *mockHealableCheck) Run(ctx context.Context) Result {
	return m.result
}

func (m *mockHealableCheck) RecoveryActions(lastResult *Result) []RecoveryAction {
	return m.actions
}

func (m *mockHealableCheck) ExecuteAction(ctx context.Context, actionID string) ActionResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executedActions = append(m.executedActions, actionID)
	m.executeResult.ActionID = actionID
	m.executeResult.CheckID = m.id
	return m.executeResult
}

// mockConfigProvider implements ConfigProvider for testing.
type mockConfigProvider struct {
	enabledChecks  map[string]bool
	autoHealChecks map[string]bool
	autoHealOn     map[string]string
}

type fixedClock struct {
	current time.Time
}

func (c *fixedClock) Now() time.Time {
	return c.current
}

func (m *mockConfigProvider) IsCheckEnabled(checkID string) bool {
	if m.enabledChecks == nil {
		return true
	}
	enabled, exists := m.enabledChecks[checkID]
	return !exists || enabled
}

func (m *mockConfigProvider) IsAutoHealEnabled(checkID string) bool {
	if m.autoHealChecks == nil {
		return false
	}
	return m.autoHealChecks[checkID]
}

func (m *mockConfigProvider) GetAutoHealOn(checkID string) string {
	if m.autoHealOn == nil {
		return "critical"
	}
	if v, ok := m.autoHealOn[checkID]; ok && v != "" {
		return v
	}
	return "critical"
}

// testPlatform returns a mock platform for testing.
func testPlatform() *platform.Capabilities {
	return &platform.Capabilities{
		Platform:            platform.Linux,
		HasDocker:           true,
		SupportsSystemd:     true,
		SupportsLaunchd:     false,
		SupportsWindowsSvc:  false,
		SupportsRDP:         false,
		IsWSL:               false,
		IsHeadlessServer:    true,
		SupportsCloudflared: true,
	}
}

func newTestRegistry() *Registry {
	reg := NewRegistry(testPlatform())
	_ = reg.SetAutoHealPolicy(AutoHealPolicy{
		BaseCooldown:       5 * time.Minute,
		MaxRestartAttempts: 3,
	})
	return reg
}
