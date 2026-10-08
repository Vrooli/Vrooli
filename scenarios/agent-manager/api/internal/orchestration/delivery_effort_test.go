package orchestration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/rolepolicy"

	"github.com/vrooli/api-core/effortauthority"
)

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
			before := effortauthority.Digest(cfg)
			err := o.validateCurrentDeliveryEffort(context.Background(), cfg, candidate)
			if (err == nil) != (name == "match") || r.calls != 1 || before != effortauthority.Digest(cfg) {
				t.Fatal("wrong current effort fence", err)
			}
		})
	}
}

// Actual native admission uses disposable existing owners and a closed
// dispatcher. No CLI, model, unit, grant or credential is installed live.
func TestDeliveryNativeAdmissionHighMismatchHasZeroEffectsAndMediumRemainsCompatible(t *testing.T) {
	t.Setenv(domain.DeliveryEffortEnforceEnv, "1")
	for _, value := range []domain.Effort{domain.EffortHigh, domain.EffortMedium} {
		t.Run(string(value), func(t *testing.T) {
			ctx := context.Background()
			o, a, p, task, profile, key, _ := nativeEffortFixture(t)
			profile.RoleRef = "code.economy.delivery"
			profile.Effort = value
			if err := o.profiles.Update(ctx, profile); err != nil {
				t.Fatal(err)
			}
			persisted, err := o.profiles.Get(ctx, profile.ID)
			if err != nil {
				t.Fatal(err)
			}
			profile = persisted
			p.ID += "-delivery-" + string(value)
			p.Profiles[profile.ProfileKey] = EffortProfileDigest(profile)
			if err = a.Approve(ctx, "fixture-human", p); err != nil {
				t.Fatal(err)
			}
			installFixtureNativeFactory(t, o, p, profile)
			path := filepath.Join(t.TempDir(), "roles.json")
			raw := `{"schemaVersion":1,"metadata":{"catalogId":"delivery-fixture","updatedAt":"2026-10-07"},"defaultRole":"code.economy.delivery","roles":{"code.economy.delivery":{"description":"fixture","intent":"fixture","candidates":[{"runner":"codex","resourceRole":"code.delivery"}]}}}`
			if err = os.WriteFile(path, []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}
			state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
			if err != nil {
				t.Fatal(err)
			}
			o.rolePolicy = state
			o.roleResolver = &deliveryEffortFixtureResolver{effort: domain.EffortMedium}
			before, err := a.Store.Get(ctx, p.ID)
			if err != nil {
				t.Fatal(err)
			}
			taskBefore, err := o.tasks.Get(ctx, task.ID)
			if err != nil {
				t.Fatal(err)
			}
			profileBefore := EffortProfileDigest(profile)
			req := effortRequest(t, o, p, task, profile, key, "delivery-"+string(value))
			_, err = o.CreateRun(ctx, req)
			if value == domain.EffortMedium {
				if !errors.Is(err, spawn.ErrDispatcherClosed) {
					t.Fatal("compatible medium did not reach closed disposable dispatcher", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "effort") {
				t.Fatal("high mismatch did not refuse at effort gate", err)
			}
			after, e := a.Store.Get(ctx, p.ID)
			if e != nil || effortauthority.Digest(before) != effortauthority.Digest(after) {
				t.Fatal("refusal charged authority")
			}
			run, e := o.runs.GetByIdempotencyKey(ctx, req.IdempotencyKey)
			if e != nil || run != nil {
				t.Fatal("refusal created run", e)
			}
			taskAfter, e := o.tasks.Get(ctx, task.ID)
			if e != nil || effortauthority.Digest(taskBefore) != effortauthority.Digest(taskAfter) {
				t.Fatal("refusal changed task")
			}
			profileAfter, e := o.profiles.Get(ctx, profile.ID)
			if e != nil || profileBefore != EffortProfileDigest(profileAfter) {
				t.Fatal("refusal changed profile")
			}
		})
	}
}

func TestDeliveryFallbackWithoutExactEffortEvidenceRefusesBeforeOwnerRead(t *testing.T) {
	r := &deliveryEffortFixtureResolver{effort: domain.EffortMedium}
	o := &Orchestrator{roleResolver: r}
	candidate := domain.ExecutionCandidate{RunnerType: domain.RunnerTypeCodex, ResourceRole: "code.delivery", Model: "fixture-model", DeclaredEffort: domain.EffortMedium}
	cfg := &domain.RunConfig{RoleRef: "code.economy.delivery", Model: "fixture-model", Effort: domain.EffortMedium}
	before := effortauthority.Digest(cfg)
	allowed, err := o.currentCandidateAllowed(context.Background(), cfg, candidate, "unbound-fallback")
	if allowed || err == nil || r.calls != 0 || before != effortauthority.Digest(cfg) {
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
