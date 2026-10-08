package hostbin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePrefersPath(t *testing.T) {
	defer restore()
	lookPath = func(name string) (string, error) {
		if name == "pnpm" {
			return "/usr/bin/pnpm", nil
		}
		return "", errors.New("not found")
	}
	got, ok := Resolve([]string{"yarn", "pnpm", "npm"})
	if !ok || got != "/usr/bin/pnpm" {
		t.Fatalf("Resolve = %q,%v; want resolved PATH executable", got, ok)
	}
}

func TestResolveFallsBackToUserBinDir(t *testing.T) {
	defer restore()
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "bats"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	lookPath = func(string) (string, error) { return "", errors.New("not on PATH") }
	userHomeDir = func() (string, error) { return home, nil }

	got, ok := Resolve([]string{"bats"})
	if !ok || got != filepath.Join(home, ".local", "bin", "bats") {
		t.Fatalf("Resolve = %q,%v; want resolved per-user executable", got, ok)
	}
}

func TestResolveFallsBackToWindowsExecutableExtension(t *testing.T) {
	defer restore()
	home := filepath.Join(t.TempDir(), "home with spaces")
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "pytest.exe"), []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	lookPath = func(string) (string, error) { return "", errors.New("not on PATH") }
	userHomeDir = func() (string, error) { return home, nil }

	got, ok := Resolve([]string{"pytest"})
	if !ok || got != filepath.Join(bin, "pytest.exe") {
		t.Fatalf("Resolve = %q,%v; want executable-extension fallback", got, ok)
	}
}

func TestResolveNoneAvailable(t *testing.T) {
	defer restore()
	lookPath = func(string) (string, error) { return "", errors.New("nope") }
	userHomeDir = func() (string, error) { return t.TempDir(), nil }
	if got, ok := Resolve([]string{"pnpm", "yarn"}); ok {
		t.Fatalf("Resolve = %q,true; want _,false", got)
	}
}

func restore() {
	lookPath = osLookPath
	userHomeDir = os.UserHomeDir
}

// osLookPath captured at init so restore() is independent of test order.
var osLookPath = lookPath
