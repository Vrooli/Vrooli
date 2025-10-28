package server

import "time"

// Settings validation constraints
const (
	// Max turns constraints
	MinMaxTurns     = 1
	MaxMaxTurns     = 500
	DefaultMaxTurns = 80

	// Agent execution timeout constraints (in seconds)
	// NOTE: This is the total runtime limit for the entire agent execution.
	// Separate from max_turns which limits conversation iterations.
	MinTimeoutSeconds     = 60    // 1 minute
	MaxTimeoutSeconds     = 14400 // 4 hours
	DefaultTimeoutSeconds = 4800  // 80 minutes

	// Concurrent slots constraints
	MinConcurrentSlots     = 1
	MaxConcurrentSlots     = 20
	DefaultConcurrentSlots = 2

	// Refresh interval constraints (in seconds)
	MinRefreshInterval     = 10  // 10 seconds
	MaxRefreshInterval     = 600 // 10 minutes
	DefaultRefreshInterval = 45  // 45 seconds
)

// Internal operation timeouts (not user-configurable)
const (
	// ScenarioRestartTimeout is the maximum time allowed for a scenario restart operation
	ScenarioRestartTimeout = 2 * time.Minute

	// RateLimitCacheTTL is how long rate limit status is cached before re-checking
	RateLimitCacheTTL = 1 * time.Second

	// DefaultRateLimitResetDuration is the fallback duration when rate limit reset time cannot be parsed
	DefaultRateLimitResetDuration = 5 * time.Minute
)

// SettingConstraint describes the allowed range for a numeric setting
type SettingConstraint struct {
	Min         int    `json:"min"`
	Max         int    `json:"max"`
	Default     int    `json:"default"`
	Description string `json:"description"`
}

// ProviderOption describes an available AI provider option
type ProviderOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// SettingsConstraints contains all validation constraints and options
type SettingsConstraints struct {
	Numeric   map[string]SettingConstraint `json:"numeric"`
	Providers []ProviderOption             `json:"providers"`
}

// GetSettingsConstraints returns all setting constraints for the UI
func GetSettingsConstraints() SettingsConstraints {
	return SettingsConstraints{
		Numeric: map[string]SettingConstraint{
			"max_turns": {
				Min:         MinMaxTurns,
				Max:         MaxMaxTurns,
				Default:     DefaultMaxTurns,
				Description: "Maximum number of turns the agent can take before stopping",
			},
			"timeout_seconds": {
				Min:         MinTimeoutSeconds,
				Max:         MaxTimeoutSeconds,
				Default:     DefaultTimeoutSeconds,
				Description: "Maximum execution time in seconds (total runtime limit)",
			},
			"concurrent_slots": {
				Min:         MinConcurrentSlots,
				Max:         MaxConcurrentSlots,
				Default:     DefaultConcurrentSlots,
				Description: "Number of issues that can be processed simultaneously",
			},
			"refresh_interval": {
				Min:         MinRefreshInterval,
				Max:         MaxRefreshInterval,
				Default:     DefaultRefreshInterval,
				Description: "How often the processor checks for new issues (in seconds)",
			},
		},
		Providers: []ProviderOption{
			{
				Value:       "codex",
				Label:       "Codex",
				Description: "Anthropic's Codex agent - optimized for longer tasks and complex problem solving",
			},
			{
				Value:       "claude-code",
				Label:       "Claude Code",
				Description: "Claude Code CLI - standard agent for general purpose tasks",
			},
		},
	}
}
