// Package compiler provides workflow compilation and format conversion utilities.
package compiler

import (
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// BuildActionDefinition creates a typed ActionDefinition proto from step type and params.
// This converts flat parameter maps (extracted from V2 action fields during compilation)
// into fully typed proto messages for type-safe execution.
// Used by ContractPlanCompiler to populate CompiledInstruction.Action.
func BuildActionDefinition(stepType string, params map[string]any) (*basactions.ActionDefinition, error) {
	action := &basactions.ActionDefinition{}

	// Map step type to ActionType enum
	actionType := mapStepTypeToActionType(stepType)
	action.Type = actionType

	// Build typed params based on action type
	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		action.Params = &basactions.ActionDefinition_Navigate{
			Navigate: buildNavigateParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_CLICK:
		action.Params = &basactions.ActionDefinition_Click{
			Click: buildClickParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_INPUT:
		action.Params = &basactions.ActionDefinition_Input{
			Input: buildInputParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_WAIT:
		action.Params = &basactions.ActionDefinition_Wait{
			Wait: buildWaitParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		action.Params = &basactions.ActionDefinition_Assert{
			Assert: buildAssertParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		action.Params = &basactions.ActionDefinition_Scroll{
			Scroll: buildScrollParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SELECT:
		action.Params = &basactions.ActionDefinition_SelectOption{
			SelectOption: buildSelectParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		action.Params = &basactions.ActionDefinition_Evaluate{
			Evaluate: buildEvaluateParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		action.Params = &basactions.ActionDefinition_Keyboard{
			Keyboard: buildKeyboardParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_HOVER:
		action.Params = &basactions.ActionDefinition_Hover{
			Hover: buildHoverParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		action.Params = &basactions.ActionDefinition_Screenshot{
			Screenshot: buildScreenshotParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		action.Params = &basactions.ActionDefinition_Focus{
			Focus: buildFocusParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_BLUR:
		action.Params = &basactions.ActionDefinition_Blur{
			Blur: buildBlurParams(params),
		}
	}

	// Build metadata if present in params
	action.Metadata = buildActionMetadata(params)

	return action, nil
}

// mapStepTypeToActionType converts step type string to ActionType enum.
// Delegates to typeconv.StringToActionType for the canonical implementation.
func mapStepTypeToActionType(stepType string) basactions.ActionType {
	return typeconv.StringToActionType(stepType)
}
