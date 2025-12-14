package typeconv

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// ToString safely converts various types to string.
// Returns empty string if conversion fails.
func ToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	default:
		return ""
	}
}

// ToInt safely converts various numeric types to int.
// Returns 0 if conversion fails.
func ToInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return 0
}

// ToBool safely converts various types to bool.
// Returns false if conversion fails.
func ToBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return false
}

// ToFloat safely converts various numeric types to float64.
// Returns 0 if conversion fails.
func ToFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// ToTimePtr safely converts various types to *time.Time.
// Returns nil if conversion fails or value is empty.
func ToTimePtr(value any) *time.Time {
	switch v := value.(type) {
	case time.Time:
		return &v
	case *time.Time:
		return v
	case string:
		if v == "" {
			return nil
		}
		if ts, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return &ts
		}
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			return &ts
		}
	}
	return nil
}

// ToStringSlice safely converts various types to []string.
// Returns empty slice if conversion fails.
func ToStringSlice(value any) []string {
	result := make([]string, 0)
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		for _, item := range v {
			if str := ToString(item); str != "" {
				result = append(result, str)
			}
		}
	case string:
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

// ToInterfaceSlice safely converts various types to []any.
// Handles []any, []map[string]any, map[string]any (values), and JSON-serializable types.
// Returns empty slice if conversion fails.
func ToInterfaceSlice(value any) []any {
	switch typed := value.(type) {
	case nil:
		return []any{}
	case []any:
		return typed
	case []map[string]any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = typed[i]
		}
		return result
	case map[string]any:
		result := make([]any, 0, len(typed))
		for _, v := range typed {
			result = append(result, v)
		}
		return result
	default:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return []any{}
		}
		var arr []any
		if err := json.Unmarshal(bytes, &arr); err != nil {
			return []any{}
		}
		return arr
	}
}

// DeepCloneMap creates a deep copy of a map[string]any, recursively cloning nested maps and slices.
// Returns nil if input is nil.
func DeepCloneMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	clone := make(map[string]any, len(source))
	for k, v := range source {
		clone[k] = DeepCloneValue(v)
	}
	return clone
}

// DeepCloneValue creates a deep copy of a value, handling maps, slices, and primitives.
// Used by DeepCloneMap for recursive cloning.
func DeepCloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for k, v := range typed {
			cloned[k] = DeepCloneValue(v)
		}
		return cloned
	case []any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = DeepCloneValue(typed[i])
		}
		return result
	case []string:
		return append([]string{}, typed...)
	default:
		// Primitives and other types are returned as-is (they're copied by value)
		return typed
	}
}

// AnyToJsonValue converts any Go value to a commonv1.JsonValue proto message.
// Handles primitives (bool, int, float, string, bytes), maps, and slices recursively.
// Falls back to string representation for unsupported types.
func AnyToJsonValue(v any) *commonv1.JsonValue {
	if v == nil {
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{}}
	}
	switch val := v.(type) {
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: val}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: val}}
	case uint:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case uint32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case uint64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case float32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: float64(val)}}
	case float64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: val}}
	case []byte:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BytesValue{BytesValue: val}}
	case map[string]any:
		obj := make(map[string]*commonv1.JsonValue, len(val))
		for k, v := range val {
			if nested := AnyToJsonValue(v); nested != nil {
				obj[k] = nested
			}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{
			ObjectValue: &commonv1.JsonObject{Fields: obj},
		}}
	case []any:
		items := make([]*commonv1.JsonValue, 0, len(val))
		for _, item := range val {
			if nested := AnyToJsonValue(item); nested != nil {
				items = append(items, nested)
			}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{
			ListValue: &commonv1.JsonList{Values: items},
		}}
	default:
		// Fallback: try to convert to string
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: fmt.Sprintf("%v", val)}}
	}
}

// JsonValueToAny converts a commonv1.JsonValue proto message back to a Go value.
// Handles all JsonValue kinds: bool, int, double, string, null, object, and list.
// Returns nil for nil input or unrecognized kinds.
func JsonValueToAny(v *commonv1.JsonValue) any {
	if v == nil {
		return nil
	}
	switch k := v.Kind.(type) {
	case *commonv1.JsonValue_BoolValue:
		return k.BoolValue
	case *commonv1.JsonValue_IntValue:
		return k.IntValue
	case *commonv1.JsonValue_DoubleValue:
		return k.DoubleValue
	case *commonv1.JsonValue_StringValue:
		return k.StringValue
	case *commonv1.JsonValue_NullValue:
		return nil
	case *commonv1.JsonValue_BytesValue:
		return k.BytesValue
	case *commonv1.JsonValue_ObjectValue:
		if k.ObjectValue == nil {
			return nil
		}
		result := make(map[string]any, len(k.ObjectValue.Fields))
		for key, val := range k.ObjectValue.Fields {
			result[key] = JsonValueToAny(val)
		}
		return result
	case *commonv1.JsonValue_ListValue:
		if k.ListValue == nil {
			return nil
		}
		result := make([]any, 0, len(k.ListValue.Values))
		for _, val := range k.ListValue.Values {
			result = append(result, JsonValueToAny(val))
		}
		return result
	default:
		return nil
	}
}

// ToInt32 safely converts various numeric types to int32.
// Returns value and ok=true on success, 0 and ok=false otherwise.
func ToInt32(v any) (int32, bool) {
	switch val := v.(type) {
	case int:
		return int32(val), true
	case int32:
		return val, true
	case int64:
		return int32(val), true
	case float64:
		return int32(val), true
	case float32:
		return int32(val), true
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return int32(i), true
		}
	}
	return 0, false
}

// ToFloat64 safely converts various numeric types to float64.
// Returns value and ok=true on success, 0 and ok=false otherwise.
func ToFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case json.Number:
		if f, err := val.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

// StringToActionType converts an action type string to the proto ActionType enum.
// Handles various aliases for each action type (e.g., "goto" -> NAVIGATE, "fill"/"type" -> INPUT).
// Returns ACTION_TYPE_UNSPECIFIED for unrecognized types.
func StringToActionType(actionType string) basv1.ActionType {
	switch strings.ToLower(strings.TrimSpace(actionType)) {
	case "navigate", "goto":
		return basv1.ActionType_ACTION_TYPE_NAVIGATE
	case "click":
		return basv1.ActionType_ACTION_TYPE_CLICK
	case "input", "type", "fill":
		return basv1.ActionType_ACTION_TYPE_INPUT
	case "wait":
		return basv1.ActionType_ACTION_TYPE_WAIT
	case "assert":
		return basv1.ActionType_ACTION_TYPE_ASSERT
	case "scroll":
		return basv1.ActionType_ACTION_TYPE_SCROLL
	case "select", "selectoption":
		return basv1.ActionType_ACTION_TYPE_SELECT
	case "evaluate", "eval":
		return basv1.ActionType_ACTION_TYPE_EVALUATE
	case "keyboard", "keypress", "press":
		return basv1.ActionType_ACTION_TYPE_KEYBOARD
	case "hover":
		return basv1.ActionType_ACTION_TYPE_HOVER
	case "screenshot":
		return basv1.ActionType_ACTION_TYPE_SCREENSHOT
	case "focus":
		return basv1.ActionType_ACTION_TYPE_FOCUS
	case "blur":
		return basv1.ActionType_ACTION_TYPE_BLUR
	case "subflow":
		return basv1.ActionType_ACTION_TYPE_SUBFLOW
	default:
		return basv1.ActionType_ACTION_TYPE_UNSPECIFIED
	}
}

// ActionTypeToString converts an ActionType enum to its canonical string representation.
// Returns "unknown" for unrecognized types.
func ActionTypeToString(actionType basv1.ActionType) string {
	switch actionType {
	case basv1.ActionType_ACTION_TYPE_NAVIGATE:
		return "navigate"
	case basv1.ActionType_ACTION_TYPE_CLICK:
		return "click"
	case basv1.ActionType_ACTION_TYPE_INPUT:
		return "input"
	case basv1.ActionType_ACTION_TYPE_WAIT:
		return "wait"
	case basv1.ActionType_ACTION_TYPE_ASSERT:
		return "assert"
	case basv1.ActionType_ACTION_TYPE_SCROLL:
		return "scroll"
	case basv1.ActionType_ACTION_TYPE_SELECT:
		return "select"
	case basv1.ActionType_ACTION_TYPE_EVALUATE:
		return "evaluate"
	case basv1.ActionType_ACTION_TYPE_KEYBOARD:
		return "keyboard"
	case basv1.ActionType_ACTION_TYPE_HOVER:
		return "hover"
	case basv1.ActionType_ACTION_TYPE_SCREENSHOT:
		return "screenshot"
	case basv1.ActionType_ACTION_TYPE_FOCUS:
		return "focus"
	case basv1.ActionType_ACTION_TYPE_BLUR:
		return "blur"
	case basv1.ActionType_ACTION_TYPE_SUBFLOW:
		return "subflow"
	default:
		return "unknown"
	}
}

// StringToSelectorType converts a selector type string to the proto SelectorType enum.
// Handles various aliases for each selector type (e.g., "data-testid" -> DATA_TESTID).
// Returns SELECTOR_TYPE_UNSPECIFIED for unrecognized types.
func StringToSelectorType(selectorType string) basv1.SelectorType {
	switch strings.ToLower(strings.TrimSpace(selectorType)) {
	case "css":
		return basv1.SelectorType_SELECTOR_TYPE_CSS
	case "xpath":
		return basv1.SelectorType_SELECTOR_TYPE_XPATH
	case "id":
		return basv1.SelectorType_SELECTOR_TYPE_ID
	case "data-testid", "datatestid", "testid":
		return basv1.SelectorType_SELECTOR_TYPE_DATA_TESTID
	case "aria", "aria-label":
		return basv1.SelectorType_SELECTOR_TYPE_ARIA
	case "text":
		return basv1.SelectorType_SELECTOR_TYPE_TEXT
	case "role":
		return basv1.SelectorType_SELECTOR_TYPE_ROLE
	case "placeholder":
		return basv1.SelectorType_SELECTOR_TYPE_PLACEHOLDER
	case "alt", "alt-text", "alttext":
		return basv1.SelectorType_SELECTOR_TYPE_ALT_TEXT
	case "title":
		return basv1.SelectorType_SELECTOR_TYPE_TITLE
	default:
		return basv1.SelectorType_SELECTOR_TYPE_UNSPECIFIED
	}
}

// SelectorTypeToString converts a SelectorType enum to its canonical string representation.
// Returns "unknown" for unrecognized types.
func SelectorTypeToString(selectorType basv1.SelectorType) string {
	switch selectorType {
	case basv1.SelectorType_SELECTOR_TYPE_CSS:
		return "css"
	case basv1.SelectorType_SELECTOR_TYPE_XPATH:
		return "xpath"
	case basv1.SelectorType_SELECTOR_TYPE_ID:
		return "id"
	case basv1.SelectorType_SELECTOR_TYPE_DATA_TESTID:
		return "data-testid"
	case basv1.SelectorType_SELECTOR_TYPE_ARIA:
		return "aria"
	case basv1.SelectorType_SELECTOR_TYPE_TEXT:
		return "text"
	case basv1.SelectorType_SELECTOR_TYPE_ROLE:
		return "role"
	case basv1.SelectorType_SELECTOR_TYPE_PLACEHOLDER:
		return "placeholder"
	case basv1.SelectorType_SELECTOR_TYPE_ALT_TEXT:
		return "alt-text"
	case basv1.SelectorType_SELECTOR_TYPE_TITLE:
		return "title"
	default:
		return "unknown"
	}
}

// =============================================================================
// EXECUTION STATUS CONVERTERS
// =============================================================================

// StringToExecutionStatus converts an execution status string to the proto ExecutionStatus enum.
// Handles various aliases (e.g., "PENDING", "EXECUTION_STATUS_PENDING", "pending").
// Returns EXECUTION_STATUS_UNSPECIFIED for unrecognized values.
func StringToExecutionStatus(s string) basv1.ExecutionStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "PENDING", "EXECUTION_STATUS_PENDING":
		return basv1.ExecutionStatus_EXECUTION_STATUS_PENDING
	case "RUNNING", "EXECUTION_STATUS_RUNNING":
		return basv1.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case "COMPLETED", "EXECUTION_STATUS_COMPLETED":
		return basv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case "FAILED", "EXECUTION_STATUS_FAILED":
		return basv1.ExecutionStatus_EXECUTION_STATUS_FAILED
	case "CANCELLED", "CANCELED", "EXECUTION_STATUS_CANCELLED":
		return basv1.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	default:
		return basv1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	}
}

// ExecutionStatusToString converts an ExecutionStatus enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func ExecutionStatusToString(status basv1.ExecutionStatus) string {
	switch status {
	case basv1.ExecutionStatus_EXECUTION_STATUS_PENDING:
		return "pending"
	case basv1.ExecutionStatus_EXECUTION_STATUS_RUNNING:
		return "running"
	case basv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED:
		return "completed"
	case basv1.ExecutionStatus_EXECUTION_STATUS_FAILED:
		return "failed"
	case basv1.ExecutionStatus_EXECUTION_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unknown"
	}
}

// =============================================================================
// TRIGGER TYPE CONVERTERS
// =============================================================================

// StringToTriggerType converts a trigger type string to the proto TriggerType enum.
// Handles various aliases (e.g., "MANUAL", "TRIGGER_TYPE_MANUAL", "manual").
// Returns TRIGGER_TYPE_UNSPECIFIED for unrecognized values.
func StringToTriggerType(s string) basv1.TriggerType {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "MANUAL", "TRIGGER_TYPE_MANUAL":
		return basv1.TriggerType_TRIGGER_TYPE_MANUAL
	case "SCHEDULED", "TRIGGER_TYPE_SCHEDULED":
		return basv1.TriggerType_TRIGGER_TYPE_SCHEDULED
	case "API", "TRIGGER_TYPE_API":
		return basv1.TriggerType_TRIGGER_TYPE_API
	case "WEBHOOK", "TRIGGER_TYPE_WEBHOOK":
		return basv1.TriggerType_TRIGGER_TYPE_WEBHOOK
	default:
		return basv1.TriggerType_TRIGGER_TYPE_UNSPECIFIED
	}
}

// TriggerTypeToString converts a TriggerType enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func TriggerTypeToString(triggerType basv1.TriggerType) string {
	switch triggerType {
	case basv1.TriggerType_TRIGGER_TYPE_MANUAL:
		return "manual"
	case basv1.TriggerType_TRIGGER_TYPE_SCHEDULED:
		return "scheduled"
	case basv1.TriggerType_TRIGGER_TYPE_API:
		return "api"
	case basv1.TriggerType_TRIGGER_TYPE_WEBHOOK:
		return "webhook"
	default:
		return "unknown"
	}
}

// =============================================================================
// EXPORT STATUS CONVERTERS
// =============================================================================

// StringToExportStatus converts an export status string to the proto ExportStatus enum.
// Handles various aliases (e.g., "READY", "EXPORT_STATUS_READY", "NOT_READY").
// Returns EXPORT_STATUS_UNSPECIFIED for unrecognized values.
func StringToExportStatus(s string) basv1.ExportStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "READY", "EXPORT_STATUS_READY":
		return basv1.ExportStatus_EXPORT_STATUS_READY
	case "PENDING", "NOT_READY", "EXPORT_STATUS_PENDING":
		return basv1.ExportStatus_EXPORT_STATUS_PENDING
	case "ERROR", "EXPORT_STATUS_ERROR":
		return basv1.ExportStatus_EXPORT_STATUS_ERROR
	case "UNAVAILABLE", "EXPORT_STATUS_UNAVAILABLE":
		return basv1.ExportStatus_EXPORT_STATUS_UNAVAILABLE
	default:
		return basv1.ExportStatus_EXPORT_STATUS_UNSPECIFIED
	}
}

// ExportStatusToString converts an ExportStatus enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func ExportStatusToString(status basv1.ExportStatus) string {
	switch status {
	case basv1.ExportStatus_EXPORT_STATUS_READY:
		return "ready"
	case basv1.ExportStatus_EXPORT_STATUS_PENDING:
		return "pending"
	case basv1.ExportStatus_EXPORT_STATUS_ERROR:
		return "error"
	case basv1.ExportStatus_EXPORT_STATUS_UNAVAILABLE:
		return "unavailable"
	default:
		return "unknown"
	}
}

// =============================================================================
// STEP STATUS CONVERTERS
// =============================================================================

// StringToStepStatus converts a step status string to the proto StepStatus enum.
// Handles various aliases (e.g., "PENDING", "STEP_STATUS_PENDING", "pending").
// Returns STEP_STATUS_UNSPECIFIED for unrecognized values.
func StringToStepStatus(s string) basv1.StepStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "PENDING", "STEP_STATUS_PENDING":
		return basv1.StepStatus_STEP_STATUS_PENDING
	case "RUNNING", "STEP_STATUS_RUNNING":
		return basv1.StepStatus_STEP_STATUS_RUNNING
	case "COMPLETED", "STEP_STATUS_COMPLETED":
		return basv1.StepStatus_STEP_STATUS_COMPLETED
	case "FAILED", "STEP_STATUS_FAILED":
		return basv1.StepStatus_STEP_STATUS_FAILED
	case "CANCELLED", "STEP_STATUS_CANCELLED":
		return basv1.StepStatus_STEP_STATUS_CANCELLED
	case "SKIPPED", "STEP_STATUS_SKIPPED":
		return basv1.StepStatus_STEP_STATUS_SKIPPED
	case "RETRYING", "STEP_STATUS_RETRYING":
		return basv1.StepStatus_STEP_STATUS_RETRYING
	default:
		return basv1.StepStatus_STEP_STATUS_UNSPECIFIED
	}
}

// StepStatusToString converts a StepStatus enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func StepStatusToString(status basv1.StepStatus) string {
	switch status {
	case basv1.StepStatus_STEP_STATUS_PENDING:
		return "pending"
	case basv1.StepStatus_STEP_STATUS_RUNNING:
		return "running"
	case basv1.StepStatus_STEP_STATUS_COMPLETED:
		return "completed"
	case basv1.StepStatus_STEP_STATUS_FAILED:
		return "failed"
	case basv1.StepStatus_STEP_STATUS_CANCELLED:
		return "cancelled"
	case basv1.StepStatus_STEP_STATUS_SKIPPED:
		return "skipped"
	case basv1.StepStatus_STEP_STATUS_RETRYING:
		return "retrying"
	default:
		return "unknown"
	}
}

// =============================================================================
// LOG LEVEL CONVERTERS
// =============================================================================

// StringToLogLevel converts a log level string to the proto LogLevel enum.
// Handles various aliases (e.g., "DEBUG", "LOG_LEVEL_DEBUG", "debug", "WARNING" -> WARN).
// Returns LOG_LEVEL_UNSPECIFIED for unrecognized values.
func StringToLogLevel(s string) basv1.LogLevel {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "DEBUG", "LOG_LEVEL_DEBUG":
		return basv1.LogLevel_LOG_LEVEL_DEBUG
	case "INFO", "LOG_LEVEL_INFO":
		return basv1.LogLevel_LOG_LEVEL_INFO
	case "WARN", "WARNING", "LOG_LEVEL_WARN":
		return basv1.LogLevel_LOG_LEVEL_WARN
	case "ERROR", "LOG_LEVEL_ERROR":
		return basv1.LogLevel_LOG_LEVEL_ERROR
	default:
		return basv1.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}

// LogLevelToString converts a LogLevel enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func LogLevelToString(level basv1.LogLevel) string {
	switch level {
	case basv1.LogLevel_LOG_LEVEL_DEBUG:
		return "debug"
	case basv1.LogLevel_LOG_LEVEL_INFO:
		return "info"
	case basv1.LogLevel_LOG_LEVEL_WARN:
		return "warn"
	case basv1.LogLevel_LOG_LEVEL_ERROR:
		return "error"
	default:
		return "unknown"
	}
}

// =============================================================================
// ASSERTION MODE CONVERTERS
// =============================================================================

// StringToAssertionMode converts an assertion mode string to the proto AssertionMode enum.
// Uses snake_case inputs (e.g., "exists", "not_exists", "text_equals").
// Returns ASSERTION_MODE_UNSPECIFIED for unrecognized values.
func StringToAssertionMode(s string) basv1.AssertionMode {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "exists":
		return basv1.AssertionMode_ASSERTION_MODE_EXISTS
	case "not_exists":
		return basv1.AssertionMode_ASSERTION_MODE_NOT_EXISTS
	case "visible":
		return basv1.AssertionMode_ASSERTION_MODE_VISIBLE
	case "hidden":
		return basv1.AssertionMode_ASSERTION_MODE_HIDDEN
	case "text_equals":
		return basv1.AssertionMode_ASSERTION_MODE_TEXT_EQUALS
	case "text_contains":
		return basv1.AssertionMode_ASSERTION_MODE_TEXT_CONTAINS
	case "attribute_equals":
		return basv1.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS
	case "attribute_contains":
		return basv1.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS
	default:
		return basv1.AssertionMode_ASSERTION_MODE_UNSPECIFIED
	}
}

// AssertionModeToString converts an AssertionMode enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func AssertionModeToString(mode basv1.AssertionMode) string {
	switch mode {
	case basv1.AssertionMode_ASSERTION_MODE_EXISTS:
		return "exists"
	case basv1.AssertionMode_ASSERTION_MODE_NOT_EXISTS:
		return "not_exists"
	case basv1.AssertionMode_ASSERTION_MODE_VISIBLE:
		return "visible"
	case basv1.AssertionMode_ASSERTION_MODE_HIDDEN:
		return "hidden"
	case basv1.AssertionMode_ASSERTION_MODE_TEXT_EQUALS:
		return "text_equals"
	case basv1.AssertionMode_ASSERTION_MODE_TEXT_CONTAINS:
		return "text_contains"
	case basv1.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS:
		return "attribute_equals"
	case basv1.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS:
		return "attribute_contains"
	default:
		return "unknown"
	}
}

// =============================================================================
// ARTIFACT TYPE CONVERTERS
// =============================================================================

// StringToArtifactType converts an artifact type string to the proto ArtifactType enum.
// Handles various aliases (e.g., "screenshot", "dom_snapshot", "extracted_data" -> CUSTOM).
// Returns ARTIFACT_TYPE_UNSPECIFIED for unrecognized values.
func StringToArtifactType(s string) basv1.ArtifactType {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "screenshot", "artifact_type_screenshot":
		return basv1.ArtifactType_ARTIFACT_TYPE_SCREENSHOT
	case "dom", "dom_snapshot", "artifact_type_dom_snapshot":
		return basv1.ArtifactType_ARTIFACT_TYPE_DOM_SNAPSHOT
	case "timeline_frame", "artifact_type_timeline_frame":
		return basv1.ArtifactType_ARTIFACT_TYPE_TIMELINE_FRAME
	case "console_log", "artifact_type_console_log":
		return basv1.ArtifactType_ARTIFACT_TYPE_CONSOLE_LOG
	case "network_event", "artifact_type_network_event":
		return basv1.ArtifactType_ARTIFACT_TYPE_NETWORK_EVENT
	case "trace", "artifact_type_trace":
		return basv1.ArtifactType_ARTIFACT_TYPE_TRACE
	case "custom", "artifact_type_custom", "extracted_data", "video", "har":
		return basv1.ArtifactType_ARTIFACT_TYPE_CUSTOM
	default:
		return basv1.ArtifactType_ARTIFACT_TYPE_UNSPECIFIED
	}
}

// ArtifactTypeToString converts an ArtifactType enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func ArtifactTypeToString(artifactType basv1.ArtifactType) string {
	switch artifactType {
	case basv1.ArtifactType_ARTIFACT_TYPE_SCREENSHOT:
		return "screenshot"
	case basv1.ArtifactType_ARTIFACT_TYPE_DOM_SNAPSHOT:
		return "dom_snapshot"
	case basv1.ArtifactType_ARTIFACT_TYPE_TIMELINE_FRAME:
		return "timeline_frame"
	case basv1.ArtifactType_ARTIFACT_TYPE_CONSOLE_LOG:
		return "console_log"
	case basv1.ArtifactType_ARTIFACT_TYPE_NETWORK_EVENT:
		return "network_event"
	case basv1.ArtifactType_ARTIFACT_TYPE_TRACE:
		return "trace"
	case basv1.ArtifactType_ARTIFACT_TYPE_CUSTOM:
		return "custom"
	default:
		return "unknown"
	}
}

// =============================================================================
// HIGHLIGHT COLOR CONVERTERS
// =============================================================================

// StringToHighlightColor converts a color string to the proto HighlightColor enum.
// Handles color names (e.g., "red", "green", "blue", "magenta" -> PINK).
// Returns HIGHLIGHT_COLOR_UNSPECIFIED for unrecognized values.
func StringToHighlightColor(s string) basv1.HighlightColor {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "red":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_RED
	case "green":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_GREEN
	case "blue":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_BLUE
	case "yellow":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_YELLOW
	case "orange":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_ORANGE
	case "purple":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_PURPLE
	case "cyan":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_CYAN
	case "magenta", "pink":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_PINK
	case "white":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_WHITE
	case "gray", "grey":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_GRAY
	case "black":
		return basv1.HighlightColor_HIGHLIGHT_COLOR_BLACK
	default:
		return basv1.HighlightColor_HIGHLIGHT_COLOR_UNSPECIFIED
	}
}

// HighlightColorToString converts a HighlightColor enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func HighlightColorToString(color basv1.HighlightColor) string {
	switch color {
	case basv1.HighlightColor_HIGHLIGHT_COLOR_RED:
		return "red"
	case basv1.HighlightColor_HIGHLIGHT_COLOR_GREEN:
		return "green"
	case basv1.HighlightColor_HIGHLIGHT_COLOR_BLUE:
		return "blue"
	case basv1.HighlightColor_HIGHLIGHT_COLOR_YELLOW:
		return "yellow"
	case basv1.HighlightColor_HIGHLIGHT_COLOR_ORANGE:
		return "orange"
	case basv1.HighlightColor_HIGHLIGHT_COLOR_PURPLE:
		return "purple"
	case basv1.HighlightColor_HIGHLIGHT_COLOR_CYAN:
		return "cyan"
	case basv1.HighlightColor_HIGHLIGHT_COLOR_PINK:
		return "pink"
	case basv1.HighlightColor_HIGHLIGHT_COLOR_WHITE:
		return "white"
	case basv1.HighlightColor_HIGHLIGHT_COLOR_GRAY:
		return "gray"
	case basv1.HighlightColor_HIGHLIGHT_COLOR_BLACK:
		return "black"
	default:
		return "unknown"
	}
}

// =============================================================================
// MOUSE BUTTON CONVERTERS
// =============================================================================

// StringToMouseButton converts a mouse button string to the proto MouseButton enum.
// Handles button names (e.g., "left", "right", "middle").
// Returns MOUSE_BUTTON_UNSPECIFIED for unrecognized values.
func StringToMouseButton(s string) basv1.MouseButton {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "left":
		return basv1.MouseButton_MOUSE_BUTTON_LEFT
	case "right":
		return basv1.MouseButton_MOUSE_BUTTON_RIGHT
	case "middle":
		return basv1.MouseButton_MOUSE_BUTTON_MIDDLE
	default:
		return basv1.MouseButton_MOUSE_BUTTON_UNSPECIFIED
	}
}

// MouseButtonToString converts a MouseButton enum to its canonical string representation.
// Returns "left" as default for unspecified (most common case).
func MouseButtonToString(button basv1.MouseButton) string {
	switch button {
	case basv1.MouseButton_MOUSE_BUTTON_LEFT:
		return "left"
	case basv1.MouseButton_MOUSE_BUTTON_RIGHT:
		return "right"
	case basv1.MouseButton_MOUSE_BUTTON_MIDDLE:
		return "middle"
	default:
		return "left" // Default to left for unspecified
	}
}

// =============================================================================
// KEYBOARD MODIFIER CONVERTERS
// =============================================================================

// StringToKeyboardModifier converts a keyboard modifier string to the proto KeyboardModifier enum.
// Handles modifier names (e.g., "ctrl", "shift", "alt", "meta", "cmd" -> META).
// Returns KEYBOARD_MODIFIER_UNSPECIFIED for unrecognized values.
func StringToKeyboardModifier(s string) basv1.KeyboardModifier {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "ctrl", "control":
		return basv1.KeyboardModifier_KEYBOARD_MODIFIER_CTRL
	case "shift":
		return basv1.KeyboardModifier_KEYBOARD_MODIFIER_SHIFT
	case "alt":
		return basv1.KeyboardModifier_KEYBOARD_MODIFIER_ALT
	case "meta", "cmd", "command", "win", "windows":
		return basv1.KeyboardModifier_KEYBOARD_MODIFIER_META
	default:
		return basv1.KeyboardModifier_KEYBOARD_MODIFIER_UNSPECIFIED
	}
}

// KeyboardModifierToString converts a KeyboardModifier enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func KeyboardModifierToString(modifier basv1.KeyboardModifier) string {
	switch modifier {
	case basv1.KeyboardModifier_KEYBOARD_MODIFIER_CTRL:
		return "ctrl"
	case basv1.KeyboardModifier_KEYBOARD_MODIFIER_SHIFT:
		return "shift"
	case basv1.KeyboardModifier_KEYBOARD_MODIFIER_ALT:
		return "alt"
	case basv1.KeyboardModifier_KEYBOARD_MODIFIER_META:
		return "meta"
	default:
		return "unknown"
	}
}

// StringsToKeyboardModifiers converts a slice of modifier strings to KeyboardModifier enums.
// Filters out any unrecognized modifiers (UNSPECIFIED).
func StringsToKeyboardModifiers(modifiers []string) []basv1.KeyboardModifier {
	if len(modifiers) == 0 {
		return nil
	}
	result := make([]basv1.KeyboardModifier, 0, len(modifiers))
	for _, s := range modifiers {
		mod := StringToKeyboardModifier(s)
		if mod != basv1.KeyboardModifier_KEYBOARD_MODIFIER_UNSPECIFIED {
			result = append(result, mod)
		}
	}
	return result
}

// KeyboardModifiersToStrings converts a slice of KeyboardModifier enums to strings.
func KeyboardModifiersToStrings(modifiers []basv1.KeyboardModifier) []string {
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
// NETWORK EVENT TYPE CONVERTERS
// =============================================================================

// StringToNetworkEventType converts a network event type string to the proto NetworkEventType enum.
// Handles event types (e.g., "request", "response", "failure").
// Returns NETWORK_EVENT_TYPE_UNSPECIFIED for unrecognized values.
func StringToNetworkEventType(s string) basv1.NetworkEventType {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "request":
		return basv1.NetworkEventType_NETWORK_EVENT_TYPE_REQUEST
	case "response":
		return basv1.NetworkEventType_NETWORK_EVENT_TYPE_RESPONSE
	case "failure", "failed":
		return basv1.NetworkEventType_NETWORK_EVENT_TYPE_FAILURE
	default:
		return basv1.NetworkEventType_NETWORK_EVENT_TYPE_UNSPECIFIED
	}
}

// NetworkEventTypeToString converts a NetworkEventType enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func NetworkEventTypeToString(eventType basv1.NetworkEventType) string {
	switch eventType {
	case basv1.NetworkEventType_NETWORK_EVENT_TYPE_REQUEST:
		return "request"
	case basv1.NetworkEventType_NETWORK_EVENT_TYPE_RESPONSE:
		return "response"
	case basv1.NetworkEventType_NETWORK_EVENT_TYPE_FAILURE:
		return "failure"
	default:
		return "unknown"
	}
}

// =============================================================================
// CHANGE SOURCE CONVERTERS
// =============================================================================

// StringToChangeSource converts a change source string to the proto ChangeSource enum.
// Handles source names (e.g., "manual", "autosave", "import", "ai_generated", "recording").
// Returns CHANGE_SOURCE_UNSPECIFIED for unrecognized values.
func StringToChangeSource(s string) basv1.ChangeSource {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "manual":
		return basv1.ChangeSource_CHANGE_SOURCE_MANUAL
	case "autosave":
		return basv1.ChangeSource_CHANGE_SOURCE_AUTOSAVE
	case "import":
		return basv1.ChangeSource_CHANGE_SOURCE_IMPORT
	case "ai_generated", "ai-generated", "aigenerated":
		return basv1.ChangeSource_CHANGE_SOURCE_AI_GENERATED
	case "recording":
		return basv1.ChangeSource_CHANGE_SOURCE_RECORDING
	default:
		return basv1.ChangeSource_CHANGE_SOURCE_UNSPECIFIED
	}
}

// ChangeSourceToString converts a ChangeSource enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func ChangeSourceToString(source basv1.ChangeSource) string {
	switch source {
	case basv1.ChangeSource_CHANGE_SOURCE_MANUAL:
		return "manual"
	case basv1.ChangeSource_CHANGE_SOURCE_AUTOSAVE:
		return "autosave"
	case basv1.ChangeSource_CHANGE_SOURCE_IMPORT:
		return "import"
	case basv1.ChangeSource_CHANGE_SOURCE_AI_GENERATED:
		return "ai_generated"
	case basv1.ChangeSource_CHANGE_SOURCE_RECORDING:
		return "recording"
	default:
		return "unknown"
	}
}

// =============================================================================
// VALIDATION SEVERITY CONVERTERS
// =============================================================================

// StringToValidationSeverity converts a validation severity string to the proto ValidationSeverity enum.
// Handles severity names (e.g., "error", "warning", "info").
// Returns VALIDATION_SEVERITY_UNSPECIFIED for unrecognized values.
func StringToValidationSeverity(s string) basv1.ValidationSeverity {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "error":
		return basv1.ValidationSeverity_VALIDATION_SEVERITY_ERROR
	case "warning", "warn":
		return basv1.ValidationSeverity_VALIDATION_SEVERITY_WARNING
	case "info":
		return basv1.ValidationSeverity_VALIDATION_SEVERITY_INFO
	default:
		return basv1.ValidationSeverity_VALIDATION_SEVERITY_UNSPECIFIED
	}
}

// ValidationSeverityToString converts a ValidationSeverity enum to its canonical string representation.
// Returns "unknown" for unrecognized values.
func ValidationSeverityToString(severity basv1.ValidationSeverity) string {
	switch severity {
	case basv1.ValidationSeverity_VALIDATION_SEVERITY_ERROR:
		return "error"
	case basv1.ValidationSeverity_VALIDATION_SEVERITY_WARNING:
		return "warning"
	case basv1.ValidationSeverity_VALIDATION_SEVERITY_INFO:
		return "info"
	default:
		return "unknown"
	}
}
