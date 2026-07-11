package rolepolicy

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/domain"
)

type fakeResolver struct {
	responses map[domain.RunnerType]ResolvedRole
	errors    map[domain.RunnerType]error
}

func (f fakeResolver) Resolve(_ context.Context, runner domain.RunnerType, role string) (ResolvedRole, error) {
	if err := f.errors[runner]; err != nil {
		return ResolvedRole{}, err
	}
	response := f.responses[runner]
	response.Runner = runner
	response.Role = role
	return response, nil
}

func TestResolveCapturesConcreteResourceEvidenceWithoutDroppingUnavailableFallback(t *testing.T) {
	state, err := NewState(writeCatalog(t, validCatalogJSON), Requirement{Required: true})
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	resolution, err := state.Resolve(context.Background(), fakeResolver{
		responses: map[domain.RunnerType]ResolvedRole{
			domain.RunnerTypeCodex: {
				Model: "gpt-5.4", Fallbacks: []string{"gpt-5.5"},
				Provenance:  ResourceProvenance{Source: "codex catalog", ObservedAt: "2026-07-10"},
				Enforcement: EnforcementPosture{Permissions: "intent_only"}, PolicyPath: "/tmp/codex", PolicyDigest: "sha256:codex",
			},
		},
		errors: map[domain.RunnerType]error{domain.RunnerTypeClaudeCode: ErrResourceUnavailable},
	}, "code.default")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolution.RoleRef != "code.default" || resolution.CatalogDigest == "" || len(resolution.Candidates) != 2 {
		t.Fatalf("resolution = %#v", resolution)
	}
	if candidate := resolution.Candidates[0]; !candidate.Available || candidate.Model != "gpt-5.4" || candidate.PolicyDigest != "sha256:codex" {
		t.Fatalf("available candidate = %#v", candidate)
	}
	if candidate := resolution.Candidates[1]; candidate.Available || candidate.FailureCode != "resource_unavailable" {
		t.Fatalf("unavailable candidate = %#v", candidate)
	}
}

func TestResolveRejectsUnknownRole(t *testing.T) {
	state, err := NewState(writeCatalog(t, validCatalogJSON), Requirement{Required: true})
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	_, err = state.Resolve(context.Background(), fakeResolver{}, "code.unknown")
	if err == nil || errors.Is(err, ErrResourceUnavailable) {
		t.Fatalf("Resolve error = %v, want catalog validation error", err)
	}
}
