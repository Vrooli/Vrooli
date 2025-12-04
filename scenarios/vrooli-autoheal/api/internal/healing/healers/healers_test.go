package healers

import (
	"context"
	"errors"
	"testing"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/healing"
)

// mockExecutor implements checks.CommandExecutor for testing.
type mockExecutor struct {
	combinedOutputResult []byte
	combinedOutputErr    error
	outputResult         []byte
	outputErr            error
	runErr               error
}

func (m *mockExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.combinedOutputResult, m.combinedOutputErr
}

func (m *mockExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.outputResult, m.outputErr
}

func (m *mockExecutor) Run(ctx context.Context, name string, args ...string) error {
	return m.runErr
}

// --- ResourceHealer Tests ---

func TestResourceHealer_CheckID(t *testing.T) {
	h := NewResourceHealer("postgres", nil)
	if h.CheckID() != "resource-postgres" {
		t.Errorf("CheckID() = %q, want %q", h.CheckID(), "resource-postgres")
	}
}

func TestResourceHealer_Actions(t *testing.T) {
	h := NewResourceHealer("postgres", nil)

	t.Run("nil result", func(t *testing.T) {
		actions := h.Actions(nil)
		if len(actions) == 0 {
			t.Error("expected actions")
		}

		// Find start action
		var startAction *checks.RecoveryAction
		for i := range actions {
			if actions[i].ID == "start" {
				startAction = &actions[i]
				break
			}
		}
		if startAction == nil {
			t.Error("expected start action")
		}
	})

	t.Run("running result", func(t *testing.T) {
		result := &checks.Result{
			Status:  checks.StatusOK,
			Details: map[string]interface{}{"output": "running"},
		}
		actions := h.Actions(result)

		// Start should not be available when running
		for _, a := range actions {
			if a.ID == "start" && a.Available {
				t.Error("start should not be available when running")
			}
		}
	})

	t.Run("stopped result", func(t *testing.T) {
		result := &checks.Result{
			Status:  checks.StatusCritical,
			Details: map[string]interface{}{"output": "not running"},
		}
		actions := h.Actions(result)

		// Start should be available when stopped
		for _, a := range actions {
			if a.ID == "start" && !a.Available {
				t.Error("start should be available when stopped")
			}
		}
	})
}

func TestResourceHealer_Execute(t *testing.T) {
	exec := &mockExecutor{combinedOutputResult: []byte("ok")}
	h := NewResourceHealer("postgres", exec)

	t.Run("start", func(t *testing.T) {
		result := h.Execute(context.Background(), "start", nil)
		if result.ActionID != "start" {
			t.Errorf("ActionID = %q, want %q", result.ActionID, "start")
		}
		if result.CheckID != "resource-postgres" {
			t.Errorf("CheckID = %q, want %q", result.CheckID, "resource-postgres")
		}
		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
	})

	t.Run("stop", func(t *testing.T) {
		result := h.Execute(context.Background(), "stop", nil)
		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
	})

	t.Run("restart", func(t *testing.T) {
		result := h.Execute(context.Background(), "restart", nil)
		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
	})

	t.Run("logs", func(t *testing.T) {
		result := h.Execute(context.Background(), "logs", nil)
		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
	})

	t.Run("diagnose", func(t *testing.T) {
		result := h.Execute(context.Background(), "diagnose", nil)
		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		result := h.Execute(context.Background(), "unknown", nil)
		if result.Success {
			t.Error("expected failure for unknown action")
		}
		if result.Error == "" {
			t.Error("expected error message")
		}
	})
}

func TestResourceHealer_ExecuteFailure(t *testing.T) {
	exec := &mockExecutor{combinedOutputErr: errors.New("command failed")}
	h := NewResourceHealer("postgres", exec)

	result := h.Execute(context.Background(), "start", nil)
	if result.Success {
		t.Error("expected failure")
	}
}

func TestResourceHealer_ImplementsHealer(t *testing.T) {
	var _ healing.Healer = (*ResourceHealer)(nil)
}

// --- ScenarioHealer Tests ---

func TestScenarioHealer_CheckID(t *testing.T) {
	h := NewScenarioHealer("landing-manager", nil)
	if h.CheckID() != "scenario-landing-manager" {
		t.Errorf("CheckID() = %q, want %q", h.CheckID(), "scenario-landing-manager")
	}
}

func TestScenarioHealer_Actions(t *testing.T) {
	h := NewScenarioHealer("landing-manager", nil)

	t.Run("nil result", func(t *testing.T) {
		actions := h.Actions(nil)
		if len(actions) == 0 {
			t.Error("expected actions")
		}

		// Verify expected actions exist
		actionIDs := make(map[string]bool)
		for _, a := range actions {
			actionIDs[a.ID] = true
		}

		expected := []string{"start", "stop", "restart", "restart-clean", "logs", "diagnose"}
		for _, id := range expected {
			if !actionIDs[id] {
				t.Errorf("missing action: %s", id)
			}
		}
	})

	t.Run("running result", func(t *testing.T) {
		result := &checks.Result{
			Status:  checks.StatusOK,
			Details: map[string]interface{}{"output": "healthy"},
		}
		actions := h.Actions(result)

		for _, a := range actions {
			if a.ID == "start" && a.Available {
				t.Error("start should not be available when healthy")
			}
		}
	})
}

func TestScenarioHealer_Execute(t *testing.T) {
	exec := &mockExecutor{combinedOutputResult: []byte("ok")}
	h := NewScenarioHealer("landing-manager", exec)

	actions := []string{"start", "stop", "restart", "restart-clean", "logs", "diagnose"}
	for _, actionID := range actions {
		t.Run(actionID, func(t *testing.T) {
			result := h.Execute(context.Background(), actionID, nil)
			if !result.Success {
				t.Errorf("%s failed: %s", actionID, result.Error)
			}
		})
	}

	t.Run("unknown", func(t *testing.T) {
		result := h.Execute(context.Background(), "unknown", nil)
		if result.Success {
			t.Error("expected failure for unknown action")
		}
	})
}

func TestScenarioHealer_ImplementsHealer(t *testing.T) {
	var _ healing.Healer = (*ScenarioHealer)(nil)
}

// --- SystemdHealer Tests ---

func TestSystemdHealer_CheckID(t *testing.T) {
	h := NewSystemdHealer("infra-cloudflared", "cloudflared", nil)
	if h.CheckID() != "infra-cloudflared" {
		t.Errorf("CheckID() = %q, want %q", h.CheckID(), "infra-cloudflared")
	}
}

func TestSystemdHealer_Actions(t *testing.T) {
	h := NewSystemdHealer("infra-cloudflared", "cloudflared", nil)

	t.Run("nil result", func(t *testing.T) {
		actions := h.Actions(nil)
		if len(actions) == 0 {
			t.Error("expected actions")
		}

		// Verify expected actions exist
		actionIDs := make(map[string]bool)
		for _, a := range actions {
			actionIDs[a.ID] = true
		}

		expected := []string{"start", "stop", "restart", "logs", "status"}
		for _, id := range expected {
			if !actionIDs[id] {
				t.Errorf("missing action: %s", id)
			}
		}
	})

	t.Run("active service", func(t *testing.T) {
		result := &checks.Result{
			Details: map[string]interface{}{"serviceStatus": "active"},
		}
		actions := h.Actions(result)

		for _, a := range actions {
			if a.ID == "start" && a.Available {
				t.Error("start should not be available when active")
			}
			if a.ID == "stop" && !a.Available {
				t.Error("stop should be available when active")
			}
		}
	})

	t.Run("inactive service", func(t *testing.T) {
		result := &checks.Result{
			Details: map[string]interface{}{"serviceStatus": "inactive"},
		}
		actions := h.Actions(result)

		for _, a := range actions {
			if a.ID == "start" && !a.Available {
				t.Error("start should be available when inactive")
			}
			if a.ID == "stop" && a.Available {
				t.Error("stop should not be available when inactive")
			}
		}
	})
}

func TestSystemdHealer_Execute(t *testing.T) {
	exec := &mockExecutor{combinedOutputResult: []byte("ok")}
	h := NewSystemdHealer("infra-cloudflared", "cloudflared", exec)

	actions := []string{"start", "stop", "restart", "logs", "status"}
	for _, actionID := range actions {
		t.Run(actionID, func(t *testing.T) {
			result := h.Execute(context.Background(), actionID, nil)
			if !result.Success {
				t.Errorf("%s failed: %s", actionID, result.Error)
			}
		})
	}

	t.Run("unknown", func(t *testing.T) {
		result := h.Execute(context.Background(), "unknown", nil)
		if result.Success {
			t.Error("expected failure for unknown action")
		}
	})
}

func TestSystemdHealer_ImplementsHealer(t *testing.T) {
	var _ healing.Healer = (*SystemdHealer)(nil)
}

// --- Integration Tests ---

func TestHealerRegistration(t *testing.T) {
	registry := healing.NewRegistry()

	resourceHealer := NewResourceHealer("postgres", nil)
	scenarioHealer := NewScenarioHealer("landing-manager", nil)
	systemdHealer := NewSystemdHealer("infra-cloudflared", "cloudflared", nil)

	registry.Register(resourceHealer)
	registry.Register(scenarioHealer)
	registry.Register(systemdHealer)

	if !registry.IsHealable("resource-postgres") {
		t.Error("expected resource-postgres to be healable")
	}
	if !registry.IsHealable("scenario-landing-manager") {
		t.Error("expected scenario-landing-manager to be healable")
	}
	if !registry.IsHealable("infra-cloudflared") {
		t.Error("expected infra-cloudflared to be healable")
	}

	// Test actions retrieval
	actions := registry.GetActions("resource-postgres", nil)
	if len(actions) == 0 {
		t.Error("expected actions for resource-postgres")
	}
}
