package hygiene

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitRepoWithFiles builds a real git checkout containing files, and stages them
// so `git ls-files` sees them. The check reads the index, so a fixture that only
// writes to disk would prove nothing.
func gitRepoWithFiles(t *testing.T, files map[string][]byte) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	for rel, data := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	return root
}

func elfBytes() []byte {
	return append([]byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01}, make([]byte, 64)...)
}

func TestTrackedBinariesFlagsCommittedExecutables(t *testing.T) {
	root := gitRepoWithFiles(t, map[string][]byte{
		"scenarios/demo/api/main":    elfBytes(),
		"scenarios/demo/api/main.go": []byte("package main\n"),
		"README.md":                  []byte("# demo\n"),
	})

	report := Report{Root: root, Success: true}
	Service{}.checkTrackedBinaries(&report)

	finding, ok := findingCodes(report)["tracked_compiled_binary"]
	if !ok {
		t.Fatalf("expected tracked_compiled_binary finding, got %v", keys(findingCodes(report)))
	}
	if finding.Severity != SeverityError {
		t.Fatalf("tracked binary should be an error, got %q", finding.Severity)
	}
	if !containsString(finding.Locations, "scenarios/demo/api/main") {
		t.Fatalf("expected scenarios/demo/api/main, got %v", finding.Locations)
	}
	for _, notWanted := range []string{"scenarios/demo/api/main.go", "README.md"} {
		if containsString(finding.Locations, notWanted) {
			t.Fatalf("source file %s must not be flagged, got %v", notWanted, finding.Locations)
		}
	}
}

// An untracked binary is a normal build artifact, not a defect: the check must
// assert repository state, not working-tree contents.
func TestTrackedBinariesIgnoresUntrackedExecutables(t *testing.T) {
	root := gitRepoWithFiles(t, map[string][]byte{
		"scenarios/demo/api/main.go": []byte("package main\n"),
	})
	if err := os.WriteFile(filepath.Join(root, "scenarios", "demo", "api", "main"), elfBytes(), 0o755); err != nil {
		t.Fatal(err)
	}

	report := Report{Root: root, Success: true}
	Service{}.checkTrackedBinaries(&report)

	if _, ok := findingCodes(report)["tracked_compiled_binary"]; ok {
		t.Fatal("an untracked build output must not be reported")
	}
}

func TestTrackedBinariesDetectsWindowsAndMachO(t *testing.T) {
	root := gitRepoWithFiles(t, map[string][]byte{
		"tool.exe":       append([]byte{'M', 'Z', 0x90, 0x00}, make([]byte, 64)...),
		"scenarios/mach": append([]byte{0xcf, 0xfa, 0xed, 0xfe}, make([]byte, 64)...),
	})

	report := Report{Root: root, Success: true}
	Service{}.checkTrackedBinaries(&report)

	finding, ok := findingCodes(report)["tracked_compiled_binary"]
	if !ok {
		t.Fatalf("expected tracked_compiled_binary finding, got %v", keys(findingCodes(report)))
	}
	for _, want := range []string{"tool.exe", "scenarios/mach"} {
		if !containsString(finding.Locations, want) {
			t.Fatalf("expected %s in %v", want, finding.Locations)
		}
	}
}

func TestTrackedBinariesPassesOnCleanRepo(t *testing.T) {
	root := gitRepoWithFiles(t, map[string][]byte{
		"main.go":   []byte("package main\n"),
		"README.md": []byte("# clean\n"),
	})

	report := Report{Root: root, Success: true}
	Service{}.checkTrackedBinaries(&report)

	if _, ok := findingCodes(report)["tracked_compiled_binary"]; ok {
		t.Fatal("clean repo must not report a tracked binary")
	}
	var saw bool
	for _, c := range report.Checks {
		if c.Name == "tracked_binaries" {
			saw = true
			if !c.Passed {
				t.Fatalf("tracked_binaries check failed on a clean repo: %s", c.Message)
			}
		}
	}
	if !saw {
		t.Fatalf("expected a tracked_binaries check, got %+v", report.Checks)
	}
}

// git stores a symlink's target path, not the target's bytes, so a tracked
// symlink pointing at an (ignored) build output is not a tracked binary and must
// not be reported as one -- the `git rm --cached` remediation would target the
// wrong object.
func TestTrackedBinariesDoesNotFollowSymlinks(t *testing.T) {
	root := gitRepoWithFiles(t, map[string][]byte{
		"scenarios/demo/cli/keep.go": []byte("package main\n"),
	})
	realBinary := filepath.Join(root, "scenarios", "demo", "cli", "cli")
	if err := os.WriteFile(realBinary, elfBytes(), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "scenarios", "demo", "cli", "demo")
	if err := os.Symlink("cli", link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	cmd := exec.Command("git", "-C", root, "add", "scenarios/demo/cli/demo")
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add symlink: %v\n%s", err, out)
	}

	report := Report{Root: root, Success: true}
	Service{}.checkTrackedBinaries(&report)

	if f, ok := findingCodes(report)["tracked_compiled_binary"]; ok {
		t.Fatalf("symlink must not be reported as a tracked binary, got %v", f.Locations)
	}
}

// Outside a git checkout the check has nothing to assert and must not fail.
func TestTrackedBinariesSkipsOutsideGitRepo(t *testing.T) {
	report := Report{Root: t.TempDir(), Success: true}
	Service{}.checkTrackedBinaries(&report)

	if _, ok := findingCodes(report)["tracked_compiled_binary"]; ok {
		t.Fatal("non-git directory must not produce a finding")
	}
}
