// Package compiler provides workflow compilation and format conversion utilities.
package compiler

import (
	"fmt"

	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// actionDefinitionBuilder builds the typed parameter oneof for an executable
// action. Keeping enum dispatch in this table makes the supported action set
// and its sole parameter builder explicit.
type actionDefinitionBuilder func(map[string]any) *basactions.ActionDefinition

var actionDefinitionBuilders = map[basactions.ActionType]actionDefinitionBuilder{
	basactions.ActionType_ACTION_TYPE_NAVIGATE: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Navigate{Navigate: typeconv.BuildNavigateParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_CLICK: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Click{Click: typeconv.BuildClickParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_INPUT: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Input{Input: typeconv.BuildInputParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_WAIT: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Wait{Wait: typeconv.BuildWaitParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_ASSERT: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Assert{Assert: typeconv.BuildAssertParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_SCROLL: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Scroll{Scroll: typeconv.BuildScrollParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_SELECT: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_SelectOption{SelectOption: typeconv.BuildSelectParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_EVALUATE: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Evaluate{Evaluate: typeconv.BuildEvaluateParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_KEYBOARD: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Keyboard{Keyboard: typeconv.BuildKeyboardParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_DRAG_DROP: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_DragDrop{DragDrop: typeconv.BuildDragDropParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_HOVER: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Hover{Hover: typeconv.BuildHoverParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_SCREENSHOT: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Screenshot{Screenshot: typeconv.BuildScreenshotParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_FOCUS: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Focus{Focus: typeconv.BuildFocusParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_BLUR: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Blur{Blur: typeconv.BuildBlurParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_SUBFLOW: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Subflow{Subflow: typeconv.BuildSubflowParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_EXTRACT: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Extract{Extract: typeconv.BuildExtractParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_SHORTCUT: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Shortcut{Shortcut: typeconv.BuildShortcutParams(params)}}
	},
	basactions.ActionType_ACTION_TYPE_GESTURE: func(params map[string]any) *basactions.ActionDefinition {
		return &basactions.ActionDefinition{Params: &basactions.ActionDefinition_Gesture{Gesture: typeconv.BuildGestureParams(params)}}
	},
}

// BuildActionDefinition creates a typed ActionDefinition proto from step type and params.
// This converts flat parameter maps (extracted from V2 action fields during compilation)
// into fully typed proto messages for type-safe execution.
// Used by CompileWorkflowToContracts to populate CompiledInstruction.Action.
// Returns an error if stepType is unknown or has no executable typed-parameter builder.
func BuildActionDefinition(stepType string, params map[string]any) (*basactions.ActionDefinition, error) {
	actionType := enums.StringToActionType(stepType)
	if actionType == basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return nil, fmt.Errorf("unknown action type: %q", stepType)
	}

	build, ok := actionDefinitionBuilders[actionType]
	if !ok {
		return nil, fmt.Errorf("no params builder for action type %q (enum: %s)", stepType, actionType.String())
	}

	action := build(params)
	action.Type = actionType
	action.Metadata = typeconv.BuildActionMetadata(params)
	return action, nil
}
