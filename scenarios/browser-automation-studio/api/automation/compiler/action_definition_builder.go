// Package compiler provides workflow compilation and format conversion utilities.
package compiler

import (
	"sort"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// BuildActionDefinition creates a typed ActionDefinition proto from step type and params.
// This converts flat parameter maps (extracted from V2 action fields during compilation)
// into fully typed proto messages for type-safe execution.
// Used by CompileWorkflowToContracts to populate CompiledInstruction.Action.
func BuildActionDefinition(stepType string, params map[string]any) (*basactions.ActionDefinition, error) {
	action := &basactions.ActionDefinition{}

	// Map step type to ActionType enum
	actionType := enums.StringToActionType(stepType)
	action.Type = actionType

	// Build typed params based on action type using shared typeconv builders
	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		action.Params = &basactions.ActionDefinition_Navigate{
			Navigate: typeconv.BuildNavigateParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_CLICK:
		action.Params = &basactions.ActionDefinition_Click{
			Click: typeconv.BuildClickParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_INPUT:
		action.Params = &basactions.ActionDefinition_Input{
			Input: typeconv.BuildInputParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_WAIT:
		action.Params = &basactions.ActionDefinition_Wait{
			Wait: typeconv.BuildWaitParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		action.Params = &basactions.ActionDefinition_Assert{
			Assert: typeconv.BuildAssertParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		action.Params = &basactions.ActionDefinition_Scroll{
			Scroll: typeconv.BuildScrollParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SELECT:
		action.Params = &basactions.ActionDefinition_SelectOption{
			SelectOption: typeconv.BuildSelectParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		action.Params = &basactions.ActionDefinition_Evaluate{
			Evaluate: typeconv.BuildEvaluateParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		action.Params = &basactions.ActionDefinition_Keyboard{
			Keyboard: typeconv.BuildKeyboardParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_HOVER:
		action.Params = &basactions.ActionDefinition_Hover{
			Hover: typeconv.BuildHoverParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		action.Params = &basactions.ActionDefinition_Screenshot{
			Screenshot: typeconv.BuildScreenshotParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		action.Params = &basactions.ActionDefinition_Focus{
			Focus: typeconv.BuildFocusParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_BLUR:
		action.Params = &basactions.ActionDefinition_Blur{
			Blur: typeconv.BuildBlurParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SUBFLOW:
		action.Params = &basactions.ActionDefinition_Subflow{
			Subflow: typeconv.BuildSubflowParams(params),
		}
	}

	// Build metadata if present in params
	action.Metadata = typeconv.BuildActionMetadata(params)

	if actionType != basactions.ActionType_ACTION_TYPE_UNSPECIFIED && action.Params == nil {
		paramKeys := make([]string, 0, len(params))
		for key := range params {
			paramKeys = append(paramKeys, key)
		}
		sort.Strings(paramKeys)
		logrus.WithFields(logrus.Fields{
			"step_type":   stepType,
			"action_type": actionType.String(),
			"param_keys":  paramKeys,
		}).Warn("ActionDefinition params missing for step")
	}

	return action, nil
}
