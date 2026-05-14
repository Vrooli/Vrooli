package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	intai "web-console/internal/ai"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"

	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"
	intsessions "web-console/internal/sessions"
	intworkspace "web-console/internal/workspace"
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

	// Trailing wrapper/punctuation after the line number is tolerated.
	path, line = splitFileReferenceLine("scenarios/api/main.go:42`")
	if path != "scenarios/api/main.go" {
		t.Fatalf("expected stripped path, got %q", path)
	}
	if line == nil || *line != 42 {
		t.Fatalf("expected line 42, got %#v", line)
	}
}

func TestNormalizeFileReferencePath_WrappedAndLineSuffix(t *testing.T) {
	path, line, err := normalizeFileReferencePath("`scenarios/web-console/api/main.go:42`")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if path != "scenarios/web-console/api/main.go" {
		t.Fatalf("expected stripped path, got %q", path)
	}
	if line == nil || *line != 42 {
		t.Fatalf("expected line 42, got %#v", line)
	}

	path, line, err = normalizeFileReferencePath(`"docs/plan.md"`)
	if err != nil {
		t.Fatalf("normalize quoted: %v", err)
	}
	if path != "docs/plan.md" {
		t.Fatalf("expected unquoted path, got %q", path)
	}
	if line != nil {
		t.Fatalf("expected nil line, got %#v", line)
	}
}

func resolveFileRef(t *testing.T, srv *Server, sessID, path string) *conversationv1.ResolveFileReferenceResponse {
	t.Helper()
	resp, err := newConversationConnectHandlerForServer(srv).ResolveFileReference(
		context.Background(),
		connect.NewRequest(&conversationv1.ResolveFileReferenceRequest{SessionId: sessID, Path: path}),
	)
	if err != nil {
		t.Fatalf("ResolveFileReference: %v", err)
	}
	return resp.Msg
}

func TestConnect_ResolveFileReference_ProjectRootRelative(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_DEFAULT_CWD", root)
	filePath := filepath.Join(root, "docs", "plan.md")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("# plan\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fake := ptyfake.NewFakePTYWithOutput()
	fake.CurrentDirVal = filepath.Join(root, "nested")
	if err := os.MkdirAll(fake.CurrentDirVal, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    newSessionManagerWithFactory(ptyfake.Factory(fake)),
		events:      events.NewLogger(100),
		metrics:     metrics.New(),
		aiChain:     intai.NewChain(),
		shortcuts:   NewShortcutProfileStore(),
		aiConfig:    intai.NewMemConfigStore(),
		idempotency: intsessions.NewIdempotencyCache(),
		workspace:   intworkspace.NewMemStore(),
	}
	srv.conversations = NewConversationStore()
	srv.codexCheckpointStore = NewInMemoryCodexCheckpointStore()
	srv.ttsSummarization = NewTTSSummarizationService(srv.ttsSummarizer, srv.getTTSSummarizeConfig)
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp := resolveFileRef(t, srv, sess.ID, "docs/plan.md:3")
	if resp.GetResolutionBasis() != "project_root" {
		t.Fatalf("expected project_root resolution, got %q", resp.GetResolutionBasis())
	}
	if !resp.GetHasLine() || resp.GetLine() != 3 {
		t.Fatalf("expected line 3, got has=%v val=%d", resp.GetHasLine(), resp.GetLine())
	}
	if resp.GetCategory() != "markdown" || !resp.GetCanPreview() {
		t.Fatalf("expected markdown previewable response, got category=%q preview=%v", resp.GetCategory(), resp.GetCanPreview())
	}
}

func TestConnect_ResolveFileReference_SessionCwdPreferred(t *testing.T) {
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

	fake := ptyfake.NewFakePTYWithOutput()
	fake.CurrentDirVal = subdir
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    newSessionManagerWithFactory(ptyfake.Factory(fake)),
		events:      events.NewLogger(100),
		metrics:     metrics.New(),
		aiChain:     intai.NewChain(),
		shortcuts:   NewShortcutProfileStore(),
		aiConfig:    intai.NewMemConfigStore(),
		idempotency: intsessions.NewIdempotencyCache(),
		workspace:   intworkspace.NewMemStore(),
	}
	srv.conversations = NewConversationStore()
	srv.codexCheckpointStore = NewInMemoryCodexCheckpointStore()
	srv.ttsSummarization = NewTTSSummarizationService(srv.ttsSummarizer, srv.getTTSSummarizeConfig)

	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp := resolveFileRef(t, srv, sess.ID, "notes.txt")
	if resp.GetResolutionBasis() != "session_cwd" {
		t.Fatalf("expected session_cwd resolution, got %q", resp.GetResolutionBasis())
	}
	if resp.GetResolvedPath() != filepath.Clean(targetFile) {
		t.Fatalf("expected resolved path %q, got %q", targetFile, resp.GetResolvedPath())
	}
}

func TestConnect_ResolveFileReference_FileURL(t *testing.T) {
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

	resp := resolveFileRef(t, srv, sess.ID, "file://"+filePath+":9")
	if resp.GetResolvedPath() != filepath.Clean(filePath) {
		t.Fatalf("expected resolved path %q, got %q", filepath.Clean(filePath), resp.GetResolvedPath())
	}
	if !resp.GetHasLine() || resp.GetLine() != 9 {
		t.Fatalf("expected line 9, got has=%v val=%d", resp.GetHasLine(), resp.GetLine())
	}
}

func TestConnect_ResolveFileReference_AllowsOutsideProjectRoot(t *testing.T) {
	// Plans live under ~/.vrooli outside the project; the resolver must
	// preview any file the process can read, not just project-root files.
	root := t.TempDir()
	t.Setenv("WC_DEFAULT_CWD", root)

	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	outsideFile := filepath.Join(t.TempDir(), "plan.md")
	if err := os.WriteFile(outsideFile, []byte("# plan"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	resp := resolveFileRef(t, srv, sess.ID, outsideFile)
	if !resp.Exists {
		t.Fatalf("expected outside-root file to resolve, got exists=false")
	}
	if resp.ResolvedPath != outsideFile {
		t.Fatalf("expected resolved path %q, got %q", outsideFile, resp.ResolvedPath)
	}
}

func TestConnect_ResolveFileReference_ExpandsTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home dir available")
	}
	relName := ".wc-file-ref-tilde-test.md"
	abs := filepath.Join(home, relName)
	if err := os.WriteFile(abs, []byte("# tilde"), 0o644); err != nil {
		t.Fatalf("write home file: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(abs) })

	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp := resolveFileRef(t, srv, sess.ID, "~/"+relName)
	if !resp.Exists {
		t.Fatalf("expected tilde-prefixed path to resolve, got exists=false")
	}
	if resp.ResolvedPath != abs {
		t.Fatalf("expected resolved path %q, got %q", abs, resp.ResolvedPath)
	}
}

func TestConnect_GetFileReferenceContent_RejectsOversizedPreview(t *testing.T) {
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

	_, err = newConversationConnectHandlerForServer(srv).GetFileReferenceContent(
		context.Background(),
		connect.NewRequest(&conversationv1.GetFileReferenceContentRequest{SessionId: sess.ID, Path: "large.txt"}),
	)
	if err == nil {
		t.Fatalf("expected error for oversize preview, got nil")
	}
	var ce *connect.Error
	if !errors.As(err, &ce) || ce.Code() != connect.CodeFailedPrecondition {
		t.Fatalf("expected CodeFailedPrecondition, got %v", err)
	}
}

func TestConnect_GetFileReferenceContent_ReturnsContent(t *testing.T) {
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

	resp, err := newConversationConnectHandlerForServer(srv).GetFileReferenceContent(
		context.Background(),
		connect.NewRequest(&conversationv1.GetFileReferenceContentRequest{SessionId: sess.ID, Path: "src/file.ts:1"}),
	)
	if err != nil {
		t.Fatalf("GetFileReferenceContent: %v", err)
	}
	if resp.Msg.GetContent() != content {
		t.Fatalf("expected content %q, got %q", content, resp.Msg.GetContent())
	}
	if resp.Msg.GetCategory() != "code" {
		t.Fatalf("expected code category, got %q", resp.Msg.GetCategory())
	}
}
