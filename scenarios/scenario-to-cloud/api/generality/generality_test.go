package generality

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/certification"
	"scenario-to-cloud/closure"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/edge"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/fixtures"
	"scenario-to-cloud/vps"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// [REQ:STC-P0-043] P24-A01: a conforming scenario that did not exist when
// the ramp was built is deployed through declarations and configuration
// alone. The whole ramp runs in-process: closure → plan (needs_input
// handoff until onboarding supplies the operator credential) → durable
// admission → worker execution against the target owner → target receipts
// → health observation → certification evidence cell.
func TestNewcomerDeploysThroughDeclarationsOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	w := newcomerWorkload(t)
	if w.Declaration.PrimaryScenario != newcomerID || len(w.Declaration.Scenarios) == 0 || len(w.Declaration.Resources) == 0 {
		t.Fatalf("fixture must declare a scenario dependency and a resource: %+v", w.Declaration)
	}

	// Closure: every required component is present with a reason.
	c := mustClosure(t, w, productionEnv.Name)
	want := map[string]domain.ClosureComponentKind{
		newcomerID: domain.ClosureKindScenario, "ledger-service": domain.ClosureKindScenario, "ledger-store": domain.ClosureKindResource,
		"fixture/newcomer:webhook-signing-key": domain.ClosureKindCredentialDescriptor, "fixture/ledger:api-token": domain.ClosureKindCredentialDescriptor, "fixture/ledger-store:password": domain.ClosureKindCredentialDescriptor,
	}
	for id, kind := range want {
		component, ok := findComponent(c, id)
		if !ok {
			t.Fatalf("closure lacks %s (%s): %v", id, kind, componentIDs(c))
		}
		if component.Kind != kind || !component.Required {
			t.Fatalf("%s: kind %s required %v", id, component.Kind, component.Required)
		}
		if len(component.Reasons) == 0 {
			t.Fatalf("%s has no inclusion reason", id)
		}
	}
	store, _ := findComponent(c, "ledger-store")
	if !hasReason(store, domain.ClosureReasonDeclaredBy, newcomerID) || !hasReason(store, domain.ClosureReasonTransitiveVia, "") {
		t.Fatalf("ledger-store must be reached both directly and through ledger-service: %+v", store.Reasons)
	}
	if got := scenarioListeners(c); len(got) != 2 || got[0].PortName != "api" || got[0].Visibility != domain.EdgeVisibilityPublicViaEdge || got[1].PortName != "callback" || got[1].Visibility != "private" {
		t.Fatalf("declared listeners must reach the closure: %+v", got)
	}
	orders := false
	for _, data := range c.PersistentData {
		if data.ID == "newcomer-orders" && data.Owner == "ledger-store" && data.DeclaredBy == "scenario:"+newcomerID {
			orders = true
		}
	}
	if !orders {
		t.Fatalf("declared persistent data must reach the closure: %+v", c.PersistentData)
	}
	if primary, _ := findComponent(c, newcomerID); primary.Recovery == nil || primary.Recovery.SchemaStrategy != "expand_contract" {
		t.Fatalf("declared recovery contract must reach the closure: %+v", primary.Recovery)
	}

	target := newFakeTarget(productionEnv.MachineID)
	d := newDeployment(t, w, productionEnv, target, "newcomer-v1")

	// Before onboarding supplies the scenario's own descriptor the plan is a
	// single durable handoff naming exactly that address, no effect.
	pending, err := d.compile(execplan.ScopeFull, execplan.Observations{SatisfiedInputs: d.satisfiedInputs(false)})
	if err != nil {
		t.Fatalf("compile before onboarding: %v", err)
	}
	if pending.Outcome != execplan.OutcomeNeedsInput || pending.Handoff == nil || pending.Handoff.Owner != "vrooli-onboarding" {
		t.Fatalf("expected a needs_input onboarding handoff, got outcome %s handoff %+v", pending.Outcome, pending.Handoff)
	}
	if strings.Join(pending.Handoff.Missing, ",") != "fixture/newcomer:webhook-signing-key" {
		t.Fatalf("handoff must name the scenario's descriptor only: %v", pending.Handoff.Missing)
	}
	if len(target.snapshot().Calls) != 0 {
		t.Fatal("a needs_input plan must not touch the target")
	}

	// After onboarding: the plan digest is stable across compiles.
	obs := execplan.Observations{SatisfiedInputs: d.satisfiedInputs(true)}
	first := d.mustCompile(execplan.ScopeFull, obs)
	second := d.mustCompile(execplan.ScopeFull, obs)
	if first.MustDigest() != second.MustDigest() {
		t.Fatalf("plan digest must be stable: %s != %s", first.MustDigest(), second.MustDigest())
	}
	if first.ClosureDigest != c.Digest {
		t.Fatalf("plan must bind the closure digest %s, got %s", c.Digest, first.ClosureDigest)
	}
	public := publicListener(t, c)
	route := first.Action(execplan.OpEdgeRouteApply)
	if route == nil {
		t.Fatalf("plan must route the public listener: %v", first.ActionIDs())
	}
	if route.Inputs["upstream_port"] != itoa(d.manifest.Ports[public.PortName]) {
		t.Fatalf("edge.route.apply must forward to the declared public listener %s (port %d), got upstream_port=%q", public.ID, d.manifest.Ports[public.PortName], route.Inputs["upstream_port"])
	}
	if deps := first.Action(execplan.OpRuntimeStartDeps); deps == nil || !strings.Contains(deps.Inputs["resources"], "ledger-store") || !strings.Contains(deps.Inputs["scenarios"], "ledger-service") {
		t.Fatalf("plan must start the declared dependency scenario and resource: %+v", deps)
	}

	// Admit under a request key and let the worker execute it.
	plan, op, trace := d.deploy(execplan.ScopeFull, obs, "newcomer-first-deploy")
	if op.PlanDigest != first.MustDigest() {
		t.Fatalf("admitted digest %s != reviewed %s", op.PlanDigest, first.MustDigest())
	}
	executed := make([]string, 0, len(trace.Actions))
	for _, result := range trace.Actions {
		if result.Status != "succeeded" {
			t.Fatalf("action %s: %s %s", result.ID, result.Status, result.Error)
		}
		executed = append(executed, result.ID)
	}
	if strings.Join(executed, ",") != strings.Join(plan.ActionIDs(), ",") {
		t.Fatalf("executed %v != previewed %v", executed, plan.ActionIDs())
	}
	assertArgvDiscipline(t, plan, op, trace)
	assertNoSecretInTrace(t, plan, trace)
	snap := target.snapshot()
	assertReceiptsBound(t, plan, op, snap)
	if snap.Active == nil || snap.Active.Active != d.release {
		t.Fatalf("target must run the plan's release %s: %+v", d.release, snap.Active)
	}
	for _, id := range []string{newcomerID, "ledger-service"} {
		if !snap.Running[id] {
			t.Fatalf("%s must be running: %v", id, snap.Running)
		}
	}
	if !snap.Resources["ledger-store"] {
		t.Fatalf("ledger-store must be started: %v", snap.Resources)
	}
	if snap.BackedUp["newcomer-orders"] != 1 {
		t.Fatalf("the declared SQL binding must be covered by the recovery point before activation: %v", snap.BackedUp)
	}
	// The target's route is the one P14's policy derives for the declared
	// public listener: same host, upstream and listener id.
	wantRoute := d.edgeSpec().Routes[0]
	got, ok := snap.Routes[wantRoute.Host]
	if !ok || got.UpstreamPort != wantRoute.UpstreamPort || got.ListenerID != wantRoute.ListenerID || got.UpstreamPort != d.manifest.Ports[public.PortName] {
		t.Fatalf("edge must route %s to the declared public listener %s on port %d, got routes %+v", wantRoute.Host, public.ID, d.manifest.Ports[public.PortName], snap.Routes)
	}
	if len(d.provisioned) != 1 || d.provisioned[0].OperationID != op.ID || d.provisioned[0].Fence != op.Fence {
		t.Fatalf("credentials must be provisioned once under the operation identity: %+v", d.provisioned)
	}
	if strings.Join(d.provisioned[0].Operator, ",") != "NEWCOMER_WEBHOOK_SIGNING_KEY" || len(d.provisioned[0].Generated) != 2 {
		t.Fatalf("the operator descriptor arrives through onboarding, dependency credentials are generated: %+v", d.provisioned[0])
	}

	// Health observation binds the deployment id and the deployed release.
	obsHealth := d.observeHealth(time.Now())
	if obsHealth.GetDeploymentId() != d.id {
		t.Fatalf("observation deployment %q != %s", obsHealth.GetDeploymentId(), d.id)
	}
	if obsHealth.GetObservedReleaseDigest() != "sha256:"+bundleSHA(t, d.bundle) {
		t.Fatalf("observation release %q != deployed bundle", obsHealth.GetObservedReleaseDigest())
	}
	if obsHealth.GetFreshness() != healthv1.Freshness_FRESHNESS_CURRENT {
		t.Fatalf("observation freshness %s", obsHealth.GetFreshness())
	}
	if s := obsHealth.GetStatus(); s != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY && s != healthv1.HealthStatus_HEALTH_STATUS_DEGRADED {
		t.Fatalf("observation status %s: %s", s, mustJSON(obsHealth.GetChecks()))
	}

	// Evidence cell: a package-lane receipt for OPS-08 that the certification
	// registry accepts.
	receipt := ops08Receipt(t, d, op, plan)
	if err := receipt.Validate(embeddedMatrix(t)); err != nil {
		t.Fatalf("receipt: %v", err)
	}
	if os.Getenv("STC_WRITE_OPS08") != "" {
		raw, _ := json.MarshalIndent(receipt, "", "  ")
		if err := os.WriteFile(filepath.Join("..", "..", "certification", "evidence", "OPS-08.json"), append(raw, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// [REQ:STC-P0-043] P24-A02: two environments of one scenario on two targets
// have distinct deployment ids, target keys, credential bindings, edge
// specs (domains) and never read or write each other's state.
func TestTwoEnvironmentsOfOneScenarioAreIsolated(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	w := newcomerWorkload(t)
	stagingTarget := newFakeTarget(stagingEnv.MachineID)
	productionTarget := newFakeTarget(productionEnv.MachineID)
	staging := newDeployment(t, w, stagingEnv, stagingTarget, "newcomer-v1")
	production := newDeployment(t, w, productionEnv, productionTarget, "newcomer-v1")

	if staging.id == production.id {
		t.Fatalf("deployment ids must differ: %s", staging.id)
	}
	if staging.ref.MachineID == production.ref.MachineID || staging.ref.Locator.Host == production.ref.Locator.Host {
		t.Fatalf("target keys must differ: %+v / %+v", staging.ref, production.ref)
	}
	// One declaration set, one closure per environment: same components,
	// distinct digests because the environment is part of the closure.
	if staging.closure.Digest == production.closure.Digest || strings.Join(componentIDs(staging.closure), ",") != strings.Join(componentIDs(production.closure), ",") {
		t.Fatalf("closures must be environment-bound over the same components: staging=%s production=%s", staging.closure.Digest, production.closure.Digest)
	}

	// Credential bindings: one per (deployment, descriptor), disjoint sets.
	sb, pb := staging.credentialBindings(), production.credentialBindings()
	if len(sb) == 0 || len(sb) != len(pb) {
		t.Fatalf("expected the same number of bindings per environment, got %d and %d", len(sb), len(pb))
	}
	seen := map[string]string{}
	for _, b := range sb {
		seen[b.ID] = "staging"
	}
	for _, b := range pb {
		if owner, dup := seen[b.ID]; dup {
			t.Fatalf("binding id %s shared between %s and production", b.ID, owner)
		}
		if b.DeploymentID != production.id {
			t.Fatalf("binding %s belongs to %s, not %s", b.ID, b.DeploymentID, production.id)
		}
	}

	// Edge specs: distinct domains and route hosts, same private policy.
	ss, ps := staging.edgeSpec(), production.edgeSpec()
	if ss.Domain == ps.Domain || ss.Digest == ps.Digest || ss.Routes[0].Host == ps.Routes[0].Host {
		t.Fatalf("edge specs must be environment-specific: %s vs %s", ss.Domain, ps.Domain)
	}
	if ss.DeploymentID != staging.id || ps.DeploymentID != production.id {
		t.Fatalf("edge specs must bind their deployment: %s / %s", ss.DeploymentID, ps.DeploymentID)
	}

	obs := func(d *deployment) execplan.Observations {
		return execplan.Observations{SatisfiedInputs: d.satisfiedInputs(true)}
	}
	sPlan, sOp, sTrace := staging.deploy(execplan.ScopeFull, obs(staging), "two-env")
	pPlan, pOp, pTrace := production.deploy(execplan.ScopeFull, obs(production), "two-env")
	if sPlan.MustDigest() == pPlan.MustDigest() {
		t.Fatal("plans for two environments must not share a digest")
	}
	if sPlan.DeploymentID == pPlan.DeploymentID || sPlan.Target.MachineID == pPlan.Target.MachineID {
		t.Fatalf("plans must bind distinct deployment/target identities: %+v %+v", sPlan.Target, pPlan.Target)
	}
	assertArgvDiscipline(t, sPlan, sOp, sTrace)
	assertArgvDiscipline(t, pPlan, pOp, pTrace)

	// No cross-reads or cross-writes: every call a target received names
	// its own deployment; every route and receipt belongs to it.
	for _, pair := range []struct {
		d     *deployment
		other *deployment
	}{{staging, production}, {production, staging}} {
		snap := pair.d.target.snapshot()
		for _, call := range snap.Calls {
			flags, _ := flagsOf(call.Args)
			if dep := first(flags, "deployment"); dep != "" && dep != pair.d.id {
				t.Fatalf("target %s received a call for %s: %s", pair.d.env.MachineID, dep, call.Verb)
			}
			if strings.Contains(strings.Join(call.Args, " "), pair.other.id) || strings.Contains(strings.Join(call.Args, " "), pair.other.env.Domain) {
				t.Fatalf("target %s received the other environment's identity: %v", pair.d.env.MachineID, call.Args)
			}
		}
		wantHost := pair.d.edgeSpec().Routes[0].Host
		if len(snap.Routes) != 1 {
			t.Fatalf("target %s must hold exactly its own route: %v", pair.d.env.MachineID, snap.Routes)
		}
		for host, route := range snap.Routes {
			if host != wantHost || route.Deployment != pair.d.id {
				t.Fatalf("target %s routes %s for %s (want %s for %s)", pair.d.env.MachineID, host, route.Deployment, wantHost, pair.d.id)
			}
		}
		for _, r := range snap.Receipts {
			if !strings.HasPrefix(r.Operation, "op-"+pair.d.env.Name+"-") {
				t.Fatalf("target %s holds a receipt of %s", pair.d.env.MachineID, r.Operation)
			}
		}
		if pair.d.provisioned[0].DeploymentID != pair.d.id {
			t.Fatalf("credentials for %s provisioned under %s", pair.d.id, pair.d.provisioned[0].DeploymentID)
		}
	}
}

// [REQ:STC-P0-043] P24-A03: a dependency with no artifact for the target
// architecture is named by the closure and the plan refuses with
// unsupported_capability naming it; nothing is compiled.
func TestUnsupportedArchitectureNamesTheDependency(t *testing.T) {
	w := newcomerWorkload(t)
	c, err := resolveClosure(t, w, productionEnv.Name, closure.Platform{OS: "linux", Arch: "arm64"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if c.Supported() {
		t.Fatal("linux/arm64 must be unsupported for a resource that publishes only an amd64 artifact")
	}
	var entry *domain.ClosureUnsupported
	for i := range c.Unsupported {
		if c.Unsupported[i].Component == "ledger-store" {
			entry = &c.Unsupported[i]
		}
	}
	if entry == nil || entry.ReasonCode != closure.ReasonMissingPlatformArtifact || !strings.Contains(entry.Detail, "arm64") {
		t.Fatalf("closure must name ledger-store with %s and the architecture: %+v", closure.ReasonMissingPlatformArtifact, c.Unsupported)
	}
	manifest := manifestFor(&c, productionEnv)
	_, err = vps.CompilePlan(context.Background(), vps.PlanRequest{Manifest: manifest, BundlePath: "/var/lib/scenario-to-cloud/releases/newcomer/bundle.tar.gz", Closure: &c, Scope: execplan.ScopeFull})
	var typed *apierrors.Error
	if !errors.As(err, &typed) || typed.Code != apierrors.CodeUnsupportedCapability {
		t.Fatalf("expected %s, got %v", apierrors.CodeUnsupportedCapability, err)
	}
	names, _ := typed.Details["unsupported"].([]string)
	named := false
	for _, name := range names {
		if name == "ledger-store:"+closure.ReasonMissingPlatformArtifact {
			named = true
		}
	}
	if !named {
		t.Fatalf("refusal must name the exact unmet capability: %v", typed.Details)
	}
	// The scenario's own supported_targets (amd64 only) is reported beside
	// the artifact gap, so the operator sees every unmet capability at once.
	if !strings.Contains(strings.Join(names, ","), newcomerID+":"+closure.ReasonUnsupportedPlatform) {
		t.Fatalf("refusal must also name the scenario's declared platform support: %v", names)
	}
}

// [REQ:STC-P0-043] P24-A04: a hosted backend's declared public route and
// private callback listener survive an update: two plans, two activations,
// same route host, upstream and listener id; the callback listener is never
// routed; the predecessor is retained and the data binding persists.
func TestHostedBackendRoutesSurviveUpdate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	w := newcomerWorkload(t)
	target := newFakeTarget(productionEnv.MachineID)
	d := newDeployment(t, w, productionEnv, target, "newcomer-v1")
	obs := execplan.Observations{SatisfiedInputs: d.satisfiedInputs(true), DataSchemaVersion: "v1"}
	plan1, op1, _ := d.deploy(execplan.ScopeFull, obs, "v1")
	spec1 := d.edgeSpec()
	before := target.snapshot()
	// Update: a new release for the same deployment record on the same
	// target, observed against what runs now.
	v1Release := d.setRelease("newcomer-v2")
	if d.release == v1Release {
		t.Fatal("the update must carry a different release")
	}
	obs2 := execplan.Observations{SatisfiedInputs: d.satisfiedInputs(true), DataSchemaVersion: "v1", ActiveReleaseDigest: plan1.ReleaseDigest}
	plan2 := d.mustCompile(execplan.ScopeFull, obs2)
	if plan2.MustDigest() == plan1.MustDigest() {
		t.Fatal("an update must compile a different plan")
	}
	for _, id := range []string{execplan.OpDataBackup, execplan.OpReleaseActivate, execplan.OpEdgeRouteApply, execplan.OpReleaseRetainPredecesor} {
		if plan2.Action(id) == nil {
			t.Fatalf("update plan lacks %s: %v", id, plan2.ActionIDs())
		}
	}
	op2 := d.admit(plan2, "v2")
	done, trace2 := d.execute(op2)
	if done.State != domain.OperationSucceeded {
		t.Fatalf("update ended %s: %s", done.State, string(done.Error))
	}
	if done.Fence <= op1.Fence {
		t.Fatalf("update fence %d must exceed %d", done.Fence, op1.Fence)
	}
	assertArgvDiscipline(t, plan2, done, trace2)
	after := target.snapshot()
	assertReceiptsBound(t, plan2, done, after)

	spec2 := d.edgeSpec()
	if spec1.Digest != spec2.Digest {
		t.Fatalf("edge spec must not change across an update:\n%s\n%s", mustJSON(spec1), mustJSON(spec2))
	}
	public := publicListener(t, d.closure)
	if len(spec2.Routes) != 1 || !strings.HasSuffix(spec2.Routes[0].Host, productionEnv.Domain) || spec2.Routes[0].ListenerID != public.ID || spec2.Routes[0].UpstreamPort != d.manifest.Ports[public.PortName] {
		t.Fatalf("declared public route must be the only route: %+v", spec2.Routes)
	}
	routeHost := spec2.Routes[0].Host
	callback := false
	for _, l := range spec2.PrivateListeners {
		if l.ID == newcomerID+"/callback" {
			callback = true
			if l.Reason != edge.ReasonDeclaredPrivate {
				t.Fatalf("callback listener must stay private by declaration: %+v", l)
			}
		}
	}
	if !callback {
		t.Fatalf("private callback listener missing from the edge spec: %+v", spec2.PrivateListeners)
	}
	r1, r2 := before.Routes[routeHost], after.Routes[routeHost]
	if r1.UpstreamPort == 0 || r1.ListenerID != public.ID || r1 != r2 {
		t.Fatalf("target route must be unchanged across the update: before %+v after %+v", r1, r2)
	}
	if len(after.Routes) != 1 {
		t.Fatalf("only the declared public route may exist on the target: %v", after.Routes)
	}
	for _, port := range []int{d.manifest.Ports["callback"]} {
		for host, route := range after.Routes {
			if route.UpstreamPort == port {
				t.Fatalf("private callback port %d routed at %s", port, host)
			}
		}
	}
	if after.Active == nil || after.Active.Active != d.release || after.Active.Previous != v1Release {
		t.Fatalf("update must activate v2 and retain v1: %+v", after.Active)
	}
	// One recovery point per activation that touches persistent data: the
	// initial deploy and the update each capture newcomer-orders before the
	// runtime switch (design H: recovery points before destructive changes).
	if after.Backups != 2 {
		t.Fatalf("expected one recovery point per activation (2), got %d", after.Backups)
	}
	if after.BackedUp["newcomer-orders"] != 2 {
		t.Fatalf("the declared SQL binding must be covered by both recovery points: %v", after.BackedUp)
	}
	if len(d.provisioned) != 2 || d.provisioned[1].OperationID != done.ID {
		t.Fatalf("private credentials are re-provisioned server-side under the update operation: %+v", d.provisioned)
	}
	for _, r := range after.Receipts {
		if strings.Contains(r.Reply, canarySecret) {
			t.Fatalf("receipt %s/%s carries the operator secret", r.Operation, r.Step)
		}
	}
}

// [REQ:STC-P0-043] P24-A05: adopting the newcomer changes nothing for the
// generic fixture. The stateless-web plan compiles to the same digest before
// and after the newcomer journey in one test binary, and that digest equals
// the golden recorded in testdata.
func TestGenericFixtureUnchangedAfterAdoption(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cat := loadCatalog(t)
	generic, ok := cat.Workloads["stateless-web"]
	if !ok {
		t.Fatal("stateless-web fixture missing")
	}
	before := genericPlan(t, generic)

	w := newcomerWorkload(t)
	d := newDeployment(t, w, productionEnv, newFakeTarget(productionEnv.MachineID), "newcomer-v1")
	d.deploy(execplan.ScopeFull, execplan.Observations{SatisfiedInputs: d.satisfiedInputs(true)}, "adoption")

	after := genericPlan(t, generic)
	if before.MustDigest() != after.MustDigest() {
		t.Fatalf("generic plan digest moved after adoption: %s -> %s", before.MustDigest(), after.MustDigest())
	}
	goldenPath := filepath.Join("testdata", "golden", "stateless-web.plan-digest")
	if os.Getenv("STC_WRITE_GOLDEN") != "" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(after.MustDigest()+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden: %v (set STC_WRITE_GOLDEN=1 to record)", err)
	}
	if strings.TrimSpace(string(golden)) != after.MustDigest() {
		t.Fatalf("stateless-web plan digest %s differs from golden %s", after.MustDigest(), strings.TrimSpace(string(golden)))
	}
}

// genericPlan compiles the stateless-web fixture the way api/vps's fixture
// tests do, with fixed inputs so the digest is reproducible: a fixed
// artifact path (no release beside it, so release identity is pending) and
// a synthetic closure digest.
func genericPlan(t *testing.T, w *fixtures.Workload) *execplan.Plan {
	t.Helper()
	ports := domain.ManifestPorts{}
	for i, listener := range w.Declaration.Listeners {
		ports[listener.Name] = 3000 + i
	}
	if _, ok := ports["ui"]; !ok {
		ports["ui"] = 3000
	}
	manifest := domain.CloudManifest{
		Version:      "1",
		Target:       domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		Scenario:     domain.ManifestScenario{ID: w.Declaration.PrimaryScenario},
		Dependencies: domain.ManifestDependencies{Resources: append([]string(nil), w.Declaration.Resources...), Scenarios: append(append([]string(nil), w.Declaration.Scenarios...), w.Declaration.PrimaryScenario)},
		Ports:        ports,
		Edge:         domain.ManifestEdge{Domain: w.ID + ".example.test", Caddy: domain.ManifestCaddy{Enabled: true}},
	}
	c := &domain.Closure{SchemaVersion: "1", ScenarioID: w.Declaration.PrimaryScenario, Digest: "sha256:fixture-" + w.ID}
	c.Components = append(c.Components, domain.ClosureComponent{ID: w.Declaration.PrimaryScenario, Kind: domain.ClosureKindScenario, Required: true, Recovery: &domain.ClosureRecovery{CodeRollback: "any_predecessor", SchemaStrategy: "expand_contract"}})
	plan, err := vps.CompilePlan(context.Background(), vps.PlanRequest{Manifest: manifest, BundlePath: "/var/lib/scenario-to-cloud/releases/" + w.ID + "/bundle.tar.gz", Closure: c, Scope: execplan.ScopeFull, Observations: execplan.Observations{DeploymentRevision: 4}})
	if err != nil {
		t.Fatalf("compile %s: %v", w.ID, err)
	}
	return plan
}

// [REQ:STC-P0-043] P24-O05: no cloud source outside this package names the
// newcomer fixture, so the journeys above prove the fixture was adopted
// through data alone.
func TestNoCloudSourceNamesTheNewcomer(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	self, _ := filepath.Abs(".")
	var offenders []string
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path == self || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), newcomerID) || strings.Contains(string(data), "ledger-store") {
			offenders = append(offenders, strings.TrimPrefix(path, root+string(filepath.Separator)))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) != 0 {
		t.Fatalf("cloud source names the newcomer fixture: %v", offenders)
	}
}

func findComponent(c *domain.Closure, id string) (domain.ClosureComponent, bool) {
	for _, component := range c.Components {
		if component.ID == id {
			return component, true
		}
	}
	return domain.ClosureComponent{}, false
}

func componentIDs(c *domain.Closure) []string {
	out := make([]string, 0, len(c.Components))
	for _, component := range c.Components {
		out = append(out, component.ID)
	}
	return out
}

func hasReason(component domain.ClosureComponent, kind domain.ClosureReasonKind, from string) bool {
	for _, reason := range component.Reasons {
		if reason.Kind == kind && (from == "" || reason.From == from) {
			return true
		}
	}
	return false
}

func itoa(v int) string { return strconv.Itoa(v) }

func mustJSON(v any) string {
	raw, _ := json.MarshalIndent(v, "", "  ")
	return string(raw)
}

func embeddedMatrix(t *testing.T) *certification.Matrix {
	t.Helper()
	m, err := certification.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// ops08Receipt is the package-lane evidence cell for OPS-08 ("generic
// deployment and update boundary succeeds without product-specific code").
func ops08Receipt(t *testing.T, d *deployment, op *domain.CloudOperation, plan *execplan.Plan) certification.Receipt {
	t.Helper()
	return certification.Receipt{
		SchemaVersion:        1,
		CaseID:               "OPS-08",
		Verdict:              certification.VerdictPassed,
		Lane:                 certification.LanePackage,
		RequirementRefs:      []string{"STC-P0-043"},
		Candidate:            certification.Candidate{ReleaseDigest: plan.ReleaseDigest, ConfigurationDigest: plan.ConfigurationDigest, ClosureDigest: plan.ClosureDigest},
		Target:               certification.Target{MachineID: d.ref.MachineID, Architecture: "amd64", OS: "linux"},
		OperationRefs:        []string{op.ID},
		ValidationReceiptRef: "go test ./generality/ (package lane, in-process fake cloud-target owner)",
		ArtifactRefs: []string{
			"api/generality/generality_test.go#TestNewcomerDeploysThroughDeclarationsOnly",
			"api/generality/generality_test.go#TestTwoEnvironmentsOfOneScenarioAreIsolated",
			"api/generality/generality_test.go#TestUnsupportedArchitectureNamesTheDependency",
			"api/generality/generality_test.go#TestHostedBackendRoutesSurviveUpdate",
			"api/generality/generality_test.go#TestGenericFixtureUnchangedAfterAdoption",
			"api/generality/generality_test.go#TestNoCloudSourceNamesTheNewcomer",
			"fixtures/workloads/newcomer-service/fixture.json",
		},
		Assertions: []string{
			"newcomer-service (scenario dependency + resource + credential descriptors + deployment block) deployed from its declaration tree with no cloud source naming it",
			"every required closure component carries an inclusion reason; the operator descriptor produced one onboarding handoff before any effect",
			"every effectful target verb carried --deployment --operation --step --fence; every target receipt named the operation and its fence; the operator secret appeared in no plan, argv, trace or receipt",
			"plan digest stable across compiles; health observation bound to the deployment id and the deployed release",
			"staging and production on two targets: distinct deployment ids, target keys, credential binding ids, edge specs; no target received the other environment's identity",
			"linux/arm64: closure names ledger-store missing_platform_artifact and the plan refuses unsupported_capability naming it",
			"update (two plans, two activations): public route host, upstream and listener id unchanged; private callback listener never routed; predecessor retained; persistent data bound",
			"stateless-web plan digest identical before and after the newcomer journey and equal to the recorded golden",
		},
		ObservedAt: time.Date(2026, 9, 9, 18, 0, 0, 0, time.UTC),
		Limitations: []string{
			"package lane only: the real-VPS 24-hour soak and QEMU fault lanes OPS-08 requires are pending EXT-01/EXT-06/EXT-09",
			"public and origin HTTPS readiness probes are answered by an in-process prober; the fake cloud-target owner has no network edge",
			"no representative hosted consumer (LPBS) was updated or restored; its read-only observations are recorded in P24-generality.md",
		},
	}
}
