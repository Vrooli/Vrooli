package runtimesupervisor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// A service unit records an absolute path and keeps running it across restarts
// and reboots. The process performing an install is routinely NOT the installed
// binary — it is a build output, a `go run` temp file, or a one-off build — and
// pinning the fleet's supervisor to a path that every build rewrites is how it
// ends up running an image nobody chose. This happened on a live host: an
// install run from .vrooli/build/vrooli left the unit pointing there.
func TestExecutablePathPrefersTheInstalledCLIOverTheInstallingBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "permission-bit fixture is POSIX-specific")
	}
	home := t.TempDir()
	installed := filepath.Join(home, ".vrooli", "bin", cliBinaryName)
	if err := os.MkdirAll(filepath.Dir(installed), 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	if err := os.WriteFile(installed, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write installed cli: %v", err)
	}

	buildOutput := filepath.Join(t.TempDir(), ".vrooli", "build", "vrooli")
	got, canonical, err := ExecutablePath(home, buildOutput)
	if err != nil {
		t.Fatalf("ExecutablePath: %v", err)
	}
	if got != installed {
		t.Fatalf("executable = %q, want the installed CLI %q", got, installed)
	}
	if !canonical {
		t.Fatal("canonical = false, want true for the installed CLI")
	}
}

// Refusing here would break a first install, which by definition happens before
// anything is installed. The path is used, but reported as not canonical so the
// operator is told rather than left to discover it from /proc.
func TestExecutablePathFallsBackAndReportsNonCanonical(t *testing.T) {
	home := t.TempDir() // no installed CLI
	requested := filepath.Join(t.TempDir(), "vrooli")

	got, canonical, err := ExecutablePath(home, requested)
	if err != nil {
		t.Fatalf("ExecutablePath: %v", err)
	}
	if got != requested {
		t.Fatalf("executable = %q, want the requesting binary %q", got, requested)
	}
	if canonical {
		t.Fatal("canonical = true, but nothing is installed at the canonical path")
	}
}

// A directory or a non-executable file at the canonical path is not a CLI;
// treating either as one would pin the unit to something that cannot start.
func TestExecutablePathIgnoresUnusableCanonicalCandidates(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "permission-bit fixture is POSIX-specific")
	}
	for name, prepare := range map[string]func(t *testing.T, path string){
		"directory": func(t *testing.T, path string) {
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatalf("create dir: %v", err)
			}
		},
		"not executable": func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
				t.Fatalf("write file: %v", err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			candidate := filepath.Join(home, ".vrooli", "bin", cliBinaryName)
			if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
				t.Fatalf("create bin dir: %v", err)
			}
			prepare(t, candidate)

			requested := filepath.Join(t.TempDir(), "vrooli")
			got, canonical, err := ExecutablePath(home, requested)
			if err != nil {
				t.Fatalf("ExecutablePath: %v", err)
			}
			if got != requested || canonical {
				t.Fatalf("executable = %q (canonical=%v), want the fallback %q", got, canonical, requested)
			}
		})
	}
}
