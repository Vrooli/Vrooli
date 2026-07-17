package structuredresult

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"
)

type extractionRoleResolver struct{}

func (extractionRoleResolver) Resolve(_ context.Context, runnerType domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{Runner: runnerType, Role: role, Model: "fast-model"}, nil
}

func TestRunnerExtractorUsesPortableRoleAndReturnsUntrustedCandidate(t *testing.T) {
	roles, err := rolepolicy.NewState(rolepolicy.ResolvePath(), rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	registry := runner.NewRegistry()
	backend := runner.NewMockRunner(domain.RunnerTypeCodex)
	backend.ExecuteFunc = func(_ context.Context, request runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		if request.ResolvedConfig == nil || request.ResolvedConfig.RoleRef != DefaultExtractRole || request.ResolvedConfig.MaxTurns != 1 {
			t.Fatalf("extract config = %#v", request.ResolvedConfig)
		}
		return &runner.ExecuteResult{Success: true, Result: selected(`"complete"`)}, nil
	}
	if err := registry.Register(backend); err != nil {
		t.Fatal(err)
	}
	extractor := &RunnerExtractor{Roles: roles, Resolver: extractionRoleResolver{}, Runners: registry, WorkingDir: t.TempDir()}
	response, err := extractor.Extract(context.Background(), ExtractRequest{RoleRef: DefaultExtractRole, Source: "The work is done.", Schema: []byte(`{"type":"string","enum":["complete","blocked"]}`)})
	if err != nil {
		t.Fatal(err)
	}
	if string(response.Candidate) != `"complete"` || response.Provider != "codex" || response.PolicySnapshot == nil {
		t.Fatalf("response = %#v", response)
	}
}
