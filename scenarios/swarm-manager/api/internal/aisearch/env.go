package aisearch

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
)

// Env-var names are declared as constants so the resolvers, the test harness,
// and any future operator tooling share a single source of truth.
const (
	EnvOllamaURL              = "OLLAMA_URL"
	EnvQdrantURL              = "QDRANT_URL"
	EnvQdrantBaseURL          = "QDRANT_BASE_URL"
	EnvQdrantPort             = "QDRANT_PORT"
	EnvQdrantAPIKey           = "QDRANT_API_KEY" // #nosec G101 -- this is the NAME of an environment variable, not a credential value.
	EnvAISearchThreshold      = "AI_SEARCH_THRESHOLD"
	EnvAISearchBacklogColl    = "AI_SEARCH_BACKLOG_COLLECTION"
	EnvAISearchInitiativeColl = "AI_SEARCH_INITIATIVE_COLLECTION"
	EnvAISearchRecordColl     = "AI_SEARCH_RECORD_COLLECTION"

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

	DefaultThreshold = 0.5

	// Qdrant collection DOMAINS (Baseline Modes P5). The actual collection
	// names are composed variant-awarely by storage.Collection(domain), which
	// reads the lifecycle-injected VROOLI_STORAGE_NAMESPACE: the live instance
	// uses "swarm-manager_backlog" while a shadow uses
	// "swarm-manager_shadow_backlog", so the two instances never share a
	// collection. Hardcoding "swarm-manager-backlog" here would bypass shadow
	// isolation — exactly what storage-steer / test-genie flag as a finding.
	// (Adopting these is a rename from the old hyphenated names; the convergent
	// reconciler repopulates the new collections from the backlog/initiative/
	// record stores on its next sync.)
	CollectionDomainBacklog    = "backlog"
	CollectionDomainInitiative = "initiatives"
	CollectionDomainRecord     = "records"

	// scenarioSlug is this scenario's own identity, used ONLY to compose a
	// variant-safe fallback collection name when no identity environment is
	// injected at all (a non-lifecycle/dev/test context, which is inherently
	// live). It is never used to bypass an injected shadow namespace.
	scenarioSlug = "swarm-manager"

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
	Threshold            float64
	BacklogCollection    string
	InitiativeCollection string
	RecordCollection     string
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
		Threshold:            DefaultThreshold,
		BacklogCollection:    resolveCollection(EnvAISearchBacklogColl, CollectionDomainBacklog),
		InitiativeCollection: resolveCollection(EnvAISearchInitiativeColl, CollectionDomainInitiative),
		RecordCollection:     resolveCollection(EnvAISearchRecordColl, CollectionDomainRecord),
	}
	if v := strings.TrimSpace(os.Getenv(EnvAISearchThreshold)); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 {
			cfg.Threshold = parsed
		}
	}
	return cfg
}

// resolveCollection composes the variant-aware Qdrant collection name for a
// domain (Baseline Modes P5). Precedence:
//
//  1. An explicit operator override (envKey, e.g. AI_SEARCH_BACKLOG_COLLECTION)
//     wins verbatim — operators retain full control.
//  2. Otherwise the name comes from storage.Collection(domain), which reads the
//     lifecycle-injected VROOLI_STORAGE_NAMESPACE so a shadow instance gets its
//     own collection and never writes into live's.
//
// storage.Collection only errors outside the variant-aware lifecycle (no
// identity environment at all), which is necessarily a live/dev/test context.
// In that case we compose the fallback from VROOLI_VARIANT + this scenario's own
// slug using the SAME "<root>_<domain>" shape the SSOT uses, so even a manually
// mis-set non-live variant stays isolated from live — we never alias a shadow's
// collection onto live's. We never fall back to the pre-adoption hyphenated name.
func resolveCollection(envKey, domain string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	name, err := storage.Collection(domain)
	if err != nil {
		fallback := fallbackNamespaceRoot() + "_" + domain
		slog.Warn("[aisearch] variant-aware collection unavailable; composing from local identity",
			"domain", domain, "fallback", fallback, "error", err)
		return fallback
	}
	return name
}

// fallbackNamespaceRoot mirrors scenarioruntime.InstanceKey.Namespace()'s
// StorageNamespace composition ("<scenario>" for live, "<scenario>_<variant>"
// otherwise) using this scenario's known slug. It is only reached when the
// lifecycle injected no namespace environment; it preserves variant isolation so
// the fallback can never alias a shadow onto live.
func fallbackNamespaceRoot() string {
	variant := strings.ToLower(strings.TrimSpace(os.Getenv(storage.EnvVariant)))
	if variant == "" || variant == "live" {
		return scenarioSlug
	}
	return scenarioSlug + "_" + variant
}
