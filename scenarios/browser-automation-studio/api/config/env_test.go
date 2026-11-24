package config

import (
	"testing"
	"time"
)

type mapEnv map[string]string

func (m mapEnv) LookupEnv(key string) (string, bool) {
	value, ok := m[key]
	return value, ok
}

func TestLoadWithEnvReadsOperatorPayloadWithoutProcessMutation(t *testing.T) {
	cfg := LoadWithEnv(mapEnv{
		"BAS_TIMEOUT_DEFAULT_REQUEST_MS": "7300",
		"BAS_ARTIFACT_DEFAULT_PROFILE":   "minimal",
		"BAS_EXECUTION_MAX_CONCURRENT":   "7",
	})
	if cfg.Timeouts.DefaultRequest != 7300*time.Millisecond {
		t.Fatalf("injected timeout = %s, want 7.3s", cfg.Timeouts.DefaultRequest)
	}
	if cfg.ArtifactLimits.DefaultProfile != ProfileMinimal {
		t.Fatalf("injected artifact profile = %q, want %q", cfg.ArtifactLimits.DefaultProfile, ProfileMinimal)
	}
	if cfg.Execution.MaxConcurrentExecutions != 7 {
		t.Fatalf("injected concurrency cap = %d, want 7", cfg.Execution.MaxConcurrentExecutions)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("injected config should validate: %v", err)
	}
}

func TestExecutionConcurrencyCapMustBePositive(t *testing.T) {
	cfg := LoadWithEnv(mapEnv{"BAS_EXECUTION_MAX_CONCURRENT": "0"})
	if err := cfg.Validate(); err == nil {
		t.Fatal("zero execution concurrency cap unexpectedly validated")
	}
}

func TestExecutionConcurrencyCapUsesConfigOwnedDefault(t *testing.T) {
	cfg := LoadWithEnv(mapEnv{})
	if cfg.Execution.MaxConcurrentExecutions != DefaultMaxConcurrentExecutions {
		t.Fatalf("default execution concurrency cap = %d, want config default %d", cfg.Execution.MaxConcurrentExecutions, DefaultMaxConcurrentExecutions)
	}
}
