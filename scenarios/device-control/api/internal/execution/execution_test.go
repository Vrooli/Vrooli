package execution

import (
	"encoding/json"
	"testing"
)

func TestFlowWireContractUsesSnakeCase(t *testing.T) {
	encoded, err := json.Marshal(Flow{ID: "flow", Steps: []Step{{ID: "step", RequiredCapabilities: []string{"input"}, TimeoutMS: 10}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"id"`, `"steps"`, `"required_capabilities"`, `"timeout_ms"`} {
		if !contains(string(encoded), field) {
			t.Fatalf("missing %s in %s", field, encoded)
		}
	}
}

func contains(value, want string) bool {
	for i := 0; i+len(want) <= len(value); i++ {
		if value[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
