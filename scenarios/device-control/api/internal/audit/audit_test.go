package audit

import (
	"encoding/json"
	"testing"
)

func TestRecordDoesNotExposeArtifactBytesOrPaths(t *testing.T) {
	encoded, err := json.Marshal(Record{ID: "audit", Actor: "operator", DeviceID: "phone", RedactionVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"artifact_path", "artifact_bytes"} {
		if contains(string(encoded), forbidden) {
			t.Fatalf("audit exposed %s: %s", forbidden, encoded)
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
