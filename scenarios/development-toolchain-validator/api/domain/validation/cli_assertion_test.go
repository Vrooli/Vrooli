// [REQ:REQ-P0-008a] JSON Parsing and Assertion Evaluation
package validation

import (
	"context"
	"development-toolchain-validator/domain/expectation"
	"testing"
)

func TestExtractJSONPath(t *testing.T) {
	// [REQ:REQ-P0-008a] JSON parsing robust
	data := map[string]interface{}{
		"name":  "test",
		"count": float64(42),
		"items": []interface{}{
			map[string]interface{}{"id": float64(1)},
			map[string]interface{}{"id": float64(2)},
		},
		"nested": map[string]interface{}{
			"deep": map[string]interface{}{
				"value": "found",
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "root path",
			path:     "$",
			expected: data,
			wantErr:  false,
		},
		{
			name:     "simple field",
			path:     "$.name",
			expected: "test",
			wantErr:  false,
		},
		{
			name:     "numeric field",
			path:     "$.count",
			expected: float64(42),
			wantErr:  false,
		},
		{
			name:     "array access",
			path:     "$.items[0]",
			expected: map[string]interface{}{"id": float64(1)},
			wantErr:  false,
		},
		{
			name:     "array element field",
			path:     "$.items[1].id",
			expected: float64(2),
			wantErr:  false,
		},
		{
			name:     "deeply nested",
			path:     "$.nested.deep.value",
			expected: "found",
			wantErr:  false,
		},
		{
			name:     "array wildcard",
			path:     "$.items[*]",
			expected: data["items"],
			wantErr:  false,
		},
		{
			name:    "missing field",
			path:    "$.missing",
			wantErr: true,
		},
		{
			name:    "index out of bounds",
			path:    "$.items[10]",
			wantErr: true,
		},
		{
			name:    "field on array",
			path:    "$.items.name",
			wantErr: true,
		},
		{
			name:    "index on object",
			path:    "$.nested[0]",
			wantErr: true,
		},
		{
			name:    "invalid path format",
			path:    "name", // missing $
			wantErr: true,
		},
		{
			name:    "unclosed bracket",
			path:    "$.items[0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractJSONPath(data, tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !deepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluateAssertion_Equality(t *testing.T) {
	// [REQ:REQ-P0-008a] All assertion types supported - eq/neq
	tests := []struct {
		name     string
		actual   interface{}
		op       expectation.AssertionOperator
		expected interface{}
		want     bool
		wantErr  bool
	}{
		// OpEq tests
		{"eq string match", "hello", expectation.OpEq, "hello", true, false},
		{"eq string no match", "hello", expectation.OpEq, "world", false, false},
		{"eq number match", float64(42), expectation.OpEq, float64(42), true, false},
		{"eq number no match", float64(42), expectation.OpEq, float64(43), false, false},
		{"eq bool match", true, expectation.OpEq, true, true, false},
		{"eq bool no match", true, expectation.OpEq, false, false, false},
		{"eq nil match", nil, expectation.OpEq, nil, true, false},
		{"eq int to float", float64(5), expectation.OpEq, 5, true, false},

		// OpNeq tests
		{"neq string different", "hello", expectation.OpNeq, "world", true, false},
		{"neq string same", "hello", expectation.OpNeq, "hello", false, false},
		{"neq number different", float64(42), expectation.OpNeq, float64(43), true, false},
		{"neq number same", float64(42), expectation.OpNeq, float64(42), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateAssertion(tt.actual, tt.op, tt.expected)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.want {
				t.Errorf("expected %v, got %v", tt.want, result)
			}
		})
	}
}

func TestEvaluateAssertion_Comparison(t *testing.T) {
	// [REQ:REQ-P0-008a] All assertion types supported - gt/gte/lt/lte
	tests := []struct {
		name     string
		actual   interface{}
		op       expectation.AssertionOperator
		expected interface{}
		want     bool
		wantErr  bool
	}{
		// OpGt tests
		{"gt number greater", float64(10), expectation.OpGt, float64(5), true, false},
		{"gt number equal", float64(10), expectation.OpGt, float64(10), false, false},
		{"gt number less", float64(5), expectation.OpGt, float64(10), false, false},

		// OpGte tests
		{"gte number greater", float64(10), expectation.OpGte, float64(5), true, false},
		{"gte number equal", float64(10), expectation.OpGte, float64(10), true, false},
		{"gte number less", float64(5), expectation.OpGte, float64(10), false, false},

		// OpLt tests
		{"lt number less", float64(5), expectation.OpLt, float64(10), true, false},
		{"lt number equal", float64(10), expectation.OpLt, float64(10), false, false},
		{"lt number greater", float64(10), expectation.OpLt, float64(5), false, false},

		// OpLte tests
		{"lte number less", float64(5), expectation.OpLte, float64(10), true, false},
		{"lte number equal", float64(10), expectation.OpLte, float64(10), true, false},
		{"lte number greater", float64(10), expectation.OpLte, float64(5), false, false},

		// String to number conversion
		{"gt string number", "15", expectation.OpGt, float64(10), true, false},

		// Error cases
		{"gt string non-numeric", "hello", expectation.OpGt, float64(10), false, true},
		{"gt non-numeric expected", float64(10), expectation.OpGt, "hello", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateAssertion(tt.actual, tt.op, tt.expected)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.want {
				t.Errorf("expected %v, got %v", tt.want, result)
			}
		})
	}
}

func TestEvaluateAssertion_Exists(t *testing.T) {
	// [REQ:REQ-P0-008a] All assertion types supported - exists
	tests := []struct {
		name   string
		actual interface{}
		want   bool
	}{
		{"exists string", "hello", true},
		{"exists number", float64(0), true},
		{"exists empty string", "", true},
		{"exists nil", nil, false},
		{"exists array", []interface{}{1, 2}, true},
		{"exists object", map[string]interface{}{"a": 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateAssertion(tt.actual, expectation.OpExists, nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.want {
				t.Errorf("expected %v, got %v", tt.want, result)
			}
		})
	}
}

func TestEvaluateAssertion_Contains(t *testing.T) {
	// [REQ:REQ-P0-008a] All assertion types supported - contains
	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
		want     bool
		wantErr  bool
	}{
		// String contains
		{"string contains substring", "hello world", "world", true, false},
		{"string no contain", "hello world", "foo", false, false},
		{"string contains full", "hello", "hello", true, false},

		// Array contains
		{"array contains element", []interface{}{"a", "b", "c"}, "b", true, false},
		{"array no contain", []interface{}{"a", "b", "c"}, "d", false, false},
		{"array contains number", []interface{}{float64(1), float64(2)}, float64(2), true, false},

		// Error cases
		{"number contains", float64(123), "2", false, true},
		{"string wrong expected type", "hello", float64(1), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateAssertion(tt.actual, expectation.OpContains, tt.expected)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.want {
				t.Errorf("expected %v, got %v", tt.want, result)
			}
		})
	}
}

func TestEvaluateAssertion_Matches(t *testing.T) {
	// [REQ:REQ-P0-008a] All assertion types supported - matches
	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
		want     bool
		wantErr  bool
	}{
		{"matches simple", "hello123", `\d+`, true, false},
		{"matches full", "hello", `^hello$`, true, false},
		{"matches no match", "hello", `\d+`, false, false},
		{"matches email-like", "test@example.com", `.*@.*\.com`, true, false},

		// Error cases
		{"matches non-string actual", float64(123), `\d+`, false, true},
		{"matches non-string pattern", "hello", float64(123), false, true},
		{"matches invalid regex", "hello", `[invalid`, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateAssertion(tt.actual, expectation.OpMatches, tt.expected)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.want {
				t.Errorf("expected %v, got %v", tt.want, result)
			}
		})
	}
}

func TestEvaluateAssertion_Between(t *testing.T) {
	// [REQ:REQ-P0-008a] All assertion types supported - between
	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
		want     bool
		wantErr  bool
	}{
		{"between in range", float64(5), []interface{}{float64(1), float64(10)}, true, false},
		{"between at min", float64(1), []interface{}{float64(1), float64(10)}, true, false},
		{"between at max", float64(10), []interface{}{float64(1), float64(10)}, true, false},
		{"between below", float64(0), []interface{}{float64(1), float64(10)}, false, false},
		{"between above", float64(11), []interface{}{float64(1), float64(10)}, false, false},

		// Error cases
		{"between non-numeric actual", "five", []interface{}{float64(1), float64(10)}, false, true},
		{"between non-array expected", float64(5), float64(10), false, true},
		{"between wrong array length", float64(5), []interface{}{float64(1)}, false, true},
		{"between non-numeric min", float64(5), []interface{}{"one", float64(10)}, false, true},
		{"between non-numeric max", float64(5), []interface{}{float64(1), "ten"}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateAssertion(tt.actual, expectation.OpBetween, tt.expected)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.want {
				t.Errorf("expected %v, got %v", tt.want, result)
			}
		})
	}
}

func TestEvaluateAssertion_InvalidOperator(t *testing.T) {
	_, err := evaluateAssertion("test", expectation.AssertionOperator("invalid"), "test")
	if err == nil {
		t.Errorf("expected error for invalid operator")
	}
}

func TestCLIExecutor_IntegrationWithJSONPath(t *testing.T) {
	// [REQ:REQ-P0-008a] Tool integration works
	executor := NewCLIExecutor("/tmp")
	ctx := context.Background()

	tests := []struct {
		name       string
		command    string
		jsonPath   string
		operator   expectation.AssertionOperator
		expected   interface{}
		wantStatus ValidationStatus
	}{
		{
			name:       "extract nested field",
			command:    `echo '{"data": {"count": 42}}'`,
			jsonPath:   "$.data.count",
			operator:   expectation.OpEq,
			expected:   float64(42),
			wantStatus: StatusPassed,
		},
		{
			name:       "check array length via exists",
			command:    `echo '{"items": [1, 2, 3]}'`,
			jsonPath:   "$.items",
			operator:   expectation.OpExists,
			expected:   nil,
			wantStatus: StatusPassed,
		},
		{
			name:       "compare with gt",
			command:    `echo '{"score": 85}'`,
			jsonPath:   "$.score",
			operator:   expectation.OpGt,
			expected:   float64(80),
			wantStatus: StatusPassed,
		},
		{
			name:       "check string contains",
			command:    `echo '{"message": "Operation successful"}'`,
			jsonPath:   "$.message",
			operator:   expectation.OpContains,
			expected:   "successful",
			wantStatus: StatusPassed,
		},
		{
			name:       "range check",
			command:    `echo '{"percentage": 75}'`,
			jsonPath:   "$.percentage",
			operator:   expectation.OpBetween,
			expected:   []interface{}{float64(0), float64(100)},
			wantStatus: StatusPassed,
		},
		{
			name:       "regex match",
			command:    `echo '{"version": "v1.2.3"}'`,
			jsonPath:   "$.version",
			operator:   expectation.OpMatches,
			expected:   `^v\d+\.\d+\.\d+$`,
			wantStatus: StatusPassed,
		},
		{
			name:       "failed assertion",
			command:    `echo '{"status": "error"}'`,
			jsonPath:   "$.status",
			operator:   expectation.OpEq,
			expected:   "ok",
			wantStatus: StatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &expectation.CLIAssertion{
				ID:            "integration-" + tt.name,
				Command:       tt.command,
				JSONPath:      tt.jsonPath,
				Operator:      tt.operator,
				ExpectedValue: tt.expected,
			}

			result := executor.ValidateAssertion(ctx, assertion)

			if result.Status != tt.wantStatus {
				t.Errorf("expected status %s, got %s: %s", tt.wantStatus, result.Status, result.Message)
			}
		})
	}
}

func TestCountAssertionResults(t *testing.T) {
	results := []*AssertionResult{
		{Status: StatusPassed},
		{Status: StatusPassed},
		{Status: StatusFailed},
		{Status: StatusSkipped},
		{Status: StatusError},
		{Status: StatusError},
	}

	pass, fail, skip, errCount := CountAssertionResults(results)

	if pass != 2 {
		t.Errorf("expected 2 passes, got %d", pass)
	}
	if fail != 1 {
		t.Errorf("expected 1 fail, got %d", fail)
	}
	if skip != 1 {
		t.Errorf("expected 1 skip, got %d", skip)
	}
	if errCount != 2 {
		t.Errorf("expected 2 errors, got %d", errCount)
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		want   float64
		wantOK bool
	}{
		{"float64", float64(42.5), 42.5, true},
		{"int", 42, 42, true},
		{"int64", int64(42), 42, true},
		{"string number", "42.5", 42.5, true},
		{"string invalid", "not a number", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toFloat64(tt.input)
			if ok != tt.wantOK {
				t.Errorf("toFloat64() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("toFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}
