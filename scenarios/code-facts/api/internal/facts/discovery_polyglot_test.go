package facts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverNestedParseUnitsEmitsPolyglotDependencyEvidence(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{
		"web/bun.lock":           `{"lockfileVersion":1}`,
		"tools/requirements.txt": "requests==2.32.0\n",
		"native/Cargo.toml":      "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n",
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	units := discoverNestedParseUnits(root)
	seen := map[string]bool{}
	for _, unit := range units {
		seen[unit.GetLanguage()] = true
	}
	for _, language := range []string{"node", "python", "rust"} {
		if !seen[language] {
			t.Errorf("language %q not discovered: %+v", language, units)
		}
	}
}
