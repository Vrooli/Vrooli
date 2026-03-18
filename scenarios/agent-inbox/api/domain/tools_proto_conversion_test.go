package domain

import (
	"reflect"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// ToOpenAIFunction Tests

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
}

func TestToOpenAIFunction_WithParameters(t *testing.T) {
	tool := &toolspb.ToolDefinition{
		Name:        "search",
		Description: "Search for items",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"query": {Type: "string", Description: "Search query"},
				"limit": {Type: "integer", Description: "Max results"},
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

	required := params["required"].([]string)
	if len(required) != 1 || required[0] != "query" {
		t.Errorf("required = %v", required)
	}
}

// ToolParametersToMap Tests

func TestToolParametersToMap_Nil(t *testing.T) {
	result := ToolParametersToMap(nil)
	if result["type"] != "object" {
		t.Errorf("type = %v, want 'object'", result["type"])
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

// ParameterSchemaToMap Tests

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
	if !reflect.DeepEqual(result["enum"], []string{"a", "b", "c"}) {
		t.Errorf("enum = %v", result["enum"])
	}
	if result["default"] != "a" {
		t.Errorf("default = %v", result["default"])
	}
}

func TestParameterSchemaToMap_ArrayType(t *testing.T) {
	schema := &toolspb.ParameterSchema{
		Type:  "array",
		Items: &toolspb.ParameterSchema{Type: "string"},
	}

	result := ParameterSchemaToMap(schema)
	items := result["items"].(map[string]interface{})
	if items["type"] != "string" {
		t.Errorf("items.type = %v", items["type"])
	}
}

// Proto Accessor Tests

func TestGetAsyncBehavior(t *testing.T) {
	if GetAsyncBehavior(nil) != nil {
		t.Error("GetAsyncBehavior(nil) should return nil")
	}

	tool := &toolspb.ToolDefinition{Name: "test"}
	if GetAsyncBehavior(tool) != nil {
		t.Error("GetAsyncBehavior with no metadata should return nil")
	}

	tool.Metadata = &toolspb.ToolMetadata{}
	if GetAsyncBehavior(tool) != nil {
		t.Error("GetAsyncBehavior with no async should return nil")
	}

	tool.Metadata.AsyncBehavior = &toolspb.AsyncBehavior{}
	if GetAsyncBehavior(tool) == nil {
		t.Error("GetAsyncBehavior should return AsyncBehavior")
	}
}

func TestIsLongRunning(t *testing.T) {
	if IsLongRunning(nil) {
		t.Error("IsLongRunning(nil) should return false")
	}

	tool := &toolspb.ToolDefinition{
		Name:     "test",
		Metadata: &toolspb.ToolMetadata{LongRunning: true},
	}
	if !IsLongRunning(tool) {
		t.Error("IsLongRunning should return true")
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
