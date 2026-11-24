package tasks

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"program-runtime/internal/contracts"
)

func TestSpecCarriesOutputSchemaIntoRuntimeDeclaration(t *testing.T) {
	declaration := []byte(`{"name":"demo.child","output_schema":"child.output.schema.json"}`)
	schema := []byte(`{"type":"object","required":["status"]}`)
	contract := contracts.Contract{
		ID: "demo.child", Name: "child", Scenario: "demo",
		Declaration: declaration, OutputSchemaSource: schema,
	}

	spec, err := Spec(contract, nil)
	require.NoError(t, err)
	encoded, ok := spec["declaration"].(json.RawMessage)
	require.True(t, ok)
	var runtimeDeclaration map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &runtimeDeclaration))
	require.JSONEq(t, string(schema), string(runtimeDeclaration["_runtime_output_schema"]))
	require.Equal(t, declaration, contract.Declaration)
}
