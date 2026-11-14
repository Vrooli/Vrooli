package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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

type AuditorFindingsSource interface {
	ListStandardsFindings(context.Context, string) ([]debt.AuditorFinding, error)
}

func (s *Server) recommendReductions(ctx context.Context, scenario string, limit, minLines int) (debt.Report, error) {
	report, err := currentCodeFacts(ctx, s.reductionFacts, &facts.CodeTarget{Kind: facts.TargetKind_TARGET_KIND_SCENARIO, Scenario: scenario, RepoRoot: s.scenarioLocator.repoRoot, LanguageFilter: []string{"go", "typescript", "javascript"}})
	if err != nil {
		return debt.Report{}, err
	}
	registry := debt.Default()
	findings, err := currentAuditorFindings(ctx, s.auditorFindings, scenario)
	result := registry.RecommendWithAuditorFindings(report, findings, limit, minLines)
	if err != nil {
		result.Warnings = append(result.Warnings, "scenario-auditor findings unavailable: "+err.Error())
	}
	return result, nil
}

type scenarioAuditorHTTP struct {
	baseURL string
	client  *http.Client
}

func (c scenarioAuditorHTTP) ListStandardsFindings(ctx context.Context, scenario string) ([]debt.AuditorFinding, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		var err error
		baseURL, err = discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURLDefault(ctx, "scenario-auditor")
		if err != nil {
			return nil, fmt.Errorf("resolve Scenario Auditor: %w", err)
		}
	}
	requestURL := strings.TrimRight(baseURL, "/") + "/api/v1/standards/violations?scenario=" + url.QueryEscape(scenario)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	client := c.client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Scenario Auditor returned HTTP %d", response.StatusCode)
	}
	var payload struct {
		Violations []debt.AuditorFinding `json:"violations"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode Scenario Auditor findings: %w", err)
	}
	for index := range payload.Violations {
		finding := &payload.Violations[index]
		if finding.RuleID == "" {
			// Scenario Auditor's public violations DTO calls the originating rule
			// `type`; RuleID remains accepted for forward-compatible responses.
			finding.RuleID = finding.Type
		}
	}
	return payload.Violations, nil
}

func currentAuditorFindings(ctx context.Context, source AuditorFindingsSource, scenario string) ([]debt.AuditorFinding, error) {
	if source == nil {
		source = scenarioAuditorHTTP{}
	}
	return source.ListStandardsFindings(ctx, scenario)
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

// resolvedCodeFactsClient keeps the service lifecycle independent from Code
// Facts startup order while making the broker mandatory for production scans.
type resolvedCodeFactsClient struct{}

func (resolvedCodeFactsClient) DescribeCodeFacts(ctx context.Context, request *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
	baseURL, err := discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURLDefault(ctx, "code-facts")
	if err != nil {
		return nil, fmt.Errorf("resolve Code Facts: %w", err)
	}
	return factsconnect.NewCodeFactsServiceClient(http.DefaultClient, baseURL).DescribeCodeFacts(ctx, request)
}
