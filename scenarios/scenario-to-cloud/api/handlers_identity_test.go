package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/persistence"

	deploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"

	"connectrpc.com/connect"
	_ "modernc.org/sqlite"
)

// newIdentityTestServer wires the real repository over a disposable SQLite
// pool so the REST and Connect surfaces are exercised end to end.
func newIdentityTestServer(t *testing.T, name string) (*Server, *persistence.Repository) {
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
	srv.repo = repo
	srv.deploymentRepo = repo
	srv.historyRecorder = repo
	srv.setupRoutes()
	return srv, repo
}

func seedDeployment(t *testing.T, repo *persistence.Repository, id, scenario, environment, host, domainName string) {
	t.Helper()
	m := domain.CloudManifest{
		Version:  "1",
		Target:   domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: host, Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		Scenario: domain.ManifestScenario{ID: scenario},
		Edge:     domain.ManifestEdge{Domain: domainName},
	}
	raw, _ := json.Marshal(m)
	now := time.Now().UTC()
	if err := repo.CreateDeployment(context.Background(), &domain.Deployment{
		ID: id, Name: scenario + " @ " + domainName, ScenarioID: scenario, Environment: environment,
		Status: domain.StatusDeployed, Manifest: raw, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed %s: %v", id, err)
	}
}

// TestResolveEndpointReturnsTypedRefOrConflict [REQ:STC-P0-015] proves
// P03-A02 and P03-A03 on the REST surface: distinct environments resolve to
// distinct refs, an ambiguous selector is a 409 with the stable code, and an
// invalid selector is a 400.
func TestResolveEndpointReturnsTypedRefOrConflict(t *testing.T) {
	srv, repo := newIdentityTestServer(t, "handlers-resolve")
	seedDeployment(t, repo, "dep-prod", "demo-app", "production", "203.0.113.10", "demo.example")
	seedDeployment(t, repo, "dep-staging", "demo-app", "staging", "203.0.113.10", "demo.example")

	get := func(query string) (*httptest.ResponseRecorder, map[string]any) {
		t.Helper()
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/resolve?"+query, nil))
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v body=%s", query, err, rec.Body.String())
		}
		return rec, body
	}

	rec, body := get("scenario=demo-app&environment=staging")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	ref := body["ref"].(map[string]any)
	if ref["id"] != "dep-staging" || ref["environment"] != "staging" || body["schema_version"] != "1" {
		t.Fatalf("ref = %v", body)
	}
	target := ref["target"].(map[string]any)
	if target["transport"] != "ssh" || target["locator"].(map[string]any)["host"] != "203.0.113.10" {
		t.Fatalf("target = %v", target)
	}

	rec, _ = get("scenario=demo-app&host=203.0.113.10")
	if rec.Code != http.StatusConflict {
		t.Fatalf("ambiguous status = %d body=%s", rec.Code, rec.Body.String())
	}
	typed := apierrors.FromHTTP(rec.Code, rec.Body.Bytes())
	if typed.Code != apierrors.CodeDeploymentSelectorAmbiguous || len(typed.Details["candidates"].([]any)) != 2 {
		t.Fatalf("ambiguous error = %+v", typed)
	}

	rec, _ = get("scenario=demo-app")
	if rec.Code != http.StatusBadRequest || apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code != apierrors.CodeDeploymentSelectorInvalid {
		t.Fatalf("scenario-only selector: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec, _ = get("scenario=demo-app&environment=qa")
	if rec.Code != http.StatusNotFound || apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code != apierrors.CodeDeploymentNotFound {
		t.Fatalf("no match: status=%d body=%s", rec.Code, rec.Body.String())
	}

	// "resolve" must never be read as a deployment id by the {id} route.
	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/resolve", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bare resolve route status = %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestGetDeploymentNotFoundUsesTypedError [REQ:STC-P0-017] proves the
// migrated GET path writes the typed envelope.
func TestGetDeploymentNotFoundUsesTypedError(t *testing.T) {
	srv, _ := newIdentityTestServer(t, "handlers-get-typed")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/does-not-exist", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	typed := apierrors.FromHTTP(rec.Code, rec.Body.Bytes())
	if typed.Code != apierrors.CodeDeploymentNotFound || typed.NextAction == nil || typed.NextAction.Reference != "/api/v1/deployments" {
		t.Fatalf("typed error = %+v", typed)
	}
	var raw map[string]map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &raw)
	if _, legacy := raw["error"]["hint"]; legacy {
		t.Fatalf("legacy hint field must not appear: %s", rec.Body.String())
	}
}

// TestCreateDeploymentKeepsIdentityAcrossManifestResubmit [REQ:STC-P0-015]
// proves P03-A01 through the create path: resubmitting a manifest for the
// same scenario, environment and host updates the record in place with the
// same id, and a different environment creates a distinct record.
func TestCreateDeploymentKeepsIdentityAcrossManifestResubmit(t *testing.T) {
	srv, _ := newIdentityTestServer(t, "handlers-create-identity")
	manifest := func(environment string) map[string]any {
		return map[string]any{
			"version":     "1",
			"environment": environment,
			"target":      map[string]any{"type": "vps", "vps": map[string]any{"host": "203.0.113.10", "port": 22, "user": "root", "workdir": "/root/Vrooli"}},
			"scenario":    map[string]any{"id": "demo-app"},
			"dependencies": map[string]any{
				"scenarios": []string{"demo-app"},
				"resources": []string{},
				"analyzer":  map[string]any{"tool": "scenario-dependency-analyzer"},
			},
			"bundle": map[string]any{"include_packages": true, "include_autoheal": true, "scenarios": []string{"demo-app", "vrooli-autoheal"}},
			"ports":  map[string]any{"api": 15000, "ui": 3000},
			"edge":   map[string]any{"domain": "demo.example", "caddy": map[string]any{"enabled": true}},
		}
	}
	post := func(environment string) (int, map[string]any) {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{"manifest": manifest(environment)})
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/deployments", bytes.NewReader(payload)))
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v body=%s", err, rec.Body.String())
		}
		return rec.Code, body
	}
	status, first := post("production")
	if status != http.StatusCreated {
		t.Fatalf("first create status = %d body=%v", status, first)
	}
	firstID := first["deployment"].(map[string]any)["id"].(string)
	status, second := post("production")
	if status != http.StatusOK || second["updated"] != true {
		t.Fatalf("resubmit status = %d body=%v", status, second)
	}
	if second["deployment"].(map[string]any)["id"] != firstID {
		t.Fatalf("resubmit changed the deployment id: %v", second)
	}
	status, third := post("staging")
	if status != http.StatusCreated || third["deployment"].(map[string]any)["id"] == firstID {
		t.Fatalf("staging install must be a distinct record: %d %v", status, third)
	}
	if third["deployment"].(map[string]any)["environment"] != "staging" {
		t.Fatalf("environment not persisted: %v", third["deployment"])
	}
}

// TestConnectDeploymentsServiceMountedBesideREST [REQ:STC-P0-017] proves
// P03-A06: the Connect client and the REST client observe the same stable
// code for the same ambiguous selector.
func TestConnectDeploymentsServiceMountedBesideREST(t *testing.T) {
	srv, repo := newIdentityTestServer(t, "handlers-connect")
	seedDeployment(t, repo, "dep-prod", "demo-app", "production", "203.0.113.10", "demo.example")
	seedDeployment(t, repo, "dep-staging", "demo-app", "staging", "203.0.113.10", "demo.example")
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := deploymentsv1connect.NewDeploymentsServiceClient(http.DefaultClient, ts.URL)
	ctx := context.Background()
	resolved, err := client.ResolveDeployment(ctx, connect.NewRequest(&deploymentsv1.ResolveDeploymentRequest{Selector: &deploymentsv1.DeploymentSelector{ScenarioId: "demo-app", Environment: "production"}}))
	if err != nil || resolved.Msg.GetRef().GetId() != "dep-prod" {
		t.Fatalf("connect resolve = %v, %v", resolved, err)
	}

	_, err = client.ResolveDeployment(ctx, connect.NewRequest(&deploymentsv1.ResolveDeploymentRequest{Selector: &deploymentsv1.DeploymentSelector{ScenarioId: "demo-app", Host: "203.0.113.10"}}))
	connectCode := connectDetailCode(t, err)

	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/resolve?scenario=demo-app&host=203.0.113.10", nil))
	restCode := apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code
	if connectCode != restCode || restCode != apierrors.CodeDeploymentSelectorAmbiguous {
		t.Fatalf("clients disagree: connect=%q rest=%q", connectCode, restCode)
	}

	listed, err := client.ListDeployments(ctx, connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{Environment: "staging"}))
	if err != nil || len(listed.Msg.GetDeployments()) != 1 || listed.Msg.GetDeployments()[0].GetRef().GetId() != "dep-staging" {
		t.Fatalf("connect list = %v, %v", listed, err)
	}
	got, err := client.GetDeployment(ctx, connect.NewRequest(&deploymentsv1.GetDeploymentRequest{Id: "dep-prod"}))
	if err != nil || got.Msg.GetDeployment().GetRef().GetTarget().GetTransport() != identity.TransportSSH {
		t.Fatalf("connect get = %v, %v", got, err)
	}
}

func connectDetailCode(t *testing.T, err error) string {
	t.Helper()
	cerr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect error, got %T %v", err, err)
	}
	for _, detail := range cerr.Details() {
		value, err := detail.Value()
		if err != nil {
			continue
		}
		if e, ok := value.(interface{ GetCode() string }); ok {
			return e.GetCode()
		}
	}
	t.Fatalf("no typed detail on connect error %v", cerr)
	return ""
}
