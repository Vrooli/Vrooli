package domain

import (
	"reflect"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// =============================================================================
// ToOpenAIFunction Tests
// =============================================================================

func TestToOpenAIFunction_Nil(t *testing.T) {
	result := ToOpenAIFunction(nil)
	if result != nil {
		t.Errorf("ToOpenAIFunction(nil) = %v, want nil", result)
	}
}

func TestToOpenAIFunction_Basic(t *testing.T) {
	tool := &toolspb.ToolDefinition{
		Name:        "get_weather",
		Description: "Get the current weather",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
		},
	}

	result := ToOpenAIFunction(tool)

	if result["type"] != "function" {
		t.Errorf("type = %v, want 'function'", result["type"])
	}

	fn, ok := result["function"].(map[string]interface{})
	if !ok {
		t.Fatal("function is not a map")
	}

	if fn["name"] != "get_weather" {
		t.Errorf("name = %v, want 'get_weather'", fn["name"])
	}
	if fn["description"] != "Get the current weather" {
		t.Errorf("description = %v", fn["description"])
	}
}

func TestToOpenAIFunction_WithParameters(t *testing.T) {
	tool := &toolspb.ToolDefinition{
		Name:        "search",
		Description: "Search for items",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"query": {
					Type:        "string",
					Description: "Search query",
				},
				"limit": {
					Type:        "integer",
					Description: "Max results",
				},
			},
			Required: []string{"query"},
		},
	}

	result := ToOpenAIFunction(tool)
	fn := result["function"].(map[string]interface{})
	params := fn["parameters"].(map[string]interface{})

	if params["type"] != "object" {
		t.Errorf("parameters.type = %v", params["type"])
	}

	props := params["properties"].(map[string]interface{})
	if len(props) != 2 {
		t.Errorf("expected 2 properties, got %d", len(props))
	}

	query := props["query"].(map[string]interface{})
	if query["type"] != "string" {
		t.Errorf("query.type = %v", query["type"])
	}

	required := params["required"].([]string)
	if len(required) != 1 || required[0] != "query" {
		t.Errorf("required = %v", required)
	}
}

// =============================================================================
// ToolParametersToMap Tests
// =============================================================================

func TestToolParametersToMap_Nil(t *testing.T) {
	result := ToolParametersToMap(nil)

	if result["type"] != "object" {
		t.Errorf("type = %v, want 'object'", result["type"])
	}
	props := result["properties"].(map[string]interface{})
	if len(props) != 0 {
		t.Errorf("properties should be empty, got %v", props)
	}
}

func TestToolParametersToMap_WithRequired(t *testing.T) {
	params := &toolspb.ToolParameters{
		Type: "object",
		Properties: map[string]*toolspb.ParameterSchema{
			"name": {Type: "string"},
		},
		Required: []string{"name"},
	}

	result := ToolParametersToMap(params)

	required, ok := result["required"].([]string)
	if !ok {
		t.Fatal("required is not a []string")
	}
	if len(required) != 1 || required[0] != "name" {
		t.Errorf("required = %v", required)
	}
}

func TestToolParametersToMap_NoRequired(t *testing.T) {
	params := &toolspb.ToolParameters{
		Type:       "object",
		Properties: map[string]*toolspb.ParameterSchema{},
		Required:   []string{}, // Empty
	}

	result := ToolParametersToMap(params)

	if _, exists := result["required"]; exists {
		t.Error("required should not be present when empty")
	}
}

// =============================================================================
// ParameterSchemaToMap Tests
// =============================================================================

func TestParameterSchemaToMap_Nil(t *testing.T) {
	result := ParameterSchemaToMap(nil)
	if result != nil {
		t.Errorf("ParameterSchemaToMap(nil) = %v, want nil", result)
	}
}

func TestParameterSchemaToMap_AllFields(t *testing.T) {
	min := 0.0
	max := 100.0
	minLen := int32(1)
	maxLen := int32(50)

	schema := &toolspb.ParameterSchema{
		Type:        "string",
		Description: "A test field",
		Enum:        []string{"a", "b", "c"},
		Default:     &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: "a"}},
		Format:      "email",
		Minimum:     &min,
		Maximum:     &max,
		MinLength:   &minLen,
		MaxLength:   &maxLen,
		Pattern:     "^[a-z]+$",
	}

	result := ParameterSchemaToMap(schema)

	if result["type"] != "string" {
		t.Errorf("type = %v", result["type"])
	}
	if result["description"] != "A test field" {
		t.Errorf("description = %v", result["description"])
	}
	if !reflect.DeepEqual(result["enum"], []string{"a", "b", "c"}) {
		t.Errorf("enum = %v", result["enum"])
	}
	if result["default"] != "a" {
		t.Errorf("default = %v", result["default"])
	}
	if result["format"] != "email" {
		t.Errorf("format = %v", result["format"])
	}
	if result["minimum"] != 0.0 {
		t.Errorf("minimum = %v", result["minimum"])
	}
	if result["maximum"] != 100.0 {
		t.Errorf("maximum = %v", result["maximum"])
	}
	if result["minLength"] != int32(1) {
		t.Errorf("minLength = %v", result["minLength"])
	}
	if result["maxLength"] != int32(50) {
		t.Errorf("maxLength = %v", result["maxLength"])
	}
	if result["pattern"] != "^[a-z]+$" {
		t.Errorf("pattern = %v", result["pattern"])
	}
}

func TestParameterSchemaToMap_ArrayType(t *testing.T) {
	schema := &toolspb.ParameterSchema{
		Type: "array",
		Items: &toolspb.ParameterSchema{
			Type: "string",
		},
	}

	result := ParameterSchemaToMap(schema)

	if result["type"] != "array" {
		t.Errorf("type = %v", result["type"])
	}

	items := result["items"].(map[string]interface{})
	if items["type"] != "string" {
		t.Errorf("items.type = %v", items["type"])
	}
}

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
			"name": {Kind: &commonv1.JsonValue_StringValue{StringValue: "test"}},
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
		name     string
		input    interface{}
		checkFn  func(*commonv1.JsonValue) bool
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

func TestInterfaceToJsonValue_Map(t *testing.T) {
	input := map[string]interface{}{
		"name": "test",
		"age":  42,
	}

	result := InterfaceToJsonValue(input)
	k, ok := result.Kind.(*commonv1.JsonValue_ObjectValue)
	if !ok {
		t.Fatalf("expected ObjectValue, got %T", result.Kind)
	}

	if len(k.ObjectValue.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(k.ObjectValue.Fields))
	}
}

func TestInterfaceToJsonValue_Slice(t *testing.T) {
	input := []interface{}{1, 2, 3}

	result := InterfaceToJsonValue(input)
	k, ok := result.Kind.(*commonv1.JsonValue_ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", result.Kind)
	}

	if len(k.ListValue.Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(k.ListValue.Values))
	}
}

func TestInterfaceToJsonValue_UnknownType(t *testing.T) {
	// Unknown types become empty strings
	type custom struct{}
	result := InterfaceToJsonValue(custom{})

	k, ok := result.Kind.(*commonv1.JsonValue_StringValue)
	if !ok {
		t.Fatalf("expected StringValue for unknown type, got %T", result.Kind)
	}
	if k.StringValue != "" {
		t.Errorf("expected empty string, got %q", k.StringValue)
	}
}

func TestMapToJsonObject_Nil(t *testing.T) {
	result := MapToJsonObject(nil)
	if result != nil {
		t.Errorf("MapToJsonObject(nil) = %v, want nil", result)
	}
}

func TestSliceToJsonList_Nil(t *testing.T) {
	result := SliceToJsonList(nil)
	if result != nil {
		t.Errorf("SliceToJsonList(nil) = %v, want nil", result)
	}
}

// =============================================================================
// Proto Accessor Tests
// =============================================================================

func TestGetAsyncBehavior(t *testing.T) {
	// Nil tool
	if GetAsyncBehavior(nil) != nil {
		t.Error("GetAsyncBehavior(nil) should return nil")
	}

	// No metadata
	tool := &toolspb.ToolDefinition{Name: "test"}
	if GetAsyncBehavior(tool) != nil {
		t.Error("GetAsyncBehavior with no metadata should return nil")
	}

	// No async behavior
	tool.Metadata = &toolspb.ToolMetadata{}
	if GetAsyncBehavior(tool) != nil {
		t.Error("GetAsyncBehavior with no async should return nil")
	}

	// Has async behavior
	tool.Metadata.AsyncBehavior = &toolspb.AsyncBehavior{}
	if GetAsyncBehavior(tool) == nil {
		t.Error("GetAsyncBehavior should return AsyncBehavior")
	}
}

func TestIsLongRunning(t *testing.T) {
	if IsLongRunning(nil) {
		t.Error("IsLongRunning(nil) should return false")
	}

	tool := &toolspb.ToolDefinition{Name: "test"}
	if IsLongRunning(tool) {
		t.Error("IsLongRunning with no metadata should return false")
	}

	tool.Metadata = &toolspb.ToolMetadata{LongRunning: true}
	if !IsLongRunning(tool) {
		t.Error("IsLongRunning should return true")
	}
}

func TestHasAsyncBehavior(t *testing.T) {
	if HasAsyncBehavior(nil) {
		t.Error("HasAsyncBehavior(nil) should return false")
	}

	tool := &toolspb.ToolDefinition{
		Name: "test",
		Metadata: &toolspb.ToolMetadata{
			AsyncBehavior: &toolspb.AsyncBehavior{},
		},
	}
	if !HasAsyncBehavior(tool) {
		t.Error("HasAsyncBehavior should return true")
	}
}

func TestGetStatusPolling(t *testing.T) {
	if GetStatusPolling(nil) != nil {
		t.Error("GetStatusPolling(nil) should return nil")
	}

	tool := &toolspb.ToolDefinition{
		Name: "test",
		Metadata: &toolspb.ToolMetadata{
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{StatusTool: "check_status"},
			},
		},
	}
	sp := GetStatusPolling(tool)
	if sp == nil || sp.StatusTool != "check_status" {
		t.Errorf("GetStatusPolling = %v", sp)
	}
}

func TestGetCompletionConditions(t *testing.T) {
	if GetCompletionConditions(nil) != nil {
		t.Error("GetCompletionConditions(nil) should return nil")
	}

	tool := &toolspb.ToolDefinition{
		Name: "test",
		Metadata: &toolspb.ToolMetadata{
			AsyncBehavior: &toolspb.AsyncBehavior{
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"completed"},
				},
			},
		},
	}
	cc := GetCompletionConditions(tool)
	if cc == nil || cc.StatusField != "status" {
		t.Errorf("GetCompletionConditions = %v", cc)
	}
}

func TestGetProgressTracking(t *testing.T) {
	if GetProgressTracking(nil) != nil {
		t.Error("GetProgressTracking(nil) should return nil")
	}

	tool := &toolspb.ToolDefinition{
		Name: "test",
		Metadata: &toolspb.ToolMetadata{
			AsyncBehavior: &toolspb.AsyncBehavior{
				ProgressTracking: &toolspb.ProgressTracking{ProgressField: "progress"},
			},
		},
	}
	pt := GetProgressTracking(tool)
	if pt == nil || pt.ProgressField != "progress" {
		t.Errorf("GetProgressTracking = %v", pt)
	}
}

func TestGetCancellationBehavior(t *testing.T) {
	if GetCancellationBehavior(nil) != nil {
		t.Error("GetCancellationBehavior(nil) should return nil")
	}

	tool := &toolspb.ToolDefinition{
		Name: "test",
		Metadata: &toolspb.ToolMetadata{
			AsyncBehavior: &toolspb.AsyncBehavior{
				Cancellation: &toolspb.CancellationBehavior{CancelTool: "cancel_op"},
			},
		},
	}
	cb := GetCancellationBehavior(tool)
	if cb == nil || cb.CancelTool != "cancel_op" {
		t.Errorf("GetCancellationBehavior = %v", cb)
	}
}
