package telemetry

import (
	"github.com/vrooli/browser-automation-studio/automation/compiler"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// setTypedParams sets the typed params on the ActionDefinition based on action type.
func setTypedParams(def *basactions.ActionDefinition, actionType basactions.ActionType, params map[string]any) {
	if params == nil {
		return
	}

	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		def.Params = &basactions.ActionDefinition_Navigate{
			Navigate: compiler.BuildNavigateParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_CLICK:
		def.Params = &basactions.ActionDefinition_Click{
			Click: compiler.BuildClickParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_INPUT:
		def.Params = &basactions.ActionDefinition_Input{
			Input: compiler.BuildInputParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		def.Params = &basactions.ActionDefinition_Scroll{
			Scroll: compiler.BuildScrollParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_HOVER:
		def.Params = &basactions.ActionDefinition_Hover{
			Hover: compiler.BuildHoverParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		def.Params = &basactions.ActionDefinition_Focus{
			Focus: compiler.BuildFocusParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_BLUR:
		def.Params = &basactions.ActionDefinition_Blur{
			Blur: compiler.BuildBlurParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SELECT:
		def.Params = &basactions.ActionDefinition_SelectOption{
			SelectOption: compiler.BuildSelectParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		def.Params = &basactions.ActionDefinition_Keyboard{
			Keyboard: compiler.BuildKeyboardParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_WAIT:
		def.Params = &basactions.ActionDefinition_Wait{
			Wait: compiler.BuildWaitParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		def.Params = &basactions.ActionDefinition_Assert{
			Assert: compiler.BuildAssertParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		def.Params = &basactions.ActionDefinition_Screenshot{
			Screenshot: compiler.BuildScreenshotParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		def.Params = &basactions.ActionDefinition_Evaluate{
			Evaluate: compiler.BuildEvaluateParams(params),
		}
	default:
		// Don't set params for unknown types
	}
}
