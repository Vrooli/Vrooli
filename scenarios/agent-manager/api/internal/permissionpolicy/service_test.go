package permissionpolicy

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

type reconcileProjector struct {
	calls   []domain.RunnerType
	results map[domain.RunnerType]ProjectionResult
	errors  map[domain.RunnerType]error
}

func (p *reconcileProjector) Plan(context.Context, ProjectionRequest) (ProjectionResult, error) {
	return ProjectionResult{}, errors.New("not used")
}

func (p *reconcileProjector) Reconcile(_ context.Context, request ProjectionRequest, authorized bool) (ProjectionResult, error) {
	if !authorized {
		return ProjectionResult{}, ErrAuthorizationRequired
	}
	p.calls = append(p.calls, request.Runner)
	if err := p.errors[request.Runner]; err != nil {
		return ProjectionResult{}, err
	}
	return p.results[request.Runner], nil
}

type memoryAuditStore struct {
	recorded []ReconcileResult
}

func (s *memoryAuditStore) RecordReconcile(_ context.Context, result ReconcileResult) error {
	s.recorded = append(s.recorded, *result.Clone())
	return nil
}

func (s *memoryAuditStore) LastReconcile(context.Context) (*ReconcileResult, error) {
	if len(s.recorded) == 0 {
		return nil, nil
	}
	return s.recorded[len(s.recorded)-1].Clone(), nil
}

func TestReconcileRequiresExplicitAuthorization(t *testing.T) {
	// enforces invariant: permissionReconcileRequiresExplicitAuthorization
	state := activePermissionState(t, hardEnforcementCatalog(t))
	projector := &reconcileProjector{}
	service := newService(state, projector, &memoryAuditStore{}, time.Now)

	if _, err := service.Reconcile(context.Background(), false); !errors.Is(err, ErrAuthorizationRequired) {
		t.Fatalf("Reconcile error = %v, want authorization error", err)
	}
	if len(projector.calls) != 0 {
		t.Fatalf("projector calls = %#v, want none", projector.calls)
	}
}

func TestReconcilePersistsPartialFailureInDeterministicOrder(t *testing.T) {
	// enforces invariant: permissionReconcileNeverClaimsPartialSuccess
	state := activePermissionState(t, hardEnforcementCatalog(t))
	projector := &reconcileProjector{
		results: map[domain.RunnerType]ProjectionResult{
			domain.RunnerTypeClaudeCode: resultFor(domain.RunnerTypeClaudeCode, "hook_backed"),
			domain.RunnerTypeCodex:      resultFor(domain.RunnerTypeCodex, "intent_only"),
			domain.RunnerTypeOpenCode:   resultFor(domain.RunnerTypeOpenCode, "native"),
		},
		errors: map[domain.RunnerType]error{domain.RunnerTypeGrok: ErrResourceUnavailable},
	}
	audit := &memoryAuditStore{}
	now := time.Date(2026, 7, 10, 21, 30, 0, 0, time.UTC)
	service := newService(state, projector, audit, func() time.Time { return now })

	result, err := service.Reconcile(context.Background(), true)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Success {
		t.Fatalf("partial result reported global success: %#v", result)
	}
	if !result.HardEnforcementSatisfied {
		t.Fatalf("hook-backed/native resources should satisfy hard enforcement: %#v", result)
	}
	wantOrder := []domain.RunnerType{domain.RunnerTypeClaudeCode, domain.RunnerTypeCodex, domain.RunnerTypeGrok, domain.RunnerTypeOpenCode}
	if !reflect.DeepEqual(projector.calls, wantOrder) {
		t.Fatalf("projector order = %#v, want %#v", projector.calls, wantOrder)
	}
	if got := result.Resources[2]; got.Status != "unavailable" || got.Installed {
		t.Fatalf("grok result = %#v", got)
	}
	if len(audit.recorded) != 1 || !reflect.DeepEqual(*audit.recorded[0].Clone(), result) {
		t.Fatalf("audit = %#v, result = %#v", audit.recorded, result)
	}

	result.Resources[0].Changes[0] = "mutated by caller"
	last, err := service.LastReconcile(context.Background())
	if err != nil || last == nil {
		t.Fatalf("LastReconcile = %#v, %v", last, err)
	}
	if last.Resources[0].Changes[0] == "mutated by caller" {
		t.Fatalf("last reconcile exposed mutable audit result: %#v", last)
	}
}

func TestReconcileReportsMissingHardEnforcementAndContinues(t *testing.T) {
	state := activePermissionState(t, hardEnforcementCatalog(t))
	projector := &reconcileProjector{
		results: map[domain.RunnerType]ProjectionResult{
			domain.RunnerTypeCodex: resultFor(domain.RunnerTypeCodex, "intent_only"),
		},
		errors: map[domain.RunnerType]error{
			domain.RunnerTypeClaudeCode: ErrResourceUnavailable,
			domain.RunnerTypeGrok:       ErrResourceUnavailable,
			domain.RunnerTypeOpenCode:   ErrResourceUnavailable,
		},
	}
	service := newService(state, projector, &memoryAuditStore{}, time.Now)

	result, err := service.Reconcile(context.Background(), true)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Success || result.HardEnforcementSatisfied {
		t.Fatalf("result = %#v", result)
	}
	if !reflect.DeepEqual(result.MissingHardEnforcementRuleIDs, []string{"deny-root"}) {
		t.Fatalf("missing hard enforcement = %#v", result.MissingHardEnforcementRuleIDs)
	}
	if len(projector.calls) != 4 {
		t.Fatalf("projector calls = %#v", projector.calls)
	}
	if err := service.ReadinessError(context.Background()); err == nil {
		t.Fatal("ReadinessError() = nil, want required hard-enforcement failure")
	}
}

func TestReadinessRequiresCurrentHardEnforcementEvidence(t *testing.T) {
	state := activePermissionState(t, hardEnforcementCatalog(t))
	service := newService(state, &reconcileProjector{}, &memoryAuditStore{}, time.Now)
	if err := service.ReadinessError(context.Background()); err == nil {
		t.Fatal("ReadinessError() = nil, want missing assessment failure")
	}

	projector := &reconcileProjector{
		results: map[domain.RunnerType]ProjectionResult{
			domain.RunnerTypeClaudeCode: resultFor(domain.RunnerTypeClaudeCode, "hook_backed"),
			domain.RunnerTypeCodex:      resultFor(domain.RunnerTypeCodex, "intent_only"),
			domain.RunnerTypeGrok:       resultFor(domain.RunnerTypeGrok, "hook_backed"),
			domain.RunnerTypeOpenCode:   resultFor(domain.RunnerTypeOpenCode, "native"),
		},
	}
	service = newService(state, projector, &memoryAuditStore{}, time.Now)
	if _, err := service.Reconcile(context.Background(), true); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if err := service.ReadinessError(context.Background()); err != nil {
		t.Fatalf("ReadinessError() = %v, want nil after enforced reconcile", err)
	}
}

func activePermissionState(t *testing.T, catalog string) *State {
	t.Helper()
	state, err := NewState(writeCatalog(t, catalog), Requirement{Required: true, Reason: "test"})
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	return state
}

func hardEnforcementCatalog(t *testing.T) string {
	t.Helper()
	return `{
  "schemaVersion": 1,
  "metadata": {"catalogId": "test", "updatedAt": "2026-07-10"},
  "targetScopes": ["user"],
  "rules": [{
    "id": "deny-root", "action": "deny", "matcher": {"kind": "bash", "pattern": "rm -rf /"},
    "rationale": "Protect root.", "owner": "test", "targetScope": "user", "requiresHardEnforcement": true
  }]
}`
}

func resultFor(runner domain.RunnerType, posture string) ProjectionResult {
	return ProjectionResult{
		Runner:             runner,
		Scope:              "user",
		DesiredDigest:      string(runner) + "-digest",
		DesiredFingerprint: "desired",
		LiveFingerprint:    "desired",
		NativePaths:        []string{"/tmp/" + string(runner)},
		Changes:            []string{"applied managed rules"},
		Enforcement:        EnforcementPosture{Permissions: posture},
	}
}
