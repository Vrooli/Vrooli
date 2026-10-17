package cliruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBinaryPrefersCanonicalRuntimeHomeOverPATH(t *testing.T) {
	home := t.TempDir()
	canonicalDir := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(canonicalDir, 0o755); err != nil {
		t.Fatalf("mkdir canonical bin: %v", err)
	}
	canonical := filepath.Join(canonicalDir, "fixture-cli")
	if err := os.WriteFile(canonical, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write canonical CLI: %v", err)
	}

	staleDir := t.TempDir()
	stale := filepath.Join(staleDir, "fixture-cli")
	if err := os.WriteFile(stale, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write stale CLI: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", staleDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if got := ResolveBinary("fixture-cli", ""); got != canonical {
		t.Fatalf("ResolveBinary selected %q, want canonical %q", got, canonical)
	}
}

func TestResolveBinaryKeepsExplicitEnvironmentOverride(t *testing.T) {
	home := t.TempDir()
	canonicalDir := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(canonicalDir, 0o755); err != nil {
		t.Fatalf("mkdir canonical bin: %v", err)
	}
	canonical := filepath.Join(canonicalDir, "fixture-cli")
	if err := os.WriteFile(canonical, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write canonical CLI: %v", err)
	}
	override := filepath.Join(home, "override-cli")
	if err := os.WriteFile(override, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write override CLI: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("CLIRUNTIME_FIXTURE", override)

	if got := ResolveBinary("fixture-cli", "CLIRUNTIME_FIXTURE"); got != override {
		t.Fatalf("ResolveBinary ignored explicit override %q, got %q", override, got)
	}
}

func TestResolveBinaryFindsCanonicalPathWhenHomeIsSandboxed(t *testing.T) {
	canonicalRoot := t.TempDir()
	canonicalDir := filepath.Join(canonicalRoot, ".vrooli", "bin")
	if err := os.MkdirAll(canonicalDir, 0o755); err != nil {
		t.Fatalf("mkdir canonical bin: %v", err)
	}
	canonical := filepath.Join(canonicalDir, "fixture-cli")
	if err := os.WriteFile(canonical, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write canonical CLI: %v", err)
	}
	staleDir := t.TempDir()
	stale := filepath.Join(staleDir, "fixture-cli")
	if err := os.WriteFile(stale, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write stale CLI: %v", err)
	}

	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", staleDir+string(os.PathListSeparator)+canonicalDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if got := ResolveBinary("fixture-cli", ""); got != canonical {
		t.Fatalf("ResolveBinary selected %q, want canonical PATH entry %q", got, canonical)
	}
}
