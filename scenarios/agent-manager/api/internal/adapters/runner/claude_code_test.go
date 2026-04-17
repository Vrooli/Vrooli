package runner_test

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"strings"
	"testing"
	"time"

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
		TurnsUsed:           15,
		TokensInput:         10000,
		TokensOutput:        2000,
		CacheReadTokens:     300,
		CacheCreationTokens: 150,
		ToolCallCount:       25,
		CostEstimateUSD:     0.15,
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
	if metrics.CacheReadTokens != 300 {
		t.Errorf("CacheReadTokens = %d, want 300", metrics.CacheReadTokens)
	}
	if metrics.CacheCreationTokens != 150 {
		t.Errorf("CacheCreationTokens = %d, want 150", metrics.CacheCreationTokens)
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
	if metrics.CacheReadTokens != 0 {
		t.Errorf("CacheReadTokens = %d, want 0", metrics.CacheReadTokens)
	}
	if metrics.CacheCreationTokens != 0 {
		t.Errorf("CacheCreationTokens = %d, want 0", metrics.CacheCreationTokens)
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
// CLAUDE CODE BUILD ARGS TESTS
// =============================================================================

func TestClaudeCodeRunner_BuildArgs_EnableBrowser(t *testing.T) {
	r := runner.NewTestClaudeCodeRunner()

	tests := []struct {
		name       string
		config     *domain.RunConfig
		wantChrome bool
	}{
		{
			name: "EnableBrowser true adds --chrome",
			config: &domain.RunConfig{
				RunnerType: domain.RunnerTypeClaudeCode,
				Features:   domain.FeatureFlags{EnableBrowser: true},
			},
			wantChrome: true,
		},
		{
			name: "EnableBrowser false omits --chrome",
			config: &domain.RunConfig{
				RunnerType: domain.RunnerTypeClaudeCode,
				Features:   domain.FeatureFlags{EnableBrowser: false},
			},
			wantChrome: false,
		},
		{
			name: "zero features omits --chrome",
			config: &domain.RunConfig{
				RunnerType: domain.RunnerTypeClaudeCode,
				Features:   domain.FeatureFlags{},
			},
			wantChrome: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := runner.ExecuteRequest{
				RunID:          uuid.New(),
				ResolvedConfig: tt.config,
			}
			args := r.BuildArgsForTest(req)

			hasChrome := false
			for _, arg := range args {
				if arg == "--chrome" {
					hasChrome = true
					break
				}
			}
			if hasChrome != tt.wantChrome {
				t.Errorf("args = %v, wantChrome = %v, hasChrome = %v", args, tt.wantChrome, hasChrome)
			}
		})
	}
}

func TestClaudeCodeRunner_BuildArgs_ExtraFlags(t *testing.T) {
	r := runner.NewTestClaudeCodeRunner()

	tests := []struct {
		name      string
		config    *domain.RunConfig
		wantFlags []string
	}{
		{
			name: "extra flags appended",
			config: &domain.RunConfig{
				RunnerType: domain.RunnerTypeClaudeCode,
				ExtraFlags: domain.RunnerExtraFlags{
					domain.RunnerTypeClaudeCode: []string{"--verbose", "--allowedTools=Read,Write"},
				},
			},
			wantFlags: []string{"--verbose", "--allowedTools=Read,Write"},
		},
		{
			name: "no extra flags for this runner",
			config: &domain.RunConfig{
				RunnerType: domain.RunnerTypeClaudeCode,
				ExtraFlags: domain.RunnerExtraFlags{
					domain.RunnerTypeCodex: []string{"--verbose"},
				},
			},
			wantFlags: nil,
		},
		{
			name: "nil extra flags",
			config: &domain.RunConfig{
				RunnerType: domain.RunnerTypeClaudeCode,
				ExtraFlags: nil,
			},
			wantFlags: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := runner.ExecuteRequest{
				RunID:          uuid.New(),
				ResolvedConfig: tt.config,
			}
			args := r.BuildArgsForTest(req)

			for _, wantFlag := range tt.wantFlags {
				found := false
				for _, arg := range args {
					if arg == wantFlag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected flag %q in args %v", wantFlag, args)
				}
			}
		})
	}
}

func TestClaudeCodeRunner_BuildArgs_FeaturesAndExtraFlags(t *testing.T) {
	r := runner.NewTestClaudeCodeRunner()

	config := &domain.RunConfig{
		RunnerType: domain.RunnerTypeClaudeCode,
		Features:   domain.FeatureFlags{EnableBrowser: true},
		ExtraFlags: domain.RunnerExtraFlags{
			domain.RunnerTypeClaudeCode: []string{"--verbose"},
		},
	}

	req := runner.ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: config,
	}
	args := r.BuildArgsForTest(req)

	hasChrome := false
	hasVerbose := false
	for _, arg := range args {
		if arg == "--chrome" {
			hasChrome = true
		}
		if arg == "--verbose" {
			hasVerbose = true
		}
	}

	if !hasChrome {
		t.Errorf("expected --chrome in args %v", args)
	}
	if !hasVerbose {
		t.Errorf("expected --verbose in args %v", args)
	}
}

func TestClaudeCodeRunner_Capabilities_SupportedFeatures(t *testing.T) {
	r := runner.NewTestClaudeCodeRunner()
	caps := r.Capabilities()

	if len(caps.SupportedFeatures) == 0 {
		t.Fatal("expected SupportedFeatures to be non-empty")
	}
	found := false
	for _, f := range caps.SupportedFeatures {
		if f == "EnableBrowser" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'EnableBrowser' in SupportedFeatures %v", caps.SupportedFeatures)
	}
}

func TestClaudeCodeRunner_Capabilities_AllowedExtraFlags(t *testing.T) {
	r := runner.NewTestClaudeCodeRunner()
	caps := r.Capabilities()

	if len(caps.AllowedExtraFlags) == 0 {
		t.Fatal("expected AllowedExtraFlags to be non-empty")
	}
	expected := map[string]bool{
		"--disallowedTools": false,
	}
	for _, f := range caps.AllowedExtraFlags {
		if _, ok := expected[f]; ok {
			expected[f] = true
		}
	}
	for flag, found := range expected {
		if !found {
			t.Errorf("expected %q in AllowedExtraFlags %v", flag, caps.AllowedExtraFlags)
		}
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

// =============================================================================
// COMPACTION DETECTION TESTS
// =============================================================================

func TestParseCompactCommand(t *testing.T) {
	tests := []struct {
		input     string
		isCompact bool
		focus     string
	}{
		{"/compact", true, ""},
		{"/compact focus on auth", true, "auth"},
		{"/compact focus on API changes", true, "API changes"},
		{"/compact authentication flow", true, "authentication flow"},
		{"  /compact  ", true, ""},
		{"regular message", false, ""},
		{"/compacting", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			isCompact, focus := runner.ParseCompactCommandForTest(tt.input)
			if isCompact != tt.isCompact {
				t.Errorf("isCompact = %v, want %v", isCompact, tt.isCompact)
			}
			if focus != tt.focus {
				t.Errorf("focus = %q, want %q", focus, tt.focus)
			}
		})
	}
}

func TestIsCompactionSummary(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{"<summary>We worked on auth...</summary>", true},
		{"Summary of the conversation so far...", true},
		{"Here is what we did...", false},
	}

	for _, tt := range tests {
		name := tt.content
		if len(name) > 30 {
			name = name[:30]
		}
		t.Run(name, func(t *testing.T) {
			result := runner.IsCompactionSummaryForTest(tt.content)
			if result != tt.expected {
				t.Errorf("isCompactionSummary = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractSummaryContent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"<summary>Auth bug was fixed by updating token validation</summary>",
			"Auth bug was fixed by updating token validation",
		},
		{
			"Some preamble\n<summary>The actual summary</summary>\nSome epilogue",
			"The actual summary",
		},
		{
			"No tags here, just plain text",
			"No tags here, just plain text",
		},
	}

	for _, tt := range tests {
		name := tt.expected
		if len(name) > 20 {
			name = name[:20]
		}
		t.Run(name, func(t *testing.T) {
			result := runner.ExtractSummaryContentForTest(tt.input)
			if result != tt.expected {
				t.Errorf("extractSummaryContent = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseStreamEvents_CompactionFlow(t *testing.T) {
	r := runner.NewTestClaudeCodeRunner()
	runID := uuid.New()

	// Step 1: User sends /compact command via "message" event type
	events1, err := r.ParseStreamEventsForTest(
		runID,
		`{"type":"message","message":{"role":"user","content":"/compact focus on auth"}}`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events1) != 0 {
		t.Errorf("expected 0 events for /compact command, got %d", len(events1))
	}

	// Step 2: Assistant responds with summary
	events2, err := r.ParseStreamEventsForTest(
		runID,
		`{"type":"message","message":{"role":"assistant","content":"<summary>We fixed the auth bug...</summary>"}}`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events2) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events2))
	}

	event := events2[0]
	if event.EventType != domain.EventTypeCompaction {
		t.Errorf("EventType = %s, want %s", event.EventType, domain.EventTypeCompaction)
	}

	data, ok := event.Data.(*domain.CompactionEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *CompactionEventData", event.Data)
	}
	if data.Summary != "We fixed the auth bug..." {
		t.Errorf("Summary = %q, want %q", data.Summary, "We fixed the auth bug...")
	}
	if data.Trigger != "manual" {
		t.Errorf("Trigger = %s, want manual", data.Trigger)
	}
	if data.Focus != "auth" {
		t.Errorf("Focus = %s, want auth", data.Focus)
	}
	if data.OriginalCommand != "/compact focus on auth" {
		t.Errorf("OriginalCommand = %s, want '/compact focus on auth'", data.OriginalCommand)
	}
}

// =============================================================================
// EFFECTIVE PROMPT TESTS
// =============================================================================

func TestExecuteRequest_EffectivePrompt(t *testing.T) {
	tests := []struct {
		name         string
		req          runner.ExecuteRequest
		wantContains []string
		wantExact    string
	}{
		{
			name: "no system prompt returns prompt unchanged",
			req: runner.ExecuteRequest{
				Prompt:       "user message",
				SystemPrompt: "",
			},
			wantExact: "user message",
		},
		{
			name: "no user prompt returns system prompt unchanged",
			req: runner.ExecuteRequest{
				Prompt:       "",
				SystemPrompt: "instructions",
			},
			wantExact: "instructions",
		},
		{
			name: "both prompts wraps system in XML tags",
			req: runner.ExecuteRequest{
				Prompt:       "user data here",
				SystemPrompt: "you are an investigator",
			},
			wantContains: []string{
				"<system-instructions>",
				"you are an investigator",
				"</system-instructions>",
				"user data here",
			},
		},
		{
			name: "system prompt appears before user prompt",
			req: runner.ExecuteRequest{
				Prompt:       "user data",
				SystemPrompt: "system instructions",
			},
			wantContains: []string{
				"<system-instructions>",
				"</system-instructions>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.EffectivePrompt()

			if tt.wantExact != "" {
				if got != tt.wantExact {
					t.Errorf("EffectivePrompt() = %q, want %q", got, tt.wantExact)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("EffectivePrompt() missing %q, got:\n%s", want, got)
				}
			}

			// Verify ordering: system-instructions tag appears before user data
			if tt.req.SystemPrompt != "" && tt.req.Prompt != "" {
				sysIdx := strings.Index(got, "<system-instructions>")
				userIdx := strings.Index(got, tt.req.Prompt)
				if sysIdx >= userIdx {
					t.Errorf("system prompt should appear before user prompt, sys=%d user=%d", sysIdx, userIdx)
				}
			}
		})
	}
}

// =============================================================================
// CLAUDE CODE SYSTEM PROMPT CLI ARG TESTS
// =============================================================================

func TestClaudeCodeRunner_BuildArgs_SystemPrompt(t *testing.T) {
	r := runner.NewTestClaudeCodeRunner()

	t.Run("includes --append-system-prompt when set", func(t *testing.T) {
		req := runner.ExecuteRequest{
			RunID:        uuid.New(),
			SystemPrompt: "You are an investigation agent.",
			Profile: &domain.AgentProfile{
				RunnerType: domain.RunnerTypeClaudeCode,
			},
		}
		args := r.BuildArgsForTest(req)
		found := false
		for i, a := range args {
			if a == "--append-system-prompt" && i+1 < len(args) {
				found = true
				if args[i+1] != "You are an investigation agent." {
					t.Errorf("--append-system-prompt value = %q, want %q", args[i+1], "You are an investigation agent.")
				}
			}
		}
		if !found {
			t.Error("expected --append-system-prompt in args")
		}
	})

	t.Run("omits --append-system-prompt when empty", func(t *testing.T) {
		req := runner.ExecuteRequest{
			RunID:        uuid.New(),
			SystemPrompt: "",
			Profile: &domain.AgentProfile{
				RunnerType: domain.RunnerTypeClaudeCode,
			},
		}
		args := r.BuildArgsForTest(req)
		for _, a := range args {
			if a == "--append-system-prompt" {
				t.Error("expected no --append-system-prompt when system prompt is empty")
			}
		}
	})
}

// =============================================================================
// CLAUDE CODE TAG ENV VAR TESTS
// =============================================================================

func TestClaudeCodeRunner_BuildEnv_AgentTag(t *testing.T) {
	r := runner.NewTestClaudeCodeRunner()

	t.Run("includes CLAUDE_CODE_AGENT_TAG from request tag", func(t *testing.T) {
		runID := uuid.New()
		req := runner.ExecuteRequest{
			RunID: runID,
			Tag:   "heartbeat-team1-agent1-2026-01-01T00-00-00Z",
			Profile: &domain.AgentProfile{
				RunnerType: domain.RunnerTypeClaudeCode,
			},
		}
		env := r.BuildEnvForTest(req)
		found := false
		for _, e := range env {
			if strings.HasPrefix(e, "CLAUDE_CODE_AGENT_TAG=") {
				found = true
				val := strings.TrimPrefix(e, "CLAUDE_CODE_AGENT_TAG=")
				if val != "heartbeat-team1-agent1-2026-01-01T00-00-00Z" {
					t.Errorf("CLAUDE_CODE_AGENT_TAG = %q, want %q", val, "heartbeat-team1-agent1-2026-01-01T00-00-00Z")
				}
			}
		}
		if !found {
			t.Error("expected CLAUDE_CODE_AGENT_TAG in env vars")
		}
	})

	t.Run("defaults to RunID when no explicit tag", func(t *testing.T) {
		runID := uuid.New()
		req := runner.ExecuteRequest{
			RunID: runID,
			Profile: &domain.AgentProfile{
				RunnerType: domain.RunnerTypeClaudeCode,
			},
		}
		env := r.BuildEnvForTest(req)
		expected := runID.String()
		found := false
		for _, e := range env {
			if strings.HasPrefix(e, "CLAUDE_CODE_AGENT_TAG=") {
				found = true
				val := strings.TrimPrefix(e, "CLAUDE_CODE_AGENT_TAG=")
				if val != expected {
					t.Errorf("CLAUDE_CODE_AGENT_TAG = %q, want %q", val, expected)
				}
			}
		}
		if !found {
			t.Error("expected CLAUDE_CODE_AGENT_TAG in env vars")
		}
	})
}
