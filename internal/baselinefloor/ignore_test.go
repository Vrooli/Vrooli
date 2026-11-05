package baselinefloor

import (
	"os"
	"path/filepath"
	"testing"
)

// repoIgnoreFixture is a repository whose .gitignore files exercise the rules
// a restore point relies on: unanchored names, directory-only patterns,
// anchored paths, "**", negation that re-includes a tracked seed, and a
// scenario-level .gitignore that names its own binaries.
func repoIgnoreFixture(t *testing.T) (repo, scenarioDir string) {
	t.Helper()
	repo = t.TempDir()
	writeFile(t, repo, ".gitignore", `
# runtime data and build output
*.log
data/
bin/
/logs/
build/
!scenarios/**/api/build/
!scenarios/**/api/build/**
!scenarios/keeper/data/
.vrooli/*.lock
`, 0o644)
	scenarioDir = filepath.Join(repo, "scenarios", "demo")
	writeFile(t, scenarioDir, ".gitignore", "/api/demo-api\ncli/demo\n", 0o644)
	return repo, scenarioDir
}

func TestIgnoreMatcherAppliesRepositoryGitignoreRules(t *testing.T) {
	repo, scenarioDir := repoIgnoreFixture(t)
	m := NewIgnoreMatcher(IgnoreSpec{RepoRoot: repo, TreeRoot: scenarioDir, Patterns: FloorIgnorePatterns()})

	cases := []struct {
		rel     string
		isDir   bool
		ignored bool
	}{
		{"api/main.go", false, false},
		{"api/demo-api", false, true},          // scenario .gitignore, anchored
		{"ui/api/demo-api", false, false},      // anchored pattern does not float
		{"cli/demo", false, true},              // middle slash anchors to the scenario
		{"server.log", false, true},            // root *.log at any depth
		{"data", true, true},                   // data/ is directory-only
		{"data", false, false},                 // a file named data is kept
		{"bin", true, true},                    // bin/ at any depth
		{"logs", true, false},                  // /logs/ is anchored at the repo root
		{"tools/build", true, true},            // build/ ignored ...
		{"api/build", true, false},             // ... but re-included under api/
		{".vrooli/service.json", false, false}, // config is source
		{".vrooli/secrets.json", false, true},  // floor: secrets
		{".vrooli/state", true, true},          // floor: runtime state
		{".vrooli/x.lock", false, true},        // floor: locks (root .gitignore anchors .vrooli/*.lock to the repo root)
		{"api/store.db-wal", false, true},      // floor: database journals
		{"api/store.sqlite", false, true},      // floor: databases
	}
	for _, tc := range cases {
		if got := m.Ignored(tc.rel, tc.isDir); got != tc.ignored {
			t.Errorf("Ignored(%q, dir=%t) = %t, want %t", tc.rel, tc.isDir, got, tc.ignored)
		}
	}

	keeper := NewIgnoreMatcher(IgnoreSpec{RepoRoot: repo, TreeRoot: filepath.Join(repo, "scenarios", "keeper")})
	if keeper.Ignored("data", true) {
		t.Error("a negated data/ directory must be re-included for its scenario")
	}
}

func TestIgnoreMatcherNestedGitignoreAndDoubleStar(t *testing.T) {
	tree := t.TempDir()
	writeFile(t, tree, "ui/.gitignore", "*.tsbuildinfo\n/out/\n", 0o644)
	m := NewIgnoreMatcher(IgnoreSpec{TreeRoot: tree, Patterns: []string{"**/cache/**", "docs/**/draft-*.md"}})
	cases := map[string]bool{
		"ui/app.tsbuildinfo":       true,
		"app.tsbuildinfo":          false, // nested rule applies below ui/ only
		"ui/out":                   true,
		"ui/src/out":               false,
		"a/cache/x/y.bin":          true,
		"a/cache":                  false, // "x/**" matches inside x, not x
		"docs/draft-1.md":          true,  // "**" spans zero directories
		"docs/2026/q3/draft-2.md":  true,
		"docs/2026/q3/final-2.md":  false,
		"docs-other/draft-3.md":    false,
		"ui/src/components/a.tsx":  false,
		"ui/src/components/a.json": false,
	}
	for rel, want := range cases {
		isDir := rel == "ui/out" || rel == "ui/src/out" || rel == "a/cache"
		if got := m.Ignored(rel, isDir); got != want {
			t.Errorf("Ignored(%q) = %t, want %t", rel, got, want)
		}
	}
}

// TestCaptureHonorsRepositoryIgnoreRules is the restore-point scope
// regression (O16): a capture keeps source and the scenario's .vrooli
// configuration, and leaves out runtime data, logs, binaries and .vrooli
// runtime state.
func TestCaptureHonorsRepositoryIgnoreRules(t *testing.T) {
	repo, scenarioDir := repoIgnoreFixture(t)
	writeFile(t, scenarioDir, "api/main.go", "package main", 0o644)
	writeFile(t, scenarioDir, "api/demo-api", "ELF", 0o755)
	writeFile(t, scenarioDir, "api/build/keep.go", "package build", 0o644)
	writeFile(t, scenarioDir, "data/demo.db", "live data", 0o644)
	writeFile(t, scenarioDir, "data/demo.db-wal", "live wal", 0o644)
	writeFile(t, scenarioDir, "server.log", "log", 0o644)
	writeFile(t, scenarioDir, ".vrooli/service.json", `{"service":{"name":"demo"}}`, 0o644)
	writeFile(t, scenarioDir, ".vrooli/testing.json", `{}`, 0o644)
	writeFile(t, scenarioDir, ".vrooli/secrets.json", `{"token":"x"}`, 0o600)
	writeFile(t, scenarioDir, ".vrooli/state/run.ndjson", "{}", 0o644)

	rp := filepath.Join(t.TempDir(), "restore-point")
	matcher := NewIgnoreMatcher(IgnoreSpec{RepoRoot: repo, TreeRoot: scenarioDir, Patterns: FloorIgnorePatterns()})
	if _, err := Capture(scenarioDir, rp, &CopyOptions{Ignore: matcher}); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	for _, kept := range []string{"api/main.go", "api/build/keep.go", ".vrooli/service.json", ".vrooli/testing.json", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(rp, kept)); err != nil {
			t.Errorf("%s must be in the restore point: %v", kept, err)
		}
	}
	for _, dropped := range []string{"api/demo-api", "data", "server.log", ".vrooli/secrets.json", ".vrooli/state"} {
		if _, err := os.Stat(filepath.Join(rp, dropped)); !os.IsNotExist(err) {
			t.Errorf("%s must not be in the restore point (err=%v)", dropped, err)
		}
	}
}

// A restore point captured before runtime data was excluded must not overlay
// its stale database onto the working tree on abandon.
func TestRestoreDoesNotOverlayStaleRuntimeData(t *testing.T) {
	repo, scenarioDir := repoIgnoreFixture(t)
	writeFile(t, scenarioDir, "api/main.go", "package main // edited", 0o644)
	writeFile(t, scenarioDir, "data/demo.db", "today's data", 0o644)

	legacy := t.TempDir()
	writeFile(t, legacy, "api/main.go", "package main", 0o644)
	writeFile(t, legacy, "data/demo.db", "last week's data", 0o644)

	matcher := NewIgnoreMatcher(IgnoreSpec{RepoRoot: repo, TreeRoot: scenarioDir, Patterns: FloorIgnorePatterns()})
	if _, err := Restore(legacy, scenarioDir, &CopyOptions{Ignore: matcher}); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := readFile(t, filepath.Join(scenarioDir, "api/main.go")); got != "package main" {
		t.Errorf("source not rolled back: %q", got)
	}
	if got := readFile(t, filepath.Join(scenarioDir, "data/demo.db")); got != "today's data" {
		t.Errorf("restore overlaid stale runtime data: %q", got)
	}
}
