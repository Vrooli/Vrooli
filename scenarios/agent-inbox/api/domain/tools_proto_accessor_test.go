package domain

import (
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

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
