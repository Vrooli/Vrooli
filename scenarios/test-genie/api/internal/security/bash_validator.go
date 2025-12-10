// Package security provides security validation for agent execution.
// This includes bash command allowlisting, path validation, and prompt scanning.
//
// CHANGE AXIS: Security Policy Updates
// When adding new blocked commands, allowed commands, or policy rules,
// changes are localized to:
//   - defaultAllowedCommands(): Add new allowed command prefixes
//   - defaultBlockedPatterns(): Add new blocked patterns
//   - Config.ExtraAllowedBashCommands: Runtime extensions via env vars
//
// The validator is designed to be created once and reused. Use DefaultValidator()
// for production code to ensure consistent configuration.
package security

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// AllowedBashCommand represents a bash command prefix that is explicitly permitted.
//
// CHANGE_AXIS: Security Policy - Bash Command Allowlist
// When adding new allowed commands, add them to defaultAllowedCommands().
// The GlobSafe field determines whether glob patterns (*, ?) are allowed for this command.
type AllowedBashCommand struct {
	Prefix      string // Command prefix that must match (e.g., "pnpm test", "go test")
	Description string // Human-readable description
	GlobSafe    bool   // Whether glob patterns are safe for this command (e.g., test runners)
}

// BlockedPattern represents a pattern that should be blocked in prompts.
type BlockedPattern struct {
	Pattern     *regexp.Regexp
	Description string
}

// --- Seams for testability and customization ---

// AllowedCommandsProvider is a seam for providing the list of allowed bash commands.
// This enables testing and scenario-specific command customization.
type AllowedCommandsProvider interface {
	GetAllowedCommands() []AllowedBashCommand
}

// BlockedPatternsProvider is a seam for providing the list of blocked patterns.
// This enables testing and scenario-specific pattern customization.
type BlockedPatternsProvider interface {
	GetBlockedPatterns() []BlockedPattern
}

// DefaultAllowedCommandsProvider uses the built-in default allowed commands.
type DefaultAllowedCommandsProvider struct{}

// GetAllowedCommands returns the default allowed command list.
func (p *DefaultAllowedCommandsProvider) GetAllowedCommands() []AllowedBashCommand {
	return defaultAllowedCommands()
}

// DefaultBlockedPatternsProvider uses the built-in default blocked patterns.
type DefaultBlockedPatternsProvider struct{}

// GetBlockedPatterns returns the default blocked pattern list.
func (p *DefaultBlockedPatternsProvider) GetBlockedPatterns() []BlockedPattern {
	return defaultBlockedPatterns()
}

// --- End seams ---

// BashCommandValidator validates bash commands using an allowlist approach.
// SECURITY: This uses allowlist (whitelist) instead of blocklist (blacklist) for stronger security.
// Only explicitly permitted command prefixes are allowed - everything else is rejected.
type BashCommandValidator struct {
	allowedCommands []AllowedBashCommand
	blockedPatterns []BlockedPattern
	config          Config
}

// BashCommandValidatorOption configures a BashCommandValidator during construction.
type BashCommandValidatorOption func(*BashCommandValidator)

// WithAllowedCommandsProvider sets a custom allowed commands provider.
func WithAllowedCommandsProvider(provider AllowedCommandsProvider) BashCommandValidatorOption {
	return func(v *BashCommandValidator) {
		v.allowedCommands = provider.GetAllowedCommands()
	}
}

// WithBlockedPatternsProvider sets a custom blocked patterns provider.
func WithBlockedPatternsProvider(provider BlockedPatternsProvider) BashCommandValidatorOption {
	return func(v *BashCommandValidator) {
		v.blockedPatterns = provider.GetBlockedPatterns()
	}
}

// WithExtraAllowedCommands adds additional allowed commands to the defaults.
func WithExtraAllowedCommands(cmds []AllowedBashCommand) BashCommandValidatorOption {
	return func(v *BashCommandValidator) {
		v.allowedCommands = append(v.allowedCommands, cmds...)
	}
}

// WithSecurityConfig sets a custom security configuration.
func WithSecurityConfig(cfg Config) BashCommandValidatorOption {
	return func(v *BashCommandValidator) {
		v.config = cfg
		// Apply extra allowed commands from config
		if len(cfg.ExtraAllowedBashCommands) > 0 {
			v.allowedCommands = append(v.allowedCommands, cfg.ExtraAllowedBashCommands...)
		}
	}
}

// NewBashCommandValidator creates a validator with the default allowed commands.
// Options can be provided to customize the allowed commands and blocked patterns.
//
// For production code that doesn't need custom options, prefer DefaultValidator()
// which returns a cached instance with consistent configuration.
func NewBashCommandValidator(opts ...BashCommandValidatorOption) *BashCommandValidator {
	v := &BashCommandValidator{
		allowedCommands: defaultAllowedCommands(),
		blockedPatterns: defaultBlockedPatterns(),
		config:          LoadSecurityConfigFromEnv(),
	}

	// Apply any extra commands from the loaded config
	if len(v.config.ExtraAllowedBashCommands) > 0 {
		v.allowedCommands = append(v.allowedCommands, v.config.ExtraAllowedBashCommands...)
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// --- Default Validator (Singleton Pattern) ---
// CHANGE AXIS: Validator instantiation is centralized here.
// If initialization changes (e.g., loading from database), only this needs updating.

var (
	defaultValidator     *BashCommandValidator
	defaultValidatorOnce sync.Once
)

// DefaultValidator returns a cached BashCommandValidator with production defaults.
// This is the preferred way to get a validator in production code.
//
// The validator is created once on first call and cached for subsequent calls.
// This ensures consistent configuration across the application and avoids
// repeatedly parsing environment variables.
//
// Use NewBashCommandValidator() only when you need custom options for testing
// or scenario-specific configuration.
func DefaultValidator() *BashCommandValidator {
	defaultValidatorOnce.Do(func() {
		defaultValidator = NewBashCommandValidator()
	})
	return defaultValidator
}

// ResetDefaultValidator clears the cached validator.
// This is intended for testing only - it allows tests to reset state between runs.
// In production code, the validator should be created once at startup.
func ResetDefaultValidator() {
	defaultValidatorOnce = sync.Once{}
	defaultValidator = nil
}

// GetConfig returns the current security configuration.
func (v *BashCommandValidator) GetConfig() Config {
	return v.config
}

// defaultAllowedCommands returns the default set of allowed bash command prefixes.
//
// CHANGE_AXIS: Security Policy - Bash Command Allowlist
// To add a new allowed command:
//  1. Add it to this list with appropriate Prefix and Description
//  2. Set GlobSafe=true only if the command safely handles glob patterns
//     (typically test runners that execute in isolated environments)
//  3. Consider the security implications - each addition expands agent capabilities
//
// GlobSafe commands can use patterns like "bats *.bats" or "go test ./..."
// Non-GlobSafe commands will reject any glob patterns for safety.
func defaultAllowedCommands() []AllowedBashCommand {
	return []AllowedBashCommand{
		// Test runners - safe read-only operations, GlobSafe for test file patterns
		{Prefix: "pnpm test", Description: "pnpm test runner", GlobSafe: true},
		{Prefix: "pnpm run test", Description: "pnpm test script", GlobSafe: true},
		{Prefix: "npm test", Description: "npm test runner", GlobSafe: true},
		{Prefix: "npm run test", Description: "npm test script", GlobSafe: true},
		{Prefix: "go test", Description: "Go test runner", GlobSafe: true},
		{Prefix: "vitest", Description: "Vitest test runner", GlobSafe: true},
		{Prefix: "jest", Description: "Jest test runner", GlobSafe: true},
		{Prefix: "bats", Description: "BATS test runner", GlobSafe: true},
		{Prefix: "make test", Description: "Make test target", GlobSafe: true},
		{Prefix: "make check", Description: "Make check target", GlobSafe: true},
		{Prefix: "pytest", Description: "Python pytest runner", GlobSafe: true},
		{Prefix: "python -m pytest", Description: "Python pytest module", GlobSafe: true},
		// Build commands - generally safe, but globs could expand unexpectedly
		{Prefix: "pnpm build", Description: "pnpm build", GlobSafe: false},
		{Prefix: "pnpm run build", Description: "pnpm build script", GlobSafe: false},
		{Prefix: "npm run build", Description: "npm build script", GlobSafe: false},
		{Prefix: "go build", Description: "Go build", GlobSafe: false},
		{Prefix: "make build", Description: "Make build target", GlobSafe: false},
		{Prefix: "make", Description: "Make default target", GlobSafe: false},
		// Linters and formatters - read-only or in-place safe
		{Prefix: "pnpm lint", Description: "pnpm lint", GlobSafe: false},
		{Prefix: "pnpm run lint", Description: "pnpm lint script", GlobSafe: false},
		{Prefix: "npm run lint", Description: "npm lint script", GlobSafe: false},
		{Prefix: "eslint", Description: "ESLint", GlobSafe: false},
		{Prefix: "prettier", Description: "Prettier formatter", GlobSafe: false},
		{Prefix: "gofmt", Description: "Go formatter", GlobSafe: false},
		{Prefix: "gofumpt", Description: "Go strict formatter", GlobSafe: false},
		{Prefix: "golangci-lint", Description: "Go linter", GlobSafe: false},
		// Type checking - read-only
		{Prefix: "pnpm typecheck", Description: "pnpm typecheck", GlobSafe: false},
		{Prefix: "pnpm run typecheck", Description: "pnpm typecheck script", GlobSafe: false},
		{Prefix: "tsc", Description: "TypeScript compiler", GlobSafe: false},
		// Safe inspection commands - ls is GlobSafe (read-only directory listing)
		{Prefix: "ls", Description: "List directory", GlobSafe: true},
		{Prefix: "pwd", Description: "Print working directory", GlobSafe: false},
		{Prefix: "which", Description: "Locate command", GlobSafe: false},
		{Prefix: "wc", Description: "Word count", GlobSafe: false},
		{Prefix: "diff", Description: "Compare files", GlobSafe: false},
		// Git read-only commands
		{Prefix: "git status", Description: "Git status", GlobSafe: false},
		{Prefix: "git diff", Description: "Git diff", GlobSafe: false},
		{Prefix: "git log", Description: "Git log", GlobSafe: false},
		{Prefix: "git show", Description: "Git show", GlobSafe: false},
		{Prefix: "git branch", Description: "Git branch list", GlobSafe: false},
		{Prefix: "git remote", Description: "Git remote list", GlobSafe: false},
	}
}

// defaultBlockedPatterns returns patterns that should be blocked in prompts.
// This is defense-in-depth - even if an agent tries to execute commands via prompt injection,
// the allowlist validation will block them. But we still warn about suspicious patterns.
func defaultBlockedPatterns() []BlockedPattern {
	return []BlockedPattern{
		// Git destructive commands
		{regexp.MustCompile(`(?i)\bgit\s+(checkout|reset\s+--hard|branch\s+-[dD]|push|clean|rebase|merge|cherry-pick|revert|commit|add\s)`), "destructive/mutating git operation"},
		{regexp.MustCompile(`(?i)\bgit\s+stash`), "git stash operation"},
		// Filesystem destructive commands
		{regexp.MustCompile(`(?i)\brm\s`), "rm command"},
		{regexp.MustCompile(`(?i)\bmv\s`), "mv command"},
		{regexp.MustCompile(`(?i)\bcp\s`), "cp command"},
		{regexp.MustCompile(`(?i)\bmkdir\s`), "mkdir command"},
		{regexp.MustCompile(`(?i)\brmdir\s`), "rmdir command"},
		{regexp.MustCompile(`(?i)\btouch\s`), "touch command"},
		// Permission/ownership changes
		{regexp.MustCompile(`(?i)\bchmod\s`), "chmod command"},
		{regexp.MustCompile(`(?i)\bchown\s`), "chown command"},
		// System operations
		{regexp.MustCompile(`(?i)\bsudo\s`), "sudo command"},
		{regexp.MustCompile(`(?i)\bsu\s`), "su command"},
		{regexp.MustCompile(`(?i)\bsystemctl\s`), "systemctl command"},
		{regexp.MustCompile(`(?i)\bservice\s`), "service command"},
		{regexp.MustCompile(`(?i)\bkill\s`), "kill command"},
		{regexp.MustCompile(`(?i)\bpkill\s`), "pkill command"},
		// Database operations
		{regexp.MustCompile(`(?i)\bDROP\s`), "SQL DROP"},
		{regexp.MustCompile(`(?i)\bTRUNCATE\s`), "SQL TRUNCATE"},
		{regexp.MustCompile(`(?i)\bDELETE\s+FROM\s`), "SQL DELETE"},
		{regexp.MustCompile(`(?i)\bUPDATE\s+\w+\s+SET\s`), "SQL UPDATE"},
		{regexp.MustCompile(`(?i)\bINSERT\s+INTO\s`), "SQL INSERT"},
		// Package managers
		{regexp.MustCompile(`(?i)\b(npm|pnpm|yarn)\s+(install|add|remove|uninstall|update|upgrade)\s`), "package installation"},
		{regexp.MustCompile(`(?i)\bgo\s+(get|install|mod\s+tidy)\s`), "go package operation"},
		{regexp.MustCompile(`(?i)\bpip\s+(install|uninstall)\s`), "pip operation"},
		// Network operations that could be dangerous
		{regexp.MustCompile(`(?i)\bcurl\s+.*\|\s*(sh|bash)`), "remote script execution"},
		{regexp.MustCompile(`(?i)\bwget\s+.*\|\s*(sh|bash)`), "remote script execution"},
		{regexp.MustCompile(`(?i)\bcurl\s+.*(-X|--request)\s*(POST|PUT|DELETE|PATCH)`), "HTTP mutation"},
		// Shell execution
		{regexp.MustCompile(`(?i)\beval\s`), "eval command"},
		{regexp.MustCompile(`(?i)\bexec\s`), "exec command"},
		{regexp.MustCompile(`(?i)\bsource\s`), "source command"},
		{regexp.MustCompile(`(?i)\b\.\s+/`), "dot source command"},
	}
}

// CommandMatchResult describes the result of matching a command against the allowlist.
//
// CHANGE_AXIS: Glob Safety
// The GlobSafe field is now part of AllowedBashCommand struct, eliminating the need
// for a separate safeGlobPrefixes map. This ensures glob safety stays coupled to
// the command definition - a single source of truth.
type CommandMatchResult struct {
	Matched       bool
	MatchedPrefix string // The prefix that matched (if any)
	HasGlob       bool   // Whether the pattern contains glob characters
	GlobSafe      bool   // Whether the matched command allows glob patterns
}

// MatchCommandToAllowlist checks if a command pattern matches any allowed prefix.
// This is the core decision for "is this command allowed at all?"
//
// Matching rules:
//   - Exact match: "ls" matches allowed prefix "ls"
//   - Prefix with space: "ls -la" matches allowed prefix "ls"
//   - Prefix with tab: "ls\t-la" matches allowed prefix "ls"
//   - Case insensitive matching
//
// The result includes GlobSafe from the matched AllowedBashCommand, providing
// a single source of truth for whether globs are allowed with this command.
func (v *BashCommandValidator) MatchCommandToAllowlist(pattern string) CommandMatchResult {
	patternLower := strings.ToLower(strings.TrimSpace(pattern))
	hasGlob := strings.Contains(pattern, "*") || strings.Contains(pattern, "?")

	result := CommandMatchResult{
		Matched:  false,
		HasGlob:  hasGlob,
		GlobSafe: false,
	}

	for _, allowed := range v.allowedCommands {
		prefixLower := strings.ToLower(allowed.Prefix)

		// Check: exact match, or prefix followed by space/tab
		if patternLower == prefixLower ||
			strings.HasPrefix(patternLower, prefixLower+" ") ||
			strings.HasPrefix(patternLower, prefixLower+"\t") {
			result.Matched = true
			result.MatchedPrefix = allowed.Prefix
			result.GlobSafe = allowed.GlobSafe
			return result
		}
	}

	return result
}

// GlobValidationResult describes whether glob usage is permitted for a command.
type GlobValidationResult struct {
	Allowed bool
	Reason  string
}

// ValidateGlobUsage checks if glob patterns can be used based on a CommandMatchResult.
// Uses the GlobSafe field from the match result directly, avoiding redundant lookups.
func (v *BashCommandValidator) ValidateGlobUsage(matchResult CommandMatchResult) GlobValidationResult {
	if !matchResult.HasGlob {
		return GlobValidationResult{Allowed: true, Reason: "No glob patterns present"}
	}

	// Check if globs are globally disabled
	if !v.config.AllowGlobPatterns {
		return GlobValidationResult{
			Allowed: false,
			Reason:  "glob patterns (*,?) are disabled via security configuration",
		}
	}

	// Use the GlobSafe from the match result (single source of truth)
	if matchResult.GlobSafe {
		return GlobValidationResult{
			Allowed: true,
			Reason:  fmt.Sprintf("Command '%s' is safe for glob patterns", matchResult.MatchedPrefix),
		}
	}

	return GlobValidationResult{
		Allowed: false,
		Reason:  fmt.Sprintf("glob patterns (*,?) are not allowed with '%s'; only test runners support glob patterns", matchResult.MatchedPrefix),
	}
}

// ValidateBashPattern checks if a bash pattern from bash(...) syntax contains only allowed commands.
// Returns an error if any command in the pattern is not in the allowlist.
//
// Uses MatchCommandToAllowlist and ValidateGlobUsageFromMatch for the core decisions.
// The GlobSafe field from the match result is the single source of truth for glob safety.
func (v *BashCommandValidator) ValidateBashPattern(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("empty bash pattern")
	}

	// Decision 1: Does this command match any allowed prefix?
	// This also returns GlobSafe from the matched command definition.
	matchResult := v.MatchCommandToAllowlist(pattern)
	if !matchResult.Matched {
		return fmt.Errorf("bash command '%s' is not in the allowlist; only test runners and safe inspection commands are permitted", pattern)
	}

	// Decision 2: If it has globs, are they allowed for this command?
	// Uses GlobSafe from the match result (single source of truth).
	if matchResult.HasGlob {
		globResult := v.ValidateGlobUsage(matchResult)
		if !globResult.Allowed {
			return fmt.Errorf("%s", globResult.Reason)
		}
	}

	return nil
}

// Validate checks if the given allowed tools list contains only permitted commands.
func (v *BashCommandValidator) Validate(allowedTools []string) error {
	for _, tool := range allowedTools {
		toolLower := strings.ToLower(strings.TrimSpace(tool))

		if toolLower == "bash" || toolLower == "*" {
			return fmt.Errorf("unrestricted bash access is not allowed for spawned agents; use scoped patterns like 'bash(pnpm test|go test)'")
		}

		if strings.HasPrefix(toolLower, "bash(") && strings.HasSuffix(tool, ")") {
			innerPatterns := tool[5 : len(tool)-1]
			for _, pattern := range strings.Split(innerPatterns, "|") {
				pattern = strings.TrimSpace(pattern)
				if pattern == "" {
					continue
				}
				if err := v.ValidateBashPattern(pattern); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ValidatePrompt scans the prompt for obviously dangerous instructions.
// Returns an error if the prompt contains blocked patterns or exceeds length limits.
func (v *BashCommandValidator) ValidatePrompt(prompt string) error {
	// Check prompt length
	if v.config.MaxPromptLength > 0 && len(prompt) > v.config.MaxPromptLength {
		return fmt.Errorf("prompt exceeds maximum length (%d > %d characters)", len(prompt), v.config.MaxPromptLength)
	}

	// Scan for blocked patterns
	for _, blocked := range v.blockedPatterns {
		if blocked.Pattern.MatchString(prompt) {
			return fmt.Errorf("prompt contains potentially dangerous pattern: %s", blocked.Description)
		}
	}
	return nil
}

// GetAllowedCommands returns a copy of the allowed command list for documentation purposes.
func (v *BashCommandValidator) GetAllowedCommands() []AllowedBashCommand {
	result := make([]AllowedBashCommand, len(v.allowedCommands))
	copy(result, v.allowedCommands)
	return result
}

// GetBlockedPatterns returns a copy of the blocked patterns for documentation purposes.
func (v *BashCommandValidator) GetBlockedPatterns() []BlockedPattern {
	result := make([]BlockedPattern, len(v.blockedPatterns))
	copy(result, v.blockedPatterns)
	return result
}
