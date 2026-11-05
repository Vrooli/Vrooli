package storage

import (
	"errors"
	"path/filepath"
	"testing"
)

func governedInputs(t *testing.T) GovernedRootInputs {
	t.Helper()
	base := t.TempDir()
	return GovernedRootInputs{
		Home:        filepath.Join(base, "home"),
		RuntimeHome: filepath.Join(base, "home", ".vrooli"),
		RepoRoot:    filepath.Join(base, "repo"),
		TempDir:     filepath.Join(base, "tmp"),
		CacheDir:    filepath.Join(base, "home", ".cache"),
		GoCache:     filepath.Join(base, "gocache"),
		GoModCache:  filepath.Join(base, "gomodcache"),
	}
}

// Every variable the three storage-manager expansions used to handle resolves
// the same way through the one shared function.
func TestResolveGovernedRootExpandsEveryDeclaredVariable(t *testing.T) {
	in := governedInputs(t)
	cases := map[string]string{
		"$USER_HOME/.cache/go-build": filepath.Join(in.Home, ".cache", "go-build"),
		"$HOME/cache":                filepath.Join(in.Home, "cache"),
		"${HOME}/cache":              filepath.Join(in.Home, "cache"),
		"~/cache":                    filepath.Join(in.Home, "cache"),
		"$VROOLI_HOME/tmp/go-work":   filepath.Join(in.RuntimeHome, "tmp", "go-work"),
		"${VROOLI_HOME}/cache":       filepath.Join(in.RuntimeHome, "cache"),
		"$REPO_ROOT/scratch":         filepath.Join(in.RepoRoot, "scratch"),
		"$XDG_CACHE_HOME/uv":         filepath.Join(in.CacheDir, "uv"),
		"$TMPDIR/go-build*":          filepath.Join(in.TempDir, "go-build*"),
		"$GOCACHE":                   in.GoCache,
		"$GOMODCACHE/cache":          filepath.Join(in.GoModCache, "cache"),
	}
	for raw, want := range cases {
		if got := ResolveGovernedRoot(raw, in); got != want {
			t.Errorf("ResolveGovernedRoot(%q) = %q, want %q", raw, got, want)
		}
	}
}

// An unknown or unconfigured variable stays literal, and the strict form
// refuses it rather than returning a guessed path.
func TestResolveGovernedRootLeavesUnknownVariablesVisible(t *testing.T) {
	in := governedInputs(t)
	in.GoCache = ""
	if got := ResolveGovernedRoot("$GOCACHE", in); got != "$GOCACHE" {
		t.Fatalf("unconfigured $GOCACHE resolved to %q", got)
	}
	if _, err := ResolveGovernedRootStrict("$GOCACHE", in); !errors.Is(err, ErrGovernedRootUnresolved) {
		t.Fatalf("strict resolution of an unknown variable: err = %v", err)
	}
	if _, err := ResolveGovernedRootStrict("relative/path", in); !errors.Is(err, ErrGovernedRootUnresolved) {
		t.Fatalf("strict resolution of a relative path: err = %v", err)
	}
	if got, err := ResolveGovernedRootStrict("$USER_HOME/.cache", in); err != nil || got != filepath.Join(in.Home, ".cache") {
		t.Fatalf("strict resolution of a known root = %q, %v", got, err)
	}
}

// VROOLI_HOME, when set, is the runtime home; otherwise the contract decides.
func TestHostGovernedRootInputsHonoursRuntimeHomeOverride(t *testing.T) {
	override := t.TempDir()
	t.Setenv("VROOLI_HOME", override)
	in, err := HostGovernedRootInputs("")
	if err != nil {
		t.Fatalf("HostGovernedRootInputs: %v", err)
	}
	if in.RuntimeHome != override {
		t.Fatalf("RuntimeHome = %q, want %q", in.RuntimeHome, override)
	}
}
