package bundleruntime

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"scenario-to-desktop-runtime/deps"
	"scenario-to-desktop-runtime/manifest"
)

func TestCopyStringMap(t *testing.T) {
	t.Run("creates independent copy", func(t *testing.T) {
		original := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		copied := copyStringMap(original)

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
		copied := copyStringMap(original)

		if len(copied) != 0 {
			t.Errorf("copied map has %d elements, want 0", len(copied))
		}
	})

	t.Run("handles nil map", func(t *testing.T) {
		var original map[string]string
		copied := copyStringMap(original)

		if copied == nil {
			t.Error("copyStringMap(nil) should return non-nil map")
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

		got := envMapToList(env)

		if len(got) != 3 {
			t.Fatalf("envMapToList() returned %d elements, want 3", len(got))
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
				t.Errorf("envMapToList()[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("handles empty map", func(t *testing.T) {
		got := envMapToList(map[string]string{})
		if len(got) != 0 {
			t.Errorf("envMapToList({}) returned %d elements, want 0", len(got))
		}
	})

	t.Run("handles values with equals signs", func(t *testing.T) {
		env := map[string]string{
			"CONNECTION": "host=localhost;port=5432",
		}

		got := envMapToList(env)

		if len(got) != 1 {
			t.Fatalf("envMapToList() returned %d elements, want 1", len(got))
		}

		want := "CONNECTION=host=localhost;port=5432"
		if got[0] != want {
			t.Errorf("envMapToList()[0] = %q, want %q", got[0], want)
		}
	})

	t.Run("handles empty values", func(t *testing.T) {
		env := map[string]string{
			"EMPTY": "",
		}

		got := envMapToList(env)

		if len(got) != 1 {
			t.Fatalf("envMapToList() returned %d elements, want 1", len(got))
		}

		want := "EMPTY="
		if got[0] != want {
			t.Errorf("envMapToList()[0] = %q, want %q", got[0], want)
		}
	})
}

func TestParsePositiveInt(t *testing.T) {
	t.Run("parses valid positive integers", func(t *testing.T) {
		tests := []struct {
			input string
			want  int
		}{
			{"1", 1},
			{"42", 42},
			{"  100  ", 100},
			{"999999", 999999},
		}

		for _, tc := range tests {
			got, err := parsePositiveInt(tc.input)
			if err != nil {
				t.Errorf("parsePositiveInt(%q) error = %v, want nil", tc.input, err)
				continue
			}
			if got != tc.want {
				t.Errorf("parsePositiveInt(%q) = %d, want %d", tc.input, got, tc.want)
			}
		}
	})

	t.Run("rejects zero", func(t *testing.T) {
		_, err := parsePositiveInt("0")
		if err == nil {
			t.Error("parsePositiveInt(\"0\") should return error")
		}
	})

	t.Run("rejects negative numbers", func(t *testing.T) {
		_, err := parsePositiveInt("-5")
		if err == nil {
			t.Error("parsePositiveInt(\"-5\") should return error")
		}
	})

	t.Run("rejects non-numeric input", func(t *testing.T) {
		_, err := parsePositiveInt("abc")
		if err == nil {
			t.Error("parsePositiveInt(\"abc\") should return error")
		}
	})

	t.Run("rejects empty string", func(t *testing.T) {
		_, err := parsePositiveInt("")
		if err == nil {
			t.Error("parsePositiveInt(\"\") should return error")
		}
	})
}

func TestTailFile(t *testing.T) {
	// Helper to create a Supervisor with RealFileSystem for tailFile tests
	newTestSupervisor := func() *Supervisor {
		return &Supervisor{fs: RealFileSystem{}}
	}

	t.Run("returns last N lines", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.log")
		content := "line1\nline2\nline3\nline4\nline5"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		s := newTestSupervisor()
		got, err := s.tailFile(path, 3)
		if err != nil {
			t.Fatalf("tailFile() error = %v", err)
		}

		want := "line3\nline4\nline5"
		if string(got) != want {
			t.Errorf("tailFile() = %q, want %q", string(got), want)
		}
	})

	t.Run("returns all lines when N exceeds file length", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.log")
		content := "line1\nline2"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		s := newTestSupervisor()
		got, err := s.tailFile(path, 100)
		if err != nil {
			t.Fatalf("tailFile() error = %v", err)
		}

		if string(got) != content {
			t.Errorf("tailFile() = %q, want %q", string(got), content)
		}
	})

	t.Run("handles empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.log")
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		s := newTestSupervisor()
		got, err := s.tailFile(path, 10)
		if err != nil {
			t.Fatalf("tailFile() error = %v", err)
		}

		if string(got) != "" {
			t.Errorf("tailFile() = %q, want empty string", string(got))
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		s := newTestSupervisor()
		_, err := s.tailFile("/nonexistent/path/file.log", 10)
		if err == nil {
			t.Error("tailFile() should return error for missing file")
		}
	})
}

func TestIntersection(t *testing.T) {
	t.Run("finds common elements", func(t *testing.T) {
		a := []string{"apple", "banana", "cherry"}
		b := []string{"banana", "cherry", "date"}

		got := intersection(a, b)
		sort.Strings(got)

		want := []string{"banana", "cherry"}
		if len(got) != len(want) {
			t.Fatalf("intersection() returned %d elements, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("intersection()[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("returns empty for disjoint sets", func(t *testing.T) {
		a := []string{"apple", "banana"}
		b := []string{"cherry", "date"}

		got := intersection(a, b)

		if len(got) != 0 {
			t.Errorf("intersection() = %v, want empty slice", got)
		}
	})

	t.Run("handles empty first slice", func(t *testing.T) {
		a := []string{}
		b := []string{"apple", "banana"}

		got := intersection(a, b)

		if len(got) != 0 {
			t.Errorf("intersection() = %v, want empty slice", got)
		}
	})

	t.Run("handles empty second slice", func(t *testing.T) {
		a := []string{"apple", "banana"}
		b := []string{}

		got := intersection(a, b)

		if len(got) != 0 {
			t.Errorf("intersection() = %v, want empty slice", got)
		}
	})

	t.Run("handles identical slices", func(t *testing.T) {
		a := []string{"apple", "banana"}
		b := []string{"apple", "banana"}

		got := intersection(a, b)
		sort.Strings(got)

		if len(got) != 2 {
			t.Fatalf("intersection() returned %d elements, want 2", len(got))
		}
		if got[0] != "apple" || got[1] != "banana" {
			t.Errorf("intersection() = %v, want [apple banana]", got)
		}
	})
}

func TestTopoSort(t *testing.T) {
	t.Run("sorts services with no dependencies", func(t *testing.T) {
		services := []manifest.Service{
			{ID: "a"},
			{ID: "b"},
			{ID: "c"},
		}

		order, err := deps.TopoSort(services)
		if err != nil {
			t.Fatalf("deps.TopoSort() error = %v", err)
		}

		if len(order) != 3 {
			t.Errorf("deps.TopoSort() returned %d elements, want 3", len(order))
		}
	})

	t.Run("respects dependencies", func(t *testing.T) {
		services := []manifest.Service{
			{ID: "api", Dependencies: []string{"db"}},
			{ID: "db"},
			{ID: "cache"},
		}

		order, err := deps.TopoSort(services)
		if err != nil {
			t.Fatalf("deps.TopoSort() error = %v", err)
		}

		// Find indices
		dbIdx, apiIdx := -1, -1
		for i, id := range order {
			if id == "db" {
				dbIdx = i
			}
			if id == "api" {
				apiIdx = i
			}
		}

		if dbIdx == -1 || apiIdx == -1 {
			t.Fatalf("deps.TopoSort() missing expected services, got %v", order)
		}

		if dbIdx > apiIdx {
			t.Errorf("deps.TopoSort() db should come before api, got order %v", order)
		}
	})

	t.Run("handles chain of dependencies", func(t *testing.T) {
		services := []manifest.Service{
			{ID: "c", Dependencies: []string{"b"}},
			{ID: "b", Dependencies: []string{"a"}},
			{ID: "a"},
		}

		order, err := deps.TopoSort(services)
		if err != nil {
			t.Fatalf("deps.TopoSort() error = %v", err)
		}

		// a should come first, then b, then c
		if len(order) != 3 {
			t.Fatalf("deps.TopoSort() returned %d elements, want 3", len(order))
		}

		idxA, idxB, idxC := -1, -1, -1
		for i, id := range order {
			switch id {
			case "a":
				idxA = i
			case "b":
				idxB = i
			case "c":
				idxC = i
			}
		}

		if idxA > idxB || idxB > idxC {
			t.Errorf("deps.TopoSort() should order a < b < c, got %v", order)
		}
	})

	t.Run("detects cycle", func(t *testing.T) {
		services := []manifest.Service{
			{ID: "a", Dependencies: []string{"b"}},
			{ID: "b", Dependencies: []string{"a"}},
		}

		_, err := deps.TopoSort(services)
		if err == nil {
			t.Error("deps.TopoSort() should detect cycle")
		}
	})

	t.Run("handles multiple dependencies", func(t *testing.T) {
		services := []manifest.Service{
			{ID: "app", Dependencies: []string{"api", "cache"}},
			{ID: "api", Dependencies: []string{"db"}},
			{ID: "cache"},
			{ID: "db"},
		}

		order, err := deps.TopoSort(services)
		if err != nil {
			t.Fatalf("deps.TopoSort() error = %v", err)
		}

		if len(order) != 4 {
			t.Fatalf("deps.TopoSort() returned %d elements, want 4", len(order))
		}

		// app should be last since it depends on api and cache
		idxApp := -1
		for i, id := range order {
			if id == "app" {
				idxApp = i
			}
		}

		if idxApp != len(order)-1 {
			t.Errorf("deps.TopoSort() app should be last, got order %v", order)
		}
	})

	t.Run("handles empty services list", func(t *testing.T) {
		order, err := deps.TopoSort([]manifest.Service{})
		if err != nil {
			t.Fatalf("deps.TopoSort() error = %v", err)
		}
		if len(order) != 0 {
			t.Errorf("deps.TopoSort([]) should return empty, got %v", order)
		}
	})
}

func TestFindService(t *testing.T) {
	services := []manifest.Service{
		{ID: "api", Type: "api"},
		{ID: "db", Type: "database"},
		{ID: "cache", Type: "cache"},
	}

	t.Run("finds existing service", func(t *testing.T) {
		svc := deps.FindService(services, "db")
		if svc == nil {
			t.Fatal("deps.FindService() returned nil for existing service")
		}
		if svc.ID != "db" {
			t.Errorf("deps.FindService() returned wrong service, got ID=%q", svc.ID)
		}
		if svc.Type != "database" {
			t.Errorf("deps.FindService() returned wrong type, got Type=%q", svc.Type)
		}
	})

	t.Run("returns nil for non-existent service", func(t *testing.T) {
		svc := deps.FindService(services, "nonexistent")
		if svc != nil {
			t.Errorf("deps.FindService() should return nil for non-existent, got %+v", svc)
		}
	})

	t.Run("returns nil for empty slice", func(t *testing.T) {
		svc := deps.FindService([]manifest.Service{}, "api")
		if svc != nil {
			t.Errorf("deps.FindService() should return nil for empty slice, got %+v", svc)
		}
	})

	t.Run("returns pointer to original slice element", func(t *testing.T) {
		svc := deps.FindService(services, "api")
		if svc == nil {
			t.Fatal("deps.FindService() returned nil")
		}
		// Verify it's the same underlying data
		if svc != &services[0] {
			t.Error("deps.FindService() should return pointer to original slice element")
		}
	})
}
