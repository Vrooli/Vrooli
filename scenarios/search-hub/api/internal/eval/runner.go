package eval

import (
	"context"
	"fmt"
	"sort"

	"search-hub/internal/clock"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// ProviderResolver is the registry read seam the runner depends on: it turns a
// suite's provider_id into the live ProviderDescriptor whose Endpoint +
// ResultMapping the runner reuses. The SQLite registry Store satisfies it in
// production; tests inject a fake. Declared at the consumer (seam-discovery) so
// the runner never imports a concrete registry store.
type ProviderResolver interface {
	Get(ctx context.Context, providerID string) (*registryv1.ProviderDescriptor, error)
}

// SearchCallOptions carries the optional per-call query-time overrides and the
// control token one provider Search applies. The ZERO value is the baseline path
// (no overrides, no token, public search) the eval runner uses today; the sweep
// (Phase 6) fills it per arm — Overrides with the arm's factor values and
// ControlToken with the provider's token (looked up via registry Store.Token).
// Defined at the consumer (seam-discovery) so a fake never depends on transport.
type SearchCallOptions struct {
	// Scope is the optional case-level scope string from the shared corpus
	// contract: ""|"global"|"scenario:<id>"|"path:<prefix>".
	Scope string
	// Overrides, when non-nil and non-zero, are forwarded to the provider as the
	// query-time override header; the provider honors them only past its own
	// token + experiment-flag gate.
	Overrides *aisearch.SearchOverrides
	// ControlToken is presented alongside the overrides so the provider's gate
	// can authenticate them. Ignored when Overrides is nil/zero.
	ControlToken string
}

// ProviderClient calls one provider's registered endpoint and returns mapped,
// score-normalized hits, plus a best-effort config snapshot. This is the seam
// that makes the runner testable without the network: production wires an HTTP
// client (handlers/eval) that resolves the base URL and reuses
// providers.MapResults; tests inject a fake returning canned hits.
type ProviderClient interface {
	// Search calls the provider for one query and returns up to `limit` hits in
	// rank order (scores already normalized to [0,1] by the adapter). opts carries
	// optional query-time overrides + control token (zero value = baseline call).
	Search(ctx context.Context, d *registryv1.ProviderDescriptor, query string, limit int32, opts SearchCallOptions) ([]*routingv1.SearchHit, error)
	// Snapshot probes the provider's status endpoint (if any) for the config
	// that affects results. Best-effort: it never errors — an unreachable or
	// status-less provider yields a mostly-empty snapshot (honest).
	Snapshot(ctx context.Context, d *registryv1.ProviderDescriptor) *evalv1.ConfigSnapshot
}

// Runner executes a suite against its provider and builds an immutable EvalRun.
// It holds no store: the handler persists the returned run (keeps the runner
// pure and unit-testable).
type Runner struct {
	resolver ProviderResolver
	client   ProviderClient
	clock    clock.Clock
	newID    func() string
}

// NewRunner constructs a Runner. newID supplies run_ids (uuid in production, a
// deterministic counter in tests); when nil it falls back to a timestamp-based
// id so production wiring can omit it.
func NewRunner(resolver ProviderResolver, client ProviderClient, clk clock.Clock, newID func() string) *Runner {
	if newID == nil {
		var n int
		newID = func() string { n++; return fmt.Sprintf("run-%d", n) }
	}
	return &Runner{resolver: resolver, client: client, clock: clk, newID: newID}
}

// Run executes every case in suite against the suite's provider and returns a
// tagged, self-describing EvalRun. It returns an error only when the provider
// cannot be resolved (the suite references an unregistered provider). An
// individual provider-call failure is retained as an "error" case and marks
// the run degraded, so transport failure cannot masquerade as an ungraded
// or clean quality result.
//
// Run is the BASELINE path (no overrides): it evaluates the provider's live
// configuration. The sweep (Phase 6) re-runs the same suite through RunWith,
// passing per-arm query-time overrides + the provider's control token.
func (r *Runner) Run(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32) (*evalv1.EvalRun, error) {
	return r.RunWith(ctx, suite, tag, limit, SearchCallOptions{})
}

// RunWith is Run with explicit per-call SearchCallOptions: the sweep passes an
// arm's query-time Overrides + ControlToken so each case is searched under that
// arm's configuration (the provider honors the overrides only past its own
// token + experiment-flag gate; a zero opts is exactly the baseline Run path).
// The returned run is identical in shape to Run's — the overrides change only
// which configuration the provider serves, not how the result is labeled.
func (r *Runner) RunWith(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32, opts SearchCallOptions) (*evalv1.EvalRun, error) {
	if suite == nil {
		return nil, fmt.Errorf("nil suite")
	}
	desc, err := r.resolver.Get(ctx, suite.GetProviderId())
	if err != nil {
		return nil, fmt.Errorf("resolve provider %q: %w", suite.GetProviderId(), err)
	}

	effLimit := effectiveLimit(suite, limit)
	snapshot := r.client.Snapshot(ctx, desc)

	results := make([]*evalv1.CaseResult, 0, len(suite.GetCases()))
	latencies := make([]int64, 0, len(suite.GetCases()))
	var firstFailure string
	for _, c := range suite.GetCases() {
		start := r.clock.Now()
		caseOpts := opts
		caseOpts.Scope = c.GetScope()
		hits, searchErr := r.client.Search(ctx, desc, c.GetQuery(), effLimit, caseOpts)
		latencies = append(latencies, r.clock.Now().Sub(start).Milliseconds())

		top := toScoredHits(hits)
		cr := &evalv1.CaseResult{CaseId: c.GetCaseId(), Top: top}
		if searchErr != nil {
			cr.Outcome = "error"
			cr.OutcomeReason = searchErr.Error()
			if firstFailure == "" {
				firstFailure = searchErr.Error()
			}
			results = append(results, cr)
			continue
		}
		cr.Outcome, cr.ExpectedRank, cr.ObservedTopScore = labelCase(c, top, effLimit)
		results = append(results, cr)
	}

	run := &evalv1.EvalRun{
		RunId:     r.newID(),
		SuiteId:   suite.GetSuiteId(),
		Tag:       tag,
		CreatedAt: r.clock.Now().UTC().Format(timeFormat),
		Config:    snapshot,
		Results:   results,
		Aggregate: aggregate(suite, results, latencies),
		Tier:      "provider_direct",
	}
	if firstFailure != "" {
		run.Degraded = true
		run.DegradedReason = firstFailure
	}
	if run.GetAggregate().GetGradedCases() == 0 {
		run.Degraded = true
		if run.DegradedReason == "" {
			run.DegradedReason = "run produced zero graded cases"
		}
	}
	return run, nil
}

// effectiveLimit is the per-case fetch depth: the explicit request limit when
// set, else the deepest expect_within_top_k across cases, floored at 10 so the
// score-band and gibberish checks always see a meaningful pool.
func effectiveLimit(suite *evalv1.EvalSuite, limit int32) int32 {
	if limit > 0 {
		return limit
	}
	maxK := int32(0)
	for _, c := range suite.GetCases() {
		if c.GetExpectWithinTopK() > maxK {
			maxK = c.GetExpectWithinTopK()
		}
	}
	if maxK < 10 {
		return 10
	}
	return maxK
}

func toScoredHits(hits []*routingv1.SearchHit) []*evalv1.ScoredHit {
	out := make([]*evalv1.ScoredHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, &evalv1.ScoredHit{Id: h.GetId(), Title: h.GetTitle(), Score: h.GetScore()})
	}
	return out
}

// labelCase applies the soft outcome contract (plan §7.3). It never fails a
// build — the label drives UI coloring and the aggregate only.
//
//   - gibberish case (expect_no_strong_hit): "met" if no hit exceeds
//     expect_max_score; else "unexpected_hit".
//   - positive case with no expectations at all: "n/a" (informational).
//   - positive case: "met" when the expected id (if any) lands within
//     expect_within_top_k AND the observed top score is within the [min,max]
//     band (for whichever bounds are set); "below_expectation" when the id is
//     missing/too low or the score is under min; "above_expectation" when the
//     score exceeds a set max.
func labelCase(c *evalv1.EvalCase, top []*evalv1.ScoredHit, limit int32) (outcome string, expectedRank int32, observedTop float64) {
	if len(top) > 0 {
		observedTop = top[0].GetScore()
	}
	results := scoredToSearchResults(top)
	tc := protoCaseToSearchCase(c)
	expectedRank = int32(aisearch.ExpectedRank(results, tc.ExpectIDs))
	policy := aisearch.DefaultScoringPolicy
	if limit > 0 {
		policy.GateK = int(limit)
		policy.DeepK = int(limit)
	}
	switch aisearch.GradeCase(results, tc, policy) {
	case aisearch.CaseOutcomeHit, aisearch.CaseOutcomeJunkRejected:
		return "met", expectedRank, observedTop
	case aisearch.CaseOutcomeMiss:
		return "below_expectation", expectedRank, observedTop
	case aisearch.CaseOutcomeJunkLeaked:
		return "unexpected_hit", expectedRank, observedTop
	default:
		return "n/a", expectedRank, observedTop
	}
}

func protoCaseToSearchCase(c *evalv1.EvalCase) aisearch.TestCase {
	return aisearch.TestCase{
		ID:                c.GetCaseId(),
		Query:             c.GetQuery(),
		Scope:             c.GetScope(),
		Tags:              append([]string(nil), c.GetTags()...),
		ExpectIDs:         append([]string(nil), c.GetExpectIds()...),
		ExpectWithinTopK:  int(c.GetExpectWithinTopK()),
		ExpectMinScore:    c.GetExpectMinScore(),
		ExpectMaxScore:    c.GetExpectMaxScore(),
		ExpectNoStrongHit: c.GetExpectNoStrongHit(),
		Note:              c.GetNote(),
	}
}

func scoredToSearchResults(top []*evalv1.ScoredHit) []aisearch.SearchResult {
	out := make([]aisearch.SearchResult, 0, len(top))
	for _, h := range top {
		out = append(out, aisearch.SearchResult{ID: h.GetId(), Score: h.GetScore()})
	}
	return out
}

// aggregate rolls up per-case outcomes for the trend/compare views. It is a
// descriptive summary, not a verdict.
func aggregate(suite *evalv1.EvalSuite, results []*evalv1.CaseResult, latencies []int64) *evalv1.EvalAggregate {
	tagsByCase := make(map[string][]string, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		tagsByCase[c.GetCaseId()] = c.GetTags()
	}

	agg := &evalv1.EvalAggregate{Cases: int32(len(results))}
	var strongSum float64
	var strongN int
	for _, cr := range results {
		switch cr.GetOutcome() {
		case "met":
			agg.Met++
			agg.GradedCases++
		case "below_expectation":
			agg.Below++
			agg.GradedCases++
		case "above_expectation", "unexpected_hit", "answered_by_sibling", "misrouted", "thin_margin":
			// These are evaluated outcomes even though they are not counted as
			// successful or below-floor positive cases.
			agg.GradedCases++
		}
		tags := tagsByCase[cr.GetCaseId()]
		if hasTag(tags, "strong") {
			strongSum += cr.GetObservedTopScore()
			strongN++
		}
		if hasTag(tags, "gibberish") && cr.GetObservedTopScore() > agg.MaxGibberishScore {
			agg.MaxGibberishScore = cr.GetObservedTopScore()
		}
	}
	if strongN > 0 {
		agg.MeanStrongTop1 = strongSum / float64(strongN)
	}
	agg.LatencyP95Ms = int32(p95(latencies))
	return agg
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

// p95 returns the 95th-percentile latency (nearest-rank). Small-sample friendly:
// with few cases it effectively returns the slowest.
func p95(latencies []int64) int64 {
	if len(latencies) == 0 {
		return 0
	}
	sorted := append([]int64(nil), latencies...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := (95*len(sorted) + 99) / 100 // ceil(0.95*n)
	if idx > len(sorted) {
		idx = len(sorted)
	}
	return sorted[idx-1]
}
