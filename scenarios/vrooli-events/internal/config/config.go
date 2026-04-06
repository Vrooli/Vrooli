// DOC: docs/reference/configuration.md
// Package config centralizes all tunable levers for vrooli-events.
// Every value has a sane default and can be overridden via environment variables.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds all tunable parameters for vrooli-events.
type Config struct {
	// --- Store ---
	DBPath       string        // Path to SQLite database file. Env: DB_PATH
	MaxAge       time.Duration // Event retention period. Env: RETENTION_MAX_AGE (e.g. "720h" for 30 days)
	MaxSizeBytes int64         // Max total payload bytes before size-based pruning. Env: RETENTION_MAX_SIZE_BYTES

	// --- Pruner ---
	PruneInterval time.Duration // How often the background pruner runs. Env: PRUNE_INTERVAL (e.g. "6h")

	// --- Broker / SSE ---
	SubscriberBufSize int           // Per-subscriber channel buffer size. Env: SSE_SUBSCRIBER_BUF_SIZE
	HeartbeatInterval time.Duration // SSE heartbeat frequency. Env: SSE_HEARTBEAT_INTERVAL
	SSERetryMs        int           // SSE retry directive sent to clients (ms). Env: SSE_RETRY_MS
	ReplayLimit       int           // Max events replayed on SSE reconnect. Env: SSE_REPLAY_LIMIT

	// --- API ---
	MaxBodyBytes  int64 // Max request body size for event ingestion. Env: API_MAX_BODY_BYTES
	QueryLimit    int   // Default query limit when not specified. Env: API_QUERY_LIMIT_DEFAULT
	QueryLimitMax int   // Maximum allowed query limit. Env: API_QUERY_LIMIT_MAX
}

// Defaults
const (
	DefaultMaxAge            = 30 * 24 * time.Hour    // 30 days
	DefaultMaxSizeBytes      = 2 * 1024 * 1024 * 1024 // 2 GB
	DefaultPruneInterval     = 6 * time.Hour
	DefaultSubscriberBufSize = 64
	DefaultHeartbeatInterval = 30 * time.Second
	DefaultSSERetryMs        = 5000
	DefaultReplayLimit       = 1000
	DefaultMaxBodyBytes      = 1 << 20 // 1 MB
	DefaultQueryLimit        = 100
	DefaultQueryLimitMax     = 1000
)

// Load reads configuration from environment variables, applying defaults for any unset values.
func Load() Config {
	cfg := Config{
		DBPath:            envStr("DB_PATH", defaultDBPath()),
		MaxAge:            envDuration("RETENTION_MAX_AGE", DefaultMaxAge),
		MaxSizeBytes:      envInt64("RETENTION_MAX_SIZE_BYTES", DefaultMaxSizeBytes),
		PruneInterval:     envDuration("PRUNE_INTERVAL", DefaultPruneInterval),
		SubscriberBufSize: envInt("SSE_SUBSCRIBER_BUF_SIZE", DefaultSubscriberBufSize),
		HeartbeatInterval: envDuration("SSE_HEARTBEAT_INTERVAL", DefaultHeartbeatInterval),
		SSERetryMs:        envInt("SSE_RETRY_MS", DefaultSSERetryMs),
		ReplayLimit:       envInt("SSE_REPLAY_LIMIT", DefaultReplayLimit),
		MaxBodyBytes:      envInt64("API_MAX_BODY_BYTES", DefaultMaxBodyBytes),
		QueryLimit:        envInt("API_QUERY_LIMIT_DEFAULT", DefaultQueryLimit),
		QueryLimitMax:     envInt("API_QUERY_LIMIT_MAX", DefaultQueryLimitMax),
	}
	return cfg
}

func defaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".vrooli", "vrooli-events", "events.db")
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
