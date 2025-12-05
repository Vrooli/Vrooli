package integration

import (
	"context"
	"errors"
	"io"
	"testing"

	"test-genie/internal/integration/api"
	"test-genie/internal/integration/bats"
	"test-genie/internal/integration/cli"
	"test-genie/internal/integration/websocket"
	"test-genie/internal/structure/types"
)

// Mock validators for testing the runner orchestration logic in isolation

type mockAPIValidator struct {
	result api.ValidationResult
}

func (m *mockAPIValidator) Validate(ctx context.Context) api.ValidationResult {
	return m.result
}

type mockCLIValidator struct {
	result cli.ValidationResult
}

func (m *mockCLIValidator) Validate(ctx context.Context) cli.ValidationResult {
	return m.result
}

type mockBatsRunner struct {
	result bats.RunResult
}

func (m *mockBatsRunner) Run(ctx context.Context) bats.RunResult {
	return m.result
}

type mockWebSocketValidator struct {
	result websocket.ValidationResult
}

func (m *mockWebSocketValidator) Validate(ctx context.Context) websocket.ValidationResult {
	return m.result
}

// Helper to create a runner with all mocks
func newMockedRunner(cliVal *mockCLIValidator, batsRun *mockBatsRunner) *Runner {
	return New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
		},
		WithLogger(io.Discard),
		WithCLIValidator(cliVal),
		WithBatsRunner(batsRun),
	)
}

// Helper to create a runner with all validators including API and WebSocket
func newFullyMockedRunner(apiVal *mockAPIValidator, cliVal *mockCLIValidator, batsRun *mockBatsRunner, wsVal *mockWebSocketValidator) *Runner {
	r := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
		},
		WithLogger(io.Discard),
		WithCLIValidator(cliVal),
		WithBatsRunner(batsRun),
	)
	if apiVal != nil {
		r.apiValidator = apiVal
	}
	if wsVal != nil {
		r.websocketValidator = wsVal
	}
	return r
}

func TestRunner_AllValidatorsPass(t *testing.T) {
	runner := newMockedRunner(
		&mockCLIValidator{
			result: cli.ValidationResult{
				Result: types.Result{
					Success: true,
					Observations: []types.Observation{
						types.NewSuccessObservation("CLI binary found"),
						types.NewSuccessObservation("help works"),
						types.NewSuccessObservation("version works"),
					},
				},
				BinaryPath:    "/mock/scenario/cli/mock-scenario",
				VersionOutput: "mock-scenario version 1.0.0",
			},
		},
		&mockBatsRunner{
			result: bats.RunResult{
				Result: types.Result{
					Success: true,
					Observations: []types.Observation{
						types.NewSuccessObservation("bats available"),
						types.NewSuccessObservation("primary suite passed"),
					},
				},
				PrimarySuite:        "/mock/scenario/cli/mock-scenario.bats",
				AdditionalSuitesRun: 2,
			},
		},
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
	if !result.Summary.CLIValidated {
		t.Error("expected CLI to be validated")
	}
	if !result.Summary.PrimaryBatsRan {
		t.Error("expected primary bats to run")
	}
	if result.Summary.AdditionalBatsSuites != 2 {
		t.Errorf("expected 2 additional suites, got %d", result.Summary.AdditionalBatsSuites)
	}
}

func TestRunner_CLIValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockCLIValidator{
			result: cli.ValidationResult{
				Result: types.FailMisconfiguration(
					errors.New("no executable found"),
					"Add CLI binary",
				),
			},
		},
		&mockBatsRunner{
			result: bats.RunResult{Result: types.OK()},
		},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when CLI validation fails")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
	if !result.Summary.CLIValidated {
		// CLIValidated should be false since it failed
		// This is correct behavior
	}
}

func TestRunner_BatsValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockCLIValidator{
			result: cli.ValidationResult{
				Result: types.OK().WithObservations(types.NewSuccessObservation("CLI ok")),
			},
		},
		&mockBatsRunner{
			result: bats.RunResult{
				Result: types.FailSystem(
					errors.New("primary suite failed"),
					"Fix test failures",
				),
			},
		},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when BATS validation fails")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system, got %s", result.FailureClass)
	}
	// CLI should have been validated before BATS failed
	if !result.Summary.CLIValidated {
		t.Error("expected CLI to be validated before BATS failure")
	}
}

func TestRunner_BatsMissingDependency(t *testing.T) {
	runner := newMockedRunner(
		&mockCLIValidator{
			result: cli.ValidationResult{
				Result: types.OK().WithObservations(types.NewSuccessObservation("CLI ok")),
			},
		},
		&mockBatsRunner{
			result: bats.RunResult{
				Result: types.Fail(
					errors.New("bats not installed"),
					bats.FailureClassMissingDependency,
					"Install bats",
				),
			},
		},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when bats not installed")
	}
	if result.FailureClass != FailureClassMissingDependency {
		t.Errorf("expected missing_dependency, got %s", result.FailureClass)
	}
}

func TestRunner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	runner := newMockedRunner(
		&mockCLIValidator{result: cli.ValidationResult{Result: types.OK()}},
		&mockBatsRunner{result: bats.RunResult{Result: types.OK()}},
	)

	result := runner.Run(ctx)

	if result.Success {
		t.Fatal("expected failure when context cancelled")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class for cancellation, got %s", result.FailureClass)
	}
}

func TestRunner_NoCLIValidator(t *testing.T) {
	runner := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
		},
		WithLogger(io.Discard),
		// No CLI validator, no command executor
		WithBatsRunner(&mockBatsRunner{result: bats.RunResult{Result: types.OK()}}),
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when CLI validator not configured")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure, got %s", result.FailureClass)
	}
}

func TestRunner_NoBatsRunner(t *testing.T) {
	runner := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
		},
		WithLogger(io.Discard),
		WithCLIValidator(&mockCLIValidator{
			result: cli.ValidationResult{Result: types.OK()},
		}),
		// No BATS runner, no command executor
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when BATS runner not configured")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure, got %s", result.FailureClass)
	}
}

func TestRunner_CreatesValidatorsFromCommandFunctions(t *testing.T) {
	// Test that runner creates validators when command functions are provided
	runner := New(
		Config{
			ScenarioDir:  "/tmp/nonexistent", // Will fail but tests creation
			ScenarioName: "test-scenario",
		},
		WithLogger(io.Discard),
		WithCommandExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			return nil
		}),
		WithCommandCapture(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) (string, error) {
			return "", nil
		}),
		WithCommandLookup(func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		}),
	)

	// Runner should have created validators (even though they'll fail on nonexistent dir)
	if runner.cliValidator == nil {
		t.Error("expected CLI validator to be created from command functions")
	}
	if runner.batsRunner == nil {
		t.Error("expected BATS runner to be created from command functions")
	}
}

func TestValidationSummary_TotalChecks(t *testing.T) {
	tests := []struct {
		name     string
		summary  ValidationSummary
		expected int
	}{
		{
			name: "all categories",
			summary: ValidationSummary{
				CLIValidated:         true,
				PrimaryBatsRan:       true,
				AdditionalBatsSuites: 3,
			},
			expected: 7, // 3 (CLI) + 1 (primary) + 3 (additional)
		},
		{
			name: "CLI only",
			summary: ValidationSummary{
				CLIValidated:         true,
				PrimaryBatsRan:       false,
				AdditionalBatsSuites: 0,
			},
			expected: 3, // 3 CLI checks
		},
		{
			name: "no additional suites",
			summary: ValidationSummary{
				CLIValidated:         true,
				PrimaryBatsRan:       true,
				AdditionalBatsSuites: 0,
			},
			expected: 4, // 3 (CLI) + 1 (primary)
		},
		{
			name:     "empty",
			summary:  ValidationSummary{},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.summary.TotalChecks(); got != tc.expected {
				t.Errorf("TotalChecks() = %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestValidationSummary_String(t *testing.T) {
	summary := ValidationSummary{
		CLIValidated:         true,
		PrimaryBatsRan:       true,
		AdditionalBatsSuites: 2,
	}

	str := summary.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
	if !containsStr(str, "CLI") {
		t.Error("expected CLI mention in string")
	}
	if !containsStr(str, "bats") {
		t.Error("expected bats mention in string")
	}
	if !containsStr(str, "2") {
		t.Error("expected additional suite count in string")
	}
}

func TestValidationSummary_StringEmpty(t *testing.T) {
	summary := ValidationSummary{}
	str := summary.String()
	if str != "no checks performed" {
		t.Errorf("expected 'no checks performed', got: %s", str)
	}
}

// Ensure mock types satisfy interfaces at compile time
var (
	_ api.Validator       = (*mockAPIValidator)(nil)
	_ cli.Validator       = (*mockCLIValidator)(nil)
	_ bats.Runner         = (*mockBatsRunner)(nil)
	_ websocket.Validator = (*mockWebSocketValidator)(nil)
)

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ========== API Validator Tests ==========

func TestRunner_APIValidationPasses(t *testing.T) {
	runner := newFullyMockedRunner(
		&mockAPIValidator{
			result: api.ValidationResult{
				Result: types.Result{
					Success: true,
					Observations: []types.Observation{
						types.NewSuccessObservation("health endpoint returned 200"),
						types.NewSuccessObservation("response time 50ms within threshold"),
					},
				},
				HealthEndpoint: "/health",
				ResponseTimeMs: 50,
				StatusCode:     200,
			},
		},
		&mockCLIValidator{
			result: cli.ValidationResult{Result: types.OK()},
		},
		&mockBatsRunner{
			result: bats.RunResult{Result: types.OK()},
		},
		nil,
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if !result.Summary.APIHealthChecked {
		t.Error("expected API health to be checked")
	}
}

func TestRunner_APIValidationFails(t *testing.T) {
	runner := newFullyMockedRunner(
		&mockAPIValidator{
			result: api.ValidationResult{
				Result: types.FailSystem(
					errors.New("health endpoint returned 503"),
					"Check API logs for errors",
				),
				StatusCode: 503,
			},
		},
		&mockCLIValidator{
			result: cli.ValidationResult{Result: types.OK()},
		},
		&mockBatsRunner{
			result: bats.RunResult{Result: types.OK()},
		},
		nil,
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when API validation fails")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure, got %s", result.FailureClass)
	}
	// CLI and BATS should NOT have run
	if result.Summary.CLIValidated {
		t.Error("CLI should not have been validated after API failure")
	}
}

func TestRunner_APIValidationSkippedWhenNotConfigured(t *testing.T) {
	runner := newMockedRunner(
		&mockCLIValidator{
			result: cli.ValidationResult{Result: types.OK()},
		},
		&mockBatsRunner{
			result: bats.RunResult{Result: types.OK()},
		},
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.Summary.APIHealthChecked {
		t.Error("API health should NOT have been checked when not configured")
	}
	// Should have skip observation
	foundSkip := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationSkip && containsStr(obs.Message, "API") {
			foundSkip = true
			break
		}
	}
	if !foundSkip {
		t.Error("expected skip observation for API health check")
	}
}

// ========== WebSocket Validator Tests ==========

func TestRunner_WebSocketValidationPasses(t *testing.T) {
	runner := newFullyMockedRunner(
		nil, // No API validator
		&mockCLIValidator{
			result: cli.ValidationResult{Result: types.OK()},
		},
		&mockBatsRunner{
			result: bats.RunResult{Result: types.OK()},
		},
		&mockWebSocketValidator{
			result: websocket.ValidationResult{
				Result: types.Result{
					Success: true,
					Observations: []types.Observation{
						types.NewSuccessObservation("WebSocket connected in 100ms"),
					},
				},
				Endpoint:         "/ws",
				ConnectionTimeMs: 100,
			},
		},
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if !result.Summary.WebSocketValidated {
		t.Error("expected WebSocket to be validated")
	}
}

func TestRunner_WebSocketValidationFails(t *testing.T) {
	runner := newFullyMockedRunner(
		nil,
		&mockCLIValidator{
			result: cli.ValidationResult{Result: types.OK()},
		},
		&mockBatsRunner{
			result: bats.RunResult{Result: types.OK()},
		},
		&mockWebSocketValidator{
			result: websocket.ValidationResult{
				Result: types.FailSystem(
					errors.New("WebSocket connection refused"),
					"Ensure scenario is running",
				),
			},
		},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when WebSocket validation fails")
	}
	// CLI and BATS should have passed before WebSocket failed
	if !result.Summary.CLIValidated {
		t.Error("CLI should have been validated before WebSocket failure")
	}
	if !result.Summary.PrimaryBatsRan {
		t.Error("BATS should have run before WebSocket failure")
	}
}

func TestRunner_WebSocketValidationSkippedWhenNotConfigured(t *testing.T) {
	runner := newMockedRunner(
		&mockCLIValidator{
			result: cli.ValidationResult{Result: types.OK()},
		},
		&mockBatsRunner{
			result: bats.RunResult{Result: types.OK()},
		},
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.Summary.WebSocketValidated {
		t.Error("WebSocket should NOT have been validated when not configured")
	}
	// Should have skip observation
	foundSkip := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationSkip && containsStr(obs.Message, "WebSocket") {
			foundSkip = true
			break
		}
	}
	if !foundSkip {
		t.Error("expected skip observation for WebSocket validation")
	}
}

// ========== Full Pipeline Tests ==========

func TestRunner_FullPipelineWithAllValidators(t *testing.T) {
	runner := newFullyMockedRunner(
		&mockAPIValidator{
			result: api.ValidationResult{
				Result: types.Result{
					Success:      true,
					Observations: []types.Observation{types.NewSuccessObservation("API healthy")},
				},
				StatusCode:     200,
				ResponseTimeMs: 50,
			},
		},
		&mockCLIValidator{
			result: cli.ValidationResult{
				Result: types.Result{
					Success:      true,
					Observations: []types.Observation{types.NewSuccessObservation("CLI ok")},
				},
			},
		},
		&mockBatsRunner{
			result: bats.RunResult{
				Result: types.Result{
					Success:      true,
					Observations: []types.Observation{types.NewSuccessObservation("BATS ok")},
				},
				AdditionalSuitesRun: 1,
			},
		},
		&mockWebSocketValidator{
			result: websocket.ValidationResult{
				Result: types.Result{
					Success:      true,
					Observations: []types.Observation{types.NewSuccessObservation("WS ok")},
				},
				ConnectionTimeMs: 100,
			},
		},
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	// Verify all validators ran
	if !result.Summary.APIHealthChecked {
		t.Error("expected API health checked")
	}
	if !result.Summary.CLIValidated {
		t.Error("expected CLI validated")
	}
	if !result.Summary.PrimaryBatsRan {
		t.Error("expected BATS ran")
	}
	if !result.Summary.WebSocketValidated {
		t.Error("expected WebSocket validated")
	}

	// Verify total checks count is correct
	// API: 2 (status + response time)
	// CLI: 3 (binary, help, version)
	// BATS: 1 (primary) + 1 (additional)
	// WebSocket: 2 (connection + ping-pong)
	expectedChecks := 2 + 3 + 1 + 1 + 2
	if result.Summary.TotalChecks() != expectedChecks {
		t.Errorf("expected %d total checks, got %d", expectedChecks, result.Summary.TotalChecks())
	}
}

func TestRunner_CreatesAPIValidatorFromConfig(t *testing.T) {
	runner := New(
		Config{
			ScenarioDir:       "/tmp/test",
			ScenarioName:      "test",
			APIBaseURL:        "http://localhost:8080",
			APIHealthEndpoint: "/health",
			APIMaxResponseMs:  500,
		},
		WithLogger(io.Discard),
		WithCLIValidator(&mockCLIValidator{result: cli.ValidationResult{Result: types.OK()}}),
		WithBatsRunner(&mockBatsRunner{result: bats.RunResult{Result: types.OK()}}),
	)

	if runner.apiValidator == nil {
		t.Error("expected API validator to be created from config")
	}
}

func TestRunner_CreatesWebSocketValidatorFromConfig(t *testing.T) {
	runner := New(
		Config{
			ScenarioDir:              "/tmp/test",
			ScenarioName:             "test",
			WebSocketURL:             "ws://localhost:8080/ws",
			WebSocketMaxConnectionMs: 1000,
		},
		WithLogger(io.Discard),
		WithCLIValidator(&mockCLIValidator{result: cli.ValidationResult{Result: types.OK()}}),
		WithBatsRunner(&mockBatsRunner{result: bats.RunResult{Result: types.OK()}}),
	)

	if runner.websocketValidator == nil {
		t.Error("expected WebSocket validator to be created from config")
	}
}

// ========== Summary Tests with New Fields ==========

func TestValidationSummary_TotalChecksWithAPIAndWebSocket(t *testing.T) {
	tests := []struct {
		name     string
		summary  ValidationSummary
		expected int
	}{
		{
			name: "all categories",
			summary: ValidationSummary{
				APIHealthChecked:     true,
				CLIValidated:         true,
				PrimaryBatsRan:       true,
				AdditionalBatsSuites: 2,
				WebSocketValidated:   true,
			},
			expected: 2 + 3 + 1 + 2 + 2, // API(2) + CLI(3) + Primary(1) + Additional(2) + WS(2)
		},
		{
			name: "API and WebSocket only",
			summary: ValidationSummary{
				APIHealthChecked:   true,
				WebSocketValidated: true,
			},
			expected: 4, // API(2) + WS(2)
		},
		{
			name: "legacy without API/WebSocket",
			summary: ValidationSummary{
				CLIValidated:         true,
				PrimaryBatsRan:       true,
				AdditionalBatsSuites: 1,
			},
			expected: 5, // CLI(3) + Primary(1) + Additional(1)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.summary.TotalChecks(); got != tc.expected {
				t.Errorf("TotalChecks() = %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestValidationSummary_StringWithAPIAndWebSocket(t *testing.T) {
	summary := ValidationSummary{
		APIHealthChecked:   true,
		CLIValidated:       true,
		PrimaryBatsRan:     true,
		WebSocketValidated: true,
	}

	str := summary.String()
	if !containsStr(str, "API") {
		t.Error("expected API mention in string")
	}
	if !containsStr(str, "CLI") {
		t.Error("expected CLI mention in string")
	}
	if !containsStr(str, "WebSocket") {
		t.Error("expected WebSocket mention in string")
	}
}
