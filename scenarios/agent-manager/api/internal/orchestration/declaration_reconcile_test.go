package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

// fixtureProfile is a scenario-owned profile declaration in the unified layout.
const fixtureProfile = `{
  "schemaVersion": "agent-profile/v1",
  "name": "fixture-scn default",
  "profileKey": "fixture-scn/default",
  "description": "fixture profile",
  "roleRef": "code.smart",
  "maxTurns": 10,
  "sandboxConfig": {"mode": "SANDBOX_MODE_OFF"},
  "createdBy": "fixture-scn"
}`

// fixtureWorkflow is a single-run workflow whose run node references the
// profile the profile source reconciles in the same call.
const fixtureWorkflow = `{
  "schemaVersion": "agent-workflow/v1",
  "owner": "fixture-scn",
  "key": "fixture-scn/round",
  "version": "1.0.0",
  "description": "fixture workflow",
  "inputSchema": {"type": "object", "additionalProperties": true},
  "outputSchema": {"type": "object", "additionalProperties": true},
  "entryNode": "work",
  "nodes": [
    {"id": "work", "kind": "run", "run": {"profileKey": "fixture-scn/default", "promptTemplate": "Do the thing.", "maxTurns": 5, "timeoutSeconds": 300}},
    {"id": "done", "kind": "end", "end": {"status": "succeeded"}}
  ],
  "edges": [{"from": "work", "to": "done"}],
  "budgets": {"wallTimeSeconds": 1200, "maxTurns": 12, "maxTokens": 30000, "maxChargeMicroUsd": 5, "maxNodeAttempts": 2, "maxChildren": 1, "maxConcurrency": 1, "maxRecursion": 1, "maxRetries": 1, "maxWaitSeconds": 60}
}`

func newDeclarationOrchestrator(t *testing.T) *Orchestrator {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	return New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		WithEvents(eventStore),
		WithWorkflowRepository(repos.Workflows),
		WithWorkflowExecutionRepository(repos.WorkflowExecutions),
	)
}

// writeScenarioFixture materializes a scenario tree (service.json plus the given
// declaration files) under a temp dir and returns its root and manifest path.
func writeScenarioFixture(t *testing.T, files map[string]string) (scenarioRoot, servicePath string) {
	t.Helper()
	scenarioRoot = t.TempDir()
	for rel, content := range files {
		full := filepath.Join(scenarioRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	servicePath = filepath.Join(scenarioRoot, ".vrooli", "service.json")
	return scenarioRoot, servicePath
}

func declarationManifest(sources ...string) string {
	quoted := make([]string, 0, len(sources))
	for _, source := range sources {
		quoted = append(quoted, `"`+source+`"`)
	}
	return `{"dependencies":{"scenarios":{"agent-manager":{"enabled":true,"config":{"declarations":{"reconcile":true,"profileMode":"update_if_unmodified","sources":[` +
		strings.Join(quoted, ",") + `]}}}}}}`
}

// [REQ:REQ-P0-001] The unified declaration block is the single strict contract
// for scenario-owned declarations; legacy blocks and old-directory sources are
// rejected with actionable diagnostics.
func TestReadScenarioDeclarationConfig(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "service.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	for _, tc := range []struct{ name, body, want string }{
		{"unknown field", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"declarations":{"reconcile":true,"sources":[".vrooli/agent-manager/default.json"],"unknown":true}}}}}}`, "failed to parse declaration config"},
		{"duplicate source", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"declarations":{"reconcile":true,"sources":[".vrooli/agent-manager/default.json",".vrooli/agent-manager/default.json"]}}}}}}`, "duplicate declaration source"},
		{"disabled dependency", `{"dependencies":{"scenarios":{"agent-manager":{"enabled":false}}}}`, "dependency must be enabled"},
		{"missing declarations", `{"dependencies":{"scenarios":{"agent-manager":{"config":{}}}}}`, "config.declarations"},
		{"missing reconcile", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"declarations":{"sources":[".vrooli/agent-manager/default.json"]}}}}}}`, "declarations.reconcile"},
		{"missing sources", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"declarations":{"reconcile":true,"sources":[]}}}}}}`, "must declare at least one source"},
		{"invalid profileMode", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"declarations":{"reconcile":true,"profileMode":"nonsense","sources":[".vrooli/agent-manager/default.json"]}}}}}}`, "invalid profile reconcile mode"},
		{"legacy profiles block", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"profiles":{"reconcile":true,"sources":[".vrooli/agent-profiles/default.json"]}}}}}}`, "legacy declaration block is no longer supported"},
		{"legacy workflows block", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"workflows":{"reconcile":true,"sources":[".vrooli/agent-workflows/x.json"]}}}}}}`, "legacy declaration block is no longer supported"},
		{"old profile dir source", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"declarations":{"reconcile":true,"sources":[".vrooli/agent-profiles/default.json"]}}}}}}`, "must live under .vrooli/agent-manager/"},
		{"old workflow dir source", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"declarations":{"reconcile":true,"sources":[".vrooli/agent-workflows/x.json"]}}}}}}`, "must live under .vrooli/agent-manager/"},
		{"source outside dir", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"declarations":{"reconcile":true,"sources":["config/default.json"]}}}}}}`, "must live under .vrooli/agent-manager/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readScenarioDeclarationConfig(write(t, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestReadScenarioDeclarationConfigAcceptsMixedBlock(t *testing.T) {
	body := declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/round.json")
	path := filepath.Join(t.TempDir(), "service.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := readScenarioDeclarationConfig(path)
	if err != nil {
		t.Fatalf("readScenarioDeclarationConfig() error = %v", err)
	}
	if len(cfg.Declarations.Sources) != 2 || cfg.Declarations.Reconcile == nil || !*cfg.Declarations.Reconcile {
		t.Fatalf("unexpected config: %+v", cfg.Declarations)
	}
}

func TestPeekSchemaVersion(t *testing.T) {
	for _, tc := range []struct{ name, data, want string }{
		{"profile", fixtureProfile, "agent-profile/v1"},
		{"workflow", `{"schemaVersion":"agent-workflow/v1"}`, "agent-workflow/v1"},
		{"missing", `{"name":"x"}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := peekSchemaVersion([]byte(tc.data))
			if err != nil {
				t.Fatalf("peekSchemaVersion() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("peekSchemaVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReconcileScenarioDeclarationsMixedKind(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/round.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
		".vrooli/agent-manager/round.json":   fixtureWorkflow,
	})

	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.ProfilesCreated != 1 || res.ProfilesFailed != 0 {
		t.Fatalf("profiles: created=%d failed=%d results=%+v", res.ProfilesCreated, res.ProfilesFailed, res.ProfileResults)
	}
	if res.WorkflowsCreated != 1 || res.WorkflowsFailed != 0 {
		t.Fatalf("workflows: created=%d failed=%d results=%+v", res.WorkflowsCreated, res.WorkflowsFailed, res.WorkflowResults)
	}
	if len(res.ProfileResults) != 1 || res.ProfileResults[0].ProfileKey != "fixture-scn/default" {
		t.Fatalf("unexpected profile results: %+v", res.ProfileResults)
	}
	if len(res.WorkflowResults) != 1 || res.WorkflowResults[0].WorkflowKey != "fixture-scn/round" {
		t.Fatalf("unexpected workflow results: %+v", res.WorkflowResults)
	}

	// The delegates project each half; verify the projections separately.
	prof := &ReconcileScenarioProfilesResult{Scenario: res.Scenario, DryRun: res.DryRun}
	for _, item := range res.ProfileResults {
		prof.add(item)
	}
	if prof.Created != 1 {
		t.Fatalf("profile delegate projection created=%d", prof.Created)
	}
}

func TestReconcileScenarioDeclarationsProfileDrift(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
	})

	first, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil || first.ProfilesCreated != 1 {
		t.Fatalf("first reconcile: created=%d err=%v", first.ProfilesCreated, err)
	}
	second, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil || second.ProfilesUnchanged != 1 {
		t.Fatalf("second reconcile expected unchanged, got %+v err=%v", second, err)
	}

	// A local override without force mode conflicts rather than clobbering.
	id, err := uuid.Parse(first.ProfileResults[0].ProfileID)
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}
	profile, err := o.GetProfile(ctx, id)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	profile.MaxTurns++
	if _, err := o.UpdateProfile(ctx, profile); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	third, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil || third.ProfilesConflicted != 1 {
		t.Fatalf("third reconcile expected conflict, got %+v err=%v", third, err)
	}
}

func TestReconcileScenarioDeclarationsForbiddenRuntimeField(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	badProfile := `{"schemaVersion":"agent-profile/v1","name":"x","profileKey":"fixture-scn/default","roleRef":"code.smart","ownerScenario":"fixture-scn"}`
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json"),
		".vrooli/agent-manager/default.json": badProfile,
	})
	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.ProfilesFailed != 1 || !strings.Contains(res.ProfileResults[0].Message, "runtime field") {
		t.Fatalf("expected forbidden-field failure, got %+v", res.ProfileResults)
	}
}

func TestReconcileScenarioDeclarationsCanonicalToolDiagnostics(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	bad := strings.Replace(fixtureProfile, `"createdBy": "fixture-scn"`, `"allowedTools":["analyze_code"],"createdBy": "fixture-scn"`, 1)
	root, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json"),
		".vrooli/agent-manager/default.json": bad,
	})
	result, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", root, servicePath, false, false)
	if err != nil || result.ProfilesFailed != 1 || !strings.Contains(result.ProfileResults[0].Message, "analyze_code") || !strings.Contains(result.ProfileResults[0].Message, "nearest match") {
		t.Fatalf("canonical tool diagnostic: result=%+v err=%v", result, err)
	}

	warning := strings.Replace(fixtureProfile, `"createdBy": "fixture-scn"`, `"allowedTools":["read"],"skipPermissionPrompt":true,"createdBy": "fixture-scn"`, 1)
	root, servicePath = writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json"),
		".vrooli/agent-manager/default.json": warning,
	})
	result, err = o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", root, servicePath, false, false)
	if err != nil || result.ProfilesCreated != 1 || len(result.ProfileResults[0].Diagnostics) != 1 || result.ProfileResults[0].Diagnostics[0].Severity != domain.DiagnosticSeverityWarning {
		t.Fatalf("skip permission warning: result=%+v err=%v", result, err)
	}
}

func TestReconcileScenarioDeclarationsUnknownSchemaVersion(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/mystery.json"),
		".vrooli/agent-manager/mystery.json": `{"schemaVersion":"agent-mystery/v9","name":"x"}`,
	})
	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.ProfilesFailed != 1 || !strings.Contains(res.ProfileResults[0].Message, "unknown schemaVersion") {
		t.Fatalf("expected unknown-schemaVersion failure, got %+v", res.ProfileResults)
	}
}

func TestReconcileScenarioDeclarationsWorkflowAtomicWithhold(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	// One valid workflow and one with a mismatched owner; the whole batch must
	// be withheld, leaving nothing activated.
	badWorkflow := strings.Replace(fixtureWorkflow, `"key": "fixture-scn/round"`, `"key": "fixture-scn/other"`, 1)
	badWorkflow = strings.Replace(badWorkflow, `"owner": "fixture-scn"`, `"owner": "someone-else"`, 1)
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/round.json", ".vrooli/agent-manager/bad.json", ".vrooli/agent-manager/default.json"),
		".vrooli/agent-manager/round.json":   fixtureWorkflow,
		".vrooli/agent-manager/bad.json":     badWorkflow,
		".vrooli/agent-manager/default.json": fixtureProfile,
	})
	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.WorkflowsCreated != 0 || res.WorkflowsActivated != 0 {
		t.Fatalf("expected nothing activated on atomic withhold, got %+v", res.WorkflowResults)
	}
	if res.WorkflowsFailed == 0 || res.WorkflowsSkipped == 0 {
		t.Fatalf("expected a failed source and a withheld source, got %+v", res.WorkflowResults)
	}
}

func TestReconcileScenarioDeclarationsReconcileDisabled(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	manifest := `{"dependencies":{"scenarios":{"agent-manager":{"enabled":true,"config":{"declarations":{"reconcile":false,"sources":[".vrooli/agent-manager/default.json"]}}}}}}`
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               manifest,
		".vrooli/agent-manager/default.json": fixtureProfile,
	})
	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.ProfilesSkipped != 1 || res.ProfilesCreated != 0 {
		t.Fatalf("expected skipped profile when reconcile disabled, got %+v", res.ProfileResults)
	}
}

func TestReconcileDeclaringScenariosIsolation(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	repoRoot := t.TempDir()

	// good: valid new-layout declaration whose profile key is owned by "good".
	writeScenarioFile(t, repoRoot, "good", ".vrooli/service.json", declarationManifest(".vrooli/agent-manager/default.json"))
	writeScenarioFile(t, repoRoot, "good", ".vrooli/agent-manager/default.json", strings.ReplaceAll(fixtureProfile, "fixture-scn", "good"))

	// bad: declares the block but the source is missing on disk.
	writeScenarioFile(t, repoRoot, "bad", ".vrooli/service.json", declarationManifest(".vrooli/agent-manager/missing.json"))

	// nodecl: no declaration block, must be skipped.
	writeScenarioFile(t, repoRoot, "nodecl", ".vrooli/service.json", `{"dependencies":{"scenarios":{"agent-manager":{"enabled":true}}}}`)

	summary := o.ReconcileDeclaringScenarios(ctx, repoRoot)
	if summary.Declaring != 2 {
		t.Fatalf("expected 2 declaring scenarios, got %d (summary=%+v)", summary.Declaring, summary)
	}
	if summary.Reconciled != 1 || summary.Failed != 1 {
		t.Fatalf("expected 1 reconciled + 1 failed, got %+v", summary)
	}
}

// selfProfile / selfWorkflow are agent-manager's own declarations (owner and
// profileKey prefix agent-manager) exercised through the self-registration seam.
var (
	selfProfile  = strings.ReplaceAll(fixtureProfile, "fixture-scn", "agent-manager")
	selfWorkflow = strings.ReplaceAll(fixtureWorkflow, "fixture-scn", "agent-manager")
)

// TestReconcileSelfDeclarationsBypassesDependencyGate proves the self-registration
// seam: agent-manager registers its own declaration files (discovered from the
// directory, with no service.json dependency block) through the same shared
// validators, with owner agent-manager.
func TestReconcileSelfDeclarationsBypassesDependencyGate(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	scenarioRoot := t.TempDir()
	writeSelfFile(t, scenarioRoot, "default.json", selfProfile)
	writeSelfFile(t, scenarioRoot, "round.json", selfWorkflow)

	res, err := o.reconcileSelfDeclarationsAt(ctx, scenarioRoot, false, false)
	if err != nil {
		t.Fatalf("self reconcile error = %v", err)
	}
	if res.Scenario != "agent-manager" {
		t.Fatalf("owner scenario = %q, want agent-manager", res.Scenario)
	}
	if res.ProfilesCreated != 1 || res.ProfilesFailed != 0 {
		t.Fatalf("profiles: created=%d failed=%d results=%+v", res.ProfilesCreated, res.ProfilesFailed, res.ProfileResults)
	}
	if res.WorkflowsCreated != 1 || res.WorkflowsFailed != 0 {
		t.Fatalf("workflows: created=%d failed=%d results=%+v", res.WorkflowsCreated, res.WorkflowsFailed, res.WorkflowResults)
	}
	if res.ProfileResults[0].ProfileKey != "agent-manager/default" || res.WorkflowResults[0].WorkflowKey != "agent-manager/round" {
		t.Fatalf("unexpected keys: profile=%q workflow=%q", res.ProfileResults[0].ProfileKey, res.WorkflowResults[0].WorkflowKey)
	}
}

// TestReconcileSelfDeclarationsEmpty confirms the seam is inert until
// agent-manager adds its own declaration files: a missing directory is a
// successful no-op, not an error.
func TestReconcileSelfDeclarationsEmpty(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	res, err := o.reconcileSelfDeclarationsAt(context.Background(), t.TempDir(), false, false)
	if err != nil {
		t.Fatalf("empty self reconcile error = %v", err)
	}
	if len(res.ProfileResults) != 0 || len(res.WorkflowResults) != 0 {
		t.Fatalf("expected no results for empty directory, got %+v / %+v", res.ProfileResults, res.WorkflowResults)
	}
}

// TestReconcileSelfDeclarationsEnforcesValidators proves the seam bypasses only
// the dependency-declaration gate, not ownership: a file owned by another
// scenario placed under agent-manager's directory still fails validation.
func TestReconcileSelfDeclarationsEnforcesValidators(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	scenarioRoot := t.TempDir()
	foreign := strings.Replace(selfProfile, `"profileKey": "agent-manager/default"`, `"profileKey": "someone-else/default"`, 1)
	writeSelfFile(t, scenarioRoot, "foreign.json", foreign)

	res, err := o.reconcileSelfDeclarationsAt(context.Background(), scenarioRoot, false, false)
	if err != nil {
		t.Fatalf("self reconcile error = %v", err)
	}
	if res.ProfilesFailed != 1 || res.ProfilesCreated != 0 {
		t.Fatalf("expected ownership failure, got %+v", res.ProfileResults)
	}
}

// [REQ:REQ-P2-005] TestReconcileScenarioDeclarationsProfileFailureIsolatedFromWorkflow proves the
// per-kind partial-failure contract in one direction: a single bad profile fails
// in isolation (its own diagnostic) while a good profile and a good workflow that
// depends on it both reconcile. Profiles are per-source isolated, so one broken
// profile never withholds a sibling profile or the workflow batch.
func TestReconcileScenarioDeclarationsProfileFailureIsolatedFromWorkflow(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	// brokenProfile carries a forbidden runtime field; the good workflow references
	// the GOOD profile (fixture-scn/default) so its own reconcile does not depend on
	// the broken one.
	brokenProfile := `{"schemaVersion":"agent-profile/v1","name":"broken","profileKey":"fixture-scn/broken","roleRef":"code.smart","ownerScenario":"fixture-scn"}`
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/broken.json", ".vrooli/agent-manager/round.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
		".vrooli/agent-manager/broken.json":  brokenProfile,
		".vrooli/agent-manager/round.json":   fixtureWorkflow,
	})
	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.ProfilesCreated != 1 || res.ProfilesFailed != 1 {
		t.Fatalf("expected 1 good + 1 failed profile in isolation, got created=%d failed=%d results=%+v", res.ProfilesCreated, res.ProfilesFailed, res.ProfileResults)
	}
	if res.WorkflowsCreated != 1 || res.WorkflowsFailed != 0 {
		t.Fatalf("good workflow must reconcile despite the sibling profile failure, got created=%d failed=%d results=%+v", res.WorkflowsCreated, res.WorkflowsFailed, res.WorkflowResults)
	}
}

// [REQ:REQ-P2-005] TestReconcileScenarioDeclarationsWorkflowWithholdKeepsProfile proves the other
// direction: an atomic workflow-batch withhold (owner mismatch) does not roll back
// a profile that was already reconciled in the same call. Profile mutation is
// per-source isolated; workflow activation is all-or-nothing.
func TestReconcileScenarioDeclarationsWorkflowWithholdKeepsProfile(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	badWorkflow := strings.Replace(fixtureWorkflow, `"owner": "fixture-scn"`, `"owner": "someone-else"`, 1)
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/bad.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
		".vrooli/agent-manager/bad.json":     badWorkflow,
	})
	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.ProfilesCreated != 1 {
		t.Fatalf("profile must survive a workflow-batch withhold, got created=%d results=%+v", res.ProfilesCreated, res.ProfileResults)
	}
	if res.WorkflowsCreated != 0 || res.WorkflowsFailed == 0 {
		t.Fatalf("bad workflow must fail with nothing activated, got created=%d failed=%d results=%+v", res.WorkflowsCreated, res.WorkflowsFailed, res.WorkflowResults)
	}
}

// [REQ:REQ-P2-005] TestReconcileScenarioDeclarationsWorkflowWithholdOnRegistrationDefect proves the
// Phase-2 registration validators run at reconcile, not just at file validate: a
// malformed CEL edge condition and an unbound prompt placeholder each fail the
// workflow at reconcile with the precise diagnostic, activating nothing.
func TestReconcileScenarioDeclarationsWorkflowWithholdOnRegistrationDefect(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mutate   func(string) string
		wantCode string
	}{
		{
			name: "malformed CEL edge condition",
			mutate: func(w string) string {
				return strings.Replace(w, `{"from": "work", "to": "done"}`, `{"from": "work", "to": "done", "condition": "iteration <"}`, 1)
			},
			wantCode: "edge_condition",
		},
		{
			name: "unbound prompt placeholder",
			mutate: func(w string) string {
				return strings.Replace(w, `"promptTemplate": "Do the thing."`, `"promptTemplate": "Do {{.missing}}."`, 1)
			},
			wantCode: "prompt_unbound",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := newDeclarationOrchestrator(t)
			ctx := context.Background()
			scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
				".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/round.json"),
				".vrooli/agent-manager/default.json": fixtureProfile,
				".vrooli/agent-manager/round.json":   tc.mutate(fixtureWorkflow),
			})
			res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
			if err != nil {
				t.Fatalf("reconcile error = %v", err)
			}
			if res.WorkflowsCreated != 0 || res.WorkflowsActivated != 0 || res.WorkflowsFailed != 1 {
				t.Fatalf("registration defect must withhold the workflow, got %+v", res.WorkflowResults)
			}
			found := false
			for _, d := range res.WorkflowResults[0].Diagnostics {
				if d.Code == tc.wantCode {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected diagnostic %q at reconcile, got %+v", tc.wantCode, res.WorkflowResults[0].Diagnostics)
			}
		})
	}
}

// [REQ:REQ-P2-005] TestReconcileSelfDeclarationsIsolatesPerSourceFailure proves the startup
// self-registration seam does not let one broken agent-manager-owned file block
// readiness: a bad profile fails in isolation while a good sibling profile still
// registers, and the sweep returns no error.
func TestReconcileSelfDeclarationsIsolatesPerSourceFailure(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	scenarioRoot := t.TempDir()
	badSelf := `{"schemaVersion":"agent-profile/v1","name":"broken","profileKey":"agent-manager/broken","roleRef":"code.smart","ownerScenario":"agent-manager"}`
	writeSelfFile(t, scenarioRoot, "default.json", selfProfile)
	writeSelfFile(t, scenarioRoot, "broken.json", badSelf)

	res, err := o.reconcileSelfDeclarationsAt(context.Background(), scenarioRoot, false, false)
	if err != nil {
		t.Fatalf("self reconcile must not error on a broken source: %v", err)
	}
	if res.ProfilesCreated != 1 || res.ProfilesFailed != 1 {
		t.Fatalf("expected 1 good + 1 failed self profile in isolation, got created=%d failed=%d results=%+v", res.ProfilesCreated, res.ProfilesFailed, res.ProfileResults)
	}
}

// writeSelfFile writes a file under scenarioRoot/.vrooli/agent-manager/.
func writeSelfFile(t *testing.T, scenarioRoot, name, content string) {
	t.Helper()
	dir := filepath.Join(scenarioRoot, ".vrooli", "agent-manager")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeScenarioFile writes a scenario-relative file under scenarios/<scenario>/.
func writeScenarioFile(t *testing.T, repoRoot, scenario, rel, content string) {
	t.Helper()
	full := filepath.Join(repoRoot, "scenarios", scenario, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
