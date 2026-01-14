// Package toolregistry provides helper functions for constructing proto types.
package toolregistry

import (
	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// StringJsonValue creates a JsonValue containing a string.
func StringJsonValue(s string) *commonv1.JsonValue {
	return &commonv1.JsonValue{
		Kind: &commonv1.JsonValue_StringValue{StringValue: s},
	}
}

// IntJsonValue creates a JsonValue containing an integer.
func IntJsonValue(i int) *commonv1.JsonValue {
	return &commonv1.JsonValue{
		Kind: &commonv1.JsonValue_IntValue{IntValue: int64(i)},
	}
}

// BoolJsonValue creates a JsonValue containing a boolean.
func BoolJsonValue(b bool) *commonv1.JsonValue {
	return &commonv1.JsonValue{
		Kind: &commonv1.JsonValue_BoolValue{BoolValue: b},
	}
}

// NewToolExample creates a new ToolExample with the given description and arguments.
func NewToolExample(description string, args map[string]interface{}) *toolspb.ToolExample {
	fields := make(map[string]*commonv1.JsonValue)
	for k, v := range args {
		switch val := v.(type) {
		case string:
			fields[k] = StringJsonValue(val)
		case int:
			fields[k] = IntJsonValue(val)
		case int64:
			fields[k] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: val}}
		case float64:
			fields[k] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val}}
		case bool:
			fields[k] = BoolJsonValue(val)
		case []string:
			// Convert string slice to JsonList
			values := make([]*commonv1.JsonValue, len(val))
			for i, s := range val {
				values[i] = StringJsonValue(s)
			}
			fields[k] = &commonv1.JsonValue{
				Kind: &commonv1.JsonValue_ListValue{
					ListValue: &commonv1.JsonList{Values: values},
				},
			}
		}
	}

	return &toolspb.ToolExample{
		Description: description,
		Input:       &commonv1.JsonObject{Fields: fields},
	}
}

// NewStringParam creates a string parameter schema.
func NewStringParam(description string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "string",
		Description: description,
	}
}

// NewStringParamWithDefault creates a string parameter schema with a default value.
func NewStringParamWithDefault(description, defaultValue string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "string",
		Description: description,
		Default:     StringJsonValue(defaultValue),
	}
}

// NewStringParamWithEnum creates a string parameter with enumerated values.
func NewStringParamWithEnum(description string, enumValues []string, defaultValue string) *toolspb.ParameterSchema {
	schema := &toolspb.ParameterSchema{
		Type:        "string",
		Description: description,
		Enum:        enumValues,
	}
	if defaultValue != "" {
		schema.Default = StringJsonValue(defaultValue)
	}
	return schema
}

// NewIntParam creates an integer parameter schema.
func NewIntParam(description string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "integer",
		Description: description,
	}
}

// NewIntParamWithDefault creates an integer parameter schema with a default value.
func NewIntParamWithDefault(description string, defaultValue int) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "integer",
		Description: description,
		Default:     IntJsonValue(defaultValue),
	}
}

// NewBoolParam creates a boolean parameter schema.
func NewBoolParam(description string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "boolean",
		Description: description,
	}
}

// NewBoolParamWithDefault creates a boolean parameter schema with a default value.
func NewBoolParamWithDefault(description string, defaultValue bool) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "boolean",
		Description: description,
		Default:     BoolJsonValue(defaultValue),
	}
}

// NewFloatParam creates a number parameter schema.
func NewFloatParam(description string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "number",
		Description: description,
	}
}

// NewObjectParams creates a ToolParameters with the given properties and required fields.
func NewObjectParams(properties map[string]*toolspb.ParameterSchema, required []string) *toolspb.ToolParameters {
	return &toolspb.ToolParameters{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// NewEmptyParams creates a ToolParameters with no required properties.
func NewEmptyParams() *toolspb.ToolParameters {
	return &toolspb.ToolParameters{
		Type:       "object",
		Properties: map[string]*toolspb.ParameterSchema{},
	}
}
