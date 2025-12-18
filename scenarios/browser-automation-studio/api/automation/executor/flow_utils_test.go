package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"snake_case", "snakeCase"},
		{"multi_word_string", "multiWordString"},
		{"already_camel", "alreadyCamel"},
		{"a_b_c", "aBC"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := snakeToCamel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeLoopParam(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		value         any
		expectedKey   string
		expectedValue any
	}{
		{
			name:          "loop_type enum to lowercase",
			key:           "loop_type",
			value:         "LOOP_TYPE_FOREACH",
			expectedKey:   "loopType",
			expectedValue: "foreach",
		},
		{
			name:          "loop_type repeat",
			key:           "loop_type",
			value:         "LOOP_TYPE_REPEAT",
			expectedKey:   "loopType",
			expectedValue: "repeat",
		},
		{
			name:          "loop_type while",
			key:           "loop_type",
			value:         "LOOP_TYPE_WHILE",
			expectedKey:   "loopType",
			expectedValue: "while",
		},
		{
			name:          "count to loopCount",
			key:           "count",
			value:         5,
			expectedKey:   "loopCount",
			expectedValue: 5,
		},
		{
			name:          "max_iterations to loopMaxIterations",
			key:           "max_iterations",
			value:         100,
			expectedKey:   "loopMaxIterations",
			expectedValue: 100,
		},
		{
			name:          "array_source to arraySource",
			key:           "array_source",
			value:         "${items}",
			expectedKey:   "arraySource",
			expectedValue: "${items}",
		},
		{
			name:          "item_variable to itemVariable",
			key:           "item_variable",
			value:         "item",
			expectedKey:   "itemVariable",
			expectedValue: "item",
		},
		{
			name:          "index_variable to indexVariable",
			key:           "index_variable",
			value:         "i",
			expectedKey:   "indexVariable",
			expectedValue: "i",
		},
		{
			name:          "unknown key uses snakeToCamel",
			key:           "custom_field",
			value:         "value",
			expectedKey:   "customField",
			expectedValue: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value := normalizeLoopParam(tt.key, tt.value)
			assert.Equal(t, tt.expectedKey, key)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestNormalizeLoopCondition(t *testing.T) {
	input := map[string]any{
		"type":       "LOOP_CONDITION_TYPE_VARIABLE",
		"operator":   "LOOP_CONDITION_OPERATOR_EQUALS",
		"variable":   "counter",
		"value":      10,
		"expression": "counter < 10",
	}

	result := normalizeLoopCondition(input)

	assert.Equal(t, "variable", result["conditionType"])
	assert.Equal(t, "equals", result["conditionOperator"])
	assert.Equal(t, "counter", result["conditionVariable"])
	assert.Equal(t, 10, result["conditionValue"])
	assert.Equal(t, "counter < 10", result["conditionExpression"])
}

func TestNormalizeSubflowParam(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		value         any
		expectedKey   string
		expectedValue any
	}{
		{
			name:          "workflow_id to workflowId",
			key:           "workflow_id",
			value:         "uuid-123",
			expectedKey:   "workflowId",
			expectedValue: "uuid-123",
		},
		{
			name:          "workflow_path to workflowPath",
			key:           "workflow_path",
			value:         "/path/to/workflow",
			expectedKey:   "workflowPath",
			expectedValue: "/path/to/workflow",
		},
		{
			name:          "workflow_version to workflowVersion",
			key:           "workflow_version",
			value:         2,
			expectedKey:   "workflowVersion",
			expectedValue: 2,
		},
		{
			name:          "args to parameters",
			key:           "args",
			value:         map[string]any{"key": "value"},
			expectedKey:   "parameters",
			expectedValue: map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value := normalizeSubflowParam(tt.key, tt.value)
			assert.Equal(t, tt.expectedKey, key)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestFlattenActionParams_Loop(t *testing.T) {
	// Simulate what protojson produces for a loop action
	input := map[string]any{
		"type": "ACTION_TYPE_LOOP",
		"loop": map[string]any{
			"loop_type":      "LOOP_TYPE_FOREACH",
			"array_source":   "${items}",
			"item_variable":  "item",
			"index_variable": "idx",
			"max_iterations": float64(100), // JSON numbers are float64
		},
	}

	result := flattenActionParams(input)

	assert.Equal(t, "foreach", result["loopType"])
	assert.Equal(t, "${items}", result["arraySource"])
	assert.Equal(t, "item", result["itemVariable"])
	assert.Equal(t, "idx", result["indexVariable"])
	assert.Equal(t, float64(100), result["loopMaxIterations"])
	// Should not contain type
	assert.Nil(t, result["type"])
}

func TestFlattenActionParams_Subflow(t *testing.T) {
	input := map[string]any{
		"type": "ACTION_TYPE_SUBFLOW",
		"subflow": map[string]any{
			"workflow_id":      "uuid-123",
			"workflow_version": float64(1),
			"args": map[string]any{
				"baseUrl": "https://example.com",
			},
		},
	}

	result := flattenActionParams(input)

	assert.Equal(t, "uuid-123", result["workflowId"])
	assert.Equal(t, float64(1), result["workflowVersion"])
	assert.NotNil(t, result["parameters"])
	params := result["parameters"].(map[string]any)
	assert.Equal(t, "https://example.com", params["baseUrl"])
}
