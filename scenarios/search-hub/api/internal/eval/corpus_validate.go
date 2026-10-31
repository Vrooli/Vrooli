package eval

import (
	"context"
	"fmt"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// Validator re-probes a suite's positive labels through the same provider search
// seam as Runner. It is advisory: provider/probe errors become inconclusive case
// rows instead of failing the whole validation.
type Validator struct {
	resolver ProviderResolver
	client   ProviderClient
}

func NewValidator(resolver ProviderResolver, client ProviderClient) *Validator {
	return &Validator{resolver: resolver, client: client}
}

func (v *Validator) ValidateCorpus(ctx context.Context, suite *evalv1.EvalSuite, deepK int32) (*evalv1.ValidateCorpusResponse, error) {
	if suite == nil {
		return nil, fmt.Errorf("nil suite")
	}
	desc, err := v.resolver.Get(ctx, suite.GetProviderId())
	if err != nil {
		return nil, fmt.Errorf("resolve provider %q: %w", suite.GetProviderId(), err)
	}
	if deepK <= 0 {
		deepK = int32(aisearch.DefaultScoringPolicy.DeepK)
	}
	policy := aisearch.DefaultScoringPolicy
	policy.DeepK = int(deepK)
	if policy.DeepK < policy.GateK {
		policy.DeepK = policy.GateK
		deepK = int32(policy.DeepK)
	}

	out := &evalv1.ValidateCorpusResponse{
		SuiteId:    suite.GetSuiteId(),
		ProviderId: suite.GetProviderId(),
		Rollup:     &evalv1.CorpusValidationRollup{},
	}
	for _, c := range suite.GetCases() {
		if !positiveProtoCase(c) {
			continue
		}
		if c.GetStatus() == "candidate" {
			out.Rollup.Candidate++
			continue
		}
		out.Rollup.Positives++
		row := v.validateCase(ctx, desc, c, deepK, policy)
		out.Cases = append(out.Cases, row)
		switch row.GetReferential() {
		case evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_LIVE:
			out.Rollup.Live++
		case evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_HARD:
			out.Rollup.Hard++
		case evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_STALE:
			out.Rollup.Stale++
		case evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_PROVIDER_ERROR:
			out.Rollup.ProviderErrors++
		default:
			out.Rollup.Inconclusive++
		}
	}
	return out, nil
}

func (v *Validator) validateCase(ctx context.Context, desc *registryv1.ProviderDescriptor, c *evalv1.EvalCase, deepK int32, policy aisearch.ScoringPolicy) *evalv1.CorpusValidationCase {
	probes := []string{c.GetQuery()}
	opts := SearchCallOptions{Scope: c.GetScope()}
	hits, err := v.client.Search(ctx, desc, c.GetQuery(), deepK, opts)
	if err != nil {
		return providerErrorCase(c, probes, err)
	}
	ref, rank := classifyHits(hits, c, policy)
	if ref != evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_STALE {
		return &evalv1.CorpusValidationCase{CaseId: c.GetCaseId(), Referential: ref, ObservedRank: int32(rank), ProbedQueries: probes}
	}

	for _, id := range c.GetExpectIds() {
		probes = append(probes, id)
		confirmHits, confirmErr := v.client.Search(ctx, desc, id, deepK, opts)
		if confirmErr != nil {
			return providerErrorCase(c, probes, confirmErr)
		}
		ref, rank = classifyHits(confirmHits, c, policy)
		if ref != evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_STALE {
			return &evalv1.CorpusValidationCase{CaseId: c.GetCaseId(), Referential: ref, ObservedRank: int32(rank), ProbedQueries: probes}
		}
	}
	return &evalv1.CorpusValidationCase{
		CaseId:        c.GetCaseId(),
		Referential:   evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_STALE,
		ProbedQueries: probes,
		Message:       "expected id absent from case query and confirm probes",
	}
}

func classifyHits(hits []*routingv1.SearchHit, c *evalv1.EvalCase, policy aisearch.ScoringPolicy) (evalv1.ReferentialOutcome, int) {
	results := searchHitsToResults(hits)
	for index, hit := range hits {
		if matched := matchingExpectedIdentity(c, hit); matched != "" {
			results[index].ID = matched
		}
	}
	tc := protoCaseToSearchCase(c)
	rank := aisearch.ExpectedRank(results, tc.ExpectIDs)
	switch aisearch.ClassifyExpectID(results, tc, policy) {
	case aisearch.ReferentialLive:
		return evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_LIVE, rank
	case aisearch.ReferentialHard:
		return evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_HARD, rank
	case aisearch.ReferentialStale:
		return evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_STALE, rank
	default:
		return evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_INCONCLUSIVE, rank
	}
}

func searchHitsToResults(hits []*routingv1.SearchHit) []aisearch.SearchResult {
	out := make([]aisearch.SearchResult, 0, len(hits))
	for _, h := range hits {
		out = append(out, aisearch.SearchResult{ID: h.GetId(), Score: h.GetScore()})
	}
	return out
}

func providerErrorCase(c *evalv1.EvalCase, probes []string, err error) *evalv1.CorpusValidationCase {
	return &evalv1.CorpusValidationCase{
		CaseId:        c.GetCaseId(),
		Referential:   evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_PROVIDER_ERROR,
		ProbedQueries: append([]string(nil), probes...),
		Message:       err.Error(),
	}
}

func positiveProtoCase(c *evalv1.EvalCase) bool {
	return c != nil && !c.GetExpectNoStrongHit() && len(c.GetExpectIds()) > 0
}
