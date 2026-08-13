// Package vrooli tests for Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001] [REQ:SCENARIO-CHECK-001] [REQ:HEAL-ACTION-001]
package vrooli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

// TestResourceCheckInterface verifies ResourceCheck implements Check
// [REQ:RESOURCE-CHECK-001]
func TestResourceCheckInterface(t *testing.T) {
	var _ checks.Check = (*ResourceCheck)(nil)

	check := NewResourceCheck("postgres")
	if check.ID() != "resource-postgres" {
		t.Errorf("ID() = %q, want %q", check.ID(), "resource-postgres")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	// Resource checks should run on all platforms
	if check.Platforms() != nil {
		t.Error("ResourceCheck should run on all platforms")
	}
}

// TestResourceCheckCreation verifies ResourceCheck creation with different names
func TestResourceCheckCreation(t *testing.T) {
	resources := []string{"postgres", "redis", "ollama", "qdrant"}

	for _, res := range resources {
		check := NewResourceCheck(res)
		expectedID := "resource-" + res
		if check.ID() != expectedID {
			t.Errorf("NewResourceCheck(%q).ID() = %q, want %q", res, check.ID(), expectedID)
		}
		if check.resourceName != res {
			t.Errorf("resourceName = %q, want %q", check.resourceName, res)
		}
	}
}

// TestScenarioCheckInterface verifies ScenarioCheck implements Check
// [REQ:SCENARIO-CHECK-001]
func TestScenarioCheckInterface(t *testing.T) {
	var _ checks.Check = (*ScenarioCheck)(nil)

	check := NewScenarioCheck("test-scenario", true)
	if check.ID() != "scenario-test-scenario" {
		t.Errorf("ID() = %q, want %q", check.ID(), "scenario-test-scenario")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	// Scenario checks should run on all platforms
	if check.Platforms() != nil {
		t.Error("ScenarioCheck should run on all platforms")
	}
}

// TestScenarioCheckCriticality verifies critical flag affects status
func TestScenarioCheckCriticality(t *testing.T) {
	criticalCheck := NewScenarioCheck("test-crit", true)
	nonCriticalCheck := NewScenarioCheck("test-non-crit", false)

	if !criticalCheck.critical {
		t.Error("Critical check should have critical=true")
	}
	if nonCriticalCheck.critical {
		t.Error("Non-critical check should have critical=false")
	}
}

// TestResourceCheckHealable verifies ResourceCheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestResourceCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*ResourceCheck)(nil)

	check := NewResourceCheck("postgres")

	// Test recovery actions
	actions := check.RecoveryActions(nil)
	if len(actions) == 0 {
		t.Error("RecoveryActions() should return actions")
	}

	// Verify expected actions exist
	expectedActions := map[string]bool{
		"start":   false,
		"stop":    false,
		"restart": false,
		"status":  false,
		"logs":    false,
	}
	for _, action := range actions {
		if _, exists := expectedActions[action.ID]; exists {
			expectedActions[action.ID] = true
		}
	}
	for id, found := range expectedActions {
		if !found {
			t.Errorf("Expected action %q not found in RecoveryActions()", id)
		}
	}
}

// TestScenarioCheckHealable verifies ScenarioCheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestScenarioCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*ScenarioCheck)(nil)

	check := NewScenarioCheck("test-scenario", true)

	// Test recovery actions
	actions := check.RecoveryActions(nil)
	if len(actions) == 0 {
		t.Error("RecoveryActions() should return actions")
	}

	// Verify expected actions exist
	expectedActions := map[string]bool{
		"start":         false,
		"stop":          false,
		"restart":       false,
		"restart-clean": false,
		"setup-restart": false,
		"cleanup-ports": false,
		"logs":          false,
		"diagnose":      false,
	}
	for _, action := range actions {
		if _, exists := expectedActions[action.ID]; exists {
			expectedActions[action.ID] = true
		}
	}
	for id, found := range expectedActions {
		if !found {
			t.Errorf("Expected action %q not found in RecoveryActions()", id)
		}
	}
}

// =============================================================================
// Unit Tests with Mock Interfaces
// =============================================================================

// TestResourceCheckRunWithMock tests ResourceCheck.Run() using mock executor
// [REQ:RESOURCE-CHECK-001] [REQ:TEST-SEAM-001]
func TestResourceCheckRunWithMock(t *testing.T) {
	tests := []struct {
		name           string
		cliOutput      string
		cliError       error
		expectedStatus checks.Status
		expectedMsg    string
	}{
		{
			name:           "resource healthy",
			cliOutput:      `{"success":true,"name":"postgres","installed":true,"running":true,"healthy":true,"status":"healthy"}`,
			cliError:       nil,
			expectedStatus: checks.StatusOK,
			expectedMsg:    "postgres resource is healthy",
		},
		{
			name:           "resource stopped",
			cliOutput:      `{"success":true,"name":"postgres","installed":true,"running":false,"healthy":false,"status":"stopped"}`,
			cliError:       nil,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "postgres resource is stopped",
		},
		{
			name:           "resource unhealthy",
			cliOutput:      `{"success":true,"name":"postgres","installed":true,"running":true,"healthy":false,"status":"unhealthy"}`,
			cliError:       nil,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "postgres resource is unhealthy",
		},
		{
			name:           "resource not installed",
			cliOutput:      `{"success":true,"name":"postgres","installed":false,"running":false,"status":"not installed"}`,
			cliError:       nil,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "postgres resource is not installed",
		},
		{
			name:           "resource unclear status",
			cliOutput:      `{"success":true,"name":"postgres","installed":true,"running":true,"status":"unknown"}`,
			cliError:       nil,
			expectedStatus: checks.StatusWarning,
			expectedMsg:    "postgres resource status unclear",
		},
		{
			name:           "cli command failed",
			cliOutput:      "",
			cliError:       testutil.ErrCommandNotFound,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "postgres resource is not healthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			mockExecutor.Responses["vrooli resource status postgres --json"] = testutil.MockResponse{
				Output: []byte(tt.cliOutput),
				Error:  tt.cliError,
			}

			check := NewResourceCheck("postgres", WithResourceExecutor(mockExecutor))
			result := check.Run(context.Background())

			if result.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}
			if result.Message != tt.expectedMsg {
				t.Errorf("Message = %q, want %q", result.Message, tt.expectedMsg)
			}

			// Verify the mock was called
			if len(mockExecutor.Calls) != 1 {
				t.Errorf("Expected 1 executor call, got %d", len(mockExecutor.Calls))
			}
		})
	}
}

// TestResourceCheckExecuteActionWithMock tests ResourceCheck.ExecuteAction() using mock
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestResourceCheckExecuteActionWithMock(t *testing.T) {
	tests := []struct {
		name          string
		actionID      string
		cmdKey        string
		cmdOutput     string
		cmdError      error
		expectSuccess bool
	}{
		{
			name:          "logs success",
			actionID:      "logs",
			cmdKey:        "vrooli resource logs postgres --tail 50",
			cmdOutput:     "2024-01-01 Starting postgres...",
			cmdError:      nil,
			expectSuccess: true,
		},
		{
			name:          "logs failure",
			actionID:      "logs",
			cmdKey:        "vrooli resource logs postgres --tail 50",
			cmdOutput:     "",
			cmdError:      testutil.ErrCommandNotFound,
			expectSuccess: false,
		},
		{
			name:          "stop success",
			actionID:      "stop",
			cmdKey:        "vrooli resource stop postgres",
			cmdOutput:     "Stopped postgres",
			cmdError:      nil,
			expectSuccess: true,
		},
		{
			name:          "unknown action",
			actionID:      "invalid-action",
			cmdKey:        "",
			cmdOutput:     "",
			cmdError:      nil,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			if tt.cmdKey != "" {
				mockExecutor.Responses[tt.cmdKey] = testutil.MockResponse{
					Output: []byte(tt.cmdOutput),
					Error:  tt.cmdError,
				}
			}

			check := NewResourceCheck("postgres", WithResourceExecutor(mockExecutor))
			result := check.ExecuteAction(context.Background(), tt.actionID)

			if result.Success != tt.expectSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.expectSuccess)
			}
			if result.ActionID != tt.actionID {
				t.Errorf("ActionID = %q, want %q", result.ActionID, tt.actionID)
			}
			if result.CheckID != check.ID() {
				t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
			}
		})
	}
}

func TestResourceCheckWhisperCompanionDownRecovery(t *testing.T) {
	check := NewResourceCheck("whisper")
	lastResult := &checks.Result{
		CheckID: "resource-whisper",
		Status:  checks.StatusCritical,
		Message: "whisper resource is unhealthy",
		Details: map[string]interface{}{
			"running":       true,
			"companionDown": true,
		},
	}

	actions := check.RecoveryActions(lastResult)
	if len(actions) == 0 {
		t.Fatal("expected recovery actions")
	}
	if actions[0].ID != "respawn-companion" {
		t.Fatalf("first recovery action = %q, want respawn-companion", actions[0].ID)
	}
	if actions[0].Dangerous {
		t.Fatal("respawn-companion should not be marked dangerous")
	}

	mockExecutor := testutil.NewMockExecutor()
	mockExecutor.Responses["vrooli resource start whisper"] = testutil.MockResponse{
		Output: []byte("Whisper activity-edge companion started"),
		Error:  nil,
	}
	mockExecutor.Responses["vrooli resource status whisper --json"] = testutil.MockResponse{
		Output: []byte(`{"success":true,"name":"whisper","installed":true,"running":true,"healthy":true,"status":"healthy"}`),
		Error:  nil,
	}

	check = NewResourceCheck("whisper", WithResourceExecutor(mockExecutor))
	result := check.ExecuteAction(context.Background(), "respawn-companion")
	if !result.Success {
		t.Fatalf("respawn-companion success = false, error=%s output=%s", result.Error, result.Output)
	}
	if len(mockExecutor.Calls) == 0 || mockExecutor.Calls[0].Name+" "+strings.Join(mockExecutor.Calls[0].Args, " ") != "vrooli resource start whisper" {
		t.Fatalf("first command = %v, want vrooli resource start whisper", mockExecutor.Calls)
	}
}

// TestScenarioCheckRunWithMock tests ScenarioCheck.Run() using mock executor
// [REQ:SCENARIO-CHECK-001] [REQ:TEST-SEAM-001]
func TestScenarioCheckRunWithMock(t *testing.T) {
	tests := []struct {
		name           string
		scenarioName   string
		critical       bool
		cliOutput      string
		cliError       error
		expectedStatus checks.Status
		expectedMsg    string
	}{
		{
			name:           "critical scenario healthy",
			scenarioName:   "vrooli-autoheal",
			critical:       true,
			cliOutput:      `{"success":true,"scenario":{"name":"vrooli-autoheal","status":"running","health_status":"healthy"}}`,
			cliError:       nil,
			expectedStatus: checks.StatusOK,
			expectedMsg:    "vrooli-autoheal scenario is healthy",
		},
		{
			name:           "critical scenario degraded",
			scenarioName:   "vrooli-autoheal",
			critical:       true,
			cliOutput:      `{"success":true,"scenario":{"name":"vrooli-autoheal","status":"running","health_status":"degraded"}}`,
			cliError:       nil,
			expectedStatus: checks.StatusWarning,
			expectedMsg:    "vrooli-autoheal scenario is degraded",
		},
		{
			name:           "critical scenario unhealthy",
			scenarioName:   "vrooli-autoheal",
			critical:       true,
			cliOutput:      `{"success":true,"scenario":{"name":"vrooli-autoheal","status":"running","health_status":"unhealthy"}}`,
			cliError:       nil,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "vrooli-autoheal scenario is unhealthy",
		},
		{
			name:           "non-critical scenario unhealthy",
			scenarioName:   "deployment-manager",
			critical:       false,
			cliOutput:      `{"success":true,"scenario":{"name":"deployment-manager","status":"running","health_status":"unhealthy"}}`,
			cliError:       nil,
			expectedStatus: checks.StatusWarning,
			expectedMsg:    "deployment-manager scenario is unhealthy",
		},
		{
			name:           "critical scenario stopped",
			scenarioName:   "vrooli-autoheal",
			critical:       true,
			cliOutput:      `{"success":true,"scenario":{"name":"vrooli-autoheal","status":"stopped","health_status":null}}`,
			cliError:       nil,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "vrooli-autoheal scenario is stopped",
		},
		{
			name:           "non-critical scenario stopped",
			scenarioName:   "test-app",
			critical:       false,
			cliOutput:      `{"success":true,"scenario":{"name":"test-app","status":"stopped","health_status":null}}`,
			cliError:       nil,
			expectedStatus: checks.StatusWarning,
			expectedMsg:    "test-app scenario is stopped",
		},
		{
			name:           "scenario parse failure",
			scenarioName:   "my-scenario",
			critical:       true,
			cliOutput:      `not-json`,
			cliError:       nil,
			expectedStatus: checks.StatusWarning,
			expectedMsg:    "my-scenario scenario status parse failed",
		},
		{
			name:           "cli command failed",
			scenarioName:   "broken-scenario",
			critical:       true,
			cliOutput:      "",
			cliError:       testutil.ErrCommandNotFound,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "broken-scenario scenario check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			mockExecutor.Responses["vrooli scenario status "+tt.scenarioName+" --json"] = testutil.MockResponse{
				Output: []byte(tt.cliOutput),
				Error:  tt.cliError,
			}

			check := NewScenarioCheck(tt.scenarioName, tt.critical, WithScenarioExecutor(mockExecutor))
			result := check.Run(context.Background())

			if result.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}
			if result.Message != tt.expectedMsg {
				t.Errorf("Message = %q, want %q", result.Message, tt.expectedMsg)
			}

			// Verify the mock was called
			if len(mockExecutor.Calls) != 1 {
				t.Errorf("Expected 1 executor call, got %d", len(mockExecutor.Calls))
			}
		})
	}
}

func TestScenarioCheckRun_DriftSignatureFromLifecycleLog(t *testing.T) {
	// When `scenario status` reports stopped, the failure cause lives in the
	// lifecycle run log, not the status output. The check must read the log
	// tail and surface the appropriate recommendedAction.
	tests := []struct {
		name           string
		logTail        string
		expectedRoot   string
		expectedAction string
		expectedSource string
	}{
		{
			name:           "go-mod-tidy needed",
			logTail:        "[1/6] build-api\ngo: updates to go.mod needed; to update it:\n\tgo mod tidy\n",
			expectedRoot:   rootCauseGoModuleDrift,
			expectedAction: recommendedActionRecoverGo,
			expectedSource: "lifecycle-log",
		},
		{
			name:           "missing go.sum",
			logTail:        "main.go:19:2: missing go.sum entry for module providing package modernc.org/sqlite (imported by flow-verifier); to add:\n\tgo get flow-verifier\n",
			expectedRoot:   rootCauseGoModuleDrift,
			expectedAction: recommendedActionRecoverGo,
			expectedSource: "lifecycle-log",
		},
		{
			name:           "pnpm outdated lockfile",
			logTail:        "ERR_PNPM_OUTDATED_LOCKFILE Cannot install with frozen-lockfile because pnpm-lock.yaml is not up to date\n",
			expectedRoot:   rootCausePnpmInstallDrift,
			expectedAction: recommendedActionRecoverPnpm,
			expectedSource: "lifecycle-log",
		},
		{
			name:           "no drift signature",
			logTail:        "scenario started successfully then died from OOM\n",
			expectedRoot:   "",
			expectedAction: "",
			expectedSource: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			mockExecutor.Responses["vrooli scenario status broken --json"] = testutil.MockResponse{
				Output: []byte(`{"success":true,"scenario":{"name":"broken","status":"stopped","health_status":null}}`),
			}
			logTail := tt.logTail
			check := NewScenarioCheck("broken", true,
				WithScenarioExecutor(mockExecutor),
				WithScenarioLifecycleLogReader(func() string { return logTail }),
			)
			result := check.Run(context.Background())
			if result.Status != checks.StatusCritical {
				t.Fatalf("status = %v, want critical", result.Status)
			}
			gotRoot, _ := result.Details["rootCause"].(string)
			gotAction, _ := result.Details["recommendedAction"].(string)
			gotSource, _ := result.Details["driftSource"].(string)
			if gotRoot != tt.expectedRoot {
				t.Errorf("rootCause = %q, want %q", gotRoot, tt.expectedRoot)
			}
			if gotAction != tt.expectedAction {
				t.Errorf("recommendedAction = %q, want %q", gotAction, tt.expectedAction)
			}
			if gotSource != tt.expectedSource {
				t.Errorf("driftSource = %q, want %q", gotSource, tt.expectedSource)
			}
		})
	}
}

func TestScenarioCheckRun_StatusOutputDriftBeatsLog(t *testing.T) {
	// If the status output already contains a drift signature, prefer it
	// (driftSource = status-output) and do not fall back to the log.
	mockExecutor := testutil.NewMockExecutor()
	statusOut := `{"success":true,"scenario":{"name":"broken","status":"stopped","health_status":null},"info":{"setupReasons":["missing go.sum entry for module providing package x"]}}`
	mockExecutor.Responses["vrooli scenario status broken --json"] = testutil.MockResponse{
		Output: []byte(statusOut),
	}
	logReaderCalled := false
	check := NewScenarioCheck("broken", true,
		WithScenarioExecutor(mockExecutor),
		WithScenarioLifecycleLogReader(func() string {
			logReaderCalled = true
			return "ERR_PNPM_OUTDATED_LOCKFILE foo"
		}),
	)
	result := check.Run(context.Background())
	if got, _ := result.Details["driftSource"].(string); got != "status-output" {
		t.Errorf("driftSource = %q, want status-output", got)
	}
	if got, _ := result.Details["recommendedAction"].(string); got != recommendedActionRecoverGo {
		t.Errorf("recommendedAction = %q, want %q", got, recommendedActionRecoverGo)
	}
	if logReaderCalled {
		t.Error("log reader should not be invoked when status output already matched")
	}
}

func TestScenarioCheckRun_APIDownFallsBackToDirectHealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		directHealthy  bool
		directDetail   string
		critical       bool
		expectedStatus checks.Status
		expectedMsg    string
	}{
		{
			name:           "direct check confirms running",
			directHealthy:  true,
			critical:       true,
			expectedStatus: checks.StatusWarning,
			expectedMsg:    "important-scenario scenario appears running, but orchestration API is unavailable",
		},
		{
			name:           "direct check confirms stopped",
			directHealthy:  false,
			directDetail:   "no running processes found",
			critical:       true,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "important-scenario scenario appears stopped (Vrooli API unavailable and direct check failed)",
		},
	}

	apiUnavailableOutput := "[ERROR]   Vrooli API is not accessible at http://localhost:8092\n[INFO]    The API may not be running. Start it with: vrooli develop\n"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			mockExecutor.Responses["vrooli scenario status important-scenario --json"] = testutil.MockResponse{
				Output: []byte(apiUnavailableOutput),
				Error:  errors.New("exit status 1"),
			}

			check := NewScenarioCheck(
				"important-scenario",
				tt.critical,
				WithScenarioExecutor(mockExecutor),
				WithScenarioDirectHealthChecker(func(context.Context) (bool, string) {
					return tt.directHealthy, tt.directDetail
				}),
			)

			result := check.Run(context.Background())
			if result.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}
			if result.Message != tt.expectedMsg {
				t.Errorf("Message = %q, want %q", result.Message, tt.expectedMsg)
			}
			if fallback, ok := result.Details["fallback"]; !ok || fallback != "direct-health-check" {
				t.Errorf("fallback detail = %v, want direct-health-check", fallback)
			}
			if tt.directHealthy {
				if autoHealEligible, ok := result.Details["autoHealEligible"]; !ok || autoHealEligible != false {
					t.Errorf("autoHealEligible = %v, want false", autoHealEligible)
				}
			}
		})
	}
}

// TestScenarioCheckExecuteActionWithMock tests ScenarioCheck.ExecuteAction() using mock
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestScenarioCheckExecuteActionWithMock(t *testing.T) {
	tests := []struct {
		name          string
		actionID      string
		cmdKey        string
		cmdOutput     string
		cmdError      error
		expectSuccess bool
	}{
		{
			name:          "logs success",
			actionID:      "logs",
			cmdKey:        "vrooli scenario logs test-scenario --tail 100",
			cmdOutput:     "2024-01-01 Starting scenario...",
			cmdError:      nil,
			expectSuccess: true,
		},
		{
			name:          "stop success",
			actionID:      "stop",
			cmdKey:        "vrooli scenario stop test-scenario",
			cmdOutput:     "Stopped test-scenario",
			cmdError:      nil,
			expectSuccess: true,
		},
		{
			name:          "stop failure",
			actionID:      "stop",
			cmdKey:        "vrooli scenario stop test-scenario",
			cmdOutput:     "",
			cmdError:      testutil.ErrConnectionRefused,
			expectSuccess: false,
		},
		{
			name:          "unknown action",
			actionID:      "invalid-action",
			cmdKey:        "",
			cmdOutput:     "",
			cmdError:      nil,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			if tt.cmdKey != "" {
				mockExecutor.Responses[tt.cmdKey] = testutil.MockResponse{
					Output: []byte(tt.cmdOutput),
					Error:  tt.cmdError,
				}
			}

			check := NewScenarioCheck("test-scenario", true, WithScenarioExecutor(mockExecutor))
			result := check.ExecuteAction(context.Background(), tt.actionID)

			if result.Success != tt.expectSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.expectSuccess)
			}
			if result.ActionID != tt.actionID {
				t.Errorf("ActionID = %q, want %q", result.ActionID, tt.actionID)
			}
			if result.CheckID != check.ID() {
				t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
			}
		})
	}
}

// Note: CLI status classification is thoroughly tested in status_classifier_test.go

// TestExtractPorts tests the port extraction logic
// [REQ:SCENARIO-CHECK-001]
func TestExtractPorts(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []int
	}{
		{
			name:     "standard ports",
			output:   "API: 8080\nUI: 3000",
			expected: []int{8080, 3000},
		},
		{
			name:     "port with tcp suffix",
			output:   "8080/tcp\n5432/tcp",
			expected: []int{8080, 5432},
		},
		{
			name:     "colon prefix",
			output:   ":8080 :9000",
			expected: []int{8080, 9000},
		},
		{
			name:     "no ports",
			output:   "No ports found",
			expected: nil,
		},
		{
			name:     "reserved ports filtered",
			output:   "80 443 1024 8080",
			expected: []int{1024, 8080},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ports := extractPorts(tt.output)
			if len(ports) != len(tt.expected) {
				t.Errorf("extractPorts(%q) returned %d ports, want %d", tt.output, len(ports), len(tt.expected))
				return
			}
			for i, port := range ports {
				if port != tt.expected[i] {
					t.Errorf("extractPorts(%q)[%d] = %d, want %d", tt.output, i, port, tt.expected[i])
				}
			}
		})
	}
}

// =============================================================================
// Additional Action Execution Tests
// =============================================================================

// TestResourceCheckExecuteAction_AllActions tests all ResourceCheck actions
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
