// Package actions provides the unified action type system for browser automation.
// This is the single source of truth for action types used by both recording
// (capturing human actions) and execution (replaying them programmatically).
//
// Recording and execution are two sides of the same coin:
//   - Recording: Captures what a human does in the browser
//   - Execution: Replays those actions programmatically
//
// Both share the same action types and behavioral properties.
package actions

// ActionType represents a supported browser automation action.
// This type is shared between recording and execution systems.
type ActionType string

// Action type constants - the canonical set of supported actions.
// These are used by both the compiler (for execution) and live-capture (for recording).
const (
	// Navigation
	Navigate ActionType = "navigate"

	// Mouse interactions
	Click    ActionType = "click"
	Hover    ActionType = "hover"
	DragDrop ActionType = "dragDrop"
	Scroll   ActionType = "scroll"
	Gesture  ActionType = "gesture"
	Rotate   ActionType = "rotate"

	// Keyboard/input
	TypeInput ActionType = "type"
	Keyboard  ActionType = "keyboard"
	Shortcut  ActionType = "shortcut"

	// Focus management
	Focus ActionType = "focus"
	Blur  ActionType = "blur"

	// Form interactions
	Select     ActionType = "select"
	UploadFile ActionType = "uploadFile"

	// Waiting and assertions
	Wait       ActionType = "wait"
	Assert     ActionType = "assert"
	Screenshot ActionType = "screenshot"

	// Data extraction and manipulation
	Extract     ActionType = "extract"
	Evaluate    ActionType = "evaluate"
	SetVariable ActionType = "setVariable"
	UseVariable ActionType = "useVariable"

	// Context switching
	TabSwitch   ActionType = "tabSwitch"
	FrameSwitch ActionType = "frameSwitch"

	// Control flow
	Conditional ActionType = "conditional"
	Loop        ActionType = "loop"
	Subflow     ActionType = "subflow"

	// Cookie and storage management
	SetCookie    ActionType = "setCookie"
	GetCookie    ActionType = "getCookie"
	ClearCookie  ActionType = "clearCookie"
	SetStorage   ActionType = "setStorage"
	GetStorage   ActionType = "getStorage"
	ClearStorage ActionType = "clearStorage"

	// Network
	NetworkMock ActionType = "networkMock"

	// Custom/extensibility
	Custom ActionType = "custom"
)

// Loop control flow targets (used in graph execution)
const (
	LoopContinueTarget = "__loop_continue__"
	LoopBreakTarget    = "__loop_break__"
)

// Loop edge handles (for workflow graph connections)
const (
	LoopHandleBody     = "loopbody"
	LoopHandleAfter    = "loopafter"
	LoopHandleBreak    = "loopbreak"
	LoopHandleContinue = "loopcontinue"

	LoopConditionBody     = "loop_body"
	LoopConditionAfter    = "loop_next"
	LoopConditionBreak    = "loop_break"
	LoopConditionContinue = "loop_continue"
)

// String returns the string representation of the action type.
func (a ActionType) String() string {
	return string(a)
}

// IsValid returns true if this is a known action type.
func (a ActionType) IsValid() bool {
	_, ok := Registry[a]
	return ok
}

// ParseActionType converts a string to ActionType, returning Custom for unknown types.
func ParseActionType(s string) ActionType {
	at := ActionType(s)
	if at.IsValid() {
		return at
	}
	return Custom
}

// ValidateActionType returns an error if the action type is not supported.
func ValidateActionType(s string) error {
	at := ActionType(s)
	if at.IsValid() {
		return nil
	}
	return &UnsupportedActionError{ActionType: s}
}

// UnsupportedActionError indicates an unknown action type was used.
type UnsupportedActionError struct {
	ActionType string
}

func (e *UnsupportedActionError) Error() string {
	return "unsupported action type: " + e.ActionType
}
