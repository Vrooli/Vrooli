package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/gorilla/mux"
)

func TestPromptsHandlers_ListGetUpdate(t *testing.T) {
	promptsDir := setupTempPromptsDir(t)
	assembler, err := prompts.NewAssembler(promptsDir, promptsDir)
	if err != nil {
		t.Fatalf("failed to create assembler: %v", err)
	}

	handlers := NewPromptsHandlers(assembler)

	// List
	req := httptest.NewRequest(http.MethodGet, "/api/prompts", nil)
	rr := httptest.NewRecorder()
	handlers.ListPromptFilesHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from list, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "templates/resource-generator.md") {
		t.Fatalf("list response missing template file: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "phases/progress.md") {
		t.Fatalf("list response missing phase file: %s", rr.Body.String())
	}

	// Get
	req = httptest.NewRequest(http.MethodGet, "/api/prompts/templates/resource-generator.md", nil)
	req = muxSetPathVars(req, map[string]string{"path": "templates/resource-generator.md"})
	rr = httptest.NewRecorder()
	handlers.GetPromptFileHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from get, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "template-body") {
		t.Fatalf("expected file content in get response, got: %s", rr.Body.String())
	}

	// Update
	req = httptest.NewRequest(http.MethodPut, "/api/prompts/templates/resource-generator.md", strings.NewReader(`{"content":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	req = muxSetPathVars(req, map[string]string{"path": "templates/resource-generator.md"})
	rr = httptest.NewRecorder()
	handlers.UpdatePromptFileHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from update, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "updated") {
		t.Fatalf("expected updated content in response, got: %s", rr.Body.String())
	}

	data, err := os.ReadFile(filepath.Join(promptsDir, "templates", "resource-generator.md"))
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}
	if string(data) != "updated" {
		t.Fatalf("file not updated, got: %s", string(data))
	}
}

func TestPromptsHandlers_PathTraversal(t *testing.T) {
	promptsDir := setupTempPromptsDir(t)
	assembler, err := prompts.NewAssembler(promptsDir, promptsDir)
	if err != nil {
		t.Fatalf("failed to create assembler: %v", err)
	}

	handlers := NewPromptsHandlers(assembler)

	req := httptest.NewRequest(http.MethodGet, "/api/prompts/../secrets.txt", nil)
	req = muxSetPathVars(req, map[string]string{"path": "../secrets.txt"})
	rr := httptest.NewRecorder()
	handlers.GetPromptFileHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for traversal, got %d", rr.Code)
	}
}

func setupTempPromptsDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	promptsDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(filepath.Join(promptsDir, "templates"), 0o755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(promptsDir, "phases"), 0o755); err != nil {
		t.Fatalf("failed to create phases dir: %v", err)
	}

	sections := "name: test\nbase_sections: []\noperations: {}\n"
	if err := os.WriteFile(filepath.Join(promptsDir, "sections.yaml"), []byte(sections), 0o644); err != nil {
		t.Fatalf("failed to write sections.yaml: %v", err)
	}

	if err := os.WriteFile(filepath.Join(promptsDir, "templates", "resource-generator.md"), []byte("template-body"), 0o644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptsDir, "phases", "progress.md"), []byte("# progress\n"), 0o644); err != nil {
		t.Fatalf("failed to write phase file: %v", err)
	}

	return promptsDir
}

func muxSetPathVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}
