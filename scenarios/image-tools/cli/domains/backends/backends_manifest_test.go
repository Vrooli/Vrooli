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
	got := make(map[string]bool, len(group.Subcommands))
	for _, sub := range group.Subcommands {
		got[sub.Name] = true
	}
	for _, want := range []string{"doctor", "ensure"} {
		if !got[want] {
			t.Fatalf("missing %q command; got: %+v", want, group.Subcommands)
		}
	}
	if len(group.Subcommands) != 2 {
		t.Fatalf("expected exactly doctor + ensure, got: %+v", group.Subcommands)
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
