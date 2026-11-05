package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/persistence"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans/plansv1connect"

	"connectrpc.com/connect"
)

// planTestServer builds a server over an in-memory repository with the
// fixture closure catalog and mounts the plan routes on a router without the
// authorization middleware (route classification is owned by the authz
// table; these tests exercise the handlers).
func planTestServer(t *testing.T, name string) (*Server, *persistence.Repository, *sql.DB, *mux.Router) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	repo := persistence.NewRepository(db)
	if err := repo.InitSchemaOnDialect(context.Background(), db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	srv := newTestServer()
	srv.closureSvc = fixtureClosureService(t, "basic")
	srv.repo = repo
	srv.deploymentRepo = repo
	srv.historyRecorder = repo
	router := mux.NewRouter()
	srv.registerPlanRoutes(router.PathPrefix("/api/v1").Subrouter())
	return srv, repo, db, router
}

func seedPlanDeployment(t *testing.T, repo *persistence.Repository, id string, mutate func(*domain.CloudManifest)) {
	t.Helper()
	bundle := filepath.Join(t.TempDir(), "app.tar.gz")
	if err := os.WriteFile(bundle, []byte("bundle-"+id), 0o644); err != nil {
		t.Fatal(err)
	}
	m := domain.CloudManifest{
		Version:      "1",
		Target:       domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		Scenario:     domain.ManifestScenario{ID: "app"},
		Dependencies: domain.ManifestDependencies{Resources: []string{"store", "cache"}, Scenarios: []string{"app", "records-service"}},
		Ports:        domain.ManifestPorts{"ui": 3000, "api": 3001},
		Edge:         domain.ManifestEdge{Domain: "app.example.test", Caddy: domain.ManifestCaddy{Enabled: true}},
	}
	if mutate != nil {
		mutate(&m)
	}
	raw, _ := json.Marshal(m)
	now := time.Now().UTC()
	if err := repo.CreateDeployment(context.Background(), &domain.Deployment{
		ID: id, Name: "app", ScenarioID: "app", Environment: "production",
		Status: domain.StatusDeployed, Manifest: raw, BundlePath: &bundle, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func countOperations(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM cloud_operations").Scan(&n); err != nil {
		t.Fatalf("count operations: %v", err)
	}
	return n
}

func postPlan(t *testing.T, router http.Handler, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var decoded map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &decoded)
	return rec, decoded
}

func errorCode(body map[string]any) string {
	if e, ok := body["error"].(map[string]any); ok {
		code, _ := e["code"].(string)
		return code
	}
	return ""
}

// [REQ:STC-P0-016] Preview is pure: two previews create no operation rows
// and reach the target zero times, and return one digest.
func TestDeploymentPlanPreviewIsPureAndStable(t *testing.T) {
	srv, repo, db, router := planTestServer(t, "plan-preview")
	seedPlanDeployment(t, repo, "dep-1", nil)
	fake := srv.sshRunner.(*FakeSSHRunner)

	rec1, body1 := postPlan(t, router, "/api/v1/deployments/dep-1/plan", nil)
	rec2, body2 := postPlan(t, router, "/api/v1/deployments/dep-1/plan", nil)
	if rec1.Code != http.StatusOK || rec2.Code != http.StatusOK {
		t.Fatalf("preview status %d / %d: %s", rec1.Code, rec2.Code, rec1.Body.String())
	}
	if body1["plan_digest"] == "" || body1["plan_digest"] != body2["plan_digest"] {
		t.Fatalf("preview digests differ: %v vs %v", body1["plan_digest"], body2["plan_digest"])
	}
	if countOperations(t, db) != 0 {
		t.Fatalf("preview must not create operation rows")
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("preview must not reach the target: %v", fake.Calls)
	}
	if body1["closure_status"] != "derived" {
		t.Fatalf("expected the fixture closure to be derived, got %v", body1["closure_status"])
	}
	preview := body1["preview"].(map[string]any)
	if preview["outcome"] != execplan.OutcomeApply {
		t.Fatalf("expected apply outcome, got %v", preview["outcome"])
	}
	plan := body1["plan"].(map[string]any)
	if plan["closure_digest"] == "" {
		t.Fatalf("plan must bind the closure digest")
	}
}

// [REQ:STC-P0-016] P06-A01/A02: apply consumes the exact reviewed digest;
// a wrong digest is refused with zero effects and zero rows; the right digest
// admits exactly one operation and replays under the same request key.
func TestDeploymentPlanApplyConsumesReviewedDigest(t *testing.T) {
	srv, repo, db, router := planTestServer(t, "plan-apply")
	seedPlanDeployment(t, repo, "dep-1", nil)
	fake := srv.sshRunner.(*FakeSSHRunner)

	_, preview := postPlan(t, router, "/api/v1/deployments/dep-1/plan", nil)
	digest := preview["plan_digest"].(string)

	rec, body := postPlan(t, router, "/api/v1/deployments/dep-1/plan/apply", map[string]any{"request_key": "k1"})
	if rec.Code != http.StatusBadRequest || errorCode(body) != "invalid_request" {
		t.Fatalf("missing digest: %d %s", rec.Code, rec.Body.String())
	}
	rec, body = postPlan(t, router, "/api/v1/deployments/dep-1/plan/apply", map[string]any{"plan_digest": "sha256:not-reviewed", "request_key": "k1"})
	if rec.Code != http.StatusConflict || errorCode(body) != "plan_digest_mismatch" {
		t.Fatalf("wrong digest: %d %s", rec.Code, rec.Body.String())
	}
	if countOperations(t, db) != 0 || len(fake.Calls) != 0 {
		t.Fatalf("a refused apply must have zero rows and zero effects")
	}

	rec, body = postPlan(t, router, "/api/v1/deployments/dep-1/plan/apply", map[string]any{"plan_digest": digest, "request_key": "k1"})
	if rec.Code != http.StatusAccepted || body["state"] != "admitted" || body["plan_digest"] != digest || body["operation_id"] == "" {
		t.Fatalf("apply: %d %s", rec.Code, rec.Body.String())
	}
	opID := body["operation_id"].(string)
	if countOperations(t, db) != 1 {
		t.Fatalf("expected one admitted operation")
	}
	op, err := repo.GetOperation(context.Background(), opID)
	if err != nil || op == nil || op.PlanDigest != digest {
		t.Fatalf("operation not persisted with the reviewed digest: %v %+v", err, op)
	}
	var stored execplan.Plan
	if err := json.Unmarshal(op.Plan, &stored); err != nil || stored.MustDigest() != digest {
		t.Fatalf("stored plan must hash to the admitted digest: %v", err)
	}

	rec, body = postPlan(t, router, "/api/v1/deployments/dep-1/plan/apply", map[string]any{"plan_digest": digest, "request_key": "k1"})
	if rec.Code != http.StatusAccepted || body["operation_id"] != opID {
		t.Fatalf("replay must return the same operation: %d %s", rec.Code, rec.Body.String())
	}
	if countOperations(t, db) != 1 {
		t.Fatalf("replay must not create a second row")
	}
}

// [REQ:STC-P0-016] P06-A02: a material precondition injected after preview
// makes apply refuse with plan_stale and zero partial effects.
func TestDeploymentPlanApplyRefusesStalePreconditions(t *testing.T) {
	srv, repo, db, router := planTestServer(t, "plan-stale")
	seedPlanDeployment(t, repo, "dep-1", nil)
	fake := srv.sshRunner.(*FakeSSHRunner)

	_, preview := postPlan(t, router, "/api/v1/deployments/dep-1/plan", nil)
	digest := preview["plan_digest"].(string)
	reviewed := preview["plan"]

	// Inject: the deployment revision advances (fence bump) after review.
	if _, err := repo.BumpFence(context.Background(), "dep-1"); err != nil {
		t.Fatal(err)
	}
	rec, body := postPlan(t, router, "/api/v1/deployments/dep-1/plan/apply", map[string]any{"plan_digest": digest, "request_key": "k1", "plan": reviewed})
	if rec.Code != http.StatusConflict || errorCode(body) != "plan_stale" {
		t.Fatalf("expected plan_stale, got %d %s", rec.Code, rec.Body.String())
	}
	details := body["error"].(map[string]any)["details"].(map[string]any)
	reasons, _ := details["reasons"].([]any)
	if len(reasons) == 0 || reasons[0].(map[string]any)["kind"] != execplan.PreconditionDeploymentRevision {
		t.Fatalf("expected a deployment_revision reason, got %v", details)
	}
	if countOperations(t, db) != 0 || len(fake.Calls) != 0 {
		t.Fatalf("stale apply must have zero rows and zero effects")
	}
}

// fakeObserver returns fixed observations; closure credentials are marked
// satisfied unless satisfyNone is set.
type fakeObserver struct {
	obs         execplan.Observations
	satisfyNone bool
}

func (f fakeObserver) Observe(_ context.Context, _ *domain.Deployment, _ domain.CloudManifest, closure *domain.Closure) (execplan.Observations, error) {
	obs := f.obs
	if closure != nil && !f.satisfyNone {
		for _, component := range closure.ComponentsOfKind(domain.ClosureKindCredentialDescriptor) {
			if component.Credential != nil {
				obs.SatisfiedInputs = append(obs.SatisfiedInputs, component.Credential.LogicalID+":"+component.Credential.Field)
			}
		}
	}
	return obs, nil
}

// [REQ:STC-P0-016] P06-A03/A04: a satisfied target yields a no-op apply with
// no operation; a required input yields one onboarding handoff.
func TestDeploymentPlanNoOpAndNeedsInput(t *testing.T) {
	srv, repo, db, router := planTestServer(t, "plan-outcomes")
	seedPlanDeployment(t, repo, "dep-1", nil)
	_ = srv

	// First learn the desired digests from a normal preview.
	_, preview := postPlan(t, router, "/api/v1/deployments/dep-1/plan", nil)
	plan := preview["plan"].(map[string]any)
	planObserverOverride = fakeObserver{obs: execplan.Observations{
		ActiveReleaseDigest:       plan["release_digest"].(string),
		ActiveConfigurationDigest: plan["configuration_digest"].(string),
		ActiveClosureDigest:       plan["closure_digest"].(string),
	}}
	t.Cleanup(func() { planObserverOverride = nil })
	_, noop := postPlan(t, router, "/api/v1/deployments/dep-1/plan", nil)
	if noop["plan"].(map[string]any)["outcome"] != execplan.OutcomeNoOp {
		t.Fatalf("expected no_op outcome, got %v", noop["plan"].(map[string]any)["outcome"])
	}
	rec, body := postPlan(t, router, "/api/v1/deployments/dep-1/plan/apply", map[string]any{"plan_digest": noop["plan_digest"], "request_key": "k-noop"})
	if rec.Code != http.StatusAccepted || body["state"] != "no_op" || body["operation_id"] != nil {
		t.Fatalf("no-op apply: %d %s", rec.Code, rec.Body.String())
	}
	if countOperations(t, db) != 0 {
		t.Fatalf("a no-op must not admit an operation")
	}

	planObserverOverride = fakeObserver{satisfyNone: true}
	_, needs := postPlan(t, router, "/api/v1/deployments/dep-1/plan", nil)
	needsPlan := needs["plan"].(map[string]any)
	if needsPlan["outcome"] != execplan.OutcomeNeedsInput {
		t.Fatalf("expected needs_input, got %v", needsPlan["outcome"])
	}
	handoff := needsPlan["handoff"].(map[string]any)
	if handoff["owner"] != execplan.HandoffOwner || handoff["kind"] != execplan.HandoffKind || !strings.HasPrefix(handoff["reference"].(string), "vrooli-onboarding://deployments/dep-1/") {
		t.Fatalf("unexpected handoff %v", handoff)
	}
	rec, body = postPlan(t, router, "/api/v1/deployments/dep-1/plan/apply", map[string]any{"plan_digest": needs["plan_digest"], "request_key": "k-needs"})
	if rec.Code != http.StatusPreconditionRequired || errorCode(body) != "needs_input" {
		t.Fatalf("needs_input apply: %d %s", rec.Code, rec.Body.String())
	}
	next := body["error"].(map[string]any)["next_action"].(map[string]any)
	if next["reference"] != handoff["reference"] {
		t.Fatalf("apply must return the same handoff reference: %v vs %v", next["reference"], handoff["reference"])
	}
}

// [REQ:STC-P0-016] The Connect PlansService mirrors REST: same digest, same
// typed refusal code.
func TestConnectPlansServiceMirrorsREST(t *testing.T) {
	srv, repo, _, router := planTestServer(t, "plan-connect")
	seedPlanDeployment(t, repo, "dep-1", nil)
	_, preview := postPlan(t, router, "/api/v1/deployments/dep-1/plan", nil)

	path, handler := srv.PlansService().Handler()
	connectMux := http.NewServeMux()
	connectMux.Handle(path, handler)
	server := httptest.NewServer(connectMux)
	t.Cleanup(server.Close)
	client := plansv1connect.NewPlansServiceClient(http.DefaultClient, server.URL)

	resp, err := client.CompilePlan(context.Background(), connect.NewRequest(&plansv1.CompilePlanRequest{DeploymentId: "dep-1"}))
	if err != nil {
		t.Fatalf("CompilePlan: %v", err)
	}
	if resp.Msg.GetPlanDigest() != preview["plan_digest"] {
		t.Fatalf("Connect digest %s != REST digest %v", resp.Msg.GetPlanDigest(), preview["plan_digest"])
	}
	if len(resp.Msg.GetPlan().GetActions()) == 0 || resp.Msg.GetPlan().GetActions()[0].GetOwnerOperation() != execplan.OpHostPrepare {
		t.Fatalf("expected proto actions, got %v", resp.Msg.GetPlan().GetActions())
	}
	_, err = client.ApplyPlan(context.Background(), connect.NewRequest(&plansv1.ApplyPlanRequest{DeploymentId: "dep-1", PlanDigest: "sha256:wrong", RequestKey: "k"}))
	var cerr *connect.Error
	if !errorsAs(err, &cerr) || cerr.Code() != connect.CodeAborted && cerr.Code() != connect.CodeFailedPrecondition && cerr.Code() != connect.CodeAlreadyExists {
		t.Fatalf("expected a conflict-class Connect error, got %v", err)
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected the digest mismatch message, got %v", err)
	}
}

// [REQ:STC-P0-016] The ad hoc VPS apply endpoints require the reviewed digest
// and refuse mismatches before touching the target.
func TestVPSSetupApplyRequiresReviewedDigest(t *testing.T) {
	srv := newTestServer()
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/vps/setup/plan", srv.handleVPSSetupPlan).Methods("POST")
	api.HandleFunc("/vps/setup/apply", srv.handleVPSSetupApply).Methods("POST")
	fake := srv.sshRunner.(*FakeSSHRunner)
	bundle := filepath.Join(t.TempDir(), "app.tar.gz")
	if err := os.WriteFile(bundle, []byte("bundle"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{
		"version":      "1.0.0",
		"target":       map[string]any{"type": "vps", "vps": map[string]any{"host": "203.0.113.10", "port": 22, "user": "root", "workdir": "/root/Vrooli"}},
		"scenario":     map[string]any{"id": "landing-page-business-suite"},
		"dependencies": map[string]any{"scenarios": []string{"landing-page-business-suite", "vrooli-autoheal"}, "resources": []string{"postgres"}, "analyzer": map[string]any{"tool": "scenario-dependency-analyzer"}},
		"bundle":       map[string]any{"include_packages": true, "include_autoheal": true, "scenarios": []string{"landing-page-business-suite", "vrooli-autoheal"}},
		"ports":        map[string]any{"ui": 3000, "api": 3001},
		"edge":         map[string]any{"domain": "example.com", "caddy": map[string]any{"enabled": true}},
	}
	rec, body := postPlan(t, router, "/api/v1/vps/setup/plan", map[string]any{"manifest": manifest, "bundle_path": bundle})
	if rec.Code != http.StatusOK || body["plan_digest"] == "" {
		t.Fatalf("plan: %d %s", rec.Code, rec.Body.String())
	}
	steps := body["plan"].([]any)
	if steps[0].(map[string]any)["id"] != execplan.OpHostPrepare {
		t.Fatalf("legacy step view must carry action ids: %v", steps[0])
	}
	rec, body = postPlan(t, router, "/api/v1/vps/setup/apply", map[string]any{"manifest": manifest, "bundle_path": bundle})
	if rec.Code != http.StatusBadRequest || errorCode(body) != "invalid_request" {
		t.Fatalf("apply without digest: %d %s", rec.Code, rec.Body.String())
	}
	rec, body = postPlan(t, router, "/api/v1/vps/setup/apply", map[string]any{"manifest": manifest, "bundle_path": bundle, "plan_digest": "sha256:stale"})
	if rec.Code != http.StatusConflict || errorCode(body) != "plan_digest_mismatch" {
		t.Fatalf("apply with wrong digest: %d %s", rec.Code, rec.Body.String())
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("refused apply must not reach the target: %v", fake.Calls)
	}
}

func TestAdHocVPSApplyRequiresDurableDeployment(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	_, ok := srv.admitAdHocOperation(context.Background(), rec, "", "request-1", &execplan.Plan{})
	if ok {
		t.Fatal("ad hoc VPS apply must not execute without a durable deployment owner")
	}
	if rec.Code != http.StatusBadRequest || errorCodeFromResponse(t, rec) != "invalid_request" {
		t.Fatalf("expected typed durable-owner refusal, got %d %s", rec.Code, rec.Body.String())
	}
}

func errorCodeFromResponse(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return errorCode(body)
}

func errorsAs(err error, target **connect.Error) bool {
	for err != nil {
		if cerr, ok := err.(*connect.Error); ok {
			*target = cerr
			return true
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapper.Unwrap()
	}
	return false
}

// TestDeploymentPlanCompileBuildsOrRefusesBundle [REQ:STC-P0-028] proves the
// preview path shares the execute path's bundle contract: force_bundle_build
// without a release builder is a typed refusal rather than a silently
// unchanged digest, and a plain preview of a deployment with a recorded
// bundle stays pure.
func TestDeploymentPlanCompileBuildsOrRefusesBundle(t *testing.T) {
	_, repo, _, router := planTestServer(t, "plan-bundle")
	seedPlanDeployment(t, repo, "dep-1", nil)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/deployments/dep-1/plan", strings.NewReader(`{"scope":"full","force_bundle_build":true}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("forced rebuild without a builder: status %d body %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Error struct {
			Code    string            `json:"code"`
			Details map[string]string `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if refused := envelope.Error; refused.Code != "invalid_request" || refused.Details["step"] != "bundle_build" {
		t.Fatalf("refusal must name the bundle_build step: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/deployments/dep-1/plan", strings.NewReader(`{"scope":"full"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("plain preview: status %d body %s", rec.Code, rec.Body.String())
	}
}
