package cmdutil

import "testing"

func TestTierToNumber(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"local", 1},
		{"1", 1},
		{"desktop", 2},
		{"2", 2},
		{"mobile", 3},
		{"3", 3},
		{"saas", 4},
		{"4", 4},
		{"enterprise", 5},
		{"5", 5},
		// Aliases
		{"ios", 3},
		{"android", 3},
		{"cloud", 4},
		{"web", 4},
		{"on-prem", 5},
		// Case insensitivity
		{"LOCAL", 1},
		{"Desktop", 2},
		{"SAAS", 4},
		// Empty defaults to 2
		{"", 2},
		// Unknown defaults to 3
		{"unknown", 3},
		{"foobar", 3},
		// Whitespace trimming
		{"  desktop  ", 2},
		{" local ", 1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := TierToNumber(tt.input)
			if got != tt.want {
				t.Errorf("TierToNumber(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestEnsureMap_NilMap(t *testing.T) {
	result := EnsureMap(nil, "key")
	if result == nil {
		t.Fatal("expected non-nil map from nil input")
	}
}

func TestEnsureMap_ExistingMapValue(t *testing.T) {
	inner := map[string]interface{}{"existing": true}
	obj := map[string]interface{}{"nested": inner}

	result := EnsureMap(obj, "nested")
	if result == nil {
		t.Fatal("expected non-nil map")
	}
	if _, ok := result["existing"]; !ok {
		t.Fatal("expected existing key in returned map")
	}
}

func TestEnsureMap_ExistingNonMapValue(t *testing.T) {
	obj := map[string]interface{}{"key": "string-value"}

	result := EnsureMap(obj, "key")
	if result == nil {
		t.Fatal("expected non-nil map")
	}
	// The old non-map value should be replaced with a new map
	if _, ok := obj["key"].(map[string]interface{}); !ok {
		t.Fatal("expected key to now hold a map")
	}
}

func TestEnsureMap_MissingKey(t *testing.T) {
	obj := map[string]interface{}{"other": 42}

	result := EnsureMap(obj, "newkey")
	if result == nil {
		t.Fatal("expected non-nil map")
	}
	// The new map should be stored in obj
	if _, ok := obj["newkey"].(map[string]interface{}); !ok {
		t.Fatal("expected newkey to be set in obj")
	}
}

func TestResolveFormat(t *testing.T) {
	tests := []struct {
		name   string
		local  string
		global string
		want   string
	}{
		{"non-empty local overrides global", "table", "json", "table"},
		{"empty local uses global", "", "json", "json"},
		{"whitespace-only local uses global", "   ", "json", "json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global format for this test
			SetGlobalFormat(tt.global)
			got := ResolveFormat(tt.local)
			if got != tt.want {
				t.Errorf("ResolveFormat(%q) = %q, want %q (global=%q)", tt.local, got, tt.want, tt.global)
			}
		})
	}
}

func TestSetGlobalFormat_AndGlobalFormat(t *testing.T) {
	SetGlobalFormat("table")
	if got := GlobalFormat(); got != "table" {
		t.Errorf("GlobalFormat() = %q, want %q", got, "table")
	}

	SetGlobalFormat("JSON")
	if got := GlobalFormat(); got != "json" {
		t.Errorf("GlobalFormat() after uppercase = %q, want %q", got, "json")
	}

	// Empty string should not change the format
	SetGlobalFormat("")
	if got := GlobalFormat(); got != "json" {
		t.Errorf("GlobalFormat() after empty set = %q, want %q", got, "json")
	}

	// Whitespace-only should not change the format
	SetGlobalFormat("   ")
	if got := GlobalFormat(); got != "json" {
		t.Errorf("GlobalFormat() after whitespace set = %q, want %q", got, "json")
	}
}
