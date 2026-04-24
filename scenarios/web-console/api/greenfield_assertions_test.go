package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGreenfield_NoRawSetSizeOutsideGatedPaths enforces the plan
// constraint that SIGWINCH-by-SetSize only fires via
// `maybeSIGWINCHRecovery` or the public `Resize` method. Any other
// call site on these files is a regression (see
// docs/plans/terminal-session-refactor-implementation-plan.md §10.4).
// Checks both session.go (Resize) and broadcast.go
// (maybeSIGWINCHRecovery) since the method was moved there in the
// decomposition phase.
func TestGreenfield_NoRawSetSizeOutsideGatedPaths(t *testing.T) {
	for _, file := range []string{"session.go", "broadcast.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		re := regexp.MustCompile(`s\.pty\.SetSize\(`)
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if !re.MatchString(line) {
				continue
			}
			// Allowed: inside maybeSIGWINCHRecovery or Resize.
			enclosing := findEnclosingFunc(lines, i)
			switch enclosing {
			case "maybeSIGWINCHRecovery", "Resize":
				// ok
			default:
				t.Errorf("%s:%d SetSize outside gated path (enclosing func=%q): %q",
					file, i+1, enclosing, strings.TrimSpace(line))
			}
		}
	}
}

// TestGreenfield_NoRawPtmxWriteOutsidePTYFiles enforces that raw
// ptmx.Write(...) calls only appear inside pty.go / pty_tmux.go.
// Anywhere else means a caller bypassed the PTY interface's kind-
// aware WriteInput — exactly the Bug A regression we refuse to ship
// again. See refactor plan §10.4 and §14.2.
func TestGreenfield_NoRawPtmxWriteOutsidePTYFiles(t *testing.T) {
	allowed := map[string]bool{
		"pty.go":      true,
		"pty_tmux.go": true,
	}
	re := regexp.MustCompile(`\bptmx\.Write\(`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		if allowed[f] {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if re.Match(b) {
			t.Errorf("%s calls ptmx.Write(...) — must route through PTY.WriteInput instead", f)
		}
	}
}

// TestGreenfield_PTYInterfaceHasNoLegacyWrite enforces the greenfield
// rule that the old PTY.Write method was deleted, not kept as a
// compat alias. Every call site must go through WriteInput with an
// explicit InputKind. Looks for the exact legacy signature in pty.go.
func TestGreenfield_PTYInterfaceHasNoLegacyWrite(t *testing.T) {
	b, err := os.ReadFile("pty.go")
	if err != nil {
		t.Fatalf("read pty.go: %v", err)
	}
	// The PTY interface body contains `Read(...)` and `WriteInput(...)`
	// but MUST NOT contain a raw `Write(p []byte) (int, error)`
	// method declaration (the legacy shape).
	legacy := regexp.MustCompile(`\n\s*Write\(p \[\]byte\) \(int, error\)`)
	if legacy.Match(b) {
		t.Errorf("pty.go still declares the legacy PTY.Write(p []byte) (int, error) method")
	}
}

// TestGreenfield_NoReferencesToRemovedPlans ensures the deleted plan
// filenames are not referenced from in-tree source files. Dangling
// references would indicate the deletion left the codebase in an
// inconsistent state. The comparison is case-sensitive and built at
// runtime from parts so THIS test's own source doesn't count as a
// reference.
func TestGreenfield_NoReferencesToRemovedPlans(t *testing.T) {
	// Assembled at runtime so this test file doesn't literally
	// contain the full filenames.
	removed := []string{
		"terminal-session-" + "rework" + "-implementation-plan",
		"terminal-session-" + "rework" + "-phase-2-implementation-plan",
		"persistent-terminal-" + "input-protection" + "-plan",
		"detachable-" + "sessions-implementation" + "-plan",
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		// Exclude this test itself (it contains the assembly, which
		// at the byte level matches).
		if f == "greenfield_assertions_test.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, name := range removed {
			if strings.Contains(string(b), name) {
				t.Errorf("%s references deleted plan %q", f, name)
			}
		}
	}
}

// findEnclosingFunc walks backward from lineIdx to find the most
// recent `func` line and returns the function name.
func findEnclosingFunc(lines []string, lineIdx int) string {
	re := regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)`)
	for i := lineIdx; i >= 0; i-- {
		if m := re.FindStringSubmatch(lines[i]); m != nil {
			return m[1]
		}
	}
	return ""
}

// TestGreenfield_PTYStateTrackerSingleOwner enforces that the alt-
// buffer tracker is only instantiated inside the Session struct;
// duplicate trackers in other files would imply split ownership.
func TestGreenfield_PTYStateTrackerSingleOwner(t *testing.T) {
	pat := regexp.MustCompile(`\bPTYStateTracker\b`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	var producing []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if pat.MatchString(string(b)) {
			producing = append(producing, f)
		}
	}
	// Expected: pty_state.go (definition) and session.go (the one
	// owner). Anything else is a regression.
	allowed := map[string]bool{"pty_state.go": true, "session.go": true}
	for _, f := range producing {
		if !allowed[f] {
			t.Errorf("%s references PTYStateTracker; only pty_state.go and session.go should", f)
		}
	}
}
