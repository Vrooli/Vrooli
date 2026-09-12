package permissionpolicy

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

type fakeProjector struct {
	calls   []ProjectionRequest
	results map[domain.RunnerType]ProjectionResult
	errors  map[domain.RunnerType]error
}

func (f *fakeProjector) Plan(_ context.Context, request ProjectionRequest) (ProjectionResult, error) {
	f.calls = append(f.calls, request)
	if err := f.errors[request.Runner]; err != nil {
		return ProjectionResult{}, err
	}
	return f.results[request.Runner], nil
}

func (f *fakeProjector) Reconcile(context.Context, ProjectionRequest, bool) (ProjectionResult, error) {
	return ProjectionResult{}, errors.New("not used")
}

func TestAggregatePlannerReturnsEveryResourceInDeterministicOrder(t *testing.T) {
	revision := hardEnforcementRevision(t)
	projector := &fakeProjector{
		results: map[domain.RunnerType]ProjectionResult{
			domain.RunnerTypeClaudeCode: {Runner: domain.RunnerTypeClaudeCode, Scope: "user", DesiredDigest: "claude", DesiredFingerprint: "desired", LiveFingerprint: "live", Drift: true, Changes: []string{"replace deny rules"}, NativePaths: []string{"/claude"}, Enforcement: EnforcementPosture{Permissions: "hook_backed"}},
			domain.RunnerTypeCodex:      {Runner: domain.RunnerTypeCodex, Scope: "user", DesiredDigest: "codex", DesiredFingerprint: "desired", LiveFingerprint: "desired", NativePaths: []string{"/codex"}, Enforcement: EnforcementPosture{Permissions: "intent_only"}},
			domain.RunnerTypeOpenCode:   {Runner: domain.RunnerTypeOpenCode, Scope: "user", DesiredDigest: "opencode", DesiredFingerprint: "desired", LiveFingerprint: "live", NativePaths: []string{"/opencode"}, Enforcement: EnforcementPosture{Permissions: "native"}},
		},
		errors: map[domain.RunnerType]error{domain.RunnerTypeGrok: ErrResourceUnavailable},
	}

	plan, err := NewAggregatePlanner(projector).Plan(context.Background(), revision)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !plan.HardEnforcementSatisfied || len(plan.MissingHardEnforcementRuleIDs) != 0 {
		t.Fatalf("hard enforcement = %#v", plan)
	}
	if got, want := resourceRunners(plan.Resources), []domain.RunnerType{domain.RunnerTypeClaudeCode, domain.RunnerTypeCodex, domain.RunnerTypeGrok, domain.RunnerTypeOpenCode}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runner order = %#v, want %#v", got, want)
	}
	if plan.Resources[2].Status != "unavailable" || plan.Resources[2].Installed {
		t.Fatalf("grok plan = %#v", plan.Resources[2])
	}
	if plan.Resources[0].DesiredDigest != "claude" || plan.Resources[0].Enforcement.Permissions != "hook_backed" {
		t.Fatalf("claude plan = %#v", plan.Resources[0])
	}
}

func TestAggregatePlannerReportsMissingHardEnforcementWithoutHidingPlan(t *testing.T) {
	revision := hardEnforcementRevision(t)
	projector := &fakeProjector{results: map[domain.RunnerType]ProjectionResult{
		domain.RunnerTypeCodex: {Runner: domain.RunnerTypeCodex, Scope: "user", DesiredDigest: "codex", DesiredFingerprint: "desired", LiveFingerprint: "live", NativePaths: []string{"/codex"}, Enforcement: EnforcementPosture{Permissions: "intent_only"}},
	}, errors: map[domain.RunnerType]error{
		domain.RunnerTypeClaudeCode: ErrResourceUnavailable,
		domain.RunnerTypeGrok:       ErrResourceUnavailable,
		domain.RunnerTypeOpenCode:   ErrResourceUnavailable,
	}}

	plan, err := NewAggregatePlanner(projector).Plan(context.Background(), revision)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.HardEnforcementSatisfied || !reflect.DeepEqual(plan.MissingHardEnforcementRuleIDs, []string{"deny-root"}) {
		t.Fatalf("hard enforcement = %#v", plan)
	}
	if len(plan.Resources) != 4 {
		t.Fatalf("resources = %#v", plan.Resources)
	}
}

func hardEnforcementRevision(t *testing.T) *Revision {
	t.Helper()
	data := strings.Replace(validCatalogJSON, `"requiresHardEnforcement": false`, `"requiresHardEnforcement": true`, 1)
	revision, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return revision
}

func resourceRunners(resources []ResourcePlan) []domain.RunnerType {
	runners := make([]domain.RunnerType, 0, len(resources))
	for _, resource := range resources {
		runners = append(runners, resource.Runner)
	}
	return runners
}
