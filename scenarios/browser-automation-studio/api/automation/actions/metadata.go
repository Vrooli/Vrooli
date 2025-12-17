package actions

// ActionMetadata describes the behavioral properties of an action type.
// This metadata is shared between recording and execution to ensure consistent
// handling of actions throughout the automation pipeline.
type ActionMetadata struct {
	// Type is the action type this metadata describes.
	Type ActionType

	// Category groups related actions for organization and filtering.
	Category ActionCategory

	// NeedsSelectorWait indicates whether this action requires waiting for
	// its target element to exist before execution. Actions that interact
	// with DOM elements typically need this.
	NeedsSelectorWait bool

	// TriggersDOMChanges indicates whether this action might modify the DOM,
	// which would require subsequent actions to wait for stabilization.
	// Examples: click (might trigger navigation), type (might trigger autocomplete)
	TriggersDOMChanges bool

	// RequiresElement indicates whether this action needs a target element.
	// Actions like navigate or wait don't need an element selector.
	RequiresElement bool

	// CanFail indicates whether this action can fail in a recoverable way.
	// Used for retry logic and failure handling.
	CanFail bool

	// ProducesData indicates whether this action produces output data
	// (e.g., extract, screenshot, getCookie).
	ProducesData bool

	// IsControlFlow indicates whether this is a control flow action
	// (conditional, loop, subflow) rather than a browser interaction.
	IsControlFlow bool

	// Description provides a human-readable description of the action.
	Description string
}

// ActionCategory groups related action types.
type ActionCategory string

const (
	CategoryNavigation   ActionCategory = "navigation"
	CategoryMouse        ActionCategory = "mouse"
	CategoryKeyboard     ActionCategory = "keyboard"
	CategoryFocus        ActionCategory = "focus"
	CategoryForm         ActionCategory = "form"
	CategoryAssertion    ActionCategory = "assertion"
	CategoryData         ActionCategory = "data"
	CategoryContext      ActionCategory = "context"
	CategoryControlFlow  ActionCategory = "control_flow"
	CategoryStorage      ActionCategory = "storage"
	CategoryNetwork      ActionCategory = "network"
	CategoryCustom       ActionCategory = "custom"
)

// GetMetadata returns the metadata for an action type.
// Returns default metadata for unknown action types.
func GetMetadata(actionType ActionType) ActionMetadata {
	if meta, ok := Registry[actionType]; ok {
		return meta
	}
	return defaultMetadata(actionType)
}

// NeedsSelectorWait returns whether the action type requires waiting for its selector.
func NeedsSelectorWait(actionType ActionType) bool {
	return GetMetadata(actionType).NeedsSelectorWait
}

// TriggersDOMChanges returns whether the action type might modify the DOM.
func TriggersDOMChanges(actionType ActionType) bool {
	return GetMetadata(actionType).TriggersDOMChanges
}

// RequiresElement returns whether the action type needs a target element.
func RequiresElement(actionType ActionType) bool {
	return GetMetadata(actionType).RequiresElement
}

// IsControlFlow returns whether the action type is a control flow action.
func IsControlFlow(actionType ActionType) bool {
	return GetMetadata(actionType).IsControlFlow
}

// GetCategory returns the category for an action type.
func GetCategory(actionType ActionType) ActionCategory {
	return GetMetadata(actionType).Category
}

func defaultMetadata(actionType ActionType) ActionMetadata {
	return ActionMetadata{
		Type:               actionType,
		Category:           CategoryCustom,
		NeedsSelectorWait:  false,
		TriggersDOMChanges: false,
		RequiresElement:    false,
		CanFail:            true,
		ProducesData:       false,
		IsControlFlow:      false,
		Description:        "Custom action",
	}
}
