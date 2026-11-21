package disputes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// TestDisputesGroupBuildsFromManifest asserts the disputes group loads from the
// embedded manifest with both commands (list, resolve) wired to their
// FindingsService bindings. The FindingsService<->proto parity itself is
// enforced by the findings domain's coverage test, which scans the whole
// manifest including the disputes group's bindings.
func TestDisputesGroupBuildsFromManifest(t *testing.T) {
	manifest := readDisputesManifest(t)
	noop := func(cliapp.RunContext) error { return nil }
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"FindingsService.ListDisputes":   noop,
		"FindingsService.ResolveDispute": noop,
	})
	if err != nil {
		t.Fatalf("load disputes group from manifest: %v", err)
	}
	if group.Name != GroupName {
		t.Fatalf("group name = %q, want %q", group.Name, GroupName)
	}
	wantCommands := map[string]bool{"list": false, "resolve": false}
	for _, c := range group.Subcommands {
		if _, ok := wantCommands[c.Name]; ok {
			wantCommands[c.Name] = true
		}
	}
	for name, found := range wantCommands {
		if !found {
			t.Errorf("disputes group missing command %q", name)
		}
	}
}

func readDisputesManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
