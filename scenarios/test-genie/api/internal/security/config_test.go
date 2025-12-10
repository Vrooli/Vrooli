package security

import (
	"os"
	"testing"
)

func TestDefaultSecurityConfig(t *testing.T) {
	cfg := DefaultSecurityConfig()

	t.Run("extra allowed bash commands defaults to empty", func(t *testing.T) {
		if len(cfg.ExtraAllowedBashCommands) != 0 {
			t.Errorf("Expected ExtraAllowedBashCommands to be empty, got %d items", len(cfg.ExtraAllowedBashCommands))
		}
	})

	t.Run("prompt validation strict defaults to true", func(t *testing.T) {
		if !cfg.PromptValidationStrict {
			t.Error("Expected PromptValidationStrict=true")
		}
	})

	t.Run("max prompt length defaults to 100000", func(t *testing.T) {
		if cfg.MaxPromptLength != 100000 {
			t.Errorf("Expected MaxPromptLength=100000, got %d", cfg.MaxPromptLength)
		}
	})

	t.Run("allow glob patterns defaults to true", func(t *testing.T) {
		if !cfg.AllowGlobPatterns {
			t.Error("Expected AllowGlobPatterns=true")
		}
	})
}

func TestLoadSecurityConfigFromEnv(t *testing.T) {
	// Clean up any env vars after test
	defer func() {
		os.Unsetenv("SECURITY_EXTRA_ALLOWED_BASH_COMMANDS")
		os.Unsetenv("SECURITY_PROMPT_VALIDATION_STRICT")
		os.Unsetenv("SECURITY_MAX_PROMPT_LENGTH")
		os.Unsetenv("SECURITY_ALLOW_GLOB_PATTERNS")
	}()

	t.Run("loads extra allowed bash commands from environment", func(t *testing.T) {
		os.Setenv("SECURITY_EXTRA_ALLOWED_BASH_COMMANDS", "cargo test,cargo build,make deploy")

		cfg := LoadSecurityConfigFromEnv()

		if len(cfg.ExtraAllowedBashCommands) != 3 {
			t.Errorf("Expected 3 extra commands, got %d", len(cfg.ExtraAllowedBashCommands))
		}

		expectedPrefixes := []string{"cargo test", "cargo build", "make deploy"}
		for i, expected := range expectedPrefixes {
			if cfg.ExtraAllowedBashCommands[i].Prefix != expected {
				t.Errorf("Expected prefix %q at index %d, got %q", expected, i, cfg.ExtraAllowedBashCommands[i].Prefix)
			}
		}
	})

	t.Run("loads prompt validation strict from environment", func(t *testing.T) {
		os.Setenv("SECURITY_PROMPT_VALIDATION_STRICT", "false")

		cfg := LoadSecurityConfigFromEnv()

		if cfg.PromptValidationStrict {
			t.Error("Expected PromptValidationStrict=false")
		}
	})

	t.Run("loads max prompt length from environment", func(t *testing.T) {
		os.Setenv("SECURITY_MAX_PROMPT_LENGTH", "50000")

		cfg := LoadSecurityConfigFromEnv()

		if cfg.MaxPromptLength != 50000 {
			t.Errorf("Expected MaxPromptLength=50000, got %d", cfg.MaxPromptLength)
		}
	})

	t.Run("clamps max prompt length to valid range", func(t *testing.T) {
		os.Setenv("SECURITY_MAX_PROMPT_LENGTH", "500") // Below minimum 1000

		cfg := LoadSecurityConfigFromEnv()

		if cfg.MaxPromptLength != 1000 {
			t.Errorf("Expected MaxPromptLength clamped to 1000, got %d", cfg.MaxPromptLength)
		}

		os.Setenv("SECURITY_MAX_PROMPT_LENGTH", "2000000") // Above maximum 1000000

		cfg = LoadSecurityConfigFromEnv()

		if cfg.MaxPromptLength != 1000000 {
			t.Errorf("Expected MaxPromptLength clamped to 1000000, got %d", cfg.MaxPromptLength)
		}
	})

	t.Run("loads allow glob patterns from environment", func(t *testing.T) {
		os.Setenv("SECURITY_ALLOW_GLOB_PATTERNS", "false")

		cfg := LoadSecurityConfigFromEnv()

		if cfg.AllowGlobPatterns {
			t.Error("Expected AllowGlobPatterns=false")
		}
	})

	t.Run("handles empty extra commands gracefully", func(t *testing.T) {
		os.Setenv("SECURITY_EXTRA_ALLOWED_BASH_COMMANDS", "")

		cfg := LoadSecurityConfigFromEnv()

		if len(cfg.ExtraAllowedBashCommands) != 0 {
			t.Errorf("Expected empty extra commands for empty env var, got %d", len(cfg.ExtraAllowedBashCommands))
		}
	})

	t.Run("trims whitespace from extra commands", func(t *testing.T) {
		os.Setenv("SECURITY_EXTRA_ALLOWED_BASH_COMMANDS", " cargo test , cargo build , make deploy ")

		cfg := LoadSecurityConfigFromEnv()

		if len(cfg.ExtraAllowedBashCommands) != 3 {
			t.Errorf("Expected 3 extra commands, got %d", len(cfg.ExtraAllowedBashCommands))
		}

		if cfg.ExtraAllowedBashCommands[0].Prefix != "cargo test" {
			t.Errorf("Expected trimmed prefix 'cargo test', got %q", cfg.ExtraAllowedBashCommands[0].Prefix)
		}
	})
}

func TestSecurityConfigValidate(t *testing.T) {
	t.Run("default config has no warnings", func(t *testing.T) {
		cfg := DefaultSecurityConfig()
		warnings, err := cfg.Validate()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(warnings) != 0 {
			t.Errorf("Expected no warnings for default config, got %d", len(warnings))
		}
	})

	t.Run("warns about extra allowed commands", func(t *testing.T) {
		cfg := DefaultSecurityConfig()
		cfg.ExtraAllowedBashCommands = []AllowedBashCommand{
			{Prefix: "cargo test", Description: "Rust test runner"},
		}

		warnings, err := cfg.Validate()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(warnings) < 1 {
			t.Error("Expected warnings about extra commands")
		}
	})

	t.Run("warns about relaxed prompt validation", func(t *testing.T) {
		cfg := DefaultSecurityConfig()
		cfg.PromptValidationStrict = false

		warnings, err := cfg.Validate()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		found := false
		for _, w := range warnings {
			if w == "Security: Prompt validation is set to non-strict mode. Some potentially dangerous patterns may not be flagged." {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected warning about non-strict prompt validation")
		}
	})

	t.Run("warns about very large max prompt length", func(t *testing.T) {
		cfg := DefaultSecurityConfig()
		cfg.MaxPromptLength = 600000

		warnings, err := cfg.Validate()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		found := false
		for _, w := range warnings {
			if w == "Security: Max prompt length is very high (>500KB). This increases the attack surface for prompt injection." {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected warning about very large max prompt length")
		}
	})
}

func TestSecurityConfigGetAllowedBashCommands(t *testing.T) {
	t.Run("returns defaults when no extras", func(t *testing.T) {
		cfg := DefaultSecurityConfig()
		commands := cfg.GetAllowedBashCommands()

		// Should have all the default commands
		if len(commands) == 0 {
			t.Error("Expected default commands, got empty list")
		}

		// Check for a known default
		found := false
		for _, cmd := range commands {
			if cmd.Prefix == "pnpm test" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'pnpm test' in default commands")
		}
	})

	t.Run("includes extras with defaults", func(t *testing.T) {
		cfg := DefaultSecurityConfig()
		cfg.ExtraAllowedBashCommands = []AllowedBashCommand{
			{Prefix: "cargo test", Description: "Rust test runner"},
		}

		commands := cfg.GetAllowedBashCommands()

		// Should have defaults plus extra
		foundDefault := false
		foundExtra := false
		for _, cmd := range commands {
			if cmd.Prefix == "pnpm test" {
				foundDefault = true
			}
			if cmd.Prefix == "cargo test" {
				foundExtra = true
			}
		}

		if !foundDefault {
			t.Error("Expected default commands to be included")
		}
		if !foundExtra {
			t.Error("Expected extra command to be included")
		}
	})
}

func TestBashValidatorWithSecurityConfig(t *testing.T) {
	// Clean up any env vars after test
	defer func() {
		os.Unsetenv("SECURITY_EXTRA_ALLOWED_BASH_COMMANDS")
		os.Unsetenv("SECURITY_ALLOW_GLOB_PATTERNS")
		os.Unsetenv("SECURITY_MAX_PROMPT_LENGTH")
	}()

	t.Run("extra commands from config are allowed", func(t *testing.T) {
		cfg := DefaultSecurityConfig()
		cfg.ExtraAllowedBashCommands = []AllowedBashCommand{
			{Prefix: "cargo test", Description: "Rust test runner"},
		}

		validator := NewBashCommandValidator(WithSecurityConfig(cfg))

		err := validator.ValidateBashPattern("cargo test ./src/")
		if err != nil {
			t.Errorf("Expected 'cargo test' to be allowed, got error: %v", err)
		}
	})

	t.Run("respects glob patterns config", func(t *testing.T) {
		cfg := DefaultSecurityConfig()
		cfg.AllowGlobPatterns = false

		validator := NewBashCommandValidator(WithSecurityConfig(cfg))

		// Glob patterns should be rejected even for safe commands
		err := validator.ValidateBashPattern("bats *.bats")
		if err == nil {
			t.Error("Expected glob patterns to be rejected when disabled")
		}
	})

	t.Run("respects max prompt length", func(t *testing.T) {
		cfg := DefaultSecurityConfig()
		cfg.MaxPromptLength = 100

		validator := NewBashCommandValidator(WithSecurityConfig(cfg))

		longPrompt := make([]byte, 200)
		for i := range longPrompt {
			longPrompt[i] = 'a'
		}

		err := validator.ValidatePrompt(string(longPrompt))
		if err == nil {
			t.Error("Expected prompt length validation to fail")
		}
	})

	t.Run("loads config from environment by default", func(t *testing.T) {
		os.Setenv("SECURITY_EXTRA_ALLOWED_BASH_COMMANDS", "custom-command")

		validator := NewBashCommandValidator()

		// Should have loaded the extra command from environment
		err := validator.ValidateBashPattern("custom-command --flag")
		if err != nil {
			t.Errorf("Expected 'custom-command' from env to be allowed, got error: %v", err)
		}
	})
}
