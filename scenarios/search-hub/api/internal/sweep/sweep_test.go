package sweep

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"

	aisearch "github.com/vrooli/ai-go/search"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// ---------------------------------------------------------------------------
// Fakes (seam-discovery: the core depends only on the small interfaces here).
// ---------------------------------------------------------------------------

type fakeSuites struct{ s *evalv1.EvalSuite }

func (f fakeSuites) GetSuite(context.Context, string) (*evalv1.EvalSuite, error) { return f.s, nil }

type fakeProviders struct {
	d     *registryv1.ProviderDescriptor
	token string
}

func (f fakeProviders) Get(context.Context, string) (*registryv1.ProviderDescriptor, error) {
	return f.d, nil
}
func (f fakeProviders) Token(context.Context, string) (string, error) { return f.token, nil }

type runnerCall struct {
	tag       string
	overrides *aisearch.SearchOverrides
}

type fakeRunner struct {
	calls   []runnerCall
	n       int
	produce func(tag string) (met map[string]bool, agg *evalv1.EvalAggregate)
}

func (f *fakeRunner) Run(_ context.Context, suite *evalv1.EvalSuite, tag string, ov *aisearch.SearchOverrides, _ string, _ int32) (*evalv1.EvalRun, error) {
	f.calls = append(f.calls, runnerCall{tag: tag, overrides: ov})
	f.n++
	met, agg := f.produce(tag)
	return buildRun(fmt.Sprintf("run-%d", f.n), tag, suite, met, agg), nil
}

func buildRun(id, tag string, suite *evalv1.EvalSuite, met map[string]bool, agg *evalv1.EvalAggregate) *evalv1.EvalRun {
	results := make([]*evalv1.CaseResult, 0, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		outcome := "n/a"
		if isPositiveCase(c) {
			if met[c.GetCaseId()] {
				outcome = "met"
			} else {
				outcome = "below_expectation"
			}
		}
		results = append(results, &evalv1.CaseResult{CaseId: c.GetCaseId(), Outcome: outcome})
	}
	if agg == nil {
		agg = &evalv1.EvalAggregate{}
	}
	return &evalv1.EvalRun{RunId: id, SuiteId: suite.GetSuiteId(), Tag: tag, Results: results, Aggregate: agg, Config: &evalv1.ConfigSnapshot{}}
}

type fakeControl struct {
	current     aisearch.TuningConfig
	writes      []aisearch.TuningConfig
	jobs        int
	statusState string // returned by ReindexStatus (default "succeeded")
}

func (c *fakeControl) WriteConfig(_ context.Context, _ *registryv1.ProviderDescriptor, _ string, tuning *registryv1.Tuning, _ bool) (*controlv1.WriteConfigResponse, error) {
	next := tuningFromProto(tuning).WithDefaults()
	changed := c.current.IndexTimeChanged(next)
	c.writes = append(c.writes, next)
	resp := &controlv1.WriteConfigResponse{Written: true, Effective: tuning}
	if changed {
		c.jobs++
		resp.ReindexTriggered = true
		resp.ReindexJobId = fmt.Sprintf("job-%d", c.jobs)
	}
	c.current = next
	return resp, nil
}

func (c *fakeControl) ReindexStatus(_ context.Context, _ *registryv1.ProviderDescriptor, _, jobID string) (*controlv1.ReindexStatusResponse, error) {
	st := c.statusState
	if st == "" {
		st = "succeeded"
	}
	return &controlv1.ReindexStatusResponse{JobId: jobID, State: st}, nil
}

// fakeCache records the descriptor re-upserts the write-back performs to refresh
// the registry's cached tuning.
type fakeCache struct {
	upserts []*registryv1.ProviderDescriptor
	tokens  []string
}

func (c *fakeCache) Upsert(_ context.Context, d *registryv1.ProviderDescriptor, token string) (bool, string, error) {
	c.upserts = append(c.upserts, d)
	c.tokens = append(c.tokens, token)
	return false, token, nil
}

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

// makeSuite builds a suite with n positive cases (p0..pN-1) plus one gibberish.
func makeSuite(n int) *evalv1.EvalSuite {
	cases := make([]*evalv1.EvalCase, 0, n+1)
	for i := 0; i < n; i++ {
		cases = append(cases, &evalv1.EvalCase{CaseId: fmt.Sprintf("p%d", i), Query: fmt.Sprintf("q%d", i), Tags: []string{"strong"}, ExpectIds: []string{fmt.Sprintf("id%d", i)}, ExpectWithinTopK: 5})
	}
	cases = append(cases, &evalv1.EvalCase{CaseId: "g", Query: "zzzz", Tags: []string{"gibberish"}, ExpectNoStrongHit: true, ExpectMaxScore: 0.5})
	return &evalv1.EvalSuite{SuiteId: "s.primary", ProviderId: "p", Cases: cases}
}

func newHarness(t *testing.T, suite *evalv1.EvalSuite, incumbent aisearch.TuningConfig, runner *fakeRunner, control *fakeControl) *Orchestrator {
	return newHarnessWithCache(t, suite, incumbent, runner, control, nil)
}

func newHarnessWithCache(t *testing.T, suite *evalv1.EvalSuite, incumbent aisearch.TuningConfig, runner *fakeRunner, control *fakeControl, cache TuningCache) *Orchestrator {
	t.Helper()
	if control.current == (aisearch.TuningConfig{}) {
		control.current = incumbent.WithDefaults()
	}
	desc := &registryv1.ProviderDescriptor{
		ProviderId:      "p",
		Tuning:          tuningToProto(incumbent),
		ReindexEndpoint: &registryv1.Endpoint{},
		ConfigEndpoint:  &registryv1.Endpoint{},
	}
	return New(Deps{
		Suites:    fakeSuites{s: suite},
		Providers: fakeProviders{d: desc, token: "tok"},
		Runner:    runner,
		Control:   control,
		Cache:     cache,
		Clock:     schedule.System(),
		Sleep:     func(time.Duration) {},
		Rand:      newSeededRand(),
	}, Options{HeldoutFraction: 0.5, MinHeldout: 2, BootstrapIters: 500, GibberishFloor: 0.5, LatencyMultiplier: 0, PollInterval: time.Millisecond})
}

func allSet(ids ...string) map[string]bool {
	m := map[string]bool{}
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func posIDs(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("p%d", i)
	}
	return out
}

func findArm(arms []*evalv1.SweepArm, substr string) *evalv1.SweepArm {
	for _, a := range arms {
		if strings.Contains(a.GetTag(), substr) {
			return a
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSweep_PromotesClearWinner_QueryTimeOnly(t *testing.T) {
	suite := makeSuite(8)
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense}.WithDefaults()
	control := &fakeControl{}
	runner := &fakeRunner{produce: func(tag string) (map[string]bool, *evalv1.EvalAggregate) {
		agg := &evalv1.EvalAggregate{MaxGibberishScore: 0.3}
		// The full rerank-blend arm nails every case; everything else (incumbent +
		// rerank-on/blend-off) misses every case — a clean, fold-independent win.
		if strings.Contains(tag, "rerank_enabled=true,rerank_blend=true") {
			return allSet(posIDs(8)...), agg
		}
		return map[string]bool{}, agg
	}}
	o := newHarness(t, suite, inc, runner, control)

	res, err := o.Run(context.Background(), &evalv1.SweepRequest{SuiteId: "s.primary", QueryTimeOnly: true, Apply: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.GetWinnerTag() == "" || !strings.Contains(res.GetWinnerTag(), "rerank_blend=true") {
		t.Fatalf("expected the rerank-blend arm to win, got %q\n%s", res.GetWinnerTag(), res.GetRecommendation())
	}
	if !res.GetPromoted() {
		t.Fatalf("apply=true with a cleared winner must promote; recommendation: %s", res.GetRecommendation())
	}
	if len(control.writes) != 1 {
		t.Fatalf("write-back should persist exactly the winner once, got %d writes", len(control.writes))
	}
	if !control.writes[0].RerankBlend {
		t.Fatalf("write-back persisted the wrong tuning: %+v", control.writes[0])
	}
	// query_time_only: incumbent + 2 query-time arms = 3 runs, no index reindex.
	if len(runner.calls) != 3 {
		t.Fatalf("expected 3 runs (incumbent + 2 query arms), got %d: %v", len(runner.calls), runnerTags(runner))
	}
	if runner.calls[0].overrides != nil {
		t.Fatalf("incumbent arm must run with NO overrides (baseline)")
	}
	if res.GetStats().GetCiLow() <= 0 {
		t.Fatalf("a promoted winner must have CI low > 0, got %v", res.GetStats().GetCiLow())
	}
}

func TestSweep_WriteBack_RefreshesTuningCache(t *testing.T) {
	suite := makeSuite(8)
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense}.WithDefaults()
	control := &fakeControl{}
	cache := &fakeCache{}
	runner := &fakeRunner{produce: func(tag string) (map[string]bool, *evalv1.EvalAggregate) {
		agg := &evalv1.EvalAggregate{MaxGibberishScore: 0.3}
		if strings.Contains(tag, "rerank_enabled=true,rerank_blend=true") {
			return allSet(posIDs(8)...), agg
		}
		return map[string]bool{}, agg
	}}
	o := newHarnessWithCache(t, suite, inc, runner, control, cache)

	res, err := o.Run(context.Background(), &evalv1.SweepRequest{SuiteId: "s.primary", QueryTimeOnly: true, Apply: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.GetPromoted() {
		t.Fatalf("expected promotion; recommendation: %s", res.GetRecommendation())
	}
	// The registry cache was refreshed exactly once with the winning tuning, under
	// the provider's control token — no reboot needed for ListProviders to be fresh.
	if len(cache.upserts) != 1 {
		t.Fatalf("expected one cache refresh, got %d", len(cache.upserts))
	}
	if !cache.upserts[0].GetTuning().GetRerankBlend() {
		t.Fatalf("cache refreshed with the wrong tuning: %+v", cache.upserts[0].GetTuning())
	}
	if cache.tokens[0] != "tok" {
		t.Fatalf("cache refresh must present the control token, got %q", cache.tokens[0])
	}
}

func TestSweep_ApplyFalse_NeverWrites(t *testing.T) {
	suite := makeSuite(8)
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense}.WithDefaults()
	control := &fakeControl{}
	runner := &fakeRunner{produce: func(tag string) (map[string]bool, *evalv1.EvalAggregate) {
		if strings.Contains(tag, "rerank_blend=true") {
			return allSet(posIDs(8)...), &evalv1.EvalAggregate{}
		}
		return map[string]bool{}, &evalv1.EvalAggregate{}
	}}
	o := newHarness(t, suite, inc, runner, control)

	res, err := o.Run(context.Background(), &evalv1.SweepRequest{SuiteId: "s.primary", QueryTimeOnly: true, Apply: false})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.GetWinnerTag() == "" {
		t.Fatalf("a clear winner should still be IDENTIFIED in preview mode")
	}
	if res.GetPromoted() {
		t.Fatalf("apply=false must never promote")
	}
	if len(control.writes) != 0 {
		t.Fatalf("apply=false must never write, got %d writes", len(control.writes))
	}
}

func TestSweep_WithinNoise_NoPromotion(t *testing.T) {
	suite := makeSuite(8)
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense}.WithDefaults()
	// One tuning-fold case improves → beats incumbent on the fold but the paired
	// CI overlaps 0 (guard #1 blocks).
	tuning, _ := splitCases(posIDs(8), nil, 0.5)
	improve := tuning[0]
	control := &fakeControl{}
	runner := &fakeRunner{produce: func(tag string) (map[string]bool, *evalv1.EvalAggregate) {
		if strings.Contains(tag, "incumbent") {
			return map[string]bool{}, &evalv1.EvalAggregate{} // incumbent: all miss
		}
		return allSet(improve), &evalv1.EvalAggregate{} // every arm: exactly one fold case
	}}
	o := newHarness(t, suite, inc, runner, control)

	res, err := o.Run(context.Background(), &evalv1.SweepRequest{SuiteId: "s.primary", QueryTimeOnly: true, Apply: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.GetWinnerTag() != "" {
		t.Fatalf("within-noise improvement must NOT promote, got winner %q\n%s", res.GetWinnerTag(), res.GetRecommendation())
	}
	if res.GetPromoted() || len(control.writes) != 0 {
		t.Fatalf("no winner → no write-back")
	}
	if !strings.Contains(res.GetRecommendation(), "noise") {
		t.Fatalf("verdict should explain the noise band, got: %s", res.GetRecommendation())
	}
}

func TestSweep_ConstraintViolation_BlocksPromotion(t *testing.T) {
	suite := makeSuite(8)
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense}.WithDefaults()
	control := &fakeControl{}
	runner := &fakeRunner{produce: func(tag string) (map[string]bool, *evalv1.EvalAggregate) {
		if strings.Contains(tag, "rerank_blend=true") {
			// Higher recall but leaks junk far above the ceiling → infeasible.
			return allSet(posIDs(8)...), &evalv1.EvalAggregate{MaxGibberishScore: 0.95}
		}
		return allSet("p0", "p1", "p2", "p3"), &evalv1.EvalAggregate{MaxGibberishScore: 0.2}
	}}
	o := newHarness(t, suite, inc, runner, control)

	res, err := o.Run(context.Background(), &evalv1.SweepRequest{SuiteId: "s.primary", QueryTimeOnly: true, Apply: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.GetWinnerTag() != "" {
		t.Fatalf("a constraint-violating arm must not win, got %q", res.GetWinnerTag())
	}
	arm := findArm(res.GetArms(), "rerank_blend=true")
	if arm == nil || arm.GetFeasible() {
		t.Fatalf("the high-gibberish arm must be marked infeasible: %+v", arm)
	}
	if len(control.writes) != 0 {
		t.Fatalf("no promotion → no write-back")
	}
}

func TestSweep_HeldOutRegression_Blocks(t *testing.T) {
	suite := makeSuite(8)
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense}.WithDefaults()
	tuning, heldout := splitCases(posIDs(8), nil, 0.5)
	control := &fakeControl{}
	runner := &fakeRunner{produce: func(tag string) (map[string]bool, *evalv1.EvalAggregate) {
		if strings.Contains(tag, "incumbent") {
			return allSet(heldout...), &evalv1.EvalAggregate{} // wins held-out, loses tuning
		}
		return allSet(tuning...), &evalv1.EvalAggregate{} // wins tuning, regresses held-out
	}}
	o := newHarness(t, suite, inc, runner, control)

	res, err := o.Run(context.Background(), &evalv1.SweepRequest{SuiteId: "s.primary", QueryTimeOnly: true, Apply: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.GetWinnerTag() != "" {
		t.Fatalf("a held-out regression must block promotion, got %q\n%s", res.GetWinnerTag(), res.GetRecommendation())
	}
	if !strings.Contains(res.GetRecommendation(), "held-out") {
		t.Fatalf("verdict should cite the held-out failure, got: %s", res.GetRecommendation())
	}
}

func TestSweep_IndexTimeTier_VisitsArmsPollsAndRestores(t *testing.T) {
	suite := makeSuite(8)
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense, EmbedTaskPrefix: false}.WithDefaults()
	control := &fakeControl{}
	// All arms tie (no winner) so the test isolates the index-time orchestration.
	runner := &fakeRunner{produce: func(string) (map[string]bool, *evalv1.EvalAggregate) {
		return allSet("p0", "p1", "p2", "p3"), &evalv1.EvalAggregate{}
	}}
	o := newHarness(t, suite, inc, runner, control)

	res, err := o.Run(context.Background(), &evalv1.SweepRequest{SuiteId: "s.primary", Apply: false})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Coordinate-ascent index-time arms = {hybrid, prefix=true} = 2; dropped = 1.
	if res.GetStats().GetIndexTimeArms() != 2 {
		t.Fatalf("index_time_arms = %d, want 2", res.GetStats().GetIndexTimeArms())
	}
	if res.GetStats().GetDroppedIndexInteractions() != 1 {
		t.Fatalf("dropped_index_interactions = %d, want 1", res.GetStats().GetDroppedIndexInteractions())
	}
	// Writes: 2 index candidates + 1 restore-to-incumbent.
	if len(control.writes) != 3 {
		t.Fatalf("expected 3 config writes (2 arms + restore), got %d: %+v", len(control.writes), control.writes)
	}
	if control.writes[len(control.writes)-1].IndexTimeChanged(inc) {
		t.Fatalf("the final write must restore the incumbent index-time config, got %+v", control.writes[2])
	}
	// Runs: incumbent + 2 query arms + 2 index arms = 5.
	if len(runner.calls) != 5 {
		t.Fatalf("expected 5 runs, got %d: %v", len(runner.calls), runnerTags(runner))
	}
}

func TestSweep_IndexReindexFailure_DegradesArmNotSweep(t *testing.T) {
	suite := makeSuite(8)
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense}.WithDefaults()
	control := &fakeControl{statusState: "failed"}
	runner := &fakeRunner{produce: func(string) (map[string]bool, *evalv1.EvalAggregate) {
		return allSet("p0", "p1"), &evalv1.EvalAggregate{}
	}}
	o := newHarness(t, suite, inc, runner, control)

	res, err := o.Run(context.Background(), &evalv1.SweepRequest{SuiteId: "s.primary"})
	if err != nil {
		t.Fatalf("a failed reindex must not fail the whole sweep: %v", err)
	}
	it := findArm(res.GetArms(), "index_time")
	if it == nil {
		t.Fatalf("index-time arm should still appear in the ranked table")
	}
	if it.GetFeasible() || !strings.Contains(it.GetNote(), "failed") {
		t.Fatalf("a reindex-failed arm must be infeasible with a reason, got %+v", it)
	}
}

func TestSweep_NoPositiveCases_Errors(t *testing.T) {
	suite := &evalv1.EvalSuite{SuiteId: "s", ProviderId: "p", Cases: []*evalv1.EvalCase{
		{CaseId: "g", Tags: []string{"gibberish"}, ExpectNoStrongHit: true},
	}}
	inc := aisearch.TuningConfig{Engine: aisearch.EngineDense}.WithDefaults()
	o := newHarness(t, suite, inc, &fakeRunner{produce: func(string) (map[string]bool, *evalv1.EvalAggregate) {
		return nil, &evalv1.EvalAggregate{}
	}}, &fakeControl{})
	if _, err := o.Run(context.Background(), &evalv1.SweepRequest{SuiteId: "s"}); err == nil {
		t.Fatalf("a suite with no positive cases must error (nothing to optimize)")
	}
}

// newSeededRand gives every test a deterministic bootstrap resampler so a
// promotion verdict is reproducible.
func newSeededRand() *rand.Rand { return rand.New(rand.NewSource(20260608)) }

func runnerTags(r *fakeRunner) []string {
	out := make([]string, len(r.calls))
	for i, c := range r.calls {
		out[i] = c.tag
	}
	return out
}
