package healing

import (
	"context"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// mockHealer is a test double for Healer.
type mockHealer struct {
	checkID       string
	actions       []checks.RecoveryAction
	executeResult checks.ActionResult
}

func (m *mockHealer) CheckID() string {
	return m.checkID
}

func (m *mockHealer) Actions(lastResult *checks.Result) []checks.RecoveryAction {
	return m.actions
}

func (m *mockHealer) Execute(ctx context.Context, actionID string, lastResult *checks.Result) checks.ActionResult {
	m.executeResult.ActionID = actionID
	m.executeResult.CheckID = m.checkID
	return m.executeResult
}

// mockHealableCheck is a test double for checks.HealableCheck.
type mockHealableCheck struct {
	id            string
	actions       []checks.RecoveryAction
	executeResult checks.ActionResult
}

func (m *mockHealableCheck) ID() string                            { return m.id }
func (m *mockHealableCheck) Title() string                         { return "Mock Check" }
func (m *mockHealableCheck) Description() string                   { return "A mock check" }
func (m *mockHealableCheck) Importance() string                    { return "For testing" }
func (m *mockHealableCheck) Category() checks.Category             { return checks.CategoryInfrastructure }
func (m *mockHealableCheck) IntervalSeconds() int                  { return 60 }
func (m *mockHealableCheck) Platforms() []platform.Type            { return nil }
func (m *mockHealableCheck) Run(ctx context.Context) checks.Result { return checks.Result{} }
func (m *mockHealableCheck) RecoveryActions(*checks.Result) []checks.RecoveryAction {
	return m.actions
}

func (m *mockHealableCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	m.executeResult.ActionID = actionID
	return m.executeResult
}
