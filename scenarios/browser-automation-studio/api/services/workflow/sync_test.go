package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/database"
)

// TestIsWorkflowContent verifies the detection of workflow structure in JSON.
// This is critical for sync - we need to correctly identify which files are workflows
// vs other JSON files (config, package.json, etc).
func TestIsWorkflowContent(t *testing.T) {
	t.Run("valid workflow with top-level nodes", func(t *testing.T) {
		content := []byte(`{
			"id": "test",
			"nodes": [
				{"id": "1", "type": "click"},
				{"id": "2", "type": "input"}
			],
			"edges": []
		}`)

		assert.True(t, isWorkflowContent(content), "should detect workflow with top-level nodes array")
	})

	t.Run("valid workflow with nodes in flow_definition", func(t *testing.T) {
		content := []byte(`{
			"id": "test",
			"flow_definition": {
				"nodes": [{"id": "1", "type": "click"}],
				"edges": []
			}
		}`)

		assert.True(t, isWorkflowContent(content), "should detect workflow with nodes in flow_definition")
	})

	t.Run("valid workflow with nodes in definition_v2", func(t *testing.T) {
		content := []byte(`{
			"id": "test",
			"definition_v2": {
				"nodes": [{"id": "1", "action": {"type": "click"}}],
				"edges": []
			}
		}`)

		assert.True(t, isWorkflowContent(content), "should detect workflow with nodes in definition_v2")
	})

	t.Run("empty nodes array is still a workflow", func(t *testing.T) {
		content := []byte(`{"nodes": [], "edges": []}`)

		assert.True(t, isWorkflowContent(content), "empty nodes array should still be detected as workflow")
	})

	t.Run("invalid JSON returns false", func(t *testing.T) {
		content := []byte(`{not valid json`)

		assert.False(t, isWorkflowContent(content), "invalid JSON should return false")
	})

	t.Run("JSON without nodes returns false", func(t *testing.T) {
		content := []byte(`{"name": "package", "version": "1.0.0"}`)

		assert.False(t, isWorkflowContent(content), "JSON without nodes should return false")
	})

	t.Run("nodes as object (not array) returns false", func(t *testing.T) {
		content := []byte(`{"nodes": {"node1": "value"}}`)

		assert.False(t, isWorkflowContent(content), "nodes as object should return false")
	})

	t.Run("nodes as string returns false", func(t *testing.T) {
		content := []byte(`{"nodes": "not an array"}`)

		assert.False(t, isWorkflowContent(content), "nodes as string should return false")
	})

	t.Run("empty JSON returns false", func(t *testing.T) {
		content := []byte(`{}`)

		assert.False(t, isWorkflowContent(content), "empty JSON should return false")
	})

	t.Run("deeply nested nodes not detected", func(t *testing.T) {
		// We only check top-level, flow_definition, and definition_v2
		content := []byte(`{
			"some_other_key": {
				"nodes": [{"id": "1"}]
			}
		}`)

		assert.False(t, isWorkflowContent(content), "deeply nested nodes should not be detected")
	})
}

// TestIsNativeWorkflowFormat verifies detection of native format workflows.
// Native format has a valid UUID in the "id" field, distinguishing it from
// external workflows that may have arbitrary ID formats.
func TestIsNativeWorkflowFormat(t *testing.T) {
	t.Run("valid UUID id is native format", func(t *testing.T) {
		id := uuid.New().String()
		content := []byte(`{"id": "` + id + `", "nodes": []}`)

		assert.True(t, isNativeWorkflowFormat(content), "valid UUID should be detected as native format")
	})

	t.Run("UUID with project_id is native format", func(t *testing.T) {
		id := uuid.New().String()
		projectID := uuid.New().String()
		content := []byte(`{"id": "` + id + `", "project_id": "` + projectID + `", "nodes": []}`)

		assert.True(t, isNativeWorkflowFormat(content), "workflow with valid UUID and project_id should be native")
	})

	t.Run("non-UUID id is not native format", func(t *testing.T) {
		content := []byte(`{"id": "external-workflow-123", "nodes": []}`)

		assert.False(t, isNativeWorkflowFormat(content), "non-UUID id should not be native format")
	})

	t.Run("empty id is not native format", func(t *testing.T) {
		content := []byte(`{"id": "", "nodes": []}`)

		assert.False(t, isNativeWorkflowFormat(content), "empty id should not be native format")
	})

	t.Run("missing id field is not native format", func(t *testing.T) {
		content := []byte(`{"nodes": [], "edges": []}`)

		assert.False(t, isNativeWorkflowFormat(content), "missing id should not be native format")
	})

	t.Run("invalid JSON is not native format", func(t *testing.T) {
		content := []byte(`{not valid`)

		assert.False(t, isNativeWorkflowFormat(content), "invalid JSON should not be native format")
	})

	t.Run("numeric id is not native format", func(t *testing.T) {
		content := []byte(`{"id": 12345, "nodes": []}`)

		// JSON numbers for id should fail (we expect string)
		assert.False(t, isNativeWorkflowFormat(content), "numeric id should not be native format")
	})

	t.Run("partial UUID is not native format", func(t *testing.T) {
		content := []byte(`{"id": "550e8400-e29b-41d4", "nodes": []}`)

		assert.False(t, isNativeWorkflowFormat(content), "partial UUID should not be native format")
	})

	t.Run("UUID without hyphens is still valid", func(t *testing.T) {
		// UUID can be parsed with or without hyphens
		content := []byte(`{"id": "550e8400e29b41d4a716446655440000", "nodes": []}`)

		assert.True(t, isNativeWorkflowFormat(content), "UUID without hyphens should still be valid")
	})
}

// TestIsV1WorkflowFormat verifies detection of V1 vs V2 workflow formats.
// V1: node.type + node.data pattern (legacy)
// V2: node.action with typed action definitions (current)
func TestIsV1WorkflowFormat(t *testing.T) {
	t.Run("V1 format with type and data", func(t *testing.T) {
		content := []byte(`{
			"nodes": [
				{"id": "1", "type": "click", "data": {"selector": "button"}}
			]
		}`)

		assert.True(t, IsV1WorkflowFormat(content), "type+data pattern should be V1")
	})

	t.Run("V1 format with only type", func(t *testing.T) {
		content := []byte(`{
			"nodes": [
				{"id": "1", "type": "click"}
			]
		}`)

		assert.True(t, IsV1WorkflowFormat(content), "type without data should still be V1")
	})

	t.Run("V2 format with action", func(t *testing.T) {
		content := []byte(`{
			"nodes": [
				{"id": "1", "action": {"type": "ACTION_TYPE_CLICK", "click": {"selector": "button"}}}
			]
		}`)

		assert.False(t, IsV1WorkflowFormat(content), "node with action should be V2")
	})

	t.Run("mixed format with action is V2", func(t *testing.T) {
		// If a node has both type/data AND action, it's V2
		content := []byte(`{
			"nodes": [
				{"id": "1", "type": "click", "data": {}, "action": {"type": "ACTION_TYPE_CLICK"}}
			]
		}`)

		assert.False(t, IsV1WorkflowFormat(content), "node with action should be V2 even if it has type/data")
	})

	t.Run("empty nodes array is V2 (default)", func(t *testing.T) {
		content := []byte(`{"nodes": []}`)

		assert.False(t, IsV1WorkflowFormat(content), "empty workflow should default to V2")
	})

	t.Run("nodes in flow_definition", func(t *testing.T) {
		content := []byte(`{
			"flow_definition": {
				"nodes": [
					{"id": "1", "type": "click", "data": {}}
				]
			}
		}`)

		assert.True(t, IsV1WorkflowFormat(content), "should detect V1 in flow_definition")
	})

	t.Run("nodes in definition_v2", func(t *testing.T) {
		content := []byte(`{
			"definition_v2": {
				"nodes": [
					{"id": "1", "action": {"type": "ACTION_TYPE_NAVIGATE"}}
				]
			}
		}`)

		assert.False(t, IsV1WorkflowFormat(content), "should detect V2 in definition_v2")
	})

	t.Run("invalid JSON returns false", func(t *testing.T) {
		content := []byte(`{not valid`)

		assert.False(t, IsV1WorkflowFormat(content))
	})

	t.Run("non-workflow JSON returns false", func(t *testing.T) {
		content := []byte(`{"name": "package"}`)

		assert.False(t, IsV1WorkflowFormat(content))
	})
}

// TestHasNodesArray verifies the helper function for nodes array detection.
func TestHasNodesArray(t *testing.T) {
	t.Run("map with nodes array returns true", func(t *testing.T) {
		content := map[string]any{
			"nodes": []any{
				map[string]any{"id": "1"},
			},
		}

		assert.True(t, hasNodesArray(content))
	})

	t.Run("map with empty nodes array returns true", func(t *testing.T) {
		content := map[string]any{
			"nodes": []any{},
		}

		assert.True(t, hasNodesArray(content))
	})

	t.Run("map without nodes returns false", func(t *testing.T) {
		content := map[string]any{
			"edges": []any{},
		}

		assert.False(t, hasNodesArray(content))
	})

	t.Run("map with nodes as string returns false", func(t *testing.T) {
		content := map[string]any{
			"nodes": "not an array",
		}

		assert.False(t, hasNodesArray(content))
	})

	t.Run("empty map returns false", func(t *testing.T) {
		content := map[string]any{}

		assert.False(t, hasNodesArray(content))
	})
}

// TestNormalizeFolderPath verifies folder path normalization for sync consistency.
// The function ensures paths have a leading slash for internal consistency.
func TestNormalizeFolderPath(t *testing.T) {
	t.Run("empty path returns default root", func(t *testing.T) {
		// Empty paths normalize to the default workflow folder (root "/")
		assert.Equal(t, "/", normalizeFolderPath(""))
	})

	t.Run("removes trailing slash and adds leading slash", func(t *testing.T) {
		// normalizeFolderPath ensures consistent absolute-style paths
		assert.Equal(t, "/workflows/test", normalizeFolderPath("workflows/test/"))
	})

	t.Run("adds leading slash to relative path", func(t *testing.T) {
		// Paths without leading slash get one added for consistency
		assert.Equal(t, "/workflows/test", normalizeFolderPath("workflows/test"))
	})

	t.Run("handles root path", func(t *testing.T) {
		// Root path stays as root
		assert.Equal(t, "/", normalizeFolderPath("/"))
	})

	t.Run("preserves existing leading slash", func(t *testing.T) {
		// Paths already with leading slash are preserved
		assert.Equal(t, "/actions", normalizeFolderPath("/actions"))
	})
}

// TestDBState verifies the intermediate types used during sync.
func TestDBState_Lookup(t *testing.T) {
	// Test that map-based lookup works correctly for sync state tracking
	id1 := uuid.New()
	id2 := uuid.New()

	dbState := map[uuid.UUID]bool{
		id1: true,
		id2: true,
	}

	assert.True(t, dbState[id1], "should find existing ID")
	assert.True(t, dbState[id2], "should find existing ID")
	assert.False(t, dbState[uuid.New()], "should not find non-existent ID")
}

// TestWorkflowContentDetection_RealWorldExamples tests with realistic workflow structures.
func TestWorkflowContentDetection_RealWorldExamples(t *testing.T) {
	t.Run("React Flow workflow format", func(t *testing.T) {
		content := []byte(`{
			"id": "` + uuid.New().String() + `",
			"name": "Login Flow",
			"nodes": [
				{
					"id": "start-1",
					"type": "start",
					"position": {"x": 0, "y": 0},
					"data": {}
				},
				{
					"id": "action-1",
					"type": "navigate",
					"position": {"x": 100, "y": 0},
					"data": {"url": "https://example.com"}
				}
			],
			"edges": [
				{"source": "start-1", "target": "action-1"}
			]
		}`)

		require.True(t, isWorkflowContent(content), "should detect React Flow workflow")
		assert.True(t, isNativeWorkflowFormat(content), "should detect native format")
		assert.True(t, IsV1WorkflowFormat(content), "React Flow with type+data is V1")
	})

	t.Run("V2 proto-based workflow format", func(t *testing.T) {
		content := []byte(`{
			"id": "` + uuid.New().String() + `",
			"name": "Modern Flow",
			"nodes": [
				{
					"id": "node-1",
					"action": {
						"type": "ACTION_TYPE_NAVIGATE",
						"navigate": {"url": "https://example.com"}
					}
				}
			],
			"edges": []
		}`)

		require.True(t, isWorkflowContent(content), "should detect V2 workflow")
		assert.True(t, isNativeWorkflowFormat(content), "should detect native format")
		assert.False(t, IsV1WorkflowFormat(content), "should detect V2 format")
	})

	t.Run("external Playwright Codegen export", func(t *testing.T) {
		// Simulated external format - no UUID, just arbitrary ID
		content := []byte(`{
			"id": "playwright-codegen-export",
			"nodes": [
				{"id": "1", "type": "click", "selector": "button"}
			]
		}`)

		require.True(t, isWorkflowContent(content), "should detect workflow structure")
		assert.False(t, isNativeWorkflowFormat(content), "external ID should not be native")
	})

	t.Run("package.json is not a workflow", func(t *testing.T) {
		content := []byte(`{
			"name": "my-project",
			"version": "1.0.0",
			"dependencies": {
				"react": "^18.0.0"
			}
		}`)

		assert.False(t, isWorkflowContent(content), "package.json should not be detected as workflow")
	})

	t.Run("tsconfig.json is not a workflow", func(t *testing.T) {
		content := []byte(`{
			"compilerOptions": {
				"target": "ES2020",
				"module": "commonjs"
			},
			"include": ["src/**/*"]
		}`)

		assert.False(t, isWorkflowContent(content), "tsconfig.json should not be detected as workflow")
	})
}

// TestWorkflowContentEdgeCases tests edge cases in workflow detection.
func TestWorkflowContentEdgeCases(t *testing.T) {
	t.Run("very large nodes array", func(t *testing.T) {
		// Create content with many nodes
		nodes := make([]map[string]any, 100)
		for i := 0; i < 100; i++ {
			nodes[i] = map[string]any{"id": i, "type": "click"}
		}

		content, err := json.Marshal(map[string]any{"nodes": nodes})
		require.NoError(t, err)

		assert.True(t, isWorkflowContent(content), "should handle large nodes array")
	})

	t.Run("unicode in workflow", func(t *testing.T) {
		content := []byte(`{
			"id": "` + uuid.New().String() + `",
			"name": "日本語ワークフロー",
			"nodes": [{"id": "1", "type": "click", "data": {"text": "こんにちは"}}]
		}`)

		require.True(t, isWorkflowContent(content))
		assert.True(t, isNativeWorkflowFormat(content))
	})

	t.Run("null nodes field is not workflow", func(t *testing.T) {
		content := []byte(`{"nodes": null}`)

		assert.False(t, isWorkflowContent(content), "null nodes should not be workflow")
	})

	t.Run("whitespace-only content", func(t *testing.T) {
		content := []byte(`   `)

		assert.False(t, isWorkflowContent(content), "whitespace should not be workflow")
	})
}

// TestConvertExternalWorkflowReusesExistingID verifies that the converter reuses an existing ID when provided.
func TestConvertExternalWorkflowReusesExistingID(t *testing.T) {
	projectID := uuid.New()
	project := &database.WorkflowIndex{
		ID: projectID,
	}
	projectIndex := &database.ProjectIndex{
		ID:         projectID,
		Name:       "test-project",
		FolderPath: "/tmp/test-project",
	}

	existingID := uuid.New()
	content := []byte(`{
		"metadata": { "name": "test-workflow" },
		"nodes": [{"id": "1", "type": "click"}],
		"edges": []
	}`)

	result, err := ConvertExternalWorkflow(projectIndex, content, "workflows/test.json", &existingID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Workflow)

	assert.Equal(t, existingID.String(), result.Workflow.Id, "converter should reuse the existing ID")
	_ = project // silence unused variable warning
}

// TestConvertExternalWorkflowGeneratesNewIDWhenNoneProvided verifies that the converter generates a new ID when none is provided.
func TestConvertExternalWorkflowGeneratesNewIDWhenNoneProvided(t *testing.T) {
	projectID := uuid.New()
	projectIndex := &database.ProjectIndex{
		ID:         projectID,
		Name:       "test-project",
		FolderPath: "/tmp/test-project",
	}

	content := []byte(`{
		"metadata": { "name": "test-workflow" },
		"nodes": [{"id": "1", "type": "click"}],
		"edges": []
	}`)

	result, err := ConvertExternalWorkflow(projectIndex, content, "workflows/test.json", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Workflow)

	// Verify a valid UUID was generated
	parsedID, parseErr := uuid.Parse(result.Workflow.Id)
	require.NoError(t, parseErr, "should generate a valid UUID")
	assert.NotEqual(t, uuid.Nil, parsedID, "should not generate nil UUID")
}

// TestExtractExternalWorkflowMetadata verifies metadata extraction for deduplication lookup.
func TestExtractExternalWorkflowMetadata(t *testing.T) {
	t.Run("extracts name from metadata object", func(t *testing.T) {
		content := []byte(`{
			"metadata": { "name": "my-workflow", "description": "test" },
			"nodes": []
		}`)

		name, folderPath, err := ExtractExternalWorkflowMetadata(content, "actions/test.json")
		require.NoError(t, err)
		assert.Equal(t, "my-workflow", name)
		assert.Equal(t, "/actions", folderPath)
	})

	t.Run("extracts name from top-level field", func(t *testing.T) {
		content := []byte(`{
			"name": "top-level-name",
			"nodes": []
		}`)

		name, folderPath, err := ExtractExternalWorkflowMetadata(content, "workflows/test.json")
		require.NoError(t, err)
		assert.Equal(t, "top-level-name", name)
		assert.Equal(t, "/workflows", folderPath)
	})

	t.Run("falls back to filename when no name", func(t *testing.T) {
		content := []byte(`{"nodes": []}`)

		name, folderPath, err := ExtractExternalWorkflowMetadata(content, "my-workflow.json")
		require.NoError(t, err)
		assert.Equal(t, "my-workflow", name)
		assert.Equal(t, "/", folderPath)
	})

	t.Run("returns error for empty content", func(t *testing.T) {
		_, _, err := ExtractExternalWorkflowMetadata([]byte{}, "test.json")
		assert.Error(t, err)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, _, err := ExtractExternalWorkflowMetadata([]byte(`{not valid`), "test.json")
		assert.Error(t, err)
	})
}

// TestWorkflowNameKey verifies the key generation for workflow deduplication.
func TestWorkflowNameKey(t *testing.T) {
	projectID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	key := WorkflowNameKey(projectID, "my-workflow", "/actions")
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000:my-workflow:/actions", key)

	// Different folder paths should produce different keys
	key2 := WorkflowNameKey(projectID, "my-workflow", "/workflows")
	assert.NotEqual(t, key, key2)

	// Same parameters should produce same key
	key3 := WorkflowNameKey(projectID, "my-workflow", "/actions")
	assert.Equal(t, key, key3)
}

// TestDBStateWorkflowsByNameKey verifies the name key map is properly populated.
func TestDBStateWorkflowsByNameKey(t *testing.T) {
	projectID := uuid.New()
	workflowID := uuid.New()

	state := NewDBState()

	// Add a workflow with project ID
	wf := &database.WorkflowIndex{
		ID:         workflowID,
		ProjectID:  &projectID,
		Name:       "test-workflow",
		FolderPath: "/actions",
		FilePath:   "actions/test.json",
	}

	// Simulate what loadDBState does
	state.WorkflowsByID[wf.ID] = wf
	if wf.ProjectID != nil {
		key := WorkflowNameKey(*wf.ProjectID, wf.Name, wf.FolderPath)
		state.WorkflowsByNameKey[key] = wf
	}

	// Verify lookup works
	key := WorkflowNameKey(projectID, "test-workflow", "/actions")
	found, exists := state.WorkflowsByNameKey[key]
	assert.True(t, exists, "should find workflow by name key")
	assert.Equal(t, workflowID, found.ID)

	// Different name should not be found
	key2 := WorkflowNameKey(projectID, "different-workflow", "/actions")
	_, exists2 := state.WorkflowsByNameKey[key2]
	assert.False(t, exists2, "should not find workflow with different name")
}

// TestSyncDeduplicationIntegration simulates the full deduplication flow.
// This tests that when we have an existing workflow in the DB and encounter
// an external workflow with the same name/folder, we reuse the existing ID.
func TestSyncDeduplicationIntegration(t *testing.T) {
	projectID := uuid.New()
	existingWorkflowID := uuid.New()

	// Simulate existing workflow in DB
	existingWorkflow := &database.WorkflowIndex{
		ID:         existingWorkflowID,
		ProjectID:  &projectID,
		Name:       "open-demo-project",
		FolderPath: "/actions",
		FilePath:   "actions/open-demo-project.json",
		Version:    1,
	}

	// Create DBState as loadDBState would
	state := NewDBState()
	state.WorkflowsByID[existingWorkflow.ID] = existingWorkflow
	key := WorkflowNameKey(*existingWorkflow.ProjectID, existingWorkflow.Name, existingWorkflow.FolderPath)
	state.WorkflowsByNameKey[key] = existingWorkflow

	// Now simulate an external workflow file with the same name/folder
	externalContent := []byte(`{
		"metadata": { "name": "open-demo-project" },
		"nodes": [{"id": "1", "type": "navigate", "data": {"url": "https://example.com"}}],
		"edges": []
	}`)

	// Extract metadata (as syncWorkflowFile would)
	name, folderPath, err := ExtractExternalWorkflowMetadata(externalContent, "actions/open-demo-project.json")
	require.NoError(t, err)
	assert.Equal(t, "open-demo-project", name)

	// Check for existing workflow (as syncWorkflowFile would)
	normalizedFolderPath := normalizeFolderPath(folderPath)
	lookupKey := WorkflowNameKey(projectID, name, normalizedFolderPath)
	existing, found := state.WorkflowsByNameKey[lookupKey]

	// Should find the existing workflow
	require.True(t, found, "should find existing workflow by name/folder")
	assert.Equal(t, existingWorkflowID, existing.ID)

	// Convert with existing ID
	project := &database.ProjectIndex{
		ID:         projectID,
		Name:       "test-project",
		FolderPath: "/tmp/test-project",
	}
	result, convErr := ConvertExternalWorkflow(project, externalContent, "actions/open-demo-project.json", &existing.ID)
	require.NoError(t, convErr)

	// The converted workflow should have the SAME ID as existing
	assert.Equal(t, existingWorkflowID.String(), result.Workflow.Id, "converted workflow should reuse existing ID")
}

// TestSyncNoDuplicateWhenDifferentFolder verifies that workflows with same name but
// different folder paths are treated as distinct (not deduplicated).
func TestSyncNoDuplicateWhenDifferentFolder(t *testing.T) {
	projectID := uuid.New()
	existingWorkflowID := uuid.New()

	// Existing workflow in /actions folder
	existingWorkflow := &database.WorkflowIndex{
		ID:         existingWorkflowID,
		ProjectID:  &projectID,
		Name:       "test-workflow",
		FolderPath: "/actions",
		FilePath:   "actions/test.json",
	}

	// Create DBState
	state := NewDBState()
	state.WorkflowsByID[existingWorkflow.ID] = existingWorkflow
	key := WorkflowNameKey(*existingWorkflow.ProjectID, existingWorkflow.Name, existingWorkflow.FolderPath)
	state.WorkflowsByNameKey[key] = existingWorkflow

	// New external workflow with SAME name but DIFFERENT folder (/workflows instead of /actions)
	externalContent := []byte(`{
		"metadata": { "name": "test-workflow" },
		"nodes": [],
		"edges": []
	}`)

	name, folderPath, err := ExtractExternalWorkflowMetadata(externalContent, "workflows/test.json")
	require.NoError(t, err)
	assert.Equal(t, "test-workflow", name)
	assert.Equal(t, "/workflows", folderPath) // Different from /actions

	// Check for existing - should NOT find because folder is different
	normalizedFolderPath := normalizeFolderPath(folderPath)
	lookupKey := WorkflowNameKey(projectID, name, normalizedFolderPath)
	_, found := state.WorkflowsByNameKey[lookupKey]

	assert.False(t, found, "should NOT find workflow when folder path differs")

	// Convert without existing ID - should generate new ID
	project := &database.ProjectIndex{
		ID:         projectID,
		Name:       "test-project",
		FolderPath: "/tmp/test-project",
	}
	result, convErr := ConvertExternalWorkflow(project, externalContent, "workflows/test.json", nil)
	require.NoError(t, convErr)

	// Should have a NEW ID, not the existing one
	assert.NotEqual(t, existingWorkflowID.String(), result.Workflow.Id, "should generate new ID for different folder")
}

// TestMockRepositoryGetWorkflowByNameInProject verifies the mock implements the method correctly.
func TestMockRepositoryGetWorkflowByNameInProject(t *testing.T) {
	ctx := context.Background()
	mock := NewMockWorkflowSyncRepository()

	projectID := uuid.New()
	workflowID := uuid.New()

	// Add a workflow
	wf := &database.WorkflowIndex{
		ID:         workflowID,
		ProjectID:  &projectID,
		Name:       "test-workflow",
		FolderPath: "/actions",
	}
	mock.AddWorkflow(wf)

	// Should find it
	found, err := mock.GetWorkflowByNameInProject(ctx, projectID, "test-workflow", "/actions")
	require.NoError(t, err)
	assert.Equal(t, workflowID, found.ID)

	// Should not find with different name
	_, err = mock.GetWorkflowByNameInProject(ctx, projectID, "other-workflow", "/actions")
	assert.ErrorIs(t, err, database.ErrNotFound)

	// Should not find with different folder
	_, err = mock.GetWorkflowByNameInProject(ctx, projectID, "test-workflow", "/workflows")
	assert.ErrorIs(t, err, database.ErrNotFound)

	// Should not find with different project
	otherProjectID := uuid.New()
	_, err = mock.GetWorkflowByNameInProject(ctx, otherProjectID, "test-workflow", "/actions")
	assert.ErrorIs(t, err, database.ErrNotFound)
}
