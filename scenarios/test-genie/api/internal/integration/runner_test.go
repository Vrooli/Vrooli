package integration

import (
	"context"
	"errors"
	"io"
	"testing"

	"test-genie/internal/integration/cli"
	"test-genie/internal/integration/websocket"
	"test-genie/internal/shared"
)

// Mock validators for testing the runner orchestration logic in isolation

type mockCLIValidator struct {
	result cli.ValidationResult
}

func (m *mockCLIValidator) Validate(ctx context.Context) cli.ValidationResult {
	return m.result
}

type mockWebSocketValidator struct {
	result websocket.ValidationResult
}

func (m *mockWebSocketValidator) Validate(ctx context.Context) websocket.ValidationResult {
	return m.result
}

// Helper to create a runner with a mocked CLI validator
func newMockedRunner(cliVal *mockCLIValidator) *Runner {
	return New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
		},
		WithLogger(io.Discard),
		WithCLIValidator(cliVal),
	)
}

// Helper to create a runner with CLI and WebSocket validators
func newFullyMockedRunner(cliVal *mockCLIValidator, wsVal *mockWebSocketValidator) *Runner {
	r := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
		},
		WithLogger(io.Discard),
		WithCLIValidator(cliVal),
	)
	if wsVal != nil {
		r.websocketValidator = wsVal
	}
	return r
}

func TestRunner_AllValidatorsPass(t *testing.T) {
	runner := newMockedRunner(
		&mockCLIValidator{
			result: cli.ValidationResult{
				Result: shared.Result{
					Success: true,
					Observations: []shared.Observation{
						shared.NewSuccessObservation("CLI binary found"),
						shared.NewSuccessObservation("help works"),
						shared.NewSuccessObservation("version works"),
					},
				},
				BinaryPath:    "/mock/scenario/cli/mock-scenario",
				VersionOutput: "mock-scenario version 1.0.0",
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
}

func TestRunner_CLIValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockCLIValidator{
			result: cli.ValidationResult{
				Result: shared.FailMisconfiguration(
					errors.New("no executable found"),
					"Add CLI binary",
				),
			},
		},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when CLI validation fails")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
	if result.Summary.CLIValidated {
		t.Error("expected CLIValidated to remain false when CLI validation fails")
	}
}

func TestRunner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	runner := newMockedRunner(
		&mockCLIValidator{result: cli.ValidationResult{Result: shared.OK()}},
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
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when CLI validator not configured")
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
	)

	// Runner should have created the CLI validator (even though it'll fail on nonexistent dir)
	if runner.cliValidator == nil {
		t.Error("expected CLI validator to be created from command functions")
	}
}

func TestValidationSummary_TotalChecks(t *testing.T) {
	tests := []struct {
		name     string
		summary  ValidationSummary
		expected int
	}{
		{
			name: "CLI only",
			summary: ValidationSummary{
				CLIValidated: true,
			},
			expected: 3, // 3 CLI checks
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
		CLIValidated: true,
	}

	str := summary.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
	if !containsStr(str, "CLI") {
		t.Error("expected CLI mention in string")
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
	_ cli.Validator       = (*mockCLIValidator)(nil)
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

// ========== WebSocket Validator Tests ==========

func TestRunner_WebSocketValidationPasses(t *testing.T) {
	runner := newFullyMockedRunner(
		&mockCLIValidator{
			result: cli.ValidationResult{Result: shared.OK()},
		},
		&mockWebSocketValidator{
			result: websocket.ValidationResult{
				Result: shared.Result{
					Success: true,
					Observations: []shared.Observation{
						shared.NewSuccessObservation("WebSocket connected in 100ms"),
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
		&mockCLIValidator{
			result: cli.ValidationResult{Result: shared.OK()},
		},
		&mockWebSocketValidator{
			result: websocket.ValidationResult{
				Result: shared.FailSystem(
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
	// CLI should have passed before WebSocket failed
	if !result.Summary.CLIValidated {
		t.Error("CLI should have been validated before WebSocket failure")
	}
}

func TestRunner_WebSocketValidationSkippedWhenNotConfigured(t *testing.T) {
	runner := newMockedRunner(
		&mockCLIValidator{
			result: cli.ValidationResult{Result: shared.OK()},
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
		&mockCLIValidator{
			result: cli.ValidationResult{
				Result: shared.Result{
					Success:      true,
					Observations: []shared.Observation{shared.NewSuccessObservation("CLI ok")},
				},
			},
		},
		&mockWebSocketValidator{
			result: websocket.ValidationResult{
				Result: shared.Result{
					Success:      true,
					Observations: []shared.Observation{shared.NewSuccessObservation("WS ok")},
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
	if !result.Summary.CLIValidated {
		t.Error("expected CLI validated")
	}
	if !result.Summary.WebSocketValidated {
		t.Error("expected WebSocket validated")
	}

	// Verify total checks count is correct
	// CLI: 3 (binary, help, version)
	// WebSocket: 2 (connection + ping-pong)
	expectedChecks := 3 + 2
	if result.Summary.TotalChecks() != expectedChecks {
		t.Errorf("expected %d total checks, got %d", expectedChecks, result.Summary.TotalChecks())
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
		WithCLIValidator(&mockCLIValidator{result: cli.ValidationResult{Result: shared.OK()}}),
	)

	if runner.websocketValidator == nil {
		t.Error("expected WebSocket validator to be created from config")
	}
}

// ========== Summary Tests with New Fields ==========

func TestValidationSummary_TotalChecksWithWebSocket(t *testing.T) {
	tests := []struct {
		name     string
		summary  ValidationSummary
		expected int
	}{
		{
			name: "all categories",
			summary: ValidationSummary{
				CLIValidated:       true,
				WebSocketValidated: true,
			},
			expected: 3 + 2, // CLI(3) + WS(2)
		},
		{
			name: "WebSocket only",
			summary: ValidationSummary{
				WebSocketValidated: true,
			},
			expected: 2, // WS(2)
		},
		{
			name: "CLI only",
			summary: ValidationSummary{
				CLIValidated: true,
			},
			expected: 3, // CLI(3)
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

func TestValidationSummary_StringWithWebSocket(t *testing.T) {
	summary := ValidationSummary{
		CLIValidated:       true,
		WebSocketValidated: true,
	}

	str := summary.String()
	if !containsStr(str, "CLI") {
		t.Error("expected CLI mention in string")
	}
	if !containsStr(str, "WebSocket") {
		t.Error("expected WebSocket mention in string")
	}
}
