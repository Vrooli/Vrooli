package eval

import (
	"context"
	"fmt"

	"search-hub/internal/clock"
	internalrouting "search-hub/internal/routing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// QueryClient is the consumer-owned seam for the public federated query path.
// The eval domain observes the routing contract and never imports Router.
type QueryClient interface {
	Query(context.Context, *routingv1.QueryRequest) (*routingv1.QueryResponse, error)
}

const defaultFederatedMargin = 0.01

// FederatedRunner evaluates the same registered corpus through QueryService,
// recording routing, ranking, margin, and honest degradation evidence.
type FederatedRunner struct {
	resolver ProviderResolver
	query    QueryClient
	clock    clock.Clock
	newID    func() string
}

func NewFederatedRunner(resolver ProviderResolver, query QueryClient, clk clock.Clock, newID func() string) *FederatedRunner {
	if newID == nil {
		newID = func() string { return "federated-run" }
	}
	return &FederatedRunner{resolver: resolver, query: query, clock: clk, newID: newID}
}

func (r *FederatedRunner) Run(ctx context.Context, suite *evalv1.EvalSuite, tag string, limit int32) (*evalv1.EvalRun, error) {
	if suite == nil {
		return nil, fmt.Errorf("nil suite")
	}
	if _, err := r.resolver.Get(ctx, suite.GetProviderId()); err != nil {
		return nil, fmt.Errorf("resolve provider %q: %w", suite.GetProviderId(), err)
	}
	if limit <= 0 {
		limit = effectiveLimit(suite, limit)
	}
	ctx = internalrouting.WithBackgroundEvaluationProvider(ctx, suite.GetProviderId())
	results := make([]*evalv1.CaseResult, 0, len(suite.GetCases()))
	latencies := make([]int64, 0, len(suite.GetCases()))
	run := &evalv1.EvalRun{RunId: r.newID(), SuiteId: suite.GetSuiteId(), Tag: tag, Tier: "federated"}
	for _, c := range suite.GetCases() {
		start := r.clock.Now()
		response, err := r.query.Query(ctx, &routingv1.QueryRequest{Query: c.GetQuery(), Limit: limit})
		latencies = append(latencies, r.clock.Now().Sub(start).Milliseconds())
		cr := &evalv1.CaseResult{CaseId: c.GetCaseId(), ExpectedProviderId: suite.GetProviderId()}
		if err != nil {
			cr.Outcome = "error"
			cr.OutcomeReason = err.Error()
			run.Degraded = true
			run.DegradedReason = err.Error()
			results = append(results, cr)
			continue
		}
		cr.ProviderRouted = contains(response.GetCorporaSearched(), suite.GetProviderId())
		top := response.GetRanked()
		if len(top) == 0 {
			top = flattenGroups(response.GetGroups())
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
		case response.GetDegraded():
			cr.Outcome = "degraded"
			if len(response.GetRoutingExplanation()) > 0 {
				cr.OutcomeReason = response.GetRoutingExplanation()[0]
			} else {
				cr.OutcomeReason = "routing response degraded"
			}
			run.Degraded = true
			if run.DegradedReason == "" {
				run.DegradedReason = cr.OutcomeReason
			}
		case cr.ExpectedRank == 0 && len(top) > 0:
			cr.Outcome = "answered_by_sibling"
			cr.OutcomeReason = fmt.Sprintf("expected provider %q absent from corpora_searched; another provider answered", suite.GetProviderId())
		case !cr.ProviderRouted:
			cr.Outcome = "misrouted"
			cr.OutcomeReason = fmt.Sprintf("expected provider %q absent from corpora_searched", suite.GetProviderId())
		case cr.ExpectedRank == 0:
			cr.Outcome = "below_expectation"
		case c.GetExpectWithinTopK() > 0 && cr.ExpectedRank > c.GetExpectWithinTopK():
			cr.Outcome = "below_expectation"
		case cr.Margin < floor:
			cr.Outcome = "thin_margin"
		default:
			cr.Outcome = "met"
		}
		results = append(results, cr)
	}
	run.Results = results
	run.Aggregate = aggregate(suite, results, latencies)
	return run, nil
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
