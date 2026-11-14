package statusline

import (
	"path/filepath"
	"testing"

	corestorage "github.com/vrooli/api-core/storage"
)

// Agent Manager reads the feed through api-core/storage (state class, owner
// claude-code); the writer restates that rule so it works from any working
// directory. The two must name the same file whatever the storage variables
// say, or the reader silently sees an empty feed.
func TestSamplesPathAgreesWithTheStorageResolver(t *testing.T) {
	home := t.TempDir()
	homeDir := func() (string, error) { return home, nil }
	for name, env := range map[string]map[string]string{
		"defaults":            nil,
		"custom storage root": {"VROOLI_STORAGE_ROOT": t.TempDir()},
		"custom state root":   {"VROOLI_STATE_ROOT": t.TempDir()},
		"storage root wins":   {"VROOLI_STORAGE_ROOT": t.TempDir(), "VROOLI_STATE_ROOT": t.TempDir()},
	} {
		getenv := func(key string) string { return env[key] }
		writer, err := SamplesPath(getenv, homeDir)
		if err != nil {
			t.Fatalf("%s: writer: %v", name, err)
		}
		resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{EnvGet: getenv, UserHomeDir: homeDir})
		if err != nil {
			t.Fatal(err)
		}
		reader, err := resolver.Path(corestorage.Options{ScenarioID: "claude-code"}, corestorage.ClassState, "quota-samples.ndjson")
		if err != nil {
			t.Fatalf("%s: reader: %v", name, err)
		}
		if writer != reader {
			t.Errorf("%s: writer %q != reader %q", name, writer, reader)
		}
	}
}

func TestSamplesPathOverrideAndInvalidRoots(t *testing.T) {
	homeDir := func() (string, error) { return t.TempDir(), nil }
	override := filepath.Join(t.TempDir(), "feed.ndjson")
	got, err := SamplesPath(func(key string) string {
		return map[string]string{SamplesEnv: override, "VROOLI_STORAGE_ROOT": "/elsewhere"}[key]
	}, homeDir)
	if err != nil || got != override {
		t.Fatalf("override: %q, %v; want %q", got, err, override)
	}
	for _, key := range []string{"VROOLI_STORAGE_ROOT", "VROOLI_STATE_ROOT"} {
		if _, err := SamplesPath(func(k string) string {
			if k == key {
				return "relative/root"
			}
			return ""
		}, homeDir); err == nil {
			t.Errorf("relative %s accepted", key)
		}
	}
}
