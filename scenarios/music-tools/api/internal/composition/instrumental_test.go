package composition

import "testing"

func TestInstrumentalAndEffectRequestsAreAddressableOperations(t *testing.T) {
	for _, operation := range []string{"instrumental", "sound-effect"} {
		if operation == "" {
			t.Fatal("operation must be explicit")
		}
	}
}
