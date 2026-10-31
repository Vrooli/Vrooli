package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"
)

// runConfigDigest is a stable serialization used only to assert a rejected
// delivery-effort decision leaves the resolved configuration untouched.
func runConfigDigest(t *testing.T, cfg *domain.RunConfig) string {
	t.Helper()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("digest config: %v", err)
	}
	return string(raw)
}

type deliveryEffortFixtureResolver struct {
	effort domain.Effort
	err    error
	calls  int
}

func (f *deliveryEffortFixtureResolver) Resolve(_ context.Context, r domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	f.calls++
	return rolepolicy.ResolvedRole{Runner: r, Role: role, Model: "fixture-model", Effort: f.effort}, f.err
}

func TestDeliveryCurrentEffortRefusesMissingChangedAndUnavailableWithoutCoercion(t *testing.T) {
	t.Setenv(domain.DeliveryEffortEnforceEnv, "1")
	for _, name := range []string{"match", "changed", "missing", "unavailable"} {
		t.Run(name, func(t *testing.T) {
			r := &deliveryEffortFixtureResolver{effort: domain.EffortMedium}
			if name == "changed" {
				r.effort = domain.EffortHigh
			}
			if name == "missing" {
				r.effort = ""
			}
			if name == "unavailable" {
				r.err = errors.New("fixture owner unavailable")
			}
			o := &Orchestrator{roleResolver: r}
			candidate := domain.ExecutionCandidate{RunnerType: domain.RunnerTypeCodex, Model: "fixture-model", ResourceRole: "code.delivery", DeclaredEffort: domain.EffortMedium}
			cfg := &domain.RunConfig{Model: "fixture-model", RoleRef: "code.economy.delivery", RunnerType: domain.RunnerTypeCodex, Effort: domain.EffortMedium}
			before := runConfigDigest(t, cfg)
			err := o.validateCurrentDeliveryEffort(context.Background(), cfg, candidate)
			if (err == nil) != (name == "match") || r.calls != 1 || before != runConfigDigest(t, cfg) {
				t.Fatal("wrong current effort fence", err)
			}
		})
	}
}

func TestDeliveryFallbackWithoutExactEffortEvidenceRefusesBeforeOwnerRead(t *testing.T) {
	r := &deliveryEffortFixtureResolver{effort: domain.EffortMedium}
	o := &Orchestrator{roleResolver: r}
	candidate := domain.ExecutionCandidate{RunnerType: domain.RunnerTypeCodex, ResourceRole: "code.delivery", Model: "fixture-model", DeclaredEffort: domain.EffortMedium}
	cfg := &domain.RunConfig{RoleRef: "code.economy.delivery", Model: "fixture-model", Effort: domain.EffortMedium}
	before := runConfigDigest(t, cfg)
	allowed, err := o.currentCandidateAllowed(context.Background(), cfg, candidate, "unbound-fallback")
	if allowed || err == nil || r.calls != 0 || before != runConfigDigest(t, cfg) {
		t.Fatal("fallback lacked exact pre-effect refusal", err)
	}
}

func TestDeliveryCurrentEffortSkipsOwnerReadWhenGateOff(t *testing.T) {
	t.Setenv(domain.DeliveryEffortEnforceEnv, "")
	r := &deliveryEffortFixtureResolver{effort: domain.EffortMedium}
	o := &Orchestrator{roleResolver: r}
	candidate := domain.ExecutionCandidate{RunnerType: domain.RunnerTypeCodex, Model: "fixture-model", ResourceRole: "code.delivery"}
	cfg := &domain.RunConfig{Model: "fixture-model", RoleRef: "code.economy.delivery", RunnerType: domain.RunnerTypeCodex, Effort: domain.EffortHigh}
	if err := o.validateCurrentDeliveryEffort(context.Background(), cfg, candidate); err != nil || r.calls != 0 {
		t.Fatalf("gate-off continuation refused or re-read owner: err=%v calls=%d", err, r.calls)
	}
}
