package main

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type reductionSourceFunc func(context.Context, *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error)

func (f reductionSourceFunc) DescribeCodeFacts(ctx context.Context, request *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
	return f(ctx, request)
}

func TestReductionRecommendationsRequireCurrentSharedEvidence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := &Server{scenarioLocator: &ScenarioLocator{repoRoot: "/repo"}}
	calls := 0
	srv.reductionFacts = reductionSourceFunc(func(received context.Context, request *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		calls++
		if received != ctx || !request.Msg.GetIncludeGraphs() || !request.Msg.GetUseCache() || request.Msg.GetTarget().GetScenario() != "demo" {
			t.Fatalf("shared source scope lost: %v", request.Msg)
		}
		return nil, errors.New("source unavailable")
	})
	if _, err := srv.recommendReductions(ctx, "demo", 5, 0); err == nil || calls != 1 {
		t.Fatalf("unavailable evidence converted to recommendations: %v calls=%d", err, calls)
	}
	// No database or campaign manager is needed: recommendations are current
	// source facts, and fetching them cannot create a visited-tracker campaign.
	srv.reductionFacts = reductionSourceFunc(func(context.Context, *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		return connect.NewResponse(&facts.CodeFactsReport{Cache: &facts.CacheMetadata{SourceHash: "fresh"}}), nil
	})
	report, err := srv.recommendReductions(ctx, "demo", 5, 0)
	if err != nil || report.SourceHash != "fresh" || report.Count != 0 {
		t.Fatalf("fresh empty result: %+v %v", report, err)
	}
}
