package sessions

import (
	"encoding/json"
	"testing"
)

func TestSessionWireContractIsSnakeCase(t *testing.T) {
	encoded, err := json.Marshal(Session{DeviceID: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" || !contains(string(encoded), `"device_id"`) {
		t.Fatalf("unexpected session JSON: %s", encoded)
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
