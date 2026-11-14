package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testCloudDeploymentID = "3c1c9a1e-5f3a-4a4d-9b1f-0d9d9f5b2f11"
	testCloudReleaseSHA   = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
)

// healthyObservationJSON is a complete, current, healthy observation in the
// exact wire shape scenario-to-cloud produces (proto names, enum names).
func healthyObservationJSON(deploymentID, releaseDigest string, observedAt time.Time) string {
	return fmt.Sprintf(`{"schema_version":"1","observation":{"deployment_id":%q,"target_id":"host:203.0.113.10","observed_release_digest":%q,"observed_configuration_digest":"sha256:cfg","observed_at":%q,"status":"HEALTH_STATUS_HEALTHY","checks":[{"id":"host_presence","status":"CHECK_STATUS_PASSED","reason_code":"","detail":""},{"id":"application_readiness","status":"CHECK_STATUS_PASSED","reason_code":"","detail":""}],"freshness":"FRESHNESS_CURRENT","producer_ref":"scenario-to-cloud:health:v1","partial":false,"missing_dependencies":[],"next_actions":[]}}`,
		deploymentID, releaseDigest, observedAt.UTC().Format(time.RFC3339Nano))
}

func mutateObservation(t *testing.T, body string, edit func(obs map[string]any)) string {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatal(err)
	}
	obs := envelope["observation"].(map[string]any)
	edit(obs)
	out, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func newObservationServer(t *testing.T, deploymentID string, status int, body string) *HTTPCloudHealthClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/deployments/"+deploymentID+"/health/observation" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &HTTPCloudHealthClient{httpClient: srv.Client(), baseURL: srv.URL}
}

// TestCheckDeploymentHealthFailsClosed [REQ:STC-P0-034] is the consumer gate
// matrix for P16-A01..A04: each HTTP 200 body below must prevent promotion
// with the named reason, and only the complete healthy+current+matching
// observation passes.
func TestCheckDeploymentHealthFailsClosed(t *testing.T) {
	now := time.Now().UTC()
	healthy := healthyObservationJSON(testCloudDeploymentID, "sha256:"+testCloudReleaseSHA, now)
	cases := []struct {
		name   string
		body   string
		expect string // ExpectedReleaseDigest
		reason string
	}{
		{name: "unhealthy content", body: mutateObservation(t, healthy, func(o map[string]any) { o["status"] = "HEALTH_STATUS_UNHEALTHY" }), reason: HealthReasonStatusNotHealthy},
		{name: "degraded content", body: mutateObservation(t, healthy, func(o map[string]any) { o["status"] = "HEALTH_STATUS_DEGRADED" }), reason: HealthReasonStatusNotHealthy},
		{name: "unknown status", body: mutateObservation(t, healthy, func(o map[string]any) { o["status"] = "HEALTH_STATUS_UNKNOWN" }), reason: HealthReasonStatusUnknown},
		{name: "unspecified status", body: mutateObservation(t, healthy, func(o map[string]any) { o["status"] = "HEALTH_STATUS_UNSPECIFIED" }), reason: HealthReasonStatusUnknown},
		{name: "status missing", body: mutateObservation(t, healthy, func(o map[string]any) { delete(o, "status") }), reason: HealthReasonStatusUnknown},
		{name: "unknown enum value", body: mutateObservation(t, healthy, func(o map[string]any) { o["status"] = "HEALTH_STATUS_SUPER_HEALTHY" }), reason: HealthReasonMalformedReport},
		{name: "legacy ok status", body: `{"status":"ok"}`, reason: HealthReasonUnsupportedSchema},
		{name: "malformed json", body: `{"schema_version":"1","observation":{`, reason: HealthReasonMalformedReport},
		{name: "empty body", body: ``, reason: HealthReasonMalformedReport},
		{name: "future schema", body: mutateObservationEnvelope(t, healthy, func(e map[string]any) { e["schema_version"] = "2" }), reason: HealthReasonUnsupportedSchema},
		{name: "stale by producer", body: mutateObservation(t, healthy, func(o map[string]any) { o["freshness"] = "FRESHNESS_STALE" }), reason: HealthReasonStale},
		{name: "freshness unknown", body: mutateObservation(t, healthy, func(o map[string]any) { o["freshness"] = "FRESHNESS_UNKNOWN" }), reason: HealthReasonFreshnessUnknown},
		{name: "freshness missing", body: mutateObservation(t, healthy, func(o map[string]any) { delete(o, "freshness") }), reason: HealthReasonFreshnessUnknown},
		{name: "stale by time", body: healthyObservationJSON(testCloudDeploymentID, "sha256:"+testCloudReleaseSHA, now.Add(-DefaultCloudHealthMaxAge-time.Minute)), reason: HealthReasonStale},
		{name: "observed_at missing", body: mutateObservation(t, healthy, func(o map[string]any) { delete(o, "observed_at") }), reason: HealthReasonObservedAtMissing},
		{name: "wrong target deployment", body: healthyObservationJSON("00000000-0000-4000-8000-000000000000", "sha256:"+testCloudReleaseSHA, now), reason: HealthReasonDeploymentMismatch},
		{name: "scenario name as identity", body: healthyObservationJSON("landing-page-business-suite", "sha256:"+testCloudReleaseSHA, now), reason: HealthReasonDeploymentMismatch},
		{name: "identity missing", body: mutateObservation(t, healthy, func(o map[string]any) { delete(o, "deployment_id") }), reason: HealthReasonMalformedReport},
		{name: "wrong release", body: healthyObservationJSON(testCloudDeploymentID, "sha256:"+strings.Repeat("0", 64), now), expect: testCloudReleaseSHA, reason: HealthReasonReleaseMismatch},
		{name: "release absent when expected", body: mutateObservation(t, healthy, func(o map[string]any) { o["observed_release_digest"] = "" }), expect: testCloudReleaseSHA, reason: HealthReasonReleaseMismatch},
		{name: "partial observation", body: mutateObservation(t, healthy, func(o map[string]any) { o["partial"] = true; o["missing_dependencies"] = []string{"edge_tls"} }), reason: HealthReasonPartialObservation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newObservationServer(t, testCloudDeploymentID, http.StatusOK, tc.body)
			result, err := client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{DeploymentID: testCloudDeploymentID, ExpectedReleaseDigest: tc.expect})
			if err != nil {
				t.Fatalf("CheckDeploymentHealth() error = %v", err)
			}
			if result.Healthy {
				t.Fatalf("HTTP 200 body was accepted as healthy: %s", tc.body)
			}
			if result.ReasonCode != tc.reason {
				t.Fatalf("reason = %q (%s), want %q", result.ReasonCode, result.Details, tc.reason)
			}
			if result.Explain() == "" {
				t.Fatal("verdict has no explanation for the step message")
			}
		})
	}

	t.Run("healthy current matching passes", func(t *testing.T) {
		client := newObservationServer(t, testCloudDeploymentID, http.StatusOK, healthy)
		for _, expected := range []string{"", testCloudReleaseSHA, "sha256:" + testCloudReleaseSHA, strings.ToUpper(testCloudReleaseSHA)} {
			result, err := client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{DeploymentID: testCloudDeploymentID, ExpectedReleaseDigest: expected})
			if err != nil {
				t.Fatalf("CheckDeploymentHealth() error = %v", err)
			}
			if !result.Healthy {
				t.Fatalf("healthy observation refused with expected %q: %s", expected, result.Explain())
			}
			if result.DeploymentID != testCloudDeploymentID || result.ObservedReleaseDigest != "sha256:"+testCloudReleaseSHA || result.ObservedAt.IsZero() {
				t.Fatalf("result = %+v", result)
			}
		}
	})

	t.Run("tighter caller max age", func(t *testing.T) {
		client := newObservationServer(t, testCloudDeploymentID, http.StatusOK, healthyObservationJSON(testCloudDeploymentID, "sha256:"+testCloudReleaseSHA, now.Add(-30*time.Second)))
		result, err := client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{DeploymentID: testCloudDeploymentID, MaxAge: 10 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		if result.Healthy || result.ReasonCode != HealthReasonStale {
			t.Fatalf("result = %+v", result)
		}
	})
}

func mutateObservationEnvelope(t *testing.T, body string, edit func(envelope map[string]any)) string {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatal(err)
	}
	edit(envelope)
	out, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// TestCheckDeploymentHealthSurfacesCloudRefusalsAndTransportFailures
// [REQ:STC-P0-034] proves non-200 responses become explainable verdicts with
// the cloud service's stable code and next action, never Go errors that lose
// the reason.
func TestCheckDeploymentHealthSurfacesCloudRefusalsAndTransportFailures(t *testing.T) {
	client := newObservationServer(t, testCloudDeploymentID, http.StatusServiceUnavailable, `{"error":{"code":"health_unknown","message":"Target unreachable","retryable":true,"next_action":{"owner":"scenario-to-cloud","kind":"command","reference":"scenario-to-cloud ssh test 203.0.113.10","label":"Test SSH reach"}}}`)
	result, err := client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{DeploymentID: testCloudDeploymentID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Healthy || result.ReasonCode != "health_unknown" || len(result.NextActions) != 1 || !strings.Contains(result.Explain(), "scenario-to-cloud ssh test") {
		t.Fatalf("result = %+v", result)
	}

	untyped := newObservationServer(t, testCloudDeploymentID, http.StatusBadGateway, `upstream down`)
	result, err = untyped.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{DeploymentID: testCloudDeploymentID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Healthy || result.ReasonCode != HealthReasonTransportError || !strings.Contains(result.Details, "502") {
		t.Fatalf("result = %+v", result)
	}

	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()
	dead := &HTTPCloudHealthClient{httpClient: &http.Client{Timeout: time.Second}, baseURL: closed.URL}
	result, err = dead.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{DeploymentID: testCloudDeploymentID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Healthy || result.ReasonCode != HealthReasonTransportError {
		t.Fatalf("result = %+v", result)
	}
}

// TestCheckDeploymentHealthResolvesExactDeployment [REQ:STC-P0-034] proves
// P16-A05: with a profile-derived selector the exact deployment id (not the
// scenario name) is resolved through the cloud resolve endpoint and the
// observation is checked against that id.
func TestCheckDeploymentHealthResolvesExactDeployment(t *testing.T) {
	now := time.Now().UTC()
	var resolveQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/deployments/resolve":
			resolveQuery = r.URL.RawQuery
			if r.URL.Query().Get("scenario") == "ghost" {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":{"code":"deployment_not_found","message":"Deployment not found","next_action":{"owner":"scenario-to-cloud","kind":"endpoint","reference":"/api/v1/deployments","label":"List deployments"}}}`))
				return
			}
			if r.URL.Query().Get("scenario") == "twins" {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":{"code":"deployment_selector_ambiguous","message":"More than one deployment matches"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"schema_version":"1","ref":{"id":"` + testCloudDeploymentID + `","scenario_id":"landing-page-business-suite","environment":"production","target":{"transport":"ssh","locator":{"host":"203.0.113.10"}}}}`))
		case "/api/v1/deployments/" + testCloudDeploymentID + "/health/observation":
			_, _ = w.Write([]byte(healthyObservationJSON(testCloudDeploymentID, "sha256:"+testCloudReleaseSHA, now)))
		case "/api/v1/deployments/landing-page-business-suite/health":
			t.Errorf("consumer used the hardcoded scenario-slug health path")
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := &HTTPCloudHealthClient{httpClient: srv.Client(), baseURL: srv.URL}

	result, err := client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{
		Selector:              DeploymentSelector{ScenarioID: "landing-page-business-suite", Environment: "production", Domain: "vrooli.com"},
		ExpectedReleaseDigest: testCloudReleaseSHA,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Healthy || result.DeploymentID != testCloudDeploymentID {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(resolveQuery, "scenario=landing-page-business-suite") || !strings.Contains(resolveQuery, "domain=vrooli.com") || strings.Contains(resolveQuery, "environment=") {
		t.Fatalf("resolve query = %q, want scenario+domain (one facet beside scenario)", resolveQuery)
	}

	// Environment given without a domain resolves by environment.
	if _, err := client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{Selector: DeploymentSelector{ScenarioID: "landing-page-business-suite", Environment: "production"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resolveQuery, "environment=production") {
		t.Fatalf("resolve query = %q, want environment facet", resolveQuery)
	}

	// The resolved record's environment must match the caller's expectation.
	result, err = client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{Selector: DeploymentSelector{ScenarioID: "landing-page-business-suite", Environment: "staging", Domain: "vrooli.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Healthy || result.ReasonCode != HealthReasonEnvironmentMismatch {
		t.Fatalf("environment mismatch result = %+v", result)
	}

	// Typed refusals from resolve pass through with their code and next action.
	result, err = client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{Selector: DeploymentSelector{ScenarioID: "ghost", Domain: "ghost.example"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Healthy || result.ReasonCode != "deployment_not_found" || len(result.NextActions) != 1 {
		t.Fatalf("not-found result = %+v", result)
	}
	result, err = client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{Selector: DeploymentSelector{ScenarioID: "twins", Domain: "twins.example"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Healthy || result.ReasonCode != "deployment_selector_ambiguous" {
		t.Fatalf("ambiguous result = %+v", result)
	}

	// A request with neither id nor a usable selector is a caller error.
	if _, err := client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{}); err == nil {
		t.Fatal("empty request was accepted")
	}
	if _, err := client.CheckDeploymentHealth(context.Background(), DeploymentHealthRequest{Selector: DeploymentSelector{ScenarioID: "solo"}}); err == nil {
		t.Fatal("scenario-only selector was accepted")
	}
}

func TestCloudManifestSelector(t *testing.T) {
	sel := cloudManifestSelector(json.RawMessage(`{"environment":"staging","scenario":{"id":"demo"},"edge":{"domain":"demo.example"},"target":{"type":"vps","vps":{"host":"203.0.113.10"}}}`))
	if sel != (DeploymentSelector{ScenarioID: "demo", Environment: "staging", Domain: "demo.example", Host: "203.0.113.10"}) {
		t.Fatalf("selector = %+v", sel)
	}
	if !cloudManifestSelector(nil).IsZero() || !cloudManifestSelector(json.RawMessage(`{`)).IsZero() {
		t.Fatal("empty or malformed manifests must yield a zero selector")
	}
}
