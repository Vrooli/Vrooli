package redeploy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans"

	"scenario-to-cloud/cli/deployment"
	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/testfakes"
	"scenario-to-cloud/cli/internal/transport"
)

// TestRedeployIsCreateThenPlanApplyWait [REQ:STC-P0-037] proves the wrapper
// creates the record, then applies exactly the digest the plan compiled and
// waits once on the admitted operation; it carries no policy of its own.
func TestRedeployIsCreateThenPlanApplyWait(t *testing.T) {
	fake := testfakes.NewServer()
	fake.Deployments.Items = []testfakes.Deployment{{Ref: testfakes.Ref("dep-new", "demo", "production", "203.0.113.10"), Name: "demo", Status: "pending"}}
	var created map[string]any
	fake.Mux.HandleFunc("/api/v1/deployments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&created)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deployment":{"id":"dep-new","name":"demo","scenario_id":"demo","status":"pending","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"created":true,"updated":false,"timestamp":"t"}`))
	})
	fake.Plans.Compile = func(id, scope string) *plansv1.CompilePlanResponse {
		return &plansv1.CompilePlanResponse{SchemaVersion: "1", PlanDigest: "sha256:reviewed", Plan: &plansv1.ExecutablePlan{DeploymentId: id, Scope: "full", Outcome: "apply"}, Preview: &plansv1.Preview{Outcome: "apply", Changes: []*plansv1.Change{{ActionId: "release.activate", Effect: "deployment_write"}}}}
	}
	fake.Plans.Apply = func(req *plansv1.ApplyPlanRequest) *plansv1.ApplyPlanResponse {
		return &plansv1.ApplyPlanResponse{SchemaVersion: "1", OperationId: "op-7", PlanDigest: req.GetPlanDigest(), State: "admitted"}
	}
	fake.Operations.Script = []*operationsv1.OperationStanding{testfakes.Standing("op-7", "dep-new", "succeeded", true)}
	srv := httptest.NewServer(fake.Mux)
	defer srv.Close()

	manifest := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"scenario":{"id":"demo"},"target":{"vps":{"host":"203.0.113.10"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	client := deployment.NewClient(transport.ForBaseURL(srv.URL))
	if err := Run(client, []string{manifest, "--yes", "--request-key", "rk-1"}); err != nil {
		t.Fatalf("redeploy: %v", err)
	}
	if created["manifest"] == nil {
		t.Fatalf("create body missing manifest: %v", created)
	}
	if len(fake.Plans.Applied) != 1 || fake.Plans.Applied[0].GetPlanDigest() != "sha256:reviewed" || fake.Plans.Applied[0].GetRequestKey() != "rk-1" || fake.Plans.Applied[0].GetDeploymentId() != "dep-new" {
		t.Fatalf("apply = %+v", fake.Plans.Applied)
	}
	if len(fake.Operations.WaitCalls) != 1 || fake.Operations.WaitCalls[0].GetOperationId() != "op-7" {
		t.Fatalf("wait calls = %+v", fake.Operations.WaitCalls)
	}
}

// TestRedeployRequiresOneManifest [REQ:STC-P0-037] proves the legacy
// selector/if-needed/force-run modes are gone: the only input is a manifest.
func TestRedeployRequiresOneManifest(t *testing.T) {
	err := Run(nil, []string{"--domain", "vrooli.com", "--scenario", "demo", "--if-needed"})
	if err == nil {
		t.Fatal("expected a flag error for the removed selector mode")
	}
	if !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("unexpected error: %v", err)
	}
	if code := apierr.ExitCode(err); code != apierr.ExitFailed {
		t.Fatalf("exit = %d", code)
	}
}
