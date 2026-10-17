package aisearch

import (
	"strings"
	"testing"
)

// TestFactorTaxonomyConsistency asserts the Factors table and the TuningConfig
// struct cannot drift: every TuningConfig field has exactly one factor row, each
// row has a tier and a non-empty tradeoff, and enum/range factors are
// well-formed. This is the single internal-consistency guard the plan §9 requires.
func TestFactorTaxonomyConsistency(t *testing.T) {
	wantKeys := map[string]bool{
		"engine": true, "embed_model": true, "embed_task_prefix": true,
		"rerank_enabled": true, "rerank_blend": true, "rerank_shortlist": true, "rerank_preference": true,
		"hybrid_fusion": true,
		"floor_max_gap": true, "floor_hard_floor": true,
	}
	got := map[string]bool{}
	for _, f := range Factors {
		if got[f.Key] {
			t.Errorf("duplicate factor key %q", f.Key)
		}
		got[f.Key] = true
		if f.Tier != QueryTime && f.Tier != IndexTime && f.Tier != Router {
			t.Errorf("factor %q has invalid tier %q", f.Key, f.Tier)
		}
		if strings.TrimSpace(f.Tradeoff) == "" {
			t.Errorf("factor %q has no tradeoff", f.Key)
		}
		if f.Default == nil {
			t.Errorf("factor %q has no default", f.Key)
		}
		switch f.Kind {
		case FactorEnum:
			if len(f.Enum) == 0 {
				t.Errorf("enum factor %q has no legal values", f.Key)
			}
		case FactorInt, FactorFloat:
			if f.Max != 0 && f.Min > f.Max {
				t.Errorf("factor %q has min>max (%g>%g)", f.Key, f.Min, f.Max)
			}
		case FactorDuration:
			// Duration defaults are typed time.Duration values. The taxonomy
			// only needs to guarantee that a concrete default exists.
		case FactorBool:
		default:
			t.Errorf("factor %q has unknown kind %q", f.Key, f.Kind)
		}
	}
	for k := range wantKeys {
		if !got[k] {
			t.Errorf("Factors is missing a row for TuningConfig field %q", k)
		}
	}
	for k := range got {
		if row, ok := FactorByKey(k); ok && row.Tier == Router {
			continue
		}
		if !wantKeys[k] {
			t.Errorf("Factors has a row %q with no matching TuningConfig field", k)
		}
	}
}

func TestRouterFactorTaxonomyConsistency(t *testing.T) {
	rows := map[string]Factor{}
	for _, f := range Factors {
		if f.Tier == Router {
			rows[f.Key] = f
		}
	}
	if len(rows) != len(RouterFactorKeys) {
		t.Fatalf("router factor rows=%d, keys=%d", len(rows), len(RouterFactorKeys))
	}
	for _, key := range RouterFactorKeys {
		row, ok := rows[key]
		if !ok {
			t.Errorf("router factor %q has no taxonomy row", key)
			continue
		}
		if row.Tier != Router {
			t.Errorf("router factor %q has tier %q", key, row.Tier)
		}
		if strings.TrimSpace(row.Tradeoff) == "" {
			t.Errorf("router factor %q has no tradeoff", key)
		}
		if row.Default == nil {
			t.Errorf("router factor %q has no default", key)
		}
	}
}

func TestQueryVsIndexTimeFactors(t *testing.T) {
	qt := map[string]bool{}
	for _, k := range QueryTimeFactors() {
		qt[k] = true
	}
	it := map[string]bool{}
	for _, k := range IndexTimeFactors() {
		it[k] = true
	}
	// Index-time factors are the ones a reindex must follow.
	for _, k := range []string{"engine", "embed_model", "embed_task_prefix"} {
		if !it[k] {
			t.Errorf("%q must be an index-time factor", k)
		}
		if qt[k] {
			t.Errorf("%q must NOT be a query-time factor", k)
		}
	}
	// Query-time factors are per-request safe (the override channel honors these).
	for _, k := range []string{"rerank_enabled", "rerank_blend", "rerank_shortlist", "rerank_preference", "hybrid_fusion", "floor_max_gap", "floor_hard_floor"} {
		if !qt[k] {
			t.Errorf("%q must be a query-time factor", k)
		}
		if it[k] {
			t.Errorf("%q must NOT be an index-time factor", k)
		}
	}
}

func TestTuningConfigWithDefaults(t *testing.T) {
	got := TuningConfig{}.WithDefaults()
	if got.Engine != EngineDense {
		t.Errorf("default engine = %q, want %q", got.Engine, EngineDense)
	}
	if got.EmbedModel != DefaultEmbedModel {
		t.Errorf("default embed_model = %q, want %q", got.EmbedModel, DefaultEmbedModel)
	}
	if got.RerankShortlist != DefaultRerankShortlist {
		t.Errorf("default rerank_shortlist = %d, want %d", got.RerankShortlist, DefaultRerankShortlist)
	}
	if got.HybridFusion != HybridFusionRRF {
		t.Errorf("default hybrid_fusion = %q, want %q", got.HybridFusion, HybridFusionRRF)
	}
	// WithDefaults must not mutate non-zero fields.
	in := TuningConfig{Engine: EngineHybrid, EmbedModel: "m", RerankShortlist: 7}
	out := in.WithDefaults()
	if out.Engine != EngineHybrid || out.EmbedModel != "m" || out.RerankShortlist != 7 {
		t.Errorf("WithDefaults clobbered set fields: %+v", out)
	}
}

func TestTuningConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     TuningConfig
		wantErr bool
	}{
		{"command preset", CommandCorpusTuning(), false},
		{"doc preset", DocCorpusTuning(), false},
		{"empty engine", TuningConfig{}, true},
		{"bad engine", TuningConfig{Engine: "sparse"}, true},
		{"shortlist too big", TuningConfig{Engine: EngineDense, RerankShortlist: MaxRerankShortlist + 1}, true},
		{"floor out of range", TuningConfig{Engine: EngineDense, Floor: FloorTuning{MaxGap: 1.5}}, true},
		{"hard floor out of range", TuningConfig{Engine: EngineDense, Floor: FloorTuning{HardFloor: -0.1}}, true},
		{"blend without rerank", TuningConfig{Engine: EngineDense, RerankBlend: true}, true},
		{"bad hybrid fusion", TuningConfig{Engine: EngineHybrid, HybridFusion: "weighted"}, true},
		{"dbsf hybrid fusion", TuningConfig{Engine: EngineHybrid, HybridFusion: HybridFusionDBSF}, false},
		{"floor zero is legal", TuningConfig{Engine: EngineDense}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestPresetsMatchMeasuredBest(t *testing.T) {
	cmd := CommandCorpusTuning()
	if cmd.Engine != EngineDense || !cmd.EmbedTaskPrefix || !cmd.RerankEnabled || !cmd.RerankBlend {
		t.Errorf("command preset is not the measured-best config: %+v", cmd)
	}
	doc := DocCorpusTuning()
	if doc.Engine != EngineHybrid || doc.EmbedTaskPrefix || doc.RerankEnabled {
		t.Errorf("doc preset is not the KO baseline config: %+v", doc)
	}
}

// TestNewServiceForTuningIsDataDriven asserts dense-vs-hybrid is chosen from the
// tuning data, not a code literal: the engine field alone flips the SparseEncoder.
func TestNewServiceForTuningIsDataDriven(t *testing.T) {
	deps := EngineDeps{QdrantURL: "http://localhost:6333", Collection: "test-coll"}

	dense := NewServiceForTuning(TuningConfig{Engine: EngineDense}, deps)
	if dense.SparseEncoder != nil {
		t.Error("dense engine must not have a sparse encoder")
	}
	if dense.Spec.Sparse {
		t.Error("dense engine spec must be dense-only")
	}
	if dense.Embedder == nil || dense.VectorStore == nil || dense.Reranker == nil {
		t.Error("dense engine missing assembled parts")
	}
	if dense.Tuning.Engine != EngineDense || dense.Tuning.RerankShortlist != DefaultRerankShortlist {
		t.Errorf("dense engine tuning not resolved: %+v", dense.Tuning)
	}

	hybrid := NewServiceForTuning(TuningConfig{Engine: EngineHybrid}, deps)
	if hybrid.SparseEncoder == nil {
		t.Error("hybrid engine must have a sparse encoder")
	}
	if !hybrid.Spec.Sparse || hybrid.Spec.SparseModifier != DefaultSparseModifier {
		t.Errorf("hybrid engine spec must enable sparse+idf: %+v", hybrid.Spec)
	}
	if hybrid.Spec.Name != "test-coll" {
		t.Errorf("collection name not threaded: %q", hybrid.Spec.Name)
	}
}

func TestNewServiceForTuningUsesResolvedEmbeddingDeps(t *testing.T) {
	engine := NewServiceForTuning(TuningConfig{Engine: EngineDense, EmbedModel: "stale"}, EngineDeps{
		QdrantURL:       "http://localhost:6333",
		Collection:      "test-coll",
		EmbedRole:       "embedding.default",
		EmbedModel:      fixtureEmbeddingModel,
		EmbedDimensions: fixtureEmbeddingDimensions,
	})

	if engine.Spec.DenseSize != fixtureEmbeddingDimensions {
		t.Fatalf("Spec.DenseSize = %d, want %d", engine.Spec.DenseSize, fixtureEmbeddingDimensions)
	}
	if engine.Spec.Model != fixtureEmbeddingModel {
		t.Fatalf("Spec.Model = %q, want resolved model", engine.Spec.Model)
	}
	if engine.Tuning.EmbedModel != fixtureEmbeddingModel {
		t.Fatalf("Tuning.EmbedModel = %q, want resolved model", engine.Tuning.EmbedModel)
	}
}

// TestTunedEngineServiceOptions asserts the helper carries every query-time
// read-path factor from the resolved tuning into ServiceOptions (so an adopter
// can never silently drop one by hand-forwarding), wires the engine components,
// and always runs the floor (regime-safe).
func TestTunedEngineServiceOptions(t *testing.T) {
	deps := EngineDeps{QdrantURL: "http://localhost:6333", Collection: "c"}
	tuning := TuningConfig{
		Engine:          EngineHybrid,
		EmbedModel:      DefaultEmbedModel,
		RerankEnabled:   true,
		RerankBlend:     true,
		RerankShortlist: 42,
		Floor:           FloorTuning{MaxGap: 0.4, HardFloor: 0.1},
	}
	te := NewServiceForTuning(tuning, deps)
	opts := te.ServiceOptions()

	if !opts.RerankEnabled || !opts.RerankBlend {
		t.Errorf("rerank flags not forwarded: %+v", opts)
	}
	if opts.Shortlist != 42 {
		t.Errorf("shortlist not forwarded: got %d want 42", opts.Shortlist)
	}
	if opts.HybridFusion != HybridFusionRRF {
		t.Errorf("default hybrid fusion not forwarded: got %q want %q", opts.HybridFusion, HybridFusionRRF)
	}
	if !opts.ApplyFloor {
		t.Error("ApplyFloor must be true (regime floor is always safe to run)")
	}
	if opts.Floor.MaxGap != 0.4 || opts.Floor.HardFloor != 0.1 {
		t.Errorf("floor not forwarded: %+v", opts.Floor)
	}
	if opts.Embedder == nil || opts.VectorStore == nil || opts.Reranker == nil {
		t.Error("engine components not wired into ServiceOptions")
	}
	if opts.SparseEncoder == nil {
		t.Error("hybrid tuning must carry the sparse encoder into ServiceOptions")
	}

	// A dense tuning omits the sparse encoder but still forwards the factors.
	dense := NewServiceForTuning(TuningConfig{Engine: EngineDense, EmbedModel: DefaultEmbedModel}, deps).ServiceOptions()
	if dense.SparseEncoder != nil {
		t.Error("dense tuning must not carry a sparse encoder")
	}
	if dense.Shortlist != DefaultRerankShortlist {
		t.Errorf("dense shortlist default not resolved: %d", dense.Shortlist)
	}
}

// TestNewServiceForTuningTaskPrefix asserts the embed-recipe (task prefix) flag
// flows through to the embedder: with the prefix on, nomic embeds asymmetrically
// (a non-empty recipe), so a collection re-embeds; with it off, the recipe is
// empty (KO baseline byte-identical protection).
func TestNewServiceForTuningTaskPrefix(t *testing.T) {
	deps := EngineDeps{QdrantURL: "http://localhost:6333", Collection: "c"}

	off := NewServiceForTuning(TuningConfig{Engine: EngineDense, EmbedModel: DefaultEmbedModel}, deps)
	if re, ok := off.Embedder.(RecipeEmbedder); ok && re.EmbedRecipe() != "" {
		t.Errorf("task-prefix off must yield an empty embed recipe, got %q", re.EmbedRecipe())
	}

	on := NewServiceForTuning(TuningConfig{Engine: EngineDense, EmbedModel: DefaultEmbedModel, EmbedTaskPrefix: true}, deps)
	re, ok := on.Embedder.(RecipeEmbedder)
	if !ok || re.EmbedRecipe() == "" {
		t.Error("task-prefix on must yield a non-empty embed recipe (forces re-embed)")
	}
}
