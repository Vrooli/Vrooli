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
// Control-surface map (see tuning.go for the factor taxonomy SSOT). The fields
// here fall in three groups:
//
//   - WIRING/operational (the source of truth, always): sync cadence,
//     parallelism, embed batch cap, Qdrant address, reranker resource endpoints.
//     These are deployment facts, not search "factors".
//   - SEARCH FACTORS (EmbedModel, EmbedTaskPrefix, Relevance*, Rerank{Enabled,
//     Blend,Model,Shortlist}): these are GENUINE per-corpus tuning factors and
//     are now owned by TuningConfig / `.vrooli/search.json` (the SSOT). They are
//     retained on Config only for env-driven consumers that have not yet migrated
//     to search.json; a migrated adopter (cli-health, KO) reads them from the
//     SSOT via NewServiceForTuning and ignores the env reads below.
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

	// --- SEARCH FACTORS (now owned by TuningConfig / search.json; see tuning.go) ---
	EmbedModel string
	// EmbedTaskPrefix opts into asymmetric task-instruction prefixes for the
	// embedding model (read from <prefix>_EMBED_TASK_PREFIX). For nomic-embed-text
	// this applies "search_query:"/"search_document:", a measured +0.20 recall on
	// the cli-health command corpus. Default off: flipping it on changes the
	// embedding space, so the adopter must reindex (the recipe-aware drift hash
	// triggers the re-embed automatically). Symmetric corpora / already-tuned
	// adopters (e.g. KO's guarded baseline) leave it off until validated.
	EmbedTaskPrefix bool
	// RelevanceMaxGap / RelevanceHardFloor are consumer *overrides* for the
	// ApplyRelevanceFloor band (WS2), read from <prefix>_RELEVANCE_MAX_GAP /
	// _RELEVANCE_HARD_FLOOR. They default to 0 ("unset") so FloorForMethodLeg supplies
	// the regime-appropriate default; a non-zero value overrides it. The package
	// owns the right band per regime — these exist only to override, not to seed.
	RelevanceMaxGap    float64
	RelevanceHardFloor float64
	// RerankEnabled gates the reranker chain (WS4) — the one genuine per-corpus
	// lever (precision/junk-rejection corpora win; recall corpora don't). Default
	// off so a resource-less consumer degrades cleanly to dense order. RerankModel
	// selects the LLM-fallback model; RerankShortlist is the over-fetch depth.
	// Read from <prefix>_RERANK_ENABLED / _RERANK_MODEL / _RERANK_SHORTLIST.
	RerankEnabled   bool
	RerankModel     string
	RerankShortlist int
	// RerankBlend fuses the reranker order with the retrieval order via RRF rather
	// than letting the reranker reorder outright (read from <prefix>_RERANK_BLEND).
	// It keeps the reranker's junk rejection while not burying strongly-retrieved
	// canonical results — a measured +0.20 recall on the cli-health command corpus
	// with no precision loss. Default off (opt-in, like the other rerank levers).
	RerankBlend bool
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
		EmbedTaskPrefix:      envBool(key("EMBED_TASK_PREFIX")),
		RelevanceMaxGap:      envFloat(key("RELEVANCE_MAX_GAP"), 0),
		RelevanceHardFloor:   envFloat(key("RELEVANCE_HARD_FLOOR"), 0),
		RerankEnabled:        envBool(key("RERANK_ENABLED")),
		RerankBlend:          envBool(key("RERANK_BLEND")),
		RerankModel:          envString(key("RERANK_MODEL"), DefaultRerankModel),
		RerankShortlist:      envInt(key("RERANK_SHORTLIST"), DefaultRerankShortlist, MinRerankShortlist, MaxRerankShortlist),
		RerankerURL:          envString(key("RERANKER_URL"), ""),
		RerankerModel:        envString(key("RERANKER_MODEL"), ""),
	}
}

func envFloat(name string, def float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		log.Printf("[aisearch] invalid env %s=%q, using default %g", name, raw, def)
		return def
	}
	return v
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
