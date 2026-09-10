package deployments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewHTTPCloudHealthClientUsesConfiguredURL(t *testing.T) {
	previous := os.Getenv("SCENARIO_TO_CLOUD_URL")
	t.Cleanup(func() { _ = os.Setenv("SCENARIO_TO_CLOUD_URL", previous) })
	if err := os.Setenv("SCENARIO_TO_CLOUD_URL", "http://cloud.example/"); err != nil {
		t.Fatal(err)
	}
	client, err := NewHTTPCloudHealthClient(nil)
	if err != nil {
		t.Fatalf("NewHTTPCloudHealthClient() error = %v", err)
	}
	if client.baseURL != "http://cloud.example" {
		t.Fatalf("baseURL = %q", client.baseURL)
	}
}

func TestHTTPCloudDeploymentClientRequiresDurableReceipt(t *testing.T) {
	var statusCalls, observationCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/deployments":
			if r.Method != http.MethodPost {
				t.Fatalf("create method = %s", r.Method)
			}
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create request: %v", err)
			}
			if string(body["manifest"]) != `{"scenario":{"id":"demo"}}` {
				t.Fatalf("manifest = %s", body["manifest"])
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"deployment":{"id":"dep-1"}}`))
		case "/api/v1/deployments/dep-1/execute":
			if r.Method != http.MethodPost {
				t.Fatalf("execute method = %s", r.Method)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"run_id":"run-1"}`))
		case "/api/v1/deployments/dep-1":
			call := statusCalls.Add(1)
			status := "deploying"
			if call > 1 {
				status = "deployed"
			}
			_, _ = w.Write([]byte(`{"deployment":{"status":"` + status + `"}}`))
		case "/api/v1/deployments/dep-1/receipt":
			_, _ = w.Write([]byte(`{"receipt":{"schema_version":1,"deployment_id":"dep-1","scenario_id":"demo","target_kind":"vps","destination_id":"sha256:target","destination_host":"vps.example","destination_workdir":"/srv/demo","destination_domain":"demo.example","bundle_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","outcome":"deployed","health":"healthy","external_receipt":"scenario-to-cloud:dep-1","observed_at":"` + time.Now().UTC().Format(time.RFC3339) + `","producer_ref":"scenario-to-cloud","target_key":"host:vps.example","release_digest":"` + strings.Repeat("a", 64) + `"}}`))
		case "/api/v1/deployments/dep-1/health/observation":
			observationCalls.Add(1)
			_, _ = w.Write([]byte(healthyObservationJSON("dep-1", "sha256:"+strings.Repeat("a", 64), time.Now().UTC())))
		case "/api/v1/deployments/dep-1/evidence":
			if r.URL.Query().Get("release_digest") != strings.Repeat("a", 64) {
				t.Fatalf("evidence release_digest = %q", r.URL.Query().Get("release_digest"))
			}
			_, _ = w.Write([]byte(`{"schema_version":"1","evidence":{"schema_version":"1","profile_id":"cloud-launch-v1","release_digest":"` + strings.Repeat("a", 64) + `","target_key":"host:vps.example","cells":[{"case_id":"GOV-05","lane":"api","disposition":"passed","required":true,"record_id":"rec-1","receipt_refs":["cloud-target:op-1:GOV-05"]}],"required_cells":1,"passed":true,"producer_ref":"scenario-to-cloud"}}`))
		case "/api/v1/deployments/dep-1/publication":
			_, _ = w.Write([]byte(`{"schema_version":"1","publication":{"id":"pub-1","request_key":"req-1","review_ref":"rr-1","release_digest":"` + strings.Repeat("a", 64) + `","state":"published","activated_release_digest":"` + strings.Repeat("a", 64) + `","predecessor_release_digest":"` + strings.Repeat("9", 64) + `","target_key":"host:vps.example"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := &HTTPCloudHealthClient{httpClient: srv.Client(), baseURL: srv.URL, pollInterval: time.Millisecond}
	receipt, err := client.DeployCloud(context.Background(), &CloudDeploymentRequest{Manifest: json.RawMessage(`{"scenario":{"id":"demo"}}`), RunPreflight: true})
	if err != nil {
		t.Fatalf("DeployCloud() error = %v", err)
	}
	if receipt.DeploymentID != "dep-1" || receipt.DestinationID != "sha256:target" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if receipt.Evidence == nil || !receipt.Evidence.Passed || receipt.Publication == nil || receipt.Publication.PredecessorReleaseDigest != strings.Repeat("9", 64) {
		t.Fatalf("receipt evidence/publication = %#v / %#v", receipt.Evidence, receipt.Publication)
	}
	if statusCalls.Load() < 2 {
		t.Fatalf("status calls = %d, expected wait for deployed state", statusCalls.Load())
	}
	if observationCalls.Load() != 1 {
		t.Fatalf("observation calls = %d, the receipt must be backed by the typed observation", observationCalls.Load())
	}
}

// TestHTTPCloudDeploymentClientRejectsReceiptWithoutHealthyObservation
// [REQ:STC-P0-034] proves a receipt that says healthy is not accepted when
// the typed observation disagrees.
func TestHTTPCloudDeploymentClientRejectsReceiptWithoutHealthyObservation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/deployments":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"deployment":{"id":"dep-3"}}`))
		case "/api/v1/deployments/dep-3/execute":
			w.WriteHeader(http.StatusAccepted)
		case "/api/v1/deployments/dep-3":
			_, _ = w.Write([]byte(`{"deployment":{"status":"deployed"}}`))
		case "/api/v1/deployments/dep-3/receipt":
			_, _ = w.Write([]byte(`{"receipt":{"schema_version":1,"deployment_id":"dep-3","scenario_id":"demo","target_kind":"vps","destination_id":"sha256:target","destination_host":"vps.example","destination_workdir":"/srv/demo","destination_domain":"demo.example","bundle_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","outcome":"deployed","health":"healthy","external_receipt":"scenario-to-cloud:dep-3","observed_at":"` + time.Now().UTC().Format(time.RFC3339) + `","producer_ref":"scenario-to-cloud","target_key":"host:vps.example","release_digest":"` + strings.Repeat("a", 64) + `"}}`))
		case "/api/v1/deployments/dep-3/health/observation":
			body := strings.Replace(healthyObservationJSON("dep-3", "sha256:"+strings.Repeat("a", 64), time.Now().UTC()), "HEALTH_STATUS_HEALTHY", "HEALTH_STATUS_UNHEALTHY", 1)
			_, _ = w.Write([]byte(body))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := &HTTPCloudHealthClient{httpClient: srv.Client(), baseURL: srv.URL, pollInterval: time.Millisecond}
	_, err := client.DeployCloud(context.Background(), &CloudDeploymentRequest{Manifest: json.RawMessage(`{"scenario":{"id":"demo"}}`)})
	if err == nil || !strings.Contains(err.Error(), "status_not_healthy") {
		t.Fatalf("DeployCloud() error = %v, want the observation verdict", err)
	}
}

func TestHTTPCloudDeploymentClientRejectsIncompleteReceipt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/deployments":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"deployment":{"id":"dep-2"}}`))
		case "/api/v1/deployments/dep-2/execute":
			w.WriteHeader(http.StatusAccepted)
		case "/api/v1/deployments/dep-2":
			_, _ = w.Write([]byte(`{"deployment":{"status":"deployed"}}`))
		case "/api/v1/deployments/dep-2/receipt":
			_, _ = w.Write([]byte(`{"receipt":{"schema_version":1,"deployment_id":"dep-2","outcome":"deployed"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := &HTTPCloudHealthClient{httpClient: srv.Client(), baseURL: srv.URL, pollInterval: time.Millisecond}
	if _, err := client.DeployCloud(context.Background(), &CloudDeploymentRequest{Manifest: json.RawMessage(`{"scenario":{"id":"demo"}}`)}); err == nil {
		t.Fatal("incomplete receipt was accepted")
	}
}

func TestHTTPCloudRecoveryClientRequiresOwnerReceipt(t *testing.T) {
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/deployments/dep-1/recovery" || r.Method != http.MethodPost {
			t.Fatalf("recovery request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode recovery request: %v", err)
		}
		_, _ = w.Write([]byte(`{"receipt":{"schema_version":1,"deployment_id":"dep-1","action":"halt","outcome":"halted","health":"stopped","external_receipt":"scenario-to-cloud:recovery:dep-1","observed_at":"2026-09-08T21:00:00Z"}}`))
	}))
	defer srv.Close()
	client := &HTTPCloudHealthClient{httpClient: srv.Client(), baseURL: srv.URL}
	receipt, err := client.RecoverCloud(context.Background(), &CloudRecoveryRequest{DeploymentID: "dep-1", Action: "halt", ExpectedBundleSHA: "sha256:bundle", Confirmation: "halt dep-1"})
	if err != nil {
		t.Fatalf("RecoverCloud() error = %v", err)
	}
	if receipt.Outcome != "halted" || received["confirmation"] != "halt dep-1" || received["expected_bundle_sha256"] != "sha256:bundle" {
		t.Fatalf("receipt/request = %#v / %#v", receipt, received)
	}
}

func TestHTTPCloudRecoveryClientPollsDurableRepairOperation(t *testing.T) {
	var statusCalls atomic.Int32
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/deployments/dep-1/recovery":
			if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
				t.Fatalf("decode recovery request: %v", err)
			}
			_, _ = w.Write([]byte(`{"receipt":{"schema_version":1,"deployment_id":"dep-1","operation_id":"op-1","action":"rollback","outcome":"pending","health":"unknown"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/deployments/dep-1/recovery/op-1":
			call := statusCalls.Add(1)
			if call == 1 {
				_, _ = w.Write([]byte(`{"receipt":{"schema_version":1,"deployment_id":"dep-1","operation_id":"op-1","action":"rollback","outcome":"running","health":"unknown"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"receipt":{"schema_version":1,"deployment_id":"dep-1","operation_id":"op-1","action":"rollback","outcome":"rolled_back","health":"healthy","bundle_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","external_receipt":"scenario-to-cloud:recovery:op-1","observed_at":"2026-09-08T21:00:00Z"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &HTTPCloudHealthClient{httpClient: srv.Client(), baseURL: srv.URL, pollInterval: time.Millisecond}
	receipt, err := client.RecoverCloud(context.Background(), &CloudRecoveryRequest{
		DeploymentID: "dep-1", Action: "rollback", ExpectedBundleSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RepairBundleSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", DataCompatibility: "compatible",
		IdempotencyKey: "recovery-key", Confirmation: "rollback dep-1",
	})
	if err != nil {
		t.Fatalf("RecoverCloud() error = %v", err)
	}
	if receipt == nil || receipt.OperationID != "op-1" || receipt.Outcome != "rolled_back" || receipt.BundleSHA256 == "" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if statusCalls.Load() < 2 {
		t.Fatalf("status calls = %d, expected durable operation polling", statusCalls.Load())
	}
	if received["repair_bundle_sha256"] != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" || received["data_compatibility"] != "compatible" || received["idempotency_key"] != "recovery-key" {
		t.Fatalf("recovery request = %#v", received)
	}
}

func TestDesktopPackagerClientHTTPWorkflow(t *testing.T) {
	var statusCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/desktop/generate/quick":
			var request QuickGenerateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode quick request: %v", err)
			}
			if request.ScenarioName != "demo" || request.DeploymentMode != "release" {
				t.Errorf("quick request = %+v", request)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"build_id":"b1","status":"building","scenario_name":"demo","desktop_path":"/tmp/demo","status_url":"/status/b1"}`))
		case "/api/v1/desktop/build/demo":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"build_id":"b2","status":"building","scenario":"demo","desktop_path":"/tmp/demo","platforms":["linux"],"status_url":"/status/b2"}`))
		case "/api/v1/desktop/status/b1":
			call := statusCalls.Add(1)
			status := "building"
			if call > 1 {
				status = "ready"
			}
			_, _ = w.Write([]byte(`{"build_id":"b1","scenario_name":"demo","status":"` + status + `","platforms":["linux"]}`))
		case "/api/v1/signing/demo/ready":
			_, _ = w.Write([]byte(`{"ready":true,"scenario":"demo","platforms":{"linux":{"ready":true}}}`))
		case "/health":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &DesktopPackagerClient{httpClient: srv.Client(), baseURL: srv.URL, log: func(string, map[string]interface{}) {}}
	generated, err := client.QuickGenerate(context.Background(), &QuickGenerateRequest{ScenarioName: "demo", DeploymentMode: "release", Platforms: []string{"linux"}})
	if err != nil || generated.BuildID != "b1" {
		t.Fatalf("QuickGenerate() = %+v, %v", generated, err)
	}
	built, err := client.BuildScenario(context.Background(), "demo", &ScenarioBuildRequest{Platforms: []string{"linux"}})
	if err != nil || built.BuildID != "b2" {
		t.Fatalf("BuildScenario() = %+v, %v", built, err)
	}
	ready, err := client.WaitForBuild(context.Background(), "b1", time.Millisecond)
	if err != nil || ready.Status != "ready" {
		t.Fatalf("WaitForBuild() = %+v, %v", ready, err)
	}
	if !client.IsAvailable(context.Background()) {
		t.Fatal("IsAvailable() = false")
	}
	signing, err := client.CheckSigningReadiness(context.Background(), "demo")
	if err != nil || !signing.Ready {
		t.Fatalf("CheckSigningReadiness() = %+v, %v", signing, err)
	}
}

func TestDesktopPackagerClientBuildFailureAndCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/desktop/status/fail":
			_, _ = w.Write([]byte(`{"build_id":"fail","status":"failed","error":"compiler failed"}`))
		case "/api/v1/signing/missing/ready":
			w.WriteHeader(http.StatusNotFound)
		case "/api/v1/desktop/status/bad":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("bad gateway"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := &DesktopPackagerClient{httpClient: srv.Client(), baseURL: srv.URL, log: func(string, map[string]interface{}) {}}

	failed, err := client.WaitForBuild(context.Background(), "fail", time.Millisecond)
	if failed == nil || err == nil {
		t.Fatalf("failed WaitForBuild() = %+v, %v", failed, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.WaitForBuild(ctx, "fail", time.Millisecond); err == nil {
		t.Fatal("cancelled WaitForBuild() returned nil error")
	}
	missing, err := client.CheckSigningReadiness(context.Background(), "missing")
	if err != nil || missing.Ready || len(missing.Issues) == 0 {
		t.Fatalf("missing signing readiness = %+v, %v", missing, err)
	}
	if _, err := client.GetBuildStatus(context.Background(), "bad"); err == nil {
		t.Fatal("non-200 GetBuildStatus() returned nil error")
	}
}
