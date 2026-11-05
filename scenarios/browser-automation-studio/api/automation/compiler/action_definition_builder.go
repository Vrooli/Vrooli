// Package compiler provides workflow compilation and format conversion utilities.
package compiler

import (
	"fmt"

	"github.com/vrooli/browser-automation-studio/internal/enums"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type actionParamsBuilder func(map[string]any) proto.Message

func adaptActionParamsBuilder[T proto.Message](build func(map[string]any) T) actionParamsBuilder {
	return func(params map[string]any) proto.Message { return build(params) }
}

// Parameter conversion remains explicit because each action accepts different
// legacy aliases. The proto JSON field names and oneof relationship are resolved
// from the generated descriptor rather than repeated in this registry.
var actionParamsBuilders = map[string]actionParamsBuilder{
	"navigate":     adaptActionParamsBuilder(BuildNavigateParams),
	"click":        adaptActionParamsBuilder(BuildClickParams),
	"input":        adaptActionParamsBuilder(BuildInputParams),
	"wait":         adaptActionParamsBuilder(BuildWaitParams),
	"assert":       adaptActionParamsBuilder(BuildAssertParams),
	"scroll":       adaptActionParamsBuilder(BuildScrollParams),
	"selectOption": adaptActionParamsBuilder(BuildSelectParams),
	"evaluate":     adaptActionParamsBuilder(BuildEvaluateParams),
	"keyboard":     adaptActionParamsBuilder(BuildKeyboardParams),
	"dragDrop":     adaptActionParamsBuilder(BuildDragDropParams),
	"hover":        adaptActionParamsBuilder(BuildHoverParams),
	"screenshot":   adaptActionParamsBuilder(BuildScreenshotParams),
	"focus":        adaptActionParamsBuilder(BuildFocusParams),
	"blur":         adaptActionParamsBuilder(BuildBlurParams),
	"subflow":      adaptActionParamsBuilder(BuildSubflowParams),
	"extract":      adaptActionParamsBuilder(BuildExtractParams),
	"shortcut":     adaptActionParamsBuilder(BuildShortcutParams),
	"gesture":      adaptActionParamsBuilder(BuildGestureParams),
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

	paramsField := enums.ActionTypeParamsField(actionType)
	build, ok := actionParamsBuilders[paramsField]
	if !ok {
		return nil, fmt.Errorf("no params builder for action type %q (enum: %s)", stepType, actionType.String())
	}

	paramsMessage := build(params)
	action := &basactions.ActionDefinition{Type: actionType}
	field := action.ProtoReflect().Descriptor().Fields().ByJSONName(paramsField)
	if field == nil {
		return nil, fmt.Errorf("no params field for action type %q (enum: %s)", stepType, actionType.String())
	}
	action.ProtoReflect().Set(field, protoreflect.ValueOfMessage(paramsMessage.ProtoReflect()))
	action.Metadata = BuildActionMetadata(params)
	return action, nil
}
