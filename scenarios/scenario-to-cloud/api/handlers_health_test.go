package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/health"
	"scenario-to-cloud/persistence"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health/healthv1connect"
)

const healthTestBundleSHA = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

// newHealthTestServer wires the real repository over SQLite, an SSH runner
// that refuses every command, and no DNS/TLS services, so the producer path
// is exercised end to end without network access.
func newHealthTestServer(t *testing.T, name string) (*Server, *persistence.Repository) {
	t.Helper()
	srv, repo := newIdentityTestServer(t, name)
	srv.sshRunner = &FakeSSHRunner{DefaultErr: errors.New("dial tcp 203.0.113.10:22: connection refused")}
	srv.dnsService = nil
	srv.tlsService = nil
	previous := evaluateFreshnessForHealth
	evaluateFreshnessForHealth = func(*Server, context.Context, *domain.Deployment, domain.CloudManifest) *domain.FreshnessStatus {
		return &domain.FreshnessStatus{Status: domain.FreshnessUnknown, Summary: "not evaluated in tests"}
	}
	t.Cleanup(func() { evaluateFreshnessForHealth = previous })
	return srv, repo
}

func seedHealthDeployment(t *testing.T, repo *persistence.Repository, id, scenario string) {
	t.Helper()
	m := domain.CloudManifest{
		Version:  "1",
		Target:   domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		Scenario: domain.ManifestScenario{ID: scenario},
		Edge:     domain.ManifestEdge{Domain: "demo.example"},
	}
	raw, _ := json.Marshal(m)
	now := time.Now().UTC()
	sha := healthTestBundleSHA
	if err := repo.CreateDeployment(context.Background(), &domain.Deployment{
		ID: id, Name: scenario + " @ demo.example", ScenarioID: scenario, Environment: "production",
		Status: domain.StatusDeployed, Manifest: raw, BundleSHA256: &sha, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed %s: %v", id, err)
	}
}

// TestHealthObservationEndpointFailsClosedOnUnreachableTarget
// [REQ:STC-P0-033] proves the REST observation route: HTTP 200 carries an
// observation whose status and freshness are UNKNOWN when the target cannot
// be inspected, bound to the deployment id (which differs from the scenario
// name, P16-A05) and the recorded release digest.
func TestHealthObservationEndpointFailsClosedOnUnreachableTarget(t *testing.T) {
	srv, repo := newHealthTestServer(t, "handlers-health-observation")
	const depID = "3c1c9a1e-5f3a-4a4d-9b1f-0d9d9f5b2f11"
	seedHealthDeployment(t, repo, depID, "landing-page-business-suite")

	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/"+depID+"/health/observation", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp healthv1.GetHealthObservationResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode observation: %v body=%s", err, rec.Body.String())
	}
	if resp.GetSchemaVersion() != health.SchemaVersion {
		t.Fatalf("schema_version = %q", resp.GetSchemaVersion())
	}
	obs := resp.GetObservation()
	if obs.GetDeploymentId() != depID {
		t.Fatalf("deployment_id = %q, want %q", obs.GetDeploymentId(), depID)
	}
	if obs.GetObservedReleaseDigest() != "sha256:"+healthTestBundleSHA {
		t.Fatalf("observed_release_digest = %q", obs.GetObservedReleaseDigest())
	}
	if obs.GetTargetId() != "host:203.0.113.10" {
		t.Fatalf("target_id = %q", obs.GetTargetId())
	}
	if obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN {
		t.Fatalf("status = %s, want UNKNOWN (HTTP 200 must not imply health)", obs.GetStatus())
	}
	if obs.GetFreshness() != healthv1.Freshness_FRESHNESS_UNKNOWN {
		t.Fatalf("freshness = %s, want UNKNOWN", obs.GetFreshness())
	}
	if !obs.GetPartial() {
		t.Fatal("observation with no live state, DNS or TLS evidence was not partial")
	}
	if obs.GetObservedAt() == nil || obs.GetObservedAt().AsTime().IsZero() {
		t.Fatal("observed_at missing")
	}

	// Unknown deployment id is a typed 404, not an observation.
	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/missing/health/observation", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing deployment status = %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestLegacyHealthReportEmbedsObservation [REQ:STC-P0-033] proves the old
// /health report now carries the same typed truth so existing consumers
// cannot read a different verdict.
func TestLegacyHealthReportEmbedsObservation(t *testing.T) {
	srv, repo := newHealthTestServer(t, "handlers-health-legacy")
	const depID = "6b7d1f0e-2a3b-4c5d-8e9f-0a1b2c3d4e5f"
	seedHealthDeployment(t, repo, depID, "demo-app")

	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/"+depID+"/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var report domain.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Health != domain.HealthUnknown {
		t.Fatalf("legacy health = %s, want unknown", report.Health)
	}
	if len(report.Observation) == 0 {
		t.Fatal("legacy report has no embedded observation")
	}
	var obs healthv1.HealthObservation
	if err := protojson.Unmarshal(report.Observation, &obs); err != nil {
		t.Fatalf("decode embedded observation: %v", err)
	}
	if obs.GetDeploymentId() != depID || obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN {
		t.Fatalf("embedded observation = %s/%s", obs.GetDeploymentId(), obs.GetStatus())
	}
}

// TestConnectHealthServiceMatchesREST [REQ:STC-P0-033] proves the Connect
// surface serves the same observation and the same typed not-found.
func TestConnectHealthServiceMatchesREST(t *testing.T) {
	srv, repo := newHealthTestServer(t, "handlers-health-connect")
	const depID = "0f0e0d0c-0b0a-4908-8706-050403020100"
	seedHealthDeployment(t, repo, depID, "demo-app")
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := healthv1connect.NewHealthServiceClient(ts.Client(), ts.URL)
	resp, err := client.GetHealthObservation(context.Background(), connect.NewRequest(&healthv1.GetHealthObservationRequest{DeploymentId: depID}))
	if err != nil {
		t.Fatalf("GetHealthObservation: %v", err)
	}
	obs := resp.Msg.GetObservation()
	if obs.GetDeploymentId() != depID || obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN || obs.GetFreshness() != healthv1.Freshness_FRESHNESS_UNKNOWN {
		t.Fatalf("connect observation = %s/%s/%s", obs.GetDeploymentId(), obs.GetStatus(), obs.GetFreshness())
	}

	_, err = client.GetHealthObservation(context.Background(), connect.NewRequest(&healthv1.GetHealthObservationRequest{DeploymentId: "missing"}))
	var cerr *connect.Error
	if !errors.As(err, &cerr) || cerr.Code() != connect.CodeNotFound {
		t.Fatalf("missing deployment error = %v", err)
	}
	_, err = client.GetHealthObservation(context.Background(), connect.NewRequest(&healthv1.GetHealthObservationRequest{}))
	if !errors.As(err, &cerr) || cerr.Code() != connect.CodeInvalidArgument {
		t.Fatalf("empty id error = %v", err)
	}
}
