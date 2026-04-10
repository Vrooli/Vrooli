// Package config provides centralized configuration for the Agent Inbox scenario.
// This file implements configuration validation logic.
package config

import (
	"fmt"
	"time"
)

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

	// Storage validation
	if c.Storage.BasePath == "" {
		return fmt.Errorf("storage.base_path must not be empty")
	}
	if c.Storage.MaxFileSize < 1024*1024 {
		return fmt.Errorf("storage.max_file_size must be >= 1MB, got %d", c.Storage.MaxFileSize)
	}
	if c.Storage.MaxFileSize > 100*1024*1024 {
		return fmt.Errorf("storage.max_file_size must be <= 100MB, got %d", c.Storage.MaxFileSize)
	}

	return nil
}
