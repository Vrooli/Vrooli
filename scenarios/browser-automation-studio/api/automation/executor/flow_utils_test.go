package executor

import (
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// ============================================================================
// Graph routing decisions
// ============================================================================

func TestNextNodeIDConditionalAliases(t *testing.T) {
	exec := &SimpleExecutor{}

	tests := []struct {
		name      string
		condition bool
		edges     []contracts.PlanEdge
		want      string
	}{
		{
			name:      "true branch matches success aliases",
			condition: true,
			edges: []contracts.PlanEdge{
				{Target: "false-edge", Condition: "false"},
				{Target: "true-edge", Condition: "pass"},
			},
			want: "true-edge",
		},
		{
			name:      "false branch matches failure aliases",
			condition: false,
			edges: []contracts.PlanEdge{
				{Target: "true-edge", Condition: "yes"},
				{Target: "false-edge", Condition: "no"},
			},
			want: "false-edge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := contracts.PlanStep{
				Type:     "conditional",
				Outgoing: tt.edges,
			}
			outcome := contracts.StepOutcome{
				Success:   true,
				Condition: &contracts.ConditionOutcome{Outcome: tt.condition},
			}

			got := exec.nextNodeID(step, outcome)
			if got != tt.want {
				t.Fatalf("nextNodeID = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNextNodeIDPrefersFailureOverSuccess(t *testing.T) {
	exec := &SimpleExecutor{}
	step := contracts.PlanStep{
		Outgoing: []contracts.PlanEdge{
			{Target: "success-edge", Condition: "success"},
			{Target: "failure-edge", Condition: "failure"},
		},
	}
	outcome := contracts.StepOutcome{
		Success: false,
		Failure: &contracts.StepFailure{Kind: contracts.FailureKindEngine},
	}

	got := exec.nextNodeID(step, outcome)
	if got != "failure-edge" {
		t.Fatalf("nextNodeID = %s, want failure-edge", got)
	}
}

func TestNextNodeIDUsesImplicitSuccessRoute(t *testing.T) {
	exec := &SimpleExecutor{}
	step := contracts.PlanStep{
		Outgoing: []contracts.PlanEdge{
			{Target: "explicit-success", Condition: "success"},
			{Target: "implicit-success", Condition: ""},
		},
	}
	outcome := contracts.StepOutcome{Success: true}

	got := exec.nextNodeID(step, outcome)
	if got != "explicit-success" {
		t.Fatalf("nextNodeID = %s, want explicit-success", got)
	}
}

func TestNextNodeIDFallsBackToFirstEdge(t *testing.T) {
	exec := &SimpleExecutor{}
	step := contracts.PlanStep{
		Outgoing: []contracts.PlanEdge{
			{Target: "first", Condition: "maybe"},
			{Target: "second", Condition: "later"},
		},
	}

	got := exec.nextNodeID(step, contracts.StepOutcome{Success: false})
	if got != "first" {
		t.Fatalf("nextNodeID = %s, want first", got)
	}
}

func TestEvaluateExpressionBoolLiteral(t *testing.T) {
	if ok, valid := evaluateExpression("true", nil); !valid || !ok {
		t.Fatalf("expected true literal to be valid/true, got ok=%v valid=%v", ok, valid)
	}
	if ok, valid := evaluateExpression("false", nil); !valid || ok {
		t.Fatalf("expected false literal to be valid/false, got ok=%v valid=%v", ok, valid)
	}
}

func TestEvaluateExpressionVariableComparisons(t *testing.T) {
	state := newFlowState(map[string]any{
		"flag":  true,
		"count": 3,
		"name":  "Ada",
		"items": []any{"a", "b"},
	})

	if ok, valid := evaluateExpression("${flag} == true", state); !valid || !ok {
		t.Fatalf("expected flag comparison true, got ok=%v valid=%v", ok, valid)
	}
	if ok, valid := evaluateExpression("${count} > 2", state); !valid || !ok {
		t.Fatalf("expected count > 2 true, got ok=%v valid=%v", ok, valid)
	}
	if ok, valid := evaluateExpression("${count} < 2", state); !valid || ok {
		t.Fatalf("expected count < 2 false, got ok=%v valid=%v", ok, valid)
	}
	if ok, valid := evaluateExpression("${name} == \"Ada\"", state); !valid || !ok {
		t.Fatalf("expected name equality true, got ok=%v valid=%v", ok, valid)
	}
	if ok, valid := evaluateExpression("${items.1} == b", state); !valid || !ok {
		t.Fatalf("expected items.1 == b true, got ok=%v valid=%v", ok, valid)
	}
}

func TestEvaluateExpressionInvalid(t *testing.T) {
	if _, valid := evaluateExpression("", nil); valid {
		t.Fatalf("empty expression should be invalid")
	}
	if _, valid := evaluateExpression("a b", nil); valid {
		t.Fatalf("malformed expression should be invalid")
	}
}

// ============================================================================
// FlowState Tests
// ============================================================================

func TestFlowStateGetSet(t *testing.T) {
	state := newFlowState(map[string]any{"initial": "value"})

	val, ok := state.get("initial")
	if !ok || val != "value" {
		t.Fatalf("expected initial value, got ok=%v val=%v", ok, val)
	}

	state.set("newKey", 42)
	val, ok = state.get("newKey")
	if !ok || val != 42 {
		t.Fatalf("expected newKey=42, got ok=%v val=%v", ok, val)
	}

	_, ok = state.get("nonexistent")
	if ok {
		t.Fatalf("expected nonexistent key to not be found")
	}
}

func TestFlowStateResolve(t *testing.T) {
	state := newFlowState(map[string]any{
		"user": map[string]any{
			"name":    "Alice",
			"address": map[string]any{"city": "Wonderland"},
		},
		"items": []any{"first", "second", "third"},
	})

	// Test simple values
	if v, ok := state.resolve("user.name"); !ok || v != "Alice" {
		t.Errorf("resolve(user.name): expected Alice, got %v", v)
	}
	if v, ok := state.resolve("user.address.city"); !ok || v != "Wonderland" {
		t.Errorf("resolve(user.address.city): expected Wonderland, got %v", v)
	}
	if v, ok := state.resolve("items.0"); !ok || v != "first" {
		t.Errorf("resolve(items.0): expected first, got %v", v)
	}
	if v, ok := state.resolve("items.1"); !ok || v != "second" {
		t.Errorf("resolve(items.1): expected second, got %v", v)
	}
	if v, ok := state.resolve("items.2"); !ok || v != "third" {
		t.Errorf("resolve(items.2): expected third, got %v", v)
	}

	// Test map retrieval (check type, not equality)
	if v, ok := state.resolve("user"); !ok {
		t.Error("resolve(user): expected to find user map")
	} else if _, isMap := v.(map[string]any); !isMap {
		t.Errorf("resolve(user): expected map, got %T", v)
	}

	// Test not found cases
	notFoundCases := []string{
		"items.3",         // out of bounds
		"items.-1",        // negative index
		"user.unknown",    // unknown key
		"nonexistent",     // nonexistent root
		"user.name.extra", // trying to traverse string
	}
	for _, path := range notFoundCases {
		if _, ok := state.resolve(path); ok {
			t.Errorf("resolve(%q): expected not found", path)
		}
	}
}

func TestFlowStateNilSafety(t *testing.T) {
	var nilState *flowState

	// All operations should be safe on nil state
	if _, ok := nilState.get("key"); ok {
		t.Error("nil state get should return false")
	}
	if _, ok := nilState.resolve("key"); ok {
		t.Error("nil state resolve should return false")
	}
	nilState.set("key", "value")                   // should not panic
	nilState.merge(map[string]any{"key": "value"}) // should not panic
	nilState.markEntryChecked()                    // should not panic
	if nilState.hasCheckedEntry() {
		t.Error("nil state hasCheckedEntry should return false")
	}
}

func TestFlowStateMerge(t *testing.T) {
	state := newFlowState(map[string]any{
		"existing": "value",
	})

	state.merge(map[string]any{
		"new":      "value2",
		"existing": "overwritten",
	})

	if val, ok := state.get("new"); !ok || val != "value2" {
		t.Errorf("expected new key to be added")
	}
	if val, ok := state.get("existing"); !ok || val != "overwritten" {
		t.Errorf("expected existing key to be overwritten")
	}
}

func TestFlowStateEntryCheck(t *testing.T) {
	state := newFlowState(nil)

	if state.hasCheckedEntry() {
		t.Error("new state should not have entry checked")
	}

	state.markEntryChecked()

	if !state.hasCheckedEntry() {
		t.Error("state should have entry checked after marking")
	}
}

// ============================================================================
// Interpolation Tests
// ============================================================================

func TestInterpolateString(t *testing.T) {
	state := newFlowState(map[string]any{
		"name":    "World",
		"count":   42,
		"flag":    true,
		"nested":  map[string]any{"value": "deep"},
		"items":   []any{"a", "b"},
		"wrapped": map[string]any{"value": "unwrapped"},
	})

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", "Hello"},
		{"Hello ${name}", "Hello World"},
		{"Count: ${count}", "Count: 42"},
		{"Flag: ${flag}", "Flag: true"},
		{"Nested: ${nested.value}", "Nested: deep"},
		{"Item: ${items.0}", "Item: a"},
		{"{{name}} works too", "World works too"},
		{"Mixed ${name} and {{count}}", "Mixed World and 42"},
		{"Unknown ${unknown} removed", "Unknown  removed"},
		{"Wrapped: ${wrapped}", "Wrapped: unwrapped"}, // Tests extracted data unwrapping
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := interpolateString(tc.input, state)
			if result != tc.expected {
				t.Errorf("interpolateString(%q): expected %q, got %q", tc.input, tc.expected, result)
			}
		})
	}
}

func TestInterpolateValue(t *testing.T) {
	state := newFlowState(map[string]any{
		"var": "replaced",
	})

	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{"string", "Hello ${var}", "Hello replaced"},
		{"int passthrough", 42, 42},
		{"bool passthrough", true, true},
		{"nil passthrough", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := interpolateValue(tc.input, state)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// ============================================================================
// Type Coercion Tests (coerceArray only - others in type_coercion_test.go)
// ============================================================================

func TestCoerceArray(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []any
	}{
		{"[]any", []any{"a", "b"}, []any{"a", "b"}},
		{"[]string", []string{"x", "y"}, []any{"x", "y"}},
		{"json string", `["1", "2"]`, []any{"1", "2"}},
		{"empty string", "", nil},
		{"invalid json", "not json", nil},
		{"nil", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := coerceArray(tc.input)
			if tc.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tc.expected) {
				t.Errorf("expected len %d, got %d", len(tc.expected), len(result))
			}
		})
	}
}

// ============================================================================
// CompareValues Tests
// ============================================================================

func TestCompareValues(t *testing.T) {
	tests := []struct {
		current  any
		expected any
		op       string
		result   bool
	}{
		// Equality
		{42, 42, "==", true},
		{42, "42", "==", true}, // String coercion
		{"hello", "hello", "==", true},
		{true, true, "==", true},
		{42, 43, "==", false},

		// Inequality
		{42, 43, "!=", true},
		{"a", "b", "!=", true},
		{42, 42, "!=", false},

		// Greater than
		{10, 5, ">", true},
		{5, 10, ">", false},
		{5, 5, ">", false},

		// Greater or equal
		{10, 5, ">=", true},
		{5, 5, ">=", true},
		{5, 10, ">=", false},

		// Less than
		{5, 10, "<", true},
		{10, 5, "<", false},
		{5, 5, "<", false},

		// Less or equal
		{5, 10, "<=", true},
		{5, 5, "<=", true},
		{10, 5, "<=", false},

		// Aliases
		{42, 42, "eq", true},
		{42, 43, "ne", true},
		{10, 5, "gt", true},
		{10, 5, "gte", true},
		{5, 10, "lt", true},
		{5, 10, "lte", true},
	}

	for _, tc := range tests {
		name := tc.op
		t.Run(name, func(t *testing.T) {
			result := compareValues(tc.current, tc.expected, tc.op)
			if result != tc.result {
				t.Errorf("compareValues(%v, %v, %q): expected %v, got %v",
					tc.current, tc.expected, tc.op, tc.result, result)
			}
		})
	}
}

// ============================================================================
// Helper Value Extraction Tests (stringValue/intValue/boolValue in type_coercion_test.go)
// ============================================================================

func TestStringValueNilMap(t *testing.T) {
	// Additional nil-safety test not covered in type_coercion_test.go
	if v := stringValue(nil, "any"); v != "" {
		t.Errorf("expected empty string for nil map, got %q", v)
	}
}

// ============================================================================
// Graph Helper Tests
// ============================================================================

func TestFirstPresent(t *testing.T) {
	m := map[string]any{
		"second": "found",
		"third":  "also",
	}

	if v := firstPresent(m, "first", "second", "third"); v != "found" {
		t.Errorf("expected 'found', got %v", v)
	}
	if v := firstPresent(m, "missing", "also_missing"); v != nil {
		t.Errorf("expected nil, got %v", v)
	}
	if v := firstPresent(nil); v != nil {
		t.Errorf("expected nil for nil map, got %v", v)
	}
}

func TestNormalizeVariableValue(t *testing.T) {
	// Test boolean normalization (handles string bool parsing)
	if v := normalizeVariableValue("true", "boolean"); v != true {
		t.Errorf("bool true: expected true, got %v", v)
	}
	if v := normalizeVariableValue("false", "bool"); v != false {
		t.Errorf("bool false: expected false, got %v", v)
	}

	// Test number normalization - expects already-parsed numeric values (like from JSON)
	// Note: toFloat only handles int, int64, float32, float64 - not strings
	if v, ok := normalizeVariableValue(float64(42), "number").(float64); !ok || v != 42.0 {
		t.Errorf("number from float64: expected float64(42), got %T(%v)", v, v)
	}
	if v, ok := normalizeVariableValue(float64(3.14), "float").(float64); !ok || v != 3.14 {
		t.Errorf("float from float64: expected float64(3.14), got %T(%v)", v, v)
	}
	if v, ok := normalizeVariableValue(float64(100), "int").(int); !ok || v != 100 {
		t.Errorf("int type from float64: expected int(100), got %T(%v)", v, v)
	}
	if v, ok := normalizeVariableValue(42, "int").(int); !ok || v != 42 {
		t.Errorf("int type from int: expected int(42), got %T(%v)", v, v)
	}

	// Test JSON normalization (handles string JSON parsing)
	if v, ok := normalizeVariableValue(`["a","b"]`, "json").([]any); !ok || len(v) != 2 {
		t.Errorf("json array: expected []any with 2 elements, got %T", v)
	}
	if v, ok := normalizeVariableValue(`{"key":"value"}`, "json").(map[string]any); !ok || v["key"] != "value" {
		t.Errorf("json object: expected map with key=value, got %T", v)
	}

	// Test passthrough for unknown types
	if v := normalizeVariableValue("hello", "string"); v != "hello" {
		t.Errorf("passthrough: expected hello, got %v", v)
	}
	if v := normalizeVariableValue(42, "unknown"); v != 42 {
		t.Errorf("passthrough unknown: expected 42, got %v", v)
	}
	// String input for number type without toFloat support - returns original value
	if v := normalizeVariableValue("42", "number"); v != "42" {
		t.Errorf("string number passthrough: expected '42', got %v", v)
	}
}

func TestMinMaxInt(t *testing.T) {
	if minInt(3, 5) != 3 {
		t.Error("minInt(3,5) should be 3")
	}
	if minInt(5, 3) != 3 {
		t.Error("minInt(5,3) should be 3")
	}
	if maxInt(3, 5) != 5 {
		t.Error("maxInt(3,5) should be 5")
	}
	if maxInt(5, 3) != 5 {
		t.Error("maxInt(5,3) should be 5")
	}
}
