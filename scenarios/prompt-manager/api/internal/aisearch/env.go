package aisearch

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env vars that tune the reconciler / sync loop. See plan §8.1.
const (
	EnvAISearchSyncInterval         = "AI_SEARCH_SYNC_INTERVAL"
	EnvAISearchSyncDisabled         = "AI_SEARCH_SYNC_DISABLED"
	EnvAISearchReconcileParallelism = "AI_SEARCH_RECONCILE_PARALLELISM"

	DefaultSyncInterval         = 5 * time.Minute
	DefaultReconcileParallelism = 4
	MaxReconcileParallelism     = 16
)

// Config holds the reconciler/sync-loop tunables read from environment.
type Config struct {
	SyncInterval         time.Duration
	SyncDisabled         bool
	ReconcileParallelism int
}

// LoadConfigFromEnv reads tunables from the environment, falling back to
// defaults (with warning logs) when values are absent or malformed.
func LoadConfigFromEnv() Config {
	return Config{
		SyncInterval:         envDuration(EnvAISearchSyncInterval, DefaultSyncInterval),
		SyncDisabled:         envBool(EnvAISearchSyncDisabled),
		ReconcileParallelism: envInt(EnvAISearchReconcileParallelism, DefaultReconcileParallelism, 1, MaxReconcileParallelism),
	}
}

func envDuration(name string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("[aisearch] invalid env %s=%q, using default %s", name, raw, def)
		return def
	}
	return d
}

func envInt(name string, def, min, max int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("[aisearch] invalid env %s=%q, using default %d", name, raw, def)
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func envBool(name string) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
