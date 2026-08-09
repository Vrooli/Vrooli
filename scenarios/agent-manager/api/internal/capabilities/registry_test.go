package capabilities

import (
	"context"
	"encoding/json"
	"testing"
)

func TestDescribeIncludesDeclaredScenarioDependenciesAndRecoveryActions(t *testing.T) {
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
	if len(payload.Definitions) != 2 || len(payload.States) != 2 {
		t.Fatalf("descriptor counts = %d definitions, %d states; want 2 and 2", len(payload.Definitions), len(payload.States))
	}
	for _, slug := range []string{"workspace-sandbox", "vrooli-events"} {
		found := false
		for _, definition := range payload.Definitions {
			if definition.DependencySlug == slug {
				found = true
				if definition.Description == "" || definition.ActionKind == "" || definition.OperatorCommand == "" {
					t.Fatalf("%s definition is incomplete: %#v", slug, definition)
				}
			}
		}
		if !found {
			t.Fatalf("missing dependency definition %q", slug)
		}
	}
}

func TestCheckDoesNotClaimAvailability(t *testing.T) {
	status, message := NewRegistry().Check(context.Background(), "vrooli-events")
	if status != StatusUnknown || message == "" {
		t.Fatalf("Check() = (%q, %q), want unknown status and explanation", status, message)
	}
}
