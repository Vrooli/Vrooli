package main

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
	debt "tidiness-manager/internal/discovery"
)

// ReductionFacts is the shared source owner; the recommender never scans or
// reparses files and never substitutes stale file-metric rows for current facts.
type ReductionFacts interface {
	DescribeCodeFacts(context.Context, *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error)
}

func (s *Server) recommendReductions(ctx context.Context, scenario string, limit, minLines int) (debt.Report, error) {
	report, err := currentCodeFacts(ctx, s.reductionFacts, &facts.CodeTarget{Kind: facts.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario, RepoRoot: s.scenarioLocator.repoRoot, LanguageFilter: []string{"go"}})
	if err != nil {
		return debt.Report{}, err
	}
	return debt.New().Recommend(report, limit, minLines), nil
}

func currentCodeFacts(ctx context.Context, client ReductionFacts, target *facts.CodeTarget) (*facts.CodeFactsReport, error) {
	if client == nil {
		url, err := discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURLDefault(ctx, "code-facts")
		if err != nil {
			return nil, fmt.Errorf("resolve reduction facts: %w", err)
		}
		client = factsconnect.NewCodeFactsServiceClient(http.DefaultClient, url)
	}
	response, err := client.DescribeCodeFacts(ctx, connect.NewRequest(&facts.DescribeCodeFactsRequest{
		Target: target, Include: []facts.FactFamily{facts.FactFamily_FACT_FAMILY_SYMBOLS}, IncludeGraphs: true, UseCache: true,
	}))
	if err != nil {
		return nil, fmt.Errorf("current source evidence: %w", err)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("Code Facts returned no source evidence")
	}
	return response.Msg, nil
}
