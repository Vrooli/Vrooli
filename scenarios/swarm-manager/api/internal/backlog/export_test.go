package backlog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

func TestExport_NotesRoundTripMarkerAndExistingContent(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	createTestItem(t, tmpDir, KindIdea, BacklogItem{
		Name:     "noted-item",
		Title:    "Noted Item",
		Status:   StatusBacklog,
		Priority: 5,
		Created:  "2026-08-01T00:00:00Z",
		Updated:  "2026-08-01T00:00:00Z",
	})
	notesPath := filepath.Join(tmpDir, backlogKindDirs[KindIdea], "noted-item", "notes.md")
	if err := os.WriteFile(notesPath, []byte("Existing operator note.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	body := exportWithBody(t, h, `{"includeTemplate":false}`)
	if !strings.Contains(body, "### Notes\n\n<!-- notes:idea/noted-item -->\nExisting operator note.") {
		t.Fatalf("export did not emit parseable notes subsection:\n%s", body)
	}

	changes, errs := h.parseImportMarkdown(body)
	if len(errs) != 0 {
		t.Fatalf("round-trip parse errors: %v", errs)
	}
	if len(changes) != 0 {
		t.Fatalf("unchanged exported notes produced changes: %+v", changes)
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

func TestExport_FrontmatterArithmeticClosesAndDisclosesEveryFilter(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	archivedAt := "2026-08-01T00:00:00Z"
	createTestItem(t, tmpDir, KindIdea, BacklogItem{Name: "visible", Title: "Visible", Status: StatusBacklog, Priority: 2})
	createTestItem(t, tmpDir, KindIdea, BacklogItem{Name: "archived", Title: "Archived", Status: StatusBacklog, Priority: 2, ArchivedAt: &archivedAt})
	createTestItem(t, tmpDir, KindIdea, BacklogItem{Name: "dropped", Title: "Dropped", Status: StatusDropped, Priority: 2})
	createTestItem(t, tmpDir, KindIdea, BacklogItem{Name: "low-priority", Title: "Low priority", Status: StatusBacklog, Priority: 9})

	body := exportWithBody(t, h, `{"priorityMax":5,"includeTemplate":false}`)
	preFilter, exported, excluded := exportArithmetic(t, body)
	if preFilter != 4 || exported != 1 || excluded != 3 {
		t.Fatalf("frontmatter arithmetic = pre:%d excluded:%d exported:%d, want 4-3=1\n%s", preFilter, excluded, exported, body)
	}
	for _, filter := range []string{`filter: "archived"`, `filter: "status"`, `filter: "priority_max"`} {
		if !strings.Contains(body, filter) {
			t.Fatalf("frontmatter missing disclosed %s filter\n%s", filter, body)
		}
	}

	withArchived := exportWithBody(t, h, `{"priorityMax":5,"includeArchived":true,"includeTemplate":false}`)
	preFilter, exported, excluded = exportArithmetic(t, withArchived)
	if preFilter != 4 || exported != 2 || excluded != 2 {
		t.Fatalf("include-archived arithmetic = pre:%d excluded:%d exported:%d, want 4-2=2\n%s", preFilter, excluded, exported, withArchived)
	}
}

func exportArithmetic(t *testing.T, body string) (preFilter, exported, excluded int) {
	t.Helper()
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		var target *int
		switch {
		case strings.HasPrefix(line, "pre_filter_total:"):
			target = &preFilter
			line = strings.TrimSpace(strings.TrimPrefix(line, "pre_filter_total:"))
		case strings.HasPrefix(line, "items_count:"):
			target = &exported
			line = strings.TrimSpace(strings.TrimPrefix(line, "items_count:"))
		case strings.HasPrefix(line, "count:"):
			target = new(int)
			line = strings.TrimSpace(strings.TrimPrefix(line, "count:"))
		default:
			continue
		}
		value, err := strconv.Atoi(line)
		if err != nil {
			t.Fatalf("parse frontmatter count %q: %v", line, err)
		}
		if strings.HasPrefix(strings.TrimSpace(raw), "count:") {
			excluded += value
		} else {
			*target = value
		}
	}
	if preFilter-excluded != exported {
		t.Fatalf("frontmatter arithmetic does not close: %d-%d != %d", preFilter, excluded, exported)
	}
	return preFilter, exported, excluded
}
