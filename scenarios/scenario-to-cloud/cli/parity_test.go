package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	deploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans"
	"google.golang.org/protobuf/proto"

	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/protoout"
	"scenario-to-cloud/cli/internal/testfakes"
)

// parityServer is a fake API with one deployment, a scripted plan and a
// scripted operation, wrapped so every request's Authorization header is
// recorded.
type parityServer struct {
	*testfakes.Server
	URL     string
	headers []string
}

func newParityServer(t *testing.T) *parityServer {
	t.Helper()
	fake := testfakes.NewServer()
	fake.Deployments.Items = []testfakes.Deployment{{Ref: testfakes.Ref("dep-1", "demo", "production", "203.0.113.10"), Name: "demo", Status: "deployed", Domain: "demo.example.com", BundleSHA: "ad99"}}
	fake.Plans.Compile = func(id, scope string) *plansv1.CompilePlanResponse {
		if scope == "" {
			scope = "full"
		}
		return &plansv1.CompilePlanResponse{
			SchemaVersion: "1", PlanDigest: "sha256:reviewed-" + scope,
			Plan: &plansv1.ExecutablePlan{SchemaVersion: "1", DeploymentId: id, ScenarioId: "demo", Environment: "production", Scope: scope, Outcome: "apply", DesiredRevision: 4, ReleaseDigest: "sha256:ad99", Presentation: &plansv1.Presentation{Summary: "deliver and activate release ad99"}},
			Preview: &plansv1.Preview{
				Target: "host:203.0.113.10", Outcome: "apply",
				Changes:          []*plansv1.Change{{ActionId: "release.stage", Effect: "deployment_write", Capability: "cloud-target:release.stage", Verification: "staged_tree_complete", Recovery: "discard_owned_inactive_stage", Retry: "safe_replay", CancelPoint: true}, {ActionId: "release.activate", Effect: "deployment_write", Retry: "recover"}},
				DataEffects:      []*plansv1.DataEffect{{ActionId: "release.activate", Subject: "postgres:app", Effect: "preserved"}},
				Downtime:         &plansv1.Downtime{ExpectedSeconds: 60, Reason: "maintenance strategy"},
				RecoveryStrategy: "rollback_to_predecessor",
				ShellPreview:     []*plansv1.ShellPreviewLine{{ActionId: "release.stage", Command: "vrooli cloud-target release stage --release sha256:ad99"}},
			},
			ClosureStatus: "derived",
		}
	}
	fake.Plans.Apply = func(req *plansv1.ApplyPlanRequest) *plansv1.ApplyPlanResponse {
		return &plansv1.ApplyPlanResponse{SchemaVersion: "1", OperationId: "op-1", PlanDigest: req.GetPlanDigest(), State: "admitted"}
	}
	fake.Operations.Script = []*operationsv1.OperationStanding{testfakes.Standing("op-1", "dep-1", "succeeded", true)}
	ps := &parityServer{Server: fake}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ps.headers = append(ps.headers, r.Header.Get("Authorization"))
		fake.Mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	ps.URL = srv.URL
	t.Setenv("SCENARIO_TO_CLOUD_API_BASE", srv.URL)
	return ps
}

func runCapture(t *testing.T, app *App, args ...string) (string, error) {
	t.Helper()
	var err error
	out := captureStdout(t, func() { err = app.Run(args) })
	return out, err
}

// TestExecuteIsPlanReviewApplyWait [REQ:STC-P0-037] proves `deployment
// execute` compiles the plan, prints the review, applies exactly the digest
// it showed and waits once on the admitted operation (exit 0), printing the
// identities consistently.
func TestExecuteIsPlanReviewApplyWait(t *testing.T) {
	app := newTestApp(t)
	ps := newParityServer(t)
	out, err := runCapture(t, app, "deployment", "execute", "--scenario", "demo", "--environment", "production", "--request-key", "rk-1", "--show-commands")
	if err != nil {
		t.Fatalf("execute: %v\n%s", err, out)
	}
	for _, want := range []string{
		"deployment: dep-1  scenario: demo  environment: production  target: host:203.0.113.10 (ssh)",
		"plan digest: sha256:reviewed-full", "release digest: sha256:ad99",
		"1. release.stage  [deployment_write]", "capability=cloud-target:release.stage", "cancel-point",
		"data effects:", "postgres:app: preserved", "downtime: ~60s (maintenance strategy)", "recovery strategy: rollback_to_predecessor",
		"shell preview (derived", "vrooli cloud-target release stage",
		"operation: op-1  state: admitted", "request key: rk-1", "Operation op-1 (deployment dep-1)", "lifecycle: succeeded",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("execute output missing %q:\n%s", want, out)
		}
	}
	if len(ps.Plans.Applied) != 1 || ps.Plans.Applied[0].GetPlanDigest() != "sha256:reviewed-full" || ps.Plans.Applied[0].GetRequestKey() != "rk-1" || ps.Plans.Applied[0].GetRunPreflight() {
		t.Fatalf("applied = %+v", ps.Plans.Applied)
	}
	if len(ps.Operations.WaitCalls) != 1 || ps.Operations.WaitCalls[0].GetOperationId() != "op-1" || ps.Operations.WaitCalls[0].GetTimeoutSeconds() != 300 {
		t.Fatalf("wait calls = %+v", ps.Operations.WaitCalls)
	}
	if len(ps.Operations.GetCalls) != 0 {
		t.Fatalf("execute must wait server-side, never poll: %d get calls", len(ps.Operations.GetCalls))
	}
}

// TestPlanThenApplyUsesTheReviewedDigest [REQ:STC-P0-037] proves the
// two-step review: `plan` prints a digest, `apply --plan-digest` admits it,
// and a stale digest is the server's typed refusal (exit 2, nothing waited).
func TestPlanThenApplyUsesTheReviewedDigest(t *testing.T) {
	app := newTestApp(t)
	ps := newParityServer(t)
	out, err := runCapture(t, app, "deployment", "plan", "dep-1", "--scope", "start")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if !strings.Contains(out, "plan digest: sha256:reviewed-start") || len(ps.Plans.Applied) != 0 {
		t.Fatalf("plan must preview only:\n%s", out)
	}
	if _, err := runCapture(t, app, "deployment", "apply", "dep-1", "--scope", "start", "--plan-digest", "sha256:reviewed-start", "--no-wait"); apierr.ExitCode(err) != apierr.ExitPending {
		t.Fatalf("apply --no-wait must exit 3 (admitted, pending): %v", err)
	}
	if len(ps.Plans.Applied) != 1 || ps.Plans.Applied[0].GetPlanDigest() != "sha256:reviewed-start" || ps.Plans.Applied[0].GetScope() != "start" {
		t.Fatalf("applied = %+v", ps.Plans.Applied)
	}
	if _, err := runCapture(t, app, "deployment", "apply", "dep-1"); apierr.ExitCode(err) != apierr.ExitRefused {
		t.Fatalf("apply without --plan-digest must be refused locally (exit 2): %v", err)
	}
	ps.Plans.ApplyError = testfakes.TypedError(connect.CodeAborted, "plan_digest_mismatch", "The reviewed plan no longer matches", map[string]any{"reviewed": "sha256:old", "current": "sha256:reviewed-full"})
	_, err = runCapture(t, app, "deployment", "apply", "dep-1", "--plan-digest", "sha256:old")
	if apierr.ExitCode(err) != apierr.ExitRefused || !strings.Contains(apierr.Format(err), "plan_digest_mismatch") {
		t.Fatalf("stale digest must exit 2 with the stable code: %d %s", apierr.ExitCode(err), apierr.Format(err))
	}
	if len(ps.Operations.WaitCalls) != 0 {
		t.Fatal("a refused apply must not wait on anything")
	}
}

// TestNoOpAndNeedsInputOutcomes [REQ:STC-P0-037] proves a satisfied desired
// state exits 0 with "no change" and no apply call, and a plan that needs
// input exits 3 with the durable onboarding handoff, never prompting.
func TestNoOpAndNeedsInputOutcomes(t *testing.T) {
	app := newTestApp(t)
	ps := newParityServer(t)
	ps.Plans.Compile = func(id, scope string) *plansv1.CompilePlanResponse {
		return &plansv1.CompilePlanResponse{SchemaVersion: "1", PlanDigest: "sha256:same", Plan: &plansv1.ExecutablePlan{DeploymentId: id, Outcome: "no_op"}, Preview: &plansv1.Preview{Outcome: "no_op"}}
	}
	out, err := runCapture(t, app, "deployment", "execute", "dep-1")
	if err != nil || !strings.Contains(out, "no change") {
		t.Fatalf("no_op must exit 0 with no change: err=%v\n%s", err, out)
	}
	ps.Plans.Compile = func(id, scope string) *plansv1.CompilePlanResponse {
		handoff := &plansv1.Handoff{Owner: "vrooli-onboarding", Kind: "resume_handoff", Reference: "vrooli-onboarding://deployments/dep-1/resume/abc", Missing: []string{"postgres:password"}}
		return &plansv1.CompilePlanResponse{SchemaVersion: "1", PlanDigest: "sha256:needs", Plan: &plansv1.ExecutablePlan{DeploymentId: id, Outcome: "needs_input", Handoff: handoff}, Preview: &plansv1.Preview{Outcome: "needs_input", Handoff: handoff}}
	}
	out, err = runCapture(t, app, "deployment", "execute", "dep-1")
	if apierr.ExitCode(err) != apierr.ExitPending {
		t.Fatalf("needs_input must exit 3: %v", err)
	}
	msg := apierr.Format(err)
	for _, want := range []string{"vrooli-onboarding://deployments/dep-1/resume/abc", "postgres:password"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("handoff message missing %q: %s", want, msg)
		}
	}
	if !strings.Contains(out, "handoff: vrooli-onboarding (resume_handoff)") {
		t.Fatalf("review must print the handoff:\n%s", out)
	}
	if len(ps.Plans.Applied) != 0 {
		t.Fatalf("no_op and needs_input must never apply: %+v", ps.Plans.Applied)
	}
}

// TestAmbiguousSelectorRefusesBeforeAnyMutation [REQ:STC-P0-037] proves an
// ambiguous selector is the server's typed refusal (exit 2, candidates
// listed) and that no plan is compiled or applied.
func TestAmbiguousSelectorRefusesBeforeAnyMutation(t *testing.T) {
	app := newTestApp(t)
	ps := newParityServer(t)
	ps.Deployments.Items = append(ps.Deployments.Items, testfakes.Deployment{Ref: testfakes.Ref("dep-2", "demo", "staging", "203.0.113.11"), Name: "demo-staging", Status: "deployed", Domain: "demo.example.com"})
	_, err := runCapture(t, app, "deployment", "execute", "--scenario", "demo", "--domain", "demo.example.com", "--yes")
	if apierr.ExitCode(err) != apierr.ExitRefused {
		t.Fatalf("ambiguous selector must exit 2: %v", err)
	}
	msg := apierr.Format(err)
	for _, want := range []string{"deployment_selector_ambiguous", "Candidates:", "--deployment dep-1", "--deployment dep-2", "environment=staging", "Select by deployment id"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("ambiguity message missing %q: %s", want, msg)
		}
	}
	if len(ps.Plans.Compiled) != 0 || len(ps.Plans.Applied) != 0 || len(ps.Operations.WaitCalls) != 0 {
		t.Fatalf("ambiguity must stop before any plan or apply: compiled=%v applied=%v", ps.Plans.Compiled, ps.Plans.Applied)
	}
	if _, err := runCapture(t, app, "deployment", "execute", "--scenario", "demo", "--domain", "demo.example.com", "--host", "203.0.113.10"); apierr.ExitCode(err) != apierr.ExitRefused {
		t.Fatalf("two facets must be refused locally: %v", err)
	}
	if len(ps.Deployments.Calls) != 1 {
		t.Fatalf("a grammar violation must not reach the server: %v", ps.Deployments.Calls)
	}
}

// TestFailedAndTimedOutOperationsNeverImplySuccess [REQ:STC-P0-037] proves a
// failed operation exits 1 and an observer timeout exits 124 leaving the
// operation untouched and naming the reattach command.
func TestFailedAndTimedOutOperationsNeverImplySuccess(t *testing.T) {
	app := newTestApp(t)
	ps := newParityServer(t)
	failed := testfakes.Standing("op-1", "dep-1", "failed", true)
	failed.Result = &operationsv1.OperationResult{Outcome: "failed", RecoveryOutcome: "service_restored", CompletedSteps: 1}
	ps.Operations.Script = []*operationsv1.OperationStanding{failed}
	out, err := runCapture(t, app, "deployment", "execute", "dep-1", "--yes")
	if apierr.ExitCode(err) != apierr.ExitFailed || !strings.Contains(out, "recovery: service_restored") {
		t.Fatalf("failed operation must exit 1 even when recovery restored service: %v\n%s", err, out)
	}
	still := testfakes.Standing("op-1", "dep-1", "running", false)
	still.StillPending = true
	ps.Operations.Script = []*operationsv1.OperationStanding{still}
	_, err = runCapture(t, app, "deployment", "execute", "dep-1", "--yes", "--timeout", "2s")
	if apierr.ExitCode(err) != apierr.ExitObserverTimeout {
		t.Fatalf("observer timeout must exit 124: %v", err)
	}
	if !strings.Contains(apierr.Format(err), "scenario-to-cloud operation wait op-1") || len(ps.Operations.Cancelled) != 0 {
		t.Fatalf("timeout must print the reattach command and cancel nothing: %s", apierr.Format(err))
	}
	if ps.Operations.WaitCalls[len(ps.Operations.WaitCalls)-1].GetTimeoutSeconds() != 2 {
		t.Fatalf("wait bound not forwarded: %+v", ps.Operations.WaitCalls)
	}
}

// TestAPIStartedOperationIsResumedByTheCLI [REQ:STC-P0-037] proves an
// operation admitted elsewhere (API or UI) is attached by id: resume and
// list report the same identity and state without creating anything.
func TestAPIStartedOperationIsResumedByTheCLI(t *testing.T) {
	app := newTestApp(t)
	ps := newParityServer(t)
	ps.Operations.Script = []*operationsv1.OperationStanding{testfakes.Standing("op-ui-42", "dep-1", "verifying", false)}
	out, err := runCapture(t, app, "operation", "resume", "op-ui-42")
	if apierr.ExitCode(err) != apierr.ExitPending {
		t.Fatalf("resume of a live operation must exit 3: %v", err)
	}
	for _, want := range []string{"Operation op-ui-42 (deployment dep-1)", "lifecycle: verifying", "plan digest: sha256:op-ui-42", "request key: key-op-ui-42"} {
		if !strings.Contains(out, want) {
			t.Fatalf("resume output missing %q:\n%s", want, out)
		}
	}
	out, err = runCapture(t, app, "operation", "list", "--scenario", "demo", "--host", "203.0.113.10")
	if err != nil || !strings.Contains(out, "op-ui-42") || !strings.Contains(out, "verifying") {
		t.Fatalf("list must show the API-started operation: %v\n%s", err, out)
	}
	if len(ps.Plans.Applied) != 0 {
		t.Fatal("resume must not admit anything")
	}
}

// TestJSONOutputIsLosslessProtoJSON [REQ:STC-P0-037] proves --json output
// round-trips through the generated message with proto field names.
func TestJSONOutputIsLosslessProtoJSON(t *testing.T) {
	app := newTestApp(t)
	newParityServer(t)
	out, err := runCapture(t, app, "deployment", "resolve", "--scenario", "demo", "--host", "203.0.113.10", "--json")
	if err != nil {
		t.Fatalf("resolve --json: %v", err)
	}
	var resolved deploymentsv1.ResolveDeploymentResponse
	if err := protoout.Unmarshal([]byte(out), &resolved); err != nil {
		t.Fatalf("output is not proto JSON: %v\n%s", err, out)
	}
	if !proto.Equal(resolved.GetRef(), testfakes.Ref("dep-1", "demo", "production", "203.0.113.10")) {
		t.Fatalf("round trip changed the reference:\n%s", out)
	}
	// protojson varies its whitespace on purpose; assert field names only.
	for _, want := range []string{`"scenario_id":`, `"schema_version":`, `"locator":`} {
		if !strings.Contains(out, want) {
			t.Fatalf("proto JSON missing %s:\n%s", want, out)
		}
	}
	out, err = runCapture(t, app, "deployment", "plan", "dep-1", "--json")
	if err != nil {
		t.Fatalf("plan --json: %v", err)
	}
	var compiled plansv1.CompilePlanResponse
	if err := protoout.Unmarshal([]byte(out), &compiled); err != nil || compiled.GetPlanDigest() != "sha256:reviewed-full" {
		t.Fatalf("plan JSON round trip: %v digest=%s\n%s", err, compiled.GetPlanDigest(), out)
	}
}

// TestOperatorTokenIsAttachedToConnectCalls [REQ:STC-P0-037] [REQ:STC-P0-012]
// proves the generated-client path carries the operator bearer token.
func TestOperatorTokenIsAttachedToConnectCalls(t *testing.T) {
	app := newTestApp(t)
	ps := newParityServer(t)
	if _, err := runCapture(t, app, "deployment", "get", "dep-1"); err != nil {
		t.Fatalf("get: %v", err)
	}
	seen := 0
	for _, h := range ps.headers {
		if h == "Bearer test-token" {
			seen++
		}
	}
	if seen < 2 {
		t.Fatalf("expected the bearer token on the resolve and get Connect calls, headers=%v", ps.headers)
	}
}

// TestLogsAreBoundedAndCarryNoCanary [REQ:STC-P0-037] proves log retrieval
// is bounded (--tail capped, --max-bytes honoured) and prints only what the
// API returns after its redaction: the fake stores a canary and redacts it.
func TestLogsAreBoundedAndCarryNoCanary(t *testing.T) {
	const canary = "canary-secret-9f3a7c1e"
	app := newTestApp(t)
	ps := newParityServer(t)
	stored := []string{"listening on :3000", "connecting with password " + canary, "ready"}
	var tails []string
	ps.Mux.HandleFunc("/api/v1/deployments/dep-1/logs", func(w http.ResponseWriter, r *http.Request) {
		tails = append(tails, r.URL.Query().Get("tail"))
		w.Header().Set("Content-Type", "application/json")
		entries := make([]string, 0, len(stored))
		for _, line := range stored {
			entries = append(entries, fmt.Sprintf(`{"timestamp":"t","source":"app","level":"info","message":%q}`, strings.ReplaceAll(line, canary, "[REDACTED]")))
		}
		fmt.Fprintf(w, `{"deployment_id":"dep-1","logs":[%s],"total_count":%d,"has_more":false,"timestamp":"t"}`, strings.Join(entries, ","), len(stored))
	})
	out, err := runCapture(t, app, "inspect", "logs", "dep-1", "--tail", "50")
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if strings.Contains(out, canary) || !strings.Contains(out, "[REDACTED]") || tails[0] != "50" {
		t.Fatalf("logs output must be the API's redacted content: tail=%v\n%s", tails, out)
	}
	if _, err := runCapture(t, app, "inspect", "logs", "dep-1", "--tail", "5000"); err == nil || !strings.Contains(err.Error(), "exceeds the API bound") {
		t.Fatalf("tail above the API bound must be refused: %v", err)
	}
	out, err = runCapture(t, app, "inspect", "logs", "dep-1", "--max-bytes", "80")
	if err != nil || !strings.Contains(out, "output bounded at 80 bytes") {
		t.Fatalf("max-bytes must bound the output: %v\n%s", err, out)
	}
}

// TestLegacyAliasesThatBypassedPlanChecksAreGone [REQ:STC-P0-037] proves the
// removed flags and modes are refused rather than silently ignored.
func TestLegacyAliasesThatBypassedPlanChecksAreGone(t *testing.T) {
	app := newTestApp(t)
	newParityServer(t)
	cases := [][]string{
		{"redeploy", "--domain", "demo.example.com", "--scenario", "demo", "--if-needed"},
		{"redeploy", "--domain", "demo.example.com", "--scenario", "demo", "--force-run"},
		{"deployment", "execute", "dep-1", "--stream"},
		{"deployment", "execute", "dep-1", "--wait"},
		{"deployment", "resolve", "--target", "demo.example.com"},
		{"deployment", "health", "--target", "demo.example.com"},
	}
	for _, args := range cases {
		_, err := runCapture(t, app, args...)
		if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
			t.Fatalf("%v must be refused as an unknown flag, got %v", args, err)
		}
	}
}
