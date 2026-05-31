package autosteer

import (
	"context"
	"errors"
	"testing"

	"github.com/ecosystem-manager/api/pkg/dimensions"
	"github.com/ecosystem-manager/api/pkg/dtv"
	"github.com/ecosystem-manager/api/pkg/effectiveness"
)

func snapshotOf(fits map[string]dtv.Fitness) FitnessSnapshot { return NewFitnessSnapshot(fits) }

// ── Phase 3: DTVEligibilityFilter (Layer-1 gate) ────────────────────────────

func TestDTVEligibilityFilter_DeniesOnlyRed(t *testing.T) {
	snap := snapshotOf(map[string]dtv.Fitness{
		"red":    {Verdict: dtv.VerdictRed},
		"green":  {Verdict: dtv.VerdictGreen},
		"yellow": {Verdict: dtv.VerdictYellow},
		// "absent" is UNKNOWN by omission.
	})
	f := NewDTVEligibilityFilter(snap)
	cases := map[string]bool{"red": false, "green": true, "yellow": true, "absent": true}
	for skill, want := range cases {
		if got := f.Allow(skill, "standards"); got != want {
			t.Errorf("Allow(%q) = %v, want %v (UNKNOWN must fail open)", skill, got, want)
		}
	}
}

func TestSelector_GatesRedSkillBeforeRanking(t *testing.T) {
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	// refactor is RED (gated); lint-fix is green and must be chosen.
	snap := snapshotOf(map[string]dtv.Fitness{"refactor": {Verdict: dtv.VerdictRed}})
	sel := NewSelectorWithConfig(SelectorConfig{
		Resolver:      res,
		Effectiveness: effectiveness.NewMemoryStore(),
		Filter:        NewDTVEligibilityFilter(snap),
	}).SelectNextSkill(standardsState(), profile)

	if sel.SkillID != "lint-fix" {
		t.Fatalf("expected RED 'refactor' gated out, got %q", sel.SkillID)
	}
	if len(sel.ExcludedSkills) != 1 || sel.ExcludedSkills[0] != "refactor" {
		t.Fatalf("excluded skills = %v, want [refactor]", sel.ExcludedSkills)
	}
	if sel.GateOverride {
		t.Fatal("gate override must not fire when an eligible skill remains")
	}
}

func TestSelector_AllRedDoesNotStall(t *testing.T) {
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	snap := snapshotOf(map[string]dtv.Fitness{
		"refactor": {Verdict: dtv.VerdictRed},
		"lint-fix": {Verdict: dtv.VerdictRed},
	})
	sel := NewSelectorWithConfig(SelectorConfig{
		Resolver:      res,
		Effectiveness: effectiveness.NewMemoryStore(),
		Filter:        NewDTVEligibilityFilter(snap),
	}).SelectNextSkill(standardsState(), profile)

	if sel.SkillID == "" {
		t.Fatal("all-red must not stall — the safety valve should still pick a skill")
	}
	if !sel.GateOverride {
		t.Fatal("expected GateOverride flagged when every candidate is gated")
	}
	if len(sel.ExcludedSkills) != 2 {
		t.Fatalf("expected both skills recorded as excluded, got %v", sel.ExcludedSkills)
	}
}

func TestSelector_GateDisabledAllowsAll(t *testing.T) {
	// AllowAllFilter is what the orchestrator wires when dtv.gate_enabled=false.
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	sel := NewSelectorWithConfig(SelectorConfig{
		Resolver:      res,
		Effectiveness: effectiveness.NewMemoryStore(),
		Filter:        AllowAllFilter{},
	}).SelectNextSkill(standardsState(), profile)
	if sel.SkillID != "refactor" || len(sel.ExcludedSkills) != 0 {
		t.Fatalf("allow-all must not exclude anything: skill=%q excluded=%v", sel.SkillID, sel.ExcludedSkills)
	}
}

// ── Phase 4: DTVPriorProvider (trust/cost prior) ────────────────────────────

func TestDTVPriorProvider_MappingTable(t *testing.T) {
	cfg := DTVPriorConfig{} // defaults: weight 1, base 1, minRuns 2, convK 1, stale 0.5
	prov := func(f dtv.Fitness) float64 {
		return NewDTVPriorProvider(snapshotOf(map[string]dtv.Fitness{"s": f}), cfg).Prior("s", "standards")
	}
	tests := []struct {
		name string
		f    dtv.Fitness
		want float64
	}{
		{"unknown→0", dtv.Fitness{Verdict: dtv.VerdictUnknown, PassRate: 1, TotalRuns: 9}, 0},
		{"thin-evidence→0", dtv.Fitness{Verdict: dtv.VerdictGreen, PassRate: 1, TotalRuns: 1, UniqueDiffHashes: 1}, 0},
		{"green-converged→1.0", dtv.Fitness{Verdict: dtv.VerdictGreen, PassRate: 1, TotalRuns: 5, UniqueDiffHashes: 1}, 1.0},
		{"yellow-half-trust→0.5", dtv.Fitness{Verdict: dtv.VerdictYellow, PassRate: 0.5, TotalRuns: 5, UniqueDiffHashes: 1}, 0.5},
		{"non-convergent-damped", dtv.Fitness{Verdict: dtv.VerdictGreen, PassRate: 1, TotalRuns: 5, UniqueDiffHashes: 2}, 0.5},
		{"stale-damped", dtv.Fitness{Verdict: dtv.VerdictGreen, PassRate: 1, TotalRuns: 5, UniqueDiffHashes: 1, AnyStale: true}, 0.5},
		{"zero-passrate→0", dtv.Fitness{Verdict: dtv.VerdictGreen, PassRate: 0, TotalRuns: 5, UniqueDiffHashes: 1}, 0},
	}
	for _, tc := range tests {
		if got := prov(tc.f); got != tc.want {
			t.Errorf("%s: prior = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestDTVPriorProvider_WeightAndTrustFloor(t *testing.T) {
	f := dtv.Fitness{Verdict: dtv.VerdictGreen, PassRate: 0.6, TotalRuns: 5, UniqueDiffHashes: 1}
	snap := snapshotOf(map[string]dtv.Fitness{"s": f})

	// Weight scales the prior.
	half := NewDTVPriorProvider(snap, DTVPriorConfig{Weight: 0.5}).Prior("s", "standards")
	if half != 0.3 { // 0.5 * 1 * 0.6 * 1
		t.Errorf("weighted prior = %v, want 0.3", half)
	}
	// TrustFloor above pass_rate zeroes the prior.
	floored := NewDTVPriorProvider(snap, DTVPriorConfig{TrustFloor: 0.7}).Prior("s", "standards")
	if floored != 0 {
		t.Errorf("trust floor should zero a below-floor skill, got %v", floored)
	}
}

func TestSelector_DTVPriorReordersColdStart(t *testing.T) {
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	// lint-fix is DTV-mature (green), refactor unknown — lint-fix must win at n=0
	// even though refactor is first in the allow-set (greedy would pick refactor).
	snap := snapshotOf(map[string]dtv.Fitness{
		"lint-fix": {Verdict: dtv.VerdictGreen, PassRate: 1, TotalRuns: 6, UniqueDiffHashes: 1},
	})
	sel := NewSelectorWithConfig(SelectorConfig{
		Resolver:      res,
		Effectiveness: effectiveness.NewMemoryStore(),
		Prior:         NewDTVPriorProvider(snap, DTVPriorConfig{}),
	}).SelectNextSkill(standardsState(), profile)
	if sel.SkillID != "lint-fix" {
		t.Fatalf("DTV-mature skill must win cold start; got %q (%s)", sel.SkillID, sel.Rationale)
	}
}

func TestSelector_IdenticalFitnessReproducesGreedy(t *testing.T) {
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	same := dtv.Fitness{Verdict: dtv.VerdictGreen, PassRate: 1, TotalRuns: 6, UniqueDiffHashes: 1}
	snap := snapshotOf(map[string]dtv.Fitness{"refactor": same, "lint-fix": same})
	greedy := NewSelector(res).SelectNextSkill(standardsState(), profile)
	dtvSel := NewSelectorWithConfig(SelectorConfig{
		Resolver:      res,
		Effectiveness: effectiveness.NewMemoryStore(),
		Prior:         NewDTVPriorProvider(snap, DTVPriorConfig{}),
	}).SelectNextSkill(standardsState(), profile)
	if dtvSel.SkillID != greedy.SkillID {
		t.Fatalf("identical DTV fitness must reproduce greedy order: greedy=%q dtv=%q", greedy.SkillID, dtvSel.SkillID)
	}
}

func TestSelector_AbsentFitnessReproducesP1(t *testing.T) {
	res := standardsResolver("refactor", "lint-fix")
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "lint-fix"}}
	store := effectiveness.NewMemoryStore()
	store.Seed(effectiveness.Stat{SkillID: "lint-fix", Dimension: "standards", ClosedCount: 20, TotalRuns: 5, TotalTokens: 5000})

	p1 := NewSelectorWithConfig(SelectorConfig{Resolver: res, Effectiveness: store, Prior: UniformPrior{Value: coldStartPrior}}).
		SelectNextSkill(standardsState(), profile)
	// DTV prior over an empty snapshot ⇒ every prior 0 ⇒ identical to P1.
	dtvP := NewSelectorWithConfig(SelectorConfig{Resolver: res, Effectiveness: store, Prior: NewDTVPriorProvider(snapshotOf(nil), DTVPriorConfig{})}).
		SelectNextSkill(standardsState(), profile)
	if p1.SkillID != dtvP.SkillID {
		t.Fatalf("absent DTV data must reproduce exact P1 selection: p1=%q dtv=%q", p1.SkillID, dtvP.SkillID)
	}
}

// ── Phase 5: snapshot TTL + fail-open ───────────────────────────────────────

type countingProvider struct {
	calls int
	fits  map[string]dtv.Fitness
	err   error
}

func (c *countingProvider) Fitness(_ context.Context, skillID string) (dtv.Fitness, error) {
	c.calls++
	if c.err != nil {
		return dtv.Fitness{SkillID: skillID}, c.err
	}
	return c.fits[skillID], nil
}

func newOrchestratorForSnapshot(p dtv.SkillFitnessProvider) *ExecutionOrchestrator {
	return &ExecutionOrchestrator{
		fitnessProvider: p,
		fitnessSnaps:    make(map[string]*taskFitness),
		degradedLogged:  make(map[string]bool),
	}
}

func TestFitnessSnapshot_TTLRefresh(t *testing.T) {
	prov := &countingProvider{fits: map[string]dtv.Fitness{"a": {Verdict: dtv.VerdictGreen}}}
	o := newOrchestratorForSnapshot(prov)
	profile := &AutoSteerProfile{AllowedSkills: []string{"a"}, DTV: &DTVObjective{RefreshIters: 3}}
	state := &ProfileExecutionState{TaskID: "t1"}

	// Iter 0 fetch (1 skill ⇒ 1 call); iters 1,2 reuse cache; iter 3 refreshes.
	for _, iter := range []int{0, 1, 2} {
		state.Iteration = iter
		o.fitnessSnapshot(state, profile)
	}
	if prov.calls != 1 {
		t.Fatalf("within TTL the provider must be hit once, got %d", prov.calls)
	}
	state.Iteration = 3
	o.fitnessSnapshot(state, profile)
	if prov.calls != 2 {
		t.Fatalf("crossing the TTL must refresh once more, got %d calls", prov.calls)
	}
}

func TestFitnessSnapshot_FailsOpenWhenDegraded(t *testing.T) {
	prov := &countingProvider{err: errors.New("dtv down")}
	o := newOrchestratorForSnapshot(prov)
	profile := &AutoSteerProfile{AllowedSkills: []string{"a", "b"}}
	state := &ProfileExecutionState{TaskID: "t1"}

	snap, degraded := o.fitnessSnapshot(state, profile)
	if !degraded {
		t.Fatal("a provider error must mark the snapshot degraded")
	}
	// Fail open: every skill resolves UNKNOWN ⇒ gate allows, prior 0.
	if snap.Get("a").Verdict != dtv.VerdictUnknown {
		t.Fatalf("degraded snapshot must yield UNKNOWN fitness, got %v", snap.Get("a").Verdict)
	}
	if NewDTVEligibilityFilter(snap).Allow("a", "standards") != true {
		t.Fatal("degraded gate must allow all (fail open)")
	}
	if NewDTVPriorProvider(snap, DTVPriorConfig{}).Prior("a", "standards") != 0 {
		t.Fatal("degraded prior must be 0 (uniform P1)")
	}
}

func TestDTVSeams_DisabledWithoutProvider(t *testing.T) {
	o := newOrchestratorForSnapshot(nil)
	prior, filter, info := o.dtvSeams(&ProfileExecutionState{TaskID: "t1"}, &AutoSteerProfile{})
	if info.active {
		t.Fatal("no provider ⇒ DTV inactive")
	}
	if _, ok := filter.(AllowAllFilter); !ok {
		t.Fatalf("expected AllowAllFilter, got %T", filter)
	}
	if up, ok := prior.(UniformPrior); !ok || up.Value != coldStartPrior {
		t.Fatalf("expected UniformPrior{%v}, got %#v", coldStartPrior, prior)
	}
}

// ── Phase 5: decision-trace DTV annotation ──────────────────────────────────

func TestAnnotateDTVTrace_PopulatesProvenance(t *testing.T) {
	snap := snapshotOf(map[string]dtv.Fitness{
		"lint-fix": {Verdict: dtv.VerdictGreen, PassRate: 1, TotalRuns: 5, UniqueDiffHashes: 1},
		"refactor": {Verdict: dtv.VerdictRed},
	})
	prior := NewDTVPriorProvider(snap, DTVPriorConfig{})
	sel := Selection{SkillID: "lint-fix", Dimension: dimensions.Dimension("standards"), ExcludedSkills: []string{"refactor"}}
	info := dtvSelectionInfo{active: true, snapshot: snap, prior: prior}

	var entry DecisionTraceEntry
	annotateDTVTrace(&entry, sel, info)

	if entry.DTVVerdict != "green" {
		t.Errorf("DTVVerdict = %q, want green", entry.DTVVerdict)
	}
	if entry.DTVPrior != 1.0 {
		t.Errorf("DTVPrior = %v, want 1.0", entry.DTVPrior)
	}
	if entry.DTVExcluded["refactor"] != "dtv:red" {
		t.Errorf("DTVExcluded[refactor] = %q, want dtv:red", entry.DTVExcluded["refactor"])
	}
}

func TestAnnotateDTVTrace_NoopWhenInactive(t *testing.T) {
	var entry DecisionTraceEntry
	annotateDTVTrace(&entry, Selection{SkillID: "x"}, dtvSelectionInfo{active: false})
	if entry.DTVVerdict != "" || entry.DTVPrior != 0 || entry.DTVExcluded != nil {
		t.Fatalf("inactive DTV must leave the trace untouched: %+v", entry)
	}
}
