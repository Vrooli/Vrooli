package hostpaths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// TestResolve_FindsTheTempRoot asserts the one root that must exist on every
// host is found. This is the root whose absence caused the incident: the temp
// provider was wired with an empty list and therefore reclaimed nothing while
// /tmp grew to 70 GB.
func TestResolve_FindsTheTempRoot(t *testing.T) {
	roots := Resolve()

	if len(roots.Tmp) == 0 {
		t.Fatal("no temp root resolved; every supported platform has one")
	}
	if got := roots.Tmp[0]; got != filepath.Clean(os.TempDir()) {
		t.Errorf("temp root = %q, want %q", got, filepath.Clean(os.TempDir()))
	}
}

// TestGoBuildCacheRoot_HonoursGOCACHE asserts an operator who relocated their
// build cache gets that location cleaned rather than silently skipped.
func TestGoBuildCacheRoot_HonoursGOCACHE(t *testing.T) {
	relocated := t.TempDir()
	t.Setenv("GOCACHE", relocated)

	if got := goBuildCacheRoot(); got != relocated {
		t.Errorf("goBuildCacheRoot() = %q, want the GOCACHE value %q", got, relocated)
	}
}

// TestGoBuildCacheRoot_TreatsOffAsNoRoot asserts the sentinel value is not
// mistaken for a directory name. GOCACHE=off means "no cache"; treating it as a
// path would produce a root named "off" relative to nothing.
func TestGoBuildCacheRoot_TreatsOffAsNoRoot(t *testing.T) {
	for _, value := range []string{"off", "OFF", "Off"} {
		t.Setenv("GOCACHE", value)
		if got := goBuildCacheRoot(); got != "" {
			t.Errorf("GOCACHE=%s resolved to %q, want no root", value, got)
		}
	}
}

// TestGoBuildCacheRoot_FallsBackToUserCacheDir asserts the default matches
// Go's own, which os.UserCacheDir already resolves per-OS.
func TestGoBuildCacheRoot_FallsBackToUserCacheDir(t *testing.T) {
	t.Setenv("GOCACHE", "")

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Skipf("no user cache dir on this host: %v", err)
	}
	if got, want := goBuildCacheRoot(), filepath.Join(cacheDir, "go-build"); got != want {
		t.Errorf("goBuildCacheRoot() = %q, want %q", got, want)
	}
}

// TestPlaywrightCacheRoot_TreatsZeroAsNoRoot asserts Playwright's own sentinel
// is respected. PLAYWRIGHT_BROWSERS_PATH=0 means "install beside the package",
// not "use the directory named 0".
func TestPlaywrightCacheRoot_TreatsZeroAsNoRoot(t *testing.T) {
	t.Setenv("PLAYWRIGHT_BROWSERS_PATH", "0")

	if got := playwrightCacheRoot(); got != "" {
		t.Errorf("PLAYWRIGHT_BROWSERS_PATH=0 resolved to %q, want no root", got)
	}
}

func TestPlaywrightCacheRoot_HonoursExplicitPath(t *testing.T) {
	relocated := t.TempDir()
	t.Setenv("PLAYWRIGHT_BROWSERS_PATH", relocated)

	if got := playwrightCacheRoot(); got != relocated {
		t.Errorf("playwrightCacheRoot() = %q, want %q", got, relocated)
	}
}

// TestExisting_FiltersUnusableCandidates asserts the root list only ever
// contains absolute, present directories.
//
// A missing directory is an ordinary condition — a host with no Playwright
// install has no Playwright cache — and must not reach a provider, where it
// would surface as a walk error rather than as "nothing to clean".
func TestExisting_FiltersUnusableCandidates(t *testing.T) {
	real := t.TempDir()

	fileNotDir := filepath.Join(real, "a-file")
	if err := os.WriteFile(fileNotDir, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := existing(
		real,
		real,                               // duplicate
		filepath.Join(real, "not-present"), // missing
		fileNotDir,                         // a file, not a directory
		"relative/path",                    // not absolute
		"",                                 // empty
		"   ",                              // whitespace
	)

	if len(got) != 1 || got[0] != real {
		t.Fatalf("existing() = %v, want exactly [%s]", got, real)
	}
}

func TestExisting_ReturnsNilWhenNothingUsable(t *testing.T) {
	if got := existing("", "relative", filepath.Join(t.TempDir(), "missing")); got != nil {
		t.Errorf("existing() = %v, want nil", got)
	}
}

// TestTrashRoots_MatchesPlatformConvention asserts the one genuinely
// OS-divergent root resolves to the right convention per platform.
func TestTrashRoots_MatchesPlatformConvention(t *testing.T) {
	roots := trashRoots()

	switch runtime.GOOS {
	case "windows":
		// The Recycle Bin is not a directory that may be emptied by path.
		if roots != nil {
			t.Errorf("trashRoots() = %v on windows, want nil", roots)
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("no home dir: %v", err)
		}
		if len(roots) != 1 || roots[0] != filepath.Join(home, ".Trash") {
			t.Errorf("trashRoots() = %v, want [~/.Trash]", roots)
		}
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("no home dir: %v", err)
		}
		base := filepath.Join(home, ".local", "share", "Trash")
		want := []string{filepath.Join(base, "files"), filepath.Join(base, "info")}
		if len(roots) != 2 || roots[0] != want[0] || roots[1] != want[1] {
			t.Errorf("trashRoots() = %v, want %v", roots, want)
		}
	}
}

// TestTrashRoots_HonoursXDGDataHome asserts the freedesktop override is
// respected on the platforms that follow the spec.
func TestTrashRoots_HonoursXDGDataHome(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		repocontracttest.SkipPlatform(t, "XDG_DATA_HOME is not the trash convention on this platform")
	}
	relocated := t.TempDir()
	t.Setenv("XDG_DATA_HOME", relocated)

	roots := trashRoots()
	base := filepath.Join(relocated, "Trash")
	want := []string{filepath.Join(base, "files"), filepath.Join(base, "info")}
	if len(roots) != 2 || roots[0] != want[0] || roots[1] != want[1] {
		t.Errorf("trashRoots() = %v, want %v", roots, want)
	}
}
