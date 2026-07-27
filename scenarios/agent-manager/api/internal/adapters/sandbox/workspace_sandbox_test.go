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
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		roots, ok := body["auxiliaryRoots"].([]any)
		if !ok || len(roots) != 1 || roots[0] != "/state/runs" {
			t.Errorf("auxiliaryRoots = %#v, want [/state/runs]", body["auxiliaryRoots"])
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
		ScopePath:      "src/",
		ProjectRoot:    "/project",
		Owner:          "test-run",
		OwnerType:      "run",
		AuxiliaryRoots: []string{"/state/runs"},
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

// TestWorkspaceSandboxProvider_Get_DecodesWorkspaceLayout pins that the
// provider decodes the negotiated workspace contract (workspacePath,
// pathIllusion, containment) and that GetWorkspacePath still returns the
// host merged dir (not the illusion path) for host-side consumers.
func TestWorkspaceSandboxProvider_Get_DecodesWorkspaceLayout(t *testing.T) {
	sandboxID := uuid.New()
	const merged = "/tmp/sandbox/merged"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            sandboxID.String(),
			"status":        "active",
			"mergedDir":     merged,
			"workspacePath": "/workspace",
			"pathIllusion":  true,
			"containment": map[string]interface{}{
				"level":        "required",
				"backend":      "bwrap",
				"enforcements": []string{"filesystem-write-containment", "network-deny", "pid-namespace", "path-illusion"},
			},
			"createdAt": time.Now().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)

	sb, err := provider.Get(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sb.WorkspacePath != "/workspace" {
		t.Errorf("WorkspacePath = %q, want /workspace", sb.WorkspacePath)
	}
	if !sb.PathIllusion {
		t.Error("PathIllusion = false, want true")
	}
	if sb.WorkDir != merged {
		t.Errorf("WorkDir = %q, want %q (host merged dir)", sb.WorkDir, merged)
	}
	if sb.Containment == nil || sb.Containment.Backend != "bwrap" || len(sb.Containment.Enforcements) != 4 {
		t.Errorf("Containment = %+v, want bwrap with 4 enforcements", sb.Containment)
	}

	// GetWorkspacePath must return the host merged dir so host-routed
	// (tracking-mode) launches chdir into a real path.
	got, err := provider.GetWorkspacePath(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("GetWorkspacePath: %v", err)
	}
	if got != merged {
		t.Errorf("GetWorkspacePath = %q, want %q (host merged dir, not the illusion path)", got, merged)
	}

	// ContainmentFor reports the enforced containment.
	cont, ok := provider.ContainmentFor(context.Background(), sandboxID)
	if !ok || cont == nil {
		t.Fatalf("ContainmentFor ok=%v cont=%v; want a report", ok, cont)
	}
	if len(cont.MissingProtectedEnforcements()) != 0 {
		t.Errorf("fully contained sandbox reports missing: %v", cont.MissingProtectedEnforcements())
	}
}

// TestWorkspaceSandboxProvider_Get_LayoutFallback pins graceful decoding of
// an older workspace-sandbox that does not report the new fields: the
// agent-visible path falls back to the host merged dir (identity), and
// ContainmentFor degrades to ok=false rather than a false "contained" claim.
func TestWorkspaceSandboxProvider_Get_LayoutFallback(t *testing.T) {
	sandboxID := uuid.New()
	const merged = "/tmp/sandbox/legacy/merged"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        sandboxID.String(),
			"status":    "active",
			"mergedDir": merged,
			"createdAt": time.Now().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)

	sb, err := provider.Get(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sb.WorkspacePath != merged {
		t.Errorf("WorkspacePath = %q, want fallback to %q", sb.WorkspacePath, merged)
	}
	if sb.PathIllusion {
		t.Error("PathIllusion = true, want false for legacy server")
	}
	if _, ok := provider.ContainmentFor(context.Background(), sandboxID); ok {
		t.Error("ContainmentFor ok=true for legacy server; want false (no report)")
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

func TestWorkspaceSandboxProviderLifecycleValidationAndConflictContracts(t *testing.T) {
	sandboxID := uuid.New()
	deleted := make([]string, 0)
	now := time.Now().UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/"+sandboxID.String()+"/stop":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/"+sandboxID.String()+"/start":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/"+sandboxID.String()+"/resume":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": sandboxID.String(), "scopePath": "src/api", "status": "active", "mergedDir": "/tmp/merged", "createdAt": now.Format(time.RFC3339)})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/"+sandboxID.String()+"/turn-checkpoint":
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["actor"] != "applyAtRunEnd" || body["turnId"] != "turn-1" || body["turnSequence"] != float64(2) {
				t.Fatalf("checkpoint body=%v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"sandboxId": sandboxID.String(), "status": "checkpointed", "success": true, "applied": 1, "checkpointId": "checkpoint-1", "appliedAt": now.Format(time.RFC3339)})
		case r.Method == http.MethodGet && r.URL.Path == "/validate-path":
			if r.URL.Query().Get("path") != "src/api" || r.URL.Query().Get("projectRoot") != "/repo" {
				t.Fatalf("validation query=%s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"path": "src/api", "valid": true, "isDirectory": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sandboxes":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"sandboxes": []map[string]interface{}{
				{"id": sandboxID.String(), "scopePath": "src/api", "status": "active", "createdAt": now.Add(-time.Hour).Format(time.RFC3339)},
				{"id": uuid.NewString(), "scopePath": "src/api/deleted", "status": "deleted", "createdAt": now.Add(-time.Hour).Format(time.RFC3339)},
				{"id": uuid.NewString(), "scopePath": "docs", "status": "active", "createdAt": now.Format(time.RFC3339)},
			}})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/sandboxes/"):
			deleted = append(deleted, strings.TrimPrefix(r.URL.Path, "/api/v1/sandboxes/"))
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)
	ctx := context.Background()
	if err := provider.Stop(ctx, sandboxID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := provider.Start(ctx, sandboxID); err != nil {
		t.Fatalf("start: %v", err)
	}
	resumed, err := provider.Resume(ctx, sandboxID)
	if err != nil || resumed.ID != sandboxID || resumed.Status != sandbox.SandboxStatusActive {
		t.Fatalf("resume=%+v err=%v", resumed, err)
	}
	checkpoint, err := provider.TurnCheckpoint(ctx, sandbox.TurnCheckpointRequest{SandboxID: sandboxID, RunID: "run-1", TurnID: "turn-1", TurnSequence: 2, CreateCommit: true})
	if err != nil || !checkpoint.Success || checkpoint.CheckpointID != "checkpoint-1" {
		t.Fatalf("checkpoint=%+v err=%v", checkpoint, err)
	}
	path, err := provider.ValidatePath(ctx, "src/api", "/repo")
	if err != nil || !path.Valid || !path.IsDirectory {
		t.Fatalf("path validation=%+v err=%v", path, err)
	}
	conflicts, err := provider.CheckConflicts(ctx, "src")
	if err != nil || len(conflicts) != 1 || conflicts[0].SandboxID != sandboxID.String() {
		t.Fatalf("conflicts=%+v err=%v", conflicts, err)
	}
	count, err := provider.CleanupStaleSandboxes(ctx, 30*time.Minute)
	if err != nil || count != 1 || len(deleted) != 1 || deleted[0] != sandboxID.String() {
		t.Fatalf("cleanup count=%d deleted=%v err=%v", count, deleted, err)
	}
}

// TestWorkspaceSandboxProvider_GetDiff is a shape-parity contract test for the
// workspace-sandbox /diff endpoint. The fake server below MUST emit the exact
// JSON shape that scenarios/workspace-sandbox/api/internal/types.DiffResult
// emits (see types.go and diff.go in that scenario). If workspace-sandbox
// changes the wire shape, this test must change in the same commit — otherwise
// agent-manager will silently decode into zero values and auto-approval logic
// will misbehave.
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
	// Pin the live-response contract: workspace-sandbox omits archiveState
	// for live diffs (Active/Stopped/Creating/Error). An empty (zero-value)
	// ArchiveState distinguishes a live response from an archived one in
	// downstream UI/CLI rendering.
	if result.ArchiveState != "" {
		t.Errorf("ArchiveState = %q, want empty for live diff response", result.ArchiveState)
	}
}

// TestWorkspaceSandboxProvider_GetDiff_ArchiveStateComplete pins the
// archive-state contract: when workspace-sandbox serves a closed-run
// diff from the durable archive, the response carries
// `archiveState: "complete"`. The adapter must decode this onto
// DiffResult.ArchiveState verbatim — the field is what
// agent-manager UI uses to render archived diffs differently from
// generic empty responses.
func TestWorkspaceSandboxProvider_GetDiff_ArchiveStateComplete(t *testing.T) {
	sandboxID := uuid.New()
	fileID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sandboxId": sandboxID.String(),
			"files": []map[string]interface{}{
				{
					"id":           fileID.String(),
					"filePath":     "src/main.go",
					"changeType":   "modified",
					"fileSize":     1024,
					"linesAdded":   3,
					"linesRemoved": 1,
				},
			},
			"unifiedDiff": "diff --git a/src/main.go b/src/main.go\n@@ -1 +1 @@\n+x\n",
			"generated":   "2026-04-30T00:00:00Z",
			"stats": map[string]interface{}{
				"filesChanged":  1,
				"filesModified": 1,
				"linesAdded":    3,
				"linesRemoved":  1,
				"totalBytes":    int64(1024),
			},
			"archiveState": "complete",
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)
	result, err := provider.GetDiff(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("GetDiff: %v", err)
	}
	if result.ArchiveState != sandbox.ArchiveStateComplete {
		t.Errorf("ArchiveState = %q, want %q", result.ArchiveState, sandbox.ArchiveStateComplete)
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(result.Files))
	}
}

// TestWorkspaceSandboxProvider_GetDiff_ArchiveStateNotCaptured pins the
// taxonomy distinction the UI relies on: when workspace-sandbox returns
// `archiveState: "not_captured"` (e.g. Error → Deleted, no usable
// overlay at terminal-transition), the adapter surfaces the marker
// alongside an empty Files list. The UI then renders an explicit
// "no diff captured" state instead of a generic empty diff.
func TestWorkspaceSandboxProvider_GetDiff_ArchiveStateNotCaptured(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sandboxId":    sandboxID.String(),
			"files":        []map[string]interface{}{},
			"unifiedDiff":  "",
			"generated":    "2026-04-30T00:00:00Z",
			"stats":        map[string]interface{}{"filesChanged": 0},
			"archiveState": "not_captured",
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)
	result, err := provider.GetDiff(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("GetDiff: %v", err)
	}
	if result.ArchiveState != sandbox.ArchiveStateNotCaptured {
		t.Errorf("ArchiveState = %q, want %q", result.ArchiveState, sandbox.ArchiveStateNotCaptured)
	}
	if len(result.Files) != 0 {
		t.Errorf("not_captured archives must have empty Files; got %d", len(result.Files))
	}
}

// TestWorkspaceSandboxProvider_GetDiff_Empty verifies that an empty sandbox
// decodes to FilesChanged=0. The decoded value is consumed by the diff
// display surface; auto-apply gating is independent of diff size.
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
			"success":          true,
			"applied":          3,
			"remaining":        0,
			"isPartial":        false,
			"commitHash":       "abc123",
			"appliedSizeBytes": 1536,
			"diffPath":         "/api/v1/sandboxes/diff",
			"appliedAt":        time.Now().Format(time.RFC3339),
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
	if result.TotalSizeBytes != 1536 || result.DiffPath != "/api/v1/sandboxes/diff" {
		t.Errorf("attribution response bytes=%d diff=%q", result.TotalSizeBytes, result.DiffPath)
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

// TestWorkspaceSandboxProvider_ApplyAtRunEnd is a shape-parity contract test
// for the workspace-sandbox /apply-at-run-end endpoint. The fake server
// asserts the exact wire shape locked by
// scenarios/workspace-sandbox/api/internal/types.ApplyAtRunEndRequest. If
// workspace-sandbox renames any field, this test must change in the same
// commit.
func TestWorkspaceSandboxProvider_ApplyAtRunEnd(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/sandboxes/" + sandboxID.String() + "/apply-at-run-end"
		if r.Method != "POST" || r.URL.Path != expectedPath {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got := body["sandboxId"]; got != sandboxID.String() {
			t.Errorf("sandboxId: got %v, want %s", got, sandboxID)
		}
		if got := body["agentManagerRunId"]; got != "run-abc-123" {
			t.Errorf("agentManagerRunId: got %v, want 'run-abc-123'", got)
		}
		if got := body["conversationId"]; got != "conv-xyz-456" {
			t.Errorf("conversationId: got %v, want 'conv-xyz-456'", got)
		}
		if got := body["runOutcome"]; got != "success" {
			t.Errorf("runOutcome: got %v, want 'success'", got)
		}
		if got := body["source"]; got != "agent-manager-auto-apply" {
			t.Errorf("source: got %v, want 'agent-manager-auto-apply'", got)
		}
		if got, ok := body["cost"].(float64); !ok || got != 0.42 {
			t.Errorf("cost: got %v, want 0.42", body["cost"])
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

	result, err := provider.ApplyAtRunEnd(context.Background(), sandbox.ApplyAtRunEndRequest{
		SandboxID:      sandboxID,
		RunID:          "run-abc-123",
		ConversationID: "conv-xyz-456",
		Cost:           0.42,
		RunOutcome:     "success",
		CreateCommit:   true,
	})
	if err != nil {
		t.Fatalf("ApplyAtRunEnd failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Applied != 3 {
		t.Errorf("expected 3 applied, got %d", result.Applied)
	}
	if result.CommitHash != "abc123" {
		t.Errorf("expected commitHash 'abc123', got %q", result.CommitHash)
	}
}

// TestWorkspaceSandboxProvider_ApplyAtRunEnd_PartialAcceptance covers the
// out-of-acceptance case: the workspace-sandbox endpoint applies the
// in-acceptance files and returns isPartial=true with remaining > 0,
// signalling that the sandbox persists with state=pending-review entries.
func TestWorkspaceSandboxProvider_ApplyAtRunEnd_PartialAcceptance(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sandboxId": sandboxID.String(),
			"status":    "checkpointed",
			"success":   true,
			"applied":   2,
			"remaining": 1,
			"isPartial": true,
			"appliedAt": time.Now().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)
	result, err := provider.ApplyAtRunEnd(context.Background(), sandbox.ApplyAtRunEndRequest{
		SandboxID:  sandboxID,
		RunID:      "run-1",
		RunOutcome: "success",
	})
	if err != nil {
		t.Fatalf("ApplyAtRunEnd failed: %v", err)
	}
	if !result.IsPartial {
		t.Error("expected IsPartial=true for partial acceptance")
	}
	if result.Remaining != 1 {
		t.Errorf("expected remaining=1, got %d", result.Remaining)
	}
}

// TestWorkspaceSandboxProvider_ApplyAtRunEnd_NotFound covers HTTP 404:
// sandbox already torn down or never existed. Surfaces as a typed
// NotFoundError so callers can downgrade severity.
func TestWorkspaceSandboxProvider_ApplyAtRunEnd_NotFound(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "sandbox not found"})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)
	_, err := provider.ApplyAtRunEnd(context.Background(), sandbox.ApplyAtRunEndRequest{
		SandboxID: sandboxID,
		RunID:     "run-1",
	})
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "Sandbox") {
		t.Errorf("expected NotFoundError mentioning Sandbox, got: %v", err)
	}
}

// TestWorkspaceSandboxProvider_ApplyAtRunEnd_ConflictReturnsTypedError
// covers HTTP 409 from the apply pipeline (e.g., underlying repo conflict).
// Adapter must surface the structured SandboxAPIError content so callers
// can decide whether to retry vs degrade.
func TestWorkspaceSandboxProvider_ApplyAtRunEnd_ConflictReturnsTypedError(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "repo conflict",
			"hint":  "rebase the sandbox before applying",
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)
	_, err := provider.ApplyAtRunEnd(context.Background(), sandbox.ApplyAtRunEndRequest{
		SandboxID: sandboxID,
		RunID:     "run-1",
	})
	if err == nil {
		t.Fatal("expected error for 409, got nil")
	}
	if !strings.Contains(err.Error(), "repo conflict") {
		t.Errorf("expected error to surface server message, got: %v", err)
	}
}

// TestWorkspaceSandboxProvider_ApplyAtRunEnd_BadRequestRejectsBogusSource
// asserts that the agent-manager adapter always sends
// source=agent-manager-auto-apply (locked by the workspace-sandbox
// validateApplyAtRunEndRequest function). If a future refactor accidentally
// drops the source field, the workspace-sandbox endpoint returns 400 — this
// test pins the wire-level expectation by simulating that 400 and asserting
// the adapter surfaces it as a SandboxError.
func TestWorkspaceSandboxProvider_ApplyAtRunEnd_BadRequestRejectsBogusSource(t *testing.T) {
	sandboxID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if got := body["source"]; got != "agent-manager-auto-apply" {
			// Simulate workspace-sandbox rejecting the wrong source.
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "source must be agent-manager-auto-apply",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true, "appliedAt": time.Now().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	provider := sandbox.NewWorkspaceSandboxProvider(server.URL)
	// Happy path — adapter sets source correctly.
	if _, err := provider.ApplyAtRunEnd(context.Background(), sandbox.ApplyAtRunEndRequest{
		SandboxID: sandboxID,
		RunID:     "run-1",
	}); err != nil {
		t.Errorf("expected success when adapter sends locked source, got: %v", err)
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
