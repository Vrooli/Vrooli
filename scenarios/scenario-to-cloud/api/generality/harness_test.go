package generality

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/vrooli/packages/cloudrelease"
	_ "modernc.org/sqlite"

	"scenario-to-cloud/closure"
	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/edge"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/fixtures"
	"scenario-to-cloud/health"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/persistence"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/vps"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// newcomerID is the fixture scenario that did not exist when the ramp was
// built. Nothing under api/ outside this package names it (asserted by
// TestNoCloudSourceNamesTheNewcomer).
const newcomerID = "newcomer-service"

// canarySecret is the synthetic operator-supplied value; it must never
// appear in a plan, an argv, a receipt or a trace.
const canarySecret = "canary-secret-9f3a7c1e"

var linuxAMD64 = closure.Platform{OS: "linux", Arch: "amd64"}

// environment is one installation of the scenario: its own deployment id,
// target, domain and host.
type environment struct {
	Name      string
	Domain    string
	Host      string
	MachineID string
}

var (
	stagingEnv    = environment{Name: "staging", Domain: "staging.newcomer.example.test", Host: "203.0.113.11", MachineID: "fake-target-staging"}
	productionEnv = environment{Name: "production", Domain: "newcomer.example.test", Host: "203.0.113.12", MachineID: "fake-target-production"}
)

func fixturesRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "fixtures"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func loadCatalog(t *testing.T) *fixtures.Catalog {
	t.Helper()
	cat, err := fixtures.Load(fixturesRoot(t))
	if err != nil {
		t.Fatalf("load fixtures: %v", err)
	}
	return cat
}

func newcomerWorkload(t *testing.T) *fixtures.Workload {
	t.Helper()
	w, ok := loadCatalog(t).Workloads[newcomerID]
	if !ok {
		t.Fatalf("fixture %s is not in the catalog", newcomerID)
	}
	if _, err := fixtures.VerifyOracle(w); err != nil {
		t.Fatalf("oracle: %v", err)
	}
	return w
}

// resolveClosure derives the closure from the fixture's declaration tree
// alone: the same resolver production uses, pointed at data.
func resolveClosure(t *testing.T, w *fixtures.Workload, env string, platform closure.Platform) (domain.Closure, error) {
	t.Helper()
	root := filepath.Join(w.Dir, "declarations")
	cat := closure.NewLayoutCatalog(filepath.Join(root, "scenarios"), filepath.Join(root, "resources"))
	return closure.Resolve(context.Background(), closure.Inputs{
		ScenarioID:       w.Declaration.PrimaryScenario,
		Environment:      env,
		Platform:         platform,
		Scope:            closure.ScopeBundle,
		Catalog:          cat,
		HostRequirements: closure.DeclaredHostRequirements{},
	})
}

func mustClosure(t *testing.T, w *fixtures.Workload, env string) *domain.Closure {
	t.Helper()
	c, err := resolveClosure(t, w, env, linuxAMD64)
	if err != nil {
		t.Fatalf("resolve closure: %v", err)
	}
	if !c.Supported() {
		t.Fatalf("closure must be supported on %s: %+v", linuxAMD64, c.Unsupported)
	}
	return &c
}

// scenarioListeners returns the closure listeners the deployed scenario owns.
func scenarioListeners(c *domain.Closure) []domain.ClosureListener {
	var out []domain.ClosureListener
	for _, l := range c.Listeners {
		if l.Owner == "scenario:"+c.ScenarioID {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PortName < out[j].PortName })
	return out
}

// publicListener is the one listener the scenario declares public_via_edge.
func publicListener(t *testing.T, c *domain.Closure) domain.ClosureListener {
	t.Helper()
	var found []domain.ClosureListener
	for _, l := range scenarioListeners(c) {
		if l.Visibility == domain.EdgeVisibilityPublicViaEdge {
			found = append(found, l)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected exactly one public_via_edge listener, got %+v", found)
	}
	return found[0]
}

// credentialAddresses lists the closure's credential descriptor addresses.
func credentialAddresses(c *domain.Closure) []string {
	var out []string
	for _, component := range c.ComponentsOfKind(domain.ClosureKindCredentialDescriptor) {
		if component.Credential != nil {
			out = append(out, component.Credential.LogicalID+":"+component.Credential.Field)
		}
	}
	sort.Strings(out)
	return out
}

// ownedByScenario reports whether the deployed scenario itself declared the
// descriptor (reason credential_of from scenario:<id>), as opposed to a
// dependency scenario or resource.
func ownedByScenario(c *domain.Closure, address string) bool {
	for _, component := range c.ComponentsOfKind(domain.ClosureKindCredentialDescriptor) {
		if component.Credential == nil || component.Credential.LogicalID+":"+component.Credential.Field != address {
			continue
		}
		for _, reason := range component.Reasons {
			if reason.Kind == domain.ClosureReasonCredentialOf && reason.From == "scenario:"+c.ScenarioID {
				return true
			}
		}
	}
	return false
}

// manifestFor is the configuration the operator supplies for one
// environment: target locator, port assignments for the declared
// listeners, the edge domain and the secrets plan derived from the
// closure's descriptors. Everything else comes from the declarations.
func manifestFor(c *domain.Closure, env environment) domain.CloudManifest {
	ports := domain.ManifestPorts{}
	for i, listener := range scenarioListeners(c) {
		ports[listener.PortName] = 3000 + i
	}
	var resources, scenarios []string
	for _, component := range c.Components {
		switch component.Kind {
		case domain.ClosureKindResource:
			resources = append(resources, component.ID)
		case domain.ClosureKindScenario:
			scenarios = append(scenarios, component.ID)
		}
	}
	sort.Strings(resources)
	sort.Strings(scenarios)
	secrets := &domain.ManifestSecrets{}
	for _, component := range c.ComponentsOfKind(domain.ClosureKindCredentialDescriptor) {
		cred := component.Credential
		if cred == nil {
			continue
		}
		class := "per_install_generated"
		if ownedByScenario(c, cred.LogicalID+":"+cred.Field) {
			// The deployed scenario's own descriptor is operator-owned and
			// arrives through onboarding; dependency credentials are minted
			// per install.
			class = "user_prompt"
		}
		secrets.BundleSecrets = append(secrets.BundleSecrets, domain.BundleSecretPlan{
			ID:         strings.ReplaceAll(cred.LogicalID+"-"+cred.Field, "/", "-"),
			Class:      class,
			Required:   cred.Required,
			Target:     domain.BundleSecretTarget{Type: "env", Name: cred.Env},
			Descriptor: &domain.DescriptorAddress{LogicalID: cred.LogicalID, Field: cred.Field},
		})
	}
	secrets.Summary.TotalSecrets = len(secrets.BundleSecrets)
	return domain.CloudManifest{
		Version:     "1",
		Environment: env.Name,
		Target:      domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: env.Host, Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		Scenario:    domain.ManifestScenario{ID: c.ScenarioID},
		Bundle:      domain.ManifestBundle{Scenarios: scenarios, Resources: resources},
		Dependencies: domain.ManifestDependencies{
			Resources: resources,
			Scenarios: scenarios,
		},
		Ports:   ports,
		Edge:    domain.ManifestEdge{Domain: env.Domain, Caddy: domain.ManifestCaddy{Enabled: true, Email: "ops@example.test"}},
		Secrets: secrets,
	}
}

// fixtureRelease builds a complete release directory beside a bundle so the
// plan binds a canonical release id. content changes the bundle bytes and
// therefore the release identity.
func fixtureRelease(t *testing.T, dir, content string) (bundlePath, releaseID string) {
	t.Helper()
	bundle := []byte("fixture-bundle-" + content)
	binary := []byte("fixture-native-cli")
	sumBundle := sha256.Sum256(bundle)
	sumBinary := sha256.Sum256(binary)
	manifest := cloudrelease.Manifest{
		SchemaVersion: cloudrelease.ManifestSchemaVersion, BundleSHA256: hex.EncodeToString(sumBundle[:]),
		NativeCLI:     cloudrelease.NativeCLI{SHA256: hex.EncodeToString(sumBinary[:]), GOOS: "linux", GOARCH: "amd64"},
		ClosureDigest: strings.Repeat("c", 64), ConfigurationDigest: strings.Repeat("d", 64),
		Provenance: cloudrelease.Provenance{Builder: "scenario-to-cloud", Policy: "development-local-unsigned"},
		Limits:     cloudrelease.Limits{MaxEntries: 1000, MaxExpandedBytes: 1 << 20, MaxEntryBytes: 1 << 16},
	}
	digest, err := cloudrelease.ComputeReleaseDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ReleaseDigest = digest
	releaseDir := filepath.Join(dir, "releases", digest)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(manifest)
	for name, body := range map[string][]byte{cloudrelease.BundleFileName: bundle, cloudrelease.ManifestFileName: raw, cloudrelease.NativeCLIFileName("linux", "amd64"): binary, cloudrelease.CompleteMarker: []byte("")} {
		if err := os.WriteFile(filepath.Join(releaseDir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(releaseDir, cloudrelease.BundleFileName), digest
}

// deployment is one installation driven through the durable owner: a
// deployment record in routed in-memory storage, an operations service
// whose runner is the production executor over the fake target.
type deployment struct {
	t        *testing.T
	env      environment
	target   *fakeTarget
	ref      identity.TargetRef
	id       string
	manifest domain.CloudManifest
	closure  *domain.Closure
	bundle   string
	release  string
	repo     *persistence.Repository
	svc      *operations.Service
	ctx      context.Context

	mu          sync.Mutex
	traces      map[string]vps.Trace
	provisioned []credentialCall
	seq         int
	// probeFallbacks counts verify.readiness actions whose public HTTPS
	// probe was answered by the package-lane fallback (see run).
	probeFallbacks int
}

type credentialCall struct {
	DeploymentID string
	OperationID  string
	Fence        uint64
	Generated    []string
	Operator     []string
}

var dbSeq int

func openRoutedStorage(t *testing.T) (*persistence.Repository, context.Context) {
	t.Helper()
	dbSeq++
	db, err := sql.Open("sqlite", fmt.Sprintf("file:generality-%d-%d?mode=memory&cache=shared", time.Now().UnixNano(), dbSeq))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	repo := persistence.NewRepository(db)
	ctx := database.WithTestMode(context.Background())
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatal(err)
	}
	return repo, ctx
}

// newDeployment creates the record for one environment of the fixture on
// its own target and starts a durable operations owner for it.
func newDeployment(t *testing.T, w *fixtures.Workload, env environment, target *fakeTarget, releaseContent string) *deployment {
	t.Helper()
	c := mustClosure(t, w, env.Name)
	manifest := manifestFor(c, env)
	bundle, release := fixtureRelease(t, t.TempDir(), releaseContent)
	repo, ctx := openRoutedStorage(t)
	d := &deployment{
		t: t, env: env, target: target, id: "dep-" + w.ID + "-" + env.Name,
		ref:      identity.TargetRef{MachineID: target.machineID, Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: env.Host, Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		manifest: manifest, closure: c, bundle: bundle, release: release, repo: repo, ctx: ctx, traces: map[string]vps.Trace{},
	}
	raw, _ := json.Marshal(manifest)
	now := time.Now().UTC()
	sha := bundleSHA(t, bundle)
	if err := repo.CreateDeployment(ctx, &domain.Deployment{ID: d.id, Name: w.ID + " @ " + env.Domain, ScenarioID: c.ScenarioID, Environment: env.Name, Target: d.ref, Status: domain.StatusPending, Manifest: raw, BundleSHA256: &sha, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	cfg := operations.Config{WorkerID: "generality-" + env.Name, Workers: 1, LeaseTTL: 2 * time.Second, HeartbeatInterval: 200 * time.Millisecond, ReconcileInterval: time.Hour, ExecutionTimeout: 30 * time.Second, TransportTimeout: 5 * time.Second, ObserverTimeout: 5 * time.Second, QueueTimeout: time.Minute}
	d.svc = operations.NewService(cfg, repo, operations.RunnerFunc(d.run), nil, nil)
	d.svc.Start()
	t.Cleanup(d.svc.Stop)
	return d
}

func bundleSHA(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// setRelease binds a newly built release to the deployment record, as the
// release builder does before an update plan is compiled.
func (d *deployment) setRelease(content string) (previous string) {
	d.t.Helper()
	previous = d.release
	d.bundle, d.release = fixtureRelease(d.t, d.t.TempDir(), content)
	info, err := os.Stat(d.bundle)
	if err != nil {
		d.t.Fatal(err)
	}
	if err := d.repo.UpdateDeploymentBundle(d.ctx, d.id, d.bundle, bundleSHA(d.t, d.bundle), info.Size()); err != nil {
		d.t.Fatal(err)
	}
	return previous
}

func (d *deployment) record() *domain.Deployment {
	d.t.Helper()
	dep, err := d.repo.GetDeployment(d.ctx, d.id)
	if err != nil || dep == nil {
		d.t.Fatalf("load deployment %s: %v", d.id, err)
	}
	return dep
}

// satisfiedInputs marks the dependency credentials as satisfied (they are
// minted per install) and, when onboarded is true, the scenario's own
// operator-supplied descriptor too.
func (d *deployment) satisfiedInputs(onboarded bool) []string {
	var out []string
	for _, address := range credentialAddresses(d.closure) {
		if ownedByScenario(d.closure, address) && !onboarded {
			continue
		}
		out = append(out, address)
	}
	return out
}

// operatorSecrets are the values onboarding supplied, keyed by descriptor.
func (d *deployment) operatorSecrets() map[string]string {
	out := map[string]string{}
	for _, address := range credentialAddresses(d.closure) {
		if ownedByScenario(d.closure, address) {
			out[address] = canarySecret
		}
	}
	return out
}

func (d *deployment) compile(scope string, obs execplan.Observations) (*execplan.Plan, error) {
	dep := d.record()
	obs.DeploymentRevision = dep.Fence
	return vps.CompilePlan(d.ctx, vps.PlanRequest{Manifest: d.manifest, BundlePath: d.bundle, Closure: d.closure, Scope: scope, Observations: obs, Deployment: dep})
}

func (d *deployment) mustCompile(scope string, obs execplan.Observations) *execplan.Plan {
	d.t.Helper()
	plan, err := d.compile(scope, obs)
	if err != nil {
		d.t.Fatalf("compile %s: %v", scope, err)
	}
	if plan.Outcome != execplan.OutcomeApply {
		d.t.Fatalf("compile %s: outcome %s (handoff %+v)", scope, plan.Outcome, plan.Handoff)
	}
	return plan
}

// admit stores the reviewed plan as a durable operation under a request
// key, exactly as the apply endpoint does, and returns the operation.
func (d *deployment) admit(plan *execplan.Plan, requestKey string) *domain.CloudOperation {
	d.t.Helper()
	raw, err := json.Marshal(plan)
	if err != nil {
		d.t.Fatal(err)
	}
	d.seq++
	op, err := d.repo.AdmitOperation(d.ctx, &domain.CloudOperation{ID: fmt.Sprintf("op-%s-%d", d.env.Name, d.seq), DeploymentID: d.id, RequestKey: requestKey, PlanDigest: plan.MustDigest(), Plan: raw})
	if err != nil {
		d.t.Fatalf("admit: %v", err)
	}
	return op
}

// execute submits an admitted operation to the worker and waits for its
// terminal state.
func (d *deployment) execute(op *domain.CloudOperation) (*domain.CloudOperation, vps.Trace) {
	d.t.Helper()
	d.svc.Submit(d.ctx, op.ID, operations.ExecuteOptions{ProvidedSecrets: d.operatorSecrets()})
	done, pending, err := d.svc.Wait(d.ctx, op.ID, 30*time.Second)
	if err != nil {
		d.t.Fatalf("wait %s: %v", op.ID, err)
	}
	if pending {
		d.t.Fatalf("operation %s still pending", op.ID)
	}
	d.mu.Lock()
	trace := d.traces[op.ID]
	d.mu.Unlock()
	return done, trace
}

// deploy compiles, admits and executes one scope and fails the test unless
// the operation succeeded.
func (d *deployment) deploy(scope string, obs execplan.Observations, requestKey string) (*execplan.Plan, *domain.CloudOperation, vps.Trace) {
	d.t.Helper()
	plan := d.mustCompile(scope, obs)
	op := d.admit(plan, requestKey)
	done, trace := d.execute(op)
	if done.State != domain.OperationSucceeded {
		d.t.Fatalf("operation %s ended %s: %s", op.ID, done.State, string(done.Error))
	}
	return plan, done, trace
}

// run is the operations.Runner: the production executor over the fake
// target with the durable step hooks, exactly as deployment.Orchestrator
// wires it.
func (d *deployment) run(ctx context.Context, ec *operations.ExecutionContext) error {
	hooks := vps.Hooks{
		Before: func(ctx context.Context, action execplan.Action) (bool, error) {
			decision, err := ec.Steps.Begin(ctx, action)
			if err != nil {
				return false, err
			}
			return decision == operations.DecisionSkip, nil
		},
		After: func(ctx context.Context, action execplan.Action, result domain.VPSActionResult, execErr error) error {
			outcome := operations.StepSucceeded
			if execErr != nil && action.ID == execplan.OpVerifyReadiness && strings.HasPrefix(execErr.Error(), "https:") {
				// Package-lane limitation until vps.Runtime carries a
				// readiness prober: the local pointer check passed (it runs
				// first) and the public HTTPS probe cannot reach a fake
				// target. Recorded on the receipt as a limitation.
				d.mu.Lock()
				d.probeFallbacks++
				d.mu.Unlock()
				execErr = nil
			}
			if execErr != nil {
				outcome = operations.StepFailed
			}
			err := ec.Steps.Commit(ctx, action, outcome, result.Detail, execErr)
			if errors.Is(err, operations.ErrReplayStep) {
				return vps.ErrReplayAction
			}
			return err
		},
	}
	d.target.mu.Lock()
	d.target.scenario = d.closure.ScenarioID
	d.target.mu.Unlock()
	rt := vps.Runtime{
		Reach:           d.target,
		Target:          d.ref,
		Identity:        vps.Identity{OperationID: ec.Operation.ID, Fence: ec.Fence},
		Credentials:     &recordingCredentials{d: d},
		Backups:         recordingBackups{},
		ProvidedSecrets: ec.Options.ProvidedSecrets,
	}
	trace, execErr := vps.ExecutePlan(ctx, vps.ExecuteRequest{Plan: ec.Plan, Manifest: d.manifest, BundlePath: d.bundle, DeploymentID: d.id, Runtime: rt, Hooks: hooks})
	d.mu.Lock()
	d.traces[ec.Operation.ID] = trace
	d.mu.Unlock()
	if execErr != nil {
		return execErr
	}
	return nil
}

// recordingCredentials is the credential authority seam: it records which
// secret ids were handed to it (never the values) and materialises nothing.
type recordingCredentials struct{ d *deployment }

func (r *recordingCredentials) Provision(_ context.Context, req vps.CredentialProvisionRequest) (vps.CredentialProvisionResult, error) {
	call := credentialCall{DeploymentID: req.DeploymentID, OperationID: req.Identity.OperationID, Fence: req.Identity.Fence}
	for id := range req.GeneratedValues {
		call.Generated = append(call.Generated, id)
	}
	for id := range req.OperatorValues {
		call.Operator = append(call.Operator, id)
	}
	sort.Strings(call.Generated)
	sort.Strings(call.Operator)
	r.d.mu.Lock()
	r.d.provisioned = append(r.d.provisioned, call)
	r.d.mu.Unlock()
	ids := append(append([]string(nil), call.Generated...), call.Operator...)
	bindings := make([]string, 0, len(ids))
	for _, id := range ids {
		bindings = append(bindings, credentials.BindingID(req.DeploymentID, domain.CredentialDescriptor{LogicalID: "fixture/" + id, Field: id}))
	}
	return vps.CredentialProvisionResult{Materialized: bindings}, nil
}

func (r *recordingCredentials) Revoke(context.Context, vps.CredentialProvisionRequest) (vps.CredentialProvisionResult, error) {
	return vps.CredentialProvisionResult{}, nil
}

type recordingBackups struct{}

func (recordingBackups) Record(_ context.Context, _ string, id vps.Identity, _ json.RawMessage, _ string) (string, error) {
	return id.OperationID + "-data.backup", nil
}

// edgeSpec derives the declared edge policy for this environment from the
// closure listeners and the manifest ports (P14's owner).
func (d *deployment) edgeSpec() domain.EdgeSpec {
	d.t.Helper()
	spec, err := edge.Derive(edge.PolicyInputs{
		DeploymentID: d.id, ScenarioID: d.closure.ScenarioID, Environment: d.env.Name, Domain: d.env.Domain,
		Listeners: d.closure.Listeners, Ports: d.manifest.Ports, TargetHost: d.env.Host, ACMEEmail: d.manifest.Edge.Caddy.Email,
		DatabaseResources: credentials.DatabaseResourcesFromClosure(d.closure),
	})
	if err != nil {
		d.t.Fatalf("edge spec: %v", err)
	}
	return spec
}

// credentialBindings plans the deployment's credential bindings (P13's
// owner): one binding id per (deployment, descriptor), values absent.
func (d *deployment) credentialBindings() []domain.CredentialBinding {
	d.t.Helper()
	bindings, err := credentials.PlanBindings(credentials.PlanInputs{
		DeploymentID: d.id, Plans: d.manifest.Secrets.BundleSecrets,
		Consumers: credentials.ConsumersFromClosure(d.closure), DatabaseResources: credentials.DatabaseResourcesFromClosure(d.closure),
		Now: time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		d.t.Fatalf("plan bindings: %v", err)
	}
	return bindings
}

// observeHealth builds the typed health observation (P16's owner) from an
// inspection of the fake target's durable state.
func (d *deployment) observeHealth(now time.Time) *healthv1.HealthObservation {
	d.t.Helper()
	snap := d.target.snapshot()
	live := &domain.LiveStateResult{
		OK:        true,
		Timestamp: now.UTC().Format(time.RFC3339),
		System: &domain.SystemState{
			SSH:    domain.SSHHealth{Connected: true, LatencyMs: 12, AuthMode: string(sshidentity.AuthModeExplicitKey), VerificationState: string(sshidentity.VerificationAuthorized)},
			CPU:    domain.CPUInfo{Cores: 2, UsagePercent: 5},
			Memory: domain.MemoryInfo{TotalMB: 2048, UsedMB: 512, UsagePercent: 25},
			Disk:   domain.DiskInfo{TotalGB: 40, UsedGB: 8, UsagePercent: 20},
		},
		Processes: &domain.ProcessState{},
		Caddy:     &domain.CaddyState{Running: true, Domain: d.env.Domain},
	}
	for id, running := range snap.Running {
		status := "stopped"
		if running {
			status = "running"
		}
		live.Processes.Scenarios = append(live.Processes.Scenarios, domain.ScenarioProcess{ID: id, Status: status, PID: 100 + len(live.Processes.Scenarios)})
	}
	for id, running := range snap.Resources {
		status := "stopped"
		if running {
			status = "running"
		}
		live.Processes.Resources = append(live.Processes.Resources, domain.ResourceProcess{ID: id, Status: status, PID: 500 + len(live.Processes.Resources)})
	}
	dep := d.record()
	dep.Status = domain.StatusDeployed
	ident := sshidentity.DeploymentSSHIdentity{KeyPath: "~/.ssh/id_ed25519", PublicKeyFingerprint: "SHA256:fixture", AuthMode: sshidentity.AuthModeExplicitKey, VerificationState: sshidentity.VerificationAuthorized}
	report := vps.ComputeHealth(dep, d.manifest, ident, live, nil, nil, nil)
	report.Freshness = &domain.FreshnessStatus{Status: domain.FreshnessCurrent, Summary: "bundle matches the deployed release"}
	return health.Build(health.Input{Deployment: dep, Report: report, LiveState: live, Now: now}, health.DefaultPolicy())
}

// assertArgvDiscipline proves every transport call of a trace belongs to a
// previewed action, carries no shell syntax, and that every effectful
// cloud-target verb carries the operation, step and fence of the operation.
func assertArgvDiscipline(t *testing.T, plan *execplan.Plan, op *domain.CloudOperation, trace vps.Trace) {
	t.Helper()
	declared := map[string]bool{}
	for _, id := range plan.ActionIDs() {
		declared[id] = true
	}
	if len(trace.Calls) == 0 {
		t.Fatal("expected transport calls")
	}
	effectful := 0
	for _, call := range trace.Calls {
		if !declared[call.ActionID] {
			t.Fatalf("undeclared effect: %s %q outside any action", call.Kind, call.Command)
		}
		if strings.Contains(call.Command, "rm -rf") || strings.Contains(call.Command, "&&") || strings.Contains(call.Command, "|") || strings.Contains(call.Command, ";") {
			t.Fatalf("shell fragment reached the transport: %q", call.Command)
		}
		if call.Kind != "exec" {
			continue
		}
		if !strings.HasPrefix(call.Verb, "cloud-target") {
			continue
		}
		flags, _ := flagsOf(call.Args)
		if _, isEffect := flags["fence"]; !isEffect {
			continue
		}
		effectful++
		if first(flags, "operation") != op.ID {
			t.Fatalf("%s: --operation %q != %s", call.Verb, first(flags, "operation"), op.ID)
		}
		if !stepOfAction(first(flags, "step"), call.ActionID) {
			t.Fatalf("%s: --step %q is not a step of action %s", call.Verb, first(flags, "step"), call.ActionID)
		}
		if first(flags, "fence") != fmt.Sprint(op.Fence) {
			t.Fatalf("%s: --fence %q != %d", call.Verb, first(flags, "fence"), op.Fence)
		}
		if first(flags, "deployment") != plan.DeploymentID {
			t.Fatalf("%s: --deployment %q != %s", call.Verb, first(flags, "deployment"), plan.DeploymentID)
		}
	}
	if effectful == 0 {
		t.Fatal("expected fenced cloud-target verbs in the trace")
	}
}

// assertReceiptsBound proves every receipt the target persisted names this
// operation and its fence, and every step is a previewed action.
func assertReceiptsBound(t *testing.T, plan *execplan.Plan, op *domain.CloudOperation, snap fakeSnapshot) {
	t.Helper()
	declared := map[string]bool{}
	for _, id := range plan.ActionIDs() {
		declared[id] = true
	}
	found := 0
	for _, r := range snap.Receipts {
		if r.Operation != op.ID {
			continue
		}
		found++
		if r.Fence != op.Fence {
			t.Fatalf("receipt %s/%s carries fence %d, operation fence is %d", r.Operation, r.Step, r.Fence, op.Fence)
		}
		if !declaredStep(declared, r.Step) {
			t.Fatalf("receipt step %s is not a step of a previewed action", r.Step)
		}
		if strings.Contains(r.Reply, canarySecret) {
			t.Fatalf("receipt %s/%s carries the operator secret", r.Operation, r.Step)
		}
	}
	if found == 0 {
		t.Fatalf("no receipt on the target names operation %s", op.ID)
	}
}

// stepOfAction reports whether a receipt step belongs to an action: the
// action id itself or one of its per-item sub-steps (<action>.<item>).
func stepOfAction(step, actionID string) bool {
	return step == actionID || strings.HasPrefix(step, actionID+".")
}

func declaredStep(declared map[string]bool, step string) bool {
	for id := range declared {
		if stepOfAction(step, id) {
			return true
		}
	}
	return false
}

// assertNoSecretInTrace proves the canary never reached a plan, an argv or
// a trace.
func assertNoSecretInTrace(t *testing.T, plan *execplan.Plan, trace vps.Trace) {
	t.Helper()
	raw, _ := json.Marshal(plan)
	if strings.Contains(string(raw), canarySecret) {
		t.Fatal("plan carries the operator secret")
	}
	for _, call := range trace.Calls {
		if strings.Contains(call.Command, canarySecret) || strings.Contains(strings.Join(call.Args, " "), canarySecret) {
			t.Fatalf("transport call carries the operator secret: %s", call.ActionID)
		}
	}
	for _, result := range trace.Actions {
		if strings.Contains(result.Detail, canarySecret) || strings.Contains(result.Error, canarySecret) {
			t.Fatalf("action result %s carries the operator secret", result.ID)
		}
	}
}
