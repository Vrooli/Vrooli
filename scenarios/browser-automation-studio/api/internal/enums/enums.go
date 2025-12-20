// Package enums provides proto enum conversion utilities.
//
// This package contains ONLY string <-> proto enum conversions with no other dependencies.
// It exists to avoid import cycles - packages like automation/events, automation/compiler,
// and services/workflow all need enum conversions but would create cycles if importing protoconv.
//
// NOTE: internal/protoconv/enum_convert.go has the same functions (re-exported from here).
// New code should import this package directly.
package enums

import (
	"strings"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

// =============================================================================
// ACTION TYPE CONVERTERS
// =============================================================================

// StringToActionType converts an action type string to the proto ActionType enum.
// Handles various aliases for each action type (e.g., "goto" -> NAVIGATE, "fill"/"type" -> INPUT).
// Returns ACTION_TYPE_UNSPECIFIED for unrecognized types.
func StringToActionType(actionType string) basactions.ActionType {
	switch strings.ToLower(strings.TrimSpace(actionType)) {
	case "navigate", "goto":
		return basactions.ActionType_ACTION_TYPE_NAVIGATE
	case "click":
		return basactions.ActionType_ACTION_TYPE_CLICK
	case "input", "type", "fill":
		return basactions.ActionType_ACTION_TYPE_INPUT
	case "wait":
		return basactions.ActionType_ACTION_TYPE_WAIT
	case "assert":
		return basactions.ActionType_ACTION_TYPE_ASSERT
	case "scroll":
		return basactions.ActionType_ACTION_TYPE_SCROLL
	case "select", "selectoption":
		return basactions.ActionType_ACTION_TYPE_SELECT
	case "evaluate", "eval":
		return basactions.ActionType_ACTION_TYPE_EVALUATE
	case "keyboard", "keypress", "press":
		return basactions.ActionType_ACTION_TYPE_KEYBOARD
	case "hover":
		return basactions.ActionType_ACTION_TYPE_HOVER
	case "screenshot":
		return basactions.ActionType_ACTION_TYPE_SCREENSHOT
	case "focus":
		return basactions.ActionType_ACTION_TYPE_FOCUS
	case "blur":
		return basactions.ActionType_ACTION_TYPE_BLUR
	case "subflow":
		return basactions.ActionType_ACTION_TYPE_SUBFLOW
	case "setvariable", "set_variable":
		return basactions.ActionType_ACTION_TYPE_SET_VARIABLE
	case "loop":
		return basactions.ActionType_ACTION_TYPE_LOOP
	case "conditional":
		return basactions.ActionType_ACTION_TYPE_CONDITIONAL
	default:
		return basactions.ActionType_ACTION_TYPE_UNSPECIFIED
	}
}

// ActionTypeToString converts an ActionType enum to a proto-aligned string representation.
// Returns "unknown" for unrecognized types.
func ActionTypeToString(actionType basactions.ActionType) string {
	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		return "navigate"
	case basactions.ActionType_ACTION_TYPE_CLICK:
		return "click"
	case basactions.ActionType_ACTION_TYPE_INPUT:
		return "input"
	case basactions.ActionType_ACTION_TYPE_WAIT:
		return "wait"
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		return "assert"
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		return "scroll"
	case basactions.ActionType_ACTION_TYPE_SELECT:
		return "select"
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		return "evaluate"
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		return "keyboard"
	case basactions.ActionType_ACTION_TYPE_HOVER:
		return "hover"
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		return "screenshot"
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		return "focus"
	case basactions.ActionType_ACTION_TYPE_BLUR:
		return "blur"
	case basactions.ActionType_ACTION_TYPE_SUBFLOW:
		return "subflow"
	case basactions.ActionType_ACTION_TYPE_EXTRACT:
		return "extract"
	case basactions.ActionType_ACTION_TYPE_UPLOAD_FILE:
		return "uploadFile"
	case basactions.ActionType_ACTION_TYPE_DOWNLOAD:
		return "download"
	case basactions.ActionType_ACTION_TYPE_FRAME_SWITCH:
		return "frameSwitch"
	case basactions.ActionType_ACTION_TYPE_TAB_SWITCH:
		return "tabSwitch"
	case basactions.ActionType_ACTION_TYPE_COOKIE_STORAGE:
		return "setCookie"
	case basactions.ActionType_ACTION_TYPE_SHORTCUT:
		return "shortcut"
	case basactions.ActionType_ACTION_TYPE_DRAG_DROP:
		return "dragDrop"
	case basactions.ActionType_ACTION_TYPE_GESTURE:
		return "gesture"
	case basactions.ActionType_ACTION_TYPE_NETWORK_MOCK:
		return "networkMock"
	case basactions.ActionType_ACTION_TYPE_ROTATE:
		return "rotate"
	case basactions.ActionType_ACTION_TYPE_SET_VARIABLE:
		return "setVariable"
	case basactions.ActionType_ACTION_TYPE_LOOP:
		return "loop"
	case basactions.ActionType_ACTION_TYPE_CONDITIONAL:
		return "conditional"
	default:
		return "unknown"
	}
}

// =============================================================================
// SELECTOR TYPE CONVERTERS
// =============================================================================

// StringToSelectorType converts a selector type string to the proto SelectorType enum.
func StringToSelectorType(selectorType string) basbase.SelectorType {
	switch strings.ToLower(strings.TrimSpace(selectorType)) {
	case "css":
		return basbase.SelectorType_SELECTOR_TYPE_CSS
	case "xpath":
		return basbase.SelectorType_SELECTOR_TYPE_XPATH
	case "id":
		return basbase.SelectorType_SELECTOR_TYPE_ID
	case "data-testid", "datatestid", "testid":
		return basbase.SelectorType_SELECTOR_TYPE_DATA_TESTID
	case "aria", "aria-label":
		return basbase.SelectorType_SELECTOR_TYPE_ARIA
	case "text":
		return basbase.SelectorType_SELECTOR_TYPE_TEXT
	case "role":
		return basbase.SelectorType_SELECTOR_TYPE_ROLE
	case "placeholder":
		return basbase.SelectorType_SELECTOR_TYPE_PLACEHOLDER
	case "alt", "alt-text", "alttext":
		return basbase.SelectorType_SELECTOR_TYPE_ALT_TEXT
	case "title":
		return basbase.SelectorType_SELECTOR_TYPE_TITLE
	default:
		return basbase.SelectorType_SELECTOR_TYPE_UNSPECIFIED
	}
}

// SelectorTypeToString converts a SelectorType enum to its canonical string representation.
func SelectorTypeToString(selectorType basbase.SelectorType) string {
	switch selectorType {
	case basbase.SelectorType_SELECTOR_TYPE_CSS:
		return "css"
	case basbase.SelectorType_SELECTOR_TYPE_XPATH:
		return "xpath"
	case basbase.SelectorType_SELECTOR_TYPE_ID:
		return "id"
	case basbase.SelectorType_SELECTOR_TYPE_DATA_TESTID:
		return "data-testid"
	case basbase.SelectorType_SELECTOR_TYPE_ARIA:
		return "aria"
	case basbase.SelectorType_SELECTOR_TYPE_TEXT:
		return "text"
	case basbase.SelectorType_SELECTOR_TYPE_ROLE:
		return "role"
	case basbase.SelectorType_SELECTOR_TYPE_PLACEHOLDER:
		return "placeholder"
	case basbase.SelectorType_SELECTOR_TYPE_ALT_TEXT:
		return "alt-text"
	case basbase.SelectorType_SELECTOR_TYPE_TITLE:
		return "title"
	default:
		return "unknown"
	}
}

// =============================================================================
// EXECUTION STATUS CONVERTERS
// =============================================================================

// StringToExecutionStatus converts an execution status string to the proto ExecutionStatus enum.
func StringToExecutionStatus(s string) basbase.ExecutionStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "PENDING", "EXECUTION_STATUS_PENDING":
		return basbase.ExecutionStatus_EXECUTION_STATUS_PENDING
	case "RUNNING", "EXECUTION_STATUS_RUNNING":
		return basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case "COMPLETED", "EXECUTION_STATUS_COMPLETED":
		return basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case "FAILED", "EXECUTION_STATUS_FAILED":
		return basbase.ExecutionStatus_EXECUTION_STATUS_FAILED
	case "CANCELLED", "CANCELED", "EXECUTION_STATUS_CANCELLED":
		return basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	default:
		return basbase.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	}
}

// ExecutionStatusToString converts an ExecutionStatus enum to its canonical string representation.
func ExecutionStatusToString(status basbase.ExecutionStatus) string {
	switch status {
	case basbase.ExecutionStatus_EXECUTION_STATUS_PENDING:
		return "pending"
	case basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING:
		return "running"
	case basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED:
		return "completed"
	case basbase.ExecutionStatus_EXECUTION_STATUS_FAILED:
		return "failed"
	case basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unknown"
	}
}

// =============================================================================
// ASSERTION MODE CONVERTERS
// =============================================================================

// StringToAssertionMode converts an assertion mode string to the proto AssertionMode enum.
func StringToAssertionMode(s string) basbase.AssertionMode {
	normalized := strings.ToLower(strings.TrimSpace(s))
	if strings.HasPrefix(normalized, "assertion_mode_") {
		normalized = strings.TrimPrefix(normalized, "assertion_mode_")
	}
	switch normalized {
	case "exists":
		return basbase.AssertionMode_ASSERTION_MODE_EXISTS
	case "not_exists":
		return basbase.AssertionMode_ASSERTION_MODE_NOT_EXISTS
	case "visible":
		return basbase.AssertionMode_ASSERTION_MODE_VISIBLE
	case "hidden":
		return basbase.AssertionMode_ASSERTION_MODE_HIDDEN
	case "text_equals":
		return basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS
	case "text_contains":
		return basbase.AssertionMode_ASSERTION_MODE_TEXT_CONTAINS
	case "attribute_equals":
		return basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS
	case "attribute_contains":
		return basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS
	default:
		return basbase.AssertionMode_ASSERTION_MODE_UNSPECIFIED
	}
}

// AssertionModeToString converts an AssertionMode enum to its canonical string representation.
func AssertionModeToString(mode basbase.AssertionMode) string {
	switch mode {
	case basbase.AssertionMode_ASSERTION_MODE_EXISTS:
		return "exists"
	case basbase.AssertionMode_ASSERTION_MODE_NOT_EXISTS:
		return "not_exists"
	case basbase.AssertionMode_ASSERTION_MODE_VISIBLE:
		return "visible"
	case basbase.AssertionMode_ASSERTION_MODE_HIDDEN:
		return "hidden"
	case basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS:
		return "text_equals"
	case basbase.AssertionMode_ASSERTION_MODE_TEXT_CONTAINS:
		return "text_contains"
	case basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS:
		return "attribute_equals"
	case basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS:
		return "attribute_contains"
	default:
		return "unknown"
	}
}

// =============================================================================
// LOG LEVEL CONVERTERS
// =============================================================================

// StringToLogLevel converts a log level string to the proto LogLevel enum.
func StringToLogLevel(s string) basbase.LogLevel {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "DEBUG", "LOG_LEVEL_DEBUG":
		return basbase.LogLevel_LOG_LEVEL_DEBUG
	case "INFO", "LOG_LEVEL_INFO":
		return basbase.LogLevel_LOG_LEVEL_INFO
	case "WARN", "WARNING", "LOG_LEVEL_WARN":
		return basbase.LogLevel_LOG_LEVEL_WARN
	case "ERROR", "LOG_LEVEL_ERROR":
		return basbase.LogLevel_LOG_LEVEL_ERROR
	default:
		return basbase.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}

// LogLevelToString converts a LogLevel enum to its canonical string representation.
func LogLevelToString(level basbase.LogLevel) string {
	switch level {
	case basbase.LogLevel_LOG_LEVEL_DEBUG:
		return "debug"
	case basbase.LogLevel_LOG_LEVEL_INFO:
		return "info"
	case basbase.LogLevel_LOG_LEVEL_WARN:
		return "warn"
	case basbase.LogLevel_LOG_LEVEL_ERROR:
		return "error"
	default:
		return "unknown"
	}
}

// =============================================================================
// NETWORK EVENT TYPE CONVERTERS
// =============================================================================

// StringToNetworkEventType converts a network event type string to the proto NetworkEventType enum.
func StringToNetworkEventType(s string) basbase.NetworkEventType {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "request":
		return basbase.NetworkEventType_NETWORK_EVENT_TYPE_REQUEST
	case "response":
		return basbase.NetworkEventType_NETWORK_EVENT_TYPE_RESPONSE
	case "failure", "failed":
		return basbase.NetworkEventType_NETWORK_EVENT_TYPE_FAILURE
	default:
		return basbase.NetworkEventType_NETWORK_EVENT_TYPE_UNSPECIFIED
	}
}

// NetworkEventTypeToString converts a NetworkEventType enum to its canonical string representation.
func NetworkEventTypeToString(eventType basbase.NetworkEventType) string {
	switch eventType {
	case basbase.NetworkEventType_NETWORK_EVENT_TYPE_REQUEST:
		return "request"
	case basbase.NetworkEventType_NETWORK_EVENT_TYPE_RESPONSE:
		return "response"
	case basbase.NetworkEventType_NETWORK_EVENT_TYPE_FAILURE:
		return "failure"
	default:
		return "unknown"
	}
}

// =============================================================================
// MOUSE BUTTON CONVERTERS
// =============================================================================

// StringToMouseButton converts a mouse button string to the proto MouseButton enum.
func StringToMouseButton(s string) basactions.MouseButton {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "left":
		return basactions.MouseButton_MOUSE_BUTTON_LEFT
	case "right":
		return basactions.MouseButton_MOUSE_BUTTON_RIGHT
	case "middle":
		return basactions.MouseButton_MOUSE_BUTTON_MIDDLE
	default:
		return basactions.MouseButton_MOUSE_BUTTON_UNSPECIFIED
	}
}

// MouseButtonToString converts a MouseButton enum to its canonical string representation.
func MouseButtonToString(button basactions.MouseButton) string {
	switch button {
	case basactions.MouseButton_MOUSE_BUTTON_LEFT:
		return "left"
	case basactions.MouseButton_MOUSE_BUTTON_RIGHT:
		return "right"
	case basactions.MouseButton_MOUSE_BUTTON_MIDDLE:
		return "middle"
	default:
		return "left" // Default to left for unspecified
	}
}

// =============================================================================
// KEYBOARD MODIFIER CONVERTERS
// =============================================================================

// StringToKeyboardModifier converts a keyboard modifier string to the proto KeyboardModifier enum.
func StringToKeyboardModifier(s string) basactions.KeyboardModifier {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "ctrl", "control":
		return basactions.KeyboardModifier_KEYBOARD_MODIFIER_CTRL
	case "shift":
		return basactions.KeyboardModifier_KEYBOARD_MODIFIER_SHIFT
	case "alt":
		return basactions.KeyboardModifier_KEYBOARD_MODIFIER_ALT
	case "meta", "cmd", "command", "win", "windows":
		return basactions.KeyboardModifier_KEYBOARD_MODIFIER_META
	default:
		return basactions.KeyboardModifier_KEYBOARD_MODIFIER_UNSPECIFIED
	}
}

// KeyboardModifierToString converts a KeyboardModifier enum to its canonical string representation.
func KeyboardModifierToString(modifier basactions.KeyboardModifier) string {
	switch modifier {
	case basactions.KeyboardModifier_KEYBOARD_MODIFIER_CTRL:
		return "ctrl"
	case basactions.KeyboardModifier_KEYBOARD_MODIFIER_SHIFT:
		return "shift"
	case basactions.KeyboardModifier_KEYBOARD_MODIFIER_ALT:
		return "alt"
	case basactions.KeyboardModifier_KEYBOARD_MODIFIER_META:
		return "meta"
	default:
		return "unknown"
	}
}

// StringsToKeyboardModifiers converts a slice of modifier strings to KeyboardModifier enums.
func StringsToKeyboardModifiers(modifiers []string) []basactions.KeyboardModifier {
	if len(modifiers) == 0 {
		return nil
	}
	result := make([]basactions.KeyboardModifier, 0, len(modifiers))
	for _, s := range modifiers {
		mod := StringToKeyboardModifier(s)
		if mod != basactions.KeyboardModifier_KEYBOARD_MODIFIER_UNSPECIFIED {
			result = append(result, mod)
		}
	}
	return result
}

// KeyboardModifiersToStrings converts a slice of KeyboardModifier enums to strings.
func KeyboardModifiersToStrings(modifiers []basactions.KeyboardModifier) []string {
	if len(modifiers) == 0 {
		return nil
	}
	result := make([]string, 0, len(modifiers))
	for _, mod := range modifiers {
		if s := KeyboardModifierToString(mod); s != "unknown" {
			result = append(result, s)
		}
	}
	return result
}

// =============================================================================
// TRIGGER TYPE CONVERTERS
// =============================================================================

// StringToTriggerType converts a trigger type string to the proto TriggerType enum.
func StringToTriggerType(s string) basbase.TriggerType {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "MANUAL", "TRIGGER_TYPE_MANUAL":
		return basbase.TriggerType_TRIGGER_TYPE_MANUAL
	case "SCHEDULED", "TRIGGER_TYPE_SCHEDULED":
		return basbase.TriggerType_TRIGGER_TYPE_SCHEDULED
	case "API", "TRIGGER_TYPE_API":
		return basbase.TriggerType_TRIGGER_TYPE_API
	case "WEBHOOK", "TRIGGER_TYPE_WEBHOOK":
		return basbase.TriggerType_TRIGGER_TYPE_WEBHOOK
	default:
		return basbase.TriggerType_TRIGGER_TYPE_UNSPECIFIED
	}
}

// TriggerTypeToString converts a TriggerType enum to its canonical string representation.
func TriggerTypeToString(triggerType basbase.TriggerType) string {
	switch triggerType {
	case basbase.TriggerType_TRIGGER_TYPE_MANUAL:
		return "manual"
	case basbase.TriggerType_TRIGGER_TYPE_SCHEDULED:
		return "scheduled"
	case basbase.TriggerType_TRIGGER_TYPE_API:
		return "api"
	case basbase.TriggerType_TRIGGER_TYPE_WEBHOOK:
		return "webhook"
	default:
		return "unknown"
	}
}

// =============================================================================
// EXPORT STATUS CONVERTERS
// =============================================================================

// StringToExportStatus converts an export status string to the proto ExportStatus enum.
func StringToExportStatus(s string) basbase.ExportStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "READY", "EXPORT_STATUS_READY":
		return basbase.ExportStatus_EXPORT_STATUS_READY
	case "PENDING", "NOT_READY", "EXPORT_STATUS_PENDING":
		return basbase.ExportStatus_EXPORT_STATUS_PENDING
	case "ERROR", "EXPORT_STATUS_ERROR":
		return basbase.ExportStatus_EXPORT_STATUS_ERROR
	case "UNAVAILABLE", "EXPORT_STATUS_UNAVAILABLE":
		return basbase.ExportStatus_EXPORT_STATUS_UNAVAILABLE
	default:
		return basbase.ExportStatus_EXPORT_STATUS_UNSPECIFIED
	}
}

// ExportStatusToString converts an ExportStatus enum to its canonical string representation.
func ExportStatusToString(status basbase.ExportStatus) string {
	switch status {
	case basbase.ExportStatus_EXPORT_STATUS_READY:
		return "ready"
	case basbase.ExportStatus_EXPORT_STATUS_PENDING:
		return "pending"
	case basbase.ExportStatus_EXPORT_STATUS_ERROR:
		return "error"
	case basbase.ExportStatus_EXPORT_STATUS_UNAVAILABLE:
		return "unavailable"
	default:
		return "unknown"
	}
}

// =============================================================================
// STEP STATUS CONVERTERS
// =============================================================================

// StringToStepStatus converts a step status string to the proto StepStatus enum.
func StringToStepStatus(s string) basbase.StepStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "PENDING", "STEP_STATUS_PENDING":
		return basbase.StepStatus_STEP_STATUS_PENDING
	case "RUNNING", "STEP_STATUS_RUNNING":
		return basbase.StepStatus_STEP_STATUS_RUNNING
	case "COMPLETED", "STEP_STATUS_COMPLETED":
		return basbase.StepStatus_STEP_STATUS_COMPLETED
	case "FAILED", "STEP_STATUS_FAILED":
		return basbase.StepStatus_STEP_STATUS_FAILED
	case "CANCELLED", "STEP_STATUS_CANCELLED":
		return basbase.StepStatus_STEP_STATUS_CANCELLED
	case "SKIPPED", "STEP_STATUS_SKIPPED":
		return basbase.StepStatus_STEP_STATUS_SKIPPED
	case "RETRYING", "STEP_STATUS_RETRYING":
		return basbase.StepStatus_STEP_STATUS_RETRYING
	default:
		return basbase.StepStatus_STEP_STATUS_UNSPECIFIED
	}
}

// StepStatusToString converts a StepStatus enum to its canonical string representation.
func StepStatusToString(status basbase.StepStatus) string {
	switch status {
	case basbase.StepStatus_STEP_STATUS_PENDING:
		return "pending"
	case basbase.StepStatus_STEP_STATUS_RUNNING:
		return "running"
	case basbase.StepStatus_STEP_STATUS_COMPLETED:
		return "completed"
	case basbase.StepStatus_STEP_STATUS_FAILED:
		return "failed"
	case basbase.StepStatus_STEP_STATUS_CANCELLED:
		return "cancelled"
	case basbase.StepStatus_STEP_STATUS_SKIPPED:
		return "skipped"
	case basbase.StepStatus_STEP_STATUS_RETRYING:
		return "retrying"
	default:
		return "unknown"
	}
}

// =============================================================================
// ARTIFACT TYPE CONVERTERS
// =============================================================================

// StringToArtifactType converts an artifact type string to the proto ArtifactType enum.
func StringToArtifactType(s string) basbase.ArtifactType {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "screenshot", "artifact_type_screenshot":
		return basbase.ArtifactType_ARTIFACT_TYPE_SCREENSHOT
	case "dom", "dom_snapshot", "artifact_type_dom_snapshot":
		return basbase.ArtifactType_ARTIFACT_TYPE_DOM_SNAPSHOT
	case "timeline_frame", "artifact_type_timeline_frame":
		return basbase.ArtifactType_ARTIFACT_TYPE_TIMELINE_FRAME
	case "console_log", "artifact_type_console_log":
		return basbase.ArtifactType_ARTIFACT_TYPE_CONSOLE_LOG
	case "network_event", "artifact_type_network_event":
		return basbase.ArtifactType_ARTIFACT_TYPE_NETWORK_EVENT
	case "trace", "artifact_type_trace":
		return basbase.ArtifactType_ARTIFACT_TYPE_TRACE
	case "custom", "artifact_type_custom", "extracted_data", "video", "har":
		return basbase.ArtifactType_ARTIFACT_TYPE_CUSTOM
	default:
		return basbase.ArtifactType_ARTIFACT_TYPE_UNSPECIFIED
	}
}

// ArtifactTypeToString converts an ArtifactType enum to its canonical string representation.
func ArtifactTypeToString(artifactType basbase.ArtifactType) string {
	switch artifactType {
	case basbase.ArtifactType_ARTIFACT_TYPE_SCREENSHOT:
		return "screenshot"
	case basbase.ArtifactType_ARTIFACT_TYPE_DOM_SNAPSHOT:
		return "dom_snapshot"
	case basbase.ArtifactType_ARTIFACT_TYPE_TIMELINE_FRAME:
		return "timeline_frame"
	case basbase.ArtifactType_ARTIFACT_TYPE_CONSOLE_LOG:
		return "console_log"
	case basbase.ArtifactType_ARTIFACT_TYPE_NETWORK_EVENT:
		return "network_event"
	case basbase.ArtifactType_ARTIFACT_TYPE_TRACE:
		return "trace"
	case basbase.ArtifactType_ARTIFACT_TYPE_CUSTOM:
		return "custom"
	default:
		return "unknown"
	}
}

// =============================================================================
// HIGHLIGHT COLOR CONVERTERS
// =============================================================================

// StringToHighlightColor converts a color string to the proto HighlightColor enum.
func StringToHighlightColor(s string) basbase.HighlightColor {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "red":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_RED
	case "green":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_GREEN
	case "blue":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_BLUE
	case "yellow":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_YELLOW
	case "orange":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_ORANGE
	case "purple":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_PURPLE
	case "cyan":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_CYAN
	case "magenta", "pink":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_PINK
	case "white":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_WHITE
	case "gray", "grey":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_GRAY
	case "black":
		return basbase.HighlightColor_HIGHLIGHT_COLOR_BLACK
	default:
		return basbase.HighlightColor_HIGHLIGHT_COLOR_UNSPECIFIED
	}
}

// HighlightColorToString converts a HighlightColor enum to its canonical string representation.
func HighlightColorToString(color basbase.HighlightColor) string {
	switch color {
	case basbase.HighlightColor_HIGHLIGHT_COLOR_RED:
		return "red"
	case basbase.HighlightColor_HIGHLIGHT_COLOR_GREEN:
		return "green"
	case basbase.HighlightColor_HIGHLIGHT_COLOR_BLUE:
		return "blue"
	case basbase.HighlightColor_HIGHLIGHT_COLOR_YELLOW:
		return "yellow"
	case basbase.HighlightColor_HIGHLIGHT_COLOR_ORANGE:
		return "orange"
	case basbase.HighlightColor_HIGHLIGHT_COLOR_PURPLE:
		return "purple"
	case basbase.HighlightColor_HIGHLIGHT_COLOR_CYAN:
		return "cyan"
	case basbase.HighlightColor_HIGHLIGHT_COLOR_PINK:
		return "pink"
	case basbase.HighlightColor_HIGHLIGHT_COLOR_WHITE:
		return "white"
	case basbase.HighlightColor_HIGHLIGHT_COLOR_GRAY:
		return "gray"
	case basbase.HighlightColor_HIGHLIGHT_COLOR_BLACK:
		return "black"
	default:
		return "unknown"
	}
}

// =============================================================================
// CHANGE SOURCE CONVERTERS
// =============================================================================

// StringToChangeSource converts a change source string to the proto ChangeSource enum.
func StringToChangeSource(s string) basbase.ChangeSource {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "manual":
		return basbase.ChangeSource_CHANGE_SOURCE_MANUAL
	case "autosave":
		return basbase.ChangeSource_CHANGE_SOURCE_AUTOSAVE
	case "import":
		return basbase.ChangeSource_CHANGE_SOURCE_IMPORT
	case "ai_generated", "ai-generated", "aigenerated":
		return basbase.ChangeSource_CHANGE_SOURCE_AI_GENERATED
	case "recording":
		return basbase.ChangeSource_CHANGE_SOURCE_RECORDING
	default:
		return basbase.ChangeSource_CHANGE_SOURCE_UNSPECIFIED
	}
}

// ChangeSourceToString converts a ChangeSource enum to its canonical string representation.
func ChangeSourceToString(source basbase.ChangeSource) string {
	switch source {
	case basbase.ChangeSource_CHANGE_SOURCE_MANUAL:
		return "manual"
	case basbase.ChangeSource_CHANGE_SOURCE_AUTOSAVE:
		return "autosave"
	case basbase.ChangeSource_CHANGE_SOURCE_IMPORT:
		return "import"
	case basbase.ChangeSource_CHANGE_SOURCE_AI_GENERATED:
		return "ai_generated"
	case basbase.ChangeSource_CHANGE_SOURCE_RECORDING:
		return "recording"
	default:
		return "unknown"
	}
}

// =============================================================================
// VALIDATION SEVERITY CONVERTERS
// =============================================================================

// StringToValidationSeverity converts a validation severity string to the proto ValidationSeverity enum.
func StringToValidationSeverity(s string) basbase.ValidationSeverity {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "error":
		return basbase.ValidationSeverity_VALIDATION_SEVERITY_ERROR
	case "warning", "warn":
		return basbase.ValidationSeverity_VALIDATION_SEVERITY_WARNING
	case "info":
		return basbase.ValidationSeverity_VALIDATION_SEVERITY_INFO
	default:
		return basbase.ValidationSeverity_VALIDATION_SEVERITY_UNSPECIFIED
	}
}

// ValidationSeverityToString converts a ValidationSeverity enum to its canonical string representation.
func ValidationSeverityToString(severity basbase.ValidationSeverity) string {
	switch severity {
	case basbase.ValidationSeverity_VALIDATION_SEVERITY_ERROR:
		return "error"
	case basbase.ValidationSeverity_VALIDATION_SEVERITY_WARNING:
		return "warning"
	case basbase.ValidationSeverity_VALIDATION_SEVERITY_INFO:
		return "info"
	default:
		return "unknown"
	}
}
