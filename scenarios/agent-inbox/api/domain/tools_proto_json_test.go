package domain

import (
	"reflect"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestParameterSchemaToMap_NestedObject(t *testing.T) {
	schema := &toolspb.ParameterSchema{
		Type: "object",
		Properties: map[string]*toolspb.ParameterSchema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
	}

	result := ParameterSchemaToMap(schema)

	props := result["properties"].(map[string]interface{})
	if len(props) != 2 {
		t.Errorf("expected 2 properties, got %d", len(props))
	}

	name := props["name"].(map[string]interface{})
	if name["type"] != "string" {
		t.Errorf("name.type = %v", name["type"])
	}
}

// =============================================================================
// JsonValue Conversion Tests
// =============================================================================

func TestJsonValueToInterface_Nil(t *testing.T) {
	result := JsonValueToInterface(nil)
	if result != nil {
		t.Errorf("JsonValueToInterface(nil) = %v, want nil", result)
	}
}

func TestJsonValueToInterface_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    *commonv1.JsonValue
		expected interface{}
	}{
		{
			name:     "bool true",
			input:    &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: true}},
			expected: true,
		},
		{
			name:     "bool false",
			input:    &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: false}},
			expected: false,
		},
		{
			name:     "int",
			input:    &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: 42}},
			expected: int64(42),
		},
		{
			name:     "double",
			input:    &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: 3.14}},
			expected: 3.14,
		},
		{
			name:     "string",
			input:    &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: "hello"}},
			expected: "hello",
		},
		{
			name:     "null",
			input:    &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{}},
			expected: nil,
		},
		{
			name:     "bytes",
			input:    &commonv1.JsonValue{Kind: &commonv1.JsonValue_BytesValue{BytesValue: []byte{1, 2, 3}}},
			expected: []byte{1, 2, 3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := JsonValueToInterface(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tc.expected, tc.expected)
			}
		})
	}
}

func TestJsonValueToInterface_Object(t *testing.T) {
	obj := &commonv1.JsonObject{
		Fields: map[string]*commonv1.JsonValue{
			"name":  {Kind: &commonv1.JsonValue_StringValue{StringValue: "test"}},
			"count": {Kind: &commonv1.JsonValue_IntValue{IntValue: 5}},
		},
	}
	input := &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{ObjectValue: obj}}

	result := JsonValueToInterface(input)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map, got %T", result)
	}

	if m["name"] != "test" {
		t.Errorf("name = %v", m["name"])
	}
	if m["count"] != int64(5) {
		t.Errorf("count = %v", m["count"])
	}
}

func TestJsonValueToInterface_List(t *testing.T) {
	list := &commonv1.JsonList{
		Values: []*commonv1.JsonValue{
			{Kind: &commonv1.JsonValue_IntValue{IntValue: 1}},
			{Kind: &commonv1.JsonValue_IntValue{IntValue: 2}},
			{Kind: &commonv1.JsonValue_IntValue{IntValue: 3}},
		},
	}
	input := &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{ListValue: list}}

	result := JsonValueToInterface(input)
	slice, ok := result.([]interface{})
	if !ok {
		t.Fatalf("result is not a slice, got %T", result)
	}

	if len(slice) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(slice))
	}
	if slice[0] != int64(1) || slice[1] != int64(2) || slice[2] != int64(3) {
		t.Errorf("slice = %v", slice)
	}
}

func TestJsonObjectToMap_Nil(t *testing.T) {
	result := JsonObjectToMap(nil)
	if result != nil {
		t.Errorf("JsonObjectToMap(nil) = %v, want nil", result)
	}

	// Also test nil fields
	result = JsonObjectToMap(&commonv1.JsonObject{Fields: nil})
	if result != nil {
		t.Errorf("JsonObjectToMap with nil fields = %v, want nil", result)
	}
}

func TestJsonListToSlice_Nil(t *testing.T) {
	result := JsonListToSlice(nil)
	if result != nil {
		t.Errorf("JsonListToSlice(nil) = %v, want nil", result)
	}

	// Also test nil values
	result = JsonListToSlice(&commonv1.JsonList{Values: nil})
	if result != nil {
		t.Errorf("JsonListToSlice with nil values = %v, want nil", result)
	}
}

// =============================================================================
// InterfaceToJsonValue Tests
// =============================================================================

func TestInterfaceToJsonValue_Nil(t *testing.T) {
	result := InterfaceToJsonValue(nil)
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if _, ok := result.Kind.(*commonv1.JsonValue_NullValue); !ok {
		t.Errorf("expected NullValue, got %T", result.Kind)
	}
}

func TestInterfaceToJsonValue_AllTypes(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		checkFn func(*commonv1.JsonValue) bool
	}{
		{
			name:  "bool",
			input: true,
			checkFn: func(v *commonv1.JsonValue) bool {
				k, ok := v.Kind.(*commonv1.JsonValue_BoolValue)
				return ok && k.BoolValue == true
			},
		},
		{
			name:  "int",
			input: 42,
			checkFn: func(v *commonv1.JsonValue) bool {
				k, ok := v.Kind.(*commonv1.JsonValue_IntValue)
				return ok && k.IntValue == 42
			},
		},
		{
			name:  "int32",
			input: int32(100),
			checkFn: func(v *commonv1.JsonValue) bool {
				k, ok := v.Kind.(*commonv1.JsonValue_IntValue)
				return ok && k.IntValue == 100
			},
		},
		{
			name:  "int64",
			input: int64(999),
			checkFn: func(v *commonv1.JsonValue) bool {
				k, ok := v.Kind.(*commonv1.JsonValue_IntValue)
				return ok && k.IntValue == 999
			},
		},
		{
			name:  "float32",
			input: float32(1.5),
			checkFn: func(v *commonv1.JsonValue) bool {
				k, ok := v.Kind.(*commonv1.JsonValue_DoubleValue)
				return ok && k.DoubleValue == float64(float32(1.5))
			},
		},
		{
			name:  "float64",
			input: 3.14,
			checkFn: func(v *commonv1.JsonValue) bool {
				k, ok := v.Kind.(*commonv1.JsonValue_DoubleValue)
				return ok && k.DoubleValue == 3.14
			},
		},
		{
			name:  "string",
			input: "hello",
			checkFn: func(v *commonv1.JsonValue) bool {
				k, ok := v.Kind.(*commonv1.JsonValue_StringValue)
				return ok && k.StringValue == "hello"
			},
		},
		{
			name:  "bytes",
			input: []byte{1, 2, 3},
			checkFn: func(v *commonv1.JsonValue) bool {
				k, ok := v.Kind.(*commonv1.JsonValue_BytesValue)
				return ok && reflect.DeepEqual(k.BytesValue, []byte{1, 2, 3})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := InterfaceToJsonValue(tc.input)
			if !tc.checkFn(result) {
				t.Errorf("conversion failed for %v (%T)", tc.input, tc.input)
			}
		})
	}
}
