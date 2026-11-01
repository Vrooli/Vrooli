package vrooli

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

func TestResourceCheckExecuteAction_AllActions(t *testing.T) {
	tests := []struct {
		name          string
		actionID      string
		cmdKey        string
		cmdOutput     string
		cmdError      error
		statusOutput  string // For verification after start/restart
		expectSuccess bool
	}{
		{
			name:          "start success",
			actionID:      "start",
			cmdKey:        "vrooli resource start postgres",
			cmdOutput:     "Started postgres resource",
			cmdError:      nil,
			statusOutput:  `{"success":true,"name":"postgres","installed":true,"running":true,"healthy":true,"status":"healthy"}`,
			expectSuccess: true,
		},
		{
			name:          "start failure - command error",
			actionID:      "start",
			cmdKey:        "vrooli resource start postgres",
			cmdOutput:     "Failed to start",
			cmdError:      testutil.ErrCommandNotFound,
			statusOutput:  "",
			expectSuccess: false,
		},
		{
			name:          "restart success",
			actionID:      "restart",
			cmdKey:        "vrooli resource restart postgres",
			cmdOutput:     "Restarted postgres resource",
			cmdError:      nil,
			statusOutput:  `{"success":true,"name":"postgres","installed":true,"running":true,"healthy":true,"status":"healthy"}`,
			expectSuccess: true,
		},
		{
			name:          "restart failure - command error",
			actionID:      "restart",
			cmdKey:        "vrooli resource restart postgres",
			cmdOutput:     "",
			cmdError:      testutil.ErrConnectionRefused,
			statusOutput:  "",
			expectSuccess: false,
		},
		{
			name:          "stop success",
			actionID:      "stop",
			cmdKey:        "vrooli resource stop postgres",
			cmdOutput:     "Stopped postgres resource",
			cmdError:      nil,
			statusOutput:  "", // Stop doesn't need verification
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			mockExecutor.Responses[tt.cmdKey] = testutil.MockResponse{
				Output: []byte(tt.cmdOutput),
				Error:  tt.cmdError,
			}
			// Add status response for verification (used by start/restart)
			if tt.statusOutput != "" {
				mockExecutor.Responses["vrooli resource status postgres --json"] = testutil.MockResponse{
					Output: []byte(tt.statusOutput),
					Error:  nil,
				}
			}

			check := NewResourceCheck("postgres", WithResourceExecutor(mockExecutor))
			result := check.ExecuteAction(context.Background(), tt.actionID)

			if result.Success != tt.expectSuccess {
				t.Errorf("Success = %v, want %v (Error: %s)", result.Success, tt.expectSuccess, result.Error)
			}
			if result.ActionID != tt.actionID {
				t.Errorf("ActionID = %q, want %q", result.ActionID, tt.actionID)
			}

			// Verify executor was called
			if len(mockExecutor.Calls) < 1 {
				t.Error("Expected executor to be called")
			}
		})
	}
}

// TestScenarioCheckExecuteAction_AllActions tests all ScenarioCheck actions
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
