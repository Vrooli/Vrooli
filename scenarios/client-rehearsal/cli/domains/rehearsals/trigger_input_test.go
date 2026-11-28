package rehearsals

import (
	"strings"
	"testing"
)

func TestTriggerInputFromJSON(t *testing.T) {
	got, err := triggerInputFromJSON(`{"body":{"id":"synthetic-1"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.AsMap()["body"] == nil {
		t.Fatalf("parsed trigger input = %#v", got)
	}
	if omitted, err := triggerInputFromJSON("  "); err != nil || omitted != nil {
		t.Fatalf("omitted input = %#v, err = %v", omitted, err)
	}
	for _, invalid := range []string{"[]", "null", "not-json"} {
		if _, err := triggerInputFromJSON(invalid); err == nil {
			t.Errorf("expected %q to be rejected", invalid)
		}
	}
	if _, err := triggerInputFromJSON(strings.Repeat("x", maxTriggerInputJSONBytes+1)); err == nil {
		t.Fatal("expected oversized trigger input to be rejected")
	}
}
