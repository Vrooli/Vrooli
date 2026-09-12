package langrecover

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func noLookPath(string) (string, error) { return "", errors.New("not on PATH") }

// The unit's PATH omitted /usr/local/go/bin on the 2026-09 host; the table
// must find a toolchain PATH cannot.
func TestFindToolFallsBackToTableWhenPathMisses(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "go")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := FindTool("go", noLookPath, []string{filepath.Join(dir, "absent"), dir})
	if err != nil || got != tool {
		t.Fatalf("FindTool = %q, %v; want %q", got, err, tool)
	}
}

func TestFindToolPrefersPath(t *testing.T) {
	got, err := FindTool("go", func(string) (string, error) { return "/from/path/go", nil }, []string{t.TempDir()})
	if err != nil || got != "/from/path/go" {
		t.Fatalf("FindTool = %q, %v; want the PATH hit", got, err)
	}
}

func TestFindToolNamesEverySearchedLocation(t *testing.T) {
	dir := t.TempDir()
	_, err := FindTool("go", noLookPath, []string{dir})
	if !errors.Is(err, ErrToolMissing) {
		t.Fatalf("err = %v, want ErrToolMissing", err)
	}
	var missing *ToolMissingError
	if !errors.As(err, &missing) || missing.Name != "go" || len(missing.Searched) != 2 {
		t.Fatalf("error must name the tool and every location: %v", err)
	}
}

func TestDefaultPathEntriesCoverTheKnownToolchainHomes(t *testing.T) {
	home := "/home/op"
	for goos, want := range map[string][]string{
		"linux":   {"/usr/local/go/bin", "/home/op/go/bin", "/home/op/.cargo/bin", "/home/op/.local/bin", "/home/op/.vrooli/bin"},
		"darwin":  {"/usr/local/go/bin", "/opt/homebrew/bin", "/home/op/go/bin", "/home/op/.vrooli/bin"},
		"windows": {filepath.Join(home, "AppData", "Local", "Microsoft", "WinGet", "Links")},
	} {
		entries := DefaultPathEntries(goos, home)
		for _, w := range want {
			if !contains(entries, filepath.FromSlash(w)) {
				t.Errorf("%s table lacks %s: %v", goos, w, entries)
			}
		}
	}
	for _, entry := range DefaultPathEntries(runtime.GOOS, "") {
		if entry == "" {
			t.Fatal("an empty home must not produce empty entries")
		}
	}
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
