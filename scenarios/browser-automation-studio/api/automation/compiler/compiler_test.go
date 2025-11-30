package compiler

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestCompileWorkflowSequential(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "test-flow",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":       "node-1",
					"type":     "navigate",
					"data":     map[string]any{"url": "https://example.com"},
					"position": map[string]any{"x": 0, "y": 0},
				},
				map[string]any{
					"id":       "node-2",
					"type":     "wait",
					"data":     map[string]any{"type": "time", "duration": 1000},
					"position": map[string]any{"x": 200, "y": 0},
				},
				map[string]any{
					"id":       "node-3",
					"type":     "screenshot",
					"data":     map[string]any{"name": "after"},
					"position": map[string]any{"x": 400, "y": 0},
				},
			},
			"edges": []any{
				map[string]any{"id": "edge-1", "source": "node-1", "target": "node-2"},
				map[string]any{"id": "edge-2", "source": "node-2", "target": "node-3"},
			},
		},
	}

	plan, err := CompileWorkflow(workflow)
	if err != nil {
		t.Fatalf("compiled failed: %v", err)
	}

	if len(plan.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(plan.Steps))
	}

	if plan.Steps[0].NodeID != "node-1" || plan.Steps[1].NodeID != "node-2" || plan.Steps[2].NodeID != "node-3" {
		t.Fatalf("unexpected step order: %+v", plan.Steps)
	}

	if plan.Steps[0].Type != StepNavigate {
		t.Fatalf("expected first step to be navigate, got %s", plan.Steps[0].Type)
	}

	if got := plan.Steps[0].OutgoingEdges; len(got) != 1 || got[0].TargetNode != "node-2" {
		t.Fatalf("unexpected outgoing edges: %+v", got)
	}
}

func TestCompileWorkflowDetectsCycles(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "cycle",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				map[string]any{"id": "b", "type": "wait", "data": map[string]any{"type": "time", "duration": 1000}},
			},
			"edges": []any{
				map[string]any{"id": "ab", "source": "a", "target": "b"},
				map[string]any{"id": "ba", "source": "b", "target": "a"},
			},
		},
	}

	if _, err := CompileWorkflow(workflow); err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
}

func TestCompileWorkflowUnsupportedType(t *testing.T) {
	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "x", "type": "foo", "data": map[string]any{}},
			},
		},
	}

	if _, err := CompileWorkflow(workflow); err == nil {
		t.Fatal("expected unsupported type error, got nil")
	}
}

func TestCompileWorkflowLoopExtractsBody(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "loop-flow",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "loop", "type": "loop", "data": map[string]any{"loopType": "forEach", "arraySource": "rows"}},
				map[string]any{"id": "body-click", "type": "click", "data": map[string]any{"selector": "#save"}},
				map[string]any{"id": "body-type", "type": "type", "data": map[string]any{"selector": "#input", "text": "value"}},
				map[string]any{"id": "after", "type": "screenshot", "data": map[string]any{"name": "after"}},
			},
			"edges": []any{
				map[string]any{"id": "loop-body", "source": "loop", "target": "body-click", "sourceHandle": "loopBody"},
				map[string]any{"id": "body-chain", "source": "body-click", "target": "body-type"},
				map[string]any{"id": "loop-continue", "source": "body-type", "target": "loop", "targetHandle": "loopContinue"},
				map[string]any{"id": "loop-after", "source": "loop", "target": "after", "sourceHandle": "loopAfter"},
			},
		},
	}

	plan, err := CompileWorkflow(workflow)
	if err != nil {
		t.Fatalf("expected loop workflow to compile, got error: %v", err)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("expected loop plan to include loop node plus after node, got %d steps", len(plan.Steps))
	}
	loopStep := plan.Steps[0]
	if loopStep.Type != StepLoop {
		t.Fatalf("expected first step to be loop, got %s", loopStep.Type)
	}
	if loopStep.LoopPlan == nil {
		t.Fatalf("expected loop plan to include nested plan")
	}
	if len(loopStep.LoopPlan.Steps) != 2 {
		t.Fatalf("expected nested plan to include 2 body nodes, got %d", len(loopStep.LoopPlan.Steps))
	}
	bodyNodes := map[string]struct{}{}
	for _, step := range loopStep.LoopPlan.Steps {
		bodyNodes[step.NodeID] = struct{}{}
	}
	for _, expected := range []string{"body-click", "body-type"} {
		if _, ok := bodyNodes[expected]; !ok {
			t.Fatalf("expected loop body to contain %s", expected)
		}
	}
	if loopStep.LoopPlan.Steps[1].LoopPlan != nil {
		t.Fatalf("unexpected nested loop plan for non-loop body node")
	}
}

func TestCompileWorkflowEntryMetadata(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "entry-flow",
		FlowDefinition: database.JSONMap{
			"settings": map[string]any{
				"entrySelector":          "[data-testid=app-ready]",
				"entrySelectorTimeoutMs": 1500,
			},
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
			},
			"edges": []any{},
		},
	}

	plan, err := CompileWorkflow(workflow)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if plan.Metadata == nil {
		t.Fatalf("expected metadata to be populated")
	}
	if got := plan.Metadata["entrySelector"]; got != "[data-testid=app-ready]" {
		t.Fatalf("unexpected entrySelector: %v", got)
	}
	if got := plan.Metadata["entrySelectorTimeoutMs"]; got != 1500 {
		t.Fatalf("unexpected entrySelectorTimeoutMs: %v", got)
	}
}

// =============================================================================
// Nil/Empty Input Tests
// =============================================================================

func TestCompileWorkflow_NilWorkflow(t *testing.T) {
	_, err := CompileWorkflow(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestCompileWorkflow_NilFlowDefinition(t *testing.T) {
	workflow := &database.Workflow{
		ID:             uuid.New(),
		Name:           "no-definition",
		FlowDefinition: nil,
	}
	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flow_definition")
}

func TestCompileWorkflow_EmptyNodes(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "empty-flow",
		FlowDefinition: database.JSONMap{
			"nodes": []any{},
			"edges": []any{},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Empty(t, plan.Steps)
	assert.Equal(t, workflow.ID, plan.WorkflowID)
}

// =============================================================================
// Step Type Validation Tests
// =============================================================================

func TestCompileWorkflow_AllSupportedStepTypes(t *testing.T) {
	supportedTypes := []string{
		"navigate", "click", "hover", "dragDrop", "focus", "blur",
		"scroll", "select", "rotate", "gesture", "uploadFile", "type",
		"shortcut", "keyboard", "wait", "screenshot", "extract", "evaluate",
		"assert", "custom", "setVariable", "useVariable", "tabSwitch",
		"frameSwitch", "conditional", "loop", "setCookie", "getCookie",
		"clearCookie", "setStorage", "getStorage", "clearStorage",
		"networkMock", "subflow",
	}

	for _, stepType := range supportedTypes {
		t.Run(stepType, func(t *testing.T) {
			// Skip loop and conditional - they need special handling
			if stepType == "loop" || stepType == "conditional" {
				t.Skip("Complex step type requires special edges")
				return
			}
			if stepType == "subflow" {
				// Subflow needs workflowDefinition
				t.Skip("Subflow requires workflowDefinition")
				return
			}

			workflow := &database.Workflow{
				ID:   uuid.New(),
				Name: "test-" + stepType,
				FlowDefinition: database.JSONMap{
					"nodes": []any{
						map[string]any{"id": "node-1", "type": stepType, "data": map[string]any{}},
					},
					"edges": []any{},
				},
			}

			plan, err := CompileWorkflow(workflow)
			require.NoError(t, err, "step type %s should be supported", stepType)
			require.Len(t, plan.Steps, 1)
			assert.Equal(t, StepType(stepType), plan.Steps[0].Type)
		})
	}
}

func TestCompileWorkflow_EmptyStepTypeError(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "empty-type",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "node-1", "type": "", "data": map[string]any{}},
			},
			"edges": []any{},
		},
	}

	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestCompileWorkflow_WhitespaceOnlyStepTypeError(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "whitespace-type",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "node-1", "type": "   ", "data": map[string]any{}},
			},
			"edges": []any{},
		},
	}

	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

// =============================================================================
// Branching and Conditional Workflow Tests
// =============================================================================

func TestCompileWorkflow_DiamondPattern(t *testing.T) {
	// Diamond pattern: A -> B, A -> C, B -> D, C -> D
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "diamond-flow",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				map[string]any{"id": "b", "type": "click", "data": map[string]any{"selector": "#btn-b"}},
				map[string]any{"id": "c", "type": "click", "data": map[string]any{"selector": "#btn-c"}},
				map[string]any{"id": "d", "type": "screenshot", "data": map[string]any{"name": "final"}},
			},
			"edges": []any{
				map[string]any{"id": "ab", "source": "a", "target": "b"},
				map[string]any{"id": "ac", "source": "a", "target": "c"},
				map[string]any{"id": "bd", "source": "b", "target": "d"},
				map[string]any{"id": "cd", "source": "c", "target": "d"},
			},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 4)

	// First step should be 'a' (entry point with no incoming edges)
	assert.Equal(t, "a", plan.Steps[0].NodeID)
	// Last step should be 'd' (has incoming edges from both b and c)
	assert.Equal(t, "d", plan.Steps[3].NodeID)
}

func TestCompileWorkflow_MultipleEntryPoints(t *testing.T) {
	// Two disconnected chains
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "multi-entry",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a1", "type": "navigate", "data": map[string]any{"url": "https://a.com"}},
				map[string]any{"id": "a2", "type": "click", "data": map[string]any{"selector": "#a"}},
				map[string]any{"id": "b1", "type": "navigate", "data": map[string]any{"url": "https://b.com"}},
				map[string]any{"id": "b2", "type": "click", "data": map[string]any{"selector": "#b"}},
			},
			"edges": []any{
				map[string]any{"id": "a1a2", "source": "a1", "target": "a2"},
				map[string]any{"id": "b1b2", "source": "b1", "target": "b2"},
			},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	// All 4 nodes should be included even though there are two disconnected chains
	require.Len(t, plan.Steps, 4)
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestCompileWorkflow_EdgeWithMissingSource(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "missing-source",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
			},
			"edges": []any{
				map[string]any{"id": "e1", "source": "", "target": "a"},
			},
		},
	}

	// Should compile without error - empty source edges are ignored
	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	assert.Len(t, plan.Steps, 1)
}

func TestCompileWorkflow_EdgeWithMissingTarget(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "missing-target",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
			},
			"edges": []any{
				map[string]any{"id": "e1", "source": "a", "target": ""},
			},
		},
	}

	// Should compile without error - empty target edges are ignored
	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	assert.Len(t, plan.Steps, 1)
}

func TestCompileWorkflow_SelfLoopDetected(t *testing.T) {
	// A node that points to itself (without loop handle) should cause cycle detection
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "self-loop",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
			},
			"edges": []any{
				map[string]any{"id": "aa", "source": "a", "target": "a"},
			},
		},
	}

	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cycle")
}

func TestCompileWorkflow_ThreeNodeCycle(t *testing.T) {
	// A -> B -> C -> A (without loop handles)
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "three-cycle",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				map[string]any{"id": "b", "type": "click", "data": map[string]any{"selector": "#b"}},
				map[string]any{"id": "c", "type": "wait", "data": map[string]any{"type": "time", "duration": 1000}},
			},
			"edges": []any{
				map[string]any{"id": "ab", "source": "a", "target": "b"},
				map[string]any{"id": "bc", "source": "b", "target": "c"},
				map[string]any{"id": "ca", "source": "c", "target": "a"},
			},
		},
	}

	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
}

// =============================================================================
// Viewport and Settings Tests
// =============================================================================

func TestCompileWorkflow_ViewportSettings(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "viewport-flow",
		FlowDefinition: database.JSONMap{
			"settings": map[string]any{
				"executionViewport": map[string]any{
					"width":  1920,
					"height": 1080,
				},
			},
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
			},
			"edges": []any{},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, plan.Metadata)

	viewport, ok := plan.Metadata["executionViewport"].(map[string]any)
	require.True(t, ok, "executionViewport should be a map")
	assert.Equal(t, 1920, viewport["width"])
	assert.Equal(t, 1080, viewport["height"])
}

func TestCompileWorkflow_NoSettingsNoMetadata(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "no-settings",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
			},
			"edges": []any{},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	// Metadata should be nil when no settings are present
	assert.Nil(t, plan.Metadata)
}

func TestCompileWorkflow_InvalidViewportDimensions(t *testing.T) {
	t.Run("zero viewport dimensions", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "zero-viewport",
			FlowDefinition: database.JSONMap{
				"settings": map[string]any{
					"executionViewport": map[string]any{
						"width":  0,
						"height": 0,
					},
				},
				"nodes": []any{
					map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				},
				"edges": []any{},
			},
		}

		plan, err := CompileWorkflow(workflow)
		require.NoError(t, err)
		// Zero dimensions should not create viewport metadata
		if plan.Metadata != nil {
			_, hasViewport := plan.Metadata["executionViewport"]
			assert.False(t, hasViewport)
		}
	})

	t.Run("negative viewport dimensions", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:   uuid.New(),
			Name: "negative-viewport",
			FlowDefinition: database.JSONMap{
				"settings": map[string]any{
					"executionViewport": map[string]any{
						"width":  -100,
						"height": -100,
					},
				},
				"nodes": []any{
					map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				},
				"edges": []any{},
			},
		}

		plan, err := CompileWorkflow(workflow)
		require.NoError(t, err)
		// Negative dimensions should not create viewport metadata
		if plan.Metadata != nil {
			_, hasViewport := plan.Metadata["executionViewport"]
			assert.False(t, hasViewport)
		}
	})
}

// =============================================================================
// Position Handling Tests
// =============================================================================

func TestCompileWorkflow_NodePositions(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "positioned-flow",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":       "a",
					"type":     "navigate",
					"data":     map[string]any{"url": "https://example.com"},
					"position": map[string]any{"x": 100.5, "y": 200.75},
				},
			},
			"edges": []any{},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 1)
	require.NotNil(t, plan.Steps[0].SourcePosition)
	assert.Equal(t, 100.5, plan.Steps[0].SourcePosition.X)
	assert.Equal(t, 200.75, plan.Steps[0].SourcePosition.Y)
}

func TestCompileWorkflow_MissingPosition(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "no-position",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":   "a",
					"type": "navigate",
					"data": map[string]any{"url": "https://example.com"},
					// No position field
				},
			},
			"edges": []any{},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 1)
	assert.Nil(t, plan.Steps[0].SourcePosition)
}

// =============================================================================
// Edge Condition Tests
// =============================================================================

func TestCompileWorkflow_EdgeConditions(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "conditional-edges",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				map[string]any{"id": "b", "type": "click", "data": map[string]any{"selector": "#btn"}},
			},
			"edges": []any{
				map[string]any{
					"id":     "ab",
					"source": "a",
					"target": "b",
					"data":   map[string]any{"condition": "success"},
				},
			},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 2)

	// Check that edge condition is preserved
	require.Len(t, plan.Steps[0].OutgoingEdges, 1)
	assert.Equal(t, "success", plan.Steps[0].OutgoingEdges[0].Condition)
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestToPositiveInt(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{"positive float64", float64(42), 42},
		{"zero float64", float64(0), 0},
		{"negative float64", float64(-10), 0},
		{"positive int", 100, 100},
		{"zero int", 0, 0},
		{"negative int", -50, 0},
		{"valid string", "123", 123},
		{"empty string", "", 0},
		{"invalid string", "abc", 0},
		{"whitespace string", "  ", 0},
		{"nil value", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toPositiveInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToPositiveFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{"positive float64", float64(42.5), 42.5},
		{"zero float64", float64(0), 0},
		{"negative float64", float64(-10.5), 0},
		{"positive int", 100, 100},
		{"valid string", "123.5", 123.5},
		{"empty string", "", 0},
		{"nil value", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toPositiveFloat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty strings filtered",
			input:    []string{"a", "", "b", "  ", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all empty",
			input:    []string{"", "  ", ""},
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uniqueStrings(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Loop Node Edge Cases
// =============================================================================

func TestCompileWorkflow_LoopWithNoBody(t *testing.T) {
	// Loop node without any loopBody edges should fail
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "empty-loop",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "loop", "type": "loop", "data": map[string]any{"loopType": "forEach", "arraySource": "items"}},
				map[string]any{"id": "after", "type": "screenshot", "data": map[string]any{"name": "after"}},
			},
			"edges": []any{
				// No loopBody edge, only loopAfter
				map[string]any{"id": "loop-after", "source": "loop", "target": "after", "sourceHandle": "loopAfter"},
			},
		},
	}

	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "body")
}

func TestCompileWorkflow_LoopWithMultipleBodySteps(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "multi-body-loop",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "loop", "type": "loop", "data": map[string]any{"loopType": "forEach", "arraySource": "rows"}},
				map[string]any{"id": "body1", "type": "click", "data": map[string]any{"selector": "#item"}},
				map[string]any{"id": "body2", "type": "wait", "data": map[string]any{"type": "time", "duration": 500}},
				map[string]any{"id": "body3", "type": "screenshot", "data": map[string]any{"name": "item"}},
				map[string]any{"id": "after", "type": "screenshot", "data": map[string]any{"name": "done"}},
			},
			"edges": []any{
				map[string]any{"id": "loop-body1", "source": "loop", "target": "body1", "sourceHandle": "loopBody"},
				map[string]any{"id": "body1-body2", "source": "body1", "target": "body2"},
				map[string]any{"id": "body2-body3", "source": "body2", "target": "body3"},
				map[string]any{"id": "body3-loop", "source": "body3", "target": "loop", "targetHandle": "loopContinue"},
				map[string]any{"id": "loop-after", "source": "loop", "target": "after", "sourceHandle": "loopAfter"},
			},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 2) // loop + after

	loopStep := plan.Steps[0]
	assert.Equal(t, StepLoop, loopStep.Type)
	require.NotNil(t, loopStep.LoopPlan)
	assert.Len(t, loopStep.LoopPlan.Steps, 3) // body1, body2, body3
}

// =============================================================================
// Data Preservation Tests
// =============================================================================

func TestCompileWorkflow_ParamsPreserved(t *testing.T) {
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
						"text":     "hello world",
						"delay":    50,
						"clear":    true,
					},
				},
			},
			"edges": []any{},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 1)

	params := plan.Steps[0].Params
	assert.Equal(t, "#input", params["selector"])
	assert.Equal(t, "hello world", params["text"])
	assert.Equal(t, float64(50), params["delay"]) // JSON numbers are float64
	assert.Equal(t, true, params["clear"])
}

func TestCompileWorkflow_WorkflowIDPreserved(t *testing.T) {
	workflowID := uuid.New()
	workflow := &database.Workflow{
		ID:   workflowID,
		Name: "id-test",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
			},
			"edges": []any{},
		},
	}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	assert.Equal(t, workflowID, plan.WorkflowID)
	assert.Equal(t, "id-test", plan.WorkflowName)
}
