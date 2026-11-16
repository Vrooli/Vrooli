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
	})
	if cfg.Timeouts.DefaultRequest != 7300*time.Millisecond {
		t.Fatalf("injected timeout = %s, want 7.3s", cfg.Timeouts.DefaultRequest)
	}
	if cfg.ArtifactLimits.DefaultProfile != ProfileMinimal {
		t.Fatalf("injected artifact profile = %q, want %q", cfg.ArtifactLimits.DefaultProfile, ProfileMinimal)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("injected config should validate: %v", err)
	}
}
