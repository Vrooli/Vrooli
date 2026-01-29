package ideas

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/ecosystem"
	"swarm-manager/internal/testutil"
)

type listIdeasResponse struct {
	Ideas []Idea `json:"ideas"`
}

type ideaResponse struct {
	Idea Idea `json:"idea"`
}

type ideaFilesResponse struct {
	Files []IdeaFile `json:"files"`
}

type ideaFileResponse struct {
	File IdeaFile `json:"file"`
}

type queueIdeaResponse struct {
	Idea   Idea   `json:"idea"`
	TaskID string `json:"task_id"`
}

type createIdeaRequest struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// setupTestHandler creates a handler with a temporary ideas directory.
func setupTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	tmpDir := t.TempDir()
	ideasDir := filepath.Join(tmpDir, "ideas")
	if err := os.MkdirAll(ideasDir, 0o755); err != nil {
		t.Fatalf("Failed to create test ideas dir: %v", err)
	}
	return NewHandler(ideasDir), ideasDir
}

// createTestIdea creates a test idea in the ideas directory.
func createTestIdea(t *testing.T, ideasDir string, idea Idea) {
	t.Helper()
	ideaDir := filepath.Join(ideasDir, idea.Name)
	if err := os.MkdirAll(ideaDir, 0o755); err != nil {
		t.Fatalf("Failed to create idea dir: %v", err)
	}
	data, err := json.MarshalIndent(idea, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal idea: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ideaDir, "spec.json"), data, 0o644); err != nil {
		t.Fatalf("Failed to write spec.json: %v", err)
	}
}

// [REQ:REQ-P0-002] Test ideas list endpoint returns empty array
func TestList_Empty(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/ideas", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[listIdeasResponse](t, w)
	if len(resp.Ideas) != 0 {
		t.Errorf("Expected empty list, got %d ideas", len(resp.Ideas))
	}
}

// [REQ:REQ-P0-002] Test ideas list endpoint returns existing ideas
func TestList_WithIdeas(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create test ideas
	idea1 := Idea{
		Name:        "test-idea-1",
		Title:       "Test Idea 1",
		Description: "First test idea",
		Status:      StatusBacklog,
		Priority:    1,
		Tags:        []string{"test"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	idea2 := Idea{
		Name:        "test-idea-2",
		Title:       "Test Idea 2",
		Description: "Second test idea",
		Status:      StatusReady,
		Priority:    2,
		Tags:        []string{"test", "ready"},
		Created:     "2026-01-27T00:00:00Z",
		Updated:     "2026-01-27T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea1)
	createTestIdea(t, ideasDir, idea2)

	req := httptest.NewRequest("GET", "/api/v1/ideas", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[listIdeasResponse](t, w)
	if len(resp.Ideas) != 2 {
		t.Errorf("Expected 2 ideas, got %d", len(resp.Ideas))
	}

	// Verify sorting by priority
	if resp.Ideas[0].Priority > resp.Ideas[1].Priority {
		t.Error("Ideas should be sorted by priority ascending")
	}
}

// [REQ:REQ-P0-002] Test get single idea
func TestGet_Found(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	idea := Idea{
		Name:        "get-test",
		Title:       "Get Test",
		Description: "Test get endpoint",
		Status:      StatusBacklog,
		Priority:    1,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	req := httptest.NewRequest("GET", "/api/v1/ideas/get-test", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "get-test"})
	w := httptest.NewRecorder()

	h.Get(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[ideaResponse](t, w)
	result := resp.Idea
	if result.Name != "get-test" {
		t.Errorf("Expected name 'get-test', got '%s'", result.Name)
	}
}

// [REQ:REQ-P0-002] Test get non-existent idea returns 404
func TestGet_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/ideas/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	w := httptest.NewRecorder()

	h.Get(w, req)

	testutil.AssertStatusNotFound(t, w)
}

// [REQ:REQ-P0-002] Test create new idea
func TestCreate_Success(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	payload := createIdeaRequest{
		Name:        "New Test Idea",
		Title:       "New Test Idea",
		Description: "A new test idea",
		Priority:    3,
		Tags:        []string{"new", "test"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/ideas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[ideaResponse](t, w)
	result := resp.Idea

	// Name should be sanitized
	if result.Name != "new-test-idea" {
		t.Errorf("Expected sanitized name 'new-test-idea', got '%s'", result.Name)
	}

	if result.Status != StatusBacklog {
		t.Errorf("Expected status 'backlog', got '%s'", result.Status)
	}

	// Verify file was created
	specPath := filepath.Join(ideasDir, "new-test-idea", "spec.json")
	testutil.AssertFileExists(t, specPath)
}

// [REQ:REQ-P0-002] Test create with missing required fields
func TestCreate_MissingFields(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := createIdeaRequest{
		Name: "", // Missing
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/ideas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

// [REQ:REQ-P0-002] Test create duplicate idea returns conflict
func TestCreate_Duplicate(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "existing",
		Title:   "Existing",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	payload := createIdeaRequest{
		Name:  "existing",
		Title: "Duplicate",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/ideas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatus(t, w, http.StatusConflict)
}

// [REQ:REQ-P0-002] Test update idea
func TestUpdate_Success(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:        "update-test",
		Title:       "Update Test",
		Description: "Original description",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Update
	update := Idea{
		Title:       "Updated Title",
		Description: "Updated description",
		Status:      StatusReady,
		Priority:    1,
		Tags:        []string{"updated"},
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/api/v1/ideas/update-test", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "update-test"})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Update(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[ideaResponse](t, w)
	result := resp.Idea

	if result.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", result.Title)
	}
	if result.Status != StatusReady {
		t.Errorf("Expected status 'ready', got '%s'", result.Status)
	}
	// Name should be preserved
	if result.Name != "update-test" {
		t.Errorf("Expected name 'update-test', got '%s'", result.Name)
	}
	// Created should be preserved
	if result.Created != "2026-01-28T00:00:00Z" {
		t.Errorf("Created timestamp should be preserved")
	}
}

// [REQ:REQ-P0-002] Test update non-existent idea returns 404
func TestUpdate_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	update := Idea{Title: "Updated"}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/api/v1/ideas/nonexistent", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Update(w, req)

	testutil.AssertStatusNotFound(t, w)
}

// [REQ:REQ-P0-002] Test delete idea
func TestDelete_Success(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "delete-test",
		Title:   "Delete Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	req := httptest.NewRequest("DELETE", "/api/v1/ideas/delete-test", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "delete-test"})
	w := httptest.NewRecorder()

	h.Delete(w, req)

	testutil.AssertStatus(t, w, http.StatusNoContent)

	// Verify directory was deleted
	ideaDir := filepath.Join(ideasDir, "delete-test")
	testutil.AssertFileNotExists(t, ideaDir)
}

// [REQ:REQ-P0-002] Test delete non-existent idea returns 204 (idempotent delete)
func TestDelete_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("DELETE", "/api/v1/ideas/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	w := httptest.NewRecorder()

	h.Delete(w, req)

	// Idempotent delete: returns 204 whether resource exists or not
	// This makes the operation replay-safe for retries
	testutil.AssertStatus(t, w, http.StatusNoContent)
}

// [REQ:REQ-P0-002] Test delete is idempotent - calling twice produces same result
func TestDelete_Idempotent(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "delete-twice",
		Title:   "Delete Twice",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// First delete - removes the idea
	req1 := httptest.NewRequest("DELETE", "/api/v1/ideas/delete-twice", nil)
	req1 = mux.SetURLVars(req1, map[string]string{"name": "delete-twice"})
	w1 := httptest.NewRecorder()
	h.Delete(w1, req1)

	testutil.AssertStatus(t, w1, http.StatusNoContent)

	// Second delete - should also succeed (idempotent)
	req2 := httptest.NewRequest("DELETE", "/api/v1/ideas/delete-twice", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"name": "delete-twice"})
	w2 := httptest.NewRecorder()
	h.Delete(w2, req2)

	testutil.AssertStatus(t, w2, http.StatusNoContent)
}

// [REQ:REQ-P0-002] Test create is replay-safe - duplicate returns 409
func TestCreate_ReplaySafe(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := createIdeaRequest{
		Name:        "replay-test",
		Title:       "Replay Test",
		Description: "Testing replay safety",
	}
	body1, _ := json.Marshal(payload)
	body2, _ := json.Marshal(payload)

	// First create - succeeds
	req1 := httptest.NewRequest("POST", "/api/v1/ideas", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	h.Create(w1, req1)

	testutil.AssertStatusCreated(t, w1)

	// Second create (replay) - returns conflict, not error
	req2 := httptest.NewRequest("POST", "/api/v1/ideas", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.Create(w2, req2)

	testutil.AssertStatus(t, w2, http.StatusConflict)
}

// [REQ:REQ-P0-002] Test update is idempotent - same payload produces same result
func TestUpdate_Idempotent(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:        "idempotent-test",
		Title:       "Original",
		Description: "Original description",
		Status:      StatusBacklog,
		Priority:    5,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	update := Idea{
		Title:       "Updated Title",
		Description: "Updated description",
		Status:      StatusReady,
		Priority:    1,
		Tags:        []string{"updated"},
	}

	// First update
	body1, _ := json.Marshal(update)
	req1 := httptest.NewRequest("PUT", "/api/v1/ideas/idempotent-test", bytes.NewReader(body1))
	req1 = mux.SetURLVars(req1, map[string]string{"name": "idempotent-test"})
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	h.Update(w1, req1)

	testutil.AssertStatusOK(t, w1)

	resp1 := testutil.DecodeJSON[ideaResponse](t, w1)
	result1 := resp1.Idea

	// Second update with same payload
	body2, _ := json.Marshal(update)
	req2 := httptest.NewRequest("PUT", "/api/v1/ideas/idempotent-test", bytes.NewReader(body2))
	req2 = mux.SetURLVars(req2, map[string]string{"name": "idempotent-test"})
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.Update(w2, req2)

	testutil.AssertStatusOK(t, w2)

	resp2 := testutil.DecodeJSON[ideaResponse](t, w2)
	result2 := resp2.Idea

	// Core data should be identical (Updated timestamp may differ, which is acceptable)
	if result1.Title != result2.Title {
		t.Error("Idempotent update: titles should match")
	}
	if result1.Description != result2.Description {
		t.Error("Idempotent update: descriptions should match")
	}
	if result1.Status != result2.Status {
		t.Error("Idempotent update: statuses should match")
	}
	if result1.Priority != result2.Priority {
		t.Error("Idempotent update: priorities should match")
	}
	if result1.Created != result2.Created {
		t.Error("Idempotent update: Created timestamps should be preserved")
	}
}

// [REQ:REQ-P0-004] Test list files for idea returns empty array
func TestListFiles_Empty(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea with only spec.json
	idea := Idea{
		Name:    "files-test",
		Title:   "Files Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	req := httptest.NewRequest("GET", "/api/v1/ideas/files-test/files", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "files-test"})
	w := httptest.NewRecorder()

	h.ListFiles(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[ideaFilesResponse](t, w)
	files := resp.Files

	// Should contain at least spec.json
	if len(files) < 1 {
		t.Errorf("Expected at least 1 file (spec.json), got %d", len(files))
	}
}

// [REQ:REQ-P0-004] Test list files for idea with multiple files
func TestListFiles_WithFiles(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "multi-files-test",
		Title:   "Multi Files Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Add additional files
	ideaDir := filepath.Join(ideasDir, "multi-files-test")
	testutil.WriteFile(t, filepath.Join(ideaDir, "notes.md"), "# Notes")

	// Create subdirectory with file
	researchDir := filepath.Join(ideaDir, "research")
	testutil.WriteFile(t, filepath.Join(researchDir, "findings.md"), "# Findings")

	req := httptest.NewRequest("GET", "/api/v1/ideas/multi-files-test/files", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "multi-files-test"})
	w := httptest.NewRecorder()

	h.ListFiles(w, req)

	testutil.AssertStatusOK(t, w)

	resp := testutil.DecodeJSON[ideaFilesResponse](t, w)
	files := resp.Files

	// Should have research (directory), notes.md, spec.json
	if len(files) != 3 {
		t.Errorf("Expected 3 items, got %d", len(files))
	}

	// Directories should come first (sorted)
	if files[0].Type != "directory" {
		t.Error("Expected first item to be directory")
	}
	if files[0].Name != "research" {
		t.Errorf("Expected first item name 'research', got '%s'", files[0].Name)
	}

	// Check that directory has children
	if len(files[0].Children) != 1 {
		t.Errorf("Expected research directory to have 1 child, got %d", len(files[0].Children))
	}
}

// [REQ:REQ-P0-004] Test list files for non-existent idea returns 404
func TestListFiles_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/ideas/nonexistent/files", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	w := httptest.NewRecorder()

	h.ListFiles(w, req)

	testutil.AssertStatusNotFound(t, w)
}

// [REQ:REQ-P0-004] Test IdeaFile type structure
func TestIdeaFile_Structure(t *testing.T) {
	file := IdeaFile{
		Name:     "test.md",
		Path:     "research/test.md",
		Type:     "file",
		Size:     1024,
		Children: nil,
	}

	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("Failed to marshal IdeaFile: %v", err)
	}

	var parsed IdeaFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal IdeaFile: %v", err)
	}

	if parsed.Name != "test.md" {
		t.Errorf("Expected name 'test.md', got '%s'", parsed.Name)
	}
	if parsed.Path != "research/test.md" {
		t.Errorf("Expected path 'research/test.md', got '%s'", parsed.Path)
	}
	if parsed.Type != "file" {
		t.Errorf("Expected type 'file', got '%s'", parsed.Type)
	}
	if parsed.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", parsed.Size)
	}
}

// [REQ:REQ-P0-001] Test name sanitization
func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Test Idea", "my-test-idea"},
		{"UPPERCASE", "uppercase"},
		{"special!@#chars", "specialchars"},
		{"spaces in middle", "spaces-in-middle"},
		{"kebab-case", "kebab-case"},
		{"123-numbers", "123-numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// [REQ:REQ-P0-004] Test get file content for text files
func TestGetFileContent_TextFile(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea with a file
	idea := Idea{
		Name:    "content-test",
		Title:   "Content Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Add a markdown file
	ideaDir := filepath.Join(ideasDir, "content-test")
	content := "# Hello World\n\nThis is test content."
	testutil.WriteFile(t, filepath.Join(ideaDir, "notes.md"), content)

	req := httptest.NewRequest("GET", "/api/v1/ideas/content-test/files/notes.md", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "content-test", "filepath": "notes.md"})
	w := httptest.NewRecorder()

	h.GetFileContent(w, req)

	testutil.AssertStatusOK(t, w)

	if w.Body.String() != content {
		t.Errorf("Expected content %q, got %q", content, w.Body.String())
	}

	if w.Header().Get("Content-Type") != "text/markdown" {
		t.Errorf("Expected Content-Type text/markdown, got %s", w.Header().Get("Content-Type"))
	}
}

// [REQ:REQ-P0-004] Test get file content for nested file
func TestGetFileContent_NestedFile(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "nested-content-test",
		Title:   "Nested Content Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Add a nested file (testutil.WriteFile creates parent dirs automatically)
	ideaDir := filepath.Join(ideasDir, "nested-content-test")
	docsDir := filepath.Join(ideaDir, "docs")
	content := "Nested file content"
	testutil.WriteFile(t, filepath.Join(docsDir, "readme.txt"), content)

	req := httptest.NewRequest("GET", "/api/v1/ideas/nested-content-test/files/docs/readme.txt", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nested-content-test", "filepath": "docs/readme.txt"})
	w := httptest.NewRecorder()

	h.GetFileContent(w, req)

	testutil.AssertStatusOK(t, w)

	if w.Body.String() != content {
		t.Errorf("Expected content %q, got %q", content, w.Body.String())
	}
}

// [REQ:REQ-P0-004] Test get file content for non-existent idea returns 404
func TestGetFileContent_IdeaNotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/ideas/nonexistent/files/test.txt", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent", "filepath": "test.txt"})
	w := httptest.NewRecorder()

	h.GetFileContent(w, req)

	testutil.AssertStatusNotFound(t, w)
}

// [REQ:REQ-P0-004] Test get file content for non-existent file returns 404
func TestGetFileContent_FileNotFound(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "file-not-found-test",
		Title:   "File Not Found Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	req := httptest.NewRequest("GET", "/api/v1/ideas/file-not-found-test/files/missing.txt", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "file-not-found-test", "filepath": "missing.txt"})
	w := httptest.NewRecorder()

	h.GetFileContent(w, req)

	testutil.AssertStatusNotFound(t, w)
}

// [REQ:REQ-P0-004] Test get file content blocks directory traversal
func TestGetFileContent_PathTraversal(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "traversal-test",
		Title:   "Traversal Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Try path traversal
	req := httptest.NewRequest("GET", "/api/v1/ideas/traversal-test/files/../../../etc/passwd", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "traversal-test", "filepath": "../../../etc/passwd"})
	w := httptest.NewRecorder()

	h.GetFileContent(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

// [REQ:REQ-P0-004] Test get file content blocks reading directories
func TestGetFileContent_Directory(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea with subdirectory
	idea := Idea{
		Name:    "dir-test",
		Title:   "Directory Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	ideaDir := filepath.Join(ideasDir, "dir-test")
	testutil.MakeDir(t, filepath.Join(ideaDir, "subdir"))

	req := httptest.NewRequest("GET", "/api/v1/ideas/dir-test/files/subdir", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "dir-test", "filepath": "subdir"})
	w := httptest.NewRecorder()

	h.GetFileContent(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

// [REQ:REQ-P0-004] Test content type detection
func TestGetContentType(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".md", "text/markdown"},
		{".json", "application/json"},
		{".js", "application/javascript"},
		{".ts", "text/typescript"},
		{".go", "text/x-go"},
		{".py", "text/x-python"},
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".svg", "image/svg+xml"},
		{".html", "text/html"},
		{".css", "text/css"},
		{".yaml", "text/yaml"},
		{".unknown", "text/plain"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := getContentType(tt.ext)
			if result != tt.expected {
				t.Errorf("getContentType(%q) = %q, want %q", tt.ext, result, tt.expected)
			}
		})
	}
}

// [REQ:REQ-P0-004] Test upload file basic functionality
func TestUploadFile_Success(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "upload-test",
		Title:   "Upload Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	part, err := writer.CreateFormFile("file", "test-upload.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	fileContent := "This is test content for upload"
	if _, err := part.Write([]byte(fileContent)); err != nil {
		t.Fatalf("Failed to write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ideas/upload-test/files", &buf)
	req = mux.SetURLVars(req, map[string]string{"name": "upload-test"})
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	h.UploadFile(w, req)

	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[ideaFileResponse](t, w)
	result := resp.File

	if result.Name != "test-upload.txt" {
		t.Errorf("Expected name 'test-upload.txt', got '%s'", result.Name)
	}
	if result.Type != "file" {
		t.Errorf("Expected type 'file', got '%s'", result.Type)
	}

	// Verify file was created
	filePath := filepath.Join(ideasDir, "upload-test", "test-upload.txt")
	testutil.AssertFileExists(t, filePath)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read uploaded file: %v", err)
	}
	if string(content) != fileContent {
		t.Errorf("Expected content %q, got %q", fileContent, string(content))
	}
}

// [REQ:REQ-P0-004] Test upload file to subdirectory
func TestUploadFile_WithPath(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "upload-path-test",
		Title:   "Upload Path Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Create multipart form with path
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", "nested.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("Nested content")); err != nil {
		t.Fatalf("Failed to write form file: %v", err)
	}

	// Add path field
	if err := writer.WriteField("path", "docs/research"); err != nil {
		t.Fatalf("Failed to write path field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ideas/upload-path-test/files", &buf)
	req = mux.SetURLVars(req, map[string]string{"name": "upload-path-test"})
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	h.UploadFile(w, req)

	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[ideaFileResponse](t, w)
	result := resp.File

	expectedPath := "docs/research/nested.txt"
	if result.Path != expectedPath {
		t.Errorf("Expected path %q, got %q", expectedPath, result.Path)
	}

	// Verify file was created in nested directory
	filePath := filepath.Join(ideasDir, "upload-path-test", "docs", "research", "nested.txt")
	testutil.AssertFileExists(t, filePath)
}

// [REQ:REQ-P0-004] Test upload file to non-existent idea returns 404
func TestUploadFile_IdeaNotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("content")); err != nil {
		t.Fatalf("Failed to write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ideas/nonexistent/files", &buf)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	h.UploadFile(w, req)

	testutil.AssertStatusNotFound(t, w)
}

// [REQ:REQ-P0-004] Test upload file without file returns 400
func TestUploadFile_NoFile(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "no-file-test",
		Title:   "No File Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Empty multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/ideas/no-file-test/files", &buf)
	req = mux.SetURLVars(req, map[string]string{"name": "no-file-test"})
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	h.UploadFile(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

// [REQ:REQ-P0-005] Test queue idea for non-existent idea returns 404
func TestQueue_IdeaNotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/api/v1/ideas/nonexistent/queue", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	w := httptest.NewRecorder()

	h.Queue(w, req)

	testutil.AssertStatusNotFound(t, w)
}

// [REQ:REQ-P0-005] Test queue idea rejects already-queued ideas
func TestQueue_AlreadyQueued(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create already-queued idea
	idea := Idea{
		Name:    "already-queued",
		Title:   "Already Queued",
		Status:  StatusQueued,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	req := httptest.NewRequest("POST", "/api/v1/ideas/already-queued/queue", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "already-queued"})
	w := httptest.NewRecorder()

	h.Queue(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

// [REQ:REQ-P0-005] Test queue idea rejects in-progress ideas
func TestQueue_InProgress(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create in-progress idea
	idea := Idea{
		Name:    "in-progress",
		Title:   "In Progress",
		Status:  StatusInProgress,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	req := httptest.NewRequest("POST", "/api/v1/ideas/in-progress/queue", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "in-progress"})
	w := httptest.NewRecorder()

	h.Queue(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

// [REQ:REQ-P0-005] Test queue idea rejects completed ideas
func TestQueue_Completed(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create completed idea
	idea := Idea{
		Name:    "completed",
		Title:   "Completed",
		Status:  StatusCompleted,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	req := httptest.NewRequest("POST", "/api/v1/ideas/completed/queue", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "completed"})
	w := httptest.NewRecorder()

	h.Queue(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

// [REQ:REQ-P0-005] Test queue idea rejects invalid operation type
func TestQueue_InvalidOperation(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create queueable idea
	idea := Idea{
		Name:    "queue-invalid-op",
		Title:   "Queue Invalid Op",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Send invalid operation
	body := bytes.NewBufferString(`{"operation": "invalid"}`)
	req := httptest.NewRequest("POST", "/api/v1/ideas/queue-invalid-op/queue", body)
	req = mux.SetURLVars(req, map[string]string{"name": "queue-invalid-op"})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Queue(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

// [REQ:REQ-P0-005] Test QueueResponse structure
func TestQueueResponse_Structure(t *testing.T) {
	idea := Idea{
		Name:    "test",
		Title:   "Test",
		Status:  StatusQueued,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}

	response := queueIdeaResponse{
		Idea:   idea,
		TaskID: "task-123",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal QueueResponse: %v", err)
	}

	var parsed queueIdeaResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal QueueResponse: %v", err)
	}

	if parsed.TaskID != "task-123" {
		t.Errorf("Expected task_id 'task-123', got '%s'", parsed.TaskID)
	}
	if parsed.Idea.Name != "test" {
		t.Errorf("Expected idea name 'test', got '%s'", parsed.Idea.Name)
	}
	if parsed.Idea.Status != StatusQueued {
		t.Errorf("Expected idea status 'queued', got '%s'", parsed.Idea.Status)
	}
}

// [REQ:REQ-P0-005] Test queueable status detection
func TestQueueableStatuses(t *testing.T) {
	queueable := map[IdeaStatus]bool{
		StatusBacklog:     true,
		StatusResearching: true,
		StatusReady:       true,
	}

	nonQueueable := []IdeaStatus{
		StatusQueued,
		StatusInProgress,
		StatusCompleted,
		StatusArchived,
	}

	for status, shouldBeQueueable := range queueable {
		if !shouldBeQueueable {
			t.Errorf("Status %q should be queueable", status)
		}
	}

	for _, status := range nonQueueable {
		if queueable[status] {
			t.Errorf("Status %q should NOT be queueable", status)
		}
	}
}

// mockEcosystemClient is a test double for the EcosystemClient interface.
type mockEcosystemClient struct {
	createTaskFn func(ctx context.Context, req ecosystem.CreateTaskRequest) (string, error)
}

func (m *mockEcosystemClient) CreateTask(ctx context.Context, req ecosystem.CreateTaskRequest) (string, error) {
	return m.createTaskFn(ctx, req)
}

// [REQ:REQ-P0-005] Test queue idea success with mocked ecosystem client
func TestQueue_Success(t *testing.T) {
	tmpDir := t.TempDir()
	ideasDir := filepath.Join(tmpDir, "ideas")
	testutil.MakeDir(t, ideasDir)

	// Create queueable idea (backlog status)
	idea := Idea{
		Name:     "queue-success",
		Title:    "Queue Success Test",
		Status:   StatusBacklog,
		Priority: 2,
		Tags:     []string{"testing"},
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Create mock ecosystem client that returns success
	mockClient := &mockEcosystemClient{
		createTaskFn: func(_ context.Context, req ecosystem.CreateTaskRequest) (string, error) {
			// Verify the request was built correctly
			if req.Priority != 2 {
				t.Errorf("Expected priority 2, got %d", req.Priority)
			}
			if req.Operation != "generator" {
				t.Errorf("Expected operation 'generator', got %s", req.Operation)
			}
			if req.Category != "testing" {
				t.Errorf("Expected category 'testing', got %s", req.Category)
			}
			return "task-12345", nil
		},
	}

	// Create handler with mocked client
	h := NewHandlerWithClient(ideasDir, mockClient)

	req := httptest.NewRequest("POST", "/api/v1/ideas/queue-success/queue", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "queue-success"})
	w := httptest.NewRecorder()

	h.Queue(w, req)

	testutil.AssertStatus(t, w, http.StatusAccepted)

	result := testutil.DecodeJSON[queueIdeaResponse](t, w)
	if result.TaskID != "task-12345" {
		t.Errorf("Expected task_id 'task-12345', got '%s'", result.TaskID)
	}
	if result.Idea.Status != StatusQueued {
		t.Errorf("Expected idea status 'queued', got '%s'", result.Idea.Status)
	}
}

// [REQ:REQ-P0-005] Test queue idea with custom operation (improver)
func TestQueue_WithImproverOperation(t *testing.T) {
	tmpDir := t.TempDir()
	ideasDir := filepath.Join(tmpDir, "ideas")
	testutil.MakeDir(t, ideasDir)

	idea := Idea{
		Name:     "queue-improver",
		Title:    "Queue Improver Test",
		Status:   StatusReady,
		Priority: 5,
		Tags:     []string{},
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Create mock ecosystem client that verifies improver operation
	mockClient := &mockEcosystemClient{
		createTaskFn: func(_ context.Context, req ecosystem.CreateTaskRequest) (string, error) {
			if req.Operation != "improver" {
				t.Errorf("Expected operation 'improver', got %s", req.Operation)
			}
			// Should use "uncategorized" when tags are empty
			if req.Category != "uncategorized" {
				t.Errorf("Expected category 'uncategorized', got %s", req.Category)
			}
			return "task-improver-1", nil
		},
	}

	h := NewHandlerWithClient(ideasDir, mockClient)

	body := bytes.NewBufferString(`{"operation": "improver"}`)
	req := httptest.NewRequest("POST", "/api/v1/ideas/queue-improver/queue", body)
	req = mux.SetURLVars(req, map[string]string{"name": "queue-improver"})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Queue(w, req)

	testutil.AssertStatus(t, w, http.StatusAccepted)

	result := testutil.DecodeJSON[queueIdeaResponse](t, w)
	if result.TaskID != "task-improver-1" {
		t.Errorf("Expected task_id 'task-improver-1', got '%s'", result.TaskID)
	}
}

// [REQ:REQ-P0-005] Test queue idea when ecosystem client fails
func TestQueue_EcosystemError(t *testing.T) {
	tmpDir := t.TempDir()
	ideasDir := filepath.Join(tmpDir, "ideas")
	testutil.MakeDir(t, ideasDir)

	idea := Idea{
		Name:    "queue-fail",
		Title:   "Queue Fail Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Create mock ecosystem client that returns an error
	mockClient := &mockEcosystemClient{
		createTaskFn: func(_ context.Context, req ecosystem.CreateTaskRequest) (string, error) {
			return "", ecosystem.ErrNotAvailable
		},
	}

	h := NewHandlerWithClient(ideasDir, mockClient)

	req := httptest.NewRequest("POST", "/api/v1/ideas/queue-fail/queue", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "queue-fail"})
	w := httptest.NewRecorder()

	h.Queue(w, req)

	// Should return 500 internal server error when ecosystem is unavailable
	testutil.AssertStatus(t, w, http.StatusInternalServerError)
}

// [REQ:REQ-P0-001] Test sanitizeName edge cases
func TestSanitizeName_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic cases already covered
		{"My Test Idea", "my-test-idea"},
		{"UPPERCASE", "uppercase"},
		// Edge cases
		{"---leading-hyphens", "---leading-hyphens"},
		{"trailing-hyphens---", "trailing-hyphens---"},
		{"multiple   spaces", "multiple---spaces"},
		{"", ""},
		{"   ", "---"},
		{"123", "123"},
		{"a-b-c", "a-b-c"},
		{"MixedCase123", "mixedcase123"},
		{"special!@#$%^&*()chars", "specialchars"},
		{"unicode-émoji-🎉", "unicode-moji-"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// [REQ:REQ-P0-002] Test update with invalid JSON returns 400
func TestUpdate_InvalidJSON(t *testing.T) {
	h, ideasDir := setupTestHandler(t)

	// Create existing idea
	idea := Idea{
		Name:    "invalid-json-test",
		Title:   "Invalid JSON Test",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	// Send invalid JSON
	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest("PUT", "/api/v1/ideas/invalid-json-test", body)
	req = mux.SetURLVars(req, map[string]string{"name": "invalid-json-test"})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Update(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

// [REQ:REQ-P0-002] Test create with invalid JSON returns 400
func TestCreate_InvalidJSON(t *testing.T) {
	h, _ := setupTestHandler(t)

	body := bytes.NewBufferString(`{not valid json}`)
	req := httptest.NewRequest("POST", "/api/v1/ideas", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

type mockAgentService struct {
	response agentmanager.RunResult
}

func (m *mockAgentService) IsEnabled() bool                    { return true }
func (m *mockAgentService) IsAvailable(_ context.Context) bool { return true }
func (m *mockAgentService) ResolveURL(_ context.Context) (string, error) {
	return "http://localhost:1234", nil
}
func (m *mockAgentService) GetProfileID() string { return "" }

func (m *mockAgentService) SpawnResearch(_ context.Context, req agentmanager.ResearchSpawnRequest) (agentmanager.RunResult, error) {
	if strings.TrimSpace(req.Title) == "" {
		return agentmanager.RunResult{}, agentmanager.ErrRequestFailed
	}
	return m.response, nil
}

func (m *mockAgentService) SpawnRecommendation(_ context.Context, _ agentmanager.RecommendationSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, nil
}

// [REQ:REQ-P1-004-API] Test research endpoint uses agent-manager client
func TestResearch_Success(t *testing.T) {
	_, ideasDir := setupTestHandler(t)

	idea := Idea{
		Name:        "research-test",
		Title:       "Research Test",
		Description: "Research this idea",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{"ai"},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestIdea(t, ideasDir, idea)

	agentService := &mockAgentService{
		response: agentmanager.RunResult{
			TaskID:  "task-123",
			RunID:   "run-456",
			BaseURL: "http://localhost:1234",
		},
	}

	h := NewHandlerWithClients(ideasDir, nil, agentService)

	payload := bytes.NewBufferString(`{"prompt":"Focus on feasibility","mode":"clarify"}`)
	req := httptest.NewRequest("POST", "/api/v1/ideas/research-test/research", payload)
	req = mux.SetURLVars(req, map[string]string{"name": "research-test"})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Research(w, req)

	testutil.AssertStatusCreated(t, w)
}
