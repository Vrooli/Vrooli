package langrecover

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	runtimeBin := "/runtime/home/bin"
	for goos, want := range map[string][]string{
		"linux":   {"/usr/local/go/bin", "/home/op/go/bin", "/home/op/.cargo/bin", "/home/op/.local/bin", runtimeBin},
		"darwin":  {"/usr/local/go/bin", "/opt/homebrew/bin", "/home/op/go/bin", runtimeBin},
		"windows": {filepath.Join(home, "AppData", "Local", "Microsoft", "WinGet", "Links"), runtimeBin},
	} {
		entries := DefaultPathEntries(goos, home, runtimeBin)
		for _, w := range want {
			if !contains(entries, filepath.FromSlash(w)) {
				t.Errorf("%s table lacks %s: %v", goos, w, entries)
			}
		}
	}
	for _, entry := range DefaultPathEntries(runtime.GOOS, "", "") {
		if entry == "" {
			t.Fatal("an empty home must not produce empty entries")
		}
	}
}

// The runtime-home bin directory belongs to the repository contract, which
// langrecover cannot read: the table carries exactly what the caller resolved
// and never invents a home-relative Vrooli directory of its own.
func TestDefaultPathEntriesTakeTheRuntimeBinFromTheCaller(t *testing.T) {
	home := filepath.FromSlash("/home/op")
	relocated := filepath.FromSlash("/relocated/runtime/bin")
	for _, goos := range []string{"linux", "darwin", "windows"} {
		entries := DefaultPathEntries(goos, home, relocated)
		if entries[len(entries)-1] != relocated {
			t.Errorf("%s: runtime bin %q is not searched last: %v", goos, relocated, entries)
		}
		for _, entry := range DefaultPathEntries(goos, home, "") {
			if strings.Contains(entry, ".vrooli") {
				t.Errorf("%s: table hand-built a runtime-home entry %q", goos, entry)
			}
		}
	}
}

func TestNewRunnerResolvesThroughTheTable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "langrecover-fixture-tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\necho ran-in \"$(pwd)\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	out, err := NewRunner([]string{dir})(context.Background(), work, "langrecover-fixture-tool")
	if err != nil {
		t.Fatalf("runner: %v (%s)", err, out)
	}
	if got := strings.TrimSpace(string(out)); !strings.HasSuffix(got, filepath.Base(work)) {
		t.Fatalf("runner output %q; want it to run in %s", got, work)
	}
	if _, err := NewRunner(nil)(context.Background(), work, "langrecover-absent-tool"); !errors.Is(err, ErrToolMissing) {
		t.Fatalf("missing tool err = %v, want ErrToolMissing", err)
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
