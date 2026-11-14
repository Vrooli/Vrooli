package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	debt "tidiness-manager/internal/discovery"
)

type reductionSourceFunc func(context.Context, *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error)

type auditorFindingsFunc func(context.Context, string) ([]debt.AuditorFinding, error)

func (f auditorFindingsFunc) ListStandardsFindings(ctx context.Context, scenario string) ([]debt.AuditorFinding, error) {
	return f(ctx, scenario)
}

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
	srv.auditorFindings = auditorFindingsFunc(func(received context.Context, scenario string) ([]debt.AuditorFinding, error) {
		if received != ctx || scenario != "demo" {
			t.Fatalf("auditor source scope lost: %s", scenario)
		}
		return []debt.AuditorFinding{{RuleID: "rule-a", Severity: "medium", FilePath: "api/a.go", Recommendation: "Fix the rule."}}, nil
	})
	report, err := srv.recommendReductions(ctx, "demo", 5, 0)
	if err != nil || report.SourceHash != "fresh" || report.Count != 1 || report.Recommendations[0].PolicyRuleID != "rule-a" {
		t.Fatalf("auditor recommendation integration: %+v %v", report, err)
	}
}

func TestScenarioAuditorAdapterReadsCachedPolicyFindings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/standards/violations" || r.URL.Query().Get("scenario") != "demo & tools" {
			t.Fatalf("wrong Auditor request: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"violations":[{"rule_id":"r-1","severity":"high","title":"Policy title","description":"Policy reason","file_path":"api/a.go","line_number":8,"recommendation":"Fix it","standard":"Vrooli","source":"scenario-auditor"}]}`))
	}))
	defer server.Close()

	findings, err := (scenarioAuditorHTTP{baseURL: server.URL, client: server.Client()}).ListStandardsFindings(context.Background(), "demo & tools")
	if err != nil || len(findings) != 1 {
		t.Fatalf("load Auditor findings: %+v %v", findings, err)
	}
	finding := findings[0]
	if finding.RuleID != "r-1" || finding.Severity != "high" || finding.Standard != "Vrooli" || finding.Recommendation != "Fix it" || finding.Source != "scenario-auditor" {
		t.Fatalf("Auditor policy semantics were lost: %+v", finding)
	}
}

func TestScenarioAuditorAdapterMapsPublicTypeFieldToPolicyRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"violations":[{"id":"finding-1","type":"application_logging","severity":"medium","title":"Unstructured Logging in API","file_path":"api/main.go","line_number":42,"recommendation":"Use structured logging","standard":"logging-v1","source":"scenario-auditor","discovered_at":"2026-11-10T18:00:00Z"}]}`))
	}))
	defer server.Close()

	findings, err := (scenarioAuditorHTTP{baseURL: server.URL, client: server.Client()}).ListStandardsFindings(context.Background(), "demo")
	if err != nil || len(findings) != 1 {
		t.Fatalf("load public Auditor violation: %+v %v", findings, err)
	}
	registry := debt.New()
	result := registry.RecommendWithAuditorFindings(&facts.CodeFactsReport{Target: &facts.TargetContext{Requested: &facts.CodeTarget{Scenario: "demo"}}}, findings, 10, 0)
	if result.Count != 1 || result.Recommendations[0].PolicyRuleID != "application_logging" || result.Recommendations[0].PolicyStandard != "logging-v1" || result.Recommendations[0].PolicyFindingID != "finding-1" || result.Recommendations[0].PolicyObservedAt != "2026-11-10T18:00:00Z" || result.Recommendations[0].Calibration.Precision != nil {
		t.Fatalf("public Auditor violation lost its policy identity or was miscalibrated: %+v", result)
	}
}
