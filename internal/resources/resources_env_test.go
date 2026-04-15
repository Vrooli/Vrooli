package resources

import (
	"path/filepath"
	"testing"
)

func TestResourceEnvForResourceIncludesCanonicalStorageVariables(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	env := resourceEnvForResource(root, home, "home-assistant")
	values := map[string]string{}
	for _, entry := range env {
		for i := 0; i < len(entry); i++ {
			if entry[i] == '=' {
				values[entry[:i]] = entry[i+1:]
				break
			}
		}
	}

	if got := values["RESOURCE_CONFIG_DIR"]; got == "" {
		t.Fatal("expected RESOURCE_CONFIG_DIR")
	}
	if got := values["RESOURCE_DATA_DIR"]; got == "" {
		t.Fatal("expected RESOURCE_DATA_DIR")
	}
	wantRoot := filepath.Join(root, "resources", "home-assistant")
	if got := values["RESOURCE_ROOT"]; got != wantRoot {
		t.Fatalf("RESOURCE_ROOT = %q, want %q", got, wantRoot)
	}
}
