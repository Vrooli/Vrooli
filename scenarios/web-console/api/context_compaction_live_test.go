//go:build umlive

package main

// Live check of the idle-compaction path against a real Claude Code session in
// a tmux session named um-p6-*. Run only on request:
//
//	UM_LIVE=1 go test -tags umlive -run TestLiveCompaction -v -timeout 15m .
//
// It sends 3 real model requests at most: one prompt, one deliberately long
// prompt that must NOT be compacted while busy, and the /compact itself. The
// driver reads the real tmux pane for activity, so this exercises the
// manager's decisions, the typed keystrokes and the transcript confirmation
// against the real harness; it does not exercise Web Console's own activity
// detector or session objects.

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"web-console/session"
	"web-console/terminal"
)

// workingPane matches Claude's busy line: a spinner glyph, a gerund with an
// ellipsis, or the "esc to interrupt" hint (which wraps in narrow panes).
var workingPane = regexp.MustCompile(`(?m)esc to interrupt|^\s*\S\s+\w+…|\(\d+s\b`)

type tmuxDriver struct {
	name, transcriptDir string
	lastHash            string
	lastChange          time.Time
	typed               []string
}

func tmuxRun(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("tmux", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("tmux %v: %v %s", args, err, out)
	}
	return string(out)
}

func (d *tmuxDriver) capture() string {
	out, _ := exec.Command("tmux", "capture-pane", "-p", "-t", d.name).Output()
	text := string(out)
	sum := fmt.Sprintf("%x", sha1.Sum(out))
	if sum != d.lastHash {
		d.lastHash, d.lastChange = sum, time.Now()
	}
	return text
}

func (d *tmuxDriver) Sessions(context.Context) []compactSession {
	var newest string
	var at time.Time
	entries, _ := os.ReadDir(d.transcriptDir)
	for _, e := range entries {
		if info, err := e.Info(); err == nil && strings.HasSuffix(e.Name(), ".jsonl") && info.ModTime().After(at) {
			newest, at = filepath.Join(d.transcriptDir, e.Name()), info.ModTime()
		}
	}
	return []compactSession{{ID: d.name, Label: d.name, Harness: "claude", Transcript: newest}}
}

func (d *tmuxDriver) Activity(string) (session.Activity, bool) {
	text := d.capture()
	state := session.ActivityIdle
	if workingPane.MatchString(text) {
		state = session.ActivityWorking
	}
	return session.Activity{State: state, Confidence: 0.9, LastOutputAt: d.lastChange}, true
}

func (d *tmuxDriver) ScreenLines(string) []string { return strings.Split(d.capture(), "\n") }

// Composer reads the pane as plain text, which carries no styling, so any text
// after the prompt marker counts as a draft.
func (d *tmuxDriver) Composer(string) composerState {
	lines := strings.Split(d.capture(), "\n")
	cells := make([][]terminal.Cell, len(lines))
	for y, line := range lines {
		for _, r := range line {
			cells[y] = append(cells[y], terminal.Cell{Rune: r})
		}
	}
	return readComposer(cells)
}

func (d *tmuxDriver) CurrentCommand(string) (string, error) {
	out, err := exec.Command("tmux", "display-message", "-t", d.name, "-p", "#{pane_current_command}").Output()
	return strings.TrimSpace(string(out)), err
}

func (d *tmuxDriver) TypeText(_, text string) error {
	d.typed = append(d.typed, text)
	return exec.Command("tmux", "send-keys", "-t", d.name, "-l", text).Run()
}

func (d *tmuxDriver) PressEnter(string) error {
	d.typed = append(d.typed, "<enter>")
	return exec.Command("tmux", "send-keys", "-t", d.name, "Enter").Run()
}

func liveWait(t *testing.T, what string, timeout time.Duration, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); time.Sleep(400 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestLiveCompactionAgainstRealClaude(t *testing.T) {
	if os.Getenv("UM_LIVE") != "1" {
		t.Skip("set UM_LIVE=1 to spend real requests")
	}
	name := fmt.Sprintf("um-p6-%d", time.Now().Unix())
	cwd := t.TempDir()
	home, _ := os.UserHomeDir()
	driver := &tmuxDriver{name: name}
	tmuxRun(t, "new-session", "-d", "-s", name, "-x", "200", "-y", "50", "-c", cwd)
	t.Cleanup(func() {
		_ = exec.Command("tmux", "kill-session", "-t", name).Run()
		// The test's transcript directory under ~/.claude/projects is left in place:
		// removing it needs the operator's confirmation (storage-manager deletion-grant).
		t.Logf("left transcript directory for operator cleanup: %s", driver.transcriptDir)
	})
	tmuxRun(t, "send-keys", "-t", name, "claude --model haiku --dangerously-skip-permissions", "Enter")
	liveWait(t, "claude to start", 90*time.Second, func() bool {
		text := driver.capture()
		if strings.Contains(text, "Yes, I trust") && strings.Contains(text, "No, exit") {
			// The default choice is "No, exit": move to "Yes" first.
			_ = exec.Command("tmux", "send-keys", "-t", name, "Down").Run()
			time.Sleep(500 * time.Millisecond)
			_ = exec.Command("tmux", "send-keys", "-t", name, "Enter").Run()
		}
		return strings.Contains(text, "? for shortcuts") || strings.Contains(text, "bypass permissions")
	})
	t.Logf("pane after start:\n%s", driver.capture())
	m := newCompactionManager(driver)
	m.sleep = time.Sleep
	oldMin := compactMinTokens
	compactMinTokens = 1000 // a 2-exchange test session is far under the 20k production floor
	defer func() { compactMinTokens = oldMin }()

	// Request 1: a tiny exchange so the transcript has a measurable context.
	tmuxRun(t, "send-keys", "-t", name, "-l", "Reply with exactly the word: ok")
	tmuxRun(t, "send-keys", "-t", name, "Enter")
	// Claude derives the project directory name from the cwd with more than the
	// separators replaced, so find it by the unique test directory name.
	liveWait(t, "the project directory", 30*time.Second, func() bool {
		matches, _ := filepath.Glob(filepath.Join(home, ".claude", "projects", "*"+filepath.Base(filepath.Dir(cwd))+"*"+filepath.Base(cwd)+"*"))
		if len(matches) == 0 {
			matches, _ = filepath.Glob(filepath.Join(home, ".claude", "projects", "*"+filepath.Base(filepath.Dir(cwd))+"*"))
		}
		if len(matches) > 0 {
			driver.transcriptDir = matches[0]
		}
		return len(matches) > 0
	})

	liveWait(t, "first reply", 120*time.Second, func() bool {
		cs := driver.Sessions(context.Background())[0]
		r := m.inspect(cs)
		return r.found && r.state == string(session.ActivityIdle) && r.quiet
	})
	candidates := m.Candidates(context.Background())
	t.Logf("candidates after the first reply: %+v", candidates)
	if len(candidates) != 1 || !candidates[0].Eligible || candidates[0].ContextTokens == 0 {
		t.Fatalf("idle session with a measured context must be eligible: %+v", candidates)
	}

	// Request 2: make the session busy, then try to compact it.
	tmuxRun(t, "send-keys", "-t", name, "-l", "Write the numbers 1 to 500, one per line, then the word done.")
	tmuxRun(t, "send-keys", "-t", name, "Enter")
	liveWait(t, "session to look busy", 30*time.Second, func() bool {
		a, _ := driver.Activity(name)
		return a.State == session.ActivityWorking
	})
	typedBefore := len(driver.typed)
	busy := m.Compact(context.Background(), name, "bulk")
	t.Logf("busy attempt: %+v", busy)
	if busy.Outcome != "skipped" || busy.Reason != skipBusy || len(driver.typed) != typedBefore {
		t.Fatalf("a busy session must be skipped untouched: %+v typed=%v", busy, driver.typed[typedBefore:])
	}

	// Wait until it is idle again, then compact it (request 3).
	liveWait(t, "second reply", 180*time.Second, func() bool {
		cs := driver.Sessions(context.Background())[0]
		r := m.inspect(cs)
		return r.state == string(session.ActivityIdle) && r.quiet
	})
	event := m.Compact(context.Background(), name, "manual")
	t.Logf("idle attempt: %+v\ntyped: %v\npane:\n%s", event, driver.typed[typedBefore:], driver.capture())
	if event.Outcome != "compacted" {
		t.Fatalf("an idle session must be compacted: %+v", event)
	}
	for _, typed := range driver.typed[typedBefore:] {
		if typed != "/compact" && typed != "<enter>" {
			t.Fatalf("typed %q; only /compact and Enter are allowed", typed)
		}
	}
}
