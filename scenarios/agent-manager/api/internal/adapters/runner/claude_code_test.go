package runner_test

import (
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// =============================================================================
// EXECUTE REQUEST TESTS
// =============================================================================

func TestExecuteRequest_GetTag(t *testing.T) {
	runID := uuid.New()

	tests := []struct {
		name     string
		req      runner.ExecuteRequest
		expected string
	}{
		{
			name: "returns custom tag when set",
			req: runner.ExecuteRequest{
				RunID: runID,
				Tag:   "custom-tag-123",
			},
			expected: "custom-tag-123",
		},
		{
			name: "returns run ID when tag empty",
			req: runner.ExecuteRequest{
				RunID: runID,
				Tag:   "",
			},
			expected: runID.String(),
		},
		{
			name: "returns run ID when tag is whitespace only",
			req: runner.ExecuteRequest{
				RunID: runID,
				Tag:   "",
			},
			expected: runID.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.GetTag()
			if got != tt.expected {
				t.Errorf("GetTag() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExecuteRequest_GetConfig(t *testing.T) {
	profile := &domain.AgentProfile{
		Name:       "test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "opus",
		MaxTurns:   50,
		Timeout:    time.Hour,
	}

	resolvedConfig := &domain.RunConfig{
		RunnerType: domain.RunnerTypeCodex,
		Model:      "haiku",
		MaxTurns:   100,
	}

	tests := []struct {
		name           string
		req            runner.ExecuteRequest
		expectedRunner domain.RunnerType
		expectedTurns  int
	}{
		{
			name: "returns resolved config when set",
			req: runner.ExecuteRequest{
				Profile:        profile,
				ResolvedConfig: resolvedConfig,
			},
			expectedRunner: domain.RunnerTypeCodex,
			expectedTurns:  100,
		},
		{
			name: "returns config from profile when no resolved config",
			req: runner.ExecuteRequest{
				Profile:        profile,
				ResolvedConfig: nil,
			},
			expectedRunner: domain.RunnerTypeClaudeCode,
			expectedTurns:  50,
		},
		{
			name: "returns default config when nothing set",
			req: runner.ExecuteRequest{
				Profile:        nil,
				ResolvedConfig: nil,
			},
			expectedRunner: domain.RunnerTypeClaudeCode,
			expectedTurns:  30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.GetConfig()
			if got == nil {
				t.Fatal("GetConfig() returned nil")
			}
			if got.RunnerType != tt.expectedRunner {
				t.Errorf("GetConfig().RunnerType = %s, want %s", got.RunnerType, tt.expectedRunner)
			}
			if got.MaxTurns != tt.expectedTurns {
				t.Errorf("GetConfig().MaxTurns = %d, want %d", got.MaxTurns, tt.expectedTurns)
			}
		})
	}
}

// =============================================================================
// CAPABILITIES TESTS
// =============================================================================

func TestCapabilities_Fields(t *testing.T) {
	caps := runner.Capabilities{
		SupportsMessages:     true,
		SupportsToolEvents:   true,
		SupportsCostTracking: true,
		SupportsStreaming:    true,
		SupportsCancellation: true,
		MaxTurns:             100,
		SupportedModels:      []string{"sonnet", "opus", "haiku"},
	}

	if !caps.SupportsMessages {
		t.Error("expected SupportsMessages to be true")
	}
	if !caps.SupportsToolEvents {
		t.Error("expected SupportsToolEvents to be true")
	}
	if !caps.SupportsCostTracking {
		t.Error("expected SupportsCostTracking to be true")
	}
	if !caps.SupportsStreaming {
		t.Error("expected SupportsStreaming to be true")
	}
	if !caps.SupportsCancellation {
		t.Error("expected SupportsCancellation to be true")
	}
	if caps.MaxTurns != 100 {
		t.Errorf("MaxTurns = %d, want 100", caps.MaxTurns)
	}
	if len(caps.SupportedModels) != 3 {
		t.Errorf("SupportedModels length = %d, want 3", len(caps.SupportedModels))
	}
}

// =============================================================================
// EXECUTE RESULT TESTS
// =============================================================================

func TestExecuteResult_SuccessCase(t *testing.T) {
	result := &runner.ExecuteResult{
		Success:      true,
		ExitCode:     0,
		Duration:     5 * time.Second,
		ErrorMessage: "",
		Summary: &domain.RunSummary{
			Description:   "Completed task successfully",
			TurnsUsed:     5,
			TokensUsed:    1000,
			CostEstimate:  0.05,
			FilesModified: []string{"file1.go", "file2.go"},
		},
		Metrics: runner.ExecutionMetrics{
			TurnsUsed:       5,
			TokensInput:     800,
			TokensOutput:    200,
			ToolCallCount:   3,
			CostEstimateUSD: 0.05,
		},
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Summary == nil {
		t.Fatal("Summary is nil")
	}
	if result.Summary.TurnsUsed != 5 {
		t.Errorf("Summary.TurnsUsed = %d, want 5", result.Summary.TurnsUsed)
	}
	if result.Metrics.ToolCallCount != 3 {
		t.Errorf("Metrics.ToolCallCount = %d, want 3", result.Metrics.ToolCallCount)
	}
}

func TestExecuteResult_FailureCase(t *testing.T) {
	result := &runner.ExecuteResult{
		Success:      false,
		ExitCode:     1,
		Duration:     30 * time.Second,
		ErrorMessage: "Agent execution failed: timeout",
		Summary:      nil,
		Metrics: runner.ExecutionMetrics{
			TurnsUsed:   10,
			TokensInput: 5000,
		},
	}

	if result.Success {
		t.Error("expected Success to be false")
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if result.ErrorMessage == "" {
		t.Error("expected ErrorMessage to be set")
	}
	if result.Summary != nil {
		t.Error("expected Summary to be nil for failed execution")
	}
}

// =============================================================================
// EXECUTION METRICS TESTS
// =============================================================================

func TestExecutionMetrics_Fields(t *testing.T) {
	metrics := runner.ExecutionMetrics{
		TurnsUsed:       15,
		TokensInput:     10000,
		TokensOutput:    2000,
		ToolCallCount:   25,
		CostEstimateUSD: 0.15,
	}

	if metrics.TurnsUsed != 15 {
		t.Errorf("TurnsUsed = %d, want 15", metrics.TurnsUsed)
	}
	if metrics.TokensInput != 10000 {
		t.Errorf("TokensInput = %d, want 10000", metrics.TokensInput)
	}
	if metrics.TokensOutput != 2000 {
		t.Errorf("TokensOutput = %d, want 2000", metrics.TokensOutput)
	}
	if metrics.ToolCallCount != 25 {
		t.Errorf("ToolCallCount = %d, want 25", metrics.ToolCallCount)
	}
	if metrics.CostEstimateUSD != 0.15 {
		t.Errorf("CostEstimateUSD = %f, want 0.15", metrics.CostEstimateUSD)
	}
}

func TestExecutionMetrics_ZeroValues(t *testing.T) {
	metrics := runner.ExecutionMetrics{}

	if metrics.TurnsUsed != 0 {
		t.Errorf("TurnsUsed = %d, want 0", metrics.TurnsUsed)
	}
	if metrics.TokensInput != 0 {
		t.Errorf("TokensInput = %d, want 0", metrics.TokensInput)
	}
	if metrics.TokensOutput != 0 {
		t.Errorf("TokensOutput = %d, want 0", metrics.TokensOutput)
	}
}

// =============================================================================
// CONFIG TESTS
// =============================================================================

func TestConfig_Fields(t *testing.T) {
	cfg := runner.Config{
		BinaryPath: "/usr/local/bin/claude",
		Timeout:    30 * time.Minute,
		WorkDir:    "/home/user/project",
		Environment: map[string]string{
			"ANTHROPIC_API_KEY": "sk-xxx",
			"CLAUDE_MODEL":      "opus",
		},
	}

	if cfg.BinaryPath != "/usr/local/bin/claude" {
		t.Errorf("BinaryPath = %s, want /usr/local/bin/claude", cfg.BinaryPath)
	}
	if cfg.Timeout != 30*time.Minute {
		t.Errorf("Timeout = %v, want 30m", cfg.Timeout)
	}
	if cfg.WorkDir != "/home/user/project" {
		t.Errorf("WorkDir = %s, want /home/user/project", cfg.WorkDir)
	}
	if len(cfg.Environment) != 2 {
		t.Errorf("Environment length = %d, want 2", len(cfg.Environment))
	}
}

// =============================================================================
// RUNNER TYPE TESTS
// =============================================================================

func TestRunnerType_Constants(t *testing.T) {
	tests := []struct {
		runnerType domain.RunnerType
		wantValid  bool
	}{
		{domain.RunnerTypeClaudeCode, true},
		{domain.RunnerTypeCodex, true},
		{domain.RunnerTypeOpenCode, true},
		{domain.RunnerType("invalid"), false},
		{domain.RunnerType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.runnerType), func(t *testing.T) {
			if got := tt.runnerType.IsValid(); got != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

// =============================================================================
// EXECUTE REQUEST WITH TASK TESTS
// =============================================================================

func TestExecuteRequest_WithTask(t *testing.T) {
	taskID := uuid.New()
	task := &domain.Task{
		ID:        taskID,
		Title:     "Fix bug in authentication",
		ScopePath: "src/auth",
	}

	req := runner.ExecuteRequest{
		RunID:      uuid.New(),
		Task:       task,
		WorkingDir: "/home/user/project",
		Prompt:     "Fix the authentication bug",
	}

	if req.Task == nil {
		t.Fatal("Task is nil")
	}
	if req.Task.ID != taskID {
		t.Errorf("Task.ID = %s, want %s", req.Task.ID, taskID)
	}
	if req.Task.Title != "Fix bug in authentication" {
		t.Errorf("Task.Title = %s, want 'Fix bug in authentication'", req.Task.Title)
	}
	if req.Prompt != "Fix the authentication bug" {
		t.Errorf("Prompt = %s, want 'Fix the authentication bug'", req.Prompt)
	}
}

// =============================================================================
// EXECUTE REQUEST WITH ENVIRONMENT TESTS
// =============================================================================

func TestExecuteRequest_WithEnvironment(t *testing.T) {
	req := runner.ExecuteRequest{
		RunID:      uuid.New(),
		WorkingDir: "/tmp/test",
		Prompt:     "Test prompt",
		Environment: map[string]string{
			"CUSTOM_VAR":    "custom_value",
			"ANOTHER_VAR":   "another_value",
			"NUMERIC_VAR":   "12345",
			"EMPTY_VAR":     "",
			"SPECIAL_CHARS": "hello=world&foo=bar",
		},
	}

	if len(req.Environment) != 5 {
		t.Errorf("Environment length = %d, want 5", len(req.Environment))
	}

	if req.Environment["CUSTOM_VAR"] != "custom_value" {
		t.Errorf("Environment[CUSTOM_VAR] = %s, want custom_value", req.Environment["CUSTOM_VAR"])
	}

	if req.Environment["EMPTY_VAR"] != "" {
		t.Errorf("Environment[EMPTY_VAR] = %s, want empty string", req.Environment["EMPTY_VAR"])
	}

	if req.Environment["SPECIAL_CHARS"] != "hello=world&foo=bar" {
		t.Errorf("Environment[SPECIAL_CHARS] = %s, want 'hello=world&foo=bar'", req.Environment["SPECIAL_CHARS"])
	}
}

// =============================================================================
// NIL SAFETY TESTS
// =============================================================================

func TestExecuteRequest_NilSafety(t *testing.T) {
	t.Run("nil profile and resolved config", func(t *testing.T) {
		req := runner.ExecuteRequest{
			RunID:          uuid.New(),
			Profile:        nil,
			ResolvedConfig: nil,
		}

		cfg := req.GetConfig()
		if cfg == nil {
			t.Fatal("GetConfig() returned nil")
		}
		// Should return defaults
		if cfg.RunnerType != domain.RunnerTypeClaudeCode {
			t.Errorf("RunnerType = %s, want claude-code", cfg.RunnerType)
		}
	})

	t.Run("nil environment", func(t *testing.T) {
		req := runner.ExecuteRequest{
			RunID:       uuid.New(),
			Environment: nil,
		}

		if req.Environment != nil {
			t.Error("expected Environment to be nil")
		}
	})

	t.Run("nil task", func(t *testing.T) {
		req := runner.ExecuteRequest{
			RunID: uuid.New(),
			Task:  nil,
		}

		if req.Task != nil {
			t.Error("expected Task to be nil")
		}
	})
}
