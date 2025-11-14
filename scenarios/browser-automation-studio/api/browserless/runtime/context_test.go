package runtime

import (
	"fmt"
	"reflect"
	"testing"
)

func TestExecutionContextSetAndGet(t *testing.T) {
	ctx := NewExecutionContext()
	if err := ctx.Set("token", "abc123"); err != nil {
		t.Fatalf("unexpected error setting variable: %v", err)
	}
	value, ok := ctx.Get("token")
	if !ok || value != "abc123" {
		t.Fatalf("expected token to be stored, got %v (ok=%v)", value, ok)
	}
}

func TestInterpolateInstructionReplacesVariables(t *testing.T) {
	ctx := NewExecutionContext()
	if err := ctx.Set("username", "Ava"); err != nil {
		t.Fatalf("failed to seed context: %v", err)
	}
	instr := Instruction{
		NodeID: "type-1",
		Type:   "type",
		Params: InstructionParam{Text: "Hello {{username}}"},
	}
	resolved, missing, err := InterpolateInstruction(instr, ctx)
	if err != nil {
		t.Fatalf("unexpected error interpolating instruction: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing variables, got %v", missing)
	}
	if resolved.Params.Text != "Hello Ava" {
		t.Fatalf("expected interpolation to replace placeholder, got %q", resolved.Params.Text)
	}
	if resolved.Context == nil || resolved.Context["username"] != "Ava" {
		t.Fatalf("expected context snapshot to include username")
	}
}

func TestInterpolateInstructionCollectsMissingVars(t *testing.T) {
	ctx := NewExecutionContext()
	instr := Instruction{
		NodeID: "type-2",
		Type:   "type",
		Params: InstructionParam{Text: "{{missing}}"},
	}
	_, missing, err := InterpolateInstruction(instr, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(missing) != 1 || missing[0] != "missing" {
		t.Fatalf("expected missing variable list, got %v", missing)
	}
}

func TestExecutionContextRejectsBlankName(t *testing.T) {
	ctx := NewExecutionContext()
	if err := ctx.Set("   ", "value"); err == nil {
		t.Fatalf("expected error when setting blank variable name")
	}
}

type stringerValue string

func (s stringerValue) String() string {
	return fmt.Sprintf("stringer:%s", string(s))
}

func TestExecutionContextGetStringCastsTypes(t *testing.T) {
	ctx := NewExecutionContext()
	if err := ctx.Set("raw", 123); err != nil {
		t.Fatalf("failed to seed context: %v", err)
	}
	if err := ctx.Set("stringer", stringerValue("value")); err != nil {
		t.Fatalf("failed to seed stringer value: %v", err)
	}
	value, err := ctx.GetString("raw")
	if err != nil {
		t.Fatalf("unexpected error coercing int to string: %v", err)
	}
	if value != "123" {
		t.Fatalf("expected numeric value to stringify, got %q", value)
	}
	str, err := ctx.GetString("stringer")
	if err != nil {
		t.Fatalf("unexpected error coercing stringer: %v", err)
	}
	if str != "stringer:value" {
		t.Fatalf("expected stringer to call String(), got %q", str)
	}
}

func TestExecutionContextGetIntConversions(t *testing.T) {
	ctx := NewExecutionContext()
	seed := map[string]any{
		"int":    42,
		"string": "19",
	}
	for key, value := range seed {
		if err := ctx.Set(key, value); err != nil {
			t.Fatalf("failed to seed %s: %v", key, err)
		}
	}
	tests := map[string]int{"int": 42, "string": 19}
	for key, want := range tests {
		got, err := ctx.GetInt(key)
		if err != nil {
			t.Fatalf("unexpected error fetching %s: %v", key, err)
		}
		if got != want {
			t.Fatalf("expected %d for %s, got %d", want, key, got)
		}
	}
	if err := ctx.Set("invalid", "abc"); err != nil {
		t.Fatalf("failed to seed invalid string: %v", err)
	}
	if _, err := ctx.GetInt("invalid"); err == nil {
		t.Fatalf("expected error for non-numeric string")
	}
}

func TestExecutionContextGetBoolConversions(t *testing.T) {
	ctx := NewExecutionContext()
	seed := map[string]any{
		"bool":   true,
		"string": "true",
		"int":    1,
	}
	for key, value := range seed {
		if err := ctx.Set(key, value); err != nil {
			t.Fatalf("failed to seed %s: %v", key, err)
		}
	}
	cases := map[string]bool{"bool": true, "string": true, "int": true}
	for key, want := range cases {
		got, err := ctx.GetBool(key)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", key, err)
		}
		if got != want {
			t.Fatalf("expected %v for %s, got %v", want, key, got)
		}
	}
	if err := ctx.Set("invalid", "not-bool"); err != nil {
		t.Fatalf("failed to seed invalid bool: %v", err)
	}
	if _, err := ctx.GetBool("invalid"); err == nil {
		t.Fatalf("expected error for invalid bool conversion")
	}
}

func TestExecutionContextSnapshotIsolation(t *testing.T) {
	ctx := NewExecutionContext()
	if err := ctx.Set("token", "first"); err != nil {
		t.Fatalf("failed to seed token: %v", err)
	}
	snapshot := ctx.Snapshot()
	if err := ctx.Set("token", "second"); err != nil {
		t.Fatalf("failed to update token: %v", err)
	}
	if snapshot["token"] != "first" {
		t.Fatalf("expected snapshot to keep original value, got %v", snapshot["token"])
	}
	if _, ok := snapshot["newKey"]; ok {
		t.Fatalf("snapshot should not include keys added later")
	}
}

func TestInterpolateMapNestedStructures(t *testing.T) {
	ctx := NewExecutionContext()
	ctx.Set("user", "Ada")
	ctx.Set("city", "Paris")
	input := map[string]any{
		"greeting": "Hello {{user}}",
		"metadata": map[string]any{
			"city":    "{{city}}",
			"missing": "{{unknown}}",
		},
		"list": []any{
			"{{user}}",
			map[string]any{"nested": "{{city}}"},
			[]any{"{{unknown}}"},
		},
	}
	result, missing := interpolateMap(input, ctx)
	if result["greeting"].(string) != "Hello Ada" {
		t.Fatalf("expected greeting interpolation, got %v", result["greeting"])
	}
	metadata := result["metadata"].(map[string]any)
	if metadata["city"].(string) != "Paris" {
		t.Fatalf("expected nested map interpolation, got %v", metadata["city"])
	}
	list := result["list"].([]any)
	if list[0].(string) != "Ada" {
		t.Fatalf("expected list interpolation, got %v", list[0])
	}
	nestedMap := list[1].(map[string]any)
	if nestedMap["nested"].(string) != "Paris" {
		t.Fatalf("expected nested map inside slice to interpolate, got %v", nestedMap["nested"])
	}
	nestedSlice := list[2].([]any)
	if nestedSlice[0].(string) != "{{unknown}}" {
		t.Fatalf("expected missing placeholder to remain unchanged, got %v", nestedSlice[0])
	}
	expectedMissing := []string{"unknown"}
	if !reflect.DeepEqual(missing, expectedMissing) {
		t.Fatalf("expected missing %v, got %v", expectedMissing, missing)
	}
}

func TestDedupeStringsSortsAndDedupes(t *testing.T) {
	input := []string{"b", "a", "b", "c", "a"}
	result := dedupeStrings(input)
	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}
