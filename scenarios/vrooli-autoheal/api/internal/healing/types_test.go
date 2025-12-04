package healing

import (
	"context"
	"testing"
	"time"

	"vrooli-autoheal/internal/checks"
)

// mockActionExecutor is a test double for ActionExecutor.
type mockActionExecutor struct {
	result checks.ActionResult
}

func (m *mockActionExecutor) Execute(ctx context.Context) checks.ActionResult {
	return m.result
}

// mockVerifier is a test double for Verifier.
type mockVerifier struct {
	success bool
}

func (m *mockVerifier) Verify(ctx context.Context) bool {
	return m.success
}

func TestActionExecutorFunc(t *testing.T) {
	called := false
	expectedResult := checks.ActionResult{
		ActionID: "test",
		Success:  true,
	}

	fn := ActionExecutorFunc(func(ctx context.Context) checks.ActionResult {
		called = true
		return expectedResult
	})

	result := fn.Execute(context.Background())

	if !called {
		t.Error("executor function was not called")
	}
	if result.ActionID != expectedResult.ActionID {
		t.Errorf("expected action ID %s, got %s", expectedResult.ActionID, result.ActionID)
	}
	if result.Success != expectedResult.Success {
		t.Errorf("expected success %v, got %v", expectedResult.Success, result.Success)
	}
}

func TestVerifierFunc(t *testing.T) {
	called := false

	fn := VerifierFunc(func(ctx context.Context) bool {
		called = true
		return true
	})

	result := fn.Verify(context.Background())

	if !called {
		t.Error("verifier function was not called")
	}
	if !result {
		t.Error("expected verifier to return true")
	}
}

func TestActionConfig_ToRecoveryAction(t *testing.T) {
	t.Run("without availability function", func(t *testing.T) {
		config := ActionConfig{
			ID:          "test-action",
			Name:        "Test Action",
			Description: "A test action",
			Dangerous:   false,
		}

		action := config.ToRecoveryAction(nil)

		if action.ID != "test-action" {
			t.Errorf("expected ID 'test-action', got %s", action.ID)
		}
		if action.Name != "Test Action" {
			t.Errorf("expected Name 'Test Action', got %s", action.Name)
		}
		if !action.Available {
			t.Error("expected action to be available when no availability function")
		}
	})

	t.Run("with availability function returning true", func(t *testing.T) {
		config := ActionConfig{
			ID:        "test-action",
			Dangerous: true,
			AvailabilityFn: func(lastResult *checks.Result) bool {
				return true
			},
		}

		action := config.ToRecoveryAction(nil)

		if !action.Available {
			t.Error("expected action to be available")
		}
		if !action.Dangerous {
			t.Error("expected action to be dangerous")
		}
	})

	t.Run("with availability function returning false", func(t *testing.T) {
		config := ActionConfig{
			ID: "test-action",
			AvailabilityFn: func(lastResult *checks.Result) bool {
				return false
			},
		}

		action := config.ToRecoveryAction(nil)

		if action.Available {
			t.Error("expected action to not be available")
		}
	})

	t.Run("availability function receives last result", func(t *testing.T) {
		lastResult := &checks.Result{
			CheckID: "test-check",
			Status:  checks.StatusCritical,
		}
		receivedResult := (*checks.Result)(nil)

		config := ActionConfig{
			ID: "test-action",
			AvailabilityFn: func(lr *checks.Result) bool {
				receivedResult = lr
				return lr != nil && lr.Status == checks.StatusCritical
			},
		}

		action := config.ToRecoveryAction(lastResult)

		if receivedResult != lastResult {
			t.Error("availability function did not receive last result")
		}
		if !action.Available {
			t.Error("expected action to be available for critical status")
		}
	})
}

func TestActionConfig_Execute(t *testing.T) {
	t.Run("no executor configured", func(t *testing.T) {
		config := ActionConfig{
			ID: "test-action",
		}

		result := config.Execute(context.Background(), "test-check", nil)

		if result.Success {
			t.Error("expected failure when no executor configured")
		}
		if result.Error == "" {
			t.Error("expected error message")
		}
	})

	t.Run("executor succeeds without verification", func(t *testing.T) {
		config := ActionConfig{
			ID: "test-action",
			ExecutorFactory: ActionFactoryFunc(func(actionID, checkID string, lastResult *checks.Result) ActionExecutor {
				return &mockActionExecutor{
					result: checks.ActionResult{
						ActionID: actionID,
						CheckID:  checkID,
						Success:  true,
						Message:  "Action completed",
					},
				}
			}),
		}

		result := config.Execute(context.Background(), "test-check", nil)

		if !result.Success {
			t.Error("expected success")
		}
		if result.Message != "Action completed" {
			t.Errorf("expected message 'Action completed', got %s", result.Message)
		}
	})

	t.Run("executor fails", func(t *testing.T) {
		config := ActionConfig{
			ID: "test-action",
			ExecutorFactory: ActionFactoryFunc(func(actionID, checkID string, lastResult *checks.Result) ActionExecutor {
				return &mockActionExecutor{
					result: checks.ActionResult{
						ActionID: actionID,
						CheckID:  checkID,
						Success:  false,
						Error:    "something went wrong",
					},
				}
			}),
		}

		result := config.Execute(context.Background(), "test-check", nil)

		if result.Success {
			t.Error("expected failure")
		}
		if result.Error != "something went wrong" {
			t.Errorf("expected error 'something went wrong', got %s", result.Error)
		}
	})

	t.Run("with verification success", func(t *testing.T) {
		config := ActionConfig{
			ID: "test-action",
			ExecutorFactory: ActionFactoryFunc(func(actionID, checkID string, lastResult *checks.Result) ActionExecutor {
				return &mockActionExecutor{
					result: checks.ActionResult{
						ActionID: actionID,
						CheckID:  checkID,
						Success:  true,
						Output:   "initial output",
					},
				}
			}),
			Verifier:    &mockVerifier{success: true},
			VerifyDelay: 1 * time.Millisecond, // Short delay for testing
		}

		result := config.Execute(context.Background(), "test-check", nil)

		if !result.Success {
			t.Error("expected success after verification")
		}
		if !containsSubstring(result.Output, "Verification") {
			t.Error("expected output to contain verification info")
		}
	})

	t.Run("with verification failure", func(t *testing.T) {
		config := ActionConfig{
			ID: "test-action",
			ExecutorFactory: ActionFactoryFunc(func(actionID, checkID string, lastResult *checks.Result) ActionExecutor {
				return &mockActionExecutor{
					result: checks.ActionResult{
						ActionID: actionID,
						CheckID:  checkID,
						Success:  true,
						Output:   "initial output",
					},
				}
			}),
			Verifier:    &mockVerifier{success: false},
			VerifyDelay: 1 * time.Millisecond,
		}

		result := config.Execute(context.Background(), "test-check", nil)

		if result.Success {
			t.Error("expected failure after verification")
		}
		if !containsSubstring(result.Error, "verification failed") {
			t.Errorf("expected error about verification failure, got %s", result.Error)
		}
	})

	t.Run("skips verification when executor fails", func(t *testing.T) {
		verifierCalled := false
		config := ActionConfig{
			ID: "test-action",
			ExecutorFactory: ActionFactoryFunc(func(actionID, checkID string, lastResult *checks.Result) ActionExecutor {
				return &mockActionExecutor{
					result: checks.ActionResult{
						ActionID: actionID,
						CheckID:  checkID,
						Success:  false,
						Error:    "executor failed",
					},
				}
			}),
			Verifier: VerifierFunc(func(ctx context.Context) bool {
				verifierCalled = true
				return true
			}),
			VerifyDelay: 1 * time.Millisecond,
		}

		result := config.Execute(context.Background(), "test-check", nil)

		if result.Success {
			t.Error("expected failure")
		}
		if verifierCalled {
			t.Error("verifier should not be called when executor fails")
		}
	})
}

func TestBaseHealer(t *testing.T) {
	t.Run("CheckID returns correct ID", func(t *testing.T) {
		healer := NewBaseHealer("test-check", nil)

		if healer.CheckID() != "test-check" {
			t.Errorf("expected check ID 'test-check', got %s", healer.CheckID())
		}
	})

	t.Run("Actions returns all configured actions", func(t *testing.T) {
		configs := []ActionConfig{
			{ID: "action-1", Name: "Action 1"},
			{ID: "action-2", Name: "Action 2", Dangerous: true},
			{ID: "action-3", Name: "Action 3", AvailabilityFn: func(*checks.Result) bool { return false }},
		}
		healer := NewBaseHealer("test-check", configs)

		actions := healer.Actions(nil)

		if len(actions) != 3 {
			t.Errorf("expected 3 actions, got %d", len(actions))
		}
		if actions[0].ID != "action-1" {
			t.Errorf("expected first action ID 'action-1', got %s", actions[0].ID)
		}
		if !actions[1].Dangerous {
			t.Error("expected second action to be dangerous")
		}
		if actions[2].Available {
			t.Error("expected third action to not be available")
		}
	})

	t.Run("Execute runs correct action", func(t *testing.T) {
		executedAction := ""
		configs := []ActionConfig{
			{
				ID: "action-1",
				ExecutorFactory: ActionFactoryFunc(func(actionID, checkID string, lr *checks.Result) ActionExecutor {
					return ActionExecutorFunc(func(ctx context.Context) checks.ActionResult {
						executedAction = actionID
						return checks.ActionResult{ActionID: actionID, Success: true}
					})
				}),
			},
			{
				ID: "action-2",
				ExecutorFactory: ActionFactoryFunc(func(actionID, checkID string, lr *checks.Result) ActionExecutor {
					return ActionExecutorFunc(func(ctx context.Context) checks.ActionResult {
						executedAction = actionID
						return checks.ActionResult{ActionID: actionID, Success: true}
					})
				}),
			},
		}
		healer := NewBaseHealer("test-check", configs)

		result := healer.Execute(context.Background(), "action-2", nil)

		if executedAction != "action-2" {
			t.Errorf("expected action-2 to be executed, got %s", executedAction)
		}
		if !result.Success {
			t.Error("expected success")
		}
	})

	t.Run("Execute returns error for unknown action", func(t *testing.T) {
		healer := NewBaseHealer("test-check", []ActionConfig{
			{ID: "action-1"},
		})

		result := healer.Execute(context.Background(), "unknown-action", nil)

		if result.Success {
			t.Error("expected failure for unknown action")
		}
		if !containsSubstring(result.Error, "unknown action") {
			t.Errorf("expected error about unknown action, got %s", result.Error)
		}
	})
}

func TestBaseHealer_ImplementsHealer(t *testing.T) {
	var _ Healer = (*BaseHealer)(nil)
}

// Helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
