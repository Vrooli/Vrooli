package backlog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/testutil"
)

func createPRD(t *testing.T, tmpDir string, kind BacklogKind, name string, content string) {
	t.Helper()
	archiveDir := filepath.Join(tmpDir, backlogKindDirs[kind], name, "archive")
	testutil.MakeDir(t, archiveDir)
	if err := os.WriteFile(filepath.Join(archiveDir, "PRD.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExport_EmptyBacklog(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	h.Export(w, httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil))
	testutil.AssertStatusOK(t, w)
	if !strings.Contains(w.Body.String(), "items_count: 0") {
		t.Error("expected empty export count")
	}
}

func TestExport_WithPRD(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	createTestItem(t, tmpDir, KindIdea, BacklogItem{Name: "test-item", Title: "Test Item", Status: StatusBacklog, Priority: 5, Created: "2026-02-10T00:00:00Z", Updated: "2026-02-10T00:00:00Z"})
	createPRD(t, tmpDir, KindIdea, "test-item", "# Product Requirements\n\nThis is the PRD content.")

	w := httptest.NewRecorder()
	h.Export(w, httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil))
	testutil.AssertStatusOK(t, w)
	if !strings.Contains(w.Body.String(), "This is the PRD content.") {
		t.Error("expected PRD content")
	}
}

func TestExport_MultipleItemsSortedByPriority(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	createTestItem(t, tmpDir, KindIdea, BacklogItem{Name: "low", Title: "Low", Status: StatusBacklog, Priority: 8})
	createTestItem(t, tmpDir, KindFix, BacklogItem{Name: "high", Title: "High", Status: StatusBacklog, Priority: 1})

	w := httptest.NewRecorder()
	h.Export(w, httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", nil))
	testutil.AssertStatusOK(t, w)
	body := w.Body.String()
	if strings.Index(body, "High") > strings.Index(body, "Low") {
		t.Error("expected high-priority item first")
	}
}

func exportWithBody(t *testing.T, h *Handler, body string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/export", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Export(w, req)
	testutil.AssertStatusOK(t, w)
	return w.Body.String()
}

func TestExport_ToggleNotesOff(t *testing.T) {
	h, _ := setupTestHandler(t)
	if !strings.Contains(exportWithBody(t, h, `{}`), "### Notes") {
		t.Error("expected notes section by default")
	}
	if strings.Contains(exportWithBody(t, h, `{"includeNotes":false,"includeTemplate":false}`), "### Notes") {
		t.Error("expected notes section to be omitted")
	}
}

func TestExport_ToggleTemplateOff(t *testing.T) {
	h, _ := setupTestHandler(t)
	if !strings.Contains(exportWithBody(t, h, `{}`), "New Item Template") {
		t.Error("expected template by default")
	}
	if strings.Contains(exportWithBody(t, h, `{"includeTemplate":false}`), "New Item Template") {
		t.Error("expected template to be omitted")
	}
}
