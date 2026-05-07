package aisearch

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env-var names are declared as constants so the resolvers, the test harness,
// and any future operator tooling share a single source of truth.
const (
	EnvOllamaURL              = "OLLAMA_URL"
	EnvQdrantURL              = "QDRANT_URL"
	EnvQdrantBaseURL          = "QDRANT_BASE_URL"
	EnvQdrantPort             = "QDRANT_PORT"
	EnvQdrantAPIKey           = "QDRANT_API_KEY"
	EnvAISearchModel          = "AI_SEARCH_MODEL"
	EnvAISearchThreshold      = "AI_SEARCH_THRESHOLD"
	EnvAISearchBacklogColl    = "AI_SEARCH_BACKLOG_COLLECTION"
	EnvAISearchInitiativeColl = "AI_SEARCH_INITIATIVE_COLLECTION"

	// EnvAISearchSyncInterval overrides the SyncLoop tick interval (e.g.
	// "30m", "24h"). Default DefaultSyncInterval. Invalid values fall back to
	// the default with a slog.Warn — never silently disable the loop.
	EnvAISearchSyncInterval = "AI_SEARCH_SYNC_INTERVAL"

	// EnvAISearchSyncDisabled, when set to "1" or "true", skips the periodic
	// SyncLoop entirely. Boot-time RunOnce still fires. Operator kill-switch.
	EnvAISearchSyncDisabled = "AI_SEARCH_SYNC_DISABLED"

	// EnvAISearchReconcileParallelism caps the in-flight Embed+Upsert calls
	// during Reconciler.Apply. Default DefaultReconcileParallelism, clamped
	// to [1, MaxReconcileParallelism].
	EnvAISearchReconcileParallelism = "AI_SEARCH_RECONCILE_PARALLELISM"

	DefaultEmbeddingModel       = "nomic-embed-text"
	DefaultVectorDimensions     = 768
	DefaultThreshold            = 0.5
	DefaultBacklogCollection    = "swarm-manager-backlog"
	DefaultInitiativeCollection = "swarm-manager-initiatives"

	// DefaultSyncInterval is the SyncLoop tick interval when unset. 5m matches
	// the original Service.StartPeriodicSync interval — the convergent
	// reconciler makes most ticks near-free, so this can stay tight.
	DefaultSyncInterval = 5 * time.Minute

	// DefaultReconcileParallelism is the Reconciler.Apply concurrency when
	// AI_SEARCH_RECONCILE_PARALLELISM is unset. Modest by design: the big
	// performance win comes from "0 embeds when nothing changed," not from
	// parallelism. Operators can dial down to 1 on weak hosts.
	DefaultReconcileParallelism = 4

	// MaxReconcileParallelism is the upper clamp. 16 keeps Ollama from
	// becoming the system-wide bottleneck even when an operator passes a
	// silly value via env.
	MaxReconcileParallelism = 16
)

// ResolveOllamaURL returns the configured Ollama URL, or the empty string if
// none is configured. Callers treat an empty URL as "AI search disabled" and
// degrade gracefully.
func ResolveOllamaURL() string {
	return strings.TrimSpace(os.Getenv(EnvOllamaURL))
}

// ResolveSyncInterval reads EnvAISearchSyncInterval as a Go time.Duration
// (e.g. "30m", "24h"). Empty or invalid values fall back to DefaultSyncInterval
// with a slog.Warn — a typo must never silently disable convergence.
func ResolveSyncInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv(EnvAISearchSyncInterval))
	if raw == "" {
		return DefaultSyncInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		slog.Warn("[aisearch] invalid AI_SEARCH_SYNC_INTERVAL, falling back to default",
			"value", raw, "default", DefaultSyncInterval)
		return DefaultSyncInterval
	}
	return d
}

// ResolveSyncDisabled reads EnvAISearchSyncDisabled. The kill-switch accepts
// "1" or "true" (case-insensitive); any other value (including unset) leaves
// the loop enabled.
func ResolveSyncDisabled() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(EnvAISearchSyncDisabled)))
	return raw == "1" || raw == "true"
}

// ResolveReconcileParallelism reads EnvAISearchReconcileParallelism, clamps it
// to [1, MaxReconcileParallelism], and falls back to DefaultReconcileParallelism
// when unset or invalid.
func ResolveReconcileParallelism() int {
	raw := strings.TrimSpace(os.Getenv(EnvAISearchReconcileParallelism))
	if raw == "" {
		return DefaultReconcileParallelism
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		slog.Warn("[aisearch] invalid AI_SEARCH_RECONCILE_PARALLELISM, falling back to default",
			"value", raw, "default", DefaultReconcileParallelism)
		return DefaultReconcileParallelism
	}
	if n > MaxReconcileParallelism {
		return MaxReconcileParallelism
	}
	return n
}

// ResolveQdrantURL resolves the Qdrant base URL from the standard precedence:
// QDRANT_URL > QDRANT_BASE_URL (Vrooli resource export) > http://localhost:{QDRANT_PORT}.
// Returns "" if none are set.
func ResolveQdrantURL() string {
	if v := strings.TrimSpace(os.Getenv(EnvQdrantURL)); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(EnvQdrantBaseURL)); v != "" {
		return v
	}
	if port := strings.TrimSpace(os.Getenv(EnvQdrantPort)); port != "" {
		return fmt.Sprintf("http://localhost:%s", port)
	}
	return ""
}

// Config is the resolved configuration for constructing a swarm-manager
// aisearch Service. Unset string fields mean "use default"; unset URL fields
// mean "that subsystem is disabled and the service degrades to fallback."
type Config struct {
	OllamaURL            string
	QdrantURL            string
	QdrantAPIKey         string
	EmbeddingModel       string
	VectorDimensions     int
	Threshold            float64
	BacklogCollection    string
	InitiativeCollection string
}

// LoadConfigFromEnv reads the full aisearch configuration from the process
// environment, applying defaults. It never returns an error — missing values
// become defaults, and missing URLs are represented as empty strings that the
// caller must interpret as "disabled."
func LoadConfigFromEnv() Config {
	cfg := Config{
		OllamaURL:            ResolveOllamaURL(),
		QdrantURL:            ResolveQdrantURL(),
		QdrantAPIKey:         strings.TrimSpace(os.Getenv(EnvQdrantAPIKey)),
		EmbeddingModel:       DefaultEmbeddingModel,
		VectorDimensions:     DefaultVectorDimensions,
		Threshold:            DefaultThreshold,
		BacklogCollection:    DefaultBacklogCollection,
		InitiativeCollection: DefaultInitiativeCollection,
	}
	if v := strings.TrimSpace(os.Getenv(EnvAISearchModel)); v != "" {
		cfg.EmbeddingModel = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvAISearchBacklogColl)); v != "" {
		cfg.BacklogCollection = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvAISearchInitiativeColl)); v != "" {
		cfg.InitiativeCollection = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvAISearchThreshold)); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 {
			cfg.Threshold = parsed
		}
	}
	return cfg
}
