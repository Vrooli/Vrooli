package resources

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Feature: a digest-correct artifact that cannot start is rejected before it is
// recorded as installed
//
//	As the acquisition path
//	I want an artifact's declared shared libraries resolved against this host
//	So that a target like the reranker CPU build, whose digest was correct and
//	whose libiomp5.so was absent, is never staged as if it worked.

// Scenario: a statically linked artifact has a satisfied closure.
func TestVerifyRuntimeClosureAcceptsAStaticBinary(t *testing.T) {
	// Given an artifact with no dynamic section
	binary := writeELFWithNeeded(t, t.TempDir())

	// When its runtime closure is verified
	verdict := VerifyRuntimeClosure(binary, nil)

	// Then it is satisfied, and the reason says why
	if verdict.State != ClosureOK {
		t.Fatalf("State = %q, want %q (verdict = %+v)", verdict.State, ClosureOK, verdict)
	}
	if len(verdict.Unresolved) != 0 {
		t.Fatalf("Unresolved = %v, want none", verdict.Unresolved)
	}
}

// Scenario: a declared library the host cannot provide is named.
//
// This is the reranker CPU target: libiomp5.so is absent from every default
// loader directory on this host, so the artifact installs clean and cannot run.
func TestVerifyRuntimeClosureRejectsAnUnresolvableDependency(t *testing.T) {
	// Given an ELF that declares a library no directory provides
	dir := t.TempDir()
	binary := writeELFWithNeeded(t, dir, "libiomp5.so", "libc.so.6")

	// When its runtime closure is verified
	verdict := VerifyRuntimeClosure(binary, nil)

	// Then the artifact is rejected and the missing library is named
	if verdict.State != ClosureUnresolved {
		t.Fatalf("State = %q, want %q (verdict = %+v)", verdict.State, ClosureUnresolved, verdict)
	}
	if !slices.Contains(verdict.Unresolved, "libiomp5.so") {
		t.Fatalf("Unresolved = %v, want it to name libiomp5.so", verdict.Unresolved)
	}
	// And every searched directory is reported, so the operator does not have
	// to reconstruct the loader's view by hand
	if len(verdict.Searched) == 0 {
		t.Fatal("Searched is empty; an unresolved verdict must say where it looked")
	}
	// And the resolvable library is not reported as missing
	if slices.Contains(verdict.Unresolved, "libc.so.6") {
		t.Fatalf("Unresolved = %v, want libc.so.6 to resolve on this host", verdict.Unresolved)
	}

	// And the typed error carries both, in one operator-readable line
	err := &RuntimeClosureError{Resource: "reranker", Artifact: binary, Verdict: verdict}
	if !errors.Is(err, ErrRuntimeClosure) {
		t.Fatal("RuntimeClosureError does not unwrap to ErrRuntimeClosure")
	}
	message := err.Error()
	for _, want := range []string{"reranker", "libiomp5.so", "searched", "The digest is correct"} {
		if !strings.Contains(message, want) {
			t.Fatalf("message = %q, want it to contain %q", message, want)
		}
	}
}

// Scenario: a dependency resolvable only through library_paths is accepted.
func TestVerifyRuntimeClosureAcceptsALibraryFromDeclaredPaths(t *testing.T) {
	// Given an ELF whose dependency exists only in a sibling directory
	dir := t.TempDir()
	binary := writeELFWithNeeded(t, dir, "libcustom.so.1")
	vendored := filepath.Join(dir, "vendor-libs")
	if err := os.MkdirAll(vendored, 0o755); err != nil {
		t.Fatalf("create vendored dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vendored, "libcustom.so.1"), []byte("shared object"), 0o644); err != nil {
		t.Fatalf("write vendored library: %v", err)
	}

	// When the closure is verified without that directory declared
	if verdict := VerifyRuntimeClosure(binary, nil); verdict.State != ClosureUnresolved {
		t.Fatalf("State = %q, want %q before the path is declared", verdict.State, ClosureUnresolved)
	}

	// Then declaring it as a relative library path satisfies the closure
	if verdict := VerifyRuntimeClosure(binary, []string{"vendor-libs"}); verdict.State != ClosureOK {
		t.Fatalf("State = %q, want %q once vendor-libs is declared (verdict = %+v)", verdict.State, ClosureOK, verdict)
	}
	// And an absolute declaration works the same way
	if verdict := VerifyRuntimeClosure(binary, []string{vendored}); verdict.State != ClosureOK {
		t.Fatalf("State = %q, want %q for an absolute library path", verdict.State, ClosureOK)
	}
}

// Scenario: a library beside the executable resolves without any declaration.
//
// This is how ollama ships: llama-server declares libllama.so.0 and friends,
// and they sit in the same directory. Rejecting that would reject a working
// artifact.
func TestVerifyRuntimeClosureFindsSiblingLibraries(t *testing.T) {
	// Given an ELF whose dependency sits beside it
	dir := t.TempDir()
	binary := writeELFWithNeeded(t, dir, "libsibling.so.0")
	if err := os.WriteFile(filepath.Join(dir, "libsibling.so.0"), []byte("shared object"), 0o644); err != nil {
		t.Fatalf("write sibling library: %v", err)
	}

	// When the closure is verified with nothing declared
	verdict := VerifyRuntimeClosure(binary, nil)

	// Then it is satisfied, because the loader would find it there too
	if verdict.State != ClosureOK {
		t.Fatalf("State = %q, want %q (verdict = %+v)", verdict.State, ClosureOK, verdict)
	}
}

// Scenario: an unreadable format is unknown, never a rejection.
//
// A Windows install must not be blocked by a check that cannot read PE import
// tables, and a missing file is a different failure than an unsatisfiable one.
func TestVerifyRuntimeClosureReportsUnknownRatherThanBlocking(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		scenario   string
		path       string
		setup      func(string)
		wantReason string
	}{
		{
			scenario:   "Given a file that is not an executable format, Then the verdict is unknown",
			path:       filepath.Join(dir, "notabinary"),
			setup:      func(p string) { _ = os.WriteFile(p, []byte("MZ this is not really a PE"), 0o755) },
			wantReason: "not an ELF or Mach-O executable",
		},
		{
			scenario:   "Given a missing artifact, Then the verdict is unknown",
			path:       filepath.Join(dir, "absent"),
			setup:      func(string) {},
			wantReason: "cannot read the staged artifact",
		},
		{
			scenario:   "Given a directory, Then the verdict is unknown",
			path:       filepath.Join(dir, "atree"),
			setup:      func(p string) { _ = os.MkdirAll(p, 0o755) },
			wantReason: "is a directory",
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given the artifact in that state
			tc.setup(tc.path)

			// When its closure is verified
			verdict := VerifyRuntimeClosure(tc.path, nil)

			// Then it is unknown with a named reason, and never unresolved
			if verdict.State != ClosureUnknown {
				t.Fatalf("State = %q, want %q", verdict.State, ClosureUnknown)
			}
			if !strings.Contains(verdict.Reason, tc.wantReason) {
				t.Fatalf("Reason = %q, want it to contain %q", verdict.Reason, tc.wantReason)
			}
		})
	}
}

// Scenario: the search order puts the artifact's own directory first.
func TestClosureSearchOrderPrefersTheArtifactsOwnDirectories(t *testing.T) {
	// Given an artifact and a declared library path
	dirs := closureSearchDirs("/opt/vrooli/artifacts/reranker/1.7.4/reranker", []string{"cuda-libs"})

	// Then the artifact's own directory is searched first, then its lib
	// subdirectories, then the declared path, then the host defaults
	want := []string{
		"/opt/vrooli/artifacts/reranker/1.7.4",
		"/opt/vrooli/artifacts/reranker/1.7.4/lib",
		"/opt/vrooli/artifacts/reranker/1.7.4/lib64",
		"/opt/vrooli/artifacts/reranker/1.7.4/cuda-libs",
	}
	if len(dirs) < len(want) {
		t.Fatalf("search dirs = %v, want at least %v", dirs, want)
	}
	for i, expected := range want {
		if dirs[i] != expected {
			t.Fatalf("search dir %d = %q, want %q (full order %v)", i, dirs[i], expected, dirs)
		}
	}
	// And nothing is searched twice
	seen := map[string]bool{}
	for _, dir := range dirs {
		if seen[dir] {
			t.Fatalf("search order repeats %q: %v", dir, dirs)
		}
		seen[dir] = true
	}
}
