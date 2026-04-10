package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/state"
)

func TestBuildSubflowState_ParameterOverride(t *testing.T) {
	// Create parent state with params
	parentState := state.New(
		map[string]any{"storeKey": "storeValue"},
		map[string]any{"parentParam": "parentValue", "sharedParam": "fromParent"},
		map[string]any{"envKey": "envValue"},
	)

	// Build child state with override params
	subflowParams := map[string]any{"childParam": "childValue", "sharedParam": "fromChild"}
	childState := buildSubflowState(parentState, subflowParams)

	// Verify child has ONLY subflowParams, not parent params
	childParams := childState.GetNamespace("params")
	assert.Equal(t, "childValue", childParams["childParam"])
	assert.Equal(t, "fromChild", childParams["sharedParam"])
	assert.Nil(t, childParams["parentParam"], "child should not inherit parent params when override specified")
}

func TestBuildSubflowState_ParameterInheritance(t *testing.T) {
	// Create parent state with params
	parentState := state.New(
		map[string]any{"storeKey": "storeValue"},
		map[string]any{"parentParam": "parentValue", "anotherParam": "anotherValue"},
		map[string]any{"envKey": "envValue"},
	)

	// Build child state with NO override params (nil)
	childState := buildSubflowState(parentState, nil)

	// Verify child inherits parent params when none specified
	childParams := childState.GetNamespace("params")
	assert.Equal(t, "parentValue", childParams["parentParam"])
	assert.Equal(t, "anotherValue", childParams["anotherParam"])
}

func TestBuildSubflowState_ParameterInheritanceEmptyMap(t *testing.T) {
	// Create parent state with params
	parentState := state.New(
		map[string]any{"storeKey": "storeValue"},
		map[string]any{"parentParam": "parentValue"},
		map[string]any{"envKey": "envValue"},
	)

	// Build child state with empty map (should still inherit)
	childState := buildSubflowState(parentState, map[string]any{})

	// Verify child inherits parent params when empty map specified
	childParams := childState.GetNamespace("params")
	assert.Equal(t, "parentValue", childParams["parentParam"])
}

func TestBuildSubflowState_StoreIsolation(t *testing.T) {
	// Create parent state with store
	parentState := state.New(
		map[string]any{"counter": 10, "name": "parent"},
		map[string]any{"param": "value"},
		map[string]any{"env": "prod"},
	)

	// Build child state
	childState := buildSubflowState(parentState, nil)

	// Verify child has copy of parent store
	childStore := childState.GetNamespace("store")
	assert.Equal(t, 10, childStore["counter"])
	assert.Equal(t, "parent", childStore["name"])

	// Modify child's store
	childState.Set("counter", 20)
	childState.Set("newKey", "newValue")

	// Verify parent store is unchanged
	parentStore := parentState.GetNamespace("store")
	assert.Equal(t, 10, parentStore["counter"], "parent store should not be modified by child")
	assert.Nil(t, parentStore["newKey"], "parent store should not have child's new keys")

	// Now merge child store back to parent
	mergeSubflowStore(parentState, childState)

	// Verify parent store now has child's changes
	updatedParentStore := parentState.GetNamespace("store")
	assert.Equal(t, 20, updatedParentStore["counter"])
	assert.Equal(t, "newValue", updatedParentStore["newKey"])
}

func TestBuildSubflowState_EnvInheritance(t *testing.T) {
	// Create parent state with env
	parentState := state.New(
		map[string]any{"storeKey": "storeValue"},
		map[string]any{"paramKey": "paramValue"},
		map[string]any{"API_KEY": "secret123", "DEBUG": "true"},
	)

	// Build child state
	childState := buildSubflowState(parentState, nil)

	// Verify child inherits env unchanged
	childEnv := childState.GetNamespace("env")
	assert.Equal(t, "secret123", childEnv["API_KEY"])
	assert.Equal(t, "true", childEnv["DEBUG"])
}

func TestSubflowParameterInterpolation(t *testing.T) {
	// Create parent state with params that will be passed to child
	parentState := state.New(
		map[string]any{},
		map[string]any{"baseUrl": "https://example.com"},
		map[string]any{},
	)

	// Build child state (inheriting params)
	childState := buildSubflowState(parentState, nil)

	// Verify child can resolve ${@params/baseUrl}
	interp := state.NewInterpolator(childState)
	result := interp.InterpolateString("Navigate to ${@params/baseUrl}/login")
	assert.Equal(t, "Navigate to https://example.com/login", result)
}

func TestSubflowParameterInterpolation_WithOverride(t *testing.T) {
	// Create parent state with params
	parentState := state.New(
		map[string]any{},
		map[string]any{"baseUrl": "https://parent.com"},
		map[string]any{},
	)

	// Build child state with override params
	subflowParams := map[string]any{"baseUrl": "https://child.com", "timeout": 5000}
	childState := buildSubflowState(parentState, subflowParams)

	// Verify child resolves with overridden values
	interp := state.NewInterpolator(childState)

	result := interp.InterpolateString("${@params/baseUrl}")
	assert.Equal(t, "https://child.com", result)

	timeoutResult := interp.InterpolateString("Timeout: ${@params/timeout}ms")
	assert.Equal(t, "Timeout: 5000ms", timeoutResult)
}

func TestSubflowNestedParameters(t *testing.T) {
	// Create parent state with nested params
	parentState := state.New(
		map[string]any{},
		map[string]any{},
		map[string]any{},
	)

	// Build child state with complex nested params
	subflowParams := map[string]any{
		"config": map[string]any{
			"api": map[string]any{
				"endpoint": "https://api.example.com",
				"version":  "v2",
			},
			"timeout": 3000,
		},
		"tags": []any{"production", "critical"},
	}
	childState := buildSubflowState(parentState, subflowParams)

	// Verify nested params are accessible
	childParams := childState.GetNamespace("params")
	require.NotNil(t, childParams["config"])

	config, ok := childParams["config"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 3000, config["timeout"])

	api, ok := config["api"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://api.example.com", api["endpoint"])
	assert.Equal(t, "v2", api["version"])

	tags, ok := childParams["tags"].([]any)
	require.True(t, ok)
	assert.Len(t, tags, 2)
	assert.Equal(t, "production", tags[0])
	assert.Equal(t, "critical", tags[1])
}

func TestSubflowEmptyParameters(t *testing.T) {
	tests := []struct {
		name          string
		parentParams  map[string]any
		subflowParams map[string]any
		wantInherit   bool
	}{
		{
			name:          "nil parent params, nil subflow params",
			parentParams:  nil,
			subflowParams: nil,
			wantInherit:   true, // inherits empty
		},
		{
			name:          "nil parent params, empty subflow params",
			parentParams:  nil,
			subflowParams: map[string]any{},
			wantInherit:   true, // inherits empty
		},
		{
			name:          "empty parent params, nil subflow params",
			parentParams:  map[string]any{},
			subflowParams: nil,
			wantInherit:   true, // inherits empty
		},
		{
			name:          "parent has params, nil subflow params inherits",
			parentParams:  map[string]any{"key": "value"},
			subflowParams: nil,
			wantInherit:   true,
		},
		{
			name:          "parent has params, empty subflow params inherits",
			parentParams:  map[string]any{"key": "value"},
			subflowParams: map[string]any{},
			wantInherit:   true,
		},
		{
			name:          "parent has params, subflow has params overrides",
			parentParams:  map[string]any{"key": "value"},
			subflowParams: map[string]any{"other": "val"},
			wantInherit:   false, // override, not inherit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parentState := state.New(nil, tt.parentParams, nil)
			childState := buildSubflowState(parentState, tt.subflowParams)
			childParams := childState.GetNamespace("params")

			if tt.wantInherit && tt.parentParams != nil {
				for k, v := range tt.parentParams {
					assert.Equal(t, v, childParams[k], "expected inherited param %s", k)
				}
			}
			if !tt.wantInherit && tt.subflowParams != nil {
				for k, v := range tt.subflowParams {
					assert.Equal(t, v, childParams[k], "expected subflow param %s", k)
				}
				// Ensure parent params are NOT present
				if tt.parentParams != nil {
					for k := range tt.parentParams {
						if _, exists := tt.subflowParams[k]; !exists {
							assert.Nil(t, childParams[k], "expected parent param %s to NOT be inherited", k)
						}
					}
				}
			}
		})
	}
}

func TestBuildSubflowState_NilParentState(t *testing.T) {
	// Build child state from nil parent
	childState := buildSubflowState(nil, map[string]any{"key": "value"})

	// Should return a valid but empty state (from NewFromStore(nil))
	require.NotNil(t, childState)

	// The returned state should have empty namespaces since parent was nil
	childStore := childState.GetNamespace("store")
	assert.Empty(t, childStore)

	childParams := childState.GetNamespace("params")
	assert.Empty(t, childParams)

	childEnv := childState.GetNamespace("env")
	assert.Empty(t, childEnv)
}

func TestMergeSubflowStore_NilStates(t *testing.T) {
	// Test that mergeSubflowStore handles nil states gracefully
	parentState := state.New(
		map[string]any{"key": "value"},
		nil,
		nil,
	)

	// Should not panic with nil child
	mergeSubflowStore(parentState, nil)
	assert.Equal(t, "value", parentState.GetNamespace("store")["key"])

	// Should not panic with nil parent
	childState := state.New(map[string]any{"other": "val"}, nil, nil)
	mergeSubflowStore(nil, childState) // Should not panic
}
