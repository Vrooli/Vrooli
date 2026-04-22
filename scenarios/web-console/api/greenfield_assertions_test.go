package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGreenfield_NoRawSetSizeOutsideGatedPaths enforces the plan
// constraint that SIGWINCH-by-SetSize in session.go only fires via
// `maybeSIGWINCHRecovery` or the public `Resize` method. Any other
// call site is a regression (see
// docs/plans/terminal-session-rework-implementation-plan.md §10.5).
func TestGreenfield_NoRawSetSizeOutsideGatedPaths(t *testing.T) {
	data, err := os.ReadFile("session.go")
	if err != nil {
		t.Fatalf("read session.go: %v", err)
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
			t.Errorf("session.go:%d SetSize outside gated path (enclosing func=%q): %q",
				i+1, enclosing, strings.TrimSpace(line))
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
