package aisearch

import (
	"context"
	"testing"
)

// rerankSvc builds a Service whose dense leg returns three hits and whose
// reranker (when reached) reverses them, so a test can tell from the result
// order AND the stub's `called` flag whether the rerank pass actually ran for a
// given override resolution. constructEnabled seeds the construction-time
// rerank_enabled; policy is the OverridePolicy under test.
func rerankSvc(constructEnabled bool, policy OverridePolicy) (*Service, *stubReranker, *queryStore) {
	store := &queryStore{available: true, results: []SearchResult{
		docResult("a", 0.80, "alpha"),
		docResult("b", 0.70, "beta"),
		docResult("c", 0.60, "gamma"),
	}}
	rr := &stubReranker{
		name:      "cross-encoder:test",
		available: true,
		// Reverse the dense order so a successful rerank is visible: c,b,a.
		scores: []RerankScore{{ID: "a", Score: 0.10}, {ID: "b", Score: 0.50}, {ID: "c", Score: 0.90}},
	}
	svc := NewService(ServiceOptions{
		Embedder:       &countingEmbedder{},
		VectorStore:    store,
		Reranker:       NewRerankerChain(rr),
		RerankEnabled:  constructEnabled,
		ApplyFloor:     false, // isolate the rerank effect from floor drops
		Project:        docProjector,
		Shortlist:      50,
		OverridePolicy: policy,
	})
	return svc, rr, store
}

// --- resolveEffective: pure resolution / clamping / invariants (white box) ---

func TestResolveEffectiveNoOptionsUsesDefaults(t *testing.T) {
	svc, _, _ := rerankSvc(true, AllowOverrides())
	eff := svc.resolveEffective()
	if !eff.rerankEnabled || eff.shortlist != 50 {
		t.Fatalf("no-option resolution must mirror construction: %+v", eff)
	}
}

func TestResolveEffectiveHybridFusionOverride(t *testing.T) {
	svc := NewService(ServiceOptions{
		Embedder:       &countingEmbedder{},
		VectorStore:    &queryStore{available: true},
		HybridFusion:   HybridFusionRRF,
		OverridePolicy: AllowOverrides(),
	})
	eff := svc.resolveEffective(WithOverrides(SearchOverrides{HybridFusion: OverrideString(HybridFusionDBSF)}))
	if eff.hybridFusion != HybridFusionDBSF {
		t.Fatalf("hybrid_fusion override = %q, want %q", eff.hybridFusion, HybridFusionDBSF)
	}
	invalid := svc.resolveEffective(WithOverrides(SearchOverrides{HybridFusion: OverrideString("weighted")}))
	if invalid.hybridFusion != HybridFusionRRF {
		t.Fatalf("invalid hybrid_fusion override = %q, want construction default %q", invalid.hybridFusion, HybridFusionRRF)
	}
}

func TestResolveEffectiveNilPolicyDeniesOverrides(t *testing.T) {
	// nil policy (constructed without one) => deny: a passed override is ignored.
	svc, _, _ := rerankSvc(false, nil)
	eff := svc.resolveEffective(WithOverrides(SearchOverrides{RerankEnabled: OverrideBool(true)}))
	if eff.rerankEnabled {
		t.Fatal("nil OverridePolicy must deny overrides (rerank stayed disabled expected)")
	}
}

func TestResolveEffectiveDenyPolicyDeniesOverrides(t *testing.T) {
	svc, _, _ := rerankSvc(false, DenyOverrides())
	eff := svc.resolveEffective(WithOverrides(SearchOverrides{RerankEnabled: OverrideBool(true)}))
	if eff.rerankEnabled {
		t.Fatal("DenyOverrides must drop the rerank_enabled override")
	}
}

func TestResolveEffectiveAllowAppliesOverrides(t *testing.T) {
	svc, _, _ := rerankSvc(false, AllowOverrides())
	eff := svc.resolveEffective(WithOverrides(SearchOverrides{
		RerankEnabled:   OverrideBool(true),
		RerankBlend:     OverrideBool(true),
		RerankShortlist: OverrideInt(25),
		FloorMaxGap:     OverrideFloat(0.4),
		FloorHardFloor:  OverrideFloat(0.2),
	}))
	if !eff.rerankEnabled || !eff.rerankBlend || eff.shortlist != 25 {
		t.Fatalf("allow policy must apply rerank overrides: %+v", eff)
	}
	if eff.floor.MaxGap != 0.4 || eff.floor.HardFloor != 0.2 {
		t.Fatalf("allow policy must apply floor overrides: %+v", eff.floor)
	}
}

func TestResolveEffectiveShortlistClampedToTaxonomy(t *testing.T) {
	svc, _, _ := rerankSvc(true, AllowOverrides())
	hi := svc.resolveEffective(WithOverrides(SearchOverrides{RerankShortlist: OverrideInt(100000)}))
	if hi.shortlist != MaxRerankShortlist {
		t.Fatalf("over-max shortlist must clamp to %d, got %d", MaxRerankShortlist, hi.shortlist)
	}
	lo := svc.resolveEffective(WithOverrides(SearchOverrides{RerankShortlist: OverrideInt(-5)}))
	if lo.shortlist != MinRerankShortlist {
		t.Fatalf("under-min shortlist must clamp to %d, got %d", MinRerankShortlist, lo.shortlist)
	}
}

func TestResolveEffectiveFloorClampedToUnitInterval(t *testing.T) {
	svc, _, _ := rerankSvc(true, AllowOverrides())
	eff := svc.resolveEffective(WithOverrides(SearchOverrides{
		FloorMaxGap:    OverrideFloat(2.0),
		FloorHardFloor: OverrideFloat(-1.0),
	}))
	if eff.floor.MaxGap != 1.0 {
		t.Fatalf("floor.max_gap must clamp to 1.0, got %g", eff.floor.MaxGap)
	}
	if eff.floor.HardFloor != 0.0 {
		t.Fatalf("floor.hard_floor must clamp to 0.0, got %g", eff.floor.HardFloor)
	}
}

func TestResolveEffectiveBlendRequiresRerankEnabled(t *testing.T) {
	// blend=true alone, with rerank resolving OFF, is dropped (the invariant).
	svc, _, _ := rerankSvc(false, AllowOverrides())
	dropped := svc.resolveEffective(WithOverrides(SearchOverrides{RerankBlend: OverrideBool(true)}))
	if dropped.rerankBlend {
		t.Fatal("rerank_blend must be dropped when rerank resolves disabled")
	}
	// blend=true alongside an enabling override is kept.
	kept := svc.resolveEffective(WithOverrides(SearchOverrides{
		RerankEnabled: OverrideBool(true),
		RerankBlend:   OverrideBool(true),
	}))
	if !kept.rerankBlend {
		t.Fatal("rerank_blend must survive when rerank is also enabled by the override")
	}
}

func TestResolveEffectiveDoesNotMutateService(t *testing.T) {
	svc, _, _ := rerankSvc(false, AllowOverrides())
	_ = svc.resolveEffective(WithOverrides(SearchOverrides{
		RerankEnabled:   OverrideBool(true),
		RerankShortlist: OverrideInt(7),
	}))
	if svc.rerankEnabled || svc.shortlist != 50 {
		t.Fatalf("override resolution must not mutate the shared Service: enabled=%v shortlist=%d",
			svc.rerankEnabled, svc.shortlist)
	}
}

// custom subset policy: permit only the shortlist factor, drop everything else.
type shortlistOnlyPolicy struct{}

func (shortlistOnlyPolicy) Permit(o SearchOverrides) SearchOverrides {
	return SearchOverrides{RerankShortlist: o.RerankShortlist}
}

func TestResolveEffectiveCustomPolicyLimitsFactors(t *testing.T) {
	svc, _, _ := rerankSvc(false, shortlistOnlyPolicy{})
	eff := svc.resolveEffective(WithOverrides(SearchOverrides{
		RerankEnabled:   OverrideBool(true), // dropped by the policy
		RerankShortlist: OverrideInt(12),    // kept
	}))
	if eff.rerankEnabled {
		t.Fatal("custom policy dropped rerank_enabled; it must not take effect")
	}
	if eff.shortlist != 12 {
		t.Fatalf("custom policy kept rerank_shortlist; want 12 got %d", eff.shortlist)
	}
}

// --- end-to-end behavior through Service.Search ---

func TestSearchOverrideEnablesRerankEndToEnd(t *testing.T) {
	svc, rr, _ := rerankSvc(false, AllowOverrides())
	// Baseline: no override → rerank off → dense order a,b,c, reranker untouched.
	base, err := svc.Search(context.Background(), SearchQuery{Query: "alpha", Mode: ModeDense, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if rr.called {
		t.Fatal("baseline must not invoke the reranker (construction rerank off)")
	}
	if base.Results[0].ID != "a" {
		t.Fatalf("baseline dense order should lead with a, got %s", base.Results[0].ID)
	}

	// Override on → rerank runs → reversed order c,b,a, reranker invoked.
	rr.called = false
	resp, err := svc.Search(context.Background(), SearchQuery{Query: "alpha", Mode: ModeDense, Limit: 10},
		WithOverrides(SearchOverrides{RerankEnabled: OverrideBool(true)}))
	if err != nil {
		t.Fatal(err)
	}
	if !rr.called {
		t.Fatal("override rerank_enabled=true must invoke the reranker")
	}
	if resp.Results[0].ID != "c" {
		t.Fatalf("reranked order should lead with c, got %s", resp.Results[0].ID)
	}
}

func TestSearchOverrideDisablesRerankEndToEnd(t *testing.T) {
	svc, rr, _ := rerankSvc(true, AllowOverrides())
	_, err := svc.Search(context.Background(), SearchQuery{Query: "alpha", Mode: ModeDense, Limit: 10},
		WithOverrides(SearchOverrides{RerankEnabled: OverrideBool(false)}))
	if err != nil {
		t.Fatal(err)
	}
	if rr.called {
		t.Fatal("override rerank_enabled=false must skip the reranker even though construction enabled it")
	}
}

func TestSearchOverrideShortlistChangesOverfetch(t *testing.T) {
	// rerank on (so overfetch is active) + AllowOverrides; the store should be
	// asked for the clamped override shortlist, not the construction shortlist.
	svc, _, store := rerankSvc(true, AllowOverrides())
	_, err := svc.Search(context.Background(), SearchQuery{Query: "alpha", Mode: ModeDense, Limit: 5},
		WithOverrides(SearchOverrides{RerankShortlist: OverrideInt(33)}))
	if err != nil {
		t.Fatal(err)
	}
	if store.lastQuery.Limit != 33 {
		t.Fatalf("over-fetch should use the override shortlist 33, got %d", store.lastQuery.Limit)
	}
}

func TestSearchOverrideIgnoredUnderDenyEndToEnd(t *testing.T) {
	svc, rr, _ := rerankSvc(false, DenyOverrides())
	_, err := svc.Search(context.Background(), SearchQuery{Query: "alpha", Mode: ModeDense, Limit: 10},
		WithOverrides(SearchOverrides{RerankEnabled: OverrideBool(true)}))
	if err != nil {
		t.Fatal(err)
	}
	if rr.called {
		t.Fatal("DenyOverrides must keep rerank off despite the override")
	}
}

func TestSearchOverrideViaTypedForwardsOptions(t *testing.T) {
	// SearchTyped must forward the option set so a typed adopter (cli-health) also
	// gets the override path, not just the raw Service.Search caller.
	svc, rr, _ := rerankSvc(false, AllowOverrides())
	_, _, err := SearchTyped(context.Background(), svc,
		SearchQuery{Query: "alpha", Mode: ModeDense, Limit: 10},
		func(r SearchResult) string { return r.ID },
		WithOverrides(SearchOverrides{RerankEnabled: OverrideBool(true)}))
	if err != nil {
		t.Fatal(err)
	}
	if !rr.called {
		t.Fatal("SearchTyped must forward WithOverrides to Service.Search")
	}
}

func TestOverridesHeaderRoundTrip(t *testing.T) {
	in := SearchOverrides{
		RerankEnabled:   OverrideBool(true),
		RerankShortlist: OverrideInt(40),
		FloorMaxGap:     OverrideFloat(0.3),
		HybridFusion:    OverrideString(HybridFusionDBSF),
	}
	val, err := MarshalOverridesHeader(in)
	if err != nil {
		t.Fatal(err)
	}
	if val == "" {
		t.Fatal("non-zero overrides must encode to a non-empty header")
	}
	out, err := ParseOverridesHeader(val)
	if err != nil {
		t.Fatal(err)
	}
	if out.RerankEnabled == nil || !*out.RerankEnabled {
		t.Fatalf("rerank_enabled did not round-trip: %+v", out)
	}
	if out.RerankShortlist == nil || *out.RerankShortlist != 40 {
		t.Fatalf("rerank_shortlist did not round-trip: %+v", out)
	}
	if out.RerankBlend != nil {
		t.Fatal("an unset factor must stay unset across the round-trip")
	}
	if out.HybridFusion == nil || *out.HybridFusion != HybridFusionDBSF {
		t.Fatalf("hybrid_fusion did not round-trip: %+v", out)
	}
}

func TestOverridesHeaderZeroAndEmpty(t *testing.T) {
	val, err := MarshalOverridesHeader(SearchOverrides{})
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Fatalf("zero overrides must encode to empty, got %q", val)
	}
	out, err := ParseOverridesHeader("   ")
	if err != nil || !out.IsZero() {
		t.Fatalf("blank header must parse to zero overrides, got %+v err=%v", out, err)
	}
	if _, err := ParseOverridesHeader("{not json"); err == nil {
		t.Fatal("malformed header must return an error (handler ignores-with-telemetry)")
	}
}

func TestSearchOverridesIsZero(t *testing.T) {
	if !(SearchOverrides{}).IsZero() {
		t.Fatal("empty overrides must report IsZero")
	}
	if (SearchOverrides{RerankEnabled: OverrideBool(false)}).IsZero() {
		t.Fatal("a set (even false) field must not report IsZero")
	}
}
