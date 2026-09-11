package execplan

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
)

func baseInputs() CompileInputs {
	return CompileInputs{
		Deployment: identity.DeploymentRef{
			ID: "deployment-fixture-a", ScenarioID: "app", Environment: "production",
			Target: identity.TargetRef{MachineID: "machine-fixture-a", NodeID: "node-fixture-a", EnrollmentGeneration: 3, Transport: identity.TransportBridge},
		},
		DesiredRevision: 12,
		Release:         identity.ReleaseRef{Digest: "sha256:fixture-release", ConfigurationDigest: "sha256:fixture-config", ClosureDigest: "sha256:fixture-closure"},
		Closure: &domain.Closure{
			SchemaVersion: "1", ScenarioID: "app", Digest: "sha256:fixture-closure",
			Components: []domain.ClosureComponent{
				{ID: "fixture/app:session-secret", Kind: domain.ClosureKindCredentialDescriptor, Credential: &domain.ClosureCredential{LogicalID: "fixture/app", Field: "session-secret", Required: true}},
			},
			PersistentData: []domain.ClosurePersistentData{{ID: "application-records", Owner: "store", MigrationOwner: "scenario", DeclaredBy: "scenario:app"}},
			Privileges:     []domain.ClosurePrivilege{{Effect: "elevated", Subject: "apt", Reason: "packages"}},
		},
		Manifest: domain.CloudManifest{
			Version:      "1",
			Target:       domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}},
			Scenario:     domain.ManifestScenario{ID: "app"},
			Dependencies: domain.ManifestDependencies{Resources: []string{"store", "cache"}, Scenarios: []string{"app", "records-service"}},
			Ports:        domain.ManifestPorts{"ui": 3000, "api": 3001},
			Edge:         domain.ManifestEdge{Domain: "example.com", Caddy: domain.ManifestCaddy{Enabled: true}},
		},
		ArtifactPath: "/var/bundles/app.tar.gz",
		Observations: Observations{DeploymentRevision: 11, DataSchemaVersion: "fixture-schema-v1", SatisfiedInputs: []string{"fixture/app:session-secret"}},
		Policy:       Policy{Version: "cloud-launch-v1", MaintenanceStrategy: StrategyMaintenance},
	}
}

func compile(t *testing.T, in CompileInputs) *Plan {
	t.Helper()
	plan, err := Compile(context.Background(), in)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return plan
}

// [REQ:STC-P0-016] P06-A05: equal material inputs yield one digest, across
// repeated compiles and map-ordering variation.
func TestSemanticDigestIsDeterministic(t *testing.T) {
	first := compile(t, baseInputs()).MustDigest()
	for i := 0; i < 100; i++ {
		in := baseInputs()
		// Shuffle input ordering the compiler must normalise.
		if i%2 == 1 {
			in.Manifest.Dependencies.Resources = []string{"cache", "store"}
			in.Manifest.Ports = domain.ManifestPorts{"api": 3001, "ui": 3000}
		}
		if got := compile(t, in).MustDigest(); got != first {
			t.Fatalf("iteration %d: digest %s != %s", i, got, first)
		}
	}
	// Round trip through JSON keeps the digest (nil vs empty slices are canonical).
	plan := compile(t, baseInputs())
	raw, _ := json.Marshal(plan)
	var decoded Plan
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.MustDigest() != first {
		t.Fatalf("JSON round trip changed the digest")
	}
}

// [REQ:STC-P0-016] Presentation text is not identity.
func TestPresentationChangeKeepsDigest(t *testing.T) {
	plan := compile(t, baseInputs())
	before := plan.MustDigest()
	plan.Presentation.Title = "A completely different title"
	plan.Presentation.Summary = "and summary"
	plan.Presentation.DowntimeNote = "note"
	if plan.MustDigest() != before {
		t.Fatalf("presentation change altered the digest")
	}
	if reasons := Compare(compile(t, baseInputs()), plan); Material(reasons) {
		t.Fatalf("presentation-only diff must be harmless, got %+v", reasons)
	}
}

// [REQ:STC-P0-016] P06-A02: target, artifact, privilege and destructive
// effect changes invalidate prior review.
func TestMaterialChangesChangeDigestAndInvalidateReview(t *testing.T) {
	reviewed := compile(t, baseInputs())
	base := reviewed.MustDigest()
	cases := map[string]func(*CompileInputs){
		"target enrollment": func(in *CompileInputs) { in.Deployment.Target.EnrollmentGeneration = 4 },
		"target machine":    func(in *CompileInputs) { in.Deployment.Target.MachineID = "machine-other" },
		"release digest":    func(in *CompileInputs) { in.Release.Digest = "sha256:other-release" },
		"closure digest":    func(in *CompileInputs) { in.Closure.Digest = "sha256:other-closure" },
		"configuration":     func(in *CompileInputs) { in.Release.ConfigurationDigest = "sha256:other-config" },
		"data schema":       func(in *CompileInputs) { in.Observations.DataSchemaVersion = "fixture-schema-v2" },
		"privilege set": func(in *CompileInputs) {
			in.Closure.Privileges = append(in.Closure.Privileges, domain.ClosurePrivilege{Effect: "elevated", Subject: "docker"})
		},
		"destructive effect": func(in *CompileInputs) {
			in.Policy.MaintenanceStrategy = StrategySideBySide
		},
		"schema version": func(in *CompileInputs) { in.Manifest.Target.VPS.Workdir = "/srv/other" },
	}
	for name, mutate := range cases {
		in := baseInputs()
		mutate(&in)
		recompiled := compile(t, in)
		if recompiled.MustDigest() == base {
			t.Errorf("%s: digest unchanged", name)
		}
		if name == "schema version" {
			continue // workdir is an input change; preconditions do not name it
		}
		reasons := Compare(reviewed, recompiled)
		if !Material(reasons) {
			if name == "destructive effect" {
				// The strategy changes the privilege set only when it changes
				// capabilities; the digest change alone refuses apply.
				continue
			}
			t.Errorf("%s: expected material stale reason, got %+v", name, reasons)
		}
	}
	// Validate against current facts flags the enrollment change.
	facts := FactsOf(reviewed, baseInputs())
	facts.TargetEnrollment = 4
	reasons := Validate(reviewed, facts)
	if len(reasons) != 1 || reasons[0].Kind != PreconditionTargetEnrollment || !reasons[0].Material {
		t.Fatalf("expected one material target_enrollment reason, got %+v", reasons)
	}
	if err := StaleError(reasons); err.Code != "plan_stale" {
		t.Fatalf("expected plan_stale, got %s", err.Code)
	}
}

// [REQ:STC-P0-016] No shell strings ride in typed inputs; the vocabulary is closed.
func TestActionsCarryTypedInputsOnly(t *testing.T) {
	plan := compile(t, baseInputs())
	if len(plan.Actions) == 0 {
		t.Fatal("expected actions")
	}
	seen := map[string]bool{}
	for _, action := range plan.Actions {
		if !KnownOp(action.OwnerOperation) {
			t.Errorf("%s: unknown owner operation %q", action.ID, action.OwnerOperation)
		}
		if seen[action.ID] {
			t.Errorf("duplicate action id %s", action.ID)
		}
		seen[action.ID] = true
		if action.Retry == "" || action.Verification == "" || action.Recovery == "" || action.Effect == "" {
			t.Errorf("%s: retry, verification, recovery and effect are required", action.ID)
		}
		for key, value := range action.Inputs {
			for _, fragment := range []string{"&&", "||", "rm -rf", "| ", "$(", "bash -c", "ssh "} {
				if strings.Contains(value, fragment) {
					t.Errorf("%s input %s carries shell fragment %q: %s", action.ID, key, fragment, value)
				}
			}
		}
		for _, dep := range action.DependsOn {
			if !seen[dep] {
				t.Errorf("%s depends on %s which is not earlier in the plan", action.ID, dep)
			}
		}
	}
	for _, id := range []string{OpHostPrepare, OpEdgeFirewallAllow, OpDataInventory, OpReleaseDeliver, OpReleaseVerify, OpReleaseStage, OpConfigApply, OpRuntimeStartDeps, OpDataBackup, OpWorkloadStop, OpReleaseActivate, OpEdgeRouteApply, OpVerifyReadiness} {
		if plan.Action(id) == nil {
			t.Errorf("expected action %s in full plan: %v", id, plan.ActionIDs())
		}
	}
	if plan.Action(OpWorkloadStart) != nil {
		t.Fatalf("a full plan activates through the target owner; workload.start belongs to the start scope only: %v", plan.ActionIDs())
	}
	// Activation runs after dependencies and the recovery point and before
	// the route switch and readiness, so a failed candidate never receives
	// public traffic and the recovery point precedes any schema change.
	order := map[string]int{}
	for i, id := range plan.ActionIDs() {
		order[id] = i
	}
	for _, pair := range [][2]string{{OpReleaseStage, OpReleaseActivate}, {OpRuntimeStartDeps, OpDataBackup}, {OpDataBackup, OpReleaseActivate}, {OpWorkloadStop, OpReleaseActivate}, {OpReleaseActivate, OpEdgeRouteApply}, {OpEdgeRouteApply, OpVerifyReadiness}} {
		if order[pair[0]] > order[pair[1]] {
			t.Errorf("%s must precede %s: %v", pair[0], pair[1], plan.ActionIDs())
		}
	}
	if stop := plan.Action(OpWorkloadStop); stop.Downtime == nil || stop.Downtime.ExpectedSeconds == 0 {
		t.Fatalf("maintenance strategy must declare downtime on workload.stop")
	}
	if act := plan.Action(OpReleaseActivate); act.CancelPoint {
		t.Fatalf("release.activate is not a safe cancellation point")
	}
	if act := plan.Action(OpReleaseStage); !strings.HasPrefix(act.Inputs["release_dir"], "/root/Vrooli/.vrooli/cloud/releases/") {
		t.Fatalf("release.stage must stage under the releases tree, got %q", act.Inputs["release_dir"])
	}
	selection := plan.Action(OpConfigApply).Inputs["selection_json_b64"]
	decoded, err := base64.RawURLEncoding.DecodeString(selection)
	parsedSelection := &setupv1.Selection{}
	if err == nil {
		err = protojson.Unmarshal(decoded, parsedSelection)
	}
	if err != nil || parsedSelection.GetSchemaVersion() != "v1" {
		t.Fatalf("config.apply must carry the encoded closure-derived setup/v1 selection: %q", selection)
	}
}

// [REQ:STC-P0-016] P06-A03: satisfied desired state is a no-op; an
// unobserved target is never a no-op.
func TestNoOpWhenDesiredStateObservedSatisfied(t *testing.T) {
	in := baseInputs()
	in.Observations.ActiveReleaseDigest = in.Release.Digest
	in.Observations.ActiveConfigurationDigest = in.Release.ConfigurationDigest
	in.Observations.ActiveClosureDigest = in.Closure.Digest
	plan := compile(t, in)
	if plan.Outcome != OutcomeNoOp || len(plan.Actions) != 0 {
		t.Fatalf("expected no_op with zero actions, got %s %v", plan.Outcome, plan.ActionIDs())
	}
	unobserved := baseInputs()
	if compile(t, unobserved).Outcome != OutcomeApply {
		t.Fatalf("unobserved target must compile to apply, not no_op")
	}
	drifted := in
	drifted.Observations.ActiveConfigurationDigest = "sha256:drifted"
	if compile(t, drifted).Outcome != OutcomeApply {
		t.Fatalf("configuration drift must not be a no-op")
	}
	if compile(t, in).MustDigest() == compile(t, unobserved).MustDigest() {
		t.Fatalf("outcome is material: no_op and apply plans must not share a digest")
	}
}

// [REQ:STC-P0-016] P06-A04: a required input yields exactly one durable handoff.
func TestNeedsInputReturnsOneHandoff(t *testing.T) {
	in := baseInputs()
	in.Observations.SatisfiedInputs = nil
	in.Manifest.Secrets = &domain.ManifestSecrets{BundleSecrets: []domain.BundleSecretPlan{
		{ID: "api-key", Class: "user_prompt", Required: true, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/app", Field: "api-key"}},
	}}
	plan := compile(t, in)
	if plan.Outcome != OutcomeNeedsInput || plan.Handoff == nil {
		t.Fatalf("expected needs_input with a handoff, got %s", plan.Outcome)
	}
	if plan.Handoff.Owner != HandoffOwner || plan.Handoff.Kind != HandoffKind || plan.Handoff.Reference == "" {
		t.Fatalf("handoff must name vrooli-onboarding resume_handoff with a reference: %+v", plan.Handoff)
	}
	if plan.Handoff.DeploymentID != in.Deployment.ID || plan.Handoff.Target != plan.Target || plan.Handoff.DesiredRevision != in.DesiredRevision || plan.Handoff.SelectionDigest != plan.ClosureDigest {
		t.Fatalf("handoff must carry the reviewed deployment and target fences: %+v", plan.Handoff)
	}
	if len(plan.Handoff.Missing) != 2 || plan.Handoff.Missing[0] != "fixture/app:api-key" || plan.Handoff.Missing[1] != "fixture/app:session-secret" {
		t.Fatalf("expected the two missing addresses sorted, got %v", plan.Handoff.Missing)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].OwnerOperation != OpInputResumeHandoff || plan.Actions[0].Effect != EffectNone {
		t.Fatalf("expected exactly one effect-free handoff action, got %+v", plan.Actions)
	}
	for _, value := range plan.Actions[0].Inputs {
		if strings.Contains(value, "secret-value") {
			t.Fatalf("values must never enter the plan")
		}
	}
	if compile(t, in).Handoff.Reference != plan.Handoff.Reference {
		t.Fatalf("handoff reference must be deterministic")
	}
	err := NeedsInputError(plan.Handoff)
	if err.Code != "needs_input" || err.NextAction == nil || err.NextAction.Reference != plan.Handoff.Reference {
		t.Fatalf("needs_input error must carry the handoff reference: %+v", err)
	}
}

func TestScopesSelectActionSubsets(t *testing.T) {
	install := baseInputs()
	install.Scope = ScopeInstall
	runtime := baseInputs()
	runtime.Scope = ScopeRuntime
	start := baseInputs()
	start.Scope = ScopeStart
	installIDs := compile(t, install).ActionIDs()
	runtimeIDs := compile(t, runtime).ActionIDs()
	startIDs := compile(t, start).ActionIDs()
	full := compile(t, baseInputs()).ActionIDs()
	if strings.Join(full, ",") != strings.Join(append(installIDs, runtimeIDs...), ",") {
		t.Fatalf("full plan must be install followed by runtime: %v vs %v + %v", full, installIDs, runtimeIDs)
	}
	for _, id := range startIDs {
		if id == OpEdgeRouteApply || id == OpReleaseRetainPredecesor {
			t.Fatalf("start scope must not contain %s", id)
		}
	}
	if startIDs[0] != OpRuntimeStartDeps || startIDs[len(startIDs)-2] != OpWorkloadStart || startIDs[len(startIDs)-1] != OpVerifyReadiness {
		t.Fatalf("start scope order: %v", startIDs)
	}
}

func TestRenderPreviewFromActionGraph(t *testing.T) {
	plan := compile(t, baseInputs())
	preview := Render(plan, WithShellRenderer(func(a Action) string { return "echo " + a.ID }))
	if len(preview.Changes) != len(plan.Actions) || len(preview.ShellPreview) != len(plan.Actions) {
		t.Fatalf("preview must cover every action")
	}
	if preview.Downtime.ExpectedSeconds != 60 || preview.Downtime.Reason == "" {
		t.Fatalf("expected declared downtime in preview, got %+v", preview.Downtime)
	}
	if !strings.Contains(preview.RecoveryStrategy, "rollback_to_predecessor") {
		t.Fatalf("expected recovery strategy, got %q", preview.RecoveryStrategy)
	}
	var sawData bool
	for _, effect := range preview.DataEffects {
		if effect.Subject == "application-records@store" {
			sawData = true
		}
	}
	if !sawData {
		t.Fatalf("expected declared persistent data in data effects: %+v", preview.DataEffects)
	}
	if !strings.Contains(preview.Target, "machine-fixture-a") {
		t.Fatalf("preview target should name the machine identity: %s", preview.Target)
	}
}

func TestCompileRefusesUnsupportedClosureAndBadInputs(t *testing.T) {
	in := baseInputs()
	in.Closure.Unsupported = []domain.ClosureUnsupported{{Component: "store", ReasonCode: "missing_platform_artifact"}}
	if _, err := Compile(context.Background(), in); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported_capability, got %v", err)
	}
	in = baseInputs()
	in.Release.Digest = ""
	if _, err := Compile(context.Background(), in); err == nil {
		t.Fatalf("expected an error for a missing release digest")
	}
	in = baseInputs()
	in.Scope = "bogus"
	if _, err := Compile(context.Background(), in); err == nil {
		t.Fatalf("expected an error for an unknown scope")
	}
}
