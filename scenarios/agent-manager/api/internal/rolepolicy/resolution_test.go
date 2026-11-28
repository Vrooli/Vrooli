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

func TestResolvePreferredReordersDeclaredRunnerAndExplainsUnknownPreference(t *testing.T) {
	state, err := NewState(writeCatalog(t, validCatalogJSON), Requirement{Required: true})
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	resolver := fakeResolver{responses: map[domain.RunnerType]ResolvedRole{
		domain.RunnerTypeClaudeCode: {Model: "claude-sonnet"},
		domain.RunnerTypeCodex:      {Model: "gpt-5.4"},
	}}
	preferred, err := state.ResolvePreferred(context.Background(), resolver, "code.default", string(domain.RunnerTypeCodex))
	if err != nil {
		t.Fatalf("ResolvePreferred: %v", err)
	}
	if preferred.SelectionReason != "preferred" || preferred.Candidates[0].Runner != domain.RunnerTypeCodex {
		t.Fatalf("preferred resolution = %#v", preferred)
	}
	unknown, err := state.ResolvePreferred(context.Background(), resolver, "code.default", "not-a-runner")
	if err != nil {
		t.Fatalf("ResolvePreferred unknown: %v", err)
	}
	if unknown.SelectionReason != "preferred_unavailable:runner_not_declared" || unknown.Candidates[0].Runner != domain.RunnerTypeCodex {
		t.Fatalf("unknown preference resolution = %#v", unknown)
	}
}

func TestBrowserCapabilityResolutionIsSeparateAndBoundToAvailableExecutionCandidate(t *testing.T) {
	path := writeCatalog(t, `{"schemaVersion":1,"metadata":{"catalogId":"test","updatedAt":"2026-07-10"},"defaultRole":"browser.agent","roles":{"browser.agent":{"description":"browser task","intent":"browser","candidates":[{"runner":"codex","resourceRole":"code.default"}],"browserCandidates":[{"runner":"codex","resourceRole":"browser.session"}]}}}`)
	state, err := NewState(path, Requirement{Required: true})
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	available, err := state.Resolve(context.Background(), fakeResolver{responses: map[domain.RunnerType]ResolvedRole{
		domain.RunnerTypeCodex: {Model: "gpt-codex", PolicyDigest: "sha256:model-policy"},
	}}, "browser.agent")
	if err != nil {
		t.Fatalf("Resolve available: %v", err)
	}
	if len(available.Candidates) != 1 || available.Candidates[0].ResourceRole != "code.default" || len(available.BrowserCandidates) != 1 || !available.BrowserCandidates[0].Available || available.BrowserCandidates[0].ResourceRole != "browser.session" {
		t.Fatalf("model and browser candidates were not kept separate: %+v", available)
	}
	snapshot := available.Snapshot()
	if len(snapshot.BrowserCandidates) != 1 || snapshot.BrowserCandidates[0].RunnerType != domain.RunnerTypeCodex || !snapshot.BrowserCandidates[0].Available {
		t.Fatalf("immutable browser capability snapshot = %+v", snapshot.BrowserCandidates)
	}
	unavailable, err := state.Resolve(context.Background(), fakeResolver{errors: map[domain.RunnerType]error{domain.RunnerTypeCodex: ErrUnknownResourceRole}}, "browser.agent")
	if err != nil {
		t.Fatalf("Resolve unavailable: %v", err)
	}
	if len(unavailable.BrowserCandidates) != 1 || unavailable.BrowserCandidates[0].Available || unavailable.BrowserCandidates[0].FailureCode != "execution_candidate_unavailable" {
		t.Fatalf("browser candidate must fail closed with its model candidate: %+v", unavailable.BrowserCandidates)
	}
}
