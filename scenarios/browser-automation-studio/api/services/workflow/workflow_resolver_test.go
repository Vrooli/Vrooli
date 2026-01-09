package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestValidateAndNormalizePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		wantResult  string
	}{
		{
			name:       "simple path",
			input:      "actions/test.json",
			wantErr:    false,
			wantResult: "actions/test.json",
		},
		{
			name:       "path with leading slash",
			input:      "/actions/test.json",
			wantErr:    false,
			wantResult: "actions/test.json",
		},
		{
			name:       "path with leading spaces",
			input:      "  actions/test.json  ",
			wantErr:    false,
			wantResult: "actions/test.json",
		},
		{
			name:    "empty path",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "path traversal attempt",
			input:   "../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "path traversal in middle",
			input:   "actions/../../../etc/passwd",
			wantErr: true,
		},
		{
			name:       "path with leading slash stripped becomes relative",
			input:      "/etc/passwd",
			wantErr:    false,
			wantResult: "etc/passwd", // Leading slash stripped, becomes valid relative path
		},
		{
			name:    "dot only",
			input:   ".",
			wantErr: true,
		},
		{
			name:       "nested path",
			input:      "flows/subdir/flow.workflow.json",
			wantErr:    false,
			wantResult: "flows/subdir/flow.workflow.json",
		},
		{
			name:       "path with backslash (windows style on linux)",
			input:      "actions\\test.json",
			wantErr:    false,
			wantResult: "actions\\test.json", // Backslash not converted on Linux (only forward slashes)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateAndNormalizePath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateAndNormalizePath(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("validateAndNormalizePath(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.wantResult {
				t.Errorf("validateAndNormalizePath(%q) = %q, want %q", tt.input, result, tt.wantResult)
			}
		})
	}
}

func TestResolveFromFilesystemRoot(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a test workflow file
	actionsDir := filepath.Join(tmpDir, "actions")
	if err := os.MkdirAll(actionsDir, 0o755); err != nil {
		t.Fatalf("Failed to create actions dir: %v", err)
	}

	workflowID := uuid.New().String()
	workflowContent := `{
  "id": "` + workflowID + `",
  "name": "test-workflow",
  "description": "A test workflow",
  "version": 1,
  "definition_v2": {
    "nodes": [],
    "edges": []
  }
}`
	workflowPath := filepath.Join(actionsDir, "test.workflow.json")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0o644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Create a WorkflowService with nil repo (not needed for filesystem resolution)
	svc := &WorkflowService{}

	tests := []struct {
		name         string
		projectRoot  string
		rel          string
		workflowPath string
		wantErr      bool
		wantName     string
	}{
		{
			name:         "valid workflow file",
			projectRoot:  tmpDir,
			rel:          "actions/test.workflow.json",
			workflowPath: "actions/test.workflow.json",
			wantErr:      false,
			wantName:     "test-workflow",
		},
		{
			name:         "non-existent file",
			projectRoot:  tmpDir,
			rel:          "actions/nonexistent.json",
			workflowPath: "actions/nonexistent.json",
			wantErr:      true,
		},
		{
			name:         "relative project root rejected",
			projectRoot:  "relative/path",
			rel:          "actions/test.workflow.json",
			workflowPath: "actions/test.workflow.json",
			wantErr:      true,
		},
		{
			name:         "path traversal blocked",
			projectRoot:  tmpDir,
			rel:          "../etc/passwd",
			workflowPath: "../etc/passwd",
			wantErr:      true,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.resolveFromFilesystemRoot(ctx, tt.projectRoot, tt.rel, tt.workflowPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("resolveFromFilesystemRoot() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("resolveFromFilesystemRoot() unexpected error: %v", err)
				return
			}
			if result == nil {
				t.Error("resolveFromFilesystemRoot() returned nil workflow")
				return
			}
			if result.Name != tt.wantName {
				t.Errorf("resolveFromFilesystemRoot() got name %q, want %q", result.Name, tt.wantName)
			}
		})
	}
}

func TestResolveFromFilesystemRoot_NestedPaths(t *testing.T) {
	// Test nested directory structures
	tmpDir := t.TempDir()

	// Create nested structure: actions/common/dismiss-tutorial.workflow.json
	nestedDir := filepath.Join(tmpDir, "actions", "common")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}

	workflowID := uuid.New().String()
	workflowContent := `{
  "id": "` + workflowID + `",
  "name": "dismiss-tutorial",
  "description": "Dismisses the tutorial modal",
  "version": 1,
  "definition_v2": {
    "nodes": [],
    "edges": []
  }
}`
	workflowPath := filepath.Join(nestedDir, "dismiss-tutorial.workflow.json")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0o644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	svc := &WorkflowService{}
	ctx := context.Background()

	result, err := svc.resolveFromFilesystemRoot(ctx, tmpDir, "actions/common/dismiss-tutorial.workflow.json", "actions/common/dismiss-tutorial.workflow.json")
	if err != nil {
		t.Fatalf("resolveFromFilesystemRoot() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("resolveFromFilesystemRoot() returned nil workflow")
	}
	if result.Name != "dismiss-tutorial" {
		t.Errorf("resolveFromFilesystemRoot() got name %q, want 'dismiss-tutorial'", result.Name)
	}
}
