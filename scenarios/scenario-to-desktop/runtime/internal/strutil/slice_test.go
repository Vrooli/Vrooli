package strutil

import (
	"sort"
	"testing"
)

func TestIntersection(t *testing.T) {
	t.Run("finds common elements", func(t *testing.T) {
		a := []string{"apple", "banana", "cherry"}
		b := []string{"banana", "cherry", "date"}

		got := Intersection(a, b)
		sort.Strings(got)

		want := []string{"banana", "cherry"}
		if len(got) != len(want) {
			t.Fatalf("Intersection() returned %d elements, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("Intersection()[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("returns empty for disjoint sets", func(t *testing.T) {
		a := []string{"apple", "banana"}
		b := []string{"cherry", "date"}

		got := Intersection(a, b)

		if len(got) != 0 {
			t.Errorf("Intersection() = %v, want empty slice", got)
		}
	})

	t.Run("handles empty first slice", func(t *testing.T) {
		a := []string{}
		b := []string{"apple", "banana"}

		got := Intersection(a, b)

		if len(got) != 0 {
			t.Errorf("Intersection() = %v, want empty slice", got)
		}
	})

	t.Run("handles empty second slice", func(t *testing.T) {
		a := []string{"apple", "banana"}
		b := []string{}

		got := Intersection(a, b)

		if len(got) != 0 {
			t.Errorf("Intersection() = %v, want empty slice", got)
		}
	})

	t.Run("handles identical slices", func(t *testing.T) {
		a := []string{"apple", "banana"}
		b := []string{"apple", "banana"}

		got := Intersection(a, b)
		sort.Strings(got)

		if len(got) != 2 {
			t.Fatalf("Intersection() returned %d elements, want 2", len(got))
		}
		if got[0] != "apple" || got[1] != "banana" {
			t.Errorf("Intersection() = %v, want [apple banana]", got)
		}
	})
}

func TestCopyStringMap(t *testing.T) {
	t.Run("creates independent copy", func(t *testing.T) {
		original := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		copied := CopyStringMap(original)

		// Modify the copy
		copied["key1"] = "modified"
		copied["key3"] = "new"

		// Original should be unchanged
		if original["key1"] != "value1" {
			t.Errorf("original[key1] = %q, want %q (should not be modified)", original["key1"], "value1")
		}
		if _, exists := original["key3"]; exists {
			t.Error("original[key3] should not exist")
		}
	})

	t.Run("handles empty map", func(t *testing.T) {
		original := map[string]string{}
		copied := CopyStringMap(original)

		if len(copied) != 0 {
			t.Errorf("copied map has %d elements, want 0", len(copied))
		}
	})

	t.Run("handles nil map", func(t *testing.T) {
		var original map[string]string
		copied := CopyStringMap(original)

		if copied == nil {
			t.Error("CopyStringMap(nil) should return non-nil map")
		}
		if len(copied) != 0 {
			t.Errorf("copied map has %d elements, want 0", len(copied))
		}
	})
}

func TestEnvMapToList(t *testing.T) {
	t.Run("converts to KEY=VALUE format", func(t *testing.T) {
		env := map[string]string{
			"PATH":      "/usr/bin",
			"HOME":      "/home/user",
			"LOG_LEVEL": "debug",
		}

		got := EnvMapToList(env)

		if len(got) != 3 {
			t.Fatalf("EnvMapToList() returned %d elements, want 3", len(got))
		}

		// Sort for deterministic comparison
		sort.Strings(got)
		want := []string{
			"HOME=/home/user",
			"LOG_LEVEL=debug",
			"PATH=/usr/bin",
		}

		for i := range want {
			if got[i] != want[i] {
				t.Errorf("EnvMapToList()[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("handles empty map", func(t *testing.T) {
		got := EnvMapToList(map[string]string{})
		if len(got) != 0 {
			t.Errorf("EnvMapToList({}) returned %d elements, want 0", len(got))
		}
	})

	t.Run("handles values with equals signs", func(t *testing.T) {
		env := map[string]string{
			"CONNECTION": "host=localhost;port=5432",
		}

		got := EnvMapToList(env)

		if len(got) != 1 {
			t.Fatalf("EnvMapToList() returned %d elements, want 1", len(got))
		}

		want := "CONNECTION=host=localhost;port=5432"
		if got[0] != want {
			t.Errorf("EnvMapToList()[0] = %q, want %q", got[0], want)
		}
	})

	t.Run("handles empty values", func(t *testing.T) {
		env := map[string]string{
			"EMPTY": "",
		}

		got := EnvMapToList(env)

		if len(got) != 1 {
			t.Fatalf("EnvMapToList() returned %d elements, want 1", len(got))
		}

		want := "EMPTY="
		if got[0] != want {
			t.Errorf("EnvMapToList()[0] = %q, want %q", got[0], want)
		}
	})
}
