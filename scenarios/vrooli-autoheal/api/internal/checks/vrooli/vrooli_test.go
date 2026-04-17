// Package vrooli tests for Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001] [REQ:SCENARIO-CHECK-001] [REQ:HEAL-ACTION-001]
package vrooli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vrooli-autoheal/internal/checks"
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

// TestAPICheckInterface verifies APICheck implements Check
// [REQ:VROOLI-API-001]
func TestAPICheckInterface(t *testing.T) {
	var _ checks.Check = (*APICheck)(nil)

	check := NewAPICheck()
	if check.ID() != "vrooli-api" {
		t.Errorf("ID() = %q, want %q", check.ID(), "vrooli-api")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	// API check should run on all platforms
	if check.Platforms() != nil {
		t.Error("APICheck should run on all platforms")
	}
}

// TestAPICheckHealable verifies APICheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestAPICheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*APICheck)(nil)

	check := NewAPICheck()

	// Test recovery actions with nil result
	actions := check.RecoveryActions(nil)
	if len(actions) == 0 {
		t.Error("RecoveryActions() should return actions")
	}

	// Verify expected actions exist
	expectedActions := map[string]bool{
		"restart":   false,
		"kill-port": false,
		"logs":      false,
		"diagnose":  false,
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

// TestAPICheckOptions verifies APICheck configuration options
func TestAPICheckOptions(t *testing.T) {
	check := NewAPICheck(
		WithAPIURL("http://localhost:9000/health"),
		WithAPITimeout(10*time.Second),
	)

	if check.url != "http://localhost:9000/health" {
		t.Errorf("url = %q, want %q", check.url, "http://localhost:9000/health")
	}
	if check.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want %v", check.timeout, 10*time.Second)
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
			cliError:       checks.ErrCommandNotFound,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "postgres resource is not healthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := checks.NewMockExecutor()
			mockExecutor.Responses["vrooli resource status postgres --json"] = checks.MockResponse{
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
			cmdError:      checks.ErrCommandNotFound,
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
			mockExecutor := checks.NewMockExecutor()
			if tt.cmdKey != "" {
				mockExecutor.Responses[tt.cmdKey] = checks.MockResponse{
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
			cliError:       checks.ErrCommandNotFound,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "broken-scenario scenario check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := checks.NewMockExecutor()
			mockExecutor.Responses["vrooli scenario status "+tt.scenarioName+" --json"] = checks.MockResponse{
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
			mockExecutor := checks.NewMockExecutor()
			mockExecutor.Responses["vrooli scenario status important-scenario --json"] = checks.MockResponse{
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
			cmdError:      checks.ErrConnectionRefused,
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
			mockExecutor := checks.NewMockExecutor()
			if tt.cmdKey != "" {
				mockExecutor.Responses[tt.cmdKey] = checks.MockResponse{
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

// TestAPICheckRunWithMock tests APICheck.Run() using mock HTTP client
// [REQ:VROOLI-API-001] [REQ:TEST-SEAM-001]
func TestAPICheckRunWithMock(t *testing.T) {
	tests := []struct {
		name           string
		httpStatus     int
		httpBody       string
		httpError      error
		expectedStatus checks.Status
		expectMsgPart  string // Check that message contains this substring
	}{
		{
			name:           "API healthy",
			httpStatus:     200,
			httpBody:       `{"status":"healthy"}`,
			httpError:      nil,
			expectedStatus: checks.StatusOK,
			expectMsgPart:  "Vrooli API healthy",
		},
		{
			name:           "API unhealthy response",
			httpStatus:     503,
			httpBody:       `{"status":"unhealthy"}`,
			httpError:      nil,
			expectedStatus: checks.StatusCritical,
			expectMsgPart:  "Vrooli API reports unhealthy status",
		},
		{
			name:           "API connection error",
			httpStatus:     0,
			httpBody:       "",
			httpError:      checks.ErrConnectionRefused,
			expectedStatus: checks.StatusCritical,
			expectMsgPart:  "Vrooli API is not responding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := checks.NewMockHTTPClient()
			mockHTTP.DefaultResponse = checks.MockHTTPResponse{
				StatusCode: tt.httpStatus,
				Body:       tt.httpBody,
				Error:      tt.httpError,
			}

			check := NewAPICheck(WithHTTPClient(mockHTTP))
			result := check.Run(context.Background())

			if result.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}
			if !strings.Contains(result.Message, tt.expectMsgPart) {
				t.Errorf("Message = %q, want to contain %q", result.Message, tt.expectMsgPart)
			}

			// Verify the mock was called
			if len(mockHTTP.Calls) != 1 {
				t.Errorf("Expected 1 HTTP call, got %d", len(mockHTTP.Calls))
			}
		})
	}
}

// TestAPICheckExecuteActionWithMock tests APICheck.ExecuteAction() using mock
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestAPICheckExecuteActionWithMock(t *testing.T) {
	tests := []struct {
		name          string
		actionID      string
		cmdKey        string
		cmdOutput     string
		cmdError      error
		expectSuccess bool
	}{
		{
			name:          "diagnose success",
			actionID:      "diagnose",
			cmdKey:        "vrooli api diagnose",
			cmdOutput:     "API diagnostics: all OK",
			cmdError:      nil,
			expectSuccess: true,
		},
		{
			name:          "logs success",
			actionID:      "logs",
			cmdKey:        "vrooli api logs --tail 100",
			cmdOutput:     "API logs...",
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
			mockExecutor := checks.NewMockExecutor()
			if tt.cmdKey != "" {
				mockExecutor.Responses[tt.cmdKey] = checks.MockResponse{
					Output: []byte(tt.cmdOutput),
					Error:  tt.cmdError,
				}
			}

			check := NewAPICheck(WithAPIExecutor(mockExecutor))
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
			cmdError:      checks.ErrCommandNotFound,
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
			cmdError:      checks.ErrConnectionRefused,
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
			mockExecutor := checks.NewMockExecutor()
			mockExecutor.Responses[tt.cmdKey] = checks.MockResponse{
				Output: []byte(tt.cmdOutput),
				Error:  tt.cmdError,
			}
			// Add status response for verification (used by start/restart)
			if tt.statusOutput != "" {
				mockExecutor.Responses["vrooli resource status postgres --json"] = checks.MockResponse{
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
func TestScenarioCheckExecuteAction_AllActions(t *testing.T) {
	tests := []struct {
		name          string
		actionID      string
		responses     map[string]checks.MockResponse
		statusOutput  string // For verification after start/restart
		expectSuccess bool
	}{
		{
			name:     "start success",
			actionID: "start",
			responses: map[string]checks.MockResponse{
				"vrooli scenario start test-scenario --best-effort": {Output: []byte("Started test-scenario"), Error: nil},
			},
			statusOutput:  `{"success":true,"scenario":{"name":"test-scenario","status":"running","health_status":"healthy"}}`,
			expectSuccess: true,
		},
		{
			name:     "start failure - command error",
			actionID: "start",
			responses: map[string]checks.MockResponse{
				"vrooli scenario start test-scenario --best-effort": {Output: []byte(""), Error: checks.ErrCommandNotFound},
			},
			statusOutput:  "",
			expectSuccess: false,
		},
		{
			name:     "restart success",
			actionID: "restart",
			responses: map[string]checks.MockResponse{
				"vrooli scenario restart test-scenario --best-effort": {Output: []byte("Restarted test-scenario"), Error: nil},
			},
			statusOutput:  `{"success":true,"scenario":{"name":"test-scenario","status":"running","health_status":"healthy"}}`,
			expectSuccess: true,
		},
		{
			name:     "restart-clean success",
			actionID: "restart-clean",
			responses: map[string]checks.MockResponse{
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
			responses: map[string]checks.MockResponse{
				"vrooli scenario stop test-scenario":             {Output: []byte("Stopped test-scenario"), Error: nil},
				"vrooli scenario port test-scenario":             {Output: []byte("8080/tcp"), Error: nil},
				"vrooli diagnose-port 8080 test-scenario --json": {Output: []byte(`{"success":true,"diagnostic":{"port":8080,"scenario":"test-scenario","listener_inspection":{"available":true}}}`), Error: nil},
				"vrooli cleanup locks":                           {Output: []byte("cleaned locks"), Error: nil},
				"vrooli cleanup orphans":                         {Output: []byte("cleaned orphans"), Error: nil},
				"vrooli scenario start test-scenario":            {Output: []byte(""), Error: checks.ErrConnectionRefused},
			},
			statusOutput:  "",
			expectSuccess: false,
		},
		{
			name:     "stop success",
			actionID: "stop",
			responses: map[string]checks.MockResponse{
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
			mockExecutor := checks.NewMockExecutor()
			for key, response := range tt.responses {
				mockExecutor.Responses[key] = response
			}
			// Add status response for verification (used by start/restart/restart-clean)
			if tt.statusOutput != "" {
				mockExecutor.Responses["vrooli scenario status test-scenario --json"] = checks.MockResponse{
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

// TestAPICheckExecuteAction_UnknownAction tests unknown action handling
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestAPICheckExecuteAction_UnknownAction(t *testing.T) {
	check := NewAPICheck()
	result := check.ExecuteAction(context.Background(), "invalid-action")

	if result.Success {
		t.Error("Success should be false for unknown action")
	}
	if result.Error != "unknown action: invalid-action" {
		t.Errorf("Error = %q, want %q", result.Error, "unknown action: invalid-action")
	}
	if result.ActionID != "invalid-action" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "invalid-action")
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
		coreOutputs   map[string]checks.MockResponse
		expectSuccess bool
	}{
		{
			name:       "no ports to clean",
			portOutput: "No ports in use",
			portError:  nil,
			coreOutputs: map[string]checks.MockResponse{
				"vrooli cleanup locks":   {Output: []byte("cleaned locks"), Error: nil},
				"vrooli cleanup orphans": {Output: []byte("cleaned orphans"), Error: nil},
			},
			expectSuccess: true,
		},
		{
			name:       "ports cleaned successfully",
			portOutput: "8080/tcp\n9000/tcp",
			portError:  nil,
			coreOutputs: map[string]checks.MockResponse{
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
			portError:     checks.ErrCommandNotFound,
			coreOutputs:   nil,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := checks.NewMockExecutor()
			// Note: cleanup-ports uses "vrooli scenario port" (not "ports")
			mockExecutor.Responses["vrooli scenario port test-scenario"] = checks.MockResponse{
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
	mockExecutor := checks.NewMockExecutor()
	mockExecutor.Responses["vrooli scenario status test-scenario --json"] = checks.MockResponse{
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
			setupError:    checks.ErrCommandNotFound,
			expectSuccess: false,
		},
		{
			name:          "start fails",
			startError:    checks.ErrConnectionRefused,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := checks.NewMockExecutor()
			mockExecutor.Responses["vrooli scenario stop test-scenario"] = checks.MockResponse{
				Output: []byte("Stopped"),
				Error:  nil,
			}
			mockExecutor.Responses["vrooli scenario setup test-scenario"] = checks.MockResponse{
				Output: []byte("Setup complete"),
				Error:  tt.setupError,
			}
			mockExecutor.Responses["vrooli scenario start test-scenario --best-effort"] = checks.MockResponse{
				Output: []byte("Started"),
				Error:  tt.startError,
			}
			if tt.setupError == nil && tt.startError == nil {
				mockExecutor.Responses["vrooli scenario status test-scenario --json"] = checks.MockResponse{
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

// TestAPICheckRecoveryActionsDangerous tests dangerous action marking
// [REQ:HEAL-ACTION-001]
func TestAPICheckRecoveryActionsDangerous(t *testing.T) {
	check := NewAPICheck()
	actions := check.RecoveryActions(nil)

	actionMap := make(map[string]checks.RecoveryAction)
	for _, a := range actions {
		actionMap[a.ID] = a
	}

	// kill-port should be dangerous
	if action, ok := actionMap["kill-port"]; ok {
		if !action.Dangerous {
			t.Error("kill-port action should be dangerous")
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

// TestResourceCheckExecutorInjection verifies executor is properly injected
func TestResourceCheckExecutorInjection(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	check := NewResourceCheck("postgres", WithResourceExecutor(mockExec))

	if check.executor != mockExec {
		t.Error("Executor was not properly injected")
	}
}

// TestScenarioCheckExecutorInjection verifies executor is properly injected
func TestScenarioCheckExecutorInjection(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	check := NewScenarioCheck("test", true, WithScenarioExecutor(mockExec))

	if check.executor != mockExec {
		t.Error("Executor was not properly injected")
	}
}

// TestAPICheckExecutorInjection verifies executor is properly injected
func TestAPICheckExecutorInjection(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	check := NewAPICheck(WithAPIExecutor(mockExec))

	if check.executor != mockExec {
		t.Error("Executor was not properly injected")
	}
}

// TestAPICheckHTTPClientInjection verifies HTTP client is properly injected
func TestAPICheckHTTPClientInjection(t *testing.T) {
	mockHTTP := checks.NewMockHTTPClient()
	check := NewAPICheck(WithHTTPClient(mockHTTP))

	if check.client != mockHTTP {
		t.Error("HTTP client was not properly injected")
	}
}
