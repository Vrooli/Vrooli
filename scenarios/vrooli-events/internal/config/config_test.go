package config

import (
	"path/filepath"
	"testing"
	"time"
)

// [REQ:REQ-ES-004] Verify config loads defaults and respects env overrides.

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()

	if cfg.MaxAge != DefaultMaxAge {
		t.Errorf("MaxAge = %v, want %v", cfg.MaxAge, DefaultMaxAge)
	}
	if cfg.MaxSizeBytes != DefaultMaxSizeBytes {
		t.Errorf("MaxSizeBytes = %d, want %d", cfg.MaxSizeBytes, DefaultMaxSizeBytes)
	}
	if cfg.PruneInterval != DefaultPruneInterval {
		t.Errorf("PruneInterval = %v, want %v", cfg.PruneInterval, DefaultPruneInterval)
	}
	if cfg.SubscriberBufSize != DefaultSubscriberBufSize {
		t.Errorf("SubscriberBufSize = %d, want %d", cfg.SubscriberBufSize, DefaultSubscriberBufSize)
	}
	if cfg.HeartbeatInterval != DefaultHeartbeatInterval {
		t.Errorf("HeartbeatInterval = %v, want %v", cfg.HeartbeatInterval, DefaultHeartbeatInterval)
	}
	if cfg.SSERetryMs != DefaultSSERetryMs {
		t.Errorf("SSERetryMs = %d, want %d", cfg.SSERetryMs, DefaultSSERetryMs)
	}
	if cfg.ReplayLimit != DefaultReplayLimit {
		t.Errorf("ReplayLimit = %d, want %d", cfg.ReplayLimit, DefaultReplayLimit)
	}
	if cfg.MaxBodyBytes != DefaultMaxBodyBytes {
		t.Errorf("MaxBodyBytes = %d, want %d", cfg.MaxBodyBytes, DefaultMaxBodyBytes)
	}
	if cfg.QueryLimit != DefaultQueryLimit {
		t.Errorf("QueryLimit = %d, want %d", cfg.QueryLimit, DefaultQueryLimit)
	}
	if cfg.QueryLimitMax != DefaultQueryLimitMax {
		t.Errorf("QueryLimitMax = %d, want %d", cfg.QueryLimitMax, DefaultQueryLimitMax)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("RETENTION_MAX_AGE", "48h")
	t.Setenv("RETENTION_MAX_SIZE_BYTES", "1073741824")
	t.Setenv("PRUNE_INTERVAL", "1h")
	t.Setenv("SSE_SUBSCRIBER_BUF_SIZE", "128")
	t.Setenv("SSE_HEARTBEAT_INTERVAL", "15s")
	t.Setenv("SSE_RETRY_MS", "3000")
	t.Setenv("SSE_REPLAY_LIMIT", "500")
	t.Setenv("API_MAX_BODY_BYTES", "2097152")
	t.Setenv("API_QUERY_LIMIT_DEFAULT", "50")
	t.Setenv("API_QUERY_LIMIT_MAX", "2000")

	cfg := Load()

	if cfg.MaxAge != 48*time.Hour {
		t.Errorf("MaxAge = %v, want 48h", cfg.MaxAge)
	}
	if cfg.MaxSizeBytes != 1073741824 {
		t.Errorf("MaxSizeBytes = %d, want 1073741824", cfg.MaxSizeBytes)
	}
	if cfg.PruneInterval != time.Hour {
		t.Errorf("PruneInterval = %v, want 1h", cfg.PruneInterval)
	}
	if cfg.SubscriberBufSize != 128 {
		t.Errorf("SubscriberBufSize = %d, want 128", cfg.SubscriberBufSize)
	}
	if cfg.HeartbeatInterval != 15*time.Second {
		t.Errorf("HeartbeatInterval = %v, want 15s", cfg.HeartbeatInterval)
	}
	if cfg.SSERetryMs != 3000 {
		t.Errorf("SSERetryMs = %d, want 3000", cfg.SSERetryMs)
	}
	if cfg.ReplayLimit != 500 {
		t.Errorf("ReplayLimit = %d, want 500", cfg.ReplayLimit)
	}
	if cfg.MaxBodyBytes != 2097152 {
		t.Errorf("MaxBodyBytes = %d, want 2097152", cfg.MaxBodyBytes)
	}
	if cfg.QueryLimit != 50 {
		t.Errorf("QueryLimit = %d, want 50", cfg.QueryLimit)
	}
	if cfg.QueryLimitMax != 2000 {
		t.Errorf("QueryLimitMax = %d, want 2000", cfg.QueryLimitMax)
	}
}

func TestLoad_InvalidEnvFallsBackToDefault(t *testing.T) {
	t.Setenv("RETENTION_MAX_AGE", "not-a-duration")
	t.Setenv("SSE_SUBSCRIBER_BUF_SIZE", "abc")
	t.Setenv("RETENTION_MAX_SIZE_BYTES", "xyz")

	cfg := Load()

	if cfg.MaxAge != DefaultMaxAge {
		t.Errorf("MaxAge = %v, want default %v for invalid env", cfg.MaxAge, DefaultMaxAge)
	}
	if cfg.SubscriberBufSize != DefaultSubscriberBufSize {
		t.Errorf("SubscriberBufSize = %d, want default %d for invalid env", cfg.SubscriberBufSize, DefaultSubscriberBufSize)
	}
	if cfg.MaxSizeBytes != DefaultMaxSizeBytes {
		t.Errorf("MaxSizeBytes = %d, want default %d for invalid env", cfg.MaxSizeBytes, DefaultMaxSizeBytes)
	}
}

func TestDefaultDBPathUsesCanonicalStorage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	got := defaultDBPath()
	want := filepath.Join(home, ".local", "share", "vrooli", "vrooli-events", "events.db")
	if got != want {
		t.Fatalf("defaultDBPath() = %q, want %q", got, want)
	}
}
