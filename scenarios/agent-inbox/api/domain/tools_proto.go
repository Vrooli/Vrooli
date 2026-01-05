// Package domain provides helper functions for working with proto-generated
// tool types from the Tool Discovery Protocol.
//
// These functions convert proto types to formats expected by external systems
// (e.g., OpenAI function calling format) and provide utilities for working
// with proto's JsonValue types.
package domain

import (
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// ToolProtocolVersion is the version of the Tool Discovery Protocol we support.
const ToolProtocolVersion = "1.0"

// -----------------------------------------------------------------------------
// OpenAI Function Format Conversion
// -----------------------------------------------------------------------------

// ToOpenAIFunction converts a proto ToolDefinition to OpenAI function format.
// This is used when sending tools to the LLM via OpenRouter.
func ToOpenAIFunction(t *toolspb.ToolDefinition) map[string]interface{} {
	if t == nil {
		return nil
	}
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  ToolParametersToMap(t.Parameters),
		},
	}
}

// ToolParametersToMap converts proto ToolParameters to a generic map for JSON encoding.
func ToolParametersToMap(p *toolspb.ToolParameters) map[string]interface{} {
	if p == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	props := make(map[string]interface{})
	for name, schema := range p.Properties {
		props[name] = ParameterSchemaToMap(schema)
	}

	result := map[string]interface{}{
		"type":       p.Type,
		"properties": props,
	}

	if len(p.Required) > 0 {
		result["required"] = p.Required
	}

	return result
}

// ParameterSchemaToMap converts proto ParameterSchema to a generic map for JSON encoding.
func ParameterSchemaToMap(s *toolspb.ParameterSchema) map[string]interface{} {
	if s == nil {
		return nil
	}

	result := map[string]interface{}{
		"type": s.Type,
	}

	if s.Description != "" {
		result["description"] = s.Description
	}
	if len(s.Enum) > 0 {
		result["enum"] = s.Enum
	}
	if s.Default != nil {
		result["default"] = JsonValueToInterface(s.Default)
	}
	if s.Format != "" {
		result["format"] = s.Format
	}
	if s.Items != nil {
		result["items"] = ParameterSchemaToMap(s.Items)
	}
	if len(s.Properties) > 0 {
		props := make(map[string]interface{})
		for name, prop := range s.Properties {
			props[name] = ParameterSchemaToMap(prop)
		}
		result["properties"] = props
	}
	if s.Minimum != nil {
		result["minimum"] = *s.Minimum
	}
	if s.Maximum != nil {
		result["maximum"] = *s.Maximum
	}
	if s.MinLength != nil {
		result["minLength"] = *s.MinLength
	}
	if s.MaxLength != nil {
		result["maxLength"] = *s.MaxLength
	}
	if s.Pattern != "" {
		result["pattern"] = s.Pattern
	}

	return result
}

// -----------------------------------------------------------------------------
// JsonValue Conversion
// -----------------------------------------------------------------------------

// JsonValueToInterface converts a proto JsonValue to a Go interface{}.
// This is useful when you need to work with dynamic values from proto types.
func JsonValueToInterface(v *commonv1.JsonValue) interface{} {
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
	case *commonv1.JsonValue_ObjectValue:
		return JsonObjectToMap(k.ObjectValue)
	case *commonv1.JsonValue_ListValue:
		return JsonListToSlice(k.ListValue)
	case *commonv1.JsonValue_NullValue:
		return nil
	case *commonv1.JsonValue_BytesValue:
		return k.BytesValue
	default:
		return nil
	}
}

// JsonObjectToMap converts a proto JsonObject to a Go map.
func JsonObjectToMap(obj *commonv1.JsonObject) map[string]interface{} {
	if obj == nil || obj.Fields == nil {
		return nil
	}

	result := make(map[string]interface{}, len(obj.Fields))
	for k, v := range obj.Fields {
		result[k] = JsonValueToInterface(v)
	}
	return result
}

// JsonListToSlice converts a proto JsonList to a Go slice.
func JsonListToSlice(list *commonv1.JsonList) []interface{} {
	if list == nil || list.Values == nil {
		return nil
	}

	result := make([]interface{}, len(list.Values))
	for i, v := range list.Values {
		result[i] = JsonValueToInterface(v)
	}
	return result
}

// -----------------------------------------------------------------------------
// Interface to JsonValue Conversion
// -----------------------------------------------------------------------------

// InterfaceToJsonValue converts a Go interface{} to a proto JsonValue.
// This is useful when building tool examples or dynamic values.
func InterfaceToJsonValue(v interface{}) *commonv1.JsonValue {
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
	case float32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: float64(val)}}
	case float64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: val}}
	case []byte:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BytesValue{BytesValue: val}}
	case map[string]interface{}:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{ObjectValue: MapToJsonObject(val)}}
	case []interface{}:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{ListValue: SliceToJsonList(val)}}
	default:
		// For unknown types, try to convert to string
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: ""}}
	}
}

// MapToJsonObject converts a Go map to a proto JsonObject.
func MapToJsonObject(m map[string]interface{}) *commonv1.JsonObject {
	if m == nil {
		return nil
	}

	fields := make(map[string]*commonv1.JsonValue, len(m))
	for k, v := range m {
		fields[k] = InterfaceToJsonValue(v)
	}
	return &commonv1.JsonObject{Fields: fields}
}

// SliceToJsonList converts a Go slice to a proto JsonList.
func SliceToJsonList(s []interface{}) *commonv1.JsonList {
	if s == nil {
		return nil
	}

	values := make([]*commonv1.JsonValue, len(s))
	for i, v := range s {
		values[i] = InterfaceToJsonValue(v)
	}
	return &commonv1.JsonList{Values: values}
}

// -----------------------------------------------------------------------------
// Convenience Accessors for Proto Types
// -----------------------------------------------------------------------------

// GetAsyncBehavior returns the async behavior from a tool definition, or nil if not set.
func GetAsyncBehavior(t *toolspb.ToolDefinition) *toolspb.AsyncBehavior {
	if t == nil || t.Metadata == nil {
		return nil
	}
	return t.Metadata.AsyncBehavior
}

// IsLongRunning returns true if the tool is marked as long-running.
func IsLongRunning(t *toolspb.ToolDefinition) bool {
	if t == nil || t.Metadata == nil {
		return false
	}
	return t.Metadata.LongRunning
}

// HasAsyncBehavior returns true if the tool has async behavior configured.
func HasAsyncBehavior(t *toolspb.ToolDefinition) bool {
	return GetAsyncBehavior(t) != nil
}

// GetStatusPolling returns the status polling config, or nil if not set.
func GetStatusPolling(t *toolspb.ToolDefinition) *toolspb.StatusPolling {
	async := GetAsyncBehavior(t)
	if async == nil {
		return nil
	}
	return async.StatusPolling
}

// GetCompletionConditions returns the completion conditions, or nil if not set.
func GetCompletionConditions(t *toolspb.ToolDefinition) *toolspb.CompletionConditions {
	async := GetAsyncBehavior(t)
	if async == nil {
		return nil
	}
	return async.CompletionConditions
}

// GetProgressTracking returns the progress tracking config, or nil if not set.
func GetProgressTracking(t *toolspb.ToolDefinition) *toolspb.ProgressTracking {
	async := GetAsyncBehavior(t)
	if async == nil {
		return nil
	}
	return async.ProgressTracking
}

// GetCancellationBehavior returns the cancellation config, or nil if not set.
func GetCancellationBehavior(t *toolspb.ToolDefinition) *toolspb.CancellationBehavior {
	async := GetAsyncBehavior(t)
	if async == nil {
		return nil
	}
	return async.Cancellation
}
