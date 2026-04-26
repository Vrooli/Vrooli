package sandbox_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/sandbox"

	"github.com/google/uuid"
)

// [REQ:REQ-P0-005] Tests for sandbox integration with workspace-sandbox

func TestWorkspaceSandboxProvider_Create(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/sandboxes" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		// Verify content type
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json content type, got %s", ct)
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          sandboxID.String(),
			"scopePath":   "src/",
			"projectRoot": "/project",
			"status":      "active",
			"mergedDir":   "/tmp/sandbox/" + sandboxID.String(),
			"createdAt":   time.Now().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)

	result, err := provider.Create(context.Background(), sandbox.CreateRequest{
		ScopePath:   "src/",
		ProjectRoot: "/project",
		Owner:       "test-run",
		OwnerType:   "run",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if result.ID != sandboxID {
		t.Errorf("expected ID %s, got %s", sandboxID, result.ID)
	}
	if result.ScopePath != "src/" {
		t.Errorf("expected scopePath 'src/', got '%s'", result.ScopePath)
	}
	if result.Status != sandbox.SandboxStatusActive {
		t.Errorf("expected status 'active', got '%s'", result.Status)
	}
}

func TestWorkspaceSandboxProvider_Get(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/sandboxes/" + sandboxID.String()
		if r.Method != "GET" || r.URL.Path != expectedPath {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          sandboxID.String(),
			"scopePath":   "src/",
			"projectRoot": "/project",
			"status":      "active",
			"mergedDir":   "/tmp/sandbox/" + sandboxID.String(),
			"createdAt":   time.Now().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)

	result, err := provider.Get(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if result.ID != sandboxID {
		t.Errorf("expected ID %s, got %s", sandboxID, result.ID)
	}
}

func TestWorkspaceSandboxProvider_Get_NotFound(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)

	_, err := provider.Get(context.Background(), sandboxID)

	if err == nil {
		t.Error("expected error for not found sandbox")
	}
}

func TestWorkspaceSandboxProvider_Delete(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/sandboxes/" + sandboxID.String()
		if r.Method != "DELETE" || r.URL.Path != expectedPath {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)

	err := provider.Delete(context.Background(), sandboxID)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}

// TestWorkspaceSandboxProvider_GetDiff is a shape-parity contract test for the
// workspace-sandbox /diff endpoint. The fake server below MUST emit the exact
// JSON shape that scenarios/workspace-sandbox/api/internal/types.DiffResult
// emits (see types.go and diff.go in that scenario). If workspace-sandbox
// changes the wire shape, this test must change in the same commit — otherwise
// agent-manager will silently decode into zero values and auto-approval logic
// will misbehave (see docs/plans/sandbox-auto-approve-and-profile-reconcile-plan.md §10.2).
func TestWorkspaceSandboxProvider_GetDiff(t *testing.T) {
	sandboxID := uuid.New()
	fileID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/sandboxes/" + sandboxID.String() + "/diff"
		if r.Method != "GET" || r.URL.Path != expectedPath {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sandboxId": sandboxID.String(),
			"files": []map[string]interface{}{
				{
					"id":           fileID.String(),
					"filePath":     "src/main.go",
					"changeType":   "modified",
					"fileSize":     1024,
					"linesAdded":   10,
					"linesRemoved": 5,
				},
			},
			"unifiedDiff": "diff --git a/tmp b/tmp\nnew file mode 040755\n--- /dev/null\n+++ b/tmp\n\ndiff --git a/src/main.go b/src/main.go\n--- a/src/main.go\n+++ b/src/main.go\n@@ -1,5 +1,10 @@\n+// Added line\n",
			"generated":   "2026-04-24T20:00:00Z",
			"stats": map[string]interface{}{
				"filesChanged":  1,
				"filesAdded":    0,
				"filesModified": 1,
				"filesDeleted":  0,
				"linesAdded":    10,
				"linesRemoved":  5,
				"totalBytes":    int64(1024),
			},
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)

	result, err := provider.GetDiff(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}

	if len(result.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(result.Files))
	}
	if result.Stats.FilesChanged != 1 {
		t.Errorf("expected 1 file changed, got %d", result.Stats.FilesChanged)
	}
	if result.Stats.FilesModified != 1 {
		t.Errorf("expected 1 file modified, got %d", result.Stats.FilesModified)
	}
	if result.Stats.LinesAdded != 10 {
		t.Errorf("expected 10 lines added, got %d", result.Stats.LinesAdded)
	}
	if result.Stats.LinesRemoved != 5 {
		t.Errorf("expected 5 lines removed, got %d", result.Stats.LinesRemoved)
	}
	if result.Stats.TotalBytes != 1024 {
		t.Errorf("expected totalBytes=1024, got %d", result.Stats.TotalBytes)
	}
	if strings.Contains(result.UnifiedDiff, "diff --git a/tmp b/tmp") {
		t.Errorf("expected directory-only diff entry to be filtered out")
	}
}

// TestWorkspaceSandboxProvider_GetDiff_Empty verifies that an empty sandbox
// decodes to FilesChanged=0. This is the happy path for auto-approve-if-empty.
func TestWorkspaceSandboxProvider_GetDiff_Empty(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sandboxId":   sandboxID.String(),
			"files":       []map[string]interface{}{},
			"unifiedDiff": "",
			"generated":   "2026-04-24T20:00:00Z",
			"stats": map[string]interface{}{
				"filesChanged":  0,
				"filesAdded":    0,
				"filesModified": 0,
				"filesDeleted":  0,
				"linesAdded":    0,
				"linesRemoved":  0,
				"totalBytes":    int64(0),
			},
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)
	result, err := provider.GetDiff(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}
	if result.Stats.FilesChanged != 0 {
		t.Errorf("expected FilesChanged=0 for empty sandbox, got %d", result.Stats.FilesChanged)
	}
	if len(result.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(result.Files))
	}
}

func TestWorkspaceSandboxProvider_Approve(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/sandboxes/" + sandboxID.String() + "/approve"
		if r.Method != "POST" || r.URL.Path != expectedPath {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"applied":    3,
			"remaining":  0,
			"isPartial":  false,
			"commitHash": "abc123",
			"appliedAt":  time.Now().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)

	result, err := provider.Approve(context.Background(), sandbox.ApproveRequest{
		SandboxID: sandboxID,
		Actor:     "test-user",
		CommitMsg: "Apply changes",
	})
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.Applied != 3 {
		t.Errorf("expected 3 applied, got %d", result.Applied)
	}
	if result.CommitHash != "abc123" {
		t.Errorf("expected commit hash 'abc123', got '%s'", result.CommitHash)
	}
}

func TestWorkspaceSandboxProvider_Reject(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/sandboxes/" + sandboxID.String() + "/reject"
		if r.Method != "POST" || r.URL.Path != expectedPath {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)

	err := provider.Reject(context.Background(), sandboxID, "test-user")
	if err != nil {
		t.Errorf("Reject failed: %v", err)
	}
}

func TestWorkspaceSandboxProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name      string
		response  func(w http.ResponseWriter)
		wantAvail bool
	}{
		{
			name: "healthy",
			response: func(w http.ResponseWriter) {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"status":    "healthy",
					"readiness": true,
				})
			},
			wantAvail: true,
		},
		{
			name: "not ready",
			response: func(w http.ResponseWriter) {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"status":    "starting",
					"readiness": false,
				})
			},
			wantAvail: false,
		},
		{
			name: "server error",
			response: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantAvail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/health" {
					t.Errorf("expected /health path, got %s", r.URL.Path)
				}
				tt.response(w)
			}))
			defer server.Close()

			provider := sandbox.NewWorkspaceSandboxProvider(server.URL)
			avail, _ := provider.IsAvailable(context.Background())

			if avail != tt.wantAvail {
				t.Errorf("expected availability %v, got %v", tt.wantAvail, avail)
			}
		})
	}
}

func TestWorkspaceSandboxProvider_PartialApprove(t *testing.T) {
	sandboxID := uuid.New()
	fileID1 := uuid.New()
	fileID2 := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/sandboxes/" + sandboxID.String() + "/partial-approve"
		if r.Method != "POST" || r.URL.Path != expectedPath {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"applied":    2,
			"remaining":  1,
			"isPartial":  true,
			"commitHash": "def456",
			"appliedAt":  time.Now().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)

	result, err := provider.PartialApprove(context.Background(), sandbox.PartialApproveRequest{
		SandboxID: sandboxID,
		Actor:     "test-user",
		FileIDs:   []uuid.UUID{fileID1, fileID2},
		CommitMsg: "Apply selected files",
	})
	if err != nil {
		t.Fatalf("PartialApprove failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
	if !result.IsPartial {
		t.Error("expected isPartial to be true")
	}
	if result.Applied != 2 {
		t.Errorf("expected 2 applied, got %d", result.Applied)
	}
	if result.Remaining != 1 {
		t.Errorf("expected 1 remaining, got %d", result.Remaining)
	}
}
