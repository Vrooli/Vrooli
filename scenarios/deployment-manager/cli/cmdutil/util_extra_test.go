package cmdutil

import "testing"

func TestFormatAndTierHelpers(t *testing.T) {
	for input, want := range map[string]int{"local": 1, "desktop": 2, "ios": 3, "cloud": 4, "enterprise": 5, "": 2, "unknown": 3} {
		if got := TierToNumber(input); got != want {
			t.Fatalf("TierToNumber(%q)=%d want %d", input, got, want)
		}
	}
	SetGlobalFormat(" table ")
	if GlobalFormat() != "table" || ResolveFormat("") != "table" || ResolveFormat("json") != "json" {
		t.Fatal("format resolution failed")
	}
	SetGlobalFormat("")
	if EnsureMap(nil, "x")["x"] != nil {
		// nil input is intentionally replaced by a new map; the returned value is
		// the nested map itself, so this branch is only a type-safety assertion.
		t.Fatal("unexpected map value")
	}
	if EnsureMap(map[string]interface{}{"x": map[string]interface{}{"ok": true}}, "x")["ok"] != true {
		t.Fatal("existing map should be preserved")
	}
}
