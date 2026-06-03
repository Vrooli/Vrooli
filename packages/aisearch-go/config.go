package aisearch

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultSyncInterval is the reconcile cadence used when no interval is set.
const DefaultSyncInterval = 5 * time.Minute

// Config holds the reconciler / sync-loop tunables a consumer reads from the
// environment. Each consumer scopes its variables under a prefix (e.g.
// "CLI_HEALTH" → CLI_HEALTH_SYNC_INTERVAL) so multiple aisearch consumers on one
// host are tuned independently.
type Config struct {
	SyncInterval         time.Duration
	SyncDisabled         bool
	ReconcileParallelism int
	// MaxEmbedsPerTick caps embeds per reconcile tick (0 = unlimited). The large
	// documentation corpus uses it so a first full index never starves Ollama
	// (§4.2); the 1:1 consumers leave it at 0.
	MaxEmbedsPerTick int
	QdrantURL        string
	QdrantAPIKey     string
	EmbedModel       string
}

// LoadConfig reads tunables under "<prefix>_<NAME>", falling back to the engine
// defaults (with a warning log) when a value is absent or malformed. An empty
// prefix reads the bare variable names.
func LoadConfig(prefix string) Config {
	p := strings.TrimRight(strings.TrimSpace(prefix), "_")
	key := func(name string) string {
		if p == "" {
			return name
		}
		return p + "_" + name
	}
	return Config{
		SyncInterval:         envDuration(key("SYNC_INTERVAL"), DefaultSyncInterval),
		SyncDisabled:         envBool(key("SYNC_DISABLED")),
		ReconcileParallelism: envInt(key("RECONCILE_PARALLELISM"), DefaultReconcileParallelism, 1, MaxReconcileParallelism),
		MaxEmbedsPerTick:     envInt(key("MAX_EMBEDS_PER_TICK"), 0, 0, 1<<30),
		QdrantURL:            envString(key("QDRANT_URL"), DefaultQdrantURL),
		QdrantAPIKey:         envString(key("QDRANT_API_KEY"), ""),
		EmbedModel:           envString(key("EMBED_MODEL"), DefaultEmbedModel),
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
	switch strings.TrimSpace(strings.ToLower(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func envString(name, def string) string {
	if raw := strings.TrimSpace(os.Getenv(name)); raw != "" {
		return raw
	}
	return def
}
