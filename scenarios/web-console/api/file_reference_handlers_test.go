package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestSplitFileReferenceLine(t *testing.T) {
	path, line := splitFileReferenceLine("/tmp/file.ts:42")
	if path != "/tmp/file.ts" {
		t.Fatalf("expected path without line suffix, got %q", path)
	}
	if line == nil || *line != 42 {
		t.Fatalf("expected line 42, got %#v", line)
	}

	path, line = splitFileReferenceLine("docs/plan.md")
	if path != "docs/plan.md" {
		t.Fatalf("expected unchanged path, got %q", path)
	}
	if line != nil {
		t.Fatalf("expected nil line, got %#v", line)
	}
}

func TestHandleResolveFileReference_ProjectRootRelative(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_DEFAULT_CWD", root)
	filePath := filepath.Join(root, "docs", "plan.md")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("# plan\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fake := newFakePTYWithOutput()
	fake.currentDir = filepath.Join(root, "nested")
	if err := os.MkdirAll(fake.currentDir, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    NewSessionManagerWithFactory(fakePTYFactory(fake)),
		events:      NewEventLogger(100),
		metrics:     NewMetrics(),
		aiChain:     NewAIProviderChain(),
		shortcuts:   NewShortcutProfileStore(),
		aiConfig:    NewAIProviderConfigStore(),
		idempotency: newIdempotencyCache(),
		workspace:   NewMemWorkspaceStore(),
	}
	srv.conversations = NewConversationStore()
	srv.codexCheckpointStore = NewInMemoryCodexCheckpointStore()
	srv.ttsSummarization = NewTTSSummarizationService(srv.ttsSummarizer, srv.getTTSSummarizeConfig)
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+sess.ID+"/files/resolve", strings.NewReader(`{"path":"docs/plan.md:3"}`))
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleResolveFileReference(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp fileReferenceResolveResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ResolutionBasis != "project_root" {
		t.Fatalf("expected project_root resolution, got %q", resp.ResolutionBasis)
	}
	if resp.Line == nil || *resp.Line != 3 {
		t.Fatalf("expected line 3, got %#v", resp.Line)
	}
	if resp.Category != "markdown" || !resp.CanPreview {
		t.Fatalf("expected markdown previewable response, got category=%q preview=%v", resp.Category, resp.CanPreview)
	}
}

func TestHandleResolveFileReference_SessionCwdPreferred(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_DEFAULT_CWD", root)
	subdir := filepath.Join(root, "scenarios", "web-console")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	targetFile := filepath.Join(subdir, "notes.txt")
	if err := os.WriteFile(targetFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fake := newFakePTYWithOutput()
	fake.currentDir = subdir
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    NewSessionManagerWithFactory(fakePTYFactory(fake)),
		events:      NewEventLogger(100),
		metrics:     NewMetrics(),
		aiChain:     NewAIProviderChain(),
		shortcuts:   NewShortcutProfileStore(),
		aiConfig:    NewAIProviderConfigStore(),
		idempotency: newIdempotencyCache(),
		workspace:   NewMemWorkspaceStore(),
	}
	srv.conversations = NewConversationStore()
	srv.codexCheckpointStore = NewInMemoryCodexCheckpointStore()
	srv.ttsSummarization = NewTTSSummarizationService(srv.ttsSummarizer, srv.getTTSSummarizeConfig)

	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+sess.ID+"/files/resolve", strings.NewReader(`{"path":"notes.txt"}`))
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleResolveFileReference(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp fileReferenceResolveResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ResolutionBasis != "session_cwd" {
		t.Fatalf("expected session_cwd resolution, got %q", resp.ResolutionBasis)
	}
	if resp.ResolvedPath != filepath.Clean(targetFile) {
		t.Fatalf("expected resolved path %q, got %q", targetFile, resp.ResolvedPath)
	}
}

func TestHandleResolveFileReference_FileURL(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_DEFAULT_CWD", root)
	filePath := filepath.Join(root, "docs", "file-url.md")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("# file url\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	body := `{"path":"file://` + filePath + `:9"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+sess.ID+"/files/resolve", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleResolveFileReference(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp fileReferenceResolveResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ResolvedPath != filepath.Clean(filePath) {
		t.Fatalf("expected resolved path %q, got %q", filepath.Clean(filePath), resp.ResolvedPath)
	}
	if resp.Line == nil || *resp.Line != 9 {
		t.Fatalf("expected line 9, got %#v", resp.Line)
	}
}

func TestHandleResolveFileReference_RejectsOutsideAllowedRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_DEFAULT_CWD", root)

	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	outsideFile := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	body := `{"path":"` + outsideFile + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+sess.ID+"/files/resolve", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleResolveFileReference(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetFileReferenceContent_RejectsOversizedPreview(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_DEFAULT_CWD", root)
	filePath := filepath.Join(root, "large.txt")
	if err := os.WriteFile(filePath, []byte(strings.Repeat("x", int(maxFilePreviewBytes)+1)), 0o644); err != nil {
		t.Fatalf("write large file: %v", err)
	}

	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sess.ID+"/files/content?path=large.txt", nil)
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
	rec := httptest.NewRecorder()

	srv.handleGetFileReferenceContent(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetFileReferenceContent_ReturnsContent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_DEFAULT_CWD", root)
	filePath := filepath.Join(root, "src", "file.ts")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "export const x = 1;\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sess.ID+"/files/content?path=src/file.ts:1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
	rec := httptest.NewRecorder()

	srv.handleGetFileReferenceContent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp fileReferenceContentResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Content != content {
		t.Fatalf("expected content %q, got %q", content, resp.Content)
	}
	if resp.Category != "code" {
		t.Fatalf("expected code category, got %q", resp.Category)
	}
}
