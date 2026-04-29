package runner_test

import (
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// =============================================================================
// ExecuteRequest tests (generic; not codec-specific)
// =============================================================================

func TestExecuteRequest_GetTag(t *testing.T) {
	runID := uuid.New()
	tests := []struct {
		name     string
		req      runner.ExecuteRequest
		expected string
	}{
		{
			name:     "returns custom tag when set",
			req:      runner.ExecuteRequest{RunID: runID, Tag: "custom-tag-123"},
			expected: "custom-tag-123",
		},
		{
			name:     "returns run ID when tag empty",
			req:      runner.ExecuteRequest{RunID: runID, Tag: ""},
			expected: runID.String(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.GetTag(); got != tt.expected {
				t.Errorf("GetTag()=%q want %q", got, tt.expected)
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
		{"resolved wins", runner.ExecuteRequest{Profile: profile, ResolvedConfig: resolvedConfig}, domain.RunnerTypeCodex, 100},
		{"profile fallback", runner.ExecuteRequest{Profile: profile}, domain.RunnerTypeClaudeCode, 50},
		{"defaults", runner.ExecuteRequest{}, domain.RunnerTypeClaudeCode, 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.GetConfig()
			if got == nil {
				t.Fatal("nil")
			}
			if got.RunnerType != tt.expectedRunner {
				t.Errorf("RunnerType=%s want %s", got.RunnerType, tt.expectedRunner)
			}
			if got.MaxTurns != tt.expectedTurns {
				t.Errorf("MaxTurns=%d want %d", got.MaxTurns, tt.expectedTurns)
			}
		})
	}
}

func TestExecuteRequest_NilSafety(t *testing.T) {
	t.Run("nil profile/config", func(t *testing.T) {
		req := runner.ExecuteRequest{RunID: uuid.New()}
		cfg := req.GetConfig()
		if cfg == nil {
			t.Fatal("GetConfig() returned nil")
		}
		if cfg.RunnerType != domain.RunnerTypeClaudeCode {
			t.Errorf("RunnerType=%s", cfg.RunnerType)
		}
	})
	t.Run("nil environment is nil", func(t *testing.T) {
		req := runner.ExecuteRequest{RunID: uuid.New(), Environment: nil}
		if req.Environment != nil {
			t.Error("expected Environment nil")
		}
	})
	t.Run("nil task is nil", func(t *testing.T) {
		req := runner.ExecuteRequest{RunID: uuid.New(), Task: nil}
		if req.Task != nil {
			t.Error("expected Task nil")
		}
	})
}

func TestExecuteRequest_WithTask(t *testing.T) {
	taskID := uuid.New()
	req := runner.ExecuteRequest{
		RunID:      uuid.New(),
		Task:       &domain.Task{ID: taskID, Title: "Fix bug", ScopePath: "src/auth"},
		WorkingDir: "/home/user/project",
		Prompt:     "Fix the authentication bug",
	}
	if req.Task == nil || req.Task.ID != taskID {
		t.Fatalf("task wiring broken: %+v", req.Task)
	}
	if req.Prompt != "Fix the authentication bug" {
		t.Errorf("Prompt=%q", req.Prompt)
	}
}

func TestExecuteRequest_WithEnvironment(t *testing.T) {
	req := runner.ExecuteRequest{
		RunID: uuid.New(),
		Environment: map[string]string{
			"CUSTOM_VAR":    "custom_value",
			"EMPTY_VAR":     "",
			"SPECIAL_CHARS": "hello=world&foo=bar",
		},
	}
	if len(req.Environment) != 3 {
		t.Errorf("env len=%d", len(req.Environment))
	}
	if req.Environment["CUSTOM_VAR"] != "custom_value" {
		t.Errorf("CUSTOM_VAR=%s", req.Environment["CUSTOM_VAR"])
	}
}

// =============================================================================
// EffectivePrompt
// =============================================================================

func TestExecuteRequest_EffectivePrompt(t *testing.T) {
	cases := []struct {
		name      string
		req       runner.ExecuteRequest
		wantExact string
		contains  []string
	}{
		{"no system prompt", runner.ExecuteRequest{Prompt: "user message"}, "user message", nil},
		{"no user prompt", runner.ExecuteRequest{SystemPrompt: "instructions"}, "instructions", nil},
		{
			name:     "both wraps system in XML",
			req:      runner.ExecuteRequest{Prompt: "user data here", SystemPrompt: "you are an investigator"},
			contains: []string{"<system-instructions>", "you are an investigator", "</system-instructions>", "user data here"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.EffectivePrompt()
			if tt.wantExact != "" {
				if got != tt.wantExact {
					t.Errorf("got=%q want=%q", got, tt.wantExact)
				}
				return
			}
			for _, want := range tt.contains {
				if !contains(got, want) {
					t.Errorf("missing %q in %q", want, got)
				}
			}
		})
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// =============================================================================
// Capabilities, Result, Metrics, Config — domain-shape sanity
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
	if !caps.SupportsMessages || !caps.SupportsToolEvents || !caps.SupportsCostTracking {
		t.Error("expected all-true Supports fields")
	}
	if caps.MaxTurns != 100 {
		t.Errorf("MaxTurns=%d", caps.MaxTurns)
	}
	if len(caps.SupportedModels) != 3 {
		t.Errorf("models=%d", len(caps.SupportedModels))
	}
}

func TestExecuteResult_SuccessCase(t *testing.T) {
	result := &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Duration: 5 * time.Second,
		Summary: &domain.RunSummary{
			Description: "ok", TurnsUsed: 5, TokensUsed: 1000, CostEstimate: 0.05,
			FilesModified: []string{"a.go", "b.go"},
		},
		Metrics: runner.ExecutionMetrics{TurnsUsed: 5, TokensInput: 800, TokensOutput: 200, ToolCallCount: 3, CostEstimateUSD: 0.05},
	}
	if !result.Success || result.ExitCode != 0 {
		t.Error("success/exit broken")
	}
	if result.Summary == nil || result.Summary.TurnsUsed != 5 {
		t.Error("summary broken")
	}
	if result.Metrics.ToolCallCount != 3 {
		t.Error("metrics broken")
	}
}

func TestExecuteResult_FailureCase(t *testing.T) {
	result := &runner.ExecuteResult{
		Success: false, ExitCode: 1, Duration: 30 * time.Second,
		ErrorMessage: "Agent execution failed: timeout",
		Summary:      nil,
		Metrics:      runner.ExecutionMetrics{TurnsUsed: 10, TokensInput: 5000},
	}
	if result.Success || result.ExitCode != 1 {
		t.Error("expected failure/1")
	}
	if result.ErrorMessage == "" || result.Summary != nil {
		t.Error("expected error message + nil summary")
	}
}

func TestExecutionMetrics_Fields(t *testing.T) {
	m := runner.ExecutionMetrics{
		TurnsUsed: 15, TokensInput: 10000, TokensOutput: 2000,
		CacheReadTokens: 300, CacheCreationTokens: 150,
		ToolCallCount: 25, CostEstimateUSD: 0.15,
	}
	if m.TurnsUsed != 15 || m.TokensInput != 10000 || m.ToolCallCount != 25 {
		t.Errorf("fields broken: %+v", m)
	}
	if m.CostEstimateUSD != 0.15 {
		t.Errorf("cost=%v", m.CostEstimateUSD)
	}
}

func TestExecutionMetrics_ZeroValues(t *testing.T) {
	m := runner.ExecutionMetrics{}
	if m.TurnsUsed != 0 || m.TokensInput != 0 || m.TokensOutput != 0 ||
		m.CacheReadTokens != 0 || m.CacheCreationTokens != 0 {
		t.Errorf("zero-value broken: %+v", m)
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := runner.Config{
		BinaryPath:  "/usr/local/bin/claude",
		Timeout:     30 * time.Minute,
		WorkDir:     "/home/user/project",
		Environment: map[string]string{"K1": "V1", "K2": "V2"},
	}
	if cfg.BinaryPath != "/usr/local/bin/claude" || cfg.Timeout != 30*time.Minute {
		t.Errorf("path/timeout broken: %+v", cfg)
	}
	if len(cfg.Environment) != 2 {
		t.Errorf("env len=%d", len(cfg.Environment))
	}
}

func TestRunnerType_Constants(t *testing.T) {
	cases := []struct {
		rt        domain.RunnerType
		wantValid bool
	}{
		{domain.RunnerTypeClaudeCode, true},
		{domain.RunnerTypeCodex, true},
		{domain.RunnerTypeOpenCode, true},
		{domain.RunnerType("invalid"), false},
		{domain.RunnerType(""), false},
	}
	for _, tt := range cases {
		t.Run(string(tt.rt), func(t *testing.T) {
			if got := tt.rt.IsValid(); got != tt.wantValid {
				t.Errorf("IsValid=%v want %v", got, tt.wantValid)
			}
		})
	}
}
