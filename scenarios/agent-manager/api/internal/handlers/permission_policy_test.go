package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/protoconv"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

type permissionProjectorFake struct {
	calls int
}

func (f *permissionProjectorFake) Plan(_ context.Context, request permissionpolicy.ProjectionRequest) (permissionpolicy.ProjectionResult, error) {
	f.calls++
	return permissionProjection(request.Runner), nil
}

func (f *permissionProjectorFake) Reconcile(_ context.Context, request permissionpolicy.ProjectionRequest, authorized bool) (permissionpolicy.ProjectionResult, error) {
	if !authorized {
		return permissionpolicy.ProjectionResult{}, permissionpolicy.ErrAuthorizationRequired
	}
	f.calls++
	if request.Runner == domain.RunnerTypeGrok {
		return permissionpolicy.ProjectionResult{}, permissionpolicy.ErrResourceUnavailable
	}
	return permissionProjection(request.Runner), nil
}

func permissionProjection(runner domain.RunnerType) permissionpolicy.ProjectionResult {
	return permissionpolicy.ProjectionResult{
		Runner: runner, Scope: "user", DesiredDigest: "digest", DesiredFingerprint: "desired", LiveFingerprint: "live",
		NativePaths: []string{"/tmp/resource"}, Enforcement: permissionpolicy.EnforcementPosture{Permissions: "native"},
	}
}

func setupPermissionPolicyHandler(t *testing.T) (*permissionProjectorFake, *mux.Router) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"metadata":{"catalogId":"test","updatedAt":"2026-07-10"},"targetScopes":["user"],"rules":[{"id":"deny-root","action":"deny","matcher":{"kind":"bash","pattern":"rm -rf /"},"rationale":"test","owner":"test","targetScope":"user","requiresHardEnforcement":false}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := permissionpolicy.NewState(path, permissionpolicy.Requirement{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	projector := &permissionProjectorFake{}
	handler := New(nil, WithPermissionPolicy(state, permissionpolicy.NewService(state, projector, nil)))
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return projector, router
}

func TestPermissionPolicyStatusAndAuthorizationGate(t *testing.T) {
	projector, router := setupPermissionPolicyHandler(t)
	status := httptest.NewRecorder()
	router.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/v1/permission-policy/status", nil))
	if status.Code != http.StatusOK {
		t.Fatalf("status code = %d: %s", status.Code, status.Body.String())
	}
	var response apipb.GetPermissionPolicyStatusResponse
	if err := protoconv.UnmarshalJSON(status.Body.Bytes(), &response); err != nil || !response.Status.Ready {
		t.Fatalf("status response = %#v, err = %v", response, err)
	}

	reconcile := httptest.NewRecorder()
	router.ServeHTTP(reconcile, httptest.NewRequest(http.MethodPost, "/api/v1/permission-policy/reconcile", nil))
	if reconcile.Code != http.StatusBadRequest || projector.calls != 0 {
		t.Fatalf("unauthorized reconcile = %d %s, calls=%d", reconcile.Code, reconcile.Body.String(), projector.calls)
	}
}

func TestPermissionPolicyReconcileReturnsPartialEvidence(t *testing.T) {
	projector, router := setupPermissionPolicyHandler(t)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/permission-policy/reconcile", strings.NewReader(`{"explicitly_authorized":true}`))
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d: %s", recorder.Code, recorder.Body.String())
	}
	var response apipb.ReconcilePermissionPolicyResponse
	if err := protoconv.UnmarshalJSON(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result == nil || response.Result.Success || len(response.Result.Resources) != 4 || projector.calls != 4 {
		t.Fatalf("response = %#v, calls=%d", response.Result, projector.calls)
	}
}

func TestPermissionPolicyPlanLogReportsDriftAndUnsupportedMatchers(t *testing.T) {
	var logs bytes.Buffer
	obs.InitWithWriter("json", "info", &logs)
	t.Cleanup(func() { obs.Init("text", "info") })

	logPermissionPolicyPlan("permission_policy_planned", permissionpolicy.AggregatePlan{
		CatalogDigest:            "catalog-digest",
		HardEnforcementSatisfied: true,
		Resources: []permissionpolicy.ResourcePlan{{
			Drift:               true,
			UnsupportedMatchers: []permissionpolicy.Matcher{{Kind: "bash", Pattern: "not logged as an attribute"}},
		}},
	})

	var record map[string]any
	if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
		t.Fatalf("decode structured log: %v; output=%s", err, logs.String())
	}
	if record["msg"] != "permission_policy_planned" || record[obs.KeyPermissionPolicyDriftCount] != float64(1) || record[obs.KeyPermissionPolicyUnsupportedCount] != float64(1) {
		t.Fatalf("log record = %#v", record)
	}
	if strings.Contains(logs.String(), "not logged as an attribute") {
		t.Fatalf("log leaked portable matcher pattern: %s", logs.String())
	}
}

var _ permissionpolicy.Projector = (*permissionProjectorFake)(nil)
