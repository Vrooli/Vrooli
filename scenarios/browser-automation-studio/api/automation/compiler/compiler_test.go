package compiler

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/internal/scenarioport"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

func assertCompiledStepType(t testing.TB, step ExecutionStep, want StepType) {
	t.Helper()
	assert.Equal(t, enums.StringToActionType(string(want)), step.Action.GetType())
}

// Helper to create a test workflow with proto types
func makeTestWorkflow(id uuid.UUID, name string, nodes []*basworkflows.WorkflowNodeV2, edges []*basworkflows.WorkflowEdgeV2) *basapi.WorkflowSummary {
	return &basapi.WorkflowSummary{
		Id:   id.String(),
		Name: name,
		FlowDefinition: &basworkflows.WorkflowDefinitionV2{
			Nodes: nodes,
			Edges: edges,
		},
	}
}

func makeTestWorkflowWithSettings(id uuid.UUID, name string, nodes []*basworkflows.WorkflowNodeV2, edges []*basworkflows.WorkflowEdgeV2, settings *basworkflows.WorkflowSettingsV2) *basapi.WorkflowSummary {
	return &basapi.WorkflowSummary{
		Id:   id.String(),
		Name: name,
		FlowDefinition: &basworkflows.WorkflowDefinitionV2{
			Nodes:    nodes,
			Edges:    edges,
			Settings: settings,
		},
	}
}

func TestCompileWorkflowSequential(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"test-flow",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "node-1",
				Action: &basactions.ActionDefinition{
					Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
					Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
			{
				Id: "node-2",
				Action: &basactions.ActionDefinition{
					Type:   basactions.ActionType_ACTION_TYPE_WAIT,
					Params: &basactions.ActionDefinition_Wait{Wait: &basactions.WaitParams{WaitFor: &basactions.WaitParams_DurationMs{DurationMs: 1000}}},
				},
				Position: &basbase.NodePosition{X: 200, Y: 0},
			},
			{
				Id: "node-3",
				Action: &basactions.ActionDefinition{
					Type:   basactions.ActionType_ACTION_TYPE_SCREENSHOT,
					Params: &basactions.ActionDefinition_Screenshot{Screenshot: &basactions.ScreenshotParams{}},
				},
				Position: &basbase.NodePosition{X: 400, Y: 0},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "edge-1", Source: "node-1", Target: "node-2"},
			{Id: "edge-2", Source: "node-2", Target: "node-3"},
		},
	)

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

	assertCompiledStepType(t, plan.Steps[0], StepNavigate)

	if got := plan.Steps[0].OutgoingEdges; len(got) != 1 || got[0].TargetNode != "node-2" {
		t.Fatalf("unexpected outgoing edges: %+v", got)
	}
}

func TestCompileWorkflowWithScenarioRootResolvesGeneratedScenarioPort(t *testing.T) {
	cli := scenarioport.NewMockScenarioCLI()
	cli.Ports["generated"] = map[string]int{"API_PORT": 6123}
	restore := scenarioport.SetScenarioCLIForTests(cli)
	defer restore()

	route := "/notes"
	workflow := makeTestWorkflow(uuid.New(), "generated-flow", []*basworkflows.WorkflowNodeV2{{
		Id: "navigate",
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
			Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{
				Scenario:     ptr("generated"),
				ScenarioPath: &route,
			}},
		},
	}}, nil)

	plan, err := CompileWorkflowWithOptions(workflow, &CompileOptions{ScenarioRoot: "/tmp/workspace/scenarios/generated"})
	if err != nil {
		t.Fatalf("CompileWorkflowWithOptions: %v", err)
	}
	if got, want := plan.Steps[0].Action.GetNavigate().GetUrl(), "http://localhost:6123/notes"; got != want {
		t.Fatalf("resolved URL = %#v, want %q", got, want)
	}
}

// ptr is a helper to create pointers to primitive values
func ptr[T any](v T) *T {
	return &v
}

func TestCompileWorkflowDetectsCycles(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"cycle",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
			{Id: "b", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_WAIT, Params: &basactions.ActionDefinition_Wait{Wait: &basactions.WaitParams{WaitFor: &basactions.WaitParams_DurationMs{DurationMs: 1000}}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "ab", Source: "a", Target: "b"},
			{Id: "ba", Source: "b", Target: "a"},
		},
	)

	if _, err := CompileWorkflow(workflow); err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
}

func TestCompileWorkflowUnsupportedType(t *testing.T) {
	// This test uses an unsupported action type - we use UNSPECIFIED which maps to "custom"
	// But we need a truly invalid type string. Since proto types are validated, we can't
	// easily create an invalid type. Instead, skip this test as the proto types enforce
	// valid action types at compile time.
	t.Skip("Proto types enforce valid action types - this test is not applicable with typed protos")
}

func TestCompileWorkflowLoopExtractsBody(t *testing.T) {
	// Create a workflow with a loop node using proper V2 ActionDefinition.
	// The loop node connects to body nodes via edges with "loopbody" source handle.
	workflow := makeTestWorkflow(
		uuid.New(),
		"loop-test",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "loop-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP,
					Params: &basactions.ActionDefinition_Loop{
						Loop: &basactions.LoopParams{
							LoopType:      basactions.LoopType_LOOP_TYPE_FOREACH,
							ArraySource:   ptr("${items}"),
							ItemVariable:  ptr("item"),
							MaxIterations: ptr(int32(100)),
						},
					},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
			{
				Id: "body-click",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_CLICK,
					Params: &basactions.ActionDefinition_Click{
						Click: &basactions.ClickParams{
							Selector: "#item-btn",
						},
					},
				},
				Position: &basbase.NodePosition{X: 100, Y: 100},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{
				Id:           "e-loop-body",
				Source:       "loop-1",
				Target:       "body-click",
				SourceHandle: ptr("loopbody"), // Special handle indicating loop body connection
			},
			{
				Id:           "e-body-continue",
				Source:       "body-click",
				Target:       "loop-1",
				TargetHandle: ptr("loopcontinue"), // Return to loop for next iteration
			},
		},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, plan)

	// The main plan should have exactly 1 step (the loop itself)
	// because body nodes are extracted into the loop's sub-plan
	require.Len(t, plan.Steps, 1, "main plan should have 1 step (the loop)")

	loopStep := plan.Steps[0]
	assert.Equal(t, "loop-1", loopStep.NodeID)
	assertCompiledStepType(t, loopStep, StepLoop)

	// Verify loop body was extracted
	require.NotNil(t, loopStep.LoopPlan, "loop step should have a body plan")
	require.Len(t, loopStep.LoopPlan.Steps, 1, "loop body should have 1 step")
	assert.Equal(t, "body-click", loopStep.LoopPlan.Steps[0].NodeID)
	assertCompiledStepType(t, loopStep.LoopPlan.Steps[0], StepClick)

	// Verify loop params were extracted
	assert.Equal(t, "${items}", loopStep.Action.GetLoop().GetArraySource())
	assert.Equal(t, "item", loopStep.Action.GetLoop().GetItemVariable())
}

func TestCompileWorkflowEntryMetadata(t *testing.T) {
	entrySelector := "[data-testid=app-ready]"
	entrySelectorTimeout := int32(1500)
	workflow := makeTestWorkflowWithSettings(
		uuid.New(),
		"entry-flow",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{},
		&basworkflows.WorkflowSettingsV2{
			EntrySelector:          &entrySelector,
			EntrySelectorTimeoutMs: &entrySelectorTimeout,
		},
	)

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
	switch v := plan.Metadata["entrySelectorTimeoutMs"].(type) {
	case int:
		if v != 1500 {
			t.Fatalf("unexpected entrySelectorTimeoutMs: %v (type %T)", v, v)
		}
	case float64:
		if int(v) != 1500 {
			t.Fatalf("unexpected entrySelectorTimeoutMs: %v (type %T)", v, v)
		}
	default:
		t.Fatalf("unexpected entrySelectorTimeoutMs: %v (type %T)", plan.Metadata["entrySelectorTimeoutMs"], plan.Metadata["entrySelectorTimeoutMs"])
	}
}

func TestCompileWorkflowMapsSessionReuseLabelToExecutionPolicy(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"isolated-validation",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "navigate", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{},
	)
	workflow.FlowDefinition.Metadata = &basworkflows.WorkflowMetadataV2{Labels: map[string]string{"session_reuse_mode": "fresh"}}

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	assert.Equal(t, "fresh", plan.Metadata["sessionReuseMode"])
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
	workflow := &basapi.WorkflowSummary{
		Id:             uuid.New().String(),
		Name:           "no-definition",
		FlowDefinition: nil,
	}
	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flow_definition")
}

func TestCompileWorkflow_EmptyNodes(t *testing.T) {
	workflowID := uuid.New()
	workflow := makeTestWorkflow(workflowID, "empty-flow", []*basworkflows.WorkflowNodeV2{}, []*basworkflows.WorkflowEdgeV2{})

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Empty(t, plan.Steps)
	assert.Equal(t, workflowID, plan.WorkflowID)
}

// =============================================================================
// Step Type Validation Tests
// =============================================================================

func TestCompileWorkflow_AllSupportedActionTypes(t *testing.T) {
	// Test all proto-supported action types
	testCases := []struct {
		name     string
		action   *basactions.ActionDefinition
		expected StepType
	}{
		{
			name: "navigate",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
				Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}},
			},
			expected: StepNavigate,
		},
		{
			name: "click",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_CLICK,
				Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#btn"}},
			},
			expected: StepClick,
		},
		{
			name: "input",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_INPUT,
				Params: &basactions.ActionDefinition_Input{Input: &basactions.InputParams{Selector: "#input", Value: "test"}},
			},
			expected: StepType("type"),
		},
		{
			name: "wait",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_WAIT,
				Params: &basactions.ActionDefinition_Wait{Wait: &basactions.WaitParams{WaitFor: &basactions.WaitParams_DurationMs{DurationMs: 1000}}},
			},
			expected: StepWait,
		},
		{
			name: "assert",
			action: &basactions.ActionDefinition{
				Type: basactions.ActionType_ACTION_TYPE_ASSERT,
				Params: &basactions.ActionDefinition_Assert{Assert: &basactions.AssertParams{
					Selector: "#el",
					Mode:     basbase.AssertionMode_ASSERTION_MODE_VISIBLE,
				}},
			},
			expected: StepAssert,
		},
		{
			name: "scroll",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_SCROLL,
				Params: &basactions.ActionDefinition_Scroll{Scroll: &basactions.ScrollParams{}},
			},
			expected: StepScroll,
		},
		{
			name: "screenshot",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_SCREENSHOT,
				Params: &basactions.ActionDefinition_Screenshot{Screenshot: &basactions.ScreenshotParams{}},
			},
			expected: StepScreenshot,
		},
		{
			name: "hover",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_HOVER,
				Params: &basactions.ActionDefinition_Hover{Hover: &basactions.HoverParams{Selector: "#btn"}},
			},
			expected: StepHover,
		},
		{
			name: "focus",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_FOCUS,
				Params: &basactions.ActionDefinition_Focus{Focus: &basactions.FocusParams{Selector: "#input"}},
			},
			expected: StepFocus,
		},
		{
			name: "blur",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_BLUR,
				Params: &basactions.ActionDefinition_Blur{Blur: &basactions.BlurParams{Selector: ptr("#input")}},
			},
			expected: StepBlur,
		},
		{
			name: "keyboard",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_KEYBOARD,
				Params: &basactions.ActionDefinition_Keyboard{Keyboard: &basactions.KeyboardParams{Key: ptr("Enter")}},
			},
			expected: StepKeyboard,
		},
		{
			name: "evaluate",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_EVALUATE,
				Params: &basactions.ActionDefinition_Evaluate{Evaluate: &basactions.EvaluateParams{Expression: "document.title"}},
			},
			expected: StepEvaluate,
		},
		{
			name: "extract",
			action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_EXTRACT,
				Params: &basactions.ActionDefinition_Extract{Extract: &basactions.ExtractParams{Selector: "#data"}},
			},
			expected: StepExtract,
		},
		{
			name: "gesture",
			action: &basactions.ActionDefinition{
				Type: basactions.ActionType_ACTION_TYPE_GESTURE,
				Params: &basactions.ActionDefinition_Gesture{Gesture: &basactions.GestureParams{
					GestureType: basactions.GestureType_GESTURE_TYPE_SWIPE,
					Selector:    ptr("[data-testid='graph-canvas']"),
				}},
			},
			expected: StepGesture,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workflow := makeTestWorkflow(
				uuid.New(),
				"test-"+tc.name,
				[]*basworkflows.WorkflowNodeV2{
					{
						Id:     "node-1",
						Action: tc.action,
					},
				},
				[]*basworkflows.WorkflowEdgeV2{},
			)

			plan, err := CompileWorkflow(workflow)
			require.NoError(t, err, "action type %s should be supported", tc.name)
			require.Len(t, plan.Steps, 1)
			assertCompiledStepType(t, plan.Steps[0], tc.expected)
		})
	}
}

func TestBuildActionDefinition_Gesture(t *testing.T) {
	action, err := BuildActionDefinition("gesture", map[string]any{
		"gesture_type":  "GESTURE_TYPE_SWIPE",
		"selector":      "[data-testid='graph-canvas']",
		"direction":     "SWIPE_DIRECTION_RIGHT",
		"steps":         float64(40),
		"step_delay_ms": float64(25),
		"trace_label":   "graph-sustained-pan",
	})
	require.NoError(t, err)
	require.Equal(t, basactions.ActionType_ACTION_TYPE_GESTURE, action.GetType())
	gesture := action.GetGesture()
	require.NotNil(t, gesture)
	assert.Equal(t, basactions.GestureType_GESTURE_TYPE_SWIPE, gesture.GetGestureType())
	assert.Equal(t, basactions.SwipeDirection_SWIPE_DIRECTION_RIGHT, gesture.GetDirection())
	assert.Equal(t, "[data-testid='graph-canvas']", gesture.GetSelector())
	assert.EqualValues(t, 40, gesture.GetSteps())
	assert.EqualValues(t, 25, gesture.GetStepDelayMs())
	assert.Equal(t, "graph-sustained-pan", gesture.GetTraceLabel())
}

func TestCompileWorkflow_EmptyStepTypeError(t *testing.T) {
	// Protojson omits default enum values (0), so an UNSPECIFIED action type ends up
	// missing `action.type` when marshalled. That should be treated as invalid.
	workflow := makeTestWorkflow(
		uuid.New(),
		"unspecified-type",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "node-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_UNSPECIFIED,
				},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{},
	)

	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unspecified action type")
}

func TestCompileWorkflow_MissingActionRejected(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"missing-action",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "node-1"},
		},
		[]*basworkflows.WorkflowEdgeV2{},
	)

	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required action field")
}

func TestCompileWorkflow_WhitespaceOnlyStepTypeError(t *testing.T) {
	// With proto types, action types are enums - there's no concept of whitespace-only type
	// Skip this test as it's not applicable to proto-based workflows
	t.Skip("Proto action types are enums - whitespace-only type is not possible")
}

// =============================================================================
// Branching and Conditional Workflow Tests
// =============================================================================

func TestCompileWorkflow_DiamondPattern(t *testing.T) {
	// Diamond pattern: A -> B, A -> C, B -> D, C -> D
	workflow := makeTestWorkflow(
		uuid.New(),
		"diamond-flow",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
			{Id: "b", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK, Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#btn-b"}}}},
			{Id: "c", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK, Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#btn-c"}}}},
			{Id: "d", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SCREENSHOT, Params: &basactions.ActionDefinition_Screenshot{Screenshot: &basactions.ScreenshotParams{}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "ab", Source: "a", Target: "b"},
			{Id: "ac", Source: "a", Target: "c"},
			{Id: "bd", Source: "b", Target: "d"},
			{Id: "cd", Source: "c", Target: "d"},
		},
	)

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
	workflow := makeTestWorkflow(
		uuid.New(),
		"multi-entry",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a1", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://a.com"}}}},
			{Id: "a2", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK, Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#a"}}}},
			{Id: "b1", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://b.com"}}}},
			{Id: "b2", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK, Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#b"}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "a1a2", Source: "a1", Target: "a2"},
			{Id: "b1b2", Source: "b1", Target: "b2"},
		},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	// All 4 nodes should be included even though there are two disconnected chains
	require.Len(t, plan.Steps, 4)
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestCompileWorkflow_EdgeWithMissingSource(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"missing-source",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "e1", Source: "", Target: "a"},
		},
	)

	// Should compile without error - empty source edges are ignored
	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	assert.Len(t, plan.Steps, 1)
}

func TestCompileWorkflow_EdgeWithMissingTarget(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"missing-target",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "e1", Source: "a", Target: ""},
		},
	)

	// Should compile without error - empty target edges are ignored
	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	assert.Len(t, plan.Steps, 1)
}

func TestCompileWorkflow_SelfLoopDetected(t *testing.T) {
	// A node that points to itself (without loop handle) should cause cycle detection
	workflow := makeTestWorkflow(
		uuid.New(),
		"self-loop",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "aa", Source: "a", Target: "a"},
		},
	)

	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cycle")
}

func TestCompileWorkflow_ThreeNodeCycle(t *testing.T) {
	// A -> B -> C -> A (without loop handles)
	workflow := makeTestWorkflow(
		uuid.New(),
		"three-cycle",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
			{Id: "b", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK, Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#b"}}}},
			{Id: "c", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_WAIT, Params: &basactions.ActionDefinition_Wait{Wait: &basactions.WaitParams{WaitFor: &basactions.WaitParams_DurationMs{DurationMs: 1000}}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "ab", Source: "a", Target: "b"},
			{Id: "bc", Source: "b", Target: "c"},
			{Id: "ca", Source: "c", Target: "a"},
		},
	)

	_, err := CompileWorkflow(workflow)
	require.Error(t, err)
}

// =============================================================================
// Viewport and Settings Tests
// =============================================================================

func TestCompileWorkflow_ViewportSettings(t *testing.T) {
	viewportWidth := int32(1920)
	viewportHeight := int32(1080)
	workflow := makeTestWorkflowWithSettings(
		uuid.New(),
		"viewport-flow",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{},
		&basworkflows.WorkflowSettingsV2{
			ViewportWidth:  &viewportWidth,
			ViewportHeight: &viewportHeight,
		},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, plan.Metadata)

	viewport, ok := plan.Metadata["executionViewport"].(map[string]any)
	require.True(t, ok, "executionViewport should be a map")
	switch v := viewport["width"].(type) {
	case int:
		assert.Equal(t, 1920, v)
	case float64:
		assert.Equal(t, 1920, int(v))
	default:
		t.Fatalf("unexpected viewport.width type %T", viewport["width"])
	}
	switch v := viewport["height"].(type) {
	case int:
		assert.Equal(t, 1080, v)
	case float64:
		assert.Equal(t, 1080, int(v))
	default:
		t.Fatalf("unexpected viewport.height type %T", viewport["height"])
	}
}

func TestCompileWorkflow_NoSettingsNoMetadata(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"no-settings",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	// Metadata should be nil when no settings are present
	assert.Nil(t, plan.Metadata)
}

func TestCompileWorkflow_InvalidViewportDimensions(t *testing.T) {
	t.Run("zero viewport dimensions", func(t *testing.T) {
		zeroVal := int32(0)
		workflow := makeTestWorkflowWithSettings(
			uuid.New(),
			"zero-viewport",
			[]*basworkflows.WorkflowNodeV2{
				{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
			},
			[]*basworkflows.WorkflowEdgeV2{},
			&basworkflows.WorkflowSettingsV2{
				ViewportWidth:  &zeroVal,
				ViewportHeight: &zeroVal,
			},
		)

		plan, err := CompileWorkflow(workflow)
		require.NoError(t, err)
		// Zero dimensions should not create viewport metadata
		if plan.Metadata != nil {
			_, hasViewport := plan.Metadata["executionViewport"]
			assert.False(t, hasViewport)
		}
	})

	t.Run("negative viewport dimensions", func(t *testing.T) {
		// Proto int32 can't have negative values affect viewport extraction since
		// the compiler checks for positive values. Proto uses uint or validates separately.
		// Skip this subtest as negative int32 values won't work the same way.
		t.Skip("Proto int32 viewport dimensions handle negatives differently")
	})
}

// =============================================================================
// Position Handling Tests
// =============================================================================

func TestCompileWorkflow_NodePositions(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"positioned-flow",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id:       "a",
				Action:   &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}},
				Position: &basbase.NodePosition{X: 100.5, Y: 200.75},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 1)
	require.NotNil(t, plan.Steps[0].SourcePosition)
	assert.Equal(t, 100.5, plan.Steps[0].SourcePosition.X)
	assert.Equal(t, 200.75, plan.Steps[0].SourcePosition.Y)
}

func TestCompileWorkflow_MissingPosition(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"no-position",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id:     "a",
				Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}},
				// No position field
			},
		},
		[]*basworkflows.WorkflowEdgeV2{},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 1)
	assert.Nil(t, plan.Steps[0].SourcePosition)
}

// =============================================================================
// Edge Condition Tests
// =============================================================================

func TestCompileWorkflow_EdgeConditions(t *testing.T) {
	// Note: V2 edges don't have a "data.condition" field in the same way V1 did.
	// Edge conditions in V2 are handled via edge type or label.
	// Skip this test until V2 edge conditions are properly defined.
	t.Skip("V2 edge conditions require different structure - skip until implemented")
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
	// A loop node without any body connections should produce an error.
	// The compiler requires at least one edge with "loopbody" source handle.
	workflow := makeTestWorkflow(
		uuid.New(),
		"loop-no-body",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "loop-orphan",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP,
					Params: &basactions.ActionDefinition_Loop{
						Loop: &basactions.LoopParams{
							LoopType:      basactions.LoopType_LOOP_TYPE_REPEAT,
							Count:         ptr(int32(5)),
							MaxIterations: ptr(int32(10)),
						},
					},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{}, // No edges - no body connection
	)

	_, err := CompileWorkflow(workflow)
	require.Error(t, err, "loop without body should produce an error")
	assert.Contains(t, err.Error(), "requires at least one body connection")
}

func TestCompileWorkflow_LoopWithMultipleBodySteps(t *testing.T) {
	// A loop with multiple body steps chained together.
	// Loop → body-step-1 → body-step-2 → body-step-3 → (continue back to loop)
	workflow := makeTestWorkflow(
		uuid.New(),
		"loop-multi-body",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "loop-main",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP,
					Params: &basactions.ActionDefinition_Loop{
						Loop: &basactions.LoopParams{
							LoopType:      basactions.LoopType_LOOP_TYPE_FOREACH,
							ArraySource:   ptr("${users}"),
							ItemVariable:  ptr("user"),
							IndexVariable: ptr("idx"),
							MaxIterations: ptr(int32(50)),
						},
					},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
			{
				Id: "body-step-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_CLICK,
					Params: &basactions.ActionDefinition_Click{
						Click: &basactions.ClickParams{Selector: "#user-${idx}"},
					},
				},
				Position: &basbase.NodePosition{X: 100, Y: 100},
			},
			{
				Id: "body-step-2",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_WAIT,
					Params: &basactions.ActionDefinition_Wait{
						Wait: &basactions.WaitParams{
							WaitFor: &basactions.WaitParams_DurationMs{DurationMs: 500},
						},
					},
				},
				Position: &basbase.NodePosition{X: 200, Y: 100},
			},
			{
				Id: "body-step-3",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SCREENSHOT,
					Params: &basactions.ActionDefinition_Screenshot{
						Screenshot: &basactions.ScreenshotParams{},
					},
				},
				Position: &basbase.NodePosition{X: 300, Y: 100},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{
			// Loop to first body step
			{
				Id:           "e-loop-to-body1",
				Source:       "loop-main",
				Target:       "body-step-1",
				SourceHandle: ptr("loopbody"),
			},
			// Chain body steps together
			{
				Id:     "e-body1-to-body2",
				Source: "body-step-1",
				Target: "body-step-2",
			},
			{
				Id:     "e-body2-to-body3",
				Source: "body-step-2",
				Target: "body-step-3",
			},
			// Last body step returns to loop
			{
				Id:           "e-body3-continue",
				Source:       "body-step-3",
				Target:       "loop-main",
				TargetHandle: ptr("loopcontinue"),
			},
		},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, plan)

	// Main plan should have only the loop step
	require.Len(t, plan.Steps, 1, "main plan should have 1 step (the loop)")

	loopStep := plan.Steps[0]
	assert.Equal(t, "loop-main", loopStep.NodeID)
	assertCompiledStepType(t, loopStep, StepLoop)

	// Loop body should have all 3 steps in correct order
	require.NotNil(t, loopStep.LoopPlan, "loop step should have a body plan")
	require.Len(t, loopStep.LoopPlan.Steps, 3, "loop body should have 3 steps")

	// Verify step order (topological sort should maintain edge order)
	assert.Equal(t, "body-step-1", loopStep.LoopPlan.Steps[0].NodeID)
	assertCompiledStepType(t, loopStep.LoopPlan.Steps[0], StepClick)

	assert.Equal(t, "body-step-2", loopStep.LoopPlan.Steps[1].NodeID)
	assertCompiledStepType(t, loopStep.LoopPlan.Steps[1], StepWait)

	assert.Equal(t, "body-step-3", loopStep.LoopPlan.Steps[2].NodeID)
	assertCompiledStepType(t, loopStep.LoopPlan.Steps[2], StepScreenshot)

	// Verify loop params
	assert.Equal(t, "${users}", loopStep.Action.GetLoop().GetArraySource())
	assert.Equal(t, "user", loopStep.Action.GetLoop().GetItemVariable())
	assert.Equal(t, "idx", loopStep.Action.GetLoop().GetIndexVariable())
}

// =============================================================================
// Data Preservation Tests
// =============================================================================

func TestCompileWorkflow_ParamsPreserved(t *testing.T) {
	workflow := makeTestWorkflow(
		uuid.New(),
		"params-test",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "input-node",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_INPUT,
					Params: &basactions.ActionDefinition_Input{
						Input: &basactions.InputParams{
							Selector: "#input",
							Value:    "hello world",
							Submit:   ptr(true),
						},
					},
				},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 1)

	input := plan.Steps[0].Action.GetInput()
	assert.Equal(t, "#input", input.GetSelector())
	assert.Equal(t, "hello world", input.GetValue())
	assert.Equal(t, true, input.GetSubmit())
}

func TestCompileWorkflow_WorkflowIDPreserved(t *testing.T) {
	workflowID := uuid.New()
	workflow := makeTestWorkflow(
		workflowID,
		"id-test",
		[]*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}}},
		},
		[]*basworkflows.WorkflowEdgeV2{},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	assert.Equal(t, workflowID, plan.WorkflowID)
	assert.Equal(t, "id-test", plan.WorkflowName)
}

// The compiler marshals flow definitions with UseProtoNames (snake_case) while
// the param builders consume protojson-default camelCase. These tests pin the
// compilation seam: typed proto node → compiled typed ActionDefinition.
// Regression: wait durations were silently dropped and the driver fell back to
// its 1000ms default (scenario-qa knw-1784773788743993536).
func TestCompileWorkflow_WaitDurationSurvivesProtoNamesMarshal(t *testing.T) {
	timeout := int32(20000)
	workflow := makeTestWorkflow(
		uuid.New(),
		"wait-duration-flow",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "nav",
				Action: &basactions.ActionDefinition{
					Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
					Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}},
				},
			},
			{
				Id: "wait",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_WAIT,
					Params: &basactions.ActionDefinition_Wait{Wait: &basactions.WaitParams{
						WaitFor:   &basactions.WaitParams_DurationMs{DurationMs: 15000},
						TimeoutMs: &timeout,
					}},
				},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "e1", Source: "nav", Target: "wait"},
		},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 2)

	waitStep := plan.Steps[1]
	assertCompiledStepType(t, waitStep, StepWait)

	waitParams := waitStep.Action.GetWait()
	require.NotNil(t, waitParams)
	assert.Equal(t, int32(15000), waitParams.GetDurationMs(), "wait duration must survive the snake_case marshal/compile roundtrip")
	assert.Equal(t, int32(20000), waitParams.GetTimeoutMs())
}

func TestCompileWorkflow_MultiWordParamsSurviveProtoNamesMarshal(t *testing.T) {
	clickCount := int32(2)
	delayMs := int32(250)
	clearFirst := true
	workflow := makeTestWorkflow(
		uuid.New(),
		"multi-word-params-flow",
		[]*basworkflows.WorkflowNodeV2{
			{
				Id: "click",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_CLICK,
					Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{
						Selector:   "#btn",
						ClickCount: &clickCount,
						DelayMs:    &delayMs,
					}},
				},
			},
			{
				Id: "input",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_INPUT,
					Params: &basactions.ActionDefinition_Input{Input: &basactions.InputParams{
						Selector:   "#field",
						Value:      "hello",
						ClearFirst: &clearFirst,
					}},
				},
			},
		},
		[]*basworkflows.WorkflowEdgeV2{
			{Id: "e1", Source: "click", Target: "input"},
		},
	)

	plan, err := CompileWorkflow(workflow)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 2)

	clickParams := plan.Steps[0].Action.GetClick()
	require.NotNil(t, clickParams)
	assert.Equal(t, int32(2), clickParams.GetClickCount())
	assert.Equal(t, int32(250), clickParams.GetDelayMs())

	inputParams := plan.Steps[1].Action.GetInput()
	require.NotNil(t, inputParams)
	assert.True(t, inputParams.GetClearFirst())
}

func TestCompileWorkflowToContractsUsesCompiledTypedActions(t *testing.T) {
	executionID := uuid.New()
	workflow := makeTestWorkflow(
		uuid.New(),
		"typed-contract-flow",
		[]*basworkflows.WorkflowNodeV2{{
			Id: "wait",
			Action: &basactions.ActionDefinition{
				Type: basactions.ActionType_ACTION_TYPE_WAIT,
				Params: &basactions.ActionDefinition_Wait{Wait: &basactions.WaitParams{
					WaitFor: &basactions.WaitParams_DurationMs{DurationMs: 4321},
				}},
			},
		}},
		nil,
	)

	plan, instructions, err := CompileWorkflowToContracts(context.Background(), executionID, workflow)
	require.NoError(t, err)
	require.Len(t, instructions, 1)
	require.NotNil(t, instructions[0].Action)
	assert.Equal(t, int32(4321), instructions[0].Action.GetWait().GetDurationMs())
	require.NotNil(t, plan.Graph)
	require.Len(t, plan.Graph.Steps, 1)
	require.NotNil(t, plan.Graph.Steps[0].Action)
	assert.Equal(t, int32(4321), plan.Graph.Steps[0].Action.GetWait().GetDurationMs())
}

func TestCompileWorkflowToContractsPreservesNodeExecutionSettings(t *testing.T) {
	continueOnError := true
	workflow := makeTestWorkflow(
		uuid.New(),
		"recoverable-contract-flow",
		[]*basworkflows.WorkflowNodeV2{{
			Id: "recoverable-click",
			Action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_CLICK,
				Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: ".missing"}},
			},
			ExecutionSettings: &basworkflows.NodeExecutionSettings{ContinueOnError: &continueOnError},
		}},
		nil,
	)

	plan, instructions, err := CompileWorkflowToContracts(context.Background(), uuid.New(), workflow)
	require.NoError(t, err)
	require.Len(t, instructions, 1)
	require.NotNil(t, plan.Graph)
	require.Len(t, plan.Graph.Steps, 1)
	assert.Equal(t, true, instructions[0].Context["continueOnError"])
	assert.Equal(t, true, plan.Graph.Steps[0].Context["continueOnError"])
}
