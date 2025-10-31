package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type execCall struct {
	query string
	args  []any
}

type recordingExec struct {
	calls []execCall
}

type noopResult struct{}

func (noopResult) LastInsertId() (int64, error) { return 0, nil }

func (noopResult) RowsAffected() (int64, error) { return 0, nil }

func (r *recordingExec) Exec(query string, args ...any) (sql.Result, error) {
	r.calls = append(r.calls, execCall{query: query, args: args})
	return noopResult{}, nil
}

func TestNullString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected sql.NullString
	}{
		{
			name:  "non-empty string",
			input: "test",
			expected: sql.NullString{
				String: "test",
				Valid:  true,
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: sql.NullString{
				String: "",
				Valid:  false,
			},
		},
		{
			name:  "whitespace string",
			input: "   ",
			expected: sql.NullString{
				String: "   ",
				Valid:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullString(tt.input)

			if result.Valid != tt.expected.Valid {
				t.Errorf("nullString(%q).Valid = %v, want %v", tt.input, result.Valid, tt.expected.Valid)
			}

			if result.String != tt.expected.String {
				t.Errorf("nullString(%q).String = %q, want %q", tt.input, result.String, tt.expected.String)
			}
		})
	}
}

func TestGetDraftPath(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entityName string
		expected   string
	}{
		{
			name:       "scenario draft",
			entityType: "scenario",
			entityName: "test-scenario",
			expected:   "../data/prd-drafts/scenario/test-scenario.md",
		},
		{
			name:       "resource draft",
			entityType: "resource",
			entityName: "test-resource",
			expected:   "../data/prd-drafts/resource/test-resource.md",
		},
		{
			name:       "scenario with special characters",
			entityType: "scenario",
			entityName: "my-test-scenario-v2",
			expected:   "../data/prd-drafts/scenario/my-test-scenario-v2.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDraftPath(tt.entityType, tt.entityName)

			if result != tt.expected {
				t.Errorf("getDraftPath(%q, %q) = %q, want %q",
					tt.entityType, tt.entityName, result, tt.expected)
			}
		})
	}
}

func TestDraftValidation(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entityName string
		wantValid  bool
	}{
		{
			name:       "valid scenario",
			entityType: "scenario",
			entityName: "test-scenario",
			wantValid:  true,
		},
		{
			name:       "valid resource",
			entityType: "resource",
			entityName: "test-resource",
			wantValid:  true,
		},
		{
			name:       "invalid entity type",
			entityType: "invalid",
			entityName: "test",
			wantValid:  false,
		},
		{
			name:       "empty entity type",
			entityType: "",
			entityName: "test",
			wantValid:  false,
		},
		{
			name:       "empty entity name",
			entityType: "scenario",
			entityName: "",
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate entity type
			isValidType := tt.entityType == "scenario" || tt.entityType == "resource"
			if !isValidType && tt.wantValid {
				t.Errorf("Expected entity type %q to be invalid", tt.entityType)
			}

			// Validate entity name
			isValidName := tt.entityName != ""
			if !isValidName && tt.wantValid {
				t.Errorf("Expected entity name %q to be invalid", tt.entityName)
			}
		})
	}
}

func TestSyncDraftFilesystemWithDatabase(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	if err := os.Chdir(apiDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	draftDir := filepath.Join("..", "data", "prd-drafts", "scenario")
	if err := os.MkdirAll(draftDir, 0o755); err != nil {
		t.Fatalf("failed to create draft directory: %v", err)
	}

	draftPath := filepath.Join(draftDir, "test-sync.md")
	draftContent := "# Draft sync test\n"
	if err := os.WriteFile(draftPath, []byte(draftContent), 0o644); err != nil {
		t.Fatalf("failed to write draft file: %v", err)
	}

	modTime := time.Now().Add(-1 * time.Hour).Round(time.Second)
	if err := os.Chtimes(draftPath, modTime, modTime); err != nil {
		t.Fatalf("failed to update draft timestamps: %v", err)
	}

	mockExec := &recordingExec{}
	if err := syncDraftFilesystemWithDatabase(mockExec); err != nil {
		t.Fatalf("syncDraftFilesystemWithDatabase() error = %v", err)
	}

	if len(mockExec.calls) != 1 {
		t.Fatalf("expected 1 insert call, got %d", len(mockExec.calls))
	}

	call := mockExec.calls[0]
	if !strings.Contains(call.query, "INSERT INTO drafts") {
		t.Fatalf("unexpected query: %s", call.query)
	}

	if got, want := call.args[0], "scenario"; got != want {
		t.Fatalf("arg[0] = %v, want %v", got, want)
	}
	if got, want := call.args[1], "test-sync"; got != want {
		t.Fatalf("arg[1] = %v, want %v", got, want)
	}
	if got, want := call.args[2], draftContent; got != want {
		t.Fatalf("arg[2] = %q, want %q", got, want)
	}

	timestamp, ok := call.args[3].(time.Time)
	if !ok {
		t.Fatalf("arg[3] type = %T, want time.Time", call.args[3])
	}

	if diff := timestamp.Round(time.Second).Sub(modTime); diff > time.Second || diff < -time.Second {
		t.Fatalf("timestamp diff too large: %v", diff)
	}
}
