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

// Config holds the OPERATIONAL / WIRING knobs a consumer reads from the
// environment. Each consumer scopes its variables under a prefix (e.g.
// "CLI_HEALTH" → CLI_HEALTH_SYNC_INTERVAL) so multiple aisearch consumers on one
// host are tuned independently.
//
// Control-surface map (see tuning.go for the factor taxonomy SSOT). This struct
// is the OPERATIONAL / WIRING layer ONLY — it no longer carries the search
// TUNING factors (engine, embed_task_prefix, rerank_enabled/blend/shortlist,
// floor band). Those are owned by TuningConfig / `.vrooli/search.json` and read
// via NewServiceForTuning; LoadConfig does not read them from the environment.
// The fields here fall in two groups:
//
//   - WIRING/operational (the source of truth, always): sync cadence,
//     parallelism, embed batch cap, Qdrant address, the deployed embed model, the
//     reranker resource endpoints + fallback-leg model. Deployment facts, not
//     search "factors". (EmbedTaskPrefix is the one tuning factor still PRESENT on
//     the struct — only as the NewEmbedderForConfig input contract — but it is set
//     from the SSOT by the adopter, never env-read here.)
//   - PACKAGE-OWNED CALIBRATION (the floor regime bands, RRF k) lives in
//     defaults.go and floor.go, not here — an adopter does not tune it.
type Config struct {
	// --- WIRING / operational ---
	SyncInterval         time.Duration
	SyncDisabled         bool
	ReconcileParallelism int
	// MaxEmbedsPerTick caps embeds per reconcile tick (0 = unlimited). The large
	// documentation corpus uses it so a first full index never starves Ollama
	// (§4.2); the 1:1 consumers leave it at 0.
	MaxEmbedsPerTick int
	QdrantURL        string
	QdrantAPIKey     string

	// --- EMBEDDER RECIPE (the Config fields NewEmbedderForConfig consumes) ---
	// EmbedModel is the dense embedding model (read from <prefix>_EMBED_MODEL —
	// the deployed/installed model is operational wiring). EmbedTaskPrefix opts
	// into asymmetric "search_query:"/"search_document:" prefixes for nomic; it is
	// a TUNING factor owned by `.vrooli/search.json` (NOT env-read here — a
	// migrated adopter passes tuning.EmbedTaskPrefix into a Config literal), kept
	// on the struct only as the NewEmbedderForConfig input.
	EmbedModel      string
	EmbedTaskPrefix bool
	// RerankModel selects the LLM-fallback leg's model (operational: which model
	// serves the degradation chain). Read from <prefix>_RERANK_MODEL.
	RerankModel string
	// RerankerURL / RerankerModel target the cross-encoder `reranker` resource.
	// Read from <prefix>_RERANKER_URL / _RERANKER_MODEL, they let two scenarios on
	// one host point at *different* rerankers. Both default to "" ("unset"): the
	// cross-encoder then falls back to the resource's own unprefixed env
	// (RERANKER_BASE_URL/RERANKER_URL/RERANKER_HOST+PORT, model RERANKER_MODEL),
	// preserving zero-config local use. Distinct from RerankModel, which is the
	// LLM *fallback* leg's model.
	// --- WIRING: reranker resource endpoints (operational, not a factor) ---
	RerankerURL   string
	RerankerModel string
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
		RerankModel:          envString(key("RERANK_MODEL"), DefaultRerankModel),
		RerankerURL:          envString(key("RERANKER_URL"), ""),
		RerankerModel:        envString(key("RERANKER_MODEL"), ""),
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
