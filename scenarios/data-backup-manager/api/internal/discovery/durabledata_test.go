package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/discovery"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func TestResolveBase(t *testing.T) {
	home := "/home/op"
	cases := []struct {
		name   string
		base   string
		want   string
		wantOK bool
	}{
		{"empty defaults to home", "", "/home/op", true},
		{"bare $HOME", "$HOME", "/home/op", true},
		{"bare tilde", "~", "/home/op", true},
		{"$HOME subdir", "$HOME/.claude", "/home/op/.claude", true},
		{"tilde subdir", "~/.codex", "/home/op/.codex", true},
		{"nested subdir", "$HOME/.local/share/opencode", "/home/op/.local/share/opencode", true},
		{"untokened absolute rejected", "/var/lib/x", "", false},
		{"untokened relative rejected", "foo/bar", "", false},
		{"traversal rejected", "$HOME/../etc", "", false},
		{"backslash rejected", "$HOME\\x", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := discovery.ResolveBaseForTest(tc.base, home)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (got %q)", ok, tc.wantOK, got)
			}
			if ok && filepath.ToSlash(got) != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLoadDurableData(t *testing.T) {
	dir := t.TempDir()

	withBlock := filepath.Join(dir, "with.json")
	if err := os.WriteFile(withBlock, []byte(`{
      "name": "x",
      "durable_data": {"base": "$HOME/.x", "entries": {"h": {"path": "h.jsonl", "kind": "file", "regenerable": false}}}
    }`), 0o644); err != nil {
		t.Fatal(err)
	}
	dd := discovery.LoadDurableDataForTest(withBlock)
	if dd == nil {
		t.Fatal("expected durable_data, got nil")
	}
	if dd.Base != "$HOME/.x" || len(dd.Entries) != 1 || dd.Entries["h"].Path != "h.jsonl" {
		t.Fatalf("unexpected decode: %+v", dd)
	}

	without := filepath.Join(dir, "without.json")
	if err := os.WriteFile(without, []byte(`{"name":"y","driver":"external-cli"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if dd := discovery.LoadDurableDataForTest(without); dd != nil {
		t.Fatalf("expected nil for manifest without durable_data, got %+v", dd)
	}

	if dd := discovery.LoadDurableDataForTest(filepath.Join(dir, "missing.json")); dd != nil {
		t.Fatalf("expected nil for missing manifest, got %+v", dd)
	}
}

func TestFilterResourcesKeepsEnabledExternalCLI(t *testing.T) {
	resources := []*cliv1.Resource{
		{Name: "claude-code", Enabled: true, Driver: "external-cli", ManifestPath: "/r/claude-code/resource.json"},
		{Name: "codex", Enabled: true, Driver: "external-cli", ManifestPath: "/r/codex/resource.json"},
		{Name: "postgres", Enabled: true, Driver: "compose-service", ManifestPath: "/r/postgres/resource.json"},
		{Name: "disabled-cli", Enabled: false, Driver: "external-cli", ManifestPath: "/r/d/resource.json"},
		{Name: "no-manifest", Enabled: true, Driver: "external-cli", ManifestPath: ""},
	}
	refs := discovery.FilterResourcesForTest(resources)
	if len(refs) != 2 {
		t.Fatalf("expected 2 enabled external-cli refs, got %d: %+v", len(refs), refs)
	}
	names := map[string]bool{}
	for _, r := range refs {
		names[r.Name] = true
	}
	if !names["claude-code"] || !names["codex"] {
		t.Fatalf("expected claude-code + codex, got %+v", refs)
	}
}
