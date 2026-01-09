package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

func ptr[T any](v T) *T {
	return &v
}

func TestValidateV2_EmptyWorkflow(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{})

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WF_NODE_EMPTY", result.Errors[0].Code)
}

func TestValidateV2_NilWorkflow(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(nil)

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WF_V2_NIL", result.Errors[0].Code)
}

func TestValidateV2_ValidNavigateWorkflow(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "nav-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
					Params: &basactions.ActionDefinition_Navigate{
						Navigate: &basactions.NavigateParams{Url: "https://example.com"},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Go to example")},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
		},
	})

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateV2_LoopForeach_Valid(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
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
					Metadata: &basactions.ActionMetadata{Label: ptr("Loop items")},
				},
				Position: &basbase.NodePosition{X: 0, Y: 0},
			},
		},
	})

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.Empty(t, result.Warnings) // Has item_variable and max_iterations
}

func TestValidateV2_LoopForeach_MissingArraySource(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "loop-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP,
					Params: &basactions.ActionDefinition_Loop{
						Loop: &basactions.LoopParams{
							LoopType:     basactions.LoopType_LOOP_TYPE_FOREACH,
							ItemVariable: ptr("item"),
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Loop")},
				},
			},
		},
	})

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WF_V2_LOOP_FOREACH_SOURCE_REQUIRED", result.Errors[0].Code)
	// Should also warn about missing max_iterations
	assert.NotEmpty(t, result.Warnings)
}

func TestValidateV2_LoopForeach_MissingItemVariable(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "loop-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP,
					Params: &basactions.ActionDefinition_Loop{
						Loop: &basactions.LoopParams{
							LoopType:    basactions.LoopType_LOOP_TYPE_FOREACH,
							ArraySource: ptr("${items}"),
							// No item_variable
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Loop")},
				},
			},
		},
	})

	assert.True(t, result.Valid) // Missing item_variable is just a warning
	assert.Empty(t, result.Errors)
	// Should warn about missing item_variable and max_iterations
	var codes []string
	for _, w := range result.Warnings {
		codes = append(codes, w.Code)
	}
	assert.Contains(t, codes, "WF_V2_LOOP_FOREACH_ITEM_VAR")
	assert.Contains(t, codes, "WF_V2_LOOP_MAX_ITERATIONS")
}

func TestValidateV2_LoopRepeat_Valid(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "loop-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP,
					Params: &basactions.ActionDefinition_Loop{
						Loop: &basactions.LoopParams{
							LoopType:      basactions.LoopType_LOOP_TYPE_REPEAT,
							Count:         ptr(int32(5)),
							MaxIterations: ptr(int32(10)),
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Repeat 5x")},
				},
			},
		},
	})

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateV2_LoopRepeat_MissingCount(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "loop-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP,
					Params: &basactions.ActionDefinition_Loop{
						Loop: &basactions.LoopParams{
							LoopType: basactions.LoopType_LOOP_TYPE_REPEAT,
							// No count
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Repeat")},
				},
			},
		},
	})

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WF_V2_LOOP_REPEAT_COUNT_REQUIRED", result.Errors[0].Code)
}

func TestValidateV2_LoopUnspecifiedType(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "loop-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP,
					Params: &basactions.ActionDefinition_Loop{
						Loop: &basactions.LoopParams{
							// No loop_type specified
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Loop")},
				},
			},
		},
	})

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WF_V2_LOOP_TYPE_REQUIRED", result.Errors[0].Code)
}

func TestValidateV2_Subflow_ValidByID(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "subflow-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowId{
								WorkflowId: "550e8400-e29b-41d4-a716-446655440000",
							},
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Run subflow")},
				},
			},
		},
	})

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateV2_Subflow_ValidByPath(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "subflow-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_SUBFLOW,
					Params: &basactions.ActionDefinition_Subflow{
						Subflow: &basactions.SubflowParams{
							Target: &basactions.SubflowParams_WorkflowPath{
								WorkflowPath: "actions/login.json",
							},
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Run login")},
				},
			},
		},
	})

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateV2_Subflow_MissingTarget(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
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
			},
		},
	})

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WF_V2_SUBFLOW_TARGET_REQUIRED", result.Errors[0].Code)
}

func TestValidateV2_Click_MissingSelector(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "click-1",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_CLICK,
					Params: &basactions.ActionDefinition_Click{
						Click: &basactions.ClickParams{
							// No selector
						},
					},
					Metadata: &basactions.ActionMetadata{Label: ptr("Click")},
				},
			},
		},
	})

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WF_V2_SELECTOR_REQUIRED", result.Errors[0].Code)
}

func TestValidateV2_EdgeValidation(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{Id: "a", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}, Metadata: &basactions.ActionMetadata{Label: ptr("Nav")}}},
			{Id: "b", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SCREENSHOT, Params: &basactions.ActionDefinition_Screenshot{Screenshot: &basactions.ScreenshotParams{FullPage: ptr(true)}}, Metadata: &basactions.ActionMetadata{Label: ptr("Screenshot")}}},
		},
		Edges: []*basworkflows.WorkflowEdgeV2{
			{Id: "e1", Source: "a", Target: "b"},
			{Id: "e2", Source: "b", Target: "unknown"}, // Unknown target
			{Id: "e3", Source: "a", Target: "a"},       // Self-loop
		},
	})

	assert.False(t, result.Valid)

	var codes []string
	for _, e := range result.Errors {
		codes = append(codes, e.Code)
	}
	assert.Contains(t, codes, "WF_EDGE_TARGET_UNKNOWN")
	assert.Contains(t, codes, "WF_EDGE_CYCLE_SELF")
}

func TestValidateV2_DuplicateNodeID(t *testing.T) {
	v := &Validator{}
	result := v.ValidateV2(&basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{Id: "node-1", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://example.com"}}, Metadata: &basactions.ActionMetadata{Label: ptr("Nav 1")}}},
			{Id: "node-1", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SCREENSHOT, Params: &basactions.ActionDefinition_Screenshot{Screenshot: &basactions.ScreenshotParams{FullPage: ptr(true)}}, Metadata: &basactions.ActionMetadata{Label: ptr("Nav 2")}}}, // Duplicate ID
		},
	})

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WF_NODE_ID_DUPLICATE", result.Errors[0].Code)
}
