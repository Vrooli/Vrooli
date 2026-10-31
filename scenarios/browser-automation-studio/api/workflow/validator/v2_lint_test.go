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

func TestValidateV2_LoopForeachCases(t *testing.T) {
	for _, tc := range []struct {
		name         string
		params       *basactions.LoopParams
		valid        bool
		errorCode    string
		warningCodes []string
	}{
		{
			name: "valid",
			params: &basactions.LoopParams{
				LoopType:    basactions.LoopType_LOOP_TYPE_FOREACH,
				ArraySource: ptr("${items}"), ItemVariable: ptr("item"),
				MaxIterations: ptr(int32(100)),
			},
			valid: true,
		},
		{
			name: "missing array source",
			params: &basactions.LoopParams{
				LoopType: basactions.LoopType_LOOP_TYPE_FOREACH, ItemVariable: ptr("item"),
			},
			errorCode:    "WF_V2_LOOP_FOREACH_SOURCE_REQUIRED",
			warningCodes: []string{"WF_V2_LOOP_MAX_ITERATIONS"},
		},
		{
			name: "missing item variable",
			params: &basactions.LoopParams{
				LoopType: basactions.LoopType_LOOP_TYPE_FOREACH, ArraySource: ptr("${items}"),
			},
			valid:        true,
			warningCodes: []string{"WF_V2_LOOP_FOREACH_ITEM_VAR", "WF_V2_LOOP_MAX_ITERATIONS"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := (&Validator{}).ValidateV2(&basworkflows.WorkflowDefinitionV2{
				Nodes: []*basworkflows.WorkflowNodeV2{{
					Id: "loop-1",
					Action: &basactions.ActionDefinition{
						Type:     basactions.ActionType_ACTION_TYPE_LOOP,
						Params:   &basactions.ActionDefinition_Loop{Loop: tc.params},
						Metadata: &basactions.ActionMetadata{Label: ptr("Loop")},
					},
				}},
			})
			assert.Equal(t, tc.valid, result.Valid)
			if tc.errorCode == "" {
				assert.Empty(t, result.Errors)
			} else {
				require.Len(t, result.Errors, 1)
				assert.Equal(t, tc.errorCode, result.Errors[0].Code)
			}
			require.Len(t, result.Warnings, len(tc.warningCodes))
			actualWarnings := make([]string, 0, len(result.Warnings))
			for _, warning := range result.Warnings {
				actualWarnings = append(actualWarnings, warning.Code)
			}
			for _, code := range tc.warningCodes {
				assert.Contains(t, actualWarnings, code)
			}
		})
	}
}

func TestValidateV2_LoopRepeatCases(t *testing.T) {
	for _, tc := range []struct {
		name      string
		params    *basactions.LoopParams
		valid     bool
		errorCode string
	}{
		{
			name: "valid",
			params: &basactions.LoopParams{
				LoopType: basactions.LoopType_LOOP_TYPE_REPEAT,
				Count:    ptr(int32(5)), MaxIterations: ptr(int32(10)),
			},
			valid: true,
		},
		{
			name:      "missing count",
			params:    &basactions.LoopParams{LoopType: basactions.LoopType_LOOP_TYPE_REPEAT},
			errorCode: "WF_V2_LOOP_REPEAT_COUNT_REQUIRED",
		},
		{
			name:      "unspecified type",
			params:    &basactions.LoopParams{},
			errorCode: "WF_V2_LOOP_TYPE_REQUIRED",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := (&Validator{}).ValidateV2(&basworkflows.WorkflowDefinitionV2{
				Nodes: []*basworkflows.WorkflowNodeV2{{
					Id: "loop-1",
					Action: &basactions.ActionDefinition{
						Type:     basactions.ActionType_ACTION_TYPE_LOOP,
						Params:   &basactions.ActionDefinition_Loop{Loop: tc.params},
						Metadata: &basactions.ActionMetadata{Label: ptr("Loop")},
					},
				}},
			})
			assert.Equal(t, tc.valid, result.Valid)
			if tc.errorCode == "" {
				assert.Empty(t, result.Errors)
				return
			}
			require.Len(t, result.Errors, 1)
			assert.Equal(t, tc.errorCode, result.Errors[0].Code)
		})
	}
}

func TestValidateV2_SubflowTargets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target *basactions.SubflowParams
		valid  bool
		code   string
	}{
		{
			name: "workflow ID",
			target: &basactions.SubflowParams{Target: &basactions.SubflowParams_WorkflowId{
				WorkflowId: "550e8400-e29b-41d4-a716-446655440000",
			}},
			valid: true,
		},
		{
			name: "workflow path",
			target: &basactions.SubflowParams{Target: &basactions.SubflowParams_WorkflowPath{
				WorkflowPath: "actions/login.json",
			}},
			valid: true,
		},
		{name: "missing target", target: &basactions.SubflowParams{}, code: "WF_V2_SUBFLOW_TARGET_REQUIRED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := (&Validator{}).ValidateV2(&basworkflows.WorkflowDefinitionV2{
				Nodes: []*basworkflows.WorkflowNodeV2{{
					Id: "subflow-1",
					Action: &basactions.ActionDefinition{
						Type:     basactions.ActionType_ACTION_TYPE_SUBFLOW,
						Params:   &basactions.ActionDefinition_Subflow{Subflow: tc.target},
						Metadata: &basactions.ActionMetadata{Label: ptr("Run subflow")},
					},
				}},
			})
			assert.Equal(t, tc.valid, result.Valid)
			if tc.code == "" {
				assert.Empty(t, result.Errors)
				return
			}
			require.Len(t, result.Errors, 1)
			assert.Equal(t, tc.code, result.Errors[0].Code)
		})
	}
}

func TestValidateV2_GestureCases(t *testing.T) {
	for _, tc := range []struct {
		name          string
		params        *basactions.GestureParams
		label         string
		valid         bool
		errorCode     string
		checkWarnings bool
		warningCodes  []string
	}{
		{
			name: "sustained swipe",
			params: &basactions.GestureParams{
				GestureType: basactions.GestureType_GESTURE_TYPE_SWIPE,
				Selector:    ptr("[data-testid='canvas']"),
				Direction:   basactions.SwipeDirection_SWIPE_DIRECTION_RIGHT.Enum(),
				Distance:    ptr(int32(520)), DurationMs: ptr(int32(900)), Steps: ptr(int32(36)),
				StepDelayMs: ptr(int32(25)), TraceLabel: ptr("graph-sustained-pan"),
			},
			label: "Sustained pan", valid: true, checkWarnings: true,
		},
		{
			name: "swipe requires direction",
			params: &basactions.GestureParams{
				GestureType: basactions.GestureType_GESTURE_TYPE_SWIPE,
				TraceLabel:  ptr("graph-sustained-pan"),
			},
			label: "Sustained pan", errorCode: "WF_V2_GESTURE_DIRECTION_REQUIRED",
		},
		{
			name: "zoom warns without trace label",
			params: &basactions.GestureParams{
				GestureType: basactions.GestureType_GESTURE_TYPE_ZOOM, Steps: ptr(int32(8)),
			},
			label: "Wheel zoom", valid: true, checkWarnings: true,
			warningCodes: []string{"WF_V2_GESTURE_TRACE_LABEL_RECOMMENDED"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := (&Validator{}).ValidateV2(&basworkflows.WorkflowDefinitionV2{
				Nodes: []*basworkflows.WorkflowNodeV2{{
					Id: "gesture-1",
					Action: &basactions.ActionDefinition{
						Type:     basactions.ActionType_ACTION_TYPE_GESTURE,
						Params:   &basactions.ActionDefinition_Gesture{Gesture: tc.params},
						Metadata: &basactions.ActionMetadata{Label: ptr(tc.label)},
					},
				}},
			})
			assert.Equal(t, tc.valid, result.Valid)
			if tc.errorCode != "" {
				require.Len(t, result.Errors, 1)
				assert.Equal(t, tc.errorCode, result.Errors[0].Code)
			} else {
				assert.Empty(t, result.Errors)
			}
			if tc.checkWarnings {
				require.Len(t, result.Warnings, len(tc.warningCodes))
				actualWarnings := make([]string, 0, len(result.Warnings))
				for _, warning := range result.Warnings {
					actualWarnings = append(actualWarnings, warning.Code)
				}
				assert.ElementsMatch(t, tc.warningCodes, actualWarnings)
			}
		})
	}
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
