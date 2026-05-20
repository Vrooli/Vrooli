package aisearch

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env vars tune the reconciler / sync loop. Mirror of prompt-manager's
// aisearch env.go with a cli-health-specific prefix so multiple aisearch
// users on the same host can be tuned independently.
const (
	EnvSyncInterval         = "CLI_HEALTH_SYNC_INTERVAL"
	EnvSyncDisabled         = "CLI_HEALTH_SYNC_DISABLED"
	EnvReconcileParallelism = "CLI_HEALTH_RECONCILE_PARALLELISM"
	EnvQdrantURL            = "CLI_HEALTH_QDRANT_URL"
	EnvQdrantAPIKey         = "CLI_HEALTH_QDRANT_API_KEY"
	EnvOllamaModel          = "CLI_HEALTH_EMBED_MODEL"

	DefaultSyncInterval         = 5 * time.Minute
	DefaultReconcileParallelism = 4
	MaxReconcileParallelism     = 16
	DefaultCollection           = "cli-health-commands"
	DefaultVectorSize           = 768
	DefaultEmbedModel           = "nomic-embed-text"
	DefaultQdrantURL            = "http://127.0.0.1:6333"
)

// Config holds reconciler/sync-loop tunables read from environment.
type Config struct {
	SyncInterval         time.Duration
	SyncDisabled         bool
	ReconcileParallelism int
	QdrantURL            string
	QdrantAPIKey         string
	EmbedModel           string
}

// LoadConfigFromEnv reads tunables from the environment, falling back to
// defaults (with warning logs) when values are absent or malformed.
func LoadConfigFromEnv() Config {
	return Config{
		SyncInterval:         envDuration(EnvSyncInterval, DefaultSyncInterval),
		SyncDisabled:         envBool(EnvSyncDisabled),
		ReconcileParallelism: envInt(EnvReconcileParallelism, DefaultReconcileParallelism, 1, MaxReconcileParallelism),
		QdrantURL:            envString(EnvQdrantURL, DefaultQdrantURL),
		QdrantAPIKey:         envString(EnvQdrantAPIKey, ""),
		EmbedModel:           envString(EnvOllamaModel, DefaultEmbedModel),
	}
}

func envDuration(name string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("[cli-health/aisearch] invalid env %s=%q, using default %s", name, raw, def)
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
		log.Printf("[cli-health/aisearch] invalid env %s=%q, using default %d", name, raw, def)
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

func envString(name, def string) string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	return raw
}
