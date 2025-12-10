package agents

import (
	"testing"
)

func TestExecutionBuilder(t *testing.T) {
	defaults := DefaultConfig()

	t.Run("builds valid config", func(t *testing.T) {
		builder := NewExecutionBuilder(defaults)
		result := builder.WithInput(ExecutionBuilderInput{
			AgentID:     "test-123",
			Scenario:    "test-genie",
			Model:       "openrouter/anthropic/claude-3-opus",
			Prompt:      "Write tests for user handler",
			RepoRoot:    "/home/user/Vrooli",
			APIEndpoint: "http://localhost:8080",
		}).Build()

		if !result.Success {
			t.Errorf("Expected success, got errors: %v", result.Errors)
		}

		if result.Config.Command != "claude" {
			t.Errorf("Expected command 'claude', got %q", result.Config.Command)
		}

		if result.Config.WorkingDir != "/home/user/Vrooli/scenarios/test-genie" {
			t.Errorf("Expected working dir to include scenario, got %q", result.Config.WorkingDir)
		}

		// Check environment variables were set
		if result.Config.Environment["TEST_GENIE_AGENT_ID"] != "test-123" {
			t.Error("Expected TEST_GENIE_AGENT_ID to be set")
		}
	})

	t.Run("requires all mandatory fields", func(t *testing.T) {
		builder := NewExecutionBuilder(defaults)
		result := builder.WithInput(ExecutionBuilderInput{
			// All fields missing
		}).Build()

		if result.Success {
			t.Error("Expected failure for missing required fields")
		}

		// Should have multiple errors
		if len(result.Errors) < 4 {
			t.Errorf("Expected at least 4 errors for missing fields, got %d: %v", len(result.Errors), result.Errors)
		}
	})

	t.Run("uses defaults for optional parameters", func(t *testing.T) {
		builder := NewExecutionBuilder(defaults)
		result := builder.WithInput(ExecutionBuilderInput{
			AgentID:     "test-123",
			Scenario:    "test-genie",
			Model:       "openrouter/anthropic/claude-3-opus",
			Prompt:      "Test prompt",
			RepoRoot:    "/home/user/Vrooli",
			APIEndpoint: "http://localhost:8080",
			// No timeout specified - should use default
		}).Build()

		if !result.Success {
			t.Errorf("Expected success, got errors: %v", result.Errors)
		}

		// Should use default timeout
		if result.Config.TimeoutSecs != defaults.DefaultTimeoutSeconds {
			t.Errorf("Expected default timeout %d, got %d", defaults.DefaultTimeoutSeconds, result.Config.TimeoutSecs)
		}
	})

	t.Run("includes allowed tools in args", func(t *testing.T) {
		builder := NewExecutionBuilder(defaults)
		result := builder.WithInput(ExecutionBuilderInput{
			AgentID:      "test-123",
			Scenario:     "test-genie",
			Model:        "openrouter/anthropic/claude-3-opus",
			Prompt:       "Test prompt",
			RepoRoot:     "/home/user/Vrooli",
			APIEndpoint:  "http://localhost:8080",
			AllowedTools: []string{"read", "edit", "bash(pnpm test)"},
		}).Build()

		if !result.Success {
			t.Errorf("Expected success, got errors: %v", result.Errors)
		}

		// Check for --allowed-tools in args
		foundTools := false
		for i, arg := range result.Config.Args {
			if arg == "--allowed-tools" {
				foundTools = true
				if i+1 < len(result.Config.Args) {
					if result.Config.Args[i+1] != "read,edit,bash(pnpm test)" {
						t.Errorf("Expected tools string, got %q", result.Config.Args[i+1])
					}
				}
				break
			}
		}
		if !foundTools {
			t.Error("Expected --allowed-tools in args")
		}
	})
}

func TestDecideTimeoutSeconds(t *testing.T) {
	defaultTimeout := 900

	t.Run("uses request value when provided", func(t *testing.T) {
		decision := DecideTimeoutSeconds(300, defaultTimeout)
		if decision.Value.(int) != 300 {
			t.Errorf("Expected 300, got %v", decision.Value)
		}
		if decision.Source != "request" {
			t.Errorf("Expected source 'request', got %q", decision.Source)
		}
	})

	t.Run("uses default when request is 0", func(t *testing.T) {
		decision := DecideTimeoutSeconds(0, defaultTimeout)
		if decision.Value.(int) != defaultTimeout {
			t.Errorf("Expected %d, got %v", defaultTimeout, decision.Value)
		}
		if decision.Source != "default" {
			t.Errorf("Expected source 'default', got %q", decision.Source)
		}
	})

	t.Run("clamps value below minimum", func(t *testing.T) {
		decision := DecideTimeoutSeconds(10, defaultTimeout) // 10s is below 60s min
		if decision.Value.(int) != 60 {
			t.Errorf("Expected 60 (minimum), got %v", decision.Value)
		}
		if !decision.WasClamped {
			t.Error("Expected WasClamped to be true")
		}
	})

	t.Run("clamps value above maximum", func(t *testing.T) {
		decision := DecideTimeoutSeconds(7200, defaultTimeout) // 7200s is above 3600s max
		if decision.Value.(int) != 3600 {
			t.Errorf("Expected 3600 (maximum), got %v", decision.Value)
		}
		if !decision.WasClamped {
			t.Error("Expected WasClamped to be true")
		}
	})
}

func TestDecideMaxTurns(t *testing.T) {
	defaultTurns := 50

	t.Run("uses request value when provided", func(t *testing.T) {
		decision := DecideMaxTurns(100, defaultTurns)
		if decision.Value.(int) != 100 {
			t.Errorf("Expected 100, got %v", decision.Value)
		}
		if decision.Source != "request" {
			t.Errorf("Expected source 'request', got %q", decision.Source)
		}
	})

	t.Run("clamps below minimum", func(t *testing.T) {
		decision := DecideMaxTurns(2, defaultTurns)
		if decision.Value.(int) != 5 { // minimum is 5
			t.Errorf("Expected 5, got %v", decision.Value)
		}
		if !decision.WasClamped {
			t.Error("Expected WasClamped to be true")
		}
	})
}

func TestDecideMaxFilesChanged(t *testing.T) {
	defaultFiles := 50

	t.Run("uses request value when provided", func(t *testing.T) {
		decision := DecideMaxFilesChanged(100, defaultFiles)
		if decision.Value.(int) != 100 {
			t.Errorf("Expected 100, got %v", decision.Value)
		}
	})

	t.Run("clamps above maximum", func(t *testing.T) {
		decision := DecideMaxFilesChanged(1000, defaultFiles) // max is 500
		if decision.Value.(int) != 500 {
			t.Errorf("Expected 500, got %v", decision.Value)
		}
		if !decision.WasClamped {
			t.Error("Expected WasClamped to be true")
		}
	})
}

func TestDecideMaxBytesWritten(t *testing.T) {
	defaultBytes := int64(1024 * 1024) // 1MB

	t.Run("uses request value when provided", func(t *testing.T) {
		decision := DecideMaxBytesWritten(5*1024*1024, defaultBytes)
		if decision.Value.(int64) != 5*1024*1024 {
			t.Errorf("Expected 5MB, got %v", decision.Value)
		}
	})

	t.Run("clamps below minimum", func(t *testing.T) {
		decision := DecideMaxBytesWritten(100, defaultBytes) // 100 bytes below 1KB min
		if decision.Value.(int64) != 1024 {
			t.Errorf("Expected 1024, got %v", decision.Value)
		}
		if !decision.WasClamped {
			t.Error("Expected WasClamped to be true")
		}
	})
}

func TestFormatCommandForDisplay(t *testing.T) {
	t.Run("truncates long prompts", func(t *testing.T) {
		config := ExecutionConfig{
			Command: "claude",
			Args:    []string{"--print", "this is a very long prompt that should definitely be truncated because it exceeds one hundred characters in length"},
		}

		display := config.FormatCommandForDisplay()
		if len(display) > 200 { // reasonable display length
			t.Errorf("Display string too long: %d chars", len(display))
		}
		if !containsSubstring(display, "...") {
			t.Error("Expected truncation indicator '...'")
		}
	})
}

// containsSubstring is defined in service_test.go - reusing that implementation
