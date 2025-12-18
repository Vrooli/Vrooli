// Package handlers provides HTTP request handlers for the sandbox API.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/types"
)

// mockPinger is a mock implementation of the Pinger interface.
type mockPinger struct {
	err error
}

func (m *mockPinger) PingContext(ctx context.Context) error {
	return m.err
}

// mockDriver is a mock implementation of the Driver interface.
type mockDriver struct {
	available bool
	err       error
}

func (m *mockDriver) Type() driver.DriverType                       { return "mock" }
func (m *mockDriver) Version() string                               { return "test" }
func (m *mockDriver) IsAvailable(ctx context.Context) (bool, error) { return m.available, m.err }
func (m *mockDriver) Mount(ctx context.Context, s *types.Sandbox) (*driver.MountPaths, error) {
	return nil, nil
}
func (m *mockDriver) Unmount(ctx context.Context, s *types.Sandbox) error  { return nil }
func (m *mockDriver) Cleanup(ctx context.Context, s *types.Sandbox) error  { return nil }
func (m *mockDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	return nil, nil
}
func (m *mockDriver) IsMounted(ctx context.Context, s *types.Sandbox) (bool, error) {
	return true, nil
}
func (m *mockDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	return nil
}
func (m *mockDriver) Exec(ctx context.Context, s *types.Sandbox, cfg driver.BwrapConfig, cmd string, args ...string) (*driver.ExecResult, error) {
	return &driver.ExecResult{ExitCode: 0}, nil
}
func (m *mockDriver) StartProcess(ctx context.Context, s *types.Sandbox, cfg driver.BwrapConfig, cmd string, args ...string) (int, error) {
	return 12345, nil
}
func (m *mockDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	return nil
}

// TestHealthHandler tests the Health endpoint handler.
// [REQ:REQ-P0-010] Health Check API Endpoint - unit test for status codes
func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		dbAvailable    bool
		driverAvail    bool
		wantStatus     string
		wantReadiness  bool
		wantHTTPStatus int
	}{
		{
			name:           "healthy when db and driver available",
			dbAvailable:    true,
			driverAvail:    true,
			wantStatus:     "healthy",
			wantReadiness:  true,
			wantHTTPStatus: http.StatusOK,
		},
		{
			name:           "unhealthy when db unavailable",
			dbAvailable:    false,
			driverAvail:    true,
			wantStatus:     "unhealthy",
			wantReadiness:  false,
			wantHTTPStatus: http.StatusOK,
		},
		{
			name:           "healthy when driver unavailable but db ok",
			dbAvailable:    true,
			driverAvail:    false,
			wantStatus:     "healthy",
			wantReadiness:  true,
			wantHTTPStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var dbErr error
			if !tc.dbAvailable {
				dbErr = context.DeadlineExceeded
			}

			h := &Handlers{
				DB:     &mockPinger{err: dbErr},
				Driver: &mockDriver{available: tc.driverAvail},
				Config: config.Config{},
			}

			req := httptest.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()

			h.Health(rr, req)

			if rr.Code != tc.wantHTTPStatus {
				t.Errorf("Health() status = %d, want %d", rr.Code, tc.wantHTTPStatus)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if resp["status"] != tc.wantStatus {
				t.Errorf("Health() status = %v, want %v", resp["status"], tc.wantStatus)
			}

			if resp["readiness"] != tc.wantReadiness {
				t.Errorf("Health() readiness = %v, want %v", resp["readiness"], tc.wantReadiness)
			}
		})
	}
}

// TestHealthResponseSchema tests that the health endpoint returns the required JSON schema.
// [REQ:REQ-P0-010] Health Check API Endpoint - JSON response schema validation
func TestHealthResponseSchema(t *testing.T) {
	h := &Handlers{
		DB:     &mockPinger{err: nil},
		Driver: &mockDriver{available: true},
		Config: config.Config{},
	}

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	h.Health(rr, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check required fields exist
	requiredFields := []string{"status", "service", "version", "readiness", "timestamp", "dependencies"}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("Health response missing required field: %s", field)
		}
	}

	// Check dependencies structure
	deps, ok := resp["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatal("dependencies should be a map")
	}

	if _, ok := deps["database"]; !ok {
		t.Error("dependencies missing 'database' field")
	}
	if _, ok := deps["driver"]; !ok {
		t.Error("dependencies missing 'driver' field")
	}
}

// TestHealthContentType verifies the health endpoint returns JSON content type.
// [REQ:REQ-P0-010] Health Check API Endpoint - content type validation
func TestHealthContentType(t *testing.T) {
	h := &Handlers{
		DB:     &mockPinger{err: nil},
		Driver: &mockDriver{available: true},
		Config: config.Config{},
	}

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	h.Health(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Health() Content-Type = %v, want application/json", contentType)
	}
}

// --- Mock Service Implementation for Handler Tests ---

// mockService implements sandbox.ServiceAPI for testing.
type mockService struct {
	createFn          func(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error)
	getFn             func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)
	listFn            func(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error)
	stopFn            func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)
	deleteFn          func(ctx context.Context, id uuid.UUID) error
	getDiffFn         func(ctx context.Context, id uuid.UUID) (*types.DiffResult, error)
	approveFn         func(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error)
	rejectFn          func(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error)
	discardFn         func(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error)
	getWorkspaceFn    func(ctx context.Context, id uuid.UUID) (string, error)
	checkConflictsFn  func(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error)
	rebaseFn          func(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error)
	validatePathFn    func(ctx context.Context, path, projectRoot string) (*types.PathValidationResult, error)
}

// Verify mockService implements ServiceAPI
var _ sandbox.ServiceAPI = (*mockService)(nil)

func (m *mockService) Create(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockService) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockService) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockService) Stop(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	if m.stopFn != nil {
		return m.stopFn(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return fmt.Errorf("not implemented")
}

func (m *mockService) GetDiff(ctx context.Context, id uuid.UUID) (*types.DiffResult, error) {
	if m.getDiffFn != nil {
		return m.getDiffFn(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockService) Approve(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error) {
	if m.approveFn != nil {
		return m.approveFn(ctx, req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockService) Reject(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error) {
	if m.rejectFn != nil {
		return m.rejectFn(ctx, id, actor)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockService) Discard(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error) {
	if m.discardFn != nil {
		return m.discardFn(ctx, req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockService) GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error) {
	if m.getWorkspaceFn != nil {
		return m.getWorkspaceFn(ctx, id)
	}
	return "", fmt.Errorf("not implemented")
}

func (m *mockService) CheckConflicts(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error) {
	if m.checkConflictsFn != nil {
		return m.checkConflictsFn(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockService) Rebase(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error) {
	if m.rebaseFn != nil {
		return m.rebaseFn(ctx, req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockService) ValidatePath(ctx context.Context, path, projectRoot string) (*types.PathValidationResult, error) {
	if m.validatePathFn != nil {
		return m.validatePathFn(ctx, path, projectRoot)
	}
	return nil, fmt.Errorf("not implemented")
}

// --- CreateSandbox Handler Tests ---

// TestCreateSandboxSuccess tests successful sandbox creation.
// [REQ:REQ-P0-001] Fast Sandbox Creation - API handler creates sandbox and returns response
func TestCreateSandboxSuccess(t *testing.T) {
	testID := uuid.New()
	now := time.Now()

	svc := &mockService{
		createFn: func(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
			return &types.Sandbox{
				ID:            testID,
				ScopePath:     req.ScopePath,
				ProjectRoot:   req.ProjectRoot,
				Owner:         req.Owner,
				Status:        types.StatusActive,
				Driver:        "overlayfs",
				DriverVersion: "1.0",
				CreatedAt:     now,
				MergedDir:     "/tmp/sandbox/" + testID.String() + "/merged",
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	body := `{"scopePath": "/project/src", "projectRoot": "/project", "owner": "test-agent"}`
	req := httptest.NewRequest("POST", "/sandboxes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSandbox(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("CreateSandbox() status = %d, want %d", rr.Code, http.StatusCreated)
	}

	var resp types.Sandbox
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.ID != testID {
		t.Errorf("CreateSandbox() ID = %v, want %v", resp.ID, testID)
	}
	if resp.Status != types.StatusActive {
		t.Errorf("CreateSandbox() status = %v, want %v", resp.Status, types.StatusActive)
	}
	if resp.ScopePath != "/project/src" {
		t.Errorf("CreateSandbox() scopePath = %v, want /project/src", resp.ScopePath)
	}
}

// TestCreateSandboxInvalidJSON tests sandbox creation with invalid JSON body.
// [REQ:REQ-P0-001] Fast Sandbox Creation - API returns 400 for invalid input
func TestCreateSandboxInvalidJSON(t *testing.T) {
	h := &Handlers{
		Service: &mockService{},
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/sandboxes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSandbox(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("CreateSandbox() status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Success {
		t.Error("CreateSandbox() success = true, want false")
	}
}

// TestCreateSandboxScopeConflict tests sandbox creation with scope conflict.
// [REQ:REQ-P0-005] Scope Path Validation - API returns 409 for conflicting scope
func TestCreateSandboxScopeConflict(t *testing.T) {
	svc := &mockService{
		createFn: func(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
			return nil, &types.ScopeConflictError{
				Conflicts: []types.PathConflict{
					{
						ExistingID:    "abc123",
						ExistingScope: "/project/src",
						NewScope:      "/project/src/sub",
						ConflictType:  types.ConflictTypeExistingContainsNew,
					},
				},
			}
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	body := `{"scopePath": "/project/src/sub", "projectRoot": "/project"}`
	req := httptest.NewRequest("POST", "/sandboxes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSandbox(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("CreateSandbox() status = %d, want %d", rr.Code, http.StatusConflict)
	}
}

// --- ListSandboxes Handler Tests ---

// TestListSandboxesEmpty tests listing sandboxes when none exist.
// [REQ:REQ-P0-007] Sandbox Lifecycle Management - API returns empty list
func TestListSandboxesEmpty(t *testing.T) {
	svc := &mockService{
		listFn: func(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
			return &types.ListResult{
				Sandboxes:  []*types.Sandbox{},
				TotalCount: 0,
				Limit:      100,
				Offset:     0,
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("GET", "/sandboxes", nil)
	rr := httptest.NewRecorder()

	h.ListSandboxes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListSandboxes() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp types.ListResult
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp.Sandboxes) != 0 {
		t.Errorf("ListSandboxes() count = %d, want 0", len(resp.Sandboxes))
	}
}

// TestListSandboxesWithFilter tests listing sandboxes with query filters.
// [REQ:REQ-P0-007] Sandbox Lifecycle Management - API filters by status
func TestListSandboxesWithFilter(t *testing.T) {
	var capturedFilter *types.ListFilter
	svc := &mockService{
		listFn: func(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
			capturedFilter = filter
			return &types.ListResult{
				Sandboxes:  []*types.Sandbox{},
				TotalCount: 0,
				Limit:      50,
				Offset:     10,
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("GET", "/sandboxes?status=active&status=stopped&owner=agent1&limit=50&offset=10", nil)
	rr := httptest.NewRecorder()

	h.ListSandboxes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListSandboxes() status = %d, want %d", rr.Code, http.StatusOK)
	}

	if capturedFilter == nil {
		t.Fatal("Filter was not captured")
	}

	if len(capturedFilter.Status) != 2 {
		t.Errorf("Filter status count = %d, want 2", len(capturedFilter.Status))
	}
	if capturedFilter.Owner != "agent1" {
		t.Errorf("Filter owner = %s, want agent1", capturedFilter.Owner)
	}
	if capturedFilter.Limit != 50 {
		t.Errorf("Filter limit = %d, want 50", capturedFilter.Limit)
	}
	if capturedFilter.Offset != 10 {
		t.Errorf("Filter offset = %d, want 10", capturedFilter.Offset)
	}
}

// TestListSandboxesWithResults tests listing sandboxes with actual results.
// [REQ:REQ-P0-007] Sandbox Lifecycle Management - API returns sandbox list
func TestListSandboxesWithResults(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	svc := &mockService{
		listFn: func(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
			return &types.ListResult{
				Sandboxes: []*types.Sandbox{
					{ID: id1, ScopePath: "/project/src", Status: types.StatusActive},
					{ID: id2, ScopePath: "/project/tests", Status: types.StatusStopped},
				},
				TotalCount: 2,
				Limit:      100,
				Offset:     0,
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("GET", "/sandboxes", nil)
	rr := httptest.NewRecorder()

	h.ListSandboxes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListSandboxes() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp types.ListResult
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp.Sandboxes) != 2 {
		t.Errorf("ListSandboxes() count = %d, want 2", len(resp.Sandboxes))
	}
	if resp.TotalCount != 2 {
		t.Errorf("ListSandboxes() totalCount = %d, want 2", resp.TotalCount)
	}
}

// --- GetSandbox Handler Tests ---

// TestGetSandboxSuccess tests successfully getting a sandbox.
// [REQ:REQ-P0-002] Stable Sandbox Identifier - API retrieves sandbox by ID
func TestGetSandboxSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &mockService{
		getFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return &types.Sandbox{
				ID:          testID,
				ScopePath:   "/project/src",
				ProjectRoot: "/project",
				Status:      types.StatusActive,
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("GET", "/sandboxes/"+testID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.GetSandbox(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GetSandbox() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp types.Sandbox
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.ID != testID {
		t.Errorf("GetSandbox() ID = %v, want %v", resp.ID, testID)
	}
}

// TestGetSandboxNotFound tests getting a non-existent sandbox.
// [REQ:REQ-P0-002] Stable Sandbox Identifier - API returns 404 for unknown ID
func TestGetSandboxNotFound(t *testing.T) {
	testID := uuid.New()
	svc := &mockService{
		getFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return nil, types.NewNotFoundError(id.String())
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("GET", "/sandboxes/"+testID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.GetSandbox(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GetSandbox() status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// TestGetSandboxInvalidID tests getting a sandbox with invalid UUID.
// [REQ:REQ-P0-002] Stable Sandbox Identifier - API validates UUID format
func TestGetSandboxInvalidID(t *testing.T) {
	h := &Handlers{
		Service: &mockService{},
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("GET", "/sandboxes/not-a-uuid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-a-uuid"})
	rr := httptest.NewRecorder()

	h.GetSandbox(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("GetSandbox() status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// --- DeleteSandbox Handler Tests ---

// TestDeleteSandboxSuccess tests successfully deleting a sandbox.
// [REQ:REQ-P0-007] Sandbox Lifecycle Management - API deletes sandbox
func TestDeleteSandboxSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &mockService{
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("DELETE", "/sandboxes/"+testID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.DeleteSandbox(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("DeleteSandbox() status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

// TestDeleteSandboxNotFound tests deleting a non-existent sandbox.
// [REQ:REQ-P0-007] Sandbox Lifecycle Management - API returns 404 for unknown ID
func TestDeleteSandboxNotFound(t *testing.T) {
	testID := uuid.New()
	svc := &mockService{
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			return types.NewNotFoundError(id.String())
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("DELETE", "/sandboxes/"+testID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.DeleteSandbox(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("DeleteSandbox() status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// --- StopSandbox Handler Tests ---

// TestStopSandboxSuccess tests successfully stopping a sandbox.
// [REQ:REQ-P0-007] Sandbox Lifecycle Management - API stops sandbox
func TestStopSandboxSuccess(t *testing.T) {
	testID := uuid.New()
	now := time.Now()
	svc := &mockService{
		stopFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return &types.Sandbox{
				ID:        testID,
				Status:    types.StatusStopped,
				StoppedAt: &now,
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("POST", "/sandboxes/"+testID.String()+"/stop", nil)
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.StopSandbox(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("StopSandbox() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp types.Sandbox
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Status != types.StatusStopped {
		t.Errorf("StopSandbox() status = %v, want %v", resp.Status, types.StatusStopped)
	}
}

// --- GetDiff Handler Tests ---

// TestGetDiffSuccess tests successfully getting sandbox diff.
// [REQ:REQ-P0-006] Diff Generation - API returns unified diff
func TestGetDiffSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &mockService{
		getDiffFn: func(ctx context.Context, id uuid.UUID) (*types.DiffResult, error) {
			return &types.DiffResult{
				SandboxID:     testID,
				UnifiedDiff:   "--- a/file.txt\n+++ b/file.txt\n@@ -1 +1 @@\n-old\n+new",
				Files:         []*types.FileChange{},
				TotalModified: 1,
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("GET", "/sandboxes/"+testID.String()+"/diff", nil)
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.GetDiff(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GetDiff() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp types.DiffResult
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.UnifiedDiff == "" {
		t.Error("GetDiff() unifiedDiff should not be empty")
	}
}

// --- Approve Handler Tests ---

// TestApproveAllSuccess tests successfully approving all changes.
// [REQ:REQ-P0-007] Patch Application - API approves all changes
func TestApproveAllSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &mockService{
		approveFn: func(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error) {
			return &types.ApprovalResult{
				Success:    true,
				Applied:    3,
				CommitHash: "abc123",
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	body := `{"mode": "all", "actor": "test-user"}`
	req := httptest.NewRequest("POST", "/sandboxes/"+testID.String()+"/approve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.Approve(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Approve() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp types.ApprovalResult
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Error("Approve() success = false, want true")
	}
	if resp.Applied != 3 {
		t.Errorf("Approve() applied = %d, want 3", resp.Applied)
	}
}

// --- Reject Handler Tests ---

// TestRejectSuccess tests successfully rejecting sandbox changes.
// [REQ:REQ-P0-007] Sandbox Lifecycle Management - API rejects sandbox
func TestRejectSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &mockService{
		rejectFn: func(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error) {
			return &types.Sandbox{
				ID:     testID,
				Status: types.StatusRejected,
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	body := `{"actor": "test-user"}`
	req := httptest.NewRequest("POST", "/sandboxes/"+testID.String()+"/reject", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.Reject(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Reject() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp types.Sandbox
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Status != types.StatusRejected {
		t.Errorf("Reject() status = %v, want %v", resp.Status, types.StatusRejected)
	}
}

// --- Discard Handler Tests ---

// TestDiscardSuccess tests successfully discarding files from a sandbox.
func TestDiscardSuccess(t *testing.T) {
	testID := uuid.New()
	fileID := uuid.New()
	svc := &mockService{
		discardFn: func(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error) {
			return &types.DiscardResult{
				Success:   true,
				Discarded: 1,
				Remaining: 2,
				Files:     []string{"test.txt"},
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	body := fmt.Sprintf(`{"fileIds": ["%s"], "actor": "test-user"}`, fileID.String())
	req := httptest.NewRequest("POST", "/sandboxes/"+testID.String()+"/discard", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.Discard(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Discard() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp types.DiscardResult
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Error("Discard() should return success=true")
	}
	if resp.Discarded != 1 {
		t.Errorf("Discard() discarded = %d, want 1", resp.Discarded)
	}
	if resp.Remaining != 2 {
		t.Errorf("Discard() remaining = %d, want 2", resp.Remaining)
	}
}

// TestDiscardWithFilePaths tests discarding using file paths instead of IDs.
func TestDiscardWithFilePaths(t *testing.T) {
	testID := uuid.New()
	svc := &mockService{
		discardFn: func(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error) {
			if len(req.FilePaths) != 1 || req.FilePaths[0] != "path/to/file.txt" {
				t.Errorf("Expected filePaths to contain path/to/file.txt, got %v", req.FilePaths)
			}
			return &types.DiscardResult{
				Success:   true,
				Discarded: 1,
				Remaining: 0,
				Files:     req.FilePaths,
			}, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	body := `{"filePaths": ["path/to/file.txt"]}`
	req := httptest.NewRequest("POST", "/sandboxes/"+testID.String()+"/discard", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.Discard(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Discard() status = %d, want %d", rr.Code, http.StatusOK)
	}
}

// TestDiscardMissingFiles tests that discard requires files to be specified.
func TestDiscardMissingFiles(t *testing.T) {
	testID := uuid.New()
	h := &Handlers{
		Service: &mockService{},
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	body := `{"actor": "test-user"}`
	req := httptest.NewRequest("POST", "/sandboxes/"+testID.String()+"/discard", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.Discard(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Discard() with no files status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// TestDiscardInvalidSandboxID tests that discard rejects invalid sandbox IDs.
func TestDiscardInvalidSandboxID(t *testing.T) {
	h := &Handlers{
		Service: &mockService{},
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	body := `{"filePaths": ["test.txt"]}`
	req := httptest.NewRequest("POST", "/sandboxes/invalid-uuid/discard", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	rr := httptest.NewRecorder()

	h.Discard(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Discard() with invalid ID status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// --- GetWorkspace Handler Tests ---

// TestGetWorkspaceSuccess tests getting workspace path.
// [REQ:REQ-P0-001] Fast Sandbox Creation - API provides workspace path
func TestGetWorkspaceSuccess(t *testing.T) {
	testID := uuid.New()
	expectedPath := "/tmp/sandbox/" + testID.String() + "/merged"
	svc := &mockService{
		getWorkspaceFn: func(ctx context.Context, id uuid.UUID) (string, error) {
			return expectedPath, nil
		},
	}

	h := &Handlers{
		Service: svc,
		DB:      &mockPinger{},
		Driver:  &mockDriver{available: true},
		Config:  config.Config{},
	}

	req := httptest.NewRequest("GET", "/sandboxes/"+testID.String()+"/workspace", nil)
	req = mux.SetURLVars(req, map[string]string{"id": testID.String()})
	rr := httptest.NewRecorder()

	h.GetWorkspace(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GetWorkspace() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["path"] != expectedPath {
		t.Errorf("GetWorkspace() path = %v, want %v", resp["path"], expectedPath)
	}
}

// --- DriverInfo Handler Tests ---

// TestDriverInfoAvailable tests getting driver info when available.
// [REQ:REQ-P0-003] Overlayfs Copy-on-Write Driver - API reports driver status
func TestDriverInfoAvailable(t *testing.T) {
	h := &Handlers{
		DB:     &mockPinger{},
		Driver: &mockDriver{available: true},
		Config: config.Config{},
	}

	req := httptest.NewRequest("GET", "/driver", nil)
	rr := httptest.NewRecorder()

	h.DriverInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("DriverInfo() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["available"] != true {
		t.Errorf("DriverInfo() available = %v, want true", resp["available"])
	}
}

// TestDriverInfoUnavailable tests getting driver info when unavailable.
// [REQ:REQ-P0-003] Overlayfs Copy-on-Write Driver - API reports unavailable status
func TestDriverInfoUnavailable(t *testing.T) {
	h := &Handlers{
		DB:     &mockPinger{},
		Driver: &mockDriver{available: false, err: fmt.Errorf("overlayfs not supported")},
		Config: config.Config{},
	}

	req := httptest.NewRequest("GET", "/driver", nil)
	rr := httptest.NewRecorder()

	h.DriverInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("DriverInfo() status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["available"] != false {
		t.Errorf("DriverInfo() available = %v, want false", resp["available"])
	}
}
