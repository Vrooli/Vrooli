package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/identity"
	"swarm-manager/internal/testutil"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

// [REQ:SWM-P0-001] backlog work intake: direct item creation
func TestCreate_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":        "New Test Idea",
		"title":       "New Test Idea",
		"description": "A new test idea",
		"priority":    3,
		"tags":        []string{"new", "test"},
		"kind":        "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	result := resp.Item

	if result.Name != "new-test-idea" {
		t.Errorf("expected sanitized name 'new-test-idea', got '%s'", result.Name)
	}
	if result.Status != StatusBacklog {
		t.Errorf("expected status 'backlog', got '%s'", result.Status)
	}
	if result.Kind != KindIdea {
		t.Errorf("expected kind 'idea', got '%s'", result.Kind)
	}

	specPath := filepath.Join(rootDir, "ideas", "new-test-idea", "spec.json")
	testutil.AssertFileExists(t, specPath)
}

func TestCreate_StoresAgentProvenanceFromIdentityMiddleware(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":  "agent-created-item",
		"title": "Agent Created Item",
		"kind":  "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Identity-Token", "valid-token")
	w := httptest.NewRecorder()

	handler := provenance.Middleware(provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		if token != "valid-token" {
			t.Fatalf("token = %q, want valid-token", token)
		}
		return &cliutil.VerifyResult{
			Valid: true,
			Claims: &cliutil.VerifiedClaims{
				RunID:      "run-agent-1",
				TaskID:     "task-agent-1",
				ProfileKey: "swarm-manager/default",
			},
		}, nil
	}))(http.HandlerFunc(h.Create))

	handler.ServeHTTP(w, req)

	testutil.AssertStatusCreated(t, w)

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "agent-created-item", "spec.json"))
	if saved.CreatedBy == nil {
		t.Fatal("expected created_by provenance")
	}
	if saved.CreatedBy.Actor != identity.TypeAgent {
		t.Fatalf("created_by.actor = %q, want %q", saved.CreatedBy.Actor, identity.TypeAgent)
	}
	if saved.CreatedBy.RunID != "run-agent-1" || saved.CreatedBy.TaskID != "task-agent-1" || saved.CreatedBy.ProfileKey != "swarm-manager/default" {
		t.Fatalf("unexpected created_by provenance: %+v", saved.CreatedBy)
	}
}

func TestCreate_RejectsUnknownField(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/api/v1/backlog", strings.NewReader(`{
		"name": "new-test-idea",
		"title": "New Test Idea",
		"kind": "idea",
		"scope": "scenarios/swarm-manager"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "invalid request body") {
		t.Fatalf("expected invalid request body error, got: %s", w.Body.String())
	}

	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "new-test-idea", "spec.json"))
}

func TestCreate_RejectsSuggestedStatusField(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/api/v1/backlog", strings.NewReader(`{
		"name": "suggested-create",
		"title": "Suggested Create",
		"kind": "fix",
		"status": "suggested"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusBadRequest(t, w)
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "fix", "suggested-create", "spec.json"))
}

func TestCreate_MultipartWithFiles(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	req := newMultipartCreateRequest(t, map[string]any{
		"name":             "Broken Preview",
		"title":            "Broken Preview",
		"description":      "Preview crashes after load",
		"kind":             "fix",
		"acceptance_allow": []string{"scenarios/app-monitor/**"},
	}, map[string][]byte{
		"evidence/report.json":    []byte(`{"message":"broken"}`),
		"evidence/screenshot.png": []byte("png-data"),
		"evidence/console.json":   []byte(`[{"level":"error"}]`),
		"evidence/element-01.png": []byte("element"),
		"evidence/lifecycle.txt":  []byte("logs"),
		"evidence/network.json":   []byte(`[]`),
		"evidence/health.json":    []byte(`[]`),
		"evidence/status.txt":     []byte("running"),
	})
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusCreated(t, w)
	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Kind != KindFix {
		t.Fatalf("kind = %q, want %q", resp.Item.Kind, KindFix)
	}

	itemDir := filepath.Join(rootDir, "fix", "broken-preview")
	testutil.AssertFileExists(t, filepath.Join(itemDir, "spec.json"))
	for rel, want := range map[string]string{
		"evidence/report.json":    `{"message":"broken"}`,
		"evidence/screenshot.png": "png-data",
		"evidence/console.json":   `[{"level":"error"}]`,
		"evidence/element-01.png": "element",
		"evidence/lifecycle.txt":  "logs",
		"evidence/network.json":   `[]`,
		"evidence/health.json":    `[]`,
		"evidence/status.txt":     "running",
	} {
		got, err := os.ReadFile(filepath.Join(itemDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if string(got) != want {
			t.Fatalf("%s = %q, want %q", rel, got, want)
		}
	}
}

func TestCreate_MultipartRejectsUnsafeFilePathAndRollsBack(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	req := newMultipartCreateRequest(t, map[string]any{
		"name":  "Unsafe Evidence",
		"title": "Unsafe Evidence",
		"kind":  "fix",
	}, map[string][]byte{
		"../outside.txt": []byte("bad"),
	})
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusBadRequest(t, w)
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "fix", "unsafe-evidence"))
}

func TestCreate_MultipartRejectsUnlistedFileAndRollsBack(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	itemPayload, err := json.Marshal(map[string]any{
		"name":  "Unlisted Evidence",
		"title": "Unlisted Evidence",
		"kind":  "fix",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("item", string(itemPayload)); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("files_manifest", `{"files":[]}`); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("extra", "report.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("{}")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusBadRequest(t, w)
	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "fix", "unlisted-evidence"))
}

func TestCreate_RejectsUnsupportedContentType(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog", strings.NewReader("name=bad"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusUnsupportedMediaType, w.Body.String())
	}
}

func newMultipartCreateRequest(t *testing.T, item map[string]any, files map[string][]byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	itemPayload, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("item", string(itemPayload)); err != nil {
		t.Fatal(err)
	}

	manifest := map[string]any{"files": []map[string]string{}}
	entries := manifest["files"].([]map[string]string)
	index := 0
	for path, content := range files {
		field := fmt.Sprintf("file_%d", index)
		entries = append(entries, map[string]string{
			"field":        field,
			"path":         path,
			"content_type": contentTypeForPath(path),
		})
		part, err := writer.CreateFormFile(field, filepath.Base(path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatal(err)
		}
		index++
	}
	manifest["files"] = entries
	manifestPayload, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("files_manifest", string(manifestPayload)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func contentTypeForPath(path string) string {
	switch filepath.Ext(path) {
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	default:
		return "text/plain"
	}
}

// ---------------------------------------------------------------------------
// Auto-initialize workshop on Create
func TestCreate_WithEffort(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":     "effort-test",
		"title":    "Effort Test",
		"kind":     "idea",
		"effort":   "L",
		"priority": 3,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Effort != "L" {
		t.Errorf("expected effort 'L', got %q", resp.Item.Effort)
	}

	// Verify persisted to disk.
	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "effort-test", "spec.json"))
	if saved.Effort != "L" {
		t.Errorf("expected saved effort 'L', got %q", saved.Effort)
	}
}

func TestCreate_EffortNormalizesCase(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := map[string]any{
		"name":   "effort-case-test",
		"title":  "Effort Case Test",
		"kind":   "fix",
		"effort": "xl",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Effort != "XL" {
		t.Errorf("expected effort 'XL', got %q", resp.Item.Effort)
	}
}

func TestCreate_InvalidEffort(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := map[string]any{
		"name":   "bad-effort",
		"title":  "Bad Effort",
		"kind":   "idea",
		"effort": "HUGE",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusBadRequest(t, w)
}

func TestCreate_EffortOptional(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := map[string]any{
		"name":  "no-effort",
		"title": "No Effort",
		"kind":  "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Effort != "" {
		t.Errorf("expected empty effort, got %q", resp.Item.Effort)
	}
}

func TestCreate_WithAcceptanceGlobs(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":             "globs-test",
		"title":            "Globs Test",
		"kind":             "fix",
		"acceptance_allow": []string{"api/**", "*.go"},
		"acceptance_deny":  []string{"vendor/**"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if len(resp.Item.AcceptanceAllow) != 2 {
		t.Errorf("expected 2 allow globs, got %d", len(resp.Item.AcceptanceAllow))
	}
	if len(resp.Item.AcceptanceDeny) != 1 {
		t.Errorf("expected 1 deny glob, got %d", len(resp.Item.AcceptanceDeny))
	}

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "fix", "globs-test", "spec.json"))
	if len(saved.AcceptanceAllow) != 2 {
		t.Errorf("expected 2 saved allow globs, got %d", len(saved.AcceptanceAllow))
	}
}

func TestCreate_WithSpawnedFrom(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":         "spawned-item",
		"title":        "Spawned Item",
		"kind":         "execute",
		"spawned_from": "research/agent-identity-standard",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.SpawnedFrom != "research/agent-identity-standard" {
		t.Errorf("expected spawned_from 'research/agent-identity-standard', got %q", resp.Item.SpawnedFrom)
	}

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "execute", "spawned-item", "spec.json"))
	if saved.SpawnedFrom != "research/agent-identity-standard" {
		t.Errorf("expected saved spawned_from 'research/agent-identity-standard', got %q", saved.SpawnedFrom)
	}
}
