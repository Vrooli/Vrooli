package discovery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/discovery"
	"data-backup-manager/internal/discovery/mocks"
	"data-backup-manager/internal/sources"
)

// fakeAgentHome builds a temp $HOME with a coding-agent tree mirroring the real
// shape: durable bits worth protecting plus regenerable junk that must not be
// suggested. It also writes a resource manifest carrying a durable_data block,
// and returns the home dir and that manifest's path.
func fakeAgentHome(t *testing.T) (home, manifestPath string) {
	t.Helper()
	home = t.TempDir()

	write := func(rel, contents string) {
		abs := filepath.Join(home, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", abs, err)
		}
		if err := os.WriteFile(abs, []byte(contents), 0o644); err != nil {
			t.Fatalf("write %s: %v", abs, err)
		}
	}

	// Durable (declared, non-regenerable) → should be suggested.
	write(".agent/history.jsonl", `{"prompt":"hi"}`)
	write(".agent/projects/p1/transcript.jsonl", "line")
	write(".agent/state.sqlite", "SQLITEfakebytes")
	write(".agent/.credentials.json", `{"token":"secret"}`)
	// Regenerable / unlisted → must NOT be suggested.
	write(".agent/cache/blob", "noise")
	write(".agent/debug/log.txt", "noise")
	// An empty declared dir → skipped (exists but no entries).
	if err := os.MkdirAll(filepath.Join(home, ".agent", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	manifestPath = filepath.Join(home, "agent-resource.json")
	manifest := `{
      "name": "agent",
      "driver": "external-cli",
      "durable_data": {
        "base": "$HOME/.agent",
        "entries": {
          "history":     {"path": "history.jsonl", "kind": "file", "regenerable": false, "rationale": "Prompt history."},
          "projects":    {"path": "projects", "kind": "dir", "regenerable": false},
          "state":       {"path": "state.sqlite", "kind": "file", "format": "sqlite", "regenerable": false},
          "credentials": {"path": ".credentials.json", "kind": "file", "sensitive": true, "regenerable": false},
          "cache":       {"path": "cache", "kind": "dir", "regenerable": true},
          "empty":       {"path": "empty", "kind": "dir", "regenerable": false},
          "missing":     {"path": "does-not-exist", "kind": "dir", "regenerable": false}
        }
      }
    }`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return home, manifestPath
}

func TestResourceDataScannerEmitsDeclaredDurableEntries(t *testing.T) {
	home, manifestPath := fakeAgentHome(t)
	enum := &mocks.FakeResourceEnumerator{Refs: []discovery.ResourceRef{
		{Name: "agent", Driver: "external-cli", Enabled: true, ManifestPath: manifestPath},
	}}
	scanner := discovery.NewResourceDataScannerWithHome(enum, home)

	got, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	byName := map[string]discovery.TargetCandidate{}
	for _, c := range got {
		byName[c.Name] = c
	}

	// history, projects, state, credentials survive; cache (regenerable),
	// empty (no entries), missing (absent) are excluded.
	want := []string{"history", "projects", "state", "credentials"}
	if len(got) != len(want) {
		t.Fatalf("expected %d candidates %v, got %d: %+v", len(want), want, len(got), got)
	}
	for _, n := range want {
		c, ok := byName[n]
		if !ok {
			t.Fatalf("missing candidate %q", n)
		}
		if c.Owner != "agent" {
			t.Errorf("%q owner = %q, want agent", n, c.Owner)
		}
		if !filepath.IsAbs(c.Locator) {
			t.Errorf("%q locator %q not absolute", n, c.Locator)
		}
	}
	for _, banned := range []string{"cache", "empty", "missing"} {
		if _, ok := byName[banned]; ok {
			t.Errorf("%q must not be suggested", banned)
		}
	}

	// state.sqlite captured as a SQLite source; others as filesystem.
	if k := byName["state"].SourceKind; k != sources.KindSQLite {
		t.Errorf("state source kind = %q, want sqlite", k)
	}
	if k := byName["history"].SourceKind; k != sources.KindFilesystem {
		t.Errorf("history source kind = %q, want filesystem", k)
	}

	// Credentials surfaced but flagged sensitive; non-secret entries are not.
	if !byName["credentials"].Sensitive {
		t.Error("credentials must be flagged sensitive")
	}
	if byName["history"].Sensitive {
		t.Error("history must not be flagged sensitive")
	}

	// Authored rationale is used when present.
	if byName["history"].Rationale != "Prompt history." {
		t.Errorf("history rationale = %q, want authored copy", byName["history"].Rationale)
	}
	// A bounded size estimate is produced for a non-empty entry.
	if byName["projects"].ApproxBytes <= 0 {
		t.Errorf("projects approx bytes = %d, want > 0", byName["projects"].ApproxBytes)
	}
}

func TestResourceDataScannerSkipsResourceWithoutDurableData(t *testing.T) {
	home := t.TempDir()
	manifestPath := filepath.Join(home, "plain.json")
	if err := os.WriteFile(manifestPath, []byte(`{"name":"plain","driver":"external-cli"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	enum := &mocks.FakeResourceEnumerator{Refs: []discovery.ResourceRef{
		{Name: "plain", Driver: "external-cli", Enabled: true, ManifestPath: manifestPath},
	}}
	got, err := discovery.NewResourceDataScannerWithHome(enum, home).Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no candidates, got %+v", got)
	}
}

func TestResourceDataScannerEmptyHomeYieldsNothing(t *testing.T) {
	enum := &mocks.FakeResourceEnumerator{Refs: []discovery.ResourceRef{{Name: "agent", ManifestPath: "/nope"}}}
	got, err := discovery.NewResourceDataScannerWithHome(enum, "").Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected nothing for empty home, got %+v", got)
	}
}
