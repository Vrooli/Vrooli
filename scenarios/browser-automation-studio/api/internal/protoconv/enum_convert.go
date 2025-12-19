// Package protoconv provides proto enum conversion utilities.
//
// DEPRECATION NOTE: The core enum conversion functions have been moved to internal/enums
// to avoid import cycles. This file now re-exports those functions for backward compatibility.
// New code should import internal/enums directly.
package protoconv

import (
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

// =============================================================================
// RE-EXPORTS FROM internal/enums
// All functions below delegate to internal/enums for the actual implementation.
// =============================================================================

// StringToActionType converts an action type string to the proto ActionType enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToActionType(actionType string) basactions.ActionType {
	return enums.StringToActionType(actionType)
}

// ActionTypeToString converts an ActionType enum to a proto-aligned string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func ActionTypeToString(actionType basactions.ActionType) string {
	return enums.ActionTypeToString(actionType)
}

// StringToSelectorType converts a selector type string to the proto SelectorType enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToSelectorType(selectorType string) basbase.SelectorType {
	return enums.StringToSelectorType(selectorType)
}

// SelectorTypeToString converts a SelectorType enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func SelectorTypeToString(selectorType basbase.SelectorType) string {
	return enums.SelectorTypeToString(selectorType)
}

// StringToExecutionStatus converts an execution status string to the proto ExecutionStatus enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToExecutionStatus(s string) basbase.ExecutionStatus {
	return enums.StringToExecutionStatus(s)
}

// ExecutionStatusToString converts an ExecutionStatus enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func ExecutionStatusToString(status basbase.ExecutionStatus) string {
	return enums.ExecutionStatusToString(status)
}

// StringToTriggerType converts a trigger type string to the proto TriggerType enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToTriggerType(s string) basbase.TriggerType {
	return enums.StringToTriggerType(s)
}

// TriggerTypeToString converts a TriggerType enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func TriggerTypeToString(triggerType basbase.TriggerType) string {
	return enums.TriggerTypeToString(triggerType)
}

// StringToExportStatus converts an export status string to the proto ExportStatus enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToExportStatus(s string) basbase.ExportStatus {
	return enums.StringToExportStatus(s)
}

// ExportStatusToString converts an ExportStatus enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func ExportStatusToString(status basbase.ExportStatus) string {
	return enums.ExportStatusToString(status)
}

// StringToStepStatus converts a step status string to the proto StepStatus enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToStepStatus(s string) basbase.StepStatus {
	return enums.StringToStepStatus(s)
}

// StepStatusToString converts a StepStatus enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StepStatusToString(status basbase.StepStatus) string {
	return enums.StepStatusToString(status)
}

// StringToLogLevel converts a log level string to the proto LogLevel enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToLogLevel(s string) basbase.LogLevel {
	return enums.StringToLogLevel(s)
}

// LogLevelToString converts a LogLevel enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func LogLevelToString(level basbase.LogLevel) string {
	return enums.LogLevelToString(level)
}

// StringToAssertionMode converts an assertion mode string to the proto AssertionMode enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToAssertionMode(s string) basbase.AssertionMode {
	return enums.StringToAssertionMode(s)
}

// AssertionModeToString converts an AssertionMode enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func AssertionModeToString(mode basbase.AssertionMode) string {
	return enums.AssertionModeToString(mode)
}

// StringToArtifactType converts an artifact type string to the proto ArtifactType enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToArtifactType(s string) basbase.ArtifactType {
	return enums.StringToArtifactType(s)
}

// ArtifactTypeToString converts an ArtifactType enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func ArtifactTypeToString(artifactType basbase.ArtifactType) string {
	return enums.ArtifactTypeToString(artifactType)
}

// StringToHighlightColor converts a color string to the proto HighlightColor enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToHighlightColor(s string) basbase.HighlightColor {
	return enums.StringToHighlightColor(s)
}

// HighlightColorToString converts a HighlightColor enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func HighlightColorToString(color basbase.HighlightColor) string {
	return enums.HighlightColorToString(color)
}

// StringToMouseButton converts a mouse button string to the proto MouseButton enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToMouseButton(s string) basactions.MouseButton {
	return enums.StringToMouseButton(s)
}

// MouseButtonToString converts a MouseButton enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func MouseButtonToString(button basactions.MouseButton) string {
	return enums.MouseButtonToString(button)
}

// StringToKeyboardModifier converts a keyboard modifier string to the proto KeyboardModifier enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToKeyboardModifier(s string) basactions.KeyboardModifier {
	return enums.StringToKeyboardModifier(s)
}

// KeyboardModifierToString converts a KeyboardModifier enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func KeyboardModifierToString(modifier basactions.KeyboardModifier) string {
	return enums.KeyboardModifierToString(modifier)
}

// StringsToKeyboardModifiers converts a slice of modifier strings to KeyboardModifier enums.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringsToKeyboardModifiers(modifiers []string) []basactions.KeyboardModifier {
	return enums.StringsToKeyboardModifiers(modifiers)
}

// KeyboardModifiersToStrings converts a slice of KeyboardModifier enums to strings.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func KeyboardModifiersToStrings(modifiers []basactions.KeyboardModifier) []string {
	return enums.KeyboardModifiersToStrings(modifiers)
}

// StringToNetworkEventType converts a network event type string to the proto NetworkEventType enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToNetworkEventType(s string) basbase.NetworkEventType {
	return enums.StringToNetworkEventType(s)
}

// NetworkEventTypeToString converts a NetworkEventType enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func NetworkEventTypeToString(eventType basbase.NetworkEventType) string {
	return enums.NetworkEventTypeToString(eventType)
}

// StringToChangeSource converts a change source string to the proto ChangeSource enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToChangeSource(s string) basbase.ChangeSource {
	return enums.StringToChangeSource(s)
}

// ChangeSourceToString converts a ChangeSource enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func ChangeSourceToString(source basbase.ChangeSource) string {
	return enums.ChangeSourceToString(source)
}

// StringToValidationSeverity converts a validation severity string to the proto ValidationSeverity enum.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func StringToValidationSeverity(s string) basbase.ValidationSeverity {
	return enums.StringToValidationSeverity(s)
}

// ValidationSeverityToString converts a ValidationSeverity enum to its canonical string representation.
// Deprecated: Import github.com/vrooli/browser-automation-studio/internal/enums instead.
func ValidationSeverityToString(severity basbase.ValidationSeverity) string {
	return enums.ValidationSeverityToString(severity)
}
