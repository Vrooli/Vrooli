package eval

import (
	"context"
	"fmt"
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
	resolver ProviderResolver
	query    QueryClient
	clock    schedule.Clock
	newID    func() string
}

func NewFederatedRunner(resolver ProviderResolver, query QueryClient, clk schedule.Clock, newID func() string) *FederatedRunner {
	if newID == nil {
		newID = func() string { return "federated-run" }
	}
	return &FederatedRunner{resolver: resolver, query: query, clock: clk, newID: newID}
}

func (r *FederatedRunner) Run(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32) (*evalv1.EvalRun, error) {
	return r.RunWithStrategy(ctx, suite, tag, limit, "")
}

// RunWithStrategy evaluates the federated suite through one registered
// retrieval strategy. Empty strategyName preserves the active-strategy path;
// non-empty names are carried on every query so the router and stored run tag
// describe the same experiment arm.
func (r *FederatedRunner) RunWithStrategy(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32, strategyName string) (*evalv1.EvalRun, error) {
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
	var unavailable []*evalv1.UnavailableCase
	run := &evalv1.EvalRun{RunId: r.newID(), SuiteId: suite.GetSuiteId(), Tag: tag, Tier: "federated"}
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
				results[index], latencies[index] = r.runFederatedCase(ctx, suite, cases[index], limit, strategyName)
			}
		}()
	}
	for index := range cases {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	for _, result := range results {
		if result == nil {
			continue
		}
		if result.GetOutcome() == "unavailable" {
			unavailable = append(unavailable, &evalv1.UnavailableCase{CaseId: result.GetCaseId(), Reason: result.GetOutcomeReason()})
		}
		if result.GetOutcome() == "error" || result.GetOutcome() == "unavailable" || result.GetOutcome() == "degraded" {
			run.Degraded = true
			if run.DegradedReason == "" {
				run.DegradedReason = result.GetOutcomeReason()
			}
		}
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

func (r *FederatedRunner) runFederatedCase(ctx context.Context, suite *evalv1.EvalSuite, c *evalv1.EvalCase, limit int32, strategyName string) (*evalv1.CaseResult, int64) {
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
	if err != nil {
		cr.Outcome = "error"
		cr.OutcomeReason = err.Error()
		if unavailableError(err) || ctx.Err() != nil || caseCtx.Err() != nil {
			cr.Outcome = "unavailable"
		}
		return cr, latency
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
	switch {
	case routingResponse.GetDegraded():
		cr.Outcome = "degraded"
		if len(routingResponse.GetRoutingExplanation()) > 0 {
			cr.OutcomeReason = routingResponse.GetRoutingExplanation()[0]
		} else {
			cr.OutcomeReason = "routing response degraded"
		}
	case cr.ExpectedRank == 0 && len(top) > 0:
		cr.Outcome = "answered_by_sibling"
		cr.OutcomeReason = fmt.Sprintf("expected provider %q absent from corpora_searched; another provider answered", expectedProvider)
	case !cr.ProviderRouted:
		cr.Outcome = "misrouted"
		cr.OutcomeReason = fmt.Sprintf("expected provider %q absent from corpora_searched", expectedProvider)
	case cr.ExpectedRank == 0:
		cr.Outcome = "below_expectation"
	case c.GetExpectWithinTopK() > 0 && cr.ExpectedRank > c.GetExpectWithinTopK():
		cr.Outcome = "below_expectation"
	case cr.Margin < floor:
		cr.Outcome = "thin_margin"
	default:
		cr.Outcome = "met"
	}
	return cr, latency
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
	for _, c := range suite.GetCases() {
		withinTopK[c.GetCaseId()] = c.GetExpectWithinTopK()
	}
	var gradeable, routed, recalled int
	for _, result := range results {
		if !federatedGradeable(result.GetOutcome()) {
			continue
		}
		gradeable++
		if !result.GetProviderRouted() {
			continue
		}
		routed++
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
	if routed > 0 {
		value := float64(recalled) / float64(routed)
		retrievalRecall = &value
	}
	return routingPrecision, retrievalRecall
}

func federatedGradeable(outcome string) bool {
	switch outcome {
	case "met", "below_expectation", "above_expectation", "unexpected_hit", "answered_by_sibling", "misrouted", "thin_margin":
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
