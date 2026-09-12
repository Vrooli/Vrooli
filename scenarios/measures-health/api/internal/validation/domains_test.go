package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProtoDomainSource_DerivesAndFilters(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "packages", "proto", "schemas", "swarm-manager", "v1", "domain")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"backlog.proto", "capture.proto", "settings.proto", "README.md"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("syntax=\"proto3\";"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ProtoDomainSource{RepoRoot: root}.StatefulDomains("swarm-manager")
	if err != nil {
		t.Fatal(err)
	}
	// backlog + capture + settings (README.md ignored); sorted.
	if len(got) != 3 {
		t.Fatalf("want 3 derived domains, got %d: %+v", len(got), got)
	}
	byName := map[string]DerivedDomain{}
	for _, d := range got {
		byName[d.Name] = d
	}
	if !byName["backlog"].Stateful || !byName["capture"].Stateful {
		t.Fatalf("backlog/capture should be stateful: %+v", got)
	}
	if byName["settings"].Stateful {
		t.Fatalf("settings should be filtered to non-stateful: %+v", byName["settings"])
	}
	if byName["settings"].Note == "" {
		t.Fatal("filtered domain should carry a note")
	}
}

func TestProtoDomainSource_NoDomainFolder(t *testing.T) {
	got, err := ProtoDomainSource{RepoRoot: t.TempDir()}.StatefulDomains("nope")
	if err != nil {
		t.Fatalf("missing domain folder must not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want no derived domains, got %+v", got)
	}
}

func TestProtoDomainSource_Mode(t *testing.T) {
	root := t.TempDir()
	// A scenario with a v1/domain/ folder (even empty) is conformant.
	confDir := filepath.Join(root, "packages", "proto", "schemas", "conf", "v1", "domain")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := ProtoDomainSource{RepoRoot: root}
	if m, err := p.Mode("conf"); err != nil || m != ModeConformant {
		t.Fatalf("conf: want conformant, got %q err=%v", m, err)
	}
	// A scenario with no v1/domain/ folder is fallback.
	if m, err := p.Mode("flat"); err != nil || m != ModeFallback {
		t.Fatalf("flat: want fallback, got %q err=%v", m, err)
	}
}

func TestNormalizeDomain(t *testing.T) {
	cases := map[string]string{
		"Agent-Session": "agent_session",
		"backlog":       "backlog",
		"  Foo-Bar  ":   "foo_bar",
	}
	for in, want := range cases {
		if got := normalizeDomain(in); got != want {
			t.Errorf("normalizeDomain(%q) = %q, want %q", in, got, want)
		}
	}
}
