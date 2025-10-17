package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helper functions

func TestGetVrooliRoot(t *testing.T) {
	tests := []struct {
		name        string
		vrooliRoot  string
		home        string
		expected    string
		description string
	}{
		{
			name:        "VROOLI_ROOT set",
			vrooliRoot:  "/custom/vrooli",
			home:        "/home/user",
			expected:    "/custom/vrooli",
			description: "Should use VROOLI_ROOT when set",
		},
		{
			name:        "HOME set, VROOLI_ROOT unset",
			vrooliRoot:  "",
			home:        "/home/user",
			expected:    "/home/user/Vrooli",
			description: "Should construct path from HOME when VROOLI_ROOT unset",
		},
		{
			name:        "Both unset",
			vrooliRoot:  "",
			home:        "",
			expected:    "",
			description: "Should return empty string when both unset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			origVrooliRoot := os.Getenv("VROOLI_ROOT")
			origHome := os.Getenv("HOME")
			defer func() {
				os.Setenv("VROOLI_ROOT", origVrooliRoot)
				os.Setenv("HOME", origHome)
			}()

			// Set test env vars
			os.Setenv("VROOLI_ROOT", tt.vrooliRoot)
			os.Setenv("HOME", tt.home)

			result := getVrooliRoot()
			if result != tt.expected {
				t.Errorf("%s: got %q, want %q", tt.description, result, tt.expected)
			}
		})
	}
}

func TestMapGitStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"A", "added"},
		{"M", "modified"},
		{"D", "deleted"},
		{"R", "renamed"},
		{"C", "copied"},
		{"X", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		t.Run("Status_"+tt.input, func(t *testing.T) {
			result := mapGitStatus(tt.input)
			if result != tt.expected {
				t.Errorf("mapGitStatus(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectScope(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"scenarios/git-control-tower/api/main.go", "scenario:git-control-tower"},
		{"scenarios/ecosystem-manager/queue/pending/task.yaml", "scenario:ecosystem-manager"},
		{"resources/postgres/lib/start.sh", "resource:postgres"},
		{"resources/ollama/cli/ollama", "resource:ollama"},
		{"docs/README.md", ""},
		{"api/main.go", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := detectScope(tt.path)
			if result != tt.expected {
				t.Errorf("detectScope(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestParseChangedFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ChangedFile
	}{
		{
			name:     "Empty input",
			input:    "",
			expected: []ChangedFile{},
		},
		{
			name:  "Single modified file",
			input: "M\tscenarios/git-control-tower/api/main.go",
			expected: []ChangedFile{
				{
					Path:   "scenarios/git-control-tower/api/main.go",
					Status: "modified",
					Scope:  "scenario:git-control-tower",
				},
			},
		},
		{
			name: "Multiple files with different statuses",
			input: `M	scenarios/git-control-tower/api/main.go
A	resources/postgres/schema.sql
D	docs/old.md`,
			expected: []ChangedFile{
				{
					Path:   "scenarios/git-control-tower/api/main.go",
					Status: "modified",
					Scope:  "scenario:git-control-tower",
				},
				{
					Path:   "resources/postgres/schema.sql",
					Status: "added",
					Scope:  "resource:postgres",
				},
				{
					Path:   "docs/old.md",
					Status: "deleted",
					Scope:  "",
				},
			},
		},
		{
			name:     "Invalid format line",
			input:    "InvalidLine",
			expected: []ChangedFile{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseChangedFiles(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseChangedFiles(%q) returned %d files, want %d", tt.name, len(result), len(tt.expected))
				return
			}
			for i, file := range result {
				if file.Path != tt.expected[i].Path ||
					file.Status != tt.expected[i].Status ||
					file.Scope != tt.expected[i].Scope {
					t.Errorf("parseChangedFiles(%q)[%d] = %+v, want %+v", tt.name, i, file, tt.expected[i])
				}
			}
		})
	}
}

// Test HTTP handlers

func TestHealthHandler(t *testing.T) {
	// Set required environment variables
	origLifecycle := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	defer os.Setenv("VROOLI_LIFECYCLE_MANAGED", origLifecycle)

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleHealth)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if response.Service != "git-control-tower" {
		t.Errorf("wrong service name: got %v want %v", response.Service, "git-control-tower")
	}

	if response.Status != "healthy" && response.Status != "degraded" {
		t.Errorf("invalid status: got %v", response.Status)
	}
}

func TestHandleHome(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handleHome(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", status, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Git Control Tower") {
		t.Errorf("homepage missing title content")
	}
	if !strings.Contains(body, "Repository Snapshot") {
		t.Errorf("homepage missing repository section")
	}
	if !strings.Contains(body, "Service Health") {
		t.Errorf("homepage missing health section")
	}
}

func TestStageHandler_InvalidJSON(t *testing.T) {
	// Set required environment variables
	origVrooliRoot := os.Getenv("VROOLI_ROOT")
	os.Setenv("VROOLI_ROOT", "/tmp/test-repo")
	defer os.Setenv("VROOLI_ROOT", origVrooliRoot)

	invalidJSON := []byte(`{invalid json}`)
	req, err := http.NewRequest("POST", "/api/v1/stage", bytes.NewBuffer(invalidJSON))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleStage)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestStageHandler_EmptyRequest(t *testing.T) {
	// Set required environment variables
	origVrooliRoot := os.Getenv("VROOLI_ROOT")
	testDir := t.TempDir()
	os.Setenv("VROOLI_ROOT", testDir)
	defer os.Setenv("VROOLI_ROOT", origVrooliRoot)

	// Initialize git repo
	if err := os.Chdir(testDir); err != nil {
		t.Skip("Cannot change to temp directory")
	}

	reqBody := StageRequest{
		Files: []string{},
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/stage", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleStage)
	handler.ServeHTTP(rr, req)

	var response StageResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if response.Success {
		t.Errorf("expected success=false for empty file list, got true")
	}
}

func TestCommitHandler_InvalidMessage(t *testing.T) {
	// Set required environment variables
	origVrooliRoot := os.Getenv("VROOLI_ROOT")
	testDir := t.TempDir()
	os.Setenv("VROOLI_ROOT", testDir)
	defer os.Setenv("VROOLI_ROOT", origVrooliRoot)

	// Test invalid conventional commit format
	reqBody := CommitRequest{
		Message: "invalid commit message without type",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/commit", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleCommit)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for invalid commit message: got %v want %v", status, http.StatusBadRequest)
	}
}

// Test conventional commit validation

func TestValidateCommitMessage(t *testing.T) {
	tests := []struct {
		message string
		valid   bool
	}{
		{"feat: add new feature", true},
		{"fix: resolve bug", true},
		{"docs: update README", true},
		{"feat(api): add endpoint", true},
		{"fix(ui): correct styling", true},
		{"chore: update dependencies", true},
		{"test: add unit tests", true},
		{"refactor: restructure code", true},
		{"style: format code", true},
		{"perf: optimize query", true},
		{"ci: update workflow", true},
		{"build: update Makefile", true},
		{"revert: undo previous change", true},
		{"invalid message", false},
		{"FEAT: wrong case", false},
		{"feat add feature", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := validateCommitMessage(tt.message)
			if result != tt.valid {
				t.Errorf("validateCommitMessage(%q) = %v, want %v", tt.message, result, tt.valid)
			}
		})
	}
}

// Benchmark tests

func BenchmarkDetectScope(b *testing.B) {
	testPaths := []string{
		"scenarios/git-control-tower/api/main.go",
		"resources/postgres/lib/start.sh",
		"docs/README.md",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			detectScope(path)
		}
	}
}

func BenchmarkMapGitStatus(b *testing.B) {
	statuses := []string{"A", "M", "D", "R", "C"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, status := range statuses {
			mapGitStatus(status)
		}
	}
}

func BenchmarkParseChangedFiles(b *testing.B) {
	input := `M	scenarios/git-control-tower/api/main.go
A	resources/postgres/schema.sql
D	docs/old.md
M	scenarios/ecosystem-manager/api/main.go`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseChangedFiles(input)
	}
}

// Helper function to create test git repository
func createTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo
	os.Chdir(dir)
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create test file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}
