package schemagate

import "testing"

// TestSchemaGateSubsetSmoke pins the two behaviours the extracted package must
// preserve: a supported object schema parses, an unsupported keyword does not.
func TestSchemaGateSubsetSmoke(t *testing.T) {
	schema := `{"type":"object","properties":{"a":{"type":"string"}},"required":["a"],"additionalProperties":false}`
	if _, err := (SchemaGate{}).Parse(schema); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, err := (SchemaGate{}).Parse(`{"type":"object","unevaluatedProperties":false}`); err == nil {
		t.Fatal("Parse() accepted a construct outside the enforceable subset")
	}
}
