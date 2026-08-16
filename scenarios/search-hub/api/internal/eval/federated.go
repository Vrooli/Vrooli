package eval

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	internalrouting "search-hub/internal/routing"

	"github.com/vrooli/api-core/schedule"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// QueryClient is the consumer-owned seam for the public federated query path.
// The eval domain observes the routing contract and never imports Router.
type QueryClient interface {
	Query(context.Context, *routingv1.QueryRequest) (*routingv1.QueryResponse, error)
}

// RoutabilityReader supplies the router's automatic-eligibility view to the
// federated evaluator. A provider that is deliberately withheld by freshness,
// lifecycle, or evidence policy is not a selector miss: the evaluator must
// record that case as unavailable so quality metrics describe only the leaves
// the automatic router was allowed to reach.
type RoutabilityReader interface {
	Status(context.Context) (*routingv1.StatusResponse, error)
}

// SubstrateSnapshotter is an optional extension implemented by the production
// routing client. It captures the live serving substrate once per federated
// run, while QueryResponse contributes the selector leg for the actual cases.
// Keeping this optional preserves the small QueryClient seam used by unit
// tests and by offline callers.
type SubstrateSnapshotter interface {
	Snapshot(context.Context) *evalv1.ConfigSnapshot
}

const defaultFederatedMargin = 0.01

const (
	// Federated grading is an unattended evidence run, not the interactive
	// query path. Give automatic classification, provider fan-out, and the
	// provider's own bounded search enough time to finish so infrastructure
	// latency is recorded as unavailable only after the router's 25s query
	// budget, rather than being mistaken for routing quality.
	defaultFederatedCaseTimeout = 30 * time.Second
	federatedEvalConcurrency    = 8
)

// FederatedRunner evaluates the same registered corpus through QueryService,
// recording routing, ranking, margin, and honest degradation evidence.
type FederatedRunner struct {
	resolver    ProviderResolver
	query       QueryClient
	clock       schedule.Clock
	newID       func() string
	routability RoutabilityReader
}

type federatedCaseObservation struct {
	result         *evalv1.CaseResult
	latency        int64
	degraded       bool
	degradedReason string
	selectorLeg    string
	rerankerLeg    string
}

func NewFederatedRunner(resolver ProviderResolver, query QueryClient, clk schedule.Clock, newID func() string) *FederatedRunner {
	return NewFederatedRunnerWithRoutability(resolver, query, clk, newID, nil)
}

// NewFederatedRunnerWithRoutability preserves the small constructor used by
// offline callers while wiring the production runner to the same eligibility
// status surface that governs automatic routing.
func NewFederatedRunnerWithRoutability(resolver ProviderResolver, query QueryClient, clk schedule.Clock, newID func() string, routability RoutabilityReader) *FederatedRunner {
	if newID == nil {
		newID = func() string { return "federated-run" }
	}
	return &FederatedRunner{resolver: resolver, query: query, clock: clk, newID: newID, routability: routability}
}

func (r *FederatedRunner) Run(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32) (*evalv1.EvalRun, error) {
	return r.RunWithStrategy(ctx, suite, tag, limit, "")
}

// RunWithStrategy evaluates the federated suite through one registered
// retrieval strategy. Empty strategyName preserves the active-strategy path;
// non-empty names are carried on every query so the router and stored run tag
// describe the same experiment arm.
func (r *FederatedRunner) RunWithStrategy(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32, strategyName string) (*evalv1.EvalRun, error) {
	return r.runWithStrategy(ctx, suite, tag, limit, strategyName, nil)
}

// RunWithStrategyAndExclusions evaluates an arm against an eligibility
// snapshot captured by the strategy-comparison coordinator. Reusing one
// snapshot across arms prevents query-triggered demotions from changing the
// denominator mid-comparison and making otherwise paired evidence
// incomparable.
func (r *FederatedRunner) RunWithStrategyAndExclusions(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32, strategyName string, exclusions map[string]string) (*evalv1.EvalRun, error) {
	if exclusions == nil {
		exclusions = map[string]string{}
	}
	return r.runWithStrategy(ctx, suite, tag, limit, strategyName, exclusions)
}

func (r *FederatedRunner) runWithStrategy(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32, strategyName string, exclusions map[string]string) (*evalv1.EvalRun, error) {
	if suite == nil {
		return nil, fmt.Errorf("nil suite")
	}
	if suite.GetSuiteId() != RouterSuiteID {
		if _, err := r.resolver.Get(ctx, suite.GetProviderId()); err != nil {
			return nil, fmt.Errorf("resolve provider %q: %w", suite.GetProviderId(), err)
		}
	}
	if limit <= 0 {
		limit = effectiveLimit(suite, limit)
	}
	if suite.GetSuiteId() == RouterSuiteID {
		ctx = internalrouting.WithRoutingEvaluation(ctx)
	} else {
		ctx = internalrouting.WithBackgroundEvaluationProvider(ctx, suite.GetProviderId())
	}
	cases := suite.GetCases()
	results := make([]*evalv1.CaseResult, len(cases))
	latencies := make([]int64, len(cases))
	observations := make([]federatedCaseObservation, len(cases))
	var unavailable []*evalv1.UnavailableCase
	run := &evalv1.EvalRun{
		RunId:   r.newID(),
		SuiteId: suite.GetSuiteId(),
		Tag:     tag,
		Tier:    "federated",
		Config:  defaultFederatedConfig(),
	}
	if snapshotter, ok := r.query.(SubstrateSnapshotter); ok {
		if snapshot := snapshotter.Snapshot(ctx); snapshot != nil {
			run.Config = snapshot
		}
	}
	if run.Config.GetSelectorLeg() == "" {
		run.Config.SelectorLeg = "unknown"
	}
	if run.Config.GetRerankerLeg() == "" {
		run.Config.RerankerLeg = "unknown"
	}
	if run.Config.GetEmbedModel() == "" {
		run.Config.EmbedModel = "unknown"
	}
	automaticExclusions := exclusions
	if automaticExclusions == nil {
		automaticExclusions = r.automaticExclusions(ctx, suite)
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	workerCount := federatedEvalConcurrency
	if len(cases) < workerCount {
		workerCount = len(cases)
	}
	for worker := 0; worker < workerCount; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				observations[index] = r.runFederatedCase(ctx, suite, cases[index], limit, strategyName, automaticExclusions)
			}
		}()
	}
	for index := range cases {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	for index, observation := range observations {
		results[index] = observation.result
		latencies[index] = observation.latency
		if observation.selectorLeg != "" && run.Config.GetSelectorLeg() == "unknown" {
			run.Config.SelectorLeg = observation.selectorLeg
		}
		if observation.rerankerLeg != "" && run.Config.GetRerankerLeg() == "unknown" {
			run.Config.RerankerLeg = observation.rerankerLeg
			run.Config.RerankEnabled = observation.rerankerLeg != "none"
		}
		if observation.degraded {
			run.Degraded = true
			if run.DegradedReason == "" {
				run.DegradedReason = observation.degradedReason
			}
		}
	}
	for _, result := range results {
		if result == nil {
			continue
		}
		if result.GetOutcome() == "unavailable" {
			unavailable = append(unavailable, &evalv1.UnavailableCase{CaseId: result.GetCaseId(), Reason: result.GetOutcomeReason()})
		}
		if result.GetOutcome() == "error" || result.GetOutcome() == "unavailable" {
			run.Degraded = true
			if run.DegradedReason == "" {
				run.DegradedReason = result.GetOutcomeReason()
			}
		}
	}
	if run.Config.GetSelectorLeg() == "unknown" && suite.GetSuiteId() != RouterSuiteID {
		run.Config.SelectorLeg = "background_provider"
	}
	run.Results = results
	run.Aggregate = aggregate(suite, results, latencies)
	run.UnavailableCases = unavailable
	run.Aggregate.UnavailableCases = int32(len(unavailable))
	run.Aggregate.RoutingPrecision, run.Aggregate.RetrievalRecall = federatedRates(suite, results)
	if run.Aggregate.GradedCases > 0 {
		run.Aggregate.PassRate = float64(run.Aggregate.Met) / float64(run.Aggregate.GradedCases)
	}
	if len(unavailable) > 0 && run.Aggregate.GradedCases == 0 {
		run.UnavailableReason = unavailable[0].GetReason()
	}
	return run, nil
}

func (r *FederatedRunner) automaticExclusions(ctx context.Context, suite *evalv1.EvalSuite) map[string]string {
	if r.routability == nil || suite.GetSuiteId() != RouterSuiteID {
		return nil
	}
	status, err := r.routability.Status(ctx)
	if err != nil || status == nil {
		// A missing status snapshot must not invent eligibility. Preserve the
		// historical grading path and let the run's query evidence stand on its
		// own; the substrate snapshot still records the serving leg.
		return nil
	}
	exclusions := make(map[string]string)
	for _, provider := range status.GetProviders() {
		if provider == nil || provider.GetProviderId() == "" || provider.GetAutomaticEligible() {
			continue
		}
		reason := strings.TrimSpace(provider.GetAutomaticExclusionReason())
		if reason == "" {
			reason = "automatic eligibility policy withheld the provider"
		}
		exclusions[provider.GetProviderId()] = reason
	}
	return exclusions
}

func (r *FederatedRunner) runFederatedCase(ctx context.Context, suite *evalv1.EvalSuite, c *evalv1.EvalCase, limit int32, strategyName string, automaticExclusions map[string]string) federatedCaseObservation {
	start := r.clock.Now()
	caseCtx := ctx
	timeout := caseTimeout(ctx)
	if timeout <= 0 {
		timeout = defaultFederatedCaseTimeout
	}
	caseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	runQuery := r.query.Query
	routingResponse, err := runQuery(caseCtx, &routingv1.QueryRequest{Query: c.GetQuery(), Limit: limit, StrategyName: strategyName})
	latency := r.clock.Now().Sub(start).Milliseconds()
	expectedProvider := c.GetExpectedProviderId()
	if expectedProvider == "" {
		expectedProvider = suite.GetProviderId()
	}
	cr := &evalv1.CaseResult{CaseId: c.GetCaseId(), ExpectedProviderId: expectedProvider}
	if reason, withheld := automaticExclusions[expectedProvider]; withheld {
		cr.Outcome = "unavailable"
		cr.OutcomeReason = fmt.Sprintf("expected provider %q withheld from automatic routing: %s", expectedProvider, reason)
		return federatedCaseObservation{result: cr, latency: latency}
	}
	if err != nil {
		cr.Outcome = "error"
		cr.OutcomeReason = err.Error()
		if unavailableError(err) || ctx.Err() != nil || caseCtx.Err() != nil {
			cr.Outcome = "unavailable"
		}
		return federatedCaseObservation{result: cr, latency: latency}
	}
	cr.ProviderRouted = contains(routingResponse.GetCorporaSearched(), expectedProvider)
	top := routingResponse.GetRanked()
	if len(top) == 0 {
		top = flattenGroups(routingResponse.GetGroups())
	}
	cr.Top = toScoredHits(top)
	cr.ExpectedRank = expectedRank(c, top)
	if len(top) > 0 {
		cr.ObservedTopScore = top[0].GetScore()
		cr.Margin = margin(top)
	}
	floor := c.GetExpectMinMargin()
	if floor <= 0 {
		floor = defaultFederatedMargin
		cr.OutcomeReason = fmt.Sprintf("default margin floor %.2f", floor)
	}
	observation := federatedCaseObservation{
		result:      cr,
		latency:     latency,
		selectorLeg: strings.TrimSpace(routingResponse.GetSelectorLeg()),
		rerankerLeg: strings.TrimSpace(routingResponse.GetRerankerLeg()),
		degraded:    routingResponse.GetDegraded(),
	}
	if observation.degraded {
		if explanation := routingResponse.GetRoutingExplanation(); len(explanation) > 0 {
			observation.degradedReason = explanation[0]
		} else {
			observation.degradedReason = routingResponse.GetRoutingDegradeReason()
		}
	}
	switch {
	case isFederatedNegativeCase(c):
		// Negative cases declare that no specific result is expected. They are
		// therefore a routing-safety constraint, not a positive retrieval
		// target: an unrelated provider answer must not turn an otherwise
		// gradeable federated run into a false positive miss. The provider-direct
		// tier already owns the score-ceiling check; federated recall excludes
		// these cases from its positive denominator below.
		cr.Outcome = "met"
		cr.OutcomeReason = "negative case: no expected result was declared"
	case len(top) == 0:
		cr.Outcome = "no_result"
		cr.OutcomeReason = fmt.Sprintf("expected provider %q returned no results", expectedProvider)
	case !cr.ProviderRouted && cr.ExpectedRank > 0:
		cr.Outcome = "answered_by_sibling"
		cr.OutcomeReason = fmt.Sprintf("expected result answered by a sibling; provider %q was absent from corpora_searched", expectedProvider)
	case !cr.ProviderRouted:
		cr.Outcome = "misrouted"
		cr.OutcomeReason = fmt.Sprintf("non-owner provider answered while expected provider %q was absent from corpora_searched", expectedProvider)
	case cr.ExpectedRank == 0:
		cr.Outcome = "below_expectation"
	case c.GetExpectWithinTopK() > 0 && cr.ExpectedRank > c.GetExpectWithinTopK():
		cr.Outcome = "below_expectation"
	case cr.Margin < floor:
		cr.Outcome = "thin_margin"
	default:
		cr.Outcome = "met"
	}
	return observation
}

func defaultFederatedConfig() *evalv1.ConfigSnapshot {
	return &evalv1.ConfigSnapshot{SelectorLeg: "unknown", RerankerLeg: "unknown", EmbedModel: "unknown"}
}

// isFederatedNegativeCase narrows the provider-direct negative-case contract
// to cases that genuinely have no positive target. A gibberish tag alone must
// not suppress grading when a suite also declares an expected identifier.
func isFederatedNegativeCase(c *evalv1.EvalCase) bool {
	return isNegativeCase(c) && len(c.GetExpectIds()) == 0
}

// federatedRates splits the end-to-end federated verdict without changing the
// historical pass_rate. Routing precision is the fraction of gradeable cases
// where the suite owner appeared in the routed provider set. Retrieval recall
// is the fraction of those owner-routed gradeable cases whose expected id was
// within the case's expect_within_top_k (or present when no K was declared).
// A nil result means the denominator was empty: proto presence preserves the
// distinction between not measured and a measured zero.
func federatedRates(suite *evalv1.EvalSuite, results []*evalv1.CaseResult) (*float64, *float64) {
	withinTopK := make(map[string]int32, len(suite.GetCases()))
	negative := make(map[string]bool, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		withinTopK[c.GetCaseId()] = c.GetExpectWithinTopK()
		negative[c.GetCaseId()] = isFederatedNegativeCase(c)
	}
	var gradeable, routed, positiveRouted, recalled int
	for _, result := range results {
		if !federatedGradeable(result.GetOutcome()) {
			continue
		}
		gradeable++
		if !result.GetProviderRouted() {
			continue
		}
		routed++
		if negative[result.GetCaseId()] {
			continue
		}
		positiveRouted++
		k := withinTopK[result.GetCaseId()]
		if result.GetExpectedRank() > 0 && (k <= 0 || result.GetExpectedRank() <= k) {
			recalled++
		}
	}
	var routingPrecision, retrievalRecall *float64
	if gradeable > 0 {
		value := float64(routed) / float64(gradeable)
		routingPrecision = &value
	}
	if positiveRouted > 0 {
		value := float64(recalled) / float64(positiveRouted)
		retrievalRecall = &value
	}
	return routingPrecision, retrievalRecall
}

func federatedGradeable(outcome string) bool {
	switch outcome {
	case "met", "below_expectation", "above_expectation", "unexpected_hit", "answered_by_sibling", "misrouted", "no_result", "thin_margin":
		return true
	default:
		return false
	}
}

func expectedRank(c *evalv1.EvalCase, hits []*routingv1.SearchHit) int32 {
	for i, hit := range hits {
		for _, want := range c.GetExpectIds() {
			if hit.GetId() == want {
				return int32(i + 1)
			}
		}
	}
	return 0
}

func margin(hits []*routingv1.SearchHit) float64 {
	if len(hits) < 2 || hits[0].GetScore() <= 0 {
		return 0
	}
	return (hits[0].GetScore() - hits[1].GetScore()) / hits[0].GetScore()
}

func flattenGroups(groups []*routingv1.ProviderResultGroup) []*routingv1.SearchHit {
	var out []*routingv1.SearchHit
	for _, group := range groups {
		if group != nil {
			out = append(out, group.GetHits()...)
		}
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
