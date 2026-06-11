// Package config provides centralized configuration for the Agent Inbox scenario.
// This file defines core configuration struct types.
package config

import "time"

// Config holds all tunable parameters for the Agent Inbox scenario.
// Each field is documented with:
//   - Default value
//   - What happens when increased/decreased
//   - Who typically adjusts it (operator, user, agent)
type Config struct {
	Server       ServerConfig
	AI           AIConfig
	Integration  IntegrationConfig
	Resilience   ResilienceConfig
	Storage      StorageConfig
	Templates    TemplatesConfig
	Skills       SkillsConfig
	PromptSync   PromptSyncConfig
	SkillSuggest SkillSuggestConfig
}

// ServerConfig controls HTTP server behavior.
// Audience: Operators deploying the service.
type ServerConfig struct {
	// Port is the HTTP server port. Set via API_PORT env var.
	Port string

	// ReadTimeout is the maximum duration for reading the entire request.
	// Higher = more tolerant of slow clients, lower = faster detection of stalled connections.
	// Default: 30s
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration for writing the response.
	// Higher = allows longer streaming responses, lower = faster timeout detection.
	// Default: 30s (non-streaming), extended automatically for streaming
	WriteTimeout time.Duration

	// IdleTimeout is how long to keep idle connections alive.
	// Higher = better connection reuse, lower = faster resource recovery.
	// Default: 120s
	IdleTimeout time.Duration

	// ShutdownTimeout is the grace period for in-flight requests during shutdown.
	// Higher = more time for requests to complete, lower = faster shutdown.
	// Default: 10s
	ShutdownTimeout time.Duration
}

// AIConfig controls AI model behavior and completion settings.
// Audience: Operators (defaults), Users (per-chat overrides).
type AIConfig struct {
	// DefaultModel is the AI model used for new chats.
	// Set via DEFAULT_AI_MODEL env var.
	// Default: "anthropic/claude-3.5-sonnet"
	DefaultModel string

	// CompletionTimeout is the maximum wait time for AI completions.
	// Higher = allows complex responses, lower = faster failure detection.
	// Default: 120s
	CompletionTimeout time.Duration

	// StreamBufferSize is the buffer size for streaming response chunks.
	// Higher = more buffering before backpressure, lower = more responsive streaming.
	// Default: 4096 bytes
	StreamBufferSize int

	// CompletionReservePercent is the percentage of context window to reserve for the response.
	// Higher = more room for AI response, lower = more room for conversation history.
	// Default: 25 (25% of context reserved for completion)
	CompletionReservePercent int

	// DefaultContextLength is the fallback context length when model info is unavailable.
	// Used when the model registry doesn't have context length data for a model.
	// Default: 8192
	DefaultContextLength int
}

// NamingConfig controls the auto-naming feature powered by local Ollama.
// Audience: Operators tuning naming quality vs speed.
type NamingConfig struct {
	// Role is the Ollama policy role for generating chat names.
	// Set via OLLAMA_NAMING_ROLE env var.
	// Default: "chat.small"
	Role string

	// Temperature controls naming creativity (0.0 = deterministic, 1.0 = creative).
	// Higher = more varied names, lower = more predictable names.
	// Default: 0.3
	Temperature float64

	// MaxTokens limits the generated name length in tokens.
	// Higher = allows longer names, lower = more concise names.
	// Default: 20
	MaxTokens int

	// SummaryMessageLimit is how many messages to include in the naming context.
	// Higher = more context for naming, lower = faster naming.
	// Default: 10
	SummaryMessageLimit int

	// SummaryContentLimit is max characters per message in naming context.
	// Higher = more context per message, lower = faster naming.
	// Default: 200
	SummaryContentLimit int

	// Timeout is the maximum wait for Ollama to generate a name.
	// Default: 30s
	Timeout time.Duration

	// FallbackName is used when naming fails.
	// Default: "New Conversation"
	FallbackName string
}

// TemplatesConfig controls template file storage and management.
// Audience: Operators configuring template storage location.
type TemplatesConfig struct {
	// BasePath is the root directory for template storage.
	// Templates are organized as: BasePath/{defaults,user}/{mode}/{submode}/{id}.json
	// Set via TEMPLATES_BASE_PATH env var.
	// Default: "../templates" (relative to api/ directory, i.e. scenario_root/templates/)
	BasePath string

	// DefaultsDir is the subdirectory within BasePath for default templates.
	// Default: "defaults"
	DefaultsDir string

	// UserDir is the subdirectory within BasePath for user templates.
	// Default: "user"
	UserDir string
}

// SkillsConfig controls skill file storage and management.
// Skills are knowledge modules that provide methodology and expertise for specific tasks.
// Audience: Operators configuring skill storage location.
type SkillsConfig struct {
	// BasePath is the root directory for skill storage.
	// Skills are organized as: BasePath/{defaults,user}/{mode}/{submode}/{id}.json
	// Set via SKILLS_BASE_PATH env var.
	// Default: "../skills" (relative to api/ directory, i.e. scenario_root/skills/)
	BasePath string

	// DefaultsDir is the subdirectory within BasePath for default skills.
	// Default: "defaults"
	DefaultsDir string

	// UserDir is the subdirectory within BasePath for user skills.
	// Default: "user"
	UserDir string
}

// SkillSuggestConfig controls AI-powered skill suggestion based on conversation context.
// Audience: Operators tuning suggestion quality vs latency.
type SkillSuggestConfig struct {
	// Enabled controls whether skill suggestions are available.
	// Set via SKILL_SUGGEST_ENABLED env var.
	// Default: true
	Enabled bool

	// Model is the Ollama role for generating search queries.
	// Set via SKILL_SUGGEST_MODEL env var.
	// Default: "chat.small"
	Model string

	// MaxMessages is the number of recent chat messages to include in context.
	// Higher = richer context for suggestions, lower = faster processing.
	// Default: 10
	MaxMessages int

	// MaxContentLen is the max characters per message in context.
	// Higher = more context per message, lower = faster processing.
	// Default: 300
	MaxContentLen int

	// CacheTTLSeconds is how long to cache suggestion results.
	// Higher = fewer Ollama/search calls, lower = fresher suggestions.
	// Default: 60
	CacheTTLSeconds int

	// MaxSuggestions is the maximum number of skill suggestions to return.
	// Default: 5
	MaxSuggestions int

	// QueryCount is the number of search queries to generate from context.
	// Higher = broader skill coverage, lower = faster response.
	// Default: 3
	QueryCount int

	// Timeout is the maximum wait for the full suggest pipeline.
	// Default: 15s
	Timeout time.Duration
}
