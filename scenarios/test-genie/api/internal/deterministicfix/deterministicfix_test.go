package deterministicfix

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type fakeClient struct {
	resp *scenariovalidationv1.FixResponse
	err  error
}

func (f fakeClient) PreviewFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.resp), nil
}

func (f fakeClient) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return f.PreviewFix(ctx, req)
}

func TestRunAggregatesAcrossProviders(t *testing.T) {
	clients := map[string]fakeClient{
		"quality-health": {resp: &scenariovalidationv1.FixResponse{Candidates: []*scenariovalidationv1.FixCandidate{
			{RuleId: "TSCONFIG_STRICT", FilePath: "ui/tsconfig.json", Description: "strict"},
		}}},
		"structure-health": {err: connect.NewError(connect.CodeUnimplemented, errors.New("no fixer"))},
		"knowledge-observatory": {resp: &scenariovalidationv1.FixResponse{Candidates: []*scenariovalidationv1.FixCandidate{
			{RuleId: "misplaced_doc", FilePath: "docs/x.md", Description: "move"},
		}}},
		"tidiness-manager": {err: connect.NewError(connect.CodeUnavailable, errors.New("down"))},
	}
	runner := &Runner{
		Providers: []string{"quality-health", "structure-health", "knowledge-observatory", "tidiness-manager", AuditorProviderScenario},
		ResolveBaseURL: func(_ context.Context, scenario string) (string, error) {
			return "http://" + scenario, nil
		},
		NewClient: func(_ time.Duration, baseURL string) FixClient {
			name := baseURL[len("http://"):]
			return clients[name]
		},
		AuditorFix: func(_ context.Context, _, _ string, ruleIDs []string, _ bool) (ProviderReport, error) {
			return ProviderReport{Status: StatusSkipped, Messages: []string{"needs rules"}}, nil
		},
	}

	report := runner.Run(context.Background(), "demo", false, nil)
	if report.TotalCandidates != 2 {
		t.Fatalf("total candidates = %d, want 2", report.TotalCandidates)
	}
	statuses := map[string]string{}
	for _, p := range report.Providers {
		statuses[p.Provider] = p.Status
	}
	if statuses["quality-health"] != StatusFixed {
		t.Fatalf("quality-health status = %q, want fixed", statuses["quality-health"])
	}
	if statuses["structure-health"] != StatusNoFixer {
		t.Fatalf("structure-health status = %q, want no_fixer", statuses["structure-health"])
	}
	if statuses["tidiness-manager"] != StatusUnreachable {
		t.Fatalf("tidiness-manager status = %q, want unreachable", statuses["tidiness-manager"])
	}
	if statuses[AuditorProviderScenario] != StatusSkipped {
		t.Fatalf("scenario-auditor status = %q, want skipped", statuses[AuditorProviderScenario])
	}
}

func TestRunResolveFailureIsUnreachable(t *testing.T) {
	runner := &Runner{
		Providers: []string{"quality-health"},
		ResolveBaseURL: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not running")
		},
		NewClient: func(_ time.Duration, _ string) FixClient { return fakeClient{} },
	}
	report := runner.Run(context.Background(), "demo", false, nil)
	if len(report.Providers) != 1 || report.Providers[0].Status != StatusUnreachable {
		t.Fatalf("expected unreachable, got %+v", report.Providers)
	}
}

func TestDefaultProvidersIncludeFixers(t *testing.T) {
	got := DefaultProviders()
	want := map[string]bool{"quality-health": false, "structure-health": false, "knowledge-observatory": false, AuditorProviderScenario: false}
	for _, p := range got {
		if _, ok := want[p]; ok {
			want[p] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("DefaultProviders missing %q (got %v)", name, got)
		}
	}
}
