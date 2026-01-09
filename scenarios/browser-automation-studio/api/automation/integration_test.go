package automation_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/workflow/validator"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

func ptr[T any](v T) *T {
	return &v
}

// TestIntegration_V2LoopWorkflow tests the full pipeline:
// V2 WorkflowDefinition → Validate → Compile → ExecutionPlan
func TestIntegration_V2LoopWorkflow(t *testing.T) {
	workflowID := uuid.New()

	// Create a V2 workflow with a loop that iterates over items
	definition := &basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "loop-items",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP,
					Params: &basactions.ActionDefinition_Loop{
						Loop: &basactions.LoopParams{
							LoopType:      basactions.LoopType_LOOP_TYPE_FOREACH,
							ArraySource:   ptr("${items}"),
							ItemVariable:  ptr("item"),
							IndexVariable: ptr("idx"),
							MaxIterations: ptr(int32(50)),
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Loop through items")},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
			{
				Id: "body-click",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_CLICK,
					Params: &basactions.ActionDefinition_Click{
						Click: &basactions.ClickParams{
							Selector: "#item-${idx}",
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Click item")},
				},
				Position: &basbase.NodePosition{X: 100, Y: 100},
			},
			{
				Id: "body-screenshot",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SCREENSHOT,
					Params: &basactions.ActionDefinition_Screenshot{
						Screenshot: &basactions.ScreenshotParams{
							FullPage: ptr(false),
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Capture screenshot")},
				},
				Position: &basbase.NodePosition{X: 200, Y: 100},
			},
		},
		Edges: []*basworkflows.WorkflowEdgeV2{
			// Loop body connection
			{
				Id:           "e-loop-to-body1",
				Source:       "loop-items",
				Target:       "body-click",
				SourceHandle: ptr("loopbody"),
			},
			// Chain body steps
			{
				Id:     "e-body1-to-body2",
				Source: "body-click",
				Target: "body-screenshot",
			},
			// Return to loop
			{
				Id:           "e-body-return",
				Source:       "body-screenshot",
				Target:       "loop-items",
				TargetHandle: ptr("loopcontinue"),
			},
		},
	}

	// Step 1: Validate with V2 validator
	t.Run("V2Validation", func(t *testing.T) {
		v := &validator.Validator{}
		result := v.ValidateV2(definition)

		assert.True(t, result.Valid, "Workflow should be valid")
		assert.Empty(t, result.Errors, "Should have no errors")
		assert.Equal(t, 3, result.Stats.NodeCount)
		assert.Equal(t, 3, result.Stats.EdgeCount)
	})

	// Step 2: Convert to WorkflowSummary for compilation
	workflowSummary := &basapi.WorkflowSummary{
		Id:             workflowID.String(),
		Name:           "Loop Integration Test",
		FlowDefinition: definition,
	}

	// Step 3: Compile to ExecutionPlan
	t.Run("Compilation", func(t *testing.T) {
		plan, err := compiler.CompileWorkflow(workflowSummary)
		require.NoError(t, err)
		require.NotNil(t, plan)

		// Main plan should have 1 step (the loop)
		// because body nodes are extracted into LoopPlan
		require.Len(t, plan.Steps, 1, "Main plan should have 1 step (the loop)")

		loopStep := plan.Steps[0]
		assert.Equal(t, "loop-items", loopStep.NodeID)
		assert.Equal(t, compiler.StepLoop, loopStep.Type)

		// Verify loop params were extracted
		assert.Equal(t, "${items}", loopStep.Params["array_source"])
		assert.Equal(t, "item", loopStep.Params["item_variable"])
		assert.Equal(t, "idx", loopStep.Params["index_variable"])

		// Verify loop body was extracted
		require.NotNil(t, loopStep.LoopPlan, "Loop should have body plan")
		require.Len(t, loopStep.LoopPlan.Steps, 2, "Loop body should have 2 steps")

		// Verify body step order and types
		assert.Equal(t, "body-click", loopStep.LoopPlan.Steps[0].NodeID)
		assert.Equal(t, compiler.StepClick, loopStep.LoopPlan.Steps[0].Type)

		assert.Equal(t, "body-screenshot", loopStep.LoopPlan.Steps[1].NodeID)
		assert.Equal(t, compiler.StepScreenshot, loopStep.LoopPlan.Steps[1].Type)
	})
}

// TestIntegration_V2SubflowWorkflow tests subflow validation and compilation.
func TestIntegration_V2SubflowWorkflow(t *testing.T) {
	workflowID := uuid.New()
	childWorkflowID := uuid.New().String()

	definition := &basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "navigate-home",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
					Params: &basactions.ActionDefinition_Navigate{
						Navigate: &basactions.NavigateParams{
							Url: "https://example.com",
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Go to home")},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
			{
				Id: "run-login-subflow",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowId{
								WorkflowId: childWorkflowID,
							},
							WorkflowVersion: ptr(int32(1)),
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Run login flow")},
				},
				Position: &basbase.NodePosition{X: 200, Y: 0},
			},
			{
				Id: "click-dashboard",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_CLICK,
					Params: &basactions.ActionDefinition_Click{
						Click: &basactions.ClickParams{
							Selector: "#dashboard-link",
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Click dashboard")},
				},
				Position: &basbase.NodePosition{X: 400, Y: 0},
			},
		},
		Edges: []*basworkflows.WorkflowEdgeV2{
			{Id: "e1", Source: "navigate-home", Target: "run-login-subflow"},
			{Id: "e2", Source: "run-login-subflow", Target: "click-dashboard"},
		},
	}

	// Step 1: Validate
	t.Run("V2Validation", func(t *testing.T) {
		v := &validator.Validator{}
		result := v.ValidateV2(definition)

		assert.True(t, result.Valid, "Workflow should be valid")
		assert.Empty(t, result.Errors)
	})

	// Step 2: Compile
	workflowSummary := &basapi.WorkflowSummary{
		Id:             workflowID.String(),
		Name:           "Subflow Integration Test",
		FlowDefinition: definition,
	}

	t.Run("Compilation", func(t *testing.T) {
		plan, err := compiler.CompileWorkflow(workflowSummary)
		require.NoError(t, err)
		require.NotNil(t, plan)

		// Should have 3 steps in sequence
		require.Len(t, plan.Steps, 3)

		// Verify step types and order
		assert.Equal(t, compiler.StepNavigate, plan.Steps[0].Type)
		assert.Equal(t, "navigate-home", plan.Steps[0].NodeID)

		assert.Equal(t, compiler.StepSubflow, plan.Steps[1].Type)
		assert.Equal(t, "run-login-subflow", plan.Steps[1].NodeID)
		assert.Equal(t, childWorkflowID, plan.Steps[1].Params["workflow_id"])

		assert.Equal(t, compiler.StepClick, plan.Steps[2].Type)
		assert.Equal(t, "click-dashboard", plan.Steps[2].NodeID)

		// Verify edges are preserved
		require.Len(t, plan.Steps[0].OutgoingEdges, 1)
		assert.Equal(t, "run-login-subflow", plan.Steps[0].OutgoingEdges[0].TargetNode)

		require.Len(t, plan.Steps[1].OutgoingEdges, 1)
		assert.Equal(t, "click-dashboard", plan.Steps[1].OutgoingEdges[0].TargetNode)
	})
}

// TestIntegration_V2SubflowByPath tests subflow with workflow_path target.
func TestIntegration_V2SubflowByPath(t *testing.T) {
	workflowID := uuid.New()

	definition := &basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "run-action-subflow",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowPath{
								WorkflowPath: "actions/dismiss-tutorial.json",
							},
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Dismiss tutorial")},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
		},
	}

	// Validate
	v := &validator.Validator{}
	result := v.ValidateV2(definition)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Compile
	workflowSummary := &basapi.WorkflowSummary{
		Id:             workflowID.String(),
		Name:           "Subflow by Path Test",
		FlowDefinition: definition,
	}

	plan, err := compiler.CompileWorkflow(workflowSummary)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 1)

	assert.Equal(t, compiler.StepSubflow, plan.Steps[0].Type)
	assert.Equal(t, "actions/dismiss-tutorial.json", plan.Steps[0].Params["workflow_path"])
}

// TestIntegration_V2LoopValidationFailures tests that invalid loops are rejected.
func TestIntegration_V2LoopValidationFailures(t *testing.T) {
	tests := []struct {
		name          string
		loopParams    *basactions.LoopParams
		expectedCode  string
		expectInvalid bool
	}{
		{
			name: "foreach without array_source",
			loopParams: &basactions.LoopParams{
				LoopType:      basactions.LoopType_LOOP_TYPE_FOREACH,
				ItemVariable:  ptr("item"),
				MaxIterations: ptr(int32(100)),
			},
			expectedCode:  "WF_V2_LOOP_FOREACH_SOURCE_REQUIRED",
			expectInvalid: true,
		},
		{
			name: "repeat without count",
			loopParams: &basactions.LoopParams{
				LoopType:      basactions.LoopType_LOOP_TYPE_REPEAT,
				MaxIterations: ptr(int32(100)),
			},
			expectedCode:  "WF_V2_LOOP_REPEAT_COUNT_REQUIRED",
			expectInvalid: true,
		},
		{
			name: "unspecified loop_type",
			loopParams: &basactions.LoopParams{
				LoopType: basactions.LoopType_LOOP_TYPE_UNSPECIFIED,
			},
			expectedCode:  "WF_V2_LOOP_TYPE_REQUIRED",
			expectInvalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition := &basworkflows.WorkflowDefinitionV2{
				Nodes: []*basworkflows.WorkflowNodeV2{
					{
						Id: "loop-1",
						Action: &basactions.ActionDefinition{
							Type: basactions.ActionType_ACTION_TYPE_LOOP,
							Params: &basactions.ActionDefinition_Loop{
								Loop: tt.loopParams,
							},
							Metadata: &basactions.ActionMetadata{Label: ptr("Loop")},
						},
						Position: &basbase.NodePosition{X: 0, Y: 0},
					},
				},
			}

			v := &validator.Validator{}
			result := v.ValidateV2(definition)

			if tt.expectInvalid {
				assert.False(t, result.Valid)
				require.NotEmpty(t, result.Errors)
				found := false
				for _, err := range result.Errors {
					if err.Code == tt.expectedCode {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected error code %s not found in %v", tt.expectedCode, result.Errors)
			}
		})
	}
}

// TestIntegration_V2SubflowValidationFailures tests that invalid subflows are rejected.
func TestIntegration_V2SubflowValidationFailures(t *testing.T) {
	definition := &basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "subflow-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							// No target specified
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Subflow")},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
		},
	}

	v := &validator.Validator{}
	result := v.ValidateV2(definition)

	assert.False(t, result.Valid)
	require.NotEmpty(t, result.Errors)
	assert.Equal(t, "WF_V2_SUBFLOW_TARGET_REQUIRED", result.Errors[0].Code)
}

// TestIntegration_V2LoopWithRepeat tests a repeat loop type.
func TestIntegration_V2LoopWithRepeat(t *testing.T) {
	workflowID := uuid.New()

	definition := &basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "repeat-loop",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP,
					Params: &basactions.ActionDefinition_Loop{
						Loop: &basactions.LoopParams{
							LoopType:      basactions.LoopType_LOOP_TYPE_REPEAT,
							Count:         ptr(int32(5)),
							MaxIterations: ptr(int32(10)),
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Repeat 5 times")},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
			{
				Id: "body-wait",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_WAIT,
					Params: &basactions.ActionDefinition_Wait{
						Wait: &basactions.WaitParams{
							WaitFor: &basactions.WaitParams_DurationMs{DurationMs: 100},
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Wait 100ms")},
				},
				Position: &basbase.NodePosition{X: 100, Y: 100},
			},
		},
		Edges: []*basworkflows.WorkflowEdgeV2{
			{
				Id:           "e-loop-body",
				Source:       "repeat-loop",
				Target:       "body-wait",
				SourceHandle: ptr("loopbody"),
			},
			{
				Id:           "e-body-return",
				Source:       "body-wait",
				Target:       "repeat-loop",
				TargetHandle: ptr("loopcontinue"),
			},
		},
	}

	// Validate
	v := &validator.Validator{}
	result := v.ValidateV2(definition)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Compile
	workflowSummary := &basapi.WorkflowSummary{
		Id:             workflowID.String(),
		Name:           "Repeat Loop Test",
		FlowDefinition: definition,
	}

	plan, err := compiler.CompileWorkflow(workflowSummary)
	require.NoError(t, err)
	require.Len(t, plan.Steps, 1)

	loopStep := plan.Steps[0]
	assert.Equal(t, compiler.StepLoop, loopStep.Type)

	// Check repeat params
	// The count param should be extracted as "count" from proto
	assert.NotNil(t, loopStep.LoopPlan)
	require.Len(t, loopStep.LoopPlan.Steps, 1)
	assert.Equal(t, compiler.StepWait, loopStep.LoopPlan.Steps[0].Type)
}
