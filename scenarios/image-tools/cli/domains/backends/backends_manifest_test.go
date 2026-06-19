package backends

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackendsGroupRegisters(t *testing.T) {
	group, err := Register(nil, readManifest(t))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if group.Name != GroupName {
		t.Fatalf("group name = %q, want %q", group.Name, GroupName)
	}
	if len(group.Subcommands) != 1 || group.Subcommands[0].Name != "doctor" {
		t.Fatalf("unexpected commands: %+v", group.Subcommands)
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
