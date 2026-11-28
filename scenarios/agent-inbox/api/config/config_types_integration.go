// Package config provides centralized configuration for the Agent Inbox scenario.
// This file defines integration, resilience, storage, and sync configuration types.
package config

import "time"

// IntegrationConfig controls external service connections.
// Audience: Operators configuring service mesh.
type IntegrationConfig struct {
	// AgentManagerURL is the agent-manager API endpoint.
	// Set via AGENT_MANAGER_API_URL env var, or discovered via vrooli CLI.
	AgentManagerURL string

	// AgentManagerTimeout is the HTTP timeout for agent-manager requests.
	// Default: 60s
	AgentManagerTimeout time.Duration

	// OpenRouterTimeout is the HTTP timeout for OpenRouter requests.
	// Note: Should be >= CompletionTimeout for streaming responses.
	// Default: 120s
	OpenRouterTimeout time.Duration

	// ModelCacheTTL is how long to cache model metadata (context lengths, etc).
	// Higher = fewer CLI calls to resource-openrouter, lower = faster model updates.
	// Default: 5 minutes
	ModelCacheTTL time.Duration

	// Naming configuration for Ollama-powered auto-naming.
	Naming NamingConfig
}

// ResilienceConfig controls failure handling and graceful degradation.
// Audience: Operators tuning reliability vs responsiveness.
type ResilienceConfig struct {
	// RetryAttempts is the number of retry attempts for transient failures.
	// Higher = more resilient to flaky services, lower = faster failure.
	// Default: 3
	RetryAttempts int

	// RetryBaseDelay is the initial delay before first retry (doubles each attempt).
	// Higher = gentler on recovering services, lower = faster recovery.
	// Default: 1s
	RetryBaseDelay time.Duration

	// RetryMaxDelay caps the exponential backoff delay.
	// Default: 10s
	RetryMaxDelay time.Duration

	// CircuitBreakerThreshold is failures before circuit opens.
	// Higher = more tolerant of errors, lower = faster protection.
	// Default: 5
	CircuitBreakerThreshold int

	// CircuitBreakerCooldown is how long circuit stays open before half-open probe.
	// Higher = more recovery time, lower = faster retry of failing services.
	// Default: 30s
	CircuitBreakerCooldown time.Duration

	// EnableNamingFallback determines if auto-naming gracefully degrades.
	// When true, failures return FallbackName instead of error.
	// Default: true
	EnableNamingFallback bool
}

// StorageConfig controls file upload and storage behavior.
// Audience: Operators configuring file storage location and limits.
type StorageConfig struct {
	// BasePath is the directory where uploaded files are stored.
	// Files are organized as: BasePath/YYYY/MM/DD/{uuid}.{ext}
	// Set via UPLOAD_BASE_PATH env var.
	// Default: "./uploads"
	BasePath string

	// BaseURL is the URL prefix for serving uploaded files.
	// Set via UPLOAD_BASE_URL env var.
	// Default: "/api/v1/uploads"
	BaseURL string

	// MaxFileSize is the maximum allowed file size in bytes.
	// Higher = allows larger files, lower = faster uploads/less storage.
	// Set via UPLOAD_MAX_SIZE_MB env var (in megabytes).
	// Default: 20MB (20 * 1024 * 1024)
	MaxFileSize int64

	// AllowedImageTypes is the list of allowed image MIME types.
	// Default: ["image/jpeg", "image/png", "image/gif", "image/webp"]
	AllowedImageTypes []string

	// AllowedDocumentTypes is the list of allowed document MIME types.
	// Default: ["application/pdf"]
	AllowedDocumentTypes []string
}

// PromptSyncConfig controls skill synchronization from prompt-manager.
// Skills are now sourced from prompt-manager's unified prompt system.
// Audience: Operators configuring prompt-manager integration.
type PromptSyncConfig struct {
	// Enabled controls whether to sync skills from prompt-manager.
	// When false, falls back to local file-based skills.
	// Set via PROMPT_SYNC_ENABLED env var.
	// Default: true
	Enabled bool

	// PromptManagerURL is the prompt-manager API endpoint.
	// Set via PROMPT_MANAGER_URL env var.
	// Default: "http://localhost:${PROMPT_MANAGER_PORT}"
	PromptManagerURL string

	// SyncIntervalSeconds is how often to check for prompt updates.
	// Higher = less network traffic, lower = faster updates.
	// Default: 60
	SyncIntervalSeconds int

	// SyncTimeout is the HTTP timeout for sync requests.
	// Default: 30s
	SyncTimeout time.Duration

	// SkillOverridesPath is the path to the local overrides file.
	// Used to customize icons and targetToolIds for specific prompts.
	// Default: "../config/skills.json"
	SkillOverridesPath string
}
