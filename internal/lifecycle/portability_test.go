package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLifecycleExecutorHasNoShellOrProcessTablePath(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bBashCommand\b`),
		regexp.MustCompile(`(?i)exec\.Command(?:Context)?\([^\n]*(?:"bash"|"sh"\s*,\s*"-c")`),
		regexp.MustCompile(`(?i)exec\.Command(?:Context)?\([^\n]*"pkill"`),
		regexp.MustCompile(`(?i)exec\.Command(?:Context)?\([^\n]*"ps"`),
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Clean(name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, pattern := range patterns {
			if pattern.Match(raw) {
				t.Errorf("%s reintroduced forbidden executor path %s", name, pattern)
			}
		}
	}
}

// posixOnlyScriptTokens are constructs a package script cannot rely on. npm and
// pnpm run scripts through `cmd.exe` on Windows, which has no command
// substitution, no `VAR=value cmd` prefix, no `${VAR:-default}` expansion, and
// none of the coreutils these reach for. Every one of them has a portable
// equivalent: a Node script under the package's own scripts/ directory, or a
// value resolved in vite.config.ts where the build already reads its
// environment.
var posixOnlyScriptTokens = []struct{ token, why string }{
	{"$(", "command substitution"},
	{"`", "backtick substitution"},
	{"${", "shell parameter expansion"},
	{"bash -c", "explicit bash invocation"},
	{"sh -c", "explicit sh invocation"},
	{"[ ", "test builtin"},
	{"rm -rf", "coreutils rm"},
	{"rm -f", "coreutils rm"},
	{"mkdir -p", "coreutils mkdir"},
	{"cp -", "coreutils cp"},
	{">/dev/null", "POSIX device path"},
	{"2>&1", "POSIX fd redirection"},
	{".sh", "shell script"},
	{"zip ", "zip binary (absent on Windows)"},
	{"|| true", "shell control operator"},
}

// TestPackageScriptsAreShellFree walks every source package.json under
// scenarios/, packages/, and templates/ and asserts two invariants.
//
//  1. No script depends on a POSIX shell. Package scripts are the one build
//     surface the lifecycle hands to a third-party runner, so a shell construct
//     here is unrunnable on Windows no matter how portable the Go around it is.
//  2. Where a UI declares `build:profile`, it is `build` plus the mode flag.
//     The lifecycle runs the profile script verbatim when
//     VROOLI_BUILD_MODE=profile, so any other difference silently produces a
//     perf bundle built by different steps than the default one.
//
// Generated trees (dist/, bin/, data/, node_modules/) are excluded: they are
// build output, and the source they are generated from is covered here.
//
// The walk asserts coverage — it reports how many manifests it read and fails
// if it read none — rather than pinning a manifest count, which would drift.
func TestPackageScriptsAreShellFree(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	modeFlag := regexp.MustCompile(`\s*--mode\s+profile`)

	skipDir := map[string]bool{
		"node_modules": true, "dist": true, "bin": true, "data": true,
		".git": true, "coverage": true, "build": true,
	}

	manifests, profiles := 0, 0
	for _, tree := range []string{"scenarios", "packages", "templates"} {
		root := filepath.Join(repoRoot, tree)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil //nolint:nilerr // an unreadable subtree is not this test's subject
			}
			if entry.IsDir() {
				if skipDir[entry.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Name() != "package.json" {
				return nil
			}
			raw, readErr := os.ReadFile(filepath.Clean(path))
			if readErr != nil {
				return nil //nolint:nilerr // ditto
			}
			rel, _ := filepath.Rel(repoRoot, path)

			var pkg struct {
				Scripts map[string]string `json:"scripts"`
			}
			if jsonErr := json.Unmarshal(raw, &pkg); jsonErr != nil {
				t.Errorf("%s: unparseable package.json: %v", rel, jsonErr)
				return nil
			}
			manifests++

			for name, script := range pkg.Scripts {
				for _, banned := range posixOnlyScriptTokens {
					if strings.Contains(script, banned.token) {
						t.Errorf("%s: script %q uses %s (%q) — package scripts run through cmd.exe on Windows; move this into a Node script or the build config\n    %s",
							rel, name, banned.why, banned.token, script)
					}
				}
			}

			build, hasBuild := pkg.Scripts["build"]
			profile, hasProfile := pkg.Scripts["build:profile"]
			if hasBuild && hasProfile {
				profiles++
				if got := strings.TrimSpace(modeFlag.ReplaceAllString(profile, "")); got != strings.TrimSpace(build) {
					t.Errorf("%s: build:profile is not build plus --mode profile\n  build:         %s\n  build:profile: %s",
						rel, build, profile)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", tree, err)
		}
	}

	if manifests == 0 {
		t.Fatal("no package.json discovered — the walk asserted nothing")
	}
	if profiles == 0 {
		t.Error("no manifest declared build:profile — the channel-parity invariant asserted nothing")
	}
	t.Logf("covered %d package manifests (%d declare a build:profile channel)", manifests, profiles)
}
