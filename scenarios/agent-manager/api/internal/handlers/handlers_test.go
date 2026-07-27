// Package handlers provides HTTP handlers for the agent-manager API.
// This file contains integration tests for API endpoints.
package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/protoconv"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// =============================================================================
// TEST SETUP HELPERS
// =============================================================================

// decodeProtoJSON decodes a proto JSON response body into a proto message.
func decodeProtoJSON(t *testing.T, body []byte, msg proto.Message) {
	t.Helper()
	// Use protojson via protoconv for consistent unmarshalling
	if err := protoconv.UnmarshalJSON(body, msg); err != nil {
		t.Fatalf("failed to decode proto JSON: %v\nBody: %s", err, string(body))
	}
}

func encodeProtoJSON(t *testing.T, msg proto.Message) []byte {
	t.Helper()
	body, err := protoconv.MarshalJSON(msg)
	if err != nil {
		t.Fatalf("failed to encode proto JSON: %v", err)
	}
	return body
}

// setupTestHandler creates a handler with SQLite-backed repositories for testing.
func setupTestHandler(t *testing.T) (*Handler, *mux.Router) {
	t.Helper()
	return setupTestHandlerWithRunner(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
}

// setupTestHandlerWithRunner lets lifecycle tests control runner completion
// without relying on scheduler timing.
func setupTestHandlerWithRunner(t *testing.T, mock *runner.MockRunner) (*Handler, *mux.Router) {
	t.Helper()
	handler, router, _, _ := setupTestHandlerWithRunnerAndRepos(t, mock)
	return handler, router
}

func setupTestHandlerWithRunnerAndRepos(t *testing.T, mock *runner.MockRunner) (*Handler, *mux.Router, *database.Repositories, event.Store) {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	// Create runner registry with mock runner
	registry := runner.NewRegistry()
	if err := registry.Register(mock); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	// Create orchestrator with dependencies. A DefaultProjectRoot is
	// supplied so handler tests that create tasks without an explicit
	// projectRoot can still reach RunStatusPending — SandboxConfig.Mode
	// is the single source of truth for RunMode, so a default-config
	// run resolves to sandboxed and preflightScopePath needs a project
	// root.
	roleState, err := rolepolicy.NewState(rolepolicy.ResolvePath(), rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("load role policy state: %v", err)
	}
	permissionState, err := permissionpolicy.NewState(permissionpolicy.ResolvePath(), permissionpolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("load permission policy state: %v", err)
	}
	orch := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:     5 * time.Minute,
			MaxConcurrentRuns:  10,
			DefaultProjectRoot: t.TempDir(),
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
		orchestration.WithRolePolicyState(roleState, handlerRoleResolver{}),
		orchestration.WithWorkflowRepository(repos.Workflows),
		orchestration.WithWorkflowExecutionRepository(repos.WorkflowExecutions),
	)

	// Create handler
	handler := New(
		orchestration.NewHandlerServices(orch),
		WithRolePolicyState(roleState),
		WithPermissionPolicy(permissionState, permissionpolicy.NewService(permissionState, handlerPermissionProjector{}, nil)),
	)

	// Create router and register routes
	r := mux.NewRouter()
	handler.RegisterRoutes(r)
	r.HandleFunc("/health", handler.Health).Methods("GET")
	r.HandleFunc("/api/v1/health", handler.Health).Methods("GET")

	return handler, r, repos, eventStore
}

func createRunnableTestRun(t *testing.T, router *mux.Router) *pb.Run {
	t.Helper()
	key := "sync-run-" + uuid.NewString()
	profileBody := encodeProtoJSON(t, &apipb.CreateProfileRequest{Profile: &pb.AgentProfile{
		Name:       key,
		ProfileKey: key,
		RoleRef:    "code.default",
	}})
	profileRR := httptest.NewRecorder()
	router.ServeHTTP(profileRR, httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(profileBody)))
	if profileRR.Code != http.StatusCreated {
		t.Fatalf("create profile status=%d body=%s", profileRR.Code, profileRR.Body.String())
	}
	var profile apipb.CreateProfileResponse
	decodeProtoJSON(t, profileRR.Body.Bytes(), &profile)

	taskBody := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: &pb.Task{Title: key, ScopePath: "src/sync-run"}})
	taskRR := httptest.NewRecorder()
	router.ServeHTTP(taskRR, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(taskBody)))
	if taskRR.Code != http.StatusCreated {
		t.Fatalf("create task status=%d body=%s", taskRR.Code, taskRR.Body.String())
	}
	var task apipb.CreateTaskResponse
	decodeProtoJSON(t, taskRR.Body.Bytes(), &task)

	profileID := profile.Profile.GetId()
	runBody := encodeProtoJSON(t, &apipb.CreateRunRequest{TaskId: task.Task.GetId(), AgentProfileId: &profileID})
	runRR := httptest.NewRecorder()
	router.ServeHTTP(runRR, httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(runBody)))
	if runRR.Code != http.StatusCreated {
		t.Fatalf("create run status=%d body=%s", runRR.Code, runRR.Body.String())
	}
	var run apipb.CreateRunResponse
	decodeProtoJSON(t, runRR.Body.Bytes(), &run)
	return run.Run
}

type handlerRoleResolver struct{}

func (handlerRoleResolver) Resolve(_ context.Context, runnerType domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{
		Runner: runnerType, Role: role, Model: "test-model", Provenance: rolepolicy.ResourceProvenance{Source: "handler test", ObservedAt: "2026-07-10"},
		Enforcement: rolepolicy.EnforcementPosture{Permissions: "native"}, PolicyPath: "/tmp/test-policy", PolicyDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}, nil
}

// handlerPermissionProjector preserves the operator-contract behavior while
// keeping handler tests independent from resource CLIs and native config files.
type handlerPermissionProjector struct{}

func (handlerPermissionProjector) Plan(_ context.Context, request permissionpolicy.ProjectionRequest) (permissionpolicy.ProjectionResult, error) {
	return handlerPermissionProjection(request), nil
}

func (handlerPermissionProjector) Reconcile(_ context.Context, request permissionpolicy.ProjectionRequest, explicitlyAuthorized bool) (permissionpolicy.ProjectionResult, error) {
	if !explicitlyAuthorized {
		return permissionpolicy.ProjectionResult{}, permissionpolicy.ErrAuthorizationRequired
	}
	return handlerPermissionProjection(request), nil
}

func handlerPermissionProjection(request permissionpolicy.ProjectionRequest) permissionpolicy.ProjectionResult {
	return permissionpolicy.ProjectionResult{
		Runner:             request.Runner,
		Scope:              request.Document.Scope,
		DesiredDigest:      "sha256:test",
		DesiredFingerprint: "desired:test",
		LiveFingerprint:    "live:test",
		Enforcement:        permissionpolicy.EnforcementPosture{Permissions: "native"},
		Changes:            []string{},
		NativePaths:        []string{},
	}
}

// =============================================================================
// PROFILE HANDLER TESTS
// =============================================================================
// [REQ:REQ-P0-001] Create Agent Profile
// [REQ:REQ-P0-002] Update Agent Profile

func TestEnsureProfileCreatesReturnsExistingAndUpdates(t *testing.T) {
	_, router := setupTestHandler(t)
	request := func(update bool, name string) *httptest.ResponseRecorder {
		body := encodeProtoJSON(t, &apipb.EnsureProfileRequest{
			ProfileKey:     "ensured-profile",
			Defaults:       &pb.AgentProfile{Name: name, ProfileKey: "ensured-profile", RoleRef: "code.default", MaxTurns: 10},
			UpdateExisting: update,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/ensure", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}
	first := request(false, "first")
	if first.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", first.Code, first.Body.String())
	}
	var created apipb.EnsureProfileResponse
	decodeProtoJSON(t, first.Body.Bytes(), &created)
	if !created.Created || created.Updated || created.Profile.GetName() != "first" {
		t.Fatalf("created response=%+v", &created)
	}
	existing := request(false, "ignored")
	var unchanged apipb.EnsureProfileResponse
	decodeProtoJSON(t, existing.Body.Bytes(), &unchanged)
	if existing.Code != http.StatusOK || unchanged.Created || unchanged.Updated || unchanged.Profile.GetName() != "first" {
		t.Fatalf("existing status=%d response=%+v", existing.Code, &unchanged)
	}
	updated := request(true, "updated")
	var updatedResponse apipb.EnsureProfileResponse
	decodeProtoJSON(t, updated.Body.Bytes(), &updatedResponse)
	if updated.Code != http.StatusOK || !updatedResponse.Updated || updatedResponse.Profile.GetName() != "updated" {
		t.Fatalf("updated status=%d response=%+v", updated.Code, &updatedResponse)
	}
}

func TestProbeRunnerReturnsTypedHealthResultAndRejectsUnknownRunner(t *testing.T) {
	_, router := setupTestHandler(t)
	valid := httptest.NewRecorder()
	router.ServeHTTP(valid, httptest.NewRequest(http.MethodPost, "/api/v1/runners/claude-code/probe", nil))
	if valid.Code != http.StatusOK {
		t.Fatalf("probe status=%d body=%s", valid.Code, valid.Body.String())
	}
	var response apipb.ProbeRunnerResponse
	decodeProtoJSON(t, valid.Body.Bytes(), &response)
	if response.GetResult() == nil || !response.GetResult().GetSuccess() {
		t.Fatalf("probe response=%+v", &response)
	}
	invalid := httptest.NewRecorder()
	router.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/api/v1/runners/not-a-runner/probe", nil))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid probe status=%d body=%s", invalid.Code, invalid.Body.String())
	}
}

func TestRolePolicyOperatorEndpointsExposeAndValidateActiveCatalog(t *testing.T) {
	_, router := setupTestHandler(t)

	status := httptest.NewRecorder()
	router.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/v1/role-policy/status", nil))
	if status.Code != http.StatusOK {
		t.Fatalf("status endpoint=%d body=%s", status.Code, status.Body.String())
	}
	var statusResponse apipb.GetRolePolicyStatusResponse
	decodeProtoJSON(t, status.Body.Bytes(), &statusResponse)
	if !statusResponse.GetStatus().GetReady() || statusResponse.GetStatus().GetActiveDigest() == "" {
		t.Fatalf("status response=%+v", statusResponse.GetStatus())
	}

	catalog := httptest.NewRecorder()
	router.ServeHTTP(catalog, httptest.NewRequest(http.MethodGet, "/api/v1/role-policy/catalog", nil))
	if catalog.Code != http.StatusOK {
		t.Fatalf("catalog endpoint=%d body=%s", catalog.Code, catalog.Body.String())
	}
	var catalogResponse apipb.GetRolePolicyCatalogResponse
	decodeProtoJSON(t, catalog.Body.Bytes(), &catalogResponse)
	if catalogResponse.GetCatalog() == nil || len(catalogResponse.GetCatalog().GetRoles()) == 0 {
		t.Fatalf("catalog response=%+v", catalogResponse.GetCatalog())
	}

	for _, path := range []string{"/api/v1/role-policy/validate", "/api/v1/role-policy/reload"} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, path, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}
	validate := httptest.NewRecorder()
	router.ServeHTTP(validate, httptest.NewRequest(http.MethodPost, "/api/v1/role-policy/validate", nil))
	var validateResponse apipb.ValidateRolePolicyCatalogResponse
	decodeProtoJSON(t, validate.Body.Bytes(), &validateResponse)
	if !validateResponse.GetValid() || validateResponse.GetCandidateDigest() == "" {
		t.Fatalf("validate response=%+v", &validateResponse)
	}
	reload := httptest.NewRecorder()
	router.ServeHTTP(reload, httptest.NewRequest(http.MethodPost, "/api/v1/role-policy/reload", nil))
	var reloadResponse apipb.ReloadRolePolicyCatalogResponse
	decodeProtoJSON(t, reload.Body.Bytes(), &reloadResponse)
	if !reloadResponse.GetActivated() || !reloadResponse.GetStatus().GetReady() {
		t.Fatalf("reload response=%+v", &reloadResponse)
	}
}

func TestExplainRolePolicyReturnsResolvedProfileSnapshot(t *testing.T) {
	_, router := setupTestHandler(t)
	run := createRunnableTestRun(t, router)
	if run.GetAgentProfileId() == "" {
		t.Fatalf("run is missing agent profile: %+v", run)
	}
	body := encodeProtoJSON(t, &apipb.ExplainRolePolicyRequest{
		Target: &apipb.ExplainRolePolicyRequest_ProfileId{ProfileId: run.GetAgentProfileId()},
	})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/role-policy/explain", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("explain status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response apipb.ExplainRolePolicyResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)
	if response.GetTargetType() != "profile" || response.GetTargetId() != run.GetAgentProfileId() || response.GetSnapshot() == nil || response.GetSnapshot().GetRoleRef() != "code.default" || response.GetSummary() == "" {
		t.Fatalf("explain response=%+v", &response)
	}

	for _, request := range [][]byte{
		[]byte(`{}`),
		encodeProtoJSON(t, &apipb.ExplainRolePolicyRequest{Target: &apipb.ExplainRolePolicyRequest_ProfileId{ProfileId: "not-a-uuid"}}),
	} {
		invalid := httptest.NewRecorder()
		router.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/api/v1/role-policy/explain", bytes.NewReader(request)))
		if invalid.Code != http.StatusBadRequest {
			t.Fatalf("invalid explain status=%d body=%s", invalid.Code, invalid.Body.String())
		}
	}
}

func TestPermissionPolicyReadOnlyOperatorEndpointsExposeActiveCatalog(t *testing.T) {
	_, router := setupTestHandler(t)

	status := httptest.NewRecorder()
	router.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/v1/permission-policy/status", nil))
	if status.Code != http.StatusOK {
		t.Fatalf("status endpoint=%d body=%s", status.Code, status.Body.String())
	}
	var statusResponse apipb.GetPermissionPolicyStatusResponse
	decodeProtoJSON(t, status.Body.Bytes(), &statusResponse)
	if !statusResponse.GetStatus().GetReady() || statusResponse.GetStatus().GetActiveDigest() == "" {
		t.Fatalf("status response=%+v", statusResponse.GetStatus())
	}

	catalog := httptest.NewRecorder()
	router.ServeHTTP(catalog, httptest.NewRequest(http.MethodGet, "/api/v1/permission-policy/catalog", nil))
	if catalog.Code != http.StatusOK {
		t.Fatalf("catalog endpoint=%d body=%s", catalog.Code, catalog.Body.String())
	}
	var catalogResponse apipb.GetPermissionPolicyCatalogResponse
	decodeProtoJSON(t, catalog.Body.Bytes(), &catalogResponse)
	if catalogResponse.GetCatalog() == nil || len(catalogResponse.GetCatalog().GetRules()) == 0 {
		t.Fatalf("catalog response=%+v", catalogResponse.GetCatalog())
	}

	validate := httptest.NewRecorder()
	router.ServeHTTP(validate, httptest.NewRequest(http.MethodPost, "/api/v1/permission-policy/validate", nil))
	if validate.Code != http.StatusOK {
		t.Fatalf("validate endpoint=%d body=%s", validate.Code, validate.Body.String())
	}
	var validateResponse apipb.ValidatePermissionPolicyCatalogResponse
	decodeProtoJSON(t, validate.Body.Bytes(), &validateResponse)
	if !validateResponse.GetValid() || validateResponse.GetCandidateDigest() == "" {
		t.Fatalf("validate response=%+v", &validateResponse)
	}

	reload := httptest.NewRecorder()
	router.ServeHTTP(reload, httptest.NewRequest(http.MethodPost, "/api/v1/permission-policy/reload", nil))
	if reload.Code != http.StatusOK {
		t.Fatalf("reload endpoint=%d body=%s", reload.Code, reload.Body.String())
	}
	var reloadResponse apipb.ReloadPermissionPolicyCatalogResponse
	decodeProtoJSON(t, reload.Body.Bytes(), &reloadResponse)
	if !reloadResponse.GetActivated() || !reloadResponse.GetStatus().GetReady() {
		t.Fatalf("reload response=%+v", &reloadResponse)
	}
}

func TestPermissionPolicyPlanDoctorAndAuthorizedReconcileUseDeclaredState(t *testing.T) {
	_, router := setupTestHandler(t)

	for _, path := range []string{"/api/v1/permission-policy/plan", "/api/v1/permission-policy/doctor"} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, path, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/api/v1/permission-policy/reconcile", strings.NewReader(`{}`)))
	if unauthorized.Code != http.StatusBadRequest {
		t.Fatalf("unauthorized reconcile status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	body := encodeProtoJSON(t, &apipb.ReconcilePermissionPolicyRequest{ExplicitlyAuthorized: true})
	authorized := httptest.NewRecorder()
	router.ServeHTTP(authorized, httptest.NewRequest(http.MethodPost, "/api/v1/permission-policy/reconcile", bytes.NewReader(body)))
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized reconcile status=%d body=%s", authorized.Code, authorized.Body.String())
	}
	var response apipb.ReconcilePermissionPolicyResponse
	decodeProtoJSON(t, authorized.Body.Bytes(), &response)
	if response.GetResult() == nil || !response.GetResult().GetSuccess() || len(response.GetResult().GetResources()) == 0 {
		t.Fatalf("reconcile response=%+v", &response)
	}
}

// TestCreateProfile_Success tests successful profile creation.
// [REQ:REQ-P0-001] Verify profile creation with valid data
func TestHealth_Success(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var result commonpb.HealthResponse
	if err := protoconv.UnmarshalJSON(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	// Verify required fields per health-api.schema.json
	if result.Status == commonpb.HealthStatus_HEALTH_STATUS_UNSPECIFIED {
		t.Error("health status should not be empty")
	}
	if result.Service != "agent-manager" {
		t.Errorf("expected service 'agent-manager', got '%s'", result.Service)
	}
	if result.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
	// Readiness should be true for a healthy system
	if !result.Readiness {
		t.Error("expected readiness to be true")
	}
}

// TestHealth_ApiV1Path tests the /api/v1/health endpoint.
// [REQ:REQ-P0-011] Verify both health endpoint paths work
func TestHealth_ApiV1Path(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

// =============================================================================
// RUN HANDLER TESTS
// =============================================================================
// [REQ:REQ-P0-004] Run Status Tracking
// [REQ:REQ-P0-005] Run Creation

// TestCreateRun_Success tests successful run creation.
// [REQ:REQ-P0-005] Verify run creation with valid data
func TestRequestIDMiddleware(t *testing.T) {
	_, router := setupTestHandler(t)

	// Test without request ID - should be assigned
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}

	// Test with existing request ID - should be preserved
	customID := uuid.New().String()
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", customID)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-ID") != customID {
		t.Errorf("expected X-Request-ID to be preserved, got '%s'", rr.Header().Get("X-Request-ID"))
	}
}

// =============================================================================
// ERROR RESPONSE TESTS
// =============================================================================

// TestErrorResponse_Format tests that error responses have proper structure.
func TestErrorResponse_Format(t *testing.T) {
	_, router := setupTestHandler(t)

	// Request non-existent resource
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	var errResp commonpb.ErrorResponse
	if err := protoconv.UnmarshalJSON(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code == "" {
		t.Error("error response should have code")
	}
	if errResp.Message == "" {
		t.Error("error response should have message")
	}
	if errResp.Details == nil || errResp.Details.Fields["request_id"] == nil {
		t.Error("error response should include request_id")
	}
}

// =============================================================================
// EDGE CASE TESTS - MALFORMED INPUT
// =============================================================================

// TestCreateProfile_MalformedJSON tests profile creation with invalid JSON.
