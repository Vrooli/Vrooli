package config_test

import (
	"testing"
	"time"

	"agent-inbox/config"
)

func TestDefault_ReturnsValidConfig(t *testing.T) {
	cfg := config.Default()

	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid, got error: %v", err)
	}
}

func TestDefault_ServerDefaults(t *testing.T) {
	cfg := config.Default()

	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("expected ReadTimeout 30s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 30*time.Second {
		t.Errorf("expected WriteTimeout 30s, got %v", cfg.Server.WriteTimeout)
	}
	if cfg.Server.IdleTimeout != 120*time.Second {
		t.Errorf("expected IdleTimeout 120s, got %v", cfg.Server.IdleTimeout)
	}
	if cfg.Server.ShutdownTimeout != 10*time.Second {
		t.Errorf("expected ShutdownTimeout 10s, got %v", cfg.Server.ShutdownTimeout)
	}
}

func TestDefault_AIDefaults(t *testing.T) {
	cfg := config.Default()

	if cfg.AI.DefaultModel != "anthropic/claude-3.5-sonnet" {
		t.Errorf("expected DefaultModel 'anthropic/claude-3.5-sonnet', got %v", cfg.AI.DefaultModel)
	}
	if cfg.AI.CompletionTimeout != 120*time.Second {
		t.Errorf("expected CompletionTimeout 120s, got %v", cfg.AI.CompletionTimeout)
	}
	if cfg.AI.StreamBufferSize != 4096 {
		t.Errorf("expected StreamBufferSize 4096, got %d", cfg.AI.StreamBufferSize)
	}
}

func TestDefault_NamingDefaults(t *testing.T) {
	cfg := config.Default()

	if cfg.Integration.Naming.Model != "llama3.1:8b" {
		t.Errorf("expected Naming.Model 'llama3.1:8b', got %v", cfg.Integration.Naming.Model)
	}
	if cfg.Integration.Naming.Temperature != 0.3 {
		t.Errorf("expected Naming.Temperature 0.3, got %v", cfg.Integration.Naming.Temperature)
	}
	if cfg.Integration.Naming.MaxTokens != 20 {
		t.Errorf("expected Naming.MaxTokens 20, got %d", cfg.Integration.Naming.MaxTokens)
	}
	if cfg.Integration.Naming.SummaryMessageLimit != 10 {
		t.Errorf("expected Naming.SummaryMessageLimit 10, got %d", cfg.Integration.Naming.SummaryMessageLimit)
	}
	if cfg.Integration.Naming.FallbackName != "New Conversation" {
		t.Errorf("expected Naming.FallbackName 'New Conversation', got %v", cfg.Integration.Naming.FallbackName)
	}
}

func TestDefault_ResilienceDefaults(t *testing.T) {
	cfg := config.Default()

	if cfg.Resilience.RetryAttempts != 3 {
		t.Errorf("expected RetryAttempts 3, got %d", cfg.Resilience.RetryAttempts)
	}
	if cfg.Resilience.CircuitBreakerThreshold != 5 {
		t.Errorf("expected CircuitBreakerThreshold 5, got %d", cfg.Resilience.CircuitBreakerThreshold)
	}
	if !cfg.Resilience.EnableNamingFallback {
		t.Error("expected EnableNamingFallback true")
	}
}

func TestValidate_RejectsInvalidReadTimeout(t *testing.T) {
	cfg := config.Default()
	cfg.Server.ReadTimeout = 500 * time.Millisecond // Too short

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for short ReadTimeout")
	}
}

func TestValidate_RejectsInvalidCompletionTimeout(t *testing.T) {
	cfg := config.Default()
	cfg.AI.CompletionTimeout = 5 * time.Second // Too short

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for short CompletionTimeout")
	}
}

func TestValidate_RejectsInvalidTemperature(t *testing.T) {
	cfg := config.Default()
	cfg.Integration.Naming.Temperature = 1.5 // Out of range

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for temperature > 1")
	}

	cfg.Integration.Naming.Temperature = -0.1 // Out of range
	err = cfg.Validate()
	if err == nil {
		t.Error("expected validation error for temperature < 0")
	}
}

func TestValidate_RejectsInvalidMaxTokens(t *testing.T) {
	cfg := config.Default()
	cfg.Integration.Naming.MaxTokens = 2 // Too few

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for MaxTokens < 5")
	}

	cfg.Integration.Naming.MaxTokens = 150 // Too many
	err = cfg.Validate()
	if err == nil {
		t.Error("expected validation error for MaxTokens > 100")
	}
}

func TestValidate_RejectsInvalidRetryAttempts(t *testing.T) {
	cfg := config.Default()
	cfg.Resilience.RetryAttempts = -1 // Invalid

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for RetryAttempts < 0")
	}

	cfg.Resilience.RetryAttempts = 15 // Too many
	err = cfg.Validate()
	if err == nil {
		t.Error("expected validation error for RetryAttempts > 10")
	}
}

func TestValidate_RejectsInvalidCircuitBreakerThreshold(t *testing.T) {
	cfg := config.Default()
	cfg.Resilience.CircuitBreakerThreshold = 0 // Invalid

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for CircuitBreakerThreshold < 1")
	}
}
