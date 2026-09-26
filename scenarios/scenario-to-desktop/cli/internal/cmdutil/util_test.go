package cmdutil

import (
	"testing"
)

func TestMapToValues(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		got := MapToValues(nil)
		if got != nil {
			t.Errorf("MapToValues(nil) = %v, want nil", got)
		}
	})

	t.Run("empty map", func(t *testing.T) {
		got := MapToValues(map[string]string{})
		if got == nil {
			t.Fatal("MapToValues({}) = nil, want non-nil empty map")
		}
		if len(got) != 0 {
			t.Errorf("MapToValues({}) len = %d, want 0", len(got))
		}
	})

	t.Run("populated map", func(t *testing.T) {
		input := map[string]string{"a": "1", "b": "2"}
		got := MapToValues(input)
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		for k, v := range input {
			vals, ok := got[k]
			if !ok {
				t.Errorf("missing key %q", k)
				continue
			}
			if len(vals) != 1 || vals[0] != v {
				t.Errorf("got[%q] = %v, want [%q]", k, vals, v)
			}
		}
	})
}

func TestResolveFormat(t *testing.T) {
	// Reset global state
	defer SetGlobalFormat("json")

	tests := []struct {
		name         string
		globalFormat string
		local        string
		expected     string
	}{
		{"local overrides global", "json", "table", "table"},
		{"empty local uses global", "json", "", "json"},
		{"whitespace local uses global", "table", "   ", "table"},
		{"custom global used", "yaml", "", "yaml"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			SetGlobalFormat(tc.globalFormat)
			got := ResolveFormat(tc.local)
			if got != tc.expected {
				t.Errorf("ResolveFormat(%q) = %q, want %q (global=%q)", tc.local, got, tc.expected, tc.globalFormat)
			}
		})
	}
}

func TestSetGlobalFormat(t *testing.T) {
	// Reset global state
	defer SetGlobalFormat("json")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "table", "table"},
		{"uppercase normalized", "JSON", "json"},
		{"mixed case normalized", "TaBlE", "table"},
		{"whitespace trimmed", "  json  ", "json"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			SetGlobalFormat(tc.input)
			got := GlobalFormat()
			if got != tc.expected {
				t.Errorf("after SetGlobalFormat(%q), GlobalFormat() = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}

	t.Run("empty string is ignored", func(t *testing.T) {
		SetGlobalFormat("table")
		SetGlobalFormat("")
		got := GlobalFormat()
		if got != "table" {
			t.Errorf("SetGlobalFormat(\"\") changed format to %q, should have been no-op", got)
		}
	})

	t.Run("whitespace only is ignored", func(t *testing.T) {
		SetGlobalFormat("yaml")
		SetGlobalFormat("   ")
		got := GlobalFormat()
		if got != "yaml" {
			t.Errorf("SetGlobalFormat(\"   \") changed format to %q, should have been no-op", got)
		}
	})
}
