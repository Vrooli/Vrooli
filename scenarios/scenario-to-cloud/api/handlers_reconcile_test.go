package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/reconcile"
)

type stubTargetObserver struct {
	state *TargetState
	err   error
}

func (s stubTargetObserver) ObserveTarget(context.Context, *DeploymentContext) (*TargetState, error) {
	return s.state, s.err
}

// inventoryReach answers `cloud-target data inventory` with a canned
// heuristic listing and records every command.
type inventoryReach struct {
	commands []reach.Command
}

func (r *inventoryReach) Exec(_ context.Context, _ identity.TargetRef, cmd reach.Command) (reach.Result, error) {
	r.commands = append(r.commands, cmd)
	if cmd.Verb != "cloud-target data inventory" {
		return reach.Result{ExitCode: 2, Stdout: `{"error":{"code":"unexpected_verb","message":"` + cmd.Verb + `"}}`}, nil
	}
	var bindings []map[string]string
	for i, a := range cmd.Args {
		if a == "--bindings" {
			raw, _ := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(cmd.Args[i+1], "b64:"))
			_ = json.Unmarshal(raw, &bindings)
		}
	}
	covered := func(path string) (bool, string) {
		for _, b := range bindings {
			if b["path"] == path {
				return true, b["id"]
			}
		}
		return false, ""
	}
	entries := []map[string]any{}
	for _, path := range []string{"api/uploads", "cache"} {
		ok, id := covered(path)
		entries = append(entries, map[string]any{"path": path, "bytes": 5, "files": 1, "covered": ok, "binding_id": id})
	}
	raw, _ := json.Marshal(map[string]any{"scenario": "app", "entries": entries})
	return reach.Result{ExitCode: 0, Stdout: string(raw)}, nil
}

func (r *inventoryReach) Deliver(context.Context, identity.TargetRef, reach.Delivery) (reach.DeliveryReceipt, error) {
	return reach.DeliveryReceipt{}, nil
}

func (r *inventoryReach) Negotiate(context.Context, identity.TargetRef) (reach.Capabilities, error) {
	return reach.Capabilities{Online: true, NativeCLI: true, Platform: "linux/amd64"}, nil
}

func reconcileTestServer(t *testing.T, name string) (*Server, *httptest.Server) {
	t.Helper()
	srv, repo, _, _ := planTestServer(t, name)
	seedPlanDeployment(t, repo, "dep-1", nil)
	srv.registerReconcileRoutes(srv.router.PathPrefix("/api/v1").Subrouter())
	runner := operations.RunnerFunc(func(context.Context, *operations.ExecutionContext) error { return nil })
	svc := operations.NewService(operations.Config{WorkerID: "reconcile-test", Workers: 1, LeaseTTL: 200 * time.Millisecond, HeartbeatInterval: 50 * time.Millisecond, ReconcileInterval: time.Hour, ObserverTimeout: 2 * time.Second, Logger: srv.log}, repo, runner, nil, nil)
	svc.Start()
	t.Cleanup(svc.Stop)
	srv.operations = svc
	prevTarget, prevHealth := targetObserverOverride, healthObserverOverride
	t.Cleanup(func() { targetObserverOverride, healthObserverOverride = prevTarget, prevHealth })
	return srv, httptest.NewServer(srv.router)
}

func do(t *testing.T, srv *Server, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var reader *strings.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = strings.NewReader(string(raw))
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	var decoded map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &decoded)
	return rec.Code, decoded
}

// [REQ:STC-P0-028] P15-A01: a stopped intent is recorded, reported through
// the desired-state contract with observation_may_restart=false, and
// reconciliation never proposes a start for it even when health says the
// workload is down.
func TestReconciliationPreservesIntentionalStop(t *testing.T) {
	srv, ts := reconcileTestServer(t, "reconcile-stop")
	defer ts.Close()
	healthObserverOverride = func(context.Context, *DeploymentContext) reconcile.Observed {
		return reconcile.Observed{ReleaseDigest: "sha256:other", Status: "unhealthy", Freshness: "current", ObservedAt: time.Now()}
	}
	targetObserverOverride = stubTargetObserver{state: &TargetState{Releases: []string{}}}

	code, body := do(t, srv, http.MethodGet, "/api/v1/deployments/dep-1/desired-state", nil)
	if code != http.StatusOK || body["desired_state"] != string(domain.DesiredRunning) {
		t.Fatalf("default desired state: %d %v", code, body)
	}
	code, body = do(t, srv, http.MethodPut, "/api/v1/deployments/dep-1/desired-state", map[string]any{"desired_state": "stopped"})
	if code != http.StatusOK || body["desired_state"] != "stopped" || body["observation_may_restart"] != false {
		t.Fatalf("set stopped: %d %v", code, body)
	}
	reboot := body["reboot_policy"].(map[string]any)
	if reboot["decision"] != reconcile.RebootLeaveStopped {
		t.Fatalf("reboot policy = %v", reboot)
	}
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/reconcile", nil)
	if code != http.StatusOK {
		t.Fatalf("reconcile: %d %v", code, body)
	}
	report := body["report"].(map[string]any)
	if report["outcome"] != string(reconcile.Unchanged) || report["correction"] != nil || body["correction"] != nil || body["operation"] != nil {
		t.Fatalf("a stopped deployment must not be corrected: %v", body)
	}
	if code, body := do(t, srv, http.MethodPut, "/api/v1/deployments/dep-1/desired-state", map[string]any{"desired_state": "retired"}); code != http.StatusBadRequest || errorCode(body) != "invalid_request" {
		t.Fatalf("retired must go through the retirement plan: %d %v", code, body)
	}
}

// [REQ:STC-P0-028] P15-A06: drift proposes a plan-scoped correction that
// is applied only through the reviewed digest and the operation owner.
func TestReconciliationCorrectsDriftOnlyThroughTheReviewedPlan(t *testing.T) {
	srv, ts := reconcileTestServer(t, "reconcile-drift")
	defer ts.Close()
	healthObserverOverride = func(context.Context, *DeploymentContext) reconcile.Observed {
		return reconcile.Observed{ReleaseDigest: "sha256:older", ConfigurationDigest: "sha256:cfg", Status: "healthy", Freshness: "current", ObservedAt: time.Now()}
	}
	targetObserverOverride = stubTargetObserver{state: &TargetState{ActiveRelease: strings.Repeat("a", 64), Releases: []string{strings.Repeat("a", 64)}}}
	code, body := do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/reconcile", nil)
	if code != http.StatusOK {
		t.Fatalf("observe: %d %v", code, body)
	}
	report := body["report"].(map[string]any)
	if report["outcome"] != string(reconcile.Changed) || body["correction"] == nil || body["operation"] != nil {
		t.Fatalf("drift must propose without applying: %v", body)
	}
	if report["desired"].(map[string]any)["desired_revision"] == nil || report["observed"].(map[string]any)["observed_at"] == nil {
		t.Fatalf("desired revision and observed timestamp must be separate: %v", report)
	}
	correction := body["correction"].(map[string]any)
	if correction["plan"].(map[string]any)["scope"] != execplan.ScopeRuntime {
		t.Fatalf("correction scope = %v", correction["plan"].(map[string]any)["scope"])
	}
	digest := correction["plan_digest"].(string)
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/reconcile", map[string]any{"plan_digest": "sha256:not-reviewed", "request_key": "k1"})
	if code != http.StatusConflict || errorCode(body) != "plan_digest_mismatch" {
		t.Fatalf("wrong digest must be refused: %d %v", code, body)
	}
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/reconcile", map[string]any{"plan_digest": digest, "request_key": "k1"})
	if code != http.StatusAccepted || body["operation"] == nil || body["operation"].(map[string]any)["plan_digest"] != digest {
		t.Fatalf("apply: %d %v", code, body)
	}

	// An interrupted activation on the target blocks and names the resolution.
	targetObserverOverride = stubTargetObserver{state: &TargetState{ActiveRelease: "old", ActivationInterrupted: true, Candidate: "new", Releases: []string{"old", "new"}}}
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/reconcile", nil)
	report = body["report"].(map[string]any)
	if code != http.StatusOK || report["outcome"] != string(reconcile.Blocked) || report["correction"].(map[string]any)["kind"] != reconcile.CorrectionResolveActivate {
		t.Fatalf("interrupted activation: %d %v", code, body)
	}
}

// [REQ:STC-P0-028] P15-O03: the retirement preview lists retained and
// deleted objects in owner order, requires an explicit data policy when the
// deployment holds data, and refuses to apply a blocked plan.
func TestRetirementPreviewListsRetainedAndDeletedObjects(t *testing.T) {
	srv, ts := reconcileTestServer(t, "reconcile-retire")
	defer ts.Close()
	targetObserverOverride = stubTargetObserver{state: &TargetState{ActiveRelease: "r2", PreviousRelease: "r1", Releases: []string{"r0", "r1", "r2"}}}
	healthObserverOverride = func(context.Context, *DeploymentContext) reconcile.Observed { return reconcile.Observed{} }

	code, body := do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/retire/plan", nil)
	if code != http.StatusOK {
		t.Fatalf("retire plan: %d %v", code, body)
	}
	retirement := body["retirement"].(map[string]any)
	if retirement["outcome"] != string(reconcile.Blocked) || body["plan"] != nil || body["plan_error"] == nil {
		t.Fatalf("persistent data without a policy must block: %v", body)
	}
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/retire/apply", map[string]any{"plan_digest": "x", "request_key": "k"})
	if code != http.StatusBadRequest {
		t.Fatalf("blocked retirement must not apply: %d %v", code, body)
	}
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/retire/plan", map[string]any{"retention_policy": "retain"})
	if code != http.StatusOK {
		t.Fatalf("retire plan retain: %d %v", code, body)
	}
	retirement = body["retirement"].(map[string]any)
	if retirement["outcome"] != string(reconcile.Changed) || body["plan"] == nil {
		t.Fatalf("retain plan = %v", body)
	}
	order := retirement["order"].([]any)
	if order[0] != reconcile.KindRoute || order[1] != reconcile.KindRuntime || order[2] != reconcile.KindGrant || order[3] != reconcile.KindData || order[4] != reconcile.KindArtifact {
		t.Fatalf("order = %v", order)
	}
	kinds := map[string][]string{}
	for _, item := range retirement["deleted"].([]any) {
		o := item.(map[string]any)
		kinds["deleted:"+o["kind"].(string)] = append(kinds["deleted:"+o["kind"].(string)], o["id"].(string))
	}
	for _, item := range retirement["retained"].([]any) {
		o := item.(map[string]any)
		kinds["retained:"+o["kind"].(string)] = append(kinds["retained:"+o["kind"].(string)], o["id"].(string))
	}
	if strings.Join(kinds["deleted:artifact"], ",") != "r0" || strings.Join(kinds["retained:artifact"], ",") != "r1,r2" || len(kinds["retained:data"]) == 0 || strings.Join(kinds["deleted:route"], ",") != "app.example.test" {
		t.Fatalf("dispositions = %v", kinds)
	}
	plan := body["plan"].(map[string]any)
	if plan["plan"].(map[string]any)["scope"] != execplan.ScopeRetire {
		t.Fatalf("plan scope = %v", plan["plan"].(map[string]any)["scope"])
	}
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/retire/apply", map[string]any{"retention_policy": "retain", "plan_digest": plan["plan_digest"], "request_key": "retire-1"})
	if code != http.StatusAccepted || body["operation_id"] == nil {
		t.Fatalf("retire apply: %d %v", code, body)
	}
	code, body = do(t, srv, http.MethodGet, "/api/v1/deployments/dep-1/desired-state", nil)
	if code != http.StatusOK || body["desired_state"] != string(domain.DesiredRetired) {
		t.Fatalf("retired intent must be recorded: %v", body)
	}
}

// [REQ:STC-P0-028] P11-A06: legacy conversion. The inventory is read-only
// and reports unmapped directories; adoption records only declared bindings
// and moves nothing; the recorded mapping reaches the next plan.
func TestDataBindingInventoryAndAdoptionRecordMappingWithoutMovingData(t *testing.T) {
	srv, ts := reconcileTestServer(t, "reconcile-bindings")
	defer ts.Close()
	fake := &inventoryReach{}
	srv.reach = fake
	code, body := do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/data-bindings/inventory", nil)
	if code != http.StatusOK {
		t.Fatalf("inventory: %d %v", code, body)
	}
	unmappedFor := func(body map[string]any, scenario string) []string {
		var out []string
		for _, item := range body["unmapped"].([]any) {
			entry := item.(map[string]any)
			if entry["scenario"] == scenario {
				out = append(out, entry["path"].(string))
			}
		}
		return out
	}
	if got := unmappedFor(body, "app"); len(got) != 2 {
		t.Fatalf("both heuristic directories are unmapped before adoption: %v", body)
	}
	for _, cmd := range fake.commands {
		if cmd.Effectful || cmd.Verb != "cloud-target data inventory" {
			t.Fatalf("inventory must be read-only: %+v", cmd)
		}
	}
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/data-bindings/adopt", map[string]any{"mappings": []map[string]string{{"binding_id": "not-declared", "scenario": "app", "path": "api/uploads"}}})
	if code != http.StatusBadRequest || errorCode(body) != "invalid_request" {
		t.Fatalf("undeclared binding must be refused: %d %v", code, body)
	}
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/data-bindings/adopt", map[string]any{"mappings": []map[string]string{{"binding_id": "application-records", "scenario": "app", "path": "../escape"}}})
	if code != http.StatusBadRequest {
		t.Fatalf("traversal must be refused: %d %v", code, body)
	}
	before := len(fake.commands)
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/data-bindings/adopt", map[string]any{"mappings": []map[string]string{{"binding_id": "application-records", "scenario": "app", "path": "api/uploads"}}})
	if code != http.StatusOK {
		t.Fatalf("adopt: %d %v", code, body)
	}
	if len(fake.commands) != before {
		t.Fatalf("adoption must not reach the target: %+v", fake.commands[before:])
	}
	code, body = do(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/data-bindings/inventory", nil)
	if got := unmappedFor(body, "app"); code != http.StatusOK || len(got) != 1 || got[0] != "cache" {
		t.Fatalf("after adoption only cache stays unmapped: %v", body["unmapped"])
	}
	_, preview := postPlan(t, srv.router, "/api/v1/deployments/dep-1/plan", nil)
	for _, raw := range preview["plan"].(map[string]any)["actions"].([]any) {
		action := raw.(map[string]any)
		if action["id"] == execplan.OpReleaseActivate {
			inputs := action["inputs"].(map[string]any)
			if inputs["data_bindings"] != "application-records=app/api/uploads" {
				t.Fatalf("recorded mapping must reach activation: %v", inputs)
			}
			return
		}
	}
	t.Fatal("release.activate missing from the plan")
}
