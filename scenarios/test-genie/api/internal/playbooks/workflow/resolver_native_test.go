package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeResolverBasicWorkflow(t *testing.T) {
	// Create temp scenario structure
	tempDir := t.TempDir()

	// Create directories
	dirs := []string{
		filepath.Join(tempDir, "test", "playbooks"),
		filepath.Join(tempDir, "requirements"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	// Create a simple workflow
	workflowContent := `{
		"metadata": {"description": "Test workflow"},
		"nodes": [{"id": "n1", "type": "navigate", "data": {"url": "http://example.com"}}],
		"edges": []
	}`
	workflowPath := filepath.Join(tempDir, "test", "playbooks", "test.json")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	// Resolve
	resolver := NewNativeResolver(tempDir)
	result, err := resolver.ResolveWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("ResolveWorkflow failed: %v", err)
	}

	// Verify
	nodes, ok := result["nodes"].([]any)
	if !ok || len(nodes) != 1 {
		t.Errorf("expected 1 node, got %v", result["nodes"])
	}

	// Verify default reset was added
	metadata := result["metadata"].(map[string]any)
	if metadata["reset"] != "none" {
		t.Errorf("expected default reset to be 'none', got %v", metadata["reset"])
	}
}

func TestNativeResolverWorkflowWithSubflowPath(t *testing.T) {
	// Tests that workflowPath references are passed through unchanged
	// (BAS resolves these at execution time)
	tempDir := t.TempDir()

	dirs := []string{
		filepath.Join(tempDir, "test", "playbooks"),
		filepath.Join(tempDir, "bas", "actions"),
	}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0o755)
	}

	// Create workflow with subflow using workflowPath (new format)
	workflowContent := `{
		"metadata": {"description": "Test workflow"},
		"nodes": [
			{
				"id": "n1",
				"type": "subflow",
				"data": {
					"label": "Call helper",
					"workflowPath": "actions/helper.json",
					"params": {
						"projectName": "${@params/projectName}",
						"projectId": "${@params/projectId}"
					}
				}
			}
		],
		"edges": []
	}`
	workflowPath := filepath.Join(tempDir, "test", "playbooks", "test.json")
	os.WriteFile(workflowPath, []byte(workflowContent), 0o644)

	// Resolve
	resolver := NewNativeResolver(tempDir)
	result, err := resolver.ResolveWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("ResolveWorkflow failed: %v", err)
	}

	// Verify workflowPath is passed through unchanged (BAS will resolve it)
	nodes := result["nodes"].([]any)
	firstNode := nodes[0].(map[string]any)
	data := firstNode["data"].(map[string]any)

	if data["workflowPath"] != "actions/helper.json" {
		t.Errorf("expected workflowPath to be passed through, got %v", data["workflowPath"])
	}

	// Verify params with ${@params/...} are passed through unchanged
	params := data["params"].(map[string]any)
	if params["projectName"] != "${@params/projectName}" {
		t.Errorf("expected params to be passed through, got %v", params["projectName"])
	}
}

func TestNativeResolverSelectorTokensPassedThrough(t *testing.T) {
	// Verifies that @selector/ tokens are NOT resolved by test-genie
	// and are passed through to BAS for resolution.
	// BAS handles selector resolution natively in compiler.go
	tempDir := t.TempDir()

	dirs := []string{
		filepath.Join(tempDir, "test", "playbooks"),
	}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0o755)
	}

	// Create workflow with @selector/ tokens
	workflowContent := `{
		"metadata": {},
		"nodes": [
			{"id": "n1", "type": "click", "data": {"selector": "@selector/button.submit"}},
			{"id": "n2", "type": "type", "data": {"selector": "@selector/input.name", "text": "test"}}
		],
		"edges": []
	}`
	workflowPath := filepath.Join(tempDir, "test", "playbooks", "test.json")
	os.WriteFile(workflowPath, []byte(workflowContent), 0o644)

	// Resolve
	resolver := NewNativeResolver(tempDir)
	result, err := resolver.ResolveWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("ResolveWorkflow failed: %v", err)
	}

	// Verify @selector/ tokens are NOT resolved (passed through for BAS to handle)
	nodes := result["nodes"].([]any)

	n1Data := nodes[0].(map[string]any)["data"].(map[string]any)
	if n1Data["selector"] != "@selector/button.submit" {
		t.Errorf("expected @selector/ token to be passed through unchanged, got %v", n1Data["selector"])
	}

	n2Data := nodes[1].(map[string]any)["data"].(map[string]any)
	if n2Data["selector"] != "@selector/input.name" {
		t.Errorf("expected @selector/ token to be passed through unchanged, got %v", n2Data["selector"])
	}
}

func TestNativeResolverLoadSeedState(t *testing.T) {
	tempDir := t.TempDir()

	// Create seed state directory and file at the correct path (coverage/runtime/seed-state.json)
	seedDir := filepath.Join(tempDir, "coverage", "runtime")
	os.MkdirAll(seedDir, 0o755)

	seedContent := `{
		"projectId": "proj-123",
		"projectName": "Demo Project",
		"workflowId": "wf-456"
	}`
	seedPath := filepath.Join(seedDir, "seed-state.json")
	os.WriteFile(seedPath, []byte(seedContent), 0o644)

	// Load seed state
	resolver := NewNativeResolver(tempDir)
	seedState, err := resolver.LoadSeedState()
	if err != nil {
		t.Fatalf("LoadSeedState failed: %v", err)
	}

	// Verify
	if seedState["projectId"] != "proj-123" {
		t.Errorf("expected projectId=proj-123, got %v", seedState["projectId"])
	}
	if seedState["projectName"] != "Demo Project" {
		t.Errorf("expected projectName=Demo Project, got %v", seedState["projectName"])
	}
}

func TestNativeResolverLoadSeedStateNotExists(t *testing.T) {
	tempDir := t.TempDir()

	// Don't create any seed state file

	// Load seed state
	resolver := NewNativeResolver(tempDir)
	seedState, err := resolver.LoadSeedState()
	if err != nil {
		t.Fatalf("LoadSeedState should not fail for missing file: %v", err)
	}

	// Verify empty map returned
	if len(seedState) != 0 {
		t.Errorf("expected empty seed state, got %v", seedState)
	}
}

func TestNativeResolverRelativePath(t *testing.T) {
	tempDir := t.TempDir()

	dirs := []string{
		filepath.Join(tempDir, "bas", "cases", "01-foundation"),
	}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0o755)
	}

	// Create workflow
	workflowContent := `{
		"metadata": {"description": "Test"},
		"nodes": [],
		"edges": []
	}`
	workflowPath := filepath.Join(tempDir, "bas", "cases", "01-foundation", "test.json")
	os.WriteFile(workflowPath, []byte(workflowContent), 0o644)

	// Resolve with relative path
	resolver := NewNativeResolver(tempDir)
	result, err := resolver.ResolveWorkflow("bas/cases/01-foundation/test.json")
	if err != nil {
		t.Fatalf("ResolveWorkflow with relative path failed: %v", err)
	}

	metadata := result["metadata"].(map[string]any)
	if metadata["description"] != "Test" {
		t.Errorf("expected description=Test, got %v", metadata["description"])
	}
}

func TestNativeResolverInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	dirs := []string{
		filepath.Join(tempDir, "test", "playbooks"),
	}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0o755)
	}

	// Create invalid JSON file
	workflowPath := filepath.Join(tempDir, "test", "playbooks", "invalid.json")
	os.WriteFile(workflowPath, []byte("{ not valid json"), 0o644)

	resolver := NewNativeResolver(tempDir)
	_, err := resolver.ResolveWorkflow(workflowPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestNativeResolverFileNotFound(t *testing.T) {
	tempDir := t.TempDir()

	resolver := NewNativeResolver(tempDir)
	_, err := resolver.ResolveWorkflow("nonexistent.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestNativeResolverPreservesExistingReset(t *testing.T) {
	tempDir := t.TempDir()

	dirs := []string{
		filepath.Join(tempDir, "test", "playbooks"),
	}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0o755)
	}

	// Create workflow with explicit reset value
	workflowContent := `{
		"metadata": {"description": "Test", "reset": "full"},
		"nodes": [],
		"edges": []
	}`
	workflowPath := filepath.Join(tempDir, "test", "playbooks", "test.json")
	os.WriteFile(workflowPath, []byte(workflowContent), 0o644)

	resolver := NewNativeResolver(tempDir)
	result, err := resolver.ResolveWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("ResolveWorkflow failed: %v", err)
	}

	// Verify existing reset is preserved
	metadata := result["metadata"].(map[string]any)
	if metadata["reset"] != "full" {
		t.Errorf("expected reset=full to be preserved, got %v", metadata["reset"])
	}
}
