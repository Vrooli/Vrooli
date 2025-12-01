package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestContractPlanCompilerProducesContractsPlan(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] compiles workflow with multiple steps", func(t *testing.T) {
		execID := uuid.New()
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "contract-plan",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "nav", "type": "navigate", "data": map[string]any{"url": "https://example.com"}, "position": map[string]any{"x": 0, "y": 0}},
					map[string]any{"id": "click", "type": "click", "data": map[string]any{"selector": "#btn"}, "position": map[string]any{"x": 200, "y": 0}},
					map[string]any{"id": "assert", "type": "assert", "data": map[string]any{"selector": "#btn", "text": "Submit"}, "position": map[string]any{"x": 400, "y": 0}},
				},
				"edges": []any{
					map[string]any{"id": "nav-click", "source": "nav", "target": "click"},
					map[string]any{"id": "click-assert", "source": "click", "target": "assert"},
				},
			},
		}

		compiler := &ContractPlanCompiler{}
		plan, instructions, err := compiler.Compile(context.Background(), execID, workflow)

		require.NoError(t, err)
		assert.Equal(t, execID, plan.ExecutionID)
		assert.Equal(t, workflow.ID, plan.WorkflowID)
		assert.Equal(t, contracts.ExecutionPlanSchemaVersion, plan.SchemaVersion)
		assert.Equal(t, contracts.PayloadVersion, plan.PayloadVersion)
		assert.Len(t, instructions, 3)
		assert.Equal(t, "nav", instructions[0].NodeID)
		assert.Equal(t, "click", instructions[1].NodeID)
		assert.Equal(t, "assert", instructions[2].NodeID)
		assert.NotNil(t, plan.Graph)
		assert.Len(t, plan.Graph.Steps, 3)
		assert.False(t, plan.CreatedAt.IsZero())
		assert.True(t, plan.CreatedAt.Before(time.Now().UTC().Add(5*time.Second)))
	})
}

func TestContractPlanCompiler_ErrorCases(t *testing.T) {
	compiler := &ContractPlanCompiler{}
	ctx := context.Background()
	execID := uuid.New()

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns error for nil workflow", func(t *testing.T) {
		plan, instructions, err := compiler.Compile(ctx, execID, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
		assert.Empty(t, instructions)
		assert.Equal(t, uuid.Nil, plan.ExecutionID)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns error for workflow without flow definition", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:             uuid.New(),
			Name:           "no-definition",
			FlowDefinition: nil,
		}

		plan, instructions, err := compiler.Compile(ctx, execID, workflow)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "flow_definition")
		assert.Empty(t, instructions)
		assert.Equal(t, uuid.Nil, plan.ExecutionID)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns error for unsupported step type", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "invalid-step-type",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "bad", "type": "unsupported_step_type_xyz", "data": map[string]any{}},
				},
				"edges": []any{},
			},
		}

		plan, instructions, err := compiler.Compile(ctx, execID, workflow)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported step type")
		assert.Empty(t, instructions)
		assert.Equal(t, uuid.Nil, plan.ExecutionID)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] returns error for empty step type", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "empty-step-type",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "bad", "type": "", "data": map[string]any{}},
				},
				"edges": []any{},
			},
		}

		plan, instructions, err := compiler.Compile(ctx, execID, workflow)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "step type cannot be empty")
		assert.Empty(t, instructions)
		assert.Equal(t, uuid.Nil, plan.ExecutionID)
	})
}

func TestContractPlanCompiler_EdgeCases(t *testing.T) {
	compiler := &ContractPlanCompiler{}
	ctx := context.Background()
	execID := uuid.New()

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] handles empty nodes list", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "empty-workflow",
			FlowDefinition: database.JSONMap{
				"nodes": []any{},
				"edges": []any{},
			},
		}

		plan, instructions, err := compiler.Compile(ctx, execID, workflow)

		require.NoError(t, err)
		assert.Equal(t, execID, plan.ExecutionID)
		assert.Equal(t, workflow.ID, plan.WorkflowID)
		assert.Empty(t, instructions)
		// Empty workflow should still produce valid plan structure
		assert.Equal(t, contracts.ExecutionPlanSchemaVersion, plan.SchemaVersion)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] handles single node workflow", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "single-node",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "nav", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				},
				"edges": []any{},
			},
		}

		plan, instructions, err := compiler.Compile(ctx, execID, workflow)

		require.NoError(t, err)
		assert.Len(t, instructions, 1)
		assert.Equal(t, "nav", instructions[0].NodeID)
		assert.Equal(t, "navigate", instructions[0].Type)
		assert.Equal(t, "https://example.com", instructions[0].Params["url"])
		require.Len(t, plan.Instructions, 1)
		assert.Equal(t, instructions[0].NodeID, plan.Instructions[0].NodeID)
		assert.Equal(t, instructions[0].Type, plan.Instructions[0].Type)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] preserves node parameters in instructions", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "params-test",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{
						"id":   "type-node",
						"type": "type",
						"data": map[string]any{
							"selector": "#input",
							"text":     "Hello World",
							"delay":    float64(100),
						},
					},
				},
				"edges": []any{},
			},
		}

		plan, instructions, err := compiler.Compile(ctx, execID, workflow)

		require.NoError(t, err)
		require.Len(t, instructions, 1)
		assert.Equal(t, "#input", instructions[0].Params["selector"])
		assert.Equal(t, "Hello World", instructions[0].Params["text"])
		assert.Equal(t, float64(100), instructions[0].Params["delay"])
		// Verify plan also has instructions
		assert.Len(t, plan.Instructions, 1)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] sets correct instruction indices", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "index-test",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
					map[string]any{"id": "b", "type": "click", "data": map[string]any{"selector": "#btn1"}},
					map[string]any{"id": "c", "type": "click", "data": map[string]any{"selector": "#btn2"}},
				},
				"edges": []any{
					map[string]any{"id": "e1", "source": "a", "target": "b"},
					map[string]any{"id": "e2", "source": "b", "target": "c"},
				},
			},
		}

		plan, instructions, err := compiler.Compile(ctx, execID, workflow)

		require.NoError(t, err)
		require.Len(t, instructions, 3)
		assert.Equal(t, 0, instructions[0].Index)
		assert.Equal(t, 1, instructions[1].Index)
		assert.Equal(t, 2, instructions[2].Index)
		// Verify graph steps also have correct indices
		require.NotNil(t, plan.Graph)
		for i, step := range plan.Graph.Steps {
			assert.Equal(t, i, step.Index)
		}
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] initializes empty maps for Context and Metadata", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "empty-maps-test",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "nav", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				},
				"edges": []any{},
			},
		}

		_, instructions, err := compiler.Compile(ctx, execID, workflow)

		require.NoError(t, err)
		require.Len(t, instructions, 1)
		assert.NotNil(t, instructions[0].Context)
		assert.NotNil(t, instructions[0].Metadata)
		assert.Empty(t, instructions[0].Context)
		assert.Empty(t, instructions[0].Metadata)
	})
}

func TestContractPlanCompiler_MetadataExtraction(t *testing.T) {
	compiler := &ContractPlanCompiler{}
	ctx := context.Background()
	execID := uuid.New()

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] extracts viewport metadata from settings", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "viewport-metadata",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "nav", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				},
				"edges": []any{},
				"settings": map[string]any{
					"executionViewport": map[string]any{
						"width":  float64(1920),
						"height": float64(1080),
					},
				},
			},
		}

		plan, _, err := compiler.Compile(ctx, execID, workflow)

		require.NoError(t, err)
		require.NotNil(t, plan.Metadata)
		viewport, ok := plan.Metadata["executionViewport"].(map[string]any)
		require.True(t, ok, "executionViewport should be in metadata")
		assert.EqualValues(t, 1920, viewport["width"])
		assert.EqualValues(t, 1080, viewport["height"])
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] extracts entry selector from settings", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "entry-selector",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "nav", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				},
				"edges": []any{},
				"settings": map[string]any{
					"entrySelector":          "#app-loaded",
					"entrySelectorTimeoutMs": float64(5000),
				},
			},
		}

		plan, _, err := compiler.Compile(ctx, execID, workflow)

		require.NoError(t, err)
		require.NotNil(t, plan.Metadata)
		assert.Equal(t, "#app-loaded", plan.Metadata["entrySelector"])
		assert.Equal(t, 5000, plan.Metadata["entrySelectorTimeoutMs"])
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] handles nil metadata gracefully", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "no-settings",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "nav", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				},
				"edges": []any{},
				// No settings key
			},
		}

		plan, _, err := compiler.Compile(ctx, execID, workflow)

		require.NoError(t, err)
		// Metadata can be nil when there's nothing to extract
		// This is acceptable behavior
		assert.Nil(t, plan.Metadata)
	})
}

func TestContractPlanCompiler_GraphStructure(t *testing.T) {
	compiler := &ContractPlanCompiler{}
	ctx := context.Background()
	execID := uuid.New()

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] produces graph with correct step types", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "graph-types",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "nav", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
					map[string]any{"id": "wait", "type": "wait", "data": map[string]any{"duration": float64(1000)}},
					map[string]any{"id": "screenshot", "type": "screenshot", "data": map[string]any{}},
				},
				"edges": []any{
					map[string]any{"id": "e1", "source": "nav", "target": "wait"},
					map[string]any{"id": "e2", "source": "wait", "target": "screenshot"},
				},
			},
		}

		plan, _, err := compiler.Compile(ctx, execID, workflow)

		require.NoError(t, err)
		require.NotNil(t, plan.Graph)
		require.Len(t, plan.Graph.Steps, 3)
		assert.Equal(t, "navigate", plan.Graph.Steps[0].Type)
		assert.Equal(t, "wait", plan.Graph.Steps[1].Type)
		assert.Equal(t, "screenshot", plan.Graph.Steps[2].Type)
	})

	t.Run("[REQ:BAS-WORKFLOW-PERSIST-CRUD] graph steps have node IDs", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "graph-node-ids",
			FlowDefinition: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "custom-id-123", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				},
				"edges": []any{},
			},
		}

		plan, _, err := compiler.Compile(ctx, execID, workflow)

		require.NoError(t, err)
		require.NotNil(t, plan.Graph)
		require.Len(t, plan.Graph.Steps, 1)
		assert.Equal(t, "custom-id-123", plan.Graph.Steps[0].NodeID)
	})
}

func TestPlanCompilerEnvOverride(t *testing.T) {
	t.Skip("BAS_PLAN_COMPILER override removed in favor of contract-native compiler")
}
