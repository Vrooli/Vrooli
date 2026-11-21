package home

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Transcript lines in Codex's rollout shape (session 82039c84, 2026-11-12).
const (
	metaTop      = `{"timestamp":"2026-11-12T00:07:55Z","type":"session_meta","payload":{"id":"%s","source":"cli","base_instructions":"…"}}`
	metaSubagent = `{"timestamp":"2026-11-12T00:07:55Z","type":"session_meta","payload":{"id":"%s","source":{"subagent":{"thread_spawn":{"parent_thread_id":"main","depth":1}}}}}`
	started      = `{"timestamp":"2026-11-12T00:08:02Z","type":"event_msg","payload":{"type":"task_started","turn_id":"t1"}}`
	completed    = `{"timestamp":"2026-11-12T00:08:06Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"t1"}}`
	aborted      = `{"timestamp":"2026-11-12T00:08:06Z","type":"event_msg","payload":{"type":"turn_aborted","turn_id":"t1"}}`
	// A message that quotes an event type is not an event.
	quoted = `{"timestamp":"2026-11-12T00:08:07Z","type":"response_item","payload":{"type":"message","content":"\"type\":\"task_started\""}}`
)

func writeRollout(t *testing.T, codexDir, session, thread string, at time.Time, lines ...string) {
	t.Helper()
	dir := filepath.Join(codexDir, session, "sessions", "2026", "11", "12")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "rollout-2026-11-12T00-07-50-"+thread+".jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

func meta(format, id string) string { return strings.Replace(format, "%s", id, 1) }

func TestThreadViewNamesTheShownThreadAndARunningTurn(t *testing.T) {
	codexDir := t.TempDir()
	r := NewReader(Config{CodexDir: codexDir}, Deps{})
	base := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	writeRollout(t, codexDir, "s", "main", base, meta(metaTop, "main"), started, completed, quoted)
	// A newer sub-agent transcript does not change what the terminal shows.
	writeRollout(t, codexDir, "s", "worker", base.Add(time.Minute), meta(metaSubagent, "worker"), started)
	view, err := r.ThreadView("s", "main")
	if err != nil || view.Shown != "main" || view.TurnRunning {
		t.Fatalf("view = %+v, %v; want main shown and idle", view, err)
	}

	writeRollout(t, codexDir, "s", "main", base.Add(2*time.Minute), meta(metaTop, "main"), started, completed, started)
	if view, _ = r.ThreadView("s", "main"); !view.TurnRunning {
		t.Fatal("a started turn without its end reads as idle")
	}
	writeRollout(t, codexDir, "s", "main", base.Add(2*time.Minute), meta(metaTop, "main"), started, aborted)
	if view, _ = r.ThreadView("s", "main"); view.TurnRunning {
		t.Fatal("an aborted turn reads as running")
	}

	// The operator started a new conversation in the same session.
	writeRollout(t, codexDir, "s", "fresh", base.Add(3*time.Minute), meta(metaTop, "fresh"))
	if view, _ = r.ThreadView("s", "main"); view.Shown != "fresh" {
		t.Fatalf("shown = %q, want the newer top-level thread", view.Shown)
	}

	// No boundary at all, or no transcript of the thread: unknown, an error.
	writeRollout(t, codexDir, "s", "quiet", base, meta(metaTop, "quiet"), quoted)
	if _, err := r.ThreadView("s", "quiet"); err == nil {
		t.Fatal("a transcript without a turn boundary gave a view")
	}
	if _, err := r.ThreadView("s", "missing"); err == nil {
		t.Fatal("a thread without a transcript gave a view")
	}
}

func TestOnlyProcessesInTheSessionsTerminalBelongToIt(t *testing.T) {
	proc := t.TempDir()
	// tmux server 10 → pane shell 11 → codex 12 → app-server 13; worker 20
	// was detached (its parent is 1) but kept the session's environment.
	tree := map[int]int{11: 10, 12: 11, 13: 12, 20: 1}
	env := "CODEX_HOME=/codex/s\x00WC_WEB_CONSOLE_SESSION_ID=s\x00TMUX=/tmp/wc.sock,10,1784\x00"
	for pid, parent := range tree {
		dir := filepath.Join(proc, strconv.Itoa(pid))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := "bash"
		if pid != 11 {
			cmd = "codex\x00--yolo"
		}
		if pid == 13 {
			cmd = "codex\x00app-server"
		}
		writeAll(t, filepath.Join(dir, "cmdline"), cmd)
		writeAll(t, filepath.Join(dir, "environ"), env)
		writeAll(t, filepath.Join(dir, "stat"), strconv.Itoa(pid)+" (co dex) S "+strconv.Itoa(parent)+" 11 11 0")
	}
	processes, known, _ := ProcTable{Root: proc}.Codex()
	if !known || len(processes) != 3 {
		t.Fatalf("processes = %+v", processes)
	}
	for _, p := range processes {
		if want := p.PID != 20; p.OfSession("s") != want || p.OfSession("other") {
			t.Fatalf("pid %d: of session = %v, want %v", p.PID, p.OfSession("s"), want)
		}
	}
}

func writeAll(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
