package vrooli

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

func TestScenarioCheckExecuteAction_AllActions(t *testing.T) {
	tests := []struct {
		name          string
		actionID      string
		responses     map[string]testutil.MockResponse
		statusOutput  string // For verification after start/restart
		expectSuccess bool
	}{
		{
			name:     "start success",
			actionID: "start",
			responses: map[string]testutil.MockResponse{
				"vrooli scenario start test-scenario --best-effort": {Output: []byte("Started test-scenario"), Error: nil},
			},
			statusOutput:  `{"success":true,"scenario":{"name":"test-scenario","status":"running","health_status":"healthy"}}`,
			expectSuccess: true,
		},
		{
			name:     "start failure - command error",
			actionID: "start",
			responses: map[string]testutil.MockResponse{
				"vrooli scenario start test-scenario --best-effort": {Output: []byte(""), Error: testutil.ErrCommandNotFound},
			},
			statusOutput:  "",
			expectSuccess: false,
		},
		{
			name:     "restart success",
			actionID: "restart",
			responses: map[string]testutil.MockResponse{
				"vrooli scenario restart test-scenario --best-effort": {Output: []byte("Restarted test-scenario"), Error: nil},
			},
			statusOutput:  `{"success":true,"scenario":{"name":"test-scenario","status":"running","health_status":"healthy"}}`,
			expectSuccess: true,
		},
		{
			name:     "restart-clean success",
			actionID: "restart-clean",
			responses: map[string]testutil.MockResponse{
				"vrooli scenario stop test-scenario":             {Output: []byte("Stopped test-scenario"), Error: nil},
				"vrooli scenario port test-scenario":             {Output: []byte("8080/tcp"), Error: nil},
				"vrooli diagnose-port 8080 test-scenario --json": {Output: []byte(`{"success":true,"diagnostic":{"port":8080,"scenario":"test-scenario","listener_inspection":{"available":true}}}`), Error: nil},
				"vrooli cleanup locks":                           {Output: []byte("cleaned locks"), Error: nil},
				"vrooli cleanup orphans":                         {Output: []byte("cleaned orphans"), Error: nil},
				"vrooli scenario start test-scenario":            {Output: []byte("Clean restart of test-scenario"), Error: nil},
			},
			statusOutput:  `{"success":true,"scenario":{"name":"test-scenario","status":"running","health_status":"healthy"}}`,
			expectSuccess: true,
		},
		{
			name:     "restart-clean failure - command error",
			actionID: "restart-clean",
			responses: map[string]testutil.MockResponse{
				"vrooli scenario stop test-scenario":             {Output: []byte("Stopped test-scenario"), Error: nil},
				"vrooli scenario port test-scenario":             {Output: []byte("8080/tcp"), Error: nil},
				"vrooli diagnose-port 8080 test-scenario --json": {Output: []byte(`{"success":true,"diagnostic":{"port":8080,"scenario":"test-scenario","listener_inspection":{"available":true}}}`), Error: nil},
				"vrooli cleanup locks":                           {Output: []byte("cleaned locks"), Error: nil},
				"vrooli cleanup orphans":                         {Output: []byte("cleaned orphans"), Error: nil},
				"vrooli scenario start test-scenario":            {Output: []byte(""), Error: testutil.ErrConnectionRefused},
			},
			statusOutput:  "",
			expectSuccess: false,
		},
		{
			name:     "stop success",
			actionID: "stop",
			responses: map[string]testutil.MockResponse{
				"vrooli scenario stop test-scenario": {Output: []byte("Stopped test-scenario"), Error: nil},
			},
			statusOutput:  "", // Stop doesn't need verification
			expectSuccess: true,
		},
		// Note: The diagnose action gathers information from multiple commands
		// and always succeeds (it collects what it can). It doesn't have a
		// failure mode since errors from individual commands are ignored.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			for key, response := range tt.responses {
				mockExecutor.Responses[key] = response
			}
			// Add status response for verification (used by start/restart/restart-clean)
			if tt.statusOutput != "" {
				mockExecutor.Responses["vrooli scenario status test-scenario --json"] = testutil.MockResponse{
					Output: []byte(tt.statusOutput),
					Error:  nil,
				}
			}

			check := NewScenarioCheck(
				"test-scenario",
				true,
				WithScenarioExecutor(mockExecutor),
				WithScenarioRecoveryPolling(100*time.Millisecond, 10*time.Millisecond, 0),
			)
			result := check.ExecuteAction(context.Background(), tt.actionID)

			if result.Success != tt.expectSuccess {
				t.Errorf("Success = %v, want %v (Error: %s)", result.Success, tt.expectSuccess, result.Error)
			}
			if result.ActionID != tt.actionID {
				t.Errorf("ActionID = %q, want %q", result.ActionID, tt.actionID)
			}
		})
	}
}

// Note: restart and kill-port actions have complex implementations that
// involve file system operations, environment variables, and HTTP calls.
// They are tested through integration tests rather than unit tests with mocks.

// TestScenarioCheckCleanupPorts tests the cleanup-ports action
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestScenarioCheckCleanupPorts(t *testing.T) {
	tests := []struct {
		name          string
		portOutput    string
		portError     error
		coreOutputs   map[string]testutil.MockResponse
		expectSuccess bool
	}{
		{
			name:       "no ports to clean",
			portOutput: "No ports in use",
			portError:  nil,
			coreOutputs: map[string]testutil.MockResponse{
				"vrooli cleanup locks":   {Output: []byte("cleaned locks"), Error: nil},
				"vrooli cleanup orphans": {Output: []byte("cleaned orphans"), Error: nil},
			},
			expectSuccess: true,
		},
		{
			name:       "ports cleaned successfully",
			portOutput: "8080/tcp\n9000/tcp",
			portError:  nil,
			coreOutputs: map[string]testutil.MockResponse{
				"vrooli diagnose-port 8080 test-scenario --json": {Output: []byte(`{"success":true,"diagnostic":{"port":8080,"scenario":"test-scenario","listener_inspection":{"available":true}}}`), Error: nil},
				"vrooli diagnose-port 9000 test-scenario --json": {Output: []byte(`{"success":true,"diagnostic":{"port":9000,"scenario":"test-scenario","listener_inspection":{"available":true}}}`), Error: nil},
				"vrooli cleanup locks":                           {Output: []byte("cleaned locks"), Error: nil},
				"vrooli cleanup orphans":                         {Output: []byte("cleaned orphans"), Error: nil},
			},
			expectSuccess: true,
		},
		{
			name:          "port query fails",
			portOutput:    "",
			portError:     testutil.ErrCommandNotFound,
			coreOutputs:   nil,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			// Note: cleanup-ports uses "vrooli scenario port" (not "ports")
			mockExecutor.Responses["vrooli scenario port test-scenario"] = testutil.MockResponse{
				Output: []byte(tt.portOutput),
				Error:  tt.portError,
			}
			for key, resp := range tt.coreOutputs {
				mockExecutor.Responses[key] = resp
			}

			check := NewScenarioCheck("test-scenario", true, WithScenarioExecutor(mockExecutor))
			result := check.ExecuteAction(context.Background(), "cleanup-ports")

			if result.Success != tt.expectSuccess {
				t.Errorf("Success = %v, want %v (Error: %s)", result.Success, tt.expectSuccess, result.Error)
			}
		})
	}
}

// TestResourceCheckRecoveryActionsDangerous tests dangerous action marking
// [REQ:HEAL-ACTION-001]
func TestResourceCheckRecoveryActionsDangerous(t *testing.T) {
	check := NewResourceCheck("postgres")
	actions := check.RecoveryActions(nil)

	actionMap := make(map[string]checks.RecoveryAction)
	for _, a := range actions {
		actionMap[a.ID] = a
	}

	// stop and restart should typically be considered potentially dangerous
	// but this depends on implementation - just verify they exist
	expectedActions := []string{"start", "stop", "restart", "logs"}
	for _, id := range expectedActions {
		if _, exists := actionMap[id]; !exists {
			t.Errorf("Expected action %q not found", id)
		}
	}

	// logs action should be safe
	if action, ok := actionMap["logs"]; ok {
		if action.Dangerous {
			t.Error("logs action should not be dangerous")
		}
	}
}

// TestScenarioCheckRecoveryActionsDangerous tests dangerous action marking
// [REQ:HEAL-ACTION-001]
func TestScenarioCheckRecoveryActionsDangerous(t *testing.T) {
	check := NewScenarioCheck("test-scenario", true)
	actions := check.RecoveryActions(nil)

	actionMap := make(map[string]checks.RecoveryAction)
	for _, a := range actions {
		actionMap[a.ID] = a
	}

	// restart-clean should be dangerous as it clears state
	if action, ok := actionMap["restart-clean"]; ok {
		if !action.Dangerous {
			t.Error("restart-clean action should be dangerous")
		}
	}

	if action, ok := actionMap["setup-restart"]; ok {
		if !action.Dangerous {
			t.Error("setup-restart action should be dangerous")
		}
	}

	// logs and diagnose should be safe
	safeActions := []string{"logs", "diagnose"}
	for _, id := range safeActions {
		if action, ok := actionMap[id]; ok {
			if action.Dangerous {
				t.Errorf("%s action should not be dangerous", id)
			}
		}
	}
}

func TestScenarioCheckRun_DetectsSharedPackageDriftRootCause(t *testing.T) {
	mockExecutor := testutil.NewMockExecutor()
	mockExecutor.Responses["vrooli scenario status test-scenario --json"] = testutil.MockResponse{
		Output: []byte(`{
			"success": true,
			"scenario": {
				"name": "test-scenario",
				"status": "running",
				"health_status": "degraded"
			},
			"diagnostics": {
				"log_analysis": {
					"recent_events": {
						"ui": {
							"message": "Error [ERR_MODULE_NOT_FOUND]: Cannot find module '/repo/packages/api-base/dist/server/perf.js'"
						}
					}
				}
			}
		}`),
		Error: nil,
	}

	check := NewScenarioCheck("test-scenario", true, WithScenarioExecutor(mockExecutor))
	result := check.Run(context.Background())

	if got, _ := result.Details["rootCause"].(string); got != "shared-package-drift" {
		t.Fatalf("rootCause = %q, want shared-package-drift", got)
	}
	if got, _ := result.Details["recommendedAction"].(string); got != "setup-restart" {
		t.Fatalf("recommendedAction = %q, want setup-restart", got)
	}
}

func TestScenarioCheckExecuteAction_SetupRestart(t *testing.T) {
	tests := []struct {
		name          string
		setupError    error
		startError    error
		expectSuccess bool
	}{
		{
			name:          "success",
			expectSuccess: true,
		},
		{
			name:          "setup fails",
			setupError:    testutil.ErrCommandNotFound,
			expectSuccess: false,
		},
		{
			name:          "start fails",
			startError:    testutil.ErrConnectionRefused,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			mockExecutor.Responses["vrooli scenario stop test-scenario"] = testutil.MockResponse{
				Output: []byte("Stopped"),
				Error:  nil,
			}
			mockExecutor.Responses["vrooli scenario setup test-scenario"] = testutil.MockResponse{
				Output: []byte("Setup complete"),
				Error:  tt.setupError,
			}
			mockExecutor.Responses["vrooli scenario start test-scenario --best-effort"] = testutil.MockResponse{
				Output: []byte("Started"),
				Error:  tt.startError,
			}
			if tt.setupError == nil && tt.startError == nil {
				mockExecutor.Responses["vrooli scenario status test-scenario --json"] = testutil.MockResponse{
					Output: []byte(`{"success":true,"scenario":{"name":"test-scenario","status":"running","health_status":"healthy"}}`),
					Error:  nil,
				}
			}

			check := NewScenarioCheck(
				"test-scenario",
				true,
				WithScenarioExecutor(mockExecutor),
				WithScenarioRecoveryPolling(100*time.Millisecond, 10*time.Millisecond, 0),
			)

			result := check.ExecuteAction(context.Background(), "setup-restart")
			if result.Success != tt.expectSuccess {
				t.Fatalf("Success = %v, want %v (err=%s)", result.Success, tt.expectSuccess, result.Error)
			}
			if result.ActionID != "setup-restart" {
				t.Fatalf("ActionID = %q, want setup-restart", result.ActionID)
			}
		})
	}
}

// TestResourceCheckExecutorInjection verifies executor is properly injected
func TestResourceCheckExecutorInjection(t *testing.T) {
	mockExec := testutil.NewMockExecutor()
	check := NewResourceCheck("postgres", WithResourceExecutor(mockExec))

	if check.executor != mockExec {
		t.Error("Executor was not properly injected")
	}
}

// TestScenarioCheckExecutorInjection verifies executor is properly injected
func TestScenarioCheckExecutorInjection(t *testing.T) {
	mockExec := testutil.NewMockExecutor()
	check := NewScenarioCheck("test", true, WithScenarioExecutor(mockExec))

	if check.executor != mockExec {
		t.Error("Executor was not properly injected")
	}
}
