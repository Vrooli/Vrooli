package capabilities

import (
	"context"
	"encoding/json"
	"testing"
)

func TestDescribeDoesNotDuplicateManifestDependencies(t *testing.T) {
	data, err := NewRegistry().Describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Definitions []Definition `json:"definitions"`
		States      []State      `json:"states"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Definitions) != 0 || len(payload.States) != 0 {
		t.Fatalf("descriptor duplicates manifest dependencies: %#v", payload)
	}
}

func TestCheckDoesNotClaimAvailability(t *testing.T) {
	status, message := NewRegistry().Check(context.Background(), "vrooli-events")
	if status != StatusUnknown || message == "" {
		t.Fatalf("Check() = (%q, %q), want unknown status and explanation", status, message)
	}
}
