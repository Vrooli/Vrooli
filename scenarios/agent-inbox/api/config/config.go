// Package config provides centralized configuration for the Agent Inbox scenario.
// This is the control surface for tunable levers - parameters that operators and
// agents can adjust to steer behavior without touching implementation code.
//
// Design principles:
//   - Sane defaults that work for common usage
//   - Clear, intention-revealing names describing tradeoffs
//   - Bounded values with validation
//   - Monotonic behavior (e.g., "higher = more thorough but slower")
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all tunable parameters for the Agent Inbox scenario.
// Each field is documented with:
//   - Default value
//   - What happens when increased/decreased
//   - Who typically adjusts it (operator, user, agent)
type Config struct {
	Server      ServerConfig
	AI          AIConfig
	Integration IntegrationConfig
	Resilience  ResilienceConfig
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
}

// NamingConfig controls the auto-naming feature powered by local Ollama.
// Audience: Operators tuning naming quality vs speed.
type NamingConfig struct {
	// Model is the Ollama model for generating chat names.
	// Set via OLLAMA_NAMING_MODEL env var.
	// Default: "llama3.1:8b"
	Model string

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

// IntegrationConfig controls external service connections.
// Audience: Operators configuring service mesh.
type IntegrationConfig struct {
	// OllamaBaseURL is the Ollama API endpoint.
	// Set via OLLAMA_BASE_URL env var, or derived from OLLAMA_PORT.
	// Default: "http://localhost:11434"
	OllamaBaseURL string

	// OllamaTimeout is the HTTP timeout for Ollama requests.
	// Default: 30s
	OllamaTimeout time.Duration

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

// Default returns the default configuration with all sane defaults.
// This configuration works well for local development and typical deployments.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnvOrDefault("API_PORT", "8080"),
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     120 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		AI: AIConfig{
			DefaultModel:      getEnvOrDefault("DEFAULT_AI_MODEL", "anthropic/claude-3.5-sonnet"),
			CompletionTimeout: 120 * time.Second,
			StreamBufferSize:  4096,
		},
		Integration: IntegrationConfig{
			OllamaBaseURL:       getOllamaBaseURL(),
			OllamaTimeout:       30 * time.Second,
			AgentManagerTimeout: 60 * time.Second,
			OpenRouterTimeout:   120 * time.Second,
			Naming: NamingConfig{
				Model:               getEnvOrDefault("OLLAMA_NAMING_MODEL", "llama3.1:8b"),
				Temperature:         0.3,
				MaxTokens:           20,
				SummaryMessageLimit: 10,
				SummaryContentLimit: 200,
				Timeout:             30 * time.Second,
				FallbackName:        "New Conversation",
			},
		},
		Resilience: ResilienceConfig{
			RetryAttempts:           3,
			RetryBaseDelay:          1 * time.Second,
			RetryMaxDelay:           10 * time.Second,
			CircuitBreakerThreshold: 5,
			CircuitBreakerCooldown:  30 * time.Second,
			EnableNamingFallback:    true,
		},
	}
}

// Validate checks that all configuration values are within acceptable bounds.
// Returns an error describing the first invalid configuration found.
func (c *Config) Validate() error {
	// Server validation
	if c.Server.ReadTimeout < time.Second {
		return fmt.Errorf("server.read_timeout must be >= 1s, got %v", c.Server.ReadTimeout)
	}
	if c.Server.WriteTimeout < time.Second {
		return fmt.Errorf("server.write_timeout must be >= 1s, got %v", c.Server.WriteTimeout)
	}
	if c.Server.ShutdownTimeout < time.Second {
		return fmt.Errorf("server.shutdown_timeout must be >= 1s, got %v", c.Server.ShutdownTimeout)
	}

	// AI validation
	if c.AI.CompletionTimeout < 10*time.Second {
		return fmt.Errorf("ai.completion_timeout must be >= 10s, got %v", c.AI.CompletionTimeout)
	}
	if c.AI.StreamBufferSize < 1024 {
		return fmt.Errorf("ai.stream_buffer_size must be >= 1024, got %d", c.AI.StreamBufferSize)
	}

	// Naming validation
	if c.Integration.Naming.Temperature < 0 || c.Integration.Naming.Temperature > 1 {
		return fmt.Errorf("integration.naming.temperature must be in [0,1], got %v", c.Integration.Naming.Temperature)
	}
	if c.Integration.Naming.MaxTokens < 5 || c.Integration.Naming.MaxTokens > 100 {
		return fmt.Errorf("integration.naming.max_tokens must be in [5,100], got %d", c.Integration.Naming.MaxTokens)
	}

	// Resilience validation
	if c.Resilience.RetryAttempts < 0 || c.Resilience.RetryAttempts > 10 {
		return fmt.Errorf("resilience.retry_attempts must be in [0,10], got %d", c.Resilience.RetryAttempts)
	}
	if c.Resilience.CircuitBreakerThreshold < 1 {
		return fmt.Errorf("resilience.circuit_breaker_threshold must be >= 1, got %d", c.Resilience.CircuitBreakerThreshold)
	}

	return nil
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getOllamaBaseURL() string {
	if url := os.Getenv("OLLAMA_BASE_URL"); url != "" {
		return url
	}
	port := getEnvOrDefault("OLLAMA_PORT", "11434")
	return fmt.Sprintf("http://localhost:%s", port)
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}
