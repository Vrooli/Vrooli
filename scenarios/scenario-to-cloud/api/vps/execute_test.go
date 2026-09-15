package vps

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/faultinject"
	"scenario-to-cloud/fixtures"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

func stubHealth(t *testing.T) {
	t.Helper()
	prevOrigin, prevPublic := checkOriginHealthFunc, checkPublicHealthFunc
	checkOriginHealthFunc = func(context.Context, string, string, time.Duration) error { return nil }
	checkPublicHealthFunc = func(context.Context, string, time.Duration) error { return nil }
	t.Cleanup(func() { checkOriginHealthFunc, checkPublicHealthFunc = prevOrigin, prevPublic })
}

func fixtureManifest(w *fixtures.Workload) domain.CloudManifest {
	ports := domain.ManifestPorts{}
	for i, listener := range w.Declaration.Listeners {
		ports[listener.Name] = 3000 + i
	}
	if _, ok := ports["ui"]; !ok {
		ports["ui"] = 3000
	}
	return domain.CloudManifest{
		Version:  "1",
		Target:   domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		Scenario: domain.ManifestScenario{ID: w.Declaration.PrimaryScenario},
		Dependencies: domain.ManifestDependencies{
			Resources: append([]string(nil), w.Declaration.Resources...),
			Scenarios: append(append([]string(nil), w.Declaration.Scenarios...), w.Declaration.PrimaryScenario),
		},
		Ports: ports,
		Edge:  domain.ManifestEdge{Domain: w.ID + ".example.test", Caddy: domain.ManifestCaddy{Enabled: true}},
	}
}

// fixtureClosure projects the fixture declaration onto a closure with its
// persistent data and recovery contract.
func fixtureClosure(w *fixtures.Workload) *domain.Closure {
	c := &domain.Closure{SchemaVersion: "1", ScenarioID: w.Declaration.PrimaryScenario, Digest: "sha256:fixture-" + w.ID}
	c.Components = append(c.Components, domain.ClosureComponent{ID: w.Declaration.PrimaryScenario, Kind: domain.ClosureKindScenario, Required: true, Recovery: &domain.ClosureRecovery{CodeRollback: "any_predecessor", SchemaStrategy: "expand_contract"}})
	for _, data := range w.Declaration.PersistentData {
		binding := "database:" + data.Database
		if data.Kind == "files" {
			binding = "dir:" + data.Mount
		}
		c.PersistentData = append(c.PersistentData, domain.ClosurePersistentData{ID: data.Binding, Owner: w.Declaration.PrimaryScenario, Binding: binding, MigrationOwner: "scenario", DeclaredBy: "scenario:" + w.Declaration.PrimaryScenario})
	}
	return c
}

func loadFixtures(t *testing.T) *fixtures.Catalog {
	t.Helper()
	cat, err := fixtures.Load(filepath.Join("..", "..", "fixtures"))
	if err != nil {
		t.Fatalf("load fixtures: %v", err)
	}
	return cat
}

func sortedWorkloads(cat *fixtures.Catalog) []*fixtures.Workload {
	ids := make([]string, 0, len(cat.Workloads))
	for id := range cat.Workloads {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]*fixtures.Workload, 0, len(ids))
	for _, id := range ids {
		out = append(out, cat.Workloads[id])
	}
	return out
}

func targetRef() identity.TargetRef {
	return identity.TargetRef{MachineID: "fake-target", Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}}
}

type harness struct {
	t        *testing.T
	target   *fakeTarget
	manifest domain.CloudManifest
	closure  *domain.Closure
	bundle   string
	release  string
	depID    string
}

func newHarness(t *testing.T, target *fakeTarget, w *fixtures.Workload, content string) *harness {
	t.Helper()
	bundle, release := fixtureRelease(t, content)
	return &harness{t: t, target: target, manifest: fixtureManifest(w), closure: fixtureClosure(w), bundle: bundle, release: release, depID: "dep-" + w.ID}
}

func (h *harness) compile(scope string, obs execplan.Observations) *execplan.Plan {
	h.t.Helper()
	plan, err := CompilePlan(context.Background(), PlanRequest{Manifest: h.manifest, BundlePath: h.bundle, Closure: h.closure, Scope: scope, Observations: obs, Deployment: &domain.Deployment{ID: h.depID, ScenarioID: h.manifest.Scenario.ID, Environment: "certification", Target: targetRef()}})
	if err != nil {
		h.t.Fatalf("compile %s: %v", scope, err)
	}
	return plan
}

func (h *harness) execute(ctx context.Context, plan *execplan.Plan, id Identity) (Trace, *ExecutionError) {
	h.target.scenario = h.manifest.Scenario.ID
	return ExecutePlan(ctx, ExecuteRequest{Plan: plan, Manifest: h.manifest, BundlePath: h.bundle, DeploymentID: h.depID, Runtime: Runtime{Reach: h.target, Target: targetRef(), Identity: id, Credentials: fakeCredentials{}, Backups: recordingBackups{}}})
}

type fakeCredentials struct{}

func (fakeCredentials) Provision(context.Context, CredentialProvisionRequest) (CredentialProvisionResult, error) {
	return CredentialProvisionResult{Materialized: []string{"cb_fixture"}}, nil
}

func (fakeCredentials) Revoke(context.Context, CredentialProvisionRequest) (CredentialProvisionResult, error) {
	return CredentialProvisionResult{Revoked: []string{"cb_fixture"}}, nil
}

type recordingBackups struct{}

func (recordingBackups) Record(_ context.Context, _ string, id Identity, _ json.RawMessage, _ string) (string, error) {
	return id.OperationID + "-data.backup", nil
}

func indexOf(list []string, prefix string) int {
	for i, v := range list {
		if strings.HasPrefix(v, prefix) {
			return i
		}
	}
	return -1
}

// [REQ:STC-P0-016] [REQ:STC-P0-028] For every fixture and scope the executed
// action sequence equals the previewed one, every transport call belongs to
// a previewed action, and every argument the target receives passes the
// argv policy: no shell syntax rides in any command.
func TestPreviewAndApplyAgreeForEveryFixture(t *testing.T) {
	stubHealth(t)
	cat := loadFixtures(t)
	for _, w := range sortedWorkloads(cat) {
		t.Run(w.ID, func(t *testing.T) {
			for _, scope := range []string{execplan.ScopeInstall, execplan.ScopeRuntime, execplan.ScopeStart, execplan.ScopeRetire} {
				target := newFakeTarget()
				h := newHarness(t, target, w, w.ID)
				obs := execplan.Observations{}
				if scope == execplan.ScopeStart || scope == execplan.ScopeRuntime {
					// A start resumes a recorded release and a runtime plan
					// activates a staged one: deploy once first.
					if _, execErr := h.execute(context.Background(), h.compile(execplan.ScopeFull, obs), Identity{OperationID: "op-0", Fence: 1}); execErr != nil {
						t.Fatalf("seed deploy: %v (action %s)", execErr, execErr.ActionID)
					}
				}
				if scope == execplan.ScopeRetire {
					h.closure.PersistentData = nil
				}
				previewed := h.compile(scope, obs)
				preview := RenderPreview(previewed, h.manifest)
				if len(preview.Changes) != len(previewed.Actions) {
					t.Fatalf("%s: preview must list every action", scope)
				}
				applied := h.compile(scope, obs)
				if applied.MustDigest() != previewed.MustDigest() {
					t.Fatalf("%s: apply digest differs from preview", scope)
				}
				trace, execErr := h.execute(context.Background(), applied, Identity{OperationID: "op-1", Fence: 2})
				if execErr != nil {
					t.Fatalf("%s: execute: %v (action %s)", scope, execErr, execErr.ActionID)
				}
				executed := make([]string, 0, len(trace.Actions))
				for _, result := range trace.Actions {
					if result.Status != "succeeded" {
						t.Fatalf("%s: action %s status %s: %s", scope, result.ID, result.Status, result.Error)
					}
					executed = append(executed, result.ID)
				}
				if strings.Join(executed, ",") != strings.Join(previewed.ActionIDs(), ",") {
					t.Fatalf("%s: executed %v != previewed %v", scope, executed, previewed.ActionIDs())
				}
				declared := map[string]bool{}
				for _, id := range previewed.ActionIDs() {
					declared[id] = true
				}
				if len(trace.Calls) == 0 {
					t.Fatalf("%s: expected transport calls", scope)
				}
				for _, call := range trace.Calls {
					if !declared[call.ActionID] {
						t.Fatalf("%s: undeclared effect: %s %q outside any action", scope, call.Kind, call.Command)
					}
					if call.Kind == "exec" {
						if err := reach.ValidateCommand(reach.Command{Verb: call.Verb, Args: call.Args}); err != nil {
							t.Fatalf("%s: %s dispatched shell-unsafe argv %q: %v", scope, call.ActionID, call.Command, err)
						}
					}
					if strings.Contains(call.Command, "rm -rf") || strings.Contains(call.Command, "&&") || strings.Contains(call.Command, "pkill") {
						t.Fatalf("%s: shell fragment reached the transport: %q", scope, call.Command)
					}
				}
			}
		})
	}
}

// [REQ:STC-P0-028] Every compiled action of every fixture derives typed
// invocations that pass the reach argument policy; the shell preview is
// rendered from those invocations and is never what runs.
func TestActionCommandsCarryNoShellSyntaxForEveryFixture(t *testing.T) {
	cat := loadFixtures(t)
	for _, w := range sortedWorkloads(cat) {
		h := newHarness(t, newFakeTarget(), w, w.ID)
		for _, scope := range []string{execplan.ScopeFull, execplan.ScopeStart} {
			plan := h.compile(scope, execplan.Observations{ActiveReleaseDigest: "sha256:" + strings.Repeat("9", 64), DataSchemaVersion: "v1", LegacyDataInventory: []string{"scenarios/" + h.manifest.Scenario.ID + "/cache"}})
			cc := CommandContext{DeploymentID: h.depID, ScenarioID: h.manifest.Scenario.ID, Identity: Identity{OperationID: "op-1", Fence: 3}, Manifest: h.manifest}
			for _, action := range plan.Actions {
				commands, err := ActionCommands(action, cc)
				if err != nil {
					t.Fatalf("%s/%s %s: %v", w.ID, scope, action.ID, err)
				}
				for _, tc := range commands {
					if err := reach.ValidateCommand(tc.Command); err != nil {
						t.Fatalf("%s/%s %s: %v: %q", w.ID, scope, action.ID, err, tc.Command.Argv())
					}
					if tc.Command.Effectful && !strings.Contains(strings.Join(tc.Command.Args, " "), "--fence 3") && strings.HasPrefix(tc.Command.Verb, "cloud-target") {
						t.Fatalf("%s/%s %s: effectful verb without the fence: %q", w.ID, scope, action.ID, tc.Command.Argv())
					}
				}
			}
			preview := RenderPreview(plan, h.manifest)
			if len(preview.ShellPreview) == 0 {
				t.Fatalf("%s/%s: preview must render the derived shell lines", w.ID, scope)
			}
		}
	}
}

// [REQ:STC-P0-028] P11-A02/A04: activation commits runtime before the
// pointer and the route switch follows a committed pointer; a crash before
// the verb, after the verb, or inside the target between runtime and pointer
// is reconciled from the target's durable state and the final active release
// is the candidate.
func TestActivationOrderingAndCrashRecovery(t *testing.T) {
	stubHealth(t)
	cat := loadFixtures(t)
	w := cat.Workloads["sql-uploads"]

	t.Run("ordering runtime then pointer then route", func(t *testing.T) {
		target := newFakeTarget()
		h := newHarness(t, target, w, "v1")
		if _, execErr := h.execute(context.Background(), h.compile(execplan.ScopeFull, execplan.Observations{}), Identity{OperationID: "op-1", Fence: 1}); execErr != nil {
			t.Fatalf("deploy: %v (%s)", execErr, execErr.ActionID)
		}
		events := target.events
		runtime, pointer, route := indexOf(events, "runtime:"), indexOf(events, "pointer:"), indexOf(events, "route:")
		stage, backup := indexOf(events, "stage:"), indexOf(events, "backup")
		if stage < 0 || backup < 0 || runtime < 0 || pointer < 0 || route < 0 {
			t.Fatalf("events = %v", events)
		}
		if !(stage < backup && backup < runtime && runtime < pointer && pointer < route) {
			t.Fatalf("activation ordering violated: %v", events)
		}
		if target.active == nil || target.active.Active != h.release || target.routes[h.manifest.Edge.Domain] != h.manifest.Ports["ui"] {
			t.Fatalf("runtime and routing must identify the same release: active=%+v routes=%v", target.active, target.routes)
		}
		if target.backups != 1 {
			t.Fatalf("expected one recovery point before activation, got %d", target.backups)
		}
	})

	for _, crash := range []string{"executor:before_switch", "executor:after_switch", "activate:after_runtime", "activate:before_runtime"} {
		t.Run("crash "+crash, func(t *testing.T) {
			target := newFakeTarget()
			v1 := newHarness(t, target, w, "v1")
			if _, execErr := v1.execute(context.Background(), v1.compile(execplan.ScopeFull, execplan.Observations{}), Identity{OperationID: "op-1", Fence: 1}); execErr != nil {
				t.Fatalf("v1 deploy: %v", execErr)
			}
			v2 := newHarness(t, target, w, "v2")
			obs := execplan.Observations{ActiveReleaseDigest: "sha256:" + strings.Repeat("1", 64), DataSchemaVersion: "v1"}
			plan := v2.compile(execplan.ScopeFull, obs)
			ctx := database.WithTestMode(context.Background())
			reg := faultinject.New()
			reg.SetCrashFunc(func(*faultinject.Fault) {})
			ctx = faultinject.WithRegistry(ctx, reg)
			switch crash {
			case "executor:before_switch":
				if err := reg.Arm(ctx, faultinject.ActivationBeforeSwitch, faultinject.Behaviour{Kind: faultinject.KindFail, Once: true}); err != nil {
					t.Fatal(err)
				}
			case "executor:after_switch":
				if err := reg.Arm(ctx, faultinject.ActivationAfterSwitch, faultinject.Behaviour{Kind: faultinject.KindFail, Once: true}); err != nil {
					t.Fatal(err)
				}
			default:
				target.crashAt = crash
			}
			_, execErr := v2.execute(ctx, plan, Identity{OperationID: "op-2", Fence: 2})
			if execErr == nil || execErr.ActionID != execplan.OpReleaseActivate {
				t.Fatalf("expected the crash at release.activate, got %v", execErr)
			}
			listing := target.listing(v2.depID)
			switch crash {
			case "executor:before_switch", "activate:before_runtime":
				if target.active.Active != v1.release {
					t.Fatalf("prior release must stay active before the switch: %+v", target.active)
				}
			case "activate:after_runtime":
				if target.active.Active != v1.release || listing["interrupted_activation"] == nil {
					t.Fatalf("a crash between runtime and pointer must leave the prior pointer and the intent: active=%+v listing=%v", target.active, listing)
				}
			case "executor:after_switch":
				if target.active.Active != v2.release {
					t.Fatalf("the target committed before the executor crashed; pointer must be the candidate: %+v", target.active)
				}
			}
			// Restart reconciliation: a successor attempt (new fence) replays
			// the plan; the target's receipts and pointer decide what runs.
			successor := Identity{OperationID: "op-2", Fence: 3}
			if crash == "activate:after_runtime" || crash == "activate:before_runtime" {
				// The target recorded a failed receipt for op-2; a new operation
				// re-invokes activation under the same reviewed plan.
				successor = Identity{OperationID: "op-3", Fence: 3}
			}
			if _, execErr := v2.execute(context.Background(), plan, successor); execErr != nil {
				t.Fatalf("successor attempt: %v (%s)", execErr, execErr.ActionID)
			}
			if target.active == nil || target.active.Active != v2.release || target.active.Previous != v1.release || target.intent != nil {
				t.Fatalf("restart reconciliation must end with the candidate active and the predecessor retained: active=%+v intent=%+v", target.active, target.intent)
			}
			if count := strings.Count(strings.Join(target.events, ","), "pointer:"+v2.release); count != 1 {
				t.Fatalf("the candidate pointer must be committed exactly once, got %d: %v", count, target.events)
			}
		})
	}
}

// [REQ:STC-P0-028] P11-A05/P12: seeded data bound to a declared binding
// survives a code update and a failed activation; the persistent root is
// never part of a release tree.
func TestSeededDataSurvivesUpdateAndFailedActivation(t *testing.T) {
	stubHealth(t)
	target := newFakeTarget()
	w := loadFixtures(t).Workloads["sql-uploads"]
	scenario := w.Declaration.PrimaryScenario
	target.legacy[scenario+"/uploads"] = map[string]string{"invoice-001.txt": "seeded upload", "photo-002.txt": "seeded photo"}
	v1 := newHarness(t, target, w, "v1")
	if _, execErr := v1.execute(context.Background(), v1.compile(execplan.ScopeFull, execplan.Observations{}), Identity{OperationID: "op-1", Fence: 1}); execErr != nil {
		t.Fatalf("v1: %v", execErr)
	}
	if files := target.persistent["uploads"]; len(files) != 2 || files["invoice-001.txt"] != "seeded upload" {
		t.Fatalf("seed must be adopted into the persistent root by the first activation: %v", target.persistent)
	}
	target.persistent["uploads"]["notes-003.txt"] = "written while v1 ran"
	obs := execplan.Observations{ActiveReleaseDigest: "sha256:" + strings.Repeat("1", 64), DataSchemaVersion: "v1"}
	v2 := newHarness(t, target, w, "v2")
	target.failNext = v2.release
	if _, execErr := v2.execute(context.Background(), v2.compile(execplan.ScopeFull, obs), Identity{OperationID: "op-2", Fence: 2}); execErr == nil || execErr.ActionID != execplan.OpReleaseActivate {
		t.Fatalf("expected the failed candidate to stop at release.activate, got %v", execErr)
	}
	if target.active.Active != v1.release {
		t.Fatalf("prior release must remain active after a failed candidate: %+v", target.active)
	}
	if files := target.persistent["uploads"]; len(files) != 3 || files["notes-003.txt"] != "written while v1 ran" {
		t.Fatalf("data lost after failed activation: %v", files)
	}
	if target.backups != 2 {
		t.Fatalf("every update must capture a recovery point before touching the runtime; got %d", target.backups)
	}
	if _, execErr := v2.execute(context.Background(), v2.compile(execplan.ScopeFull, obs), Identity{OperationID: "op-3", Fence: 3}); execErr != nil {
		t.Fatalf("v2 retry: %v (%s)", execErr, execErr.ActionID)
	}
	if target.active.Active != v2.release || target.active.Previous != v1.release {
		t.Fatalf("update must activate v2 and retain v1: %+v", target.active)
	}
	if files := target.persistent["uploads"]; len(files) != 3 {
		t.Fatalf("data lost across the update: %v", files)
	}
	if target.bound[scenario+"/uploads"] != "uploads" {
		t.Fatalf("release tree must bind uploads to the persistent root: %v", target.bound)
	}
}

// [REQ:STC-P0-028] P11-A03/P15-A02: two deployments sharing postgres on one
// host; stopping (and retiring) one keeps the resource the other demands.
func TestSharedResourceDemandSurvivesSiblingStop(t *testing.T) {
	stubHealth(t)
	target := newFakeTarget()
	cat := loadFixtures(t)
	a := newHarness(t, target, cat.Workloads["sql-uploads"], "a")
	b := newHarness(t, target, cat.Workloads["dependent-chain"], "b")
	b.manifest.Dependencies.Resources = []string{"postgres"}
	b.depID = "dep-b"
	if _, execErr := b.execute(context.Background(), b.compile(execplan.ScopeFull, execplan.Observations{}), Identity{OperationID: "op-b", Fence: 1}); execErr != nil {
		t.Fatalf("survivor deploy: %v (%s)", execErr, execErr.ActionID)
	}
	if _, execErr := a.execute(context.Background(), a.compile(execplan.ScopeFull, execplan.Observations{}), Identity{OperationID: "op-a", Fence: 2}); execErr != nil {
		t.Fatalf("subject deploy: %v (%s)", execErr, execErr.ActionID)
	}
	if !target.resources["postgres"] || len(target.demand["postgres"]) != 2 {
		t.Fatalf("shared resource must be started once and demanded twice: running=%v demand=%v", target.resources, target.demand)
	}
	if strings.Count(strings.Join(target.events, ","), "resource:start:postgres") != 2 {
		t.Fatalf("events = %v", target.events)
	}
	a.closure.PersistentData = nil
	retire := a.compile(execplan.ScopeRetire, execplan.Observations{})
	if _, execErr := a.execute(context.Background(), retire, Identity{OperationID: "op-a-retire", Fence: 3}); execErr != nil {
		t.Fatalf("retire subject: %v (%s)", execErr, execErr.ActionID)
	}
	if target.running[a.manifest.Scenario.ID] {
		t.Fatal("subject must be stopped")
	}
	if !target.running[b.manifest.Scenario.ID] || !target.resources["postgres"] || !target.demand["postgres"][b.manifest.Scenario.ID] {
		t.Fatalf("survivor and its shared resource must keep running: running=%v resources=%v demand=%v", target.running, target.resources, target.demand)
	}
	if target.routes[a.manifest.Edge.Domain] != 0 || target.routes[b.manifest.Edge.Domain] == 0 {
		t.Fatalf("subject route removed, survivor route present: %v", target.routes)
	}
	for _, event := range target.events {
		if event == "resource:stop:postgres" {
			t.Fatalf("shared resource must not be stopped while the survivor demands it: %v", target.events)
		}
	}
}

// [REQ:STC-P0-028] P11-A06: legacy conversion. Recorded mappings bind their
// directories; the heuristic applies only to unmapped ones and nothing
// unmapped is deleted.
func TestLegacyConversionRefusesDeletionOfUnmapped(t *testing.T) {
	stubHealth(t)
	target := newFakeTarget()
	w := loadFixtures(t).Workloads["stateless-web"]
	scenario := w.Declaration.PrimaryScenario
	target.legacy[scenario+"/api/uploads"] = map[string]string{"a.bin": "1"}
	target.legacy[scenario+"/data"] = map[string]string{"records.csv": "2"}
	target.legacy[scenario+"/cache"] = map[string]string{"hot": "3"}
	h := newHarness(t, target, w, "legacy")
	h.manifest.Target.VPS.PreservePaths = []string{"scenarios/" + scenario + "/api/uploads", "scenarios/" + scenario + "/data"}
	obs := execplan.Observations{PersistentDataBindings: []string{"uploads=" + scenario + "/api/uploads"}}
	plan := h.compile(execplan.ScopeFull, obs)
	activate := plan.Action(execplan.OpReleaseActivate)
	if activate.Inputs["data_bindings"] != "uploads="+scenario+"/api/uploads" {
		t.Fatalf("recorded mapping must bind first: %q", activate.Inputs["data_bindings"])
	}
	if activate.Inputs["legacy_carry"] != scenario+"/data" {
		t.Fatalf("the heuristic must apply only to unmapped legacy paths: %q", activate.Inputs["legacy_carry"])
	}
	if _, execErr := h.execute(context.Background(), plan, Identity{OperationID: "op-1", Fence: 1}); execErr != nil {
		t.Fatalf("execute: %v (%s)", execErr, execErr.ActionID)
	}
	if target.persistent["uploads"]["a.bin"] != "1" || target.persistent["legacy:"+scenario+"/data"]["records.csv"] != "2" {
		t.Fatalf("mapped and carried data must be adopted: %v", target.persistent)
	}
	if target.legacy[scenario+"/cache"]["hot"] != "3" {
		t.Fatalf("unmapped legacy data must never be deleted: %v", target.legacy)
	}
	if len(target.unmapped) != 1 || target.unmapped[0] != "legacy:"+scenario+"/cache" {
		t.Fatalf("unmapped report = %v", target.unmapped)
	}
}

// [REQ:STC-P0-028] A start of the recorded release goes through the target
// owner with pinned ports; an interrupted activation refuses the start
// until it is reconciled, and a plan naming a release that is not the
// active one is stale.
func TestWorkloadStartUsesActivePointerAndRefusesInterruptedActivation(t *testing.T) {
	stubHealth(t)
	target := newFakeTarget()
	w := loadFixtures(t).Workloads["stateless-web"]
	h := newHarness(t, target, w, "v1")
	if _, execErr := h.execute(context.Background(), h.compile(execplan.ScopeFull, execplan.Observations{}), Identity{OperationID: "op-1", Fence: 1}); execErr != nil {
		t.Fatalf("deploy: %v", execErr)
	}
	start := h.compile(execplan.ScopeStart, execplan.Observations{})
	if _, execErr := h.execute(context.Background(), start, Identity{OperationID: "op-2", Fence: 2}); execErr != nil {
		t.Fatalf("start: %v (%s)", execErr, execErr.ActionID)
	}
	if indexOf(target.events, "port:"+w.Declaration.PrimaryScenario+":ui=3000") < 0 {
		t.Fatalf("start must pin the listener ports through the owner: %v", target.events)
	}
	target.intent = &fakeIntent{Candidate: "x", Previous: h.release}
	if _, execErr := h.execute(context.Background(), start, Identity{OperationID: "op-3", Fence: 3}); execErr == nil || execErr.ActionID != execplan.OpWorkloadStart {
		t.Fatalf("interrupted activation must refuse the start: %v", execErr)
	}
	target.intent = nil
	other := newHarness(t, target, w, "v2")
	if _, execErr := other.execute(context.Background(), other.compile(execplan.ScopeStart, execplan.Observations{}), Identity{OperationID: "op-4", Fence: 4}); execErr == nil || !strings.Contains(execErr.Error(), "stale") {
		t.Fatalf("a start plan naming a non-active release must be stale: %v", execErr)
	}
}

// [REQ:STC-P0-016] Execution refuses needs_input plans and a verification
// mismatch stops before any later effect.
func TestExecutePlanRefusesNeedsInputAndStopsOnVerifyMismatch(t *testing.T) {
	stubHealth(t)
	target := newFakeTarget()
	w := loadFixtures(t).Workloads["stateless-web"]
	h := newHarness(t, target, w, "v1")
	h.manifest.Secrets = &domain.ManifestSecrets{BundleSecrets: []domain.BundleSecretPlan{{ID: "api-key", Class: "user_prompt", Required: true, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/app", Field: "api-key"}}}}
	plan := h.compile(execplan.ScopeInstall, execplan.Observations{})
	if plan.Outcome != execplan.OutcomeNeedsInput {
		t.Fatalf("expected needs_input, got %s", plan.Outcome)
	}
	trace, execErr := h.execute(context.Background(), plan, Identity{OperationID: "op-1", Fence: 1})
	if execErr == nil || !strings.Contains(execErr.Error(), "operator input") {
		t.Fatalf("expected needs_input refusal, got %v", execErr)
	}
	if len(trace.Calls) != 0 || len(target.calls) != 0 {
		t.Fatalf("needs_input must cause zero transport calls: %v", target.calls)
	}

	h.manifest.Secrets = nil
	plan = h.compile(execplan.ScopeInstall, execplan.Observations{})
	// The target never received the manifest (delivery is skipped by a hook),
	// so verification cannot pass and staging must not run.
	hooks := Hooks{Before: func(_ context.Context, action execplan.Action) (bool, error) {
		return action.ID == execplan.OpReleaseDeliver, nil
	}}
	trace, execErr = ExecutePlan(context.Background(), ExecuteRequest{Plan: plan, Manifest: h.manifest, BundlePath: h.bundle, DeploymentID: h.depID, Runtime: Runtime{Reach: target, Target: targetRef(), Identity: Identity{OperationID: "op-2", Fence: 2}}, Hooks: hooks})
	if execErr == nil || execErr.ActionID != execplan.OpReleaseVerify {
		t.Fatalf("expected release.verify to fail, got %v", execErr)
	}
	for _, call := range trace.Calls {
		if call.ActionID == execplan.OpReleaseStage || call.ActionID == execplan.OpReleaseActivate {
			t.Fatalf("no effect may follow a failed verification: %+v", call)
		}
	}
}

// [REQ:STC-P0-016] A transport call outside any action is refused.
func TestAttributedReachRefusesUndeclaredEffects(t *testing.T) {
	target := newFakeTarget()
	wrapped := &attributedReach{inner: target}
	if _, err := wrapped.Exec(context.Background(), targetRef(), reach.Command{Verb: "cloud-target release list", Args: []string{"--deployment", "dep"}}); err != ErrUndeclaredEffect {
		t.Fatalf("expected ErrUndeclaredEffect, got %v", err)
	}
	if _, err := wrapped.Deliver(context.Background(), targetRef(), reach.Delivery{Files: []reach.ArtifactFile{{LocalPath: "a", RemotePath: "/b"}}}); err != ErrUndeclaredEffect {
		t.Fatalf("expected ErrUndeclaredEffect for deliver, got %v", err)
	}
	if len(target.calls) != 0 {
		t.Fatalf("inner reach must not be reached: %v", target.calls)
	}
	wrapped.enter("host.prepare")
	if _, err := wrapped.Exec(context.Background(), targetRef(), reach.Command{Verb: "cloud-target release list", Args: []string{"--deployment", "dep", "--json"}}); err != nil {
		t.Fatalf("attributed call failed: %v", err)
	}
	wrapped.leave()
	if calls := wrapped.Calls(); len(calls) != 1 || calls[0].ActionID != "host.prepare" {
		t.Fatalf("expected one attributed call, got %+v", calls)
	}
}
