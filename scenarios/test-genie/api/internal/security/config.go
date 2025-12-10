// Package security provides security configuration for agent execution.
package security

import (
	"os"
	"strings"

	"test-genie/internal/shared"
)

// Config holds tunable levers for security validation behavior.
// These control the tradeoff between security strictness and operational flexibility.
//
// # Design Principles
//
// The security configuration follows a "secure by default, extensible when needed" approach:
//   - All defaults are conservative and production-ready
//   - Extensions ADD to the base security rules, they don't replace them
//   - Prompt validation is always enabled and cannot be disabled
//
// # Environment Variables
//
// Security-related levers that can be overridden:
//
//   - SECURITY_EXTRA_ALLOWED_BASH_COMMANDS: Additional bash command prefixes to allow (comma-separated)
//   - SECURITY_PROMPT_VALIDATION_STRICT: Enable stricter prompt validation (default: true)
//   - SECURITY_MAX_PROMPT_LENGTH: Maximum allowed prompt length in characters (default: 100000)
//   - SECURITY_ALLOW_GLOB_PATTERNS: Allow glob patterns in bash commands (default: true for safe commands only)
type Config struct {
	// --- Bash Command Allowlist Extensions ---

	// ExtraAllowedBashCommands are additional bash command prefixes to allow beyond the defaults.
	// These are ADDED to the built-in allowlist, not replacements.
	// Format: Each entry should be a command prefix like "make deploy" or "cargo test".
	// SECURITY NOTE: Be careful what you add here. Each addition expands what agents can execute.
	// Default: empty (use built-in allowlist only)
	ExtraAllowedBashCommands []AllowedBashCommand

	// --- Prompt Validation ---

	// PromptValidationStrict enables stricter prompt scanning for dangerous patterns.
	// When true, more patterns are flagged as potentially dangerous.
	// When false, only the most obvious dangerous patterns are flagged.
	// Default: true
	PromptValidationStrict bool

	// MaxPromptLength is the maximum allowed prompt length in characters.
	// Longer prompts are truncated with a warning.
	// Higher = agents can receive more context, but increases attack surface.
	// Lower = limits prompt injection attacks, but may truncate legitimate context.
	// Range: 1000-1000000 characters. Default: 100000 (100KB).
	MaxPromptLength int

	// --- Glob Pattern Handling ---

	// AllowGlobPatterns controls whether glob patterns (*, ?) are allowed in bash commands.
	// When true, globs are allowed only for commands in the safe glob prefix list.
	// When false, all glob patterns are rejected.
	// Default: true (safe commands only)
	AllowGlobPatterns bool
}

// DefaultSecurityConfig returns a Config with production-ready security defaults.
func DefaultSecurityConfig() Config {
	return Config{
		ExtraAllowedBashCommands: []AllowedBashCommand{},
		PromptValidationStrict:   true,
		MaxPromptLength:          100000,
		AllowGlobPatterns:        true,
	}
}

// LoadSecurityConfigFromEnv loads security configuration from environment variables,
// falling back to defaults for unset values.
func LoadSecurityConfigFromEnv() Config {
	cfg := DefaultSecurityConfig()

	// Parse extra allowed bash commands (comma-separated)
	if v := os.Getenv("SECURITY_EXTRA_ALLOWED_BASH_COMMANDS"); v != "" {
		for _, part := range strings.Split(v, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				cfg.ExtraAllowedBashCommands = append(cfg.ExtraAllowedBashCommands, AllowedBashCommand{
					Prefix:      trimmed,
					Description: "Custom allowed command (from environment)",
				})
			}
		}
	}

	cfg.PromptValidationStrict = shared.EnvBool("SECURITY_PROMPT_VALIDATION_STRICT", true)
	cfg.MaxPromptLength = shared.ClampInt(shared.EnvInt("SECURITY_MAX_PROMPT_LENGTH", cfg.MaxPromptLength), 1000, 1000000)
	cfg.AllowGlobPatterns = shared.EnvBool("SECURITY_ALLOW_GLOB_PATTERNS", true)

	return cfg
}

// Validate checks the configuration for consistency and safety.
// Returns a list of warnings (non-fatal) and an error (fatal).
func (c Config) Validate() (warnings []string, err error) {
	warnings = make([]string, 0)

	// Warn about extra allowed commands
	if len(c.ExtraAllowedBashCommands) > 0 {
		warnings = append(warnings,
			"Security: Extra bash commands have been added to the allowlist. "+
				"Ensure these are intentional and necessary.")
		for _, cmd := range c.ExtraAllowedBashCommands {
			warnings = append(warnings, "  - Added: "+cmd.Prefix)
		}
	}

	// Warn about relaxed prompt validation
	if !c.PromptValidationStrict {
		warnings = append(warnings,
			"Security: Prompt validation is set to non-strict mode. "+
				"Some potentially dangerous patterns may not be flagged.")
	}

	// Warn about very large max prompt length
	if c.MaxPromptLength > 500000 {
		warnings = append(warnings,
			"Security: Max prompt length is very high (>500KB). "+
				"This increases the attack surface for prompt injection.")
	}

	return warnings, nil
}

// GetAllowedBashCommands returns the complete list of allowed bash commands,
// including both defaults and any extensions.
func (c Config) GetAllowedBashCommands() []AllowedBashCommand {
	defaults := defaultAllowedCommands()
	if len(c.ExtraAllowedBashCommands) == 0 {
		return defaults
	}
	// Combine defaults with extras
	combined := make([]AllowedBashCommand, 0, len(defaults)+len(c.ExtraAllowedBashCommands))
	combined = append(combined, defaults...)
	combined = append(combined, c.ExtraAllowedBashCommands...)
	return combined
}
