package aisearch

import (
	"fmt"
	"strings"
)

// tuning.go is the single source of truth for the search *tuning factors* — the
// knobs an adopter (or the search-hub sweep) is allowed to move to trade recall,
// precision, and latency. It deliberately separates two things the old
// env-and-code surface conflated:
//
//   - WIRING/operational config (which Qdrant, which reranker resource, the sync
//     cadence) — that stays in Config/EngineDeps; it is not a search "factor".
//   - TUNING factors (engine shape, embedding recipe, rerank policy, floor band)
//     — captured here as TuningConfig, described by the Factors taxonomy, and
//     read from the scenario-owned `.vrooli/search.json` (the SSOT). Nothing
//     search-tunable is a Go literal or an env var that is the source of truth.
//
// The factor taxonomy (Factors) records, for every knob, its cost TIER
// (QueryTime — variable per request, no reindex; IndexTime — changes the stored
// vectors, needs a reindex), its value domain, default, and the one-line
// tradeoff that tells an adopter when to move it. search-hub CONSUMES this table
// (to know what is sweepable and at what cost); it must not redefine it.

// Engine shapes (the structural, index-time factor). A dense engine stores only
// the dense vector; a hybrid engine adds the BM25 sparse leg for dense+sparse
// fusion (the lexical half a terse dense vector fumbles).
const (
	EngineDense  = "dense"
	EngineHybrid = "hybrid"
)

// Hybrid fusion strategies are Qdrant's server-side ways of combining the
// dense and sparse prefetch legs. RRF is the conservative default; DBSF keeps
// the score distributions when a sparse exact-term match should outweigh a
// merely similar dense sibling.
const (
	HybridFusionRRF  = "rrf"
	HybridFusionDBSF = "dbsf"
)

const (
	RerankPreferenceCrossEncoderRequired  = "cross_encoder_required"
	RerankPreferenceCrossEncoderPreferred = "cross_encoder_preferred"
)

func normalizeHybridFusion(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case HybridFusionDBSF:
		return HybridFusionDBSF
	case HybridFusionRRF, "":
		return HybridFusionRRF
	default:
		return ""
	}
}

// FactorTier classifies whether changing a factor requires a reindex. It is the
// split the override channel (Phase 4) enforces: only QueryTime factors may be
// varied per request; IndexTime factors are swept via config-push + reindex.
type FactorTier string

const (
	// QueryTime factors vary per request and never touch the stored vectors.
	QueryTime FactorTier = "query_time"
	// IndexTime factors change the embedded/indexed representation, so moving one
	// requires a reindex (the recipe-aware drift hash triggers it automatically).
	IndexTime FactorTier = "index_time"
)

// FactorKind is the value domain of a factor (so a sweep knows how to enumerate
// it and a validator knows how to bound it).
type FactorKind string

const (
	FactorBool  FactorKind = "bool"
	FactorEnum  FactorKind = "enum"
	FactorInt   FactorKind = "int"
	FactorFloat FactorKind = "float"
)

// Factor describes one tuning knob. It is the row of the control-surface
// "dashboard": key, cost tier, value domain, default, and the tradeoff that
// governs when to move it.
type Factor struct {
	// Key is the JSON key under `tuning` in search.json (e.g. "embed_task_prefix").
	Key string
	// Tier is the cost tier (QueryTime vs IndexTime).
	Tier FactorTier
	// Kind is the value domain.
	Kind FactorKind
	// Enum lists the legal values for a FactorEnum (nil otherwise).
	Enum []string
	// Min/Max bound a FactorInt/FactorFloat (inclusive); zero/zero means unbounded.
	Min, Max float64
	// Default is the package default when the field is absent/zero in search.json.
	Default any
	// Tradeoff is the one-line decision rule for an adopter.
	Tradeoff string
}

// Factors is the complete tuning taxonomy. Every TuningConfig field has exactly
// one row here; the factor-taxonomy test asserts that invariant so the table and
// the struct can never drift.
var Factors = []Factor{
	{
		Key: "engine", Tier: IndexTime, Kind: FactorEnum,
		Enum: []string{EngineDense, EngineHybrid}, Default: EngineDense,
		Tradeoff: "hybrid adds a BM25 sparse leg (recall on keyword/long-prose corpora) at index + query cost; dense is simpler and faster for terse corpora that embed well.",
	},
	{
		Key: "embed_model", Tier: IndexTime, Kind: FactorEnum,
		Enum: []string{DefaultEmbedModel}, Default: DefaultEmbedModel,
		Tradeoff: "the dense embedding model; changing it re-embeds the whole corpus, so only switch among already-installed models of a known dimension.",
	},
	{
		Key: "embed_task_prefix", Tier: IndexTime, Kind: FactorBool, Default: false,
		Tradeoff: "asymmetric search_query:/search_document: prefixes for nomic (+0.20 recall on terse command corpora); changes the embedding space → reindex. Leave off for symmetric/already-tuned corpora.",
	},
	{
		Key: "rerank_enabled", Tier: QueryTime, Kind: FactorBool, Default: false,
		Tradeoff: "cross-encoder/LLM rerank lifts precision and junk rejection; it buys no recall on prose corpora and adds latency + a reranker resource dependency. Off degrades cleanly to retrieval order.",
	},
	{
		Key: "rerank_blend", Tier: QueryTime, Kind: FactorBool, Default: false,
		Tradeoff: "fuse the rerank order with retrieval via RRF instead of reordering outright; keeps junk rejection without burying strongly-retrieved canonical hits (+0.20 recall on the command corpus). Only meaningful when rerank_enabled.",
	},
	{
		Key: "rerank_shortlist", Tier: QueryTime, Kind: FactorInt,
		Min: float64(MinRerankShortlist), Max: float64(MaxRerankShortlist), Default: DefaultRerankShortlist,
		Tradeoff: "over-fetch depth into the reranker; higher = more recall into the rerank but more candidates to score (LLM-leg latency; negligible on the cross-encoder).",
	},
	{
		Key: "rerank_preference", Tier: QueryTime, Kind: FactorEnum,
		Enum: []string{RerankPreferenceCrossEncoderRequired, RerankPreferenceCrossEncoderPreferred}, Default: RerankPreferenceCrossEncoderPreferred,
		Tradeoff: "required preserves the cross-encoder latency/SLO contract by refusing the expensive LLM fallback; preferred keeps recall available when the cross-encoder is unavailable.",
	},
	{
		Key: "hybrid_fusion", Tier: QueryTime, Kind: FactorEnum,
		Enum: []string{HybridFusionRRF, HybridFusionDBSF}, Default: HybridFusionRRF,
		Tradeoff: "how Qdrant combines dense and sparse legs; RRF is rank-robust, while DBSF preserves score separation for exact lexical matches. Only affects hybrid engines.",
	},
	{
		Key: "floor_max_gap", Tier: QueryTime, Kind: FactorFloat, Min: 0, Max: 1, Default: 0.0,
		Tradeoff: "relative cutoff below the query's top hit; 0 = let the package pick the regime-appropriate band. Raise to cut more of the weak tail.",
	},
	{
		Key: "floor_hard_floor", Tier: QueryTime, Kind: FactorFloat, Min: 0, Max: 1, Default: 0.0,
		Tradeoff: "absolute garbage floor; 0 = let the package pick the regime default. A non-zero value overrides it (cosine regimes want a real floor; fused regimes want 0).",
	},
}

// FactorByKey returns the taxonomy row for a key (ok=false if unknown).
func FactorByKey(key string) (Factor, bool) {
	for _, f := range Factors {
		if f.Key == key {
			return f, true
		}
	}
	return Factor{}, false
}

// QueryTimeFactors / IndexTimeFactors return the factor keys for each tier — the
// split the sweep uses (query-time = cheap full-factorial; index-time =
// reindex-per-arm coordinate ascent) and the override channel enforces.
func QueryTimeFactors() []string { return factorKeysForTier(QueryTime) }
func IndexTimeFactors() []string { return factorKeysForTier(IndexTime) }

func factorKeysForTier(tier FactorTier) []string {
	out := make([]string, 0, len(Factors))
	for _, f := range Factors {
		if f.Tier == tier {
			out = append(out, f.Key)
		}
	}
	return out
}

// FloorTuning is the JSON-tagged floor block of a TuningConfig. It mirrors
// FloorConfig but carries snake_case tags for search.json; Config() converts it.
type FloorTuning struct {
	MaxGap    float64 `json:"max_gap"`
	HardFloor float64 `json:"hard_floor"`
}

// Config converts the wire shape to the package FloorConfig the floor band uses.
// The two structs share field names/types (FloorTuning only adds JSON tags), so a
// direct conversion suffices.
func (f FloorTuning) Config() FloorConfig {
	return FloorConfig(f)
}

// TuningConfig is the typed, validated value of the `tuning` block in
// search.json. It is the ONE place the knob values live at runtime: the adopter
// reads it at boot and the sweep writes it back. Every field corresponds to a
// row in Factors.
type TuningConfig struct {
	Engine           string      `json:"engine"`
	EmbedModel       string      `json:"embed_model"`
	EmbedTaskPrefix  bool        `json:"embed_task_prefix"`
	RerankEnabled    bool        `json:"rerank_enabled"`
	RerankBlend      bool        `json:"rerank_blend"`
	RerankShortlist  int         `json:"rerank_shortlist"`
	RerankPreference string      `json:"rerank_preference"`
	HybridFusion     string      `json:"hybrid_fusion"`
	Floor            FloorTuning `json:"floor"`
}

// WithDefaults returns a copy with absent/zero fields filled from the taxonomy
// defaults so a partial `tuning` block is always meaningful. It does not mutate
// the receiver.
func (t TuningConfig) WithDefaults() TuningConfig {
	if strings.TrimSpace(t.Engine) == "" {
		t.Engine = EngineDense
	}
	if strings.TrimSpace(t.EmbedModel) == "" {
		t.EmbedModel = DefaultEmbedModel
	}
	if t.RerankShortlist <= 0 {
		t.RerankShortlist = DefaultRerankShortlist
	}
	if strings.TrimSpace(t.RerankPreference) == "" {
		t.RerankPreference = RerankPreferenceCrossEncoderPreferred
	}
	if strings.TrimSpace(t.HybridFusion) == "" {
		t.HybridFusion = HybridFusionRRF
	}
	return t
}

// IndexTimeChanged reports whether any INDEX-TIME factor (engine, embed_model,
// embed_task_prefix) differs between the receiver and other. It is the predicate
// the config-write contract uses to decide whether persisting a new tuning block
// also requires a reindex: query-time-only changes take effect without touching
// the stored vectors, whereas an index-time change alters the embedded
// representation. Both sides should be defaults-filled (WithDefaults) before
// comparing so an absent field is not mistaken for a change.
func (t TuningConfig) IndexTimeChanged(other TuningConfig) bool {
	a, b := t.WithDefaults(), other.WithDefaults()
	return a.Engine != b.Engine ||
		a.EmbedModel != b.EmbedModel ||
		a.EmbedTaskPrefix != b.EmbedTaskPrefix
}

// Validate checks the config against the factor taxonomy: engine is a known
// shape, the shortlist is in range, the floors are in [0,1]. It returns the
// first violation. Floor 0 ("regime default") is always legal.
func (t TuningConfig) Validate() error {
	switch t.Engine {
	case EngineDense, EngineHybrid:
	case "":
		// treated as dense by WithDefaults, but a raw config should be explicit.
		return fmt.Errorf("tuning.engine is empty (expected %q or %q)", EngineDense, EngineHybrid)
	default:
		return fmt.Errorf("tuning.engine %q is not a known engine (expected %q or %q)", t.Engine, EngineDense, EngineHybrid)
	}
	if t.RerankShortlist != 0 && (t.RerankShortlist < MinRerankShortlist || t.RerankShortlist > MaxRerankShortlist) {
		return fmt.Errorf("tuning.rerank_shortlist %d out of range [%d,%d]", t.RerankShortlist, MinRerankShortlist, MaxRerankShortlist)
	}
	switch strings.ToLower(strings.TrimSpace(t.HybridFusion)) {
	case HybridFusionRRF, HybridFusionDBSF:
	case "":
		// WithDefaults supplies the conservative RRF default.
	default:
		return fmt.Errorf("tuning.hybrid_fusion %q is not known (expected %q or %q)", t.HybridFusion, HybridFusionRRF, HybridFusionDBSF)
	}
	switch strings.TrimSpace(t.RerankPreference) {
	case RerankPreferenceCrossEncoderRequired, RerankPreferenceCrossEncoderPreferred:
	case "":
		// WithDefaults supplies the preferred default.
	default:
		return fmt.Errorf("tuning.rerank_preference %q is not known (expected %q or %q)", t.RerankPreference, RerankPreferenceCrossEncoderRequired, RerankPreferenceCrossEncoderPreferred)
	}
	if t.Floor.MaxGap < 0 || t.Floor.MaxGap > 1 {
		return fmt.Errorf("tuning.floor.max_gap %g out of range [0,1]", t.Floor.MaxGap)
	}
	if t.Floor.HardFloor < 0 || t.Floor.HardFloor > 1 {
		return fmt.Errorf("tuning.floor.hard_floor %g out of range [0,1]", t.Floor.HardFloor)
	}
	if t.RerankBlend && !t.RerankEnabled {
		return fmt.Errorf("tuning.rerank_blend requires rerank_enabled (blend fuses the rerank order; with rerank off it is a no-op)")
	}
	return nil
}

// CommandCorpusTuning returns the measured-best TuningConfig for a terse command
// corpus (cli-health): dense engine, nomic task prefixes, and the rerank RRF
// blend — the configuration that lifts recall@5 0.50 -> 0.70 without losing junk
// rejection (see packages/ai-go/search/docs/graduation-retrospective.md).
func CommandCorpusTuning() TuningConfig {
	return TuningConfig{
		Engine:          EngineDense,
		EmbedModel:      DefaultEmbedModel,
		EmbedTaskPrefix: true,
		RerankEnabled:   true,
		RerankBlend:     true,
		RerankShortlist: DefaultRerankShortlist,
		HybridFusion:    HybridFusionRRF,
	}
}

// DocCorpusTuning returns the measured-best TuningConfig for a large prose/doc
// corpus (knowledge-observatory): hybrid engine, symmetric embeddings, rerank
// off. The validated finding is that hybrid RRF ties the cross-encoder and beats
// the LLM reranker on recall for prose — reranking buys ordering parity, not
// recall — and that the symmetric embedder preserves KO's guarded recall@5=0.818
// baseline exactly (the recipe-aware drift hash keeps an empty recipe byte
// identical). Reproduces today's KO behavior.
func DocCorpusTuning() TuningConfig {
	return TuningConfig{
		Engine:          EngineHybrid,
		EmbedModel:      DefaultEmbedModel,
		EmbedTaskPrefix: false,
		RerankEnabled:   false,
		RerankBlend:     false,
		RerankShortlist: DefaultRerankShortlist,
		HybridFusion:    HybridFusionRRF,
	}
}

// EngineDeps carries the operational (non-tuning) wiring NewServiceForTuning
// needs: the vector backend address + collection, and the reranker resource
// endpoints. These are environment/deployment facts, not search factors, so they
// stay outside TuningConfig (and outside search.json).
type EngineDeps struct {
	QdrantURL                string
	QdrantAPIKey             string
	Collection               string
	EmbedRole                string // resolved embedding role ("" => DefaultEmbedRole)
	EmbedModel               string // resolved embedding model ("" => tuning/default fallback)
	EmbedDimensions          int    // resolved embedding dimensions (0 => DefaultVectorSize fallback)
	EmbedPolicySchemaVersion string // resolved Ollama policy schema version
	RerankerURL              string // cross-encoder resource URL ("" => resource default env)
	RerankerModel            string // cross-encoder model ("" => resource default)
	RerankRole               string // LLM-fallback leg role ("" => DefaultRerankRole)
}

// TunedEngine is the assembled bundle a tuning-driven adopter wires at boot. It
// generalizes DenseEngine/HybridEngine: SparseEncoder is non-nil exactly when
// Tuning.Engine == "hybrid", and Tuning is the resolved (defaults-filled) config
// so the adopter forwards the query-time knobs (RerankEnabled/RerankBlend/
// RerankShortlist/Floor) into its read-path Service without re-deriving them.
type TunedEngine struct {
	Embedder      Embedder
	VectorStore   VectorStore
	SparseEncoder SparseEncoder // nil for dense
	Reranker      *RerankerChain
	Spec          CollectionSpec
	Tuning        TuningConfig
}

// ServiceOptions returns the ServiceOptions an adopter should start from when
// wiring the read-path Service for this tuned engine: the assembled engine
// components (Embedder/SparseEncoder/VectorStore/Reranker) plus the query-time
// read-path factors derived from the resolved TuningConfig (RerankEnabled,
// RerankBlend, Shortlist, Floor) and ApplyFloor=true. It exists so an adopter
// never hand-forwards te.Tuning.RerankEnabled/RerankBlend/RerankShortlist/Floor
// into ServiceOptions one field at a time (forget one and the Service silently
// runs misconfigured — rerank-off when the SSOT says on). The adopter overlays
// its own seams (Reconciler/Project/Filter/PostFilter/Decorate/RerankText/
// TextFallback/OverridePolicy) on the returned value and passes it to NewService.
//
// ApplyFloor is true unconditionally: the regime-aware floor (FloorForMethodLeg)
// already disables the absolute HardFloor for fused/rerank-off-hybrid legs, so
// running the floor is safe for every engine shape; "off" is no longer a regime
// workaround an adopter must opt into. Floor carries the operator override
// (max_gap/hard_floor) merged onto the regime default.
func (e TunedEngine) ServiceOptions() ServiceOptions {
	return ServiceOptions{
		Embedder:      e.Embedder,
		SparseEncoder: e.SparseEncoder,
		VectorStore:   e.VectorStore,
		Reranker:      e.Reranker,
		RerankEnabled: e.Tuning.RerankEnabled,
		RerankBlend:   e.Tuning.RerankBlend,
		Shortlist:     e.Tuning.RerankShortlist,
		HybridFusion:  e.Tuning.HybridFusion,
		ApplyFloor:    true,
		Floor:         e.Tuning.Floor.Config(),
	}
}

// NewServiceForTuning assembles the engine the TuningConfig describes — dense OR
// hybrid, decided by data (Tuning.Engine), not by a code literal. It delegates to
// the existing NewDenseEngine/NewHybridEngine assemblers so it cannot diverge
// from the hand-wired path; it only chooses between them and threads the
// embed-recipe (task prefix) decision through. The query-time knobs are returned
// in TunedEngine.Tuning for the adopter to forward into its Service.
func NewServiceForTuning(tuning TuningConfig, deps EngineDeps) TunedEngine {
	tuning = tuning.WithDefaults()
	cfg := Config{
		EmbedModel:               tuning.EmbedModel,
		EmbedRole:                deps.EmbedRole,
		EmbedDimensions:          deps.EmbedDimensions,
		EmbedPolicySchemaVersion: deps.EmbedPolicySchemaVersion,
		EmbedTaskPrefix:          tuning.EmbedTaskPrefix,
		QdrantURL:                deps.QdrantURL,
		QdrantAPIKey:             deps.QdrantAPIKey,
		RerankerURL:              deps.RerankerURL,
		RerankerModel:            deps.RerankerModel,
		RerankRole:               deps.RerankRole,
	}
	if strings.TrimSpace(cfg.RerankRole) == "" {
		cfg.RerankRole = DefaultRerankRole
	}
	if strings.TrimSpace(cfg.EmbedRole) == "" {
		cfg.EmbedRole = DefaultEmbedRole
	}
	if strings.TrimSpace(deps.EmbedModel) != "" {
		cfg.EmbedModel = deps.EmbedModel
		tuning.EmbedModel = deps.EmbedModel
	}
	switch tuning.Engine {
	case EngineHybrid:
		e := NewHybridEngine(cfg, deps.Collection)
		return TunedEngine{
			Embedder:      e.Embedder,
			VectorStore:   e.VectorStore,
			SparseEncoder: e.SparseEncoder,
			Reranker:      e.Reranker,
			Spec:          e.Spec,
			Tuning:        tuning,
		}
	default:
		e := NewDenseEngine(cfg, deps.Collection)
		return TunedEngine{
			Embedder:    e.Embedder,
			VectorStore: e.VectorStore,
			Reranker:    e.Reranker,
			Spec:        e.Spec,
			Tuning:      tuning,
		}
	}
}
