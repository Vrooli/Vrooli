package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

// The orchestrated-epoch model runs the orchestrator and its epoch workers in
// the shared tree (D5). The worker must narrow the orchestrator so live
// dependent delegation admits it, and only the orchestrator may manage children.
func TestDeliveryOrchestratorAdmitsItsEpochWorker(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	root := filepath.Clean("../../../../prompt-manager")
	load := func(name string) *domain.AgentProfile {
		t.Helper()
		result := o.reconcileProfileSource(ctx, "prompt-manager", root, ".vrooli/agent-manager/"+name+".json", profileReconcileModeUpdateIfUnmodified, false)
		if result.Status != ProfileReconcileStatusCreated {
			t.Fatalf("%s did not reconcile: %+v", name, result)
		}
		profile, err := o.profiles.GetByKey(ctx, "prompt-manager/"+name)
		if err != nil || profile == nil {
			t.Fatalf("%s missing: %v", name, err)
		}
		return profile
	}
	orchestrator, worker := load("delivery-orchestrator"), load("delivery-epoch-worker")
	if orchestrator.Effort != domain.EffortHigh || worker.Effort != domain.EffortMedium {
		t.Fatalf("efforts orchestrator=%s worker=%s, want high/medium", orchestrator.Effort, worker.Effort)
	}
	if orchestrator.MaxTurns != 0 || worker.MaxTurns != 0 {
		t.Fatalf("epoch roles must not carry per-turn caps: orchestrator=%d worker=%d", orchestrator.MaxTurns, worker.MaxTurns)
	}
	// Codec-pipe Codex runs end after one turn; the worker needs the runner's
	// native /goal loop on the interactive substrate to hold an epoch.
	if worker.SpawnPolicy == nil || len(worker.SpawnPolicy.ExecutionMode.Prefer) == 0 || worker.SpawnPolicy.ExecutionMode.Prefer[0] != "interactive" {
		t.Fatalf("epoch worker must prefer the interactive substrate: %+v", worker.SpawnPolicy)
	}
	if !slices.Contains(orchestrator.DeclaredScopes, "agent-manager:orchestrate") || slices.Contains(worker.DeclaredScopes, "agent-manager:orchestrate") {
		t.Fatalf("only the orchestrator declares the orchestrate scope: orchestrator=%v worker=%v", orchestrator.DeclaredScopes, worker.DeclaredScopes)
	}
	parent := &domain.RunConfig{NetworkAccess: orchestrator.NetworkAccess, SandboxConfig: orchestrator.SandboxConfig}
	child := &domain.RunConfig{NetworkAccess: worker.NetworkAccess, SandboxConfig: worker.SandboxConfig}
	if err := dependentExecutionPermissionsError(parent, child); err != nil {
		t.Fatalf("epoch worker must narrow the orchestrator: %v", err)
	}
}

func TestReconcileScenarioProfilesPreservesUnifiedDeclarationValidation(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	result, err := o.ReconcileScenarioProfiles(context.Background(), ReconcileScenarioProfilesRequest{})
	if err == nil || result != nil {
		t.Fatalf("empty scenario result=%+v err=%v, want validation error", result, err)
	}
}

func TestReconcileScenarioProfilesProjectsSelfDeclaredProfiles(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	result, err := o.ReconcileScenarioProfiles(context.Background(), ReconcileScenarioProfilesRequest{Scenario: agentManagerSelfScenario, DryRun: true})
	if err != nil {
		t.Fatalf("reconcile self-declared profiles: %v", err)
	}
	if result.Scenario != agentManagerSelfScenario || !result.DryRun || result.Created < 2 || result.Failed != 0 {
		t.Fatalf("reconciliation result=%+v", result)
	}
	for _, item := range result.Results {
		if item.Status != ProfileReconcileStatusCreated || item.ProfileID == "" || item.SourceHash == "" {
			t.Fatalf("profile result=%+v", item)
		}
	}
}

func TestReconcileProfileSourceHonorsLifecycleModesAndOwnership(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	root := t.TempDir()
	source := ".vrooli/agent-manager/default.json"
	path := filepath.Join(root, filepath.FromSlash(source))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(fixtureProfile)

	created := o.reconcileProfileSource(ctx, "fixture-scn", root, source, profileReconcileModeUpdateIfUnmodified, false)
	if created.Status != ProfileReconcileStatusCreated || created.ProfileID == "" || created.SourceHash == "" {
		t.Fatalf("create result=%+v", created)
	}
	unchanged := o.reconcileProfileSource(ctx, "fixture-scn", root, source, profileReconcileModeUpdateIfUnmodified, false)
	if unchanged.Status != ProfileReconcileStatusUnchanged || unchanged.ProfileID != created.ProfileID {
		t.Fatalf("unchanged result=%+v", unchanged)
	}
	if skipped := o.reconcileProfileSource(ctx, "fixture-scn", root, source, profileReconcileModeCreateOnly, false); skipped.Status != ProfileReconcileStatusSkipped {
		t.Fatalf("create-only result=%+v", skipped)
	}

	updatedSource := strings.Replace(fixtureProfile, `"name": "fixture-scn default"`, `"name": "fixture-scn updated"`, 1)
	write(updatedSource)
	if dryRun := o.reconcileProfileSource(ctx, "fixture-scn", root, source, profileReconcileModeUpdateIfUnmodified, true); dryRun.Status != ProfileReconcileStatusUpdated || dryRun.Message != "would update profile" {
		t.Fatalf("dry-run update result=%+v", dryRun)
	}
	updated := o.reconcileProfileSource(ctx, "fixture-scn", root, source, profileReconcileModeUpdateIfUnmodified, false)
	if updated.Status != ProfileReconcileStatusUpdated || updated.ProfileID != created.ProfileID {
		t.Fatalf("update result=%+v", updated)
	}

	persisted, err := o.profiles.GetByKey(ctx, "fixture-scn/default")
	if err != nil {
		t.Fatal(err)
	}
	persisted.LocalOverride = true
	if err := o.profiles.Update(ctx, persisted); err != nil {
		t.Fatal(err)
	}
	if conflicted := o.reconcileProfileSource(ctx, "fixture-scn", root, source, profileReconcileModeUpdateIfUnmodified, false); conflicted.Status != ProfileReconcileStatusConflictedLocalOverride {
		t.Fatalf("local override result=%+v", conflicted)
	}
	if forced := o.reconcileProfileSource(ctx, "fixture-scn", root, source, profileReconcileModeForce, false); forced.Status != ProfileReconcileStatusUpdated {
		t.Fatalf("force result=%+v", forced)
	}

	for _, tc := range []struct {
		name   string
		source string
		body   string
		want   ProfileReconcileStatus
	}{
		{name: "parent traversal", source: "../escape.json", want: ProfileReconcileStatusFailedValidation},
		{name: "directory", source: ".vrooli/agent-manager", want: ProfileReconcileStatusFailedValidation},
		{name: "foreign owner", source: ".vrooli/agent-manager/foreign.json", body: strings.Replace(fixtureProfile, "fixture-scn/default", "other/default", 1), want: ProfileReconcileStatusFailedValidation},
		{name: "runtime field", source: ".vrooli/agent-manager/runtime.json", body: strings.Replace(fixtureProfile, "{", `{"id":"forbidden",`, 1), want: ProfileReconcileStatusFailedValidation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.body != "" {
				candidate := filepath.Join(root, filepath.FromSlash(tc.source))
				if err := os.WriteFile(candidate, []byte(tc.body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			result := o.reconcileProfileSource(ctx, "fixture-scn", root, tc.source, profileReconcileModeUpdateIfUnmodified, false)
			if result.Status != tc.want || result.Message == "" {
				t.Fatalf("result=%+v want=%s", result, tc.want)
			}
		})
	}
}
