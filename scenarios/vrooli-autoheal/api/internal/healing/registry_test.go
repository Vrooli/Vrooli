package healing

import (
	"context"
	"testing"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
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

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.healers == nil {
		t.Error("expected healers map to be initialized")
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	healer := &mockHealer{checkID: "test-check"}

	r.Register(healer)

	if r.Get("test-check") != healer {
		t.Error("expected healer to be registered")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()
	healer := &mockHealer{checkID: "test-check"}
	r.Register(healer)

	r.Unregister("test-check")

	if r.Get("test-check") != nil {
		t.Error("expected healer to be unregistered")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	healer := &mockHealer{checkID: "test-check"}
	r.Register(healer)

	t.Run("existing healer", func(t *testing.T) {
		got := r.Get("test-check")
		if got != healer {
			t.Error("expected to get registered healer")
		}
	})

	t.Run("non-existing healer", func(t *testing.T) {
		got := r.Get("unknown-check")
		if got != nil {
			t.Error("expected nil for unknown check")
		}
	})
}

func TestRegistry_IsHealable(t *testing.T) {
	r := NewRegistry()
	healer := &mockHealer{checkID: "test-check"}
	r.Register(healer)

	if !r.IsHealable("test-check") {
		t.Error("expected IsHealable to return true for registered check")
	}
	if r.IsHealable("unknown-check") {
		t.Error("expected IsHealable to return false for unknown check")
	}
}

func TestRegistry_GetActions(t *testing.T) {
	r := NewRegistry()
	expectedActions := []checks.RecoveryAction{
		{ID: "action-1", Name: "Action 1"},
		{ID: "action-2", Name: "Action 2"},
	}
	healer := &mockHealer{
		checkID: "test-check",
		actions: expectedActions,
	}
	r.Register(healer)

	t.Run("existing healer", func(t *testing.T) {
		actions := r.GetActions("test-check", nil)
		if len(actions) != 2 {
			t.Errorf("expected 2 actions, got %d", len(actions))
		}
		if actions[0].ID != "action-1" {
			t.Errorf("expected first action ID 'action-1', got %s", actions[0].ID)
		}
	})

	t.Run("non-existing healer", func(t *testing.T) {
		actions := r.GetActions("unknown-check", nil)
		if actions != nil {
			t.Error("expected nil actions for unknown check")
		}
	})
}

func TestRegistry_Execute(t *testing.T) {
	r := NewRegistry()
	healer := &mockHealer{
		checkID: "test-check",
		executeResult: checks.ActionResult{
			Success: true,
			Message: "Action executed",
		},
	}
	r.Register(healer)

	t.Run("existing healer", func(t *testing.T) {
		result := r.Execute(context.Background(), "test-check", "action-1", nil)
		if !result.Success {
			t.Error("expected success")
		}
		if result.ActionID != "action-1" {
			t.Errorf("expected action ID 'action-1', got %s", result.ActionID)
		}
		if result.CheckID != "test-check" {
			t.Errorf("expected check ID 'test-check', got %s", result.CheckID)
		}
	})

	t.Run("non-existing healer", func(t *testing.T) {
		result := r.Execute(context.Background(), "unknown-check", "action-1", nil)
		if result.Success {
			t.Error("expected failure for unknown check")
		}
		if result.Error == "" {
			t.Error("expected error message")
		}
	})
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockHealer{checkID: "check-1"})
	r.Register(&mockHealer{checkID: "check-2"})
	r.Register(&mockHealer{checkID: "check-3"})

	ids := r.List()

	if len(ids) != 3 {
		t.Errorf("expected 3 IDs, got %d", len(ids))
	}

	// Check all IDs are present (order not guaranteed)
	found := make(map[string]bool)
	for _, id := range ids {
		found[id] = true
	}
	for _, expected := range []string{"check-1", "check-2", "check-3"} {
		if !found[expected] {
			t.Errorf("expected %s in list", expected)
		}
	}
}

func TestHealerAdapter(t *testing.T) {
	mockCheck := &mockHealableCheck{
		id: "mock-check",
		actions: []checks.RecoveryAction{
			{ID: "action-1", Name: "Action 1"},
		},
		executeResult: checks.ActionResult{
			Success: true,
			Message: "Executed",
		},
	}

	adapter := NewHealerAdapter(mockCheck)

	t.Run("CheckID", func(t *testing.T) {
		if adapter.CheckID() != "mock-check" {
			t.Errorf("expected check ID 'mock-check', got %s", adapter.CheckID())
		}
	})

	t.Run("Actions", func(t *testing.T) {
		actions := adapter.Actions(nil)
		if len(actions) != 1 {
			t.Errorf("expected 1 action, got %d", len(actions))
		}
		if actions[0].ID != "action-1" {
			t.Errorf("expected action ID 'action-1', got %s", actions[0].ID)
		}
	})

	t.Run("Execute", func(t *testing.T) {
		result := adapter.Execute(context.Background(), "action-1", nil)
		if !result.Success {
			t.Error("expected success")
		}
		if result.ActionID != "action-1" {
			t.Errorf("expected action ID 'action-1', got %s", result.ActionID)
		}
	})
}

func TestHealerAdapter_ImplementsHealer(t *testing.T) {
	var _ Healer = (*HealerAdapter)(nil)
}

// Test concurrent access to registry
func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			r.Register(&mockHealer{checkID: "check-1"})
			r.Unregister("check-1")
		}
		done <- true
	}()

	// Reader goroutines
	for i := 0; i < 3; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = r.Get("check-1")
				_ = r.IsHealable("check-1")
				_ = r.List()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}
