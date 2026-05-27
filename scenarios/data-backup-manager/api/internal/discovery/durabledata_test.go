package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/discovery"
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

func TestParseResourceListFiltersEnabledExternalCLI(t *testing.T) {
	out := []byte(`{"resources":[
      {"name":"claude-code","enabled":true,"driver":"external-cli","manifest_path":"/r/claude-code/resource.json"},
      {"name":"codex","enabled":true,"driver":"external-cli","manifest_path":"/r/codex/resource.json"},
      {"name":"postgres","enabled":true,"driver":"compose-service","manifest_path":"/r/postgres/resource.json"},
      {"name":"disabled-cli","enabled":false,"driver":"external-cli","manifest_path":"/r/d/resource.json"},
      {"name":"no-manifest","enabled":true,"driver":"external-cli","manifest_path":""}
    ]}`)
	refs := discovery.ParseResourceListForTest(out)
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

func TestParseResourceListToleratesGarbage(t *testing.T) {
	if refs := discovery.ParseResourceListForTest([]byte("not json")); len(refs) != 0 {
		t.Fatalf("expected no refs for garbage input, got %+v", refs)
	}
}
