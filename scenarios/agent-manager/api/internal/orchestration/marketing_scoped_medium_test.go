package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/rolepolicy"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
)

// This is a separately installed disposable profile, never a projection of a
// retained high run or a per-run override. Production declarations stay intact.
func newMarketingMediumFixture(t *testing.T, starts int) *serialSchedulerFixture {
	t.Helper()
	return newSerialSchedulerFixturePrepared(t, starts, func(o *Orchestrator, shared *domain.AgentProfile) *domain.AgentProfile {
		ctx := context.Background()
		shared.RoleRef = "code.economy.delivery"
		shared.Effort = domain.EffortHigh
		if err := o.profiles.Update(ctx, shared); err != nil {
			t.Fatal(err)
		}
		shared, err := o.profiles.GetByKey(ctx, shared.ProfileKey)
		if err != nil {
			t.Fatal(err)
		}
		frozen := EffortProfileDigest(shared)
		raw, err := json.Marshal(shared)
		if err != nil {
			t.Fatal(err)
		}
		var candidate domain.AgentProfile
		if err = json.Unmarshal(raw, &candidate); err != nil {
			t.Fatal(err)
		}
		candidate.ID = uuid.Nil
		candidate.Name = "Disposable Marketing medium parent"
		candidate.ProfileKey = "marketing-medium-parent-fixture"
		candidate.Effort = domain.EffortMedium
		prepared, err := o.CreateProfile(ctx, &candidate)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err = o.profiles.GetByKey(ctx, prepared.ProfileKey)
		if err != nil || prepared == nil {
			t.Fatal(err)
		}
		// Normalize only independently assigned identity/timestamps and the chosen
		// effort: every permission, native ceiling and executable field must match.
		equivalent := *prepared
		equivalent.ID, equivalent.Name, equivalent.ProfileKey, equivalent.Effort = shared.ID, shared.Name, shared.ProfileKey, shared.Effort
		equivalent.CreatedAt, equivalent.UpdatedAt = shared.CreatedAt, shared.UpdatedAt
		if EffortProfileDigest(&equivalent) != frozen {
			t.Fatal("candidate expanded or changed source contract")
		}
		t.Cleanup(func() {
			current, e := o.profiles.GetByKey(ctx, shared.ProfileKey)
			if e != nil || EffortProfileDigest(current) != frozen || current.Effort != domain.EffortHigh {
				t.Error("shared high profile changed", e)
			}
		})
		path := filepath.Join(t.TempDir(), "delivery.json")
		policy := `{"schemaVersion":1,"metadata":{"catalogId":"marketing-medium-fixture","updatedAt":"2026-10-07"},"defaultRole":"code.economy.delivery","roles":{"code.economy.delivery":{"description":"fixture","intent":"fixture","candidates":[{"runner":"codex","resourceRole":"code.delivery"}]}}}`
		if err = os.WriteFile(path, []byte(policy), 0o600); err != nil {
			t.Fatal(err)
		}
		state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
		if err != nil {
			t.Fatal(err)
		}
		o.rolePolicy = state
		o.roleResolver = &deliveryEffortFixtureResolver{effort: domain.EffortMedium}
		return prepared
	})
}

func TestMarketingSeparateMediumProfileAlternationRestartReplayAndConservation(t *testing.T) {
	ctx := context.Background()
	f := newMarketingMediumFixture(t, 3)
	originalTask, err := f.o.tasks.Get(ctx, f.task.ID)
	if err != nil {
		t.Fatal(err)
	}
	taskPin := effortauthority.Digest(originalTask)
	worker := f.successor(t, f.root)
	before, err := f.authority.Store.Get(ctx, f.policy.ID)
	if err != nil {
		t.Fatal(err)
	}
	restarted := *f.o
	if err = restarted.advanceFiniteSerialEpisode(ctx, f.root); err != nil {
		t.Fatal(err)
	}
	after, err := f.authority.Store.Get(ctx, f.policy.ID)
	if err != nil || effortauthority.Digest(before) != effortauthority.Digest(after) {
		t.Fatal("restart replay changed ledger", err)
	}
	f.complete(t, worker, f.parentProfile.ProfileKey)
	parent := f.successor(t, worker)
	if parent.ID == f.root.ID || parent.ID == worker.ID || worker.ParentRunID == nil || *worker.ParentRunID != f.root.ID || parent.ParentRunID == nil || *parent.ParentRunID != worker.ID {
		t.Fatal("exact-parent lineage changed")
	}
	if len(f.policy.Profiles) != 2 || f.policy.Profiles["finite-fixture"] != "" {
		t.Fatal("shared high profile became commissioned")
	}
	rec, err := f.authority.Store.Get(ctx, f.policy.ID)
	if err != nil || len(rec.Reservations) != 3 {
		t.Fatal("budget accounting", err)
	}
	active := 0
	for _, slot := range rec.Reservations {
		if !slot.Terminal {
			active++
		}
		if slot.Intent.Turns != f.policy.MaxTurns || slot.Intent.ToolCalls != f.policy.MaxToolCalls || slot.Intent.RunSeconds != f.policy.MaxRunSeconds {
			t.Fatal("partial charge")
		}
	}
	if active != 1 {
		t.Fatal("concurrency changed")
	}
	for _, run := range []*domain.Run{f.root, worker, parent} {
		if run.TaskID != f.task.ID || run.OwnerSubject != f.policy.Owner || run.OwnerExpiresAt == nil || !run.OwnerExpiresAt.Equal(f.policy.Deadline) || run.ResolvedConfig.Effort != domain.EffortMedium || run.ResolvedConfig.Model != "fixture-model" {
			t.Fatal("scope, effective policy or deadline changed")
		}
	}
	finalTask, err := f.o.tasks.Get(ctx, f.task.ID)
	if err != nil || effortauthority.Digest(finalTask) != taskPin {
		t.Fatal("original task changed", err)
	}
	// Starts are exhausted, not renewed by selecting a compatible profile.
	f.complete(t, parent, f.workerProfile.ProfileKey)
	if err = f.o.advanceFiniteSerialEpisode(ctx, parent); err == nil || errors.Is(err, spawn.ErrDispatcherClosed) {
		t.Fatal("start exhaustion dispatched", err)
	}
	exhausted, err := f.authority.Store.Get(ctx, f.policy.ID)
	if err != nil || len(exhausted.Reservations) != 3 {
		t.Fatal("exhaustion charged new start", err)
	}
}

func TestMarketingSeparateMediumProfileRefusalsBeforeMetadata(t *testing.T) {
	t.Setenv(domain.DeliveryEffortEnforceEnv, "1")
	for _, name := range []string{"high-receipt", "wrong-model-receipt", "absent-receipt", "changed-profile-digest", "permission-expansion", "resource-high", "resource-missing", "resource-model-drift", "changed-scope", "expired", "revoked"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			f := newMarketingMediumFixture(t, 3)
			switch name {
			case "high-receipt":
				f.root.ResolvedConfig.Admission.Receipt.EffectiveEffort = "high"
			case "wrong-model-receipt":
				f.root.ResolvedConfig.Admission.Receipt.EffectiveModel = "other-model"
			case "absent-receipt":
				f.root.ResolvedConfig.Admission.Receipt = nil
			case "changed-profile-digest":
				f.workerProfile.Description += " changed"
				if err := f.o.profiles.Update(ctx, f.workerProfile); err != nil {
					t.Fatal(err)
				}
			case "permission-expansion":
				f.workerProfile.DeclaredScopes = append(f.workerProfile.DeclaredScopes, "swarm-manager:write")
				if err := f.o.profiles.Update(ctx, f.workerProfile); err != nil {
					t.Fatal(err)
				}
			case "resource-high":
				f.o.roleResolver = &deliveryEffortFixtureResolver{effort: domain.EffortHigh}
			case "resource-missing":
				f.o.roleResolver = &deliveryEffortFixtureResolver{}
			case "resource-model-drift":
				f.o.roleResolver = marketingChangedModelResolver{}
			case "changed-scope":
				f.task.ProjectRoot = t.TempDir()
				if err := f.o.tasks.Update(ctx, f.task); err != nil {
					t.Fatal(err)
				}
			case "expired":
				*f.now = f.policy.Deadline
			case "revoked":
				if err := f.authority.Revoke(ctx, "fixture-human", f.policy.ID); err != nil {
					t.Fatal(err)
				}
			}
			// Exact-parent admission reads the durable native owner row, not an
			// edited callback copy. Persist invalid receipt evidence before snapshot.
			if name == "high-receipt" || name == "wrong-model-receipt" || name == "absent-receipt" {
				if err := f.o.runs.Update(ctx, f.root); err != nil {
					t.Fatal(err)
				}
			}
			before, err := f.authority.Store.Get(ctx, f.policy.ID)
			if err != nil {
				t.Fatal(err)
			}
			runBefore, err := f.o.runs.Get(ctx, f.root.ID)
			if err != nil {
				t.Fatal(err)
			}
			taskBefore, err := f.o.tasks.Get(ctx, f.task.ID)
			if err != nil {
				t.Fatal(err)
			}
			workerBefore, err := f.o.profiles.GetByKey(ctx, f.workerProfile.ProfileKey)
			if err != nil {
				t.Fatal(err)
			}
			if err = f.o.advanceFiniteSerialEpisode(ctx, f.root); err == nil {
				t.Fatal("invalid successor accepted")
			}
			after, err := f.authority.Store.Get(ctx, f.policy.ID)
			if err != nil || effortauthority.Digest(before) != effortauthority.Digest(after) {
				t.Fatal("refusal changed ledger/metadata", err)
			}
			next, err := f.o.runs.GetByIdempotencyKey(ctx, "serial-"+f.root.ID.String())
			if err != nil || next != nil {
				t.Fatal("refusal created successor", err)
			}
			runAfter, _ := f.o.runs.Get(ctx, f.root.ID)
			taskAfter, _ := f.o.tasks.Get(ctx, f.task.ID)
			workerAfter, _ := f.o.profiles.GetByKey(ctx, f.workerProfile.ProfileKey)
			if effortauthority.Digest(runBefore) != effortauthority.Digest(runAfter) || effortauthority.Digest(taskBefore) != effortauthority.Digest(taskAfter) || EffortProfileDigest(workerBefore) != EffortProfileDigest(workerAfter) {
				t.Fatal("refusal changed owned data")
			}
		})
	}
}

type marketingChangedModelResolver struct{}

func (marketingChangedModelResolver) Resolve(_ context.Context, runner domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{Runner: runner, Role: role, Model: "other-model", Effort: domain.EffortMedium}, nil
}
