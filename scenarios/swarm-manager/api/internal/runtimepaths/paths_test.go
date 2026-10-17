package runtimepaths

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestPathsResolveUnderStorageRoot (T-R1): with VROOLI_STORAGE_ROOT set, every
// class path resolves under that root with the "<class>/vrooli/swarm-manager"
// scoping segment.
func TestPathsResolveUnderStorageRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)

	cases := []struct {
		name  string
		got   func(string) (string, error)
		class string
		rel   string
	}{
		{"data", DataPath, "data", "ideas"},
		{"cache", CachePath, "cache", "captures"},
		{"state", StatePath, "state", "queue.json"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.got(c.rel)
			if err != nil {
				t.Fatalf("%s(%q) error = %v", c.name, c.rel, err)
			}
			want := filepath.Join(root, c.class, "vrooli", "swarm-manager", c.rel)
			if got != want {
				t.Fatalf("%s(%q) = %q, want %q", c.name, c.rel, got, want)
			}
			if !strings.HasPrefix(got, root) {
				t.Fatalf("%s path %q not under storage root %q", c.name, got, root)
			}
		})
	}
}

// TestCachePathIsDistinctFromData (T-R2): captures must resolve to the cache
// class, never data — the regression guard that disposable captures never land
// in the backed-up data class.
func TestCachePathIsDistinctFromData(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)

	cache, err := CachePath("captures")
	if err != nil {
		t.Fatalf("CachePath error = %v", err)
	}
	data, err := DataPath("captures")
	if err != nil {
		t.Fatalf("DataPath error = %v", err)
	}
	if cache == data {
		t.Fatalf("CachePath and DataPath resolved identically: %q", cache)
	}
	if !strings.Contains(cache, filepath.Join("cache", "vrooli", "swarm-manager")) {
		t.Fatalf("CachePath %q not under the cache class", cache)
	}
	if strings.Contains(cache, filepath.Join("data", "vrooli")) {
		t.Fatalf("CachePath %q leaked into the data class", cache)
	}
}
