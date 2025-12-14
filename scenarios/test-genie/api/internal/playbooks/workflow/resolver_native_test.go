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
		filepath.Join(tempDir, "test", "playbooks", "__subflows"),
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
}

func TestNativeResolverFixtureInlining(t *testing.T) {
	tempDir := t.TempDir()

	// Create directories
	dirs := []string{
		filepath.Join(tempDir, "test", "playbooks", "__subflows"),
		filepath.Join(tempDir, "requirements"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	// Create fixture
	fixtureContent := `{
		"metadata": {
			"fixture_id": "setup-state",
			"description": "Sets up initial state",
			"requirements": ["REQ-001"]
		},
		"nodes": [{"id": "setup-1", "type": "navigate", "data": {"url": "http://setup.com"}}],
		"edges": []
	}`
	fixturePath := filepath.Join(tempDir, "test", "playbooks", "__subflows", "setup-state.json")
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	// Create workflow that references fixture
	workflowContent := `{
		"metadata": {"description": "Test workflow"},
		"nodes": [
			{"id": "n1", "type": "subflow", "data": {"workflowId": "@fixture/setup-state"}},
			{"id": "n2", "type": "click", "data": {"selector": "#button"}}
		],
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

	// Verify fixture was inlined
	nodes := result["nodes"].([]any)
	firstNode := nodes[0].(map[string]any)
	data := firstNode["data"].(map[string]any)

	if _, hasWorkflowId := data["workflowId"]; hasWorkflowId {
		t.Error("workflowId should have been replaced")
	}

	definition, ok := data["workflowDefinition"].(map[string]any)
	if !ok {
		t.Fatal("expected workflowDefinition to be set")
	}

	defNodes := definition["nodes"].([]any)
	if len(defNodes) != 1 {
		t.Errorf("expected 1 node in inlined fixture, got %d", len(defNodes))
	}

	// Verify requirements were collected
	metadata := result["metadata"].(map[string]any)
	fixtureReqs, ok := metadata["requirementsFromFixtures"].([]any)
	if !ok || len(fixtureReqs) != 1 {
		t.Errorf("expected 1 fixture requirement, got %v", fixtureReqs)
	}
}

func TestNativeResolverFixtureParameters(t *testing.T) {
	tempDir := t.TempDir()

	// Create directories
	dirs := []string{
		filepath.Join(tempDir, "test", "playbooks", "__subflows"),
		filepath.Join(tempDir, "requirements"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	// Create parameterized fixture
	fixtureContent := `{
		"metadata": {
			"fixture_id": "navigate-to",
			"description": "Navigates to a URL",
			"parameters": [
				{"name": "url", "type": "string", "required": true},
				{"name": "timeout", "type": "number", "default": 30000}
			]
		},
		"nodes": [{"id": "nav", "type": "navigate", "data": {"url": "${fixture.url}", "timeout": "${fixture.timeout}"}}],
		"edges": []
	}`
	fixturePath := filepath.Join(tempDir, "test", "playbooks", "__subflows", "navigate-to.json")
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	// Create workflow with fixture parameters
	workflowContent := `{
		"metadata": {"description": "Test"},
		"nodes": [{"id": "n1", "type": "subflow", "data": {"workflowId": "@fixture/navigate-to(url=\"http://test.com\")"}}],
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

	// Verify parameter substitution
	nodes := result["nodes"].([]any)
	firstNode := nodes[0].(map[string]any)
	data := firstNode["data"].(map[string]any)
	definition := data["workflowDefinition"].(map[string]any)
	defNodes := definition["nodes"].([]any)
	navNode := defNodes[0].(map[string]any)
	navData := navNode["data"].(map[string]any)

	if navData["url"] != "http://test.com" {
		t.Errorf("expected url to be substituted, got %v", navData["url"])
	}
	if navData["timeout"] != "30000" {
		t.Errorf("expected default timeout, got %v", navData["timeout"])
	}
}

func TestNativeResolverMissingRequiredParameter(t *testing.T) {
	tempDir := t.TempDir()

	// Create directories
	dirs := []string{
		filepath.Join(tempDir, "test", "playbooks", "__subflows"),
		filepath.Join(tempDir, "requirements"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	// Create fixture with required parameter
	fixtureContent := `{
		"metadata": {
			"fixture_id": "login",
			"parameters": [{"name": "username", "type": "string", "required": true}]
		},
		"nodes": [{"id": "n", "type": "type", "data": {"text": "${fixture.username}"}}],
		"edges": []
	}`
	if err := os.WriteFile(
		filepath.Join(tempDir, "test", "playbooks", "__subflows", "login.json"),
		[]byte(fixtureContent), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	// Create workflow without required parameter
	workflowContent := `{
		"metadata": {},
		"nodes": [{"id": "n1", "type": "subflow", "data": {"workflowId": "@fixture/login"}}],
		"edges": []
	}`
	workflowPath := filepath.Join(tempDir, "test", "playbooks", "test.json")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	// Should fail
	resolver := NewNativeResolver(tempDir)
	_, err := resolver.ResolveWorkflow(workflowPath)
	if err == nil {
		t.Error("expected error for missing required parameter")
	}
}

func TestNativeResolverUnknownFixture(t *testing.T) {
	tempDir := t.TempDir()

	dirs := []string{
		filepath.Join(tempDir, "test", "playbooks", "__subflows"),
		filepath.Join(tempDir, "requirements"),
	}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0o755)
	}

	workflowContent := `{
		"metadata": {},
		"nodes": [{"id": "n1", "data": {"workflowId": "@fixture/nonexistent"}}],
		"edges": []
	}`
	workflowPath := filepath.Join(tempDir, "test", "playbooks", "test.json")
	os.WriteFile(workflowPath, []byte(workflowContent), 0o644)

	resolver := NewNativeResolver(tempDir)
	_, err := resolver.ResolveWorkflow(workflowPath)
	if err == nil {
		t.Error("expected error for unknown fixture")
	}
}

func TestParseFixtureReference(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedSlug string
		expectedArgs map[string]string
		expectError  bool
	}{
		{
			name:         "simple",
			input:        "@fixture/setup",
			expectedSlug: "setup",
			expectedArgs: map[string]string{},
		},
		{
			name:         "with single arg",
			input:        "@fixture/nav(url=\"http://test.com\")",
			expectedSlug: "nav",
			expectedArgs: map[string]string{"url": "http://test.com"},
		},
		{
			name:         "with multiple args",
			input:        "@fixture/login(user=\"admin\", pass=\"secret\")",
			expectedSlug: "login",
			expectedArgs: map[string]string{"user": "admin", "pass": "secret"},
		},
		{
			name:         "single quotes",
			input:        "@fixture/test(key='value')",
			expectedSlug: "test",
			expectedArgs: map[string]string{"key": "value"},
		},
		{
			name:         "no quotes",
			input:        "@fixture/test(num=123)",
			expectedSlug: "test",
			expectedArgs: map[string]string{"num": "123"},
		},
		{
			name:        "invalid",
			input:       "not-a-fixture",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			slug, args, err := parseFixtureReference(tc.input)
			if tc.expectError {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if slug != tc.expectedSlug {
				t.Errorf("expected slug %q, got %q", tc.expectedSlug, slug)
			}
			if len(args) != len(tc.expectedArgs) {
				t.Errorf("expected %d args, got %d", len(tc.expectedArgs), len(args))
			}
			for k, v := range tc.expectedArgs {
				if args[k] != v {
					t.Errorf("expected args[%q]=%q, got %q", k, v, args[k])
				}
			}
		})
	}
}

func TestCoerceParameterValue(t *testing.T) {
	tests := []struct {
		name        string
		param       FixtureParameter
		value       any
		expected    any
		expectError bool
	}{
		{
			name:     "string",
			param:    FixtureParameter{Type: "string"},
			value:    "hello",
			expected: "hello",
		},
		{
			name:     "number int",
			param:    FixtureParameter{Type: "number"},
			value:    "42",
			expected: int64(42),
		},
		{
			name:     "number float",
			param:    FixtureParameter{Type: "number"},
			value:    "3.14",
			expected: 3.14,
		},
		{
			name:        "number invalid",
			param:       FixtureParameter{Type: "number"},
			value:       "abc",
			expectError: true,
		},
		{
			name:     "boolean true",
			param:    FixtureParameter{Type: "boolean"},
			value:    "true",
			expected: true,
		},
		{
			name:     "boolean false",
			param:    FixtureParameter{Type: "boolean"},
			value:    "0",
			expected: false,
		},
		{
			name:        "boolean invalid",
			param:       FixtureParameter{Type: "boolean"},
			value:       "maybe",
			expectError: true,
		},
		{
			name:     "enum valid",
			param:    FixtureParameter{Type: "enum", EnumValues: []string{"a", "b", "c"}},
			value:    "b",
			expected: "b",
		},
		{
			name:        "enum invalid",
			param:       FixtureParameter{Type: "enum", EnumValues: []string{"a", "b", "c"}},
			value:       "d",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := coerceParameterValue(tc.param, tc.value)
			if tc.expectError {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("expected %v (%T), got %v (%T)", tc.expected, tc.expected, result, result)
			}
		})
	}
}

// TestSelectorTokensPassedThrough verifies that @selector/ tokens are NOT resolved
// by test-genie and are passed through to BAS for resolution.
// BAS handles selector resolution natively in compiler.go:939-1073
func TestSelectorTokensPassedThrough(t *testing.T) {
	tempDir := t.TempDir()

	// Create directories
	dirs := []string{
		filepath.Join(tempDir, "test", "playbooks", "__subflows"),
		filepath.Join(tempDir, "requirements"),
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

func TestMergeResetValues(t *testing.T) {
	tests := []struct {
		a, b     string
		expected string
	}{
		{"", "", ""},
		{"none", "", "none"},
		{"", "full", "full"},
		{"none", "none", "none"},
		{"none", "full", "full"},
		{"full", "none", "full"},
		{"full", "full", "full"},
	}

	for _, tc := range tests {
		result := mergeResetValues(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("mergeResetValues(%q, %q) = %q, expected %q", tc.a, tc.b, result, tc.expected)
		}
	}
}
