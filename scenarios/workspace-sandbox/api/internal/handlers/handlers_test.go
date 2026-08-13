// Package handlers_test runs every middleware-relevant handler test
// through the live-HTTP harness (testutil/httpx.NewLiveServer) so the
// production middleware stack wraps every request — closing the gap
// that let the 2026-04-28 SSE flusher bug ship under recorder-only
// tests.
//
// External test package (handlers_test) breaks the import cycle:
// testutil/httpx imports handlers, so handler tests must live outside
// the handlers package to use the harness. Pure helper-function unit
// tests that need no router and no middleware (process_git_allowlist,
// process_loc bound) stay in `package handlers` and use
// httptest.ResponseRecorder; that allowance is encoded in
// handler_test_pattern_test.go.
package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	httpx "github.com/vrooli/api-core/servertest"
	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
	"workspace-sandbox/internal/types"

	"github.com/vrooli/api-core/schedule"
)

// liveOpt customizes the *handlers.Handlers built by newLive before
// the harness boots. Tests reach for these to add a non-default
// driver, scope a custom DB error, etc.
type liveOpt func(*handlers.Handlers)

func withDriver(d driver.Driver) liveOpt {
	return func(h *handlers.Handlers) { h.DriverSlot = driver.NewSlot(d) }
}

func withStarter(s process.Starter) liveOpt {
	return func(h *handlers.Handlers) { h.Starter = s }
}

// newLive builds a *handlers.Handlers using a default-fake stack and
// wires it into the live-HTTP harness. The returned LiveServer carries
// a real *http.Client and an httptest.Server backed by the production
// middleware. Tests issue requests through live.Do/DoJSON.
func newLive(t *testing.T, svc sandbox.ServiceAPI, opts ...liveOpt) *httpx.LiveServer {
	t.Helper()
	h := &handlers.Handlers{
		Clock:      schedule.System(),
		Service:    svc,
		DB:         mocks.NewFakePinger(),
		DriverSlot: driver.NewSlot(mocks.NewFakeDriver()),
		Config:     config.Config{},
	}
	for _, opt := range opts {
		opt(h)
	}
	return httpx.NewLiveServer(t, h)
}

// driverWithErr returns a FakeDriver that reports unavailability with
// the given error. Used by the DriverInfo unavailable-state test.
func driverWithErr(available bool, err error) *mocks.FakeDriver {
	d := mocks.NewFakeDriver()
	d.Available = available
	d.IsAvailableErr = err
	return d
}

// sandboxesPath builds the canonical /api/v1/sandboxes... URL for a
// sandbox-scoped endpoint. Centralizes the prefix so tests don't drift.
func sandboxesPath(id uuid.UUID, suffix string) string {
	return "/api/v1/sandboxes/" + id.String() + suffix
}

// --- CreateSandbox Handler Tests ---

// TestCreateSandboxSuccess tests successful sandbox creation.
// [REQ:REQ-P0-001] Fast Sandbox Creation - API handler creates sandbox and returns response
func TestCreateSandboxSuccess(t *testing.T) {
	testID := uuid.New()
	now := time.Now()

	svc := &sandboxiface.FakeService{
		CreateFn: func(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
			return &types.Sandbox{
				ID:            testID,
				ScopePath:     req.ScopePath,
				ProjectRoot:   req.ProjectRoot,
				Owner:         req.Owner,
				Status:        types.StatusActive,
				DriverID:      "overlayfs-userns",
				DriverVersion: "1.0",
				CreatedAt:     now,
				MergedDir:     "/tmp/sandbox/" + testID.String() + "/merged",
			}, nil
		},
	}

	live := newLive(t, svc)
	resp, body := live.DoJSON(t, "POST", "/api/v1/sandboxes",
		`{"scopePath": "/project/src", "projectRoot": "/project", "owner": "test-agent"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("CreateSandbox status = %d, want %d; body=%s", resp.StatusCode, http.StatusCreated, body)
	}

	var got types.Sandbox
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.ID != testID {
		t.Errorf("ID = %v, want %v", got.ID, testID)
	}
	if got.Status != types.StatusActive {
		t.Errorf("status = %v, want %v", got.Status, types.StatusActive)
	}
	if got.ScopePath != "/project/src" {
		t.Errorf("scopePath = %q, want /project/src", got.ScopePath)
	}
}

// TestCreateSandboxInvalidJSON tests sandbox creation with invalid JSON body.
// [REQ:REQ-P0-001] Fast Sandbox Creation - API returns 400 for invalid input
func TestCreateSandboxInvalidJSON(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{})
	resp, body := live.DoJSON(t, "POST", "/api/v1/sandboxes", `{invalid json}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, body)
	}

	var got handlers.ErrorResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Success {
		t.Error("success = true, want false")
	}
}

// TestCreateSandboxScopeConflict tests sandbox creation with scope conflict.
// [REQ:REQ-P0-005] Scope Path Validation - API returns 409 for conflicting scope
func TestCreateSandboxScopeConflict(t *testing.T) {
	svc := &sandboxiface.FakeService{
		CreateFn: func(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
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
	live := newLive(t, svc)
	resp, _ := live.DoJSON(t, "POST", "/api/v1/sandboxes",
		`{"scopePath": "/project/src/sub", "projectRoot": "/project"}`)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", resp.StatusCode)
	}
}

// --- ListSandboxes Handler Tests ---

func TestListSandboxesEmpty(t *testing.T) {
	svc := &sandboxiface.FakeService{
		ListFn: func(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
			return &types.ListResult{Sandboxes: []*types.Sandbox{}, TotalCount: 0, Limit: 100}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "GET", "/api/v1/sandboxes", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.ListResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Sandboxes) != 0 {
		t.Errorf("count = %d, want 0", len(got.Sandboxes))
	}
}

func TestListSandboxesWithFilter(t *testing.T) {
	var captured *types.ListFilter
	svc := &sandboxiface.FakeService{
		ListFn: func(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
			captured = filter
			return &types.ListResult{Limit: 50, Offset: 10}, nil
		},
	}
	live := newLive(t, svc)
	resp, _ := live.Do(t, "GET",
		"/api/v1/sandboxes?status=active&status=stopped&owner=agent1&limit=50&offset=10", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if captured == nil {
		t.Fatal("filter not captured")
	}
	if len(captured.Status) != 2 {
		t.Errorf("status filter count = %d, want 2", len(captured.Status))
	}
	if captured.Owner != "agent1" {
		t.Errorf("owner = %q, want agent1", captured.Owner)
	}
	if captured.Limit != 50 || captured.Offset != 10 {
		t.Errorf("limit/offset = %d/%d, want 50/10", captured.Limit, captured.Offset)
	}
}

func TestListSandboxesWithResults(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	svc := &sandboxiface.FakeService{
		ListFn: func(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
			return &types.ListResult{
				Sandboxes: []*types.Sandbox{
					{ID: id1, ScopePath: "/project/src", Status: types.StatusActive},
					{ID: id2, ScopePath: "/project/tests", Status: types.StatusStopped},
				},
				TotalCount: 2,
				Limit:      100,
			}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "GET", "/api/v1/sandboxes", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.ListResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Sandboxes) != 2 || got.TotalCount != 2 {
		t.Errorf("got %d sandboxes / total=%d, want 2/2", len(got.Sandboxes), got.TotalCount)
	}
}

// --- GetSandbox Handler Tests ---

func TestGetSandboxSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		GetFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return &types.Sandbox{
				ID: testID, ScopePath: "/project/src", ProjectRoot: "/project",
				Status: types.StatusActive,
			}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "GET", sandboxesPath(testID, ""), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.Sandbox
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != testID {
		t.Errorf("ID = %v, want %v", got.ID, testID)
	}
}

// TestGetSandboxStampsWorkspaceLayout pins that GetSandbox reports the
// negotiated workspace contract (workspacePath, pathIllusion, containment)
// derived from the active driver's required containment and the host
// containment probe. Two layouts:
//
//   - contained (ContainmentRequired + bwrap available): pathIllusion true,
//     workspacePath "/workspace", containment backend bwrap.
//   - identity (ContainmentNone): pathIllusion false, workspacePath ==
//     mergedDir, containment backend none.
func TestGetSandboxStampsWorkspaceLayout(t *testing.T) {
	const merged = "/tmp/ws-sandbox/merged"

	cases := []struct {
		name         string
		level        driver.ContainmentLevel
		bwrap        bool
		wantPath     string
		wantIllusion bool
		wantBackend  string
		wantEnforceN int
	}{
		{"contained-bwrap", driver.ContainmentRequired, true, "/workspace", true, "bwrap", 4},
		{"identity-none", driver.ContainmentNone, false, merged, false, "none", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testID := uuid.New()
			svc := &sandboxiface.FakeService{
				GetFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
					return &types.Sandbox{
						ID: testID, Status: types.StatusActive, MergedDir: merged,
					}, nil
				},
			}
			d := mocks.NewFakeDriver()
			d.ContainmentLevelVal = tc.level
			starter := procmocks.NewFakeStarter()
			if tc.bwrap {
				starter.SetLookPath("bwrap", "/usr/bin/bwrap")
				starter.AddCommand("/usr/bin/bwrap --version", procmocks.CommandBehavior{
					Stdout: []byte("bubblewrap 0.8.0\n"),
				})
			}
			live := newLive(t, svc, withDriver(d), withStarter(starter))

			resp, body := live.Do(t, "GET", sandboxesPath(testID, ""), nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
			}
			var got types.Sandbox
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.WorkspacePath != tc.wantPath {
				t.Errorf("workspacePath = %q, want %q", got.WorkspacePath, tc.wantPath)
			}
			if got.PathIllusion != tc.wantIllusion {
				t.Errorf("pathIllusion = %v, want %v", got.PathIllusion, tc.wantIllusion)
			}
			if got.Containment == nil {
				t.Fatal("containment must be present on the response")
			}
			if got.Containment.Backend != tc.wantBackend {
				t.Errorf("containment.backend = %q, want %q", got.Containment.Backend, tc.wantBackend)
			}
			if len(got.Containment.Enforcements) != tc.wantEnforceN {
				t.Errorf("containment.enforcements = %v, want %d", got.Containment.Enforcements, tc.wantEnforceN)
			}
		})
	}
}

func TestGetSandboxNotFound(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		GetFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return nil, types.NewNotFoundError(id.String())
		},
	}
	live := newLive(t, svc)
	resp, _ := live.Do(t, "GET", sandboxesPath(testID, ""), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetSandboxInvalidID(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{})
	resp, _ := live.Do(t, "GET", "/api/v1/sandboxes/not-a-uuid", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// --- DeleteSandbox Handler Tests ---

func TestDeleteSandboxSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		DeleteFn: func(ctx context.Context, id uuid.UUID) error { return nil },
	}
	live := newLive(t, svc)
	resp, _ := live.Do(t, "DELETE", sandboxesPath(testID, ""), nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}
}

func TestDeleteSandboxNotFound(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		DeleteFn: func(ctx context.Context, id uuid.UUID) error {
			return types.NewNotFoundError(id.String())
		},
	}
	live := newLive(t, svc)
	resp, _ := live.Do(t, "DELETE", sandboxesPath(testID, ""), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// --- StopSandbox Handler Tests ---

func TestStopSandboxSuccess(t *testing.T) {
	testID := uuid.New()
	now := time.Now()
	svc := &sandboxiface.FakeService{
		StopFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return &types.Sandbox{ID: testID, Status: types.StatusStopped, StoppedAt: &now}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "POST", sandboxesPath(testID, "/stop"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.Sandbox
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != types.StatusStopped {
		t.Errorf("status = %v, want %v", got.Status, types.StatusStopped)
	}
}

// --- GetDiff Handler Tests ---

func TestGetDiffSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		// GetDiff dispatches on sandbox status; an Active sandbox routes
		// to the live diff path.
		GetFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return &types.Sandbox{ID: id, Status: types.StatusActive, UpdatedAt: time.Now()}, nil
		},
		GetDiffFn: func(ctx context.Context, id uuid.UUID) (*types.DiffResult, error) {
			return &types.DiffResult{
				SandboxID:   testID,
				UnifiedDiff: "--- a/file.txt\n+++ b/file.txt\n@@ -1 +1 @@\n-old\n+new",
				Files:       []*types.FileChange{},
				Stats: types.DiffStats{
					FilesChanged: 1, FilesModified: 1, LinesAdded: 1, LinesRemoved: 1,
				},
			}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "GET", sandboxesPath(testID, "/diff"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.DiffResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.UnifiedDiff == "" {
		t.Error("UnifiedDiff should not be empty")
	}
}

// --- Approve Handler Tests ---

func TestApproveAllSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		ApproveFn: func(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error) {
			return &types.ApprovalResult{Success: true, Applied: 3, CommitHash: "abc123"}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.DoJSON(t, "POST", sandboxesPath(testID, "/approve"),
		`{"mode": "all", "actor": "test-user"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.ApprovalResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Success {
		t.Error("success = false, want true")
	}
	if got.Applied != 3 {
		t.Errorf("Applied = %d, want 3", got.Applied)
	}
}

// --- Reject Handler Tests ---

func TestRejectSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		RejectFn: func(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error) {
			return &types.Sandbox{ID: testID, Status: types.StatusRejected}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.DoJSON(t, "POST", sandboxesPath(testID, "/reject"), `{"actor": "test-user"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.Sandbox
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != types.StatusRejected {
		t.Errorf("status = %v, want %v", got.Status, types.StatusRejected)
	}
}

// --- Discard Handler Tests ---

func TestDiscardSuccess(t *testing.T) {
	testID := uuid.New()
	fileID := uuid.New()
	svc := &sandboxiface.FakeService{
		DiscardFn: func(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error) {
			return &types.DiscardResult{
				Success: true, Discarded: 1, Remaining: 2, Files: []string{"test.txt"},
			}, nil
		},
	}
	live := newLive(t, svc)
	body := fmt.Sprintf(`{"fileIds": ["%s"], "actor": "test-user"}`, fileID.String())
	resp, respBody := live.DoJSON(t, "POST", sandboxesPath(testID, "/discard"), body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.DiscardResult
	if err := json.Unmarshal(respBody, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Success || got.Discarded != 1 || got.Remaining != 2 {
		t.Errorf("result = %+v, want success=true discarded=1 remaining=2", got)
	}
}

func TestDiscardWithFilePaths(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		DiscardFn: func(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error) {
			if len(req.FilePaths) != 1 || req.FilePaths[0] != "path/to/file.txt" {
				t.Errorf("FilePaths = %v, want [path/to/file.txt]", req.FilePaths)
			}
			return &types.DiscardResult{Success: true, Discarded: 1, Files: req.FilePaths}, nil
		},
	}
	live := newLive(t, svc)
	resp, _ := live.DoJSON(t, "POST", sandboxesPath(testID, "/discard"),
		`{"filePaths": ["path/to/file.txt"]}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestDiscardMissingFiles(t *testing.T) {
	testID := uuid.New()
	live := newLive(t, &sandboxiface.FakeService{})
	resp, _ := live.DoJSON(t, "POST", sandboxesPath(testID, "/discard"), `{"actor": "test-user"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestDiscardInvalidSandboxID(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{})
	resp, _ := live.DoJSON(t, "POST", "/api/v1/sandboxes/invalid-uuid/discard",
		`{"filePaths": ["test.txt"]}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// --- GetWorkspace Handler Tests ---

func TestGetWorkspaceSuccess(t *testing.T) {
	testID := uuid.New()
	expectedPath := "/tmp/sandbox/" + testID.String() + "/merged"
	svc := &sandboxiface.FakeService{
		GetWorkspacePathFn: func(ctx context.Context, id uuid.UUID) (string, error) {
			return expectedPath, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "GET", sandboxesPath(testID, "/workspace"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got map[string]string
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["path"] != expectedPath {
		t.Errorf("path = %q, want %q", got["path"], expectedPath)
	}
}

// --- DriverInfo Handler Tests ---

func TestDriverInfoAvailable(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{})
	resp, body := live.Do(t, "GET", "/api/v1/driver/info", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["available"] != true {
		t.Errorf("available = %v, want true", got["available"])
	}
}

func TestDriverInfoUnavailable(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{},
		withDriver(driverWithErr(false, fmt.Errorf("overlayfs not supported"))))
	resp, body := live.Do(t, "GET", "/api/v1/driver/info", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["available"] != false {
		t.Errorf("available = %v, want false", got["available"])
	}
}

// --- StartSandbox Handler Tests ---

func TestStartSandboxSuccess(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		StartFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return &types.Sandbox{
				ID: testID, Status: types.StatusActive,
				MergedDir: "/tmp/sandbox/" + testID.String() + "/merged",
			}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "POST", sandboxesPath(testID, "/start"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.Sandbox
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != types.StatusActive {
		t.Errorf("status = %v, want %v", got.Status, types.StatusActive)
	}
}

func TestStartSandboxNotFound(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		StartFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return nil, types.NewNotFoundError(id.String())
		},
	}
	live := newLive(t, svc)
	resp, _ := live.Do(t, "POST", sandboxesPath(testID, "/start"), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestStartSandboxInvalidState(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		StartFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return nil, types.NewStateError(&types.InvalidTransitionError{
				Current: types.StatusApproved, Attempted: types.StatusActive,
				Reason: "cannot start approved sandbox",
			})
		},
	}
	live := newLive(t, svc)
	resp, _ := live.Do(t, "POST", sandboxesPath(testID, "/start"), nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", resp.StatusCode)
	}
}

// --- CheckConflicts Handler Tests ---

func TestCheckConflictsSuccess(t *testing.T) {
	testID := uuid.New()
	now := time.Now()
	svc := &sandboxiface.FakeService{
		CheckConflictsFn: func(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error) {
			return &types.ConflictCheckResponse{
				HasConflict: false, BaseCommitHash: "abc123", CurrentHash: "abc123",
				RepoChangedFiles: []string{}, SandboxChangedFiles: []string{"file.txt"},
				ConflictingFiles: []string{}, CheckedAt: now,
			}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "GET", sandboxesPath(testID, "/conflicts"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.ConflictCheckResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.HasConflict {
		t.Error("HasConflict = true, want false")
	}
	if got.BaseCommitHash != "abc123" {
		t.Errorf("BaseCommitHash = %q, want abc123", got.BaseCommitHash)
	}
}

func TestCheckConflictsWithConflict(t *testing.T) {
	testID := uuid.New()
	now := time.Now()
	svc := &sandboxiface.FakeService{
		CheckConflictsFn: func(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error) {
			return &types.ConflictCheckResponse{
				HasConflict: true, BaseCommitHash: "abc123", CurrentHash: "def456",
				RepoChangedFiles:    []string{"file.txt", "config.go"},
				SandboxChangedFiles: []string{"file.txt", "main.go"},
				ConflictingFiles:    []string{"file.txt"},
				CheckedAt:           now,
			}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "GET", sandboxesPath(testID, "/conflicts"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.ConflictCheckResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.HasConflict {
		t.Error("HasConflict = false, want true")
	}
	if len(got.ConflictingFiles) != 1 || got.ConflictingFiles[0] != "file.txt" {
		t.Errorf("ConflictingFiles = %v, want [file.txt]", got.ConflictingFiles)
	}
}

func TestCheckConflictsNotFound(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		CheckConflictsFn: func(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error) {
			return nil, types.NewNotFoundError(id.String())
		},
	}
	live := newLive(t, svc)
	resp, _ := live.Do(t, "GET", sandboxesPath(testID, "/conflicts"), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// --- Rebase Handler Tests ---

func TestRebaseSuccess(t *testing.T) {
	testID := uuid.New()
	now := time.Now()
	svc := &sandboxiface.FakeService{
		RebaseFn: func(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error) {
			return &types.RebaseResult{
				Success: true, PreviousBaseHash: "abc123", NewBaseHash: "def456",
				ConflictingFiles: []string{}, RepoChangedFiles: []string{"file.txt"},
				Strategy: types.RebaseStrategyRegenerate, RebasedAt: now,
			}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.DoJSON(t, "POST", sandboxesPath(testID, "/rebase"),
		`{"strategy": "regenerate", "actor": "test-user"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.RebaseResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Success || got.PreviousBaseHash != "abc123" || got.NewBaseHash != "def456" {
		t.Errorf("got %+v, want success=true prev=abc123 new=def456", got)
	}
}

func TestRebaseWithConflicts(t *testing.T) {
	testID := uuid.New()
	now := time.Now()
	svc := &sandboxiface.FakeService{
		RebaseFn: func(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error) {
			return &types.RebaseResult{
				Success: false, PreviousBaseHash: "abc123",
				ConflictingFiles: []string{"main.go", "config.go"},
				RepoChangedFiles: []string{"main.go", "config.go", "readme.md"},
				Strategy:         types.RebaseStrategyRegenerate,
				ErrorMsg:         "cannot automatically resolve conflicts",
				RebasedAt:        now,
			}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.DoJSON(t, "POST", sandboxesPath(testID, "/rebase"), `{"strategy": "regenerate"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.RebaseResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Success {
		t.Error("Success = true, want false")
	}
	if len(got.ConflictingFiles) != 2 {
		t.Errorf("ConflictingFiles count = %d, want 2", len(got.ConflictingFiles))
	}
}

func TestRebaseNotFound(t *testing.T) {
	testID := uuid.New()
	svc := &sandboxiface.FakeService{
		RebaseFn: func(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error) {
			return nil, types.NewNotFoundError(req.SandboxID.String())
		},
	}
	live := newLive(t, svc)
	resp, _ := live.DoJSON(t, "POST", sandboxesPath(testID, "/rebase"), `{"strategy": "regenerate"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// --- ValidatePath Handler Tests ---

func TestValidatePathSuccess(t *testing.T) {
	svc := &sandboxiface.FakeService{
		ValidatePathFn: func(ctx context.Context, path, projectRoot string) (*types.PathValidationResult, error) {
			return &types.PathValidationResult{
				Path: path, ProjectRoot: projectRoot,
				Valid: true, Exists: true, IsDirectory: true, WithinProjectRoot: true,
			}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "GET", "/api/v1/validate-path?path=/project/src&projectRoot=/project", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.PathValidationResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Valid || !got.WithinProjectRoot {
		t.Errorf("got valid=%v within=%v, want both true", got.Valid, got.WithinProjectRoot)
	}
}

func TestValidatePathOutsideProject(t *testing.T) {
	svc := &sandboxiface.FakeService{
		ValidatePathFn: func(ctx context.Context, path, projectRoot string) (*types.PathValidationResult, error) {
			return &types.PathValidationResult{
				Path: path, ProjectRoot: projectRoot,
				Valid: false, Exists: true, IsDirectory: true, WithinProjectRoot: false,
				Error: "path is outside project root",
			}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.Do(t, "GET", "/api/v1/validate-path?path=/etc/passwd&projectRoot=/project", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got types.PathValidationResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Valid {
		t.Error("Valid = true, want false")
	}
	if got.WithinProjectRoot {
		t.Error("WithinProjectRoot = true, want false")
	}
	if got.Error == "" {
		t.Error("Error should not be empty")
	}
}

func TestValidatePathMissingParam(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{})
	resp, _ := live.Do(t, "GET", "/api/v1/validate-path", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// --- ApplyAtRunEnd Handler Tests ---
//
// Cover the auditability-contract endpoint added in Phase 2 of
// execute/agent-manager-sandbox-auto-apply-defaults. The handler is a thin
// JSON-decode + service-delegation layer; tests verify the wire-shape
// translation and that domain errors surface as HTTP errors.

func TestApplyAtRunEnd_Success(t *testing.T) {
	testID := uuid.New()
	var captured *types.ApplyAtRunEndRequest
	svc := &sandboxiface.FakeService{
		ApplyAtRunEndFn: func(ctx context.Context, req *types.ApplyAtRunEndRequest) (*types.ApprovalResult, error) {
			captured = req
			return &types.ApprovalResult{Success: true, Applied: 2}, nil
		},
	}
	live := newLive(t, svc)
	body := `{
		"agentManagerRunId": "run-1",
		"conversationId": "conv-7",
		"cost": 0.5,
		"runOutcome": "success",
		"source": "agent-manager-auto-apply",
		"actor": "auto-apply"
	}`
	resp, respBody := live.DoJSON(t, "POST", sandboxesPath(testID, "/apply-at-run-end"), body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, respBody)
	}
	if captured == nil {
		t.Fatal("service was not called")
	}
	if captured.SandboxID != testID {
		t.Errorf("SandboxID = %v, want %v", captured.SandboxID, testID)
	}
	if captured.AgentManagerRunID != "run-1" {
		t.Errorf("AgentManagerRunID = %q, want run-1", captured.AgentManagerRunID)
	}
	if captured.Source != types.SourceAgentManagerAutoApply {
		t.Errorf("Source = %q, want %q", captured.Source, types.SourceAgentManagerAutoApply)
	}
	if captured.RunOutcome != "success" {
		t.Errorf("RunOutcome = %q, want success", captured.RunOutcome)
	}
}

func TestApplyAtRunEnd_InvalidSandboxID(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{})
	resp, _ := live.DoJSON(t, "POST", "/api/v1/sandboxes/not-a-uuid/apply-at-run-end", `{}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestApplyAtRunEnd_MalformedBody(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{})
	id := uuid.New()
	resp, _ := live.DoJSON(t, "POST", sandboxesPath(id, "/apply-at-run-end"), `{ malformed`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}
