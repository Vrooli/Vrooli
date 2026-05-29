package migration

import (
	"os"
	"path/filepath"
	"testing"

	migrationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/migration"

	"github.com/vrooli/cli-core/cliapp"
)

// TestManifestCoversMigrationService asserts that every RPC declared on
// MigrationService either has a manifest command binding or is documented
// in the manifest's `omitted` array. Catches proto↔CLI drift: adding a new
// RPC without binding/omitting it fails here.
func TestManifestCoversMigrationService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, migrationv1.File_architecture_cartographer_v1_migration_migration_proto, "MigrationService")
}

// TestRegisterWiresAllCommands proves Register builds the group cleanly
// from cli/manifest.json — every binding resolves to a handler and every
// handler maps to a command.
func TestRegisterWiresAllCommands(t *testing.T) {
	group, err := Register(&cliapp.ScenarioApp{}, readManifest(t))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if group.Name != GroupName {
		t.Errorf("group name = %q, want %q", group.Name, GroupName)
	}
	want := map[string]bool{
		"create": false, "status": false, "next": false, "resolve": false,
		"apply": false, "reaudit": false, "close": false,
	}
	for _, c := range group.Subcommands {
		if _, ok := want[c.Name]; !ok {
			t.Errorf("unexpected command %q", c.Name)
			continue
		}
		want[c.Name] = true
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("command %q not registered", name)
		}
	}
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
