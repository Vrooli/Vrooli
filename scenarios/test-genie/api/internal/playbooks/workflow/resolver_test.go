package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolverResolveDirectFile(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	workflowContent := `{
		"nodes": [{"id": "1", "type": "navigate"}],
		"edges": []
	}`
	workflowPath := filepath.Join(scenarioDir, "test.json")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	resolver := NewResolver(tempDir, scenarioDir)
	doc, err := resolver.Resolve(context.Background(), workflowPath)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	nodes, ok := doc["nodes"].([]any)
	if !ok || len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %v", doc["nodes"])
	}
}

func TestResolverResolveRelativePath(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	workflowContent := `{"nodes": [], "edges": []}`
	workflowPath := filepath.Join(scenarioDir, "workflows", "test.json")
	if err := os.MkdirAll(filepath.Dir(workflowPath), 0o755); err != nil {
		t.Fatalf("failed to create workflows dir: %v", err)
	}
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	resolver := NewResolver(tempDir, scenarioDir)
	// Use relative path
	doc, err := resolver.Resolve(context.Background(), "workflows/test.json")
	if err != nil {
		t.Fatalf("expected success with relative path, got error: %v", err)
	}

	if doc["nodes"] == nil {
		t.Error("expected nodes key in workflow")
	}
}

func TestResolverResolveNotFound(t *testing.T) {
	tempDir := t.TempDir()
	resolver := NewResolver(tempDir, tempDir)

	_, err := resolver.Resolve(context.Background(), "/nonexistent/workflow.json")
	if err == nil {
		t.Fatal("expected error for missing workflow")
	}
}

func TestResolverResolveInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "bad.json")
	if err := os.WriteFile(workflowPath, []byte(`{invalid`), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	resolver := NewResolver(tempDir, tempDir)
	_, err := resolver.Resolve(context.Background(), workflowPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFindPython(t *testing.T) {
	// This test verifies findPython returns a non-empty string if Python is installed
	// or empty if not. Either result is valid depending on the environment.
	result := findPython()
	// Just verify it doesn't panic and returns a string
	_ = result
}

func TestParseWorkflow(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid workflow",
			input:   `{"nodes": [], "edges": []}`,
			wantErr: false,
		},
		{
			name:    "valid with nested data",
			input:   `{"nodes": [{"id": "1", "data": {"url": "http://example.com"}}]}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty object",
			input:   `{}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseWorkflow([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
