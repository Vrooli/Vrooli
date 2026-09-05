package main

// pty_tmux_payload_size_test.go: how big an input payload may be, and
// what bracketing it gets on the way in.
//
// These run against a real tmux (skipped when tmux is absent) because
// the constraint under test is tmux's own, not ours: `send-keys` packs
// its whole command into one imsg capped at MAX_IMSGSIZE, so a payload
// that fits comfortably in an exec argv can still be rejected outright.
// A mock cannot tell us that number, and getting it wrong drops user
// input silently.
//
// History worth keeping: maxKeystrokeArgvBytes was 64 KiB against a real
// tmux ceiling of ~16 KiB, so every paste in between failed. The old
// coverage only asserted that a 200 KiB payload returned no error — true,
// because 200 KiB took the working buffer path — leaving the dead band
// wide open under a green suite. Hence TestTmuxSendKeysCeiling_*: it
// measures what tmux actually accepts rather than restating a constant,
// and fails if our threshold ever creeps back above it.

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"web-console/internal/pty"
)

// bracketedPasteStart / bracketedPasteEnd are the DECSET 2004 markers a
// terminal wraps around pasted text when the application asks for them.
const (
	bracketedPasteStart = "\x1b[200~"
	bracketedPasteEnd   = "\x1b[201~"
)

// newPaneSink launches a tmux-backed PTY whose pane copies its stdin
// verbatim into a file, and returns the PTY, the tmux session name and
// the sink path.
//
// Tests assert on that file rather than on capture-pane output so they
// see the exact bytes the application received: pane rendering wraps at
// the pane width, may truncate, and interleaves echo — none of which
// survives a byte-exact comparison.
//
// requestBracketedPaste makes the pane emit DECSET 2004 before it starts
// reading, which is how a real TUI (an agent CLI, vim, a readline shell)
// tells tmux it wants paste markers.
func newPaneSink(t *testing.T, sessionID string, requestBracketedPaste bool) (pty.PTY, string, string) {
	t.Helper()

	p, err := tmuxPTYFactory(pty.LaunchSpec{
		SessionID: sessionID,
		Shell:     "/bin/sh",
		Cols:      200,
		Rows:      50,
	})
	if err != nil {
		t.Fatalf("tmuxPTYFactory: %v", err)
	}
	sessionName := tmuxSessionPrefix + sessionID
	t.Cleanup(func() {
		_ = p.Kill()
		_ = p.Close()
		_ = tmuxCmd("kill-session", "-t", sessionName).Run()
	})

	sinkPath := filepath.Join(t.TempDir(), "sink.bin")

	// `stty -echo` stops the tty echoing the payload back through tmux,
	// which for the 128 KiB case is the difference between a fast test
	// and a slow one. `exec` replaces the shell so nothing downstream can
	// consume input meant for the sink.
	setup := "stty -echo; "
	if requestBracketedPaste {
		setup += `printf '\033[?2004h'; `
	}
	setup += "exec cat > " + sinkPath + "\n"
	if err := p.WriteInput([]byte(setup), pty.KindKeystroke); err != nil {
		t.Fatalf("start pane sink: %v", err)
	}

	// Wait for cat to actually be the pane's foreground process. Waiting
	// on the file would race: the shell creates it at redirect time, a
	// moment before exec.
	waitFor(t, 5*time.Second, func() bool {
		out, qErr := tmuxCmd("display-message", "-t", sessionName, "-p", "#{pane_current_command}").Output()
		return qErr == nil && strings.TrimSpace(string(out)) == "cat"
	}, "pane sink never reached the `cat` stage")

	return p, sessionName, sinkPath
}

// waitFor polls cond until it returns true or the deadline passes.
func waitFor(t *testing.T, within time.Duration, cond func() bool, failMsg string) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("%s (waited %s)", failMsg, within)
}

// readSink waits for the sink to hold at least atLeast bytes, then lets
// it settle briefly so any trailing bytes (paste markers) land before
// the caller asserts. Returns everything received.
func readSink(t *testing.T, sinkPath string, atLeast int, within time.Duration) []byte {
	t.Helper()
	waitFor(t, within, func() bool {
		b, err := os.ReadFile(sinkPath)
		return err == nil && len(b) >= atLeast
	}, "pane sink never received "+strconv.Itoa(atLeast)+" bytes")

	time.Sleep(200 * time.Millisecond)
	got, err := os.ReadFile(sinkPath)
	if err != nil {
		t.Fatalf("read pane sink: %v", err)
	}
	return got
}

// newlineDelimitedPayload builds exactly n bytes of printable text
// broken into short lines, ending on a line boundary.
//
// The line breaks are required, not cosmetic. A pane in canonical mode
// (a plain `cat`, a shell prompt) buffers input per *line* in a 4 KiB tty
// buffer and silently discards the rest of any line that overruns it. A
// single 16 KiB line would therefore vanish for reasons that have nothing
// to do with the code under test. Real pasted text has newlines; so does
// this.
func newlineDelimitedPayload(n int) []byte {
	if n <= 0 {
		return nil
	}
	line := append(bytes.Repeat([]byte("x"), 79), '\n')
	buf := bytes.Repeat(line, n/len(line)+1)[:n]
	buf[n-1] = '\n'
	return buf
}

// TestTmuxSendKeysCeiling_IsAboveOurThreshold measures the largest
// payload the installed tmux accepts as a `send-keys` argument and
// asserts maxKeystrokeArgvBytes sits safely below it.
//
// This is the invariant that matters. Asserting the number itself would
// pass just as happily on a tmux that moved the limit; this fails.
func TestTmuxSendKeysCeiling_IsAboveOurThreshold(t *testing.T) {
	requireTmux(t)

	_, sessionName, _ := newPaneSink(t, "test-send-keys-ceiling", false)

	accepts := func(size int) bool {
		payload := strings.Repeat("x", size)
		return tmuxCmd("send-keys", "-t", sessionName, "-l", "--", payload).Run() == nil
	}

	// Binary-search the boundary: lo stays known-good, hi known-bad.
	lo, hi := 1024, 1024*1024
	if accepts(hi) {
		t.Skipf("tmux accepted a %d-byte command argument; no ceiling to measure", hi)
	}
	if !accepts(lo) {
		t.Fatalf("tmux rejected a %d-byte command argument; something is badly wrong", lo)
	}
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if accepts(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	ceiling := lo
	t.Logf("tmux send-keys ceiling for target %q: %d bytes", sessionName, ceiling)

	// Require real headroom, not a hairline pass: production session
	// names (wc-<uuid>) differ in length from this test's, and that
	// difference comes straight out of the same budget.
	const margin = 2048
	if maxKeystrokeArgvBytes+margin > ceiling {
		t.Errorf("maxKeystrokeArgvBytes = %d leaves less than %d bytes of headroom under tmux's %d-byte ceiling; "+
			"payloads between the two are rejected with \"command too long\" and dropped",
			maxKeystrokeArgvBytes, margin, ceiling)
	}
}

// TestTmuxPTY_Keystroke_DeliversAcrossArgvThreshold sends keystroke
// payloads spanning both transports — including the exact range that
// used to be dropped — and asserts every byte arrives.
func TestTmuxPTY_Keystroke_DeliversAcrossArgvThreshold(t *testing.T) {
	requireTmux(t)

	cases := []struct {
		name string
		size int
	}{
		{"below threshold, send-keys path", 4 * 1024},
		{"just above threshold, buffer path", maxKeystrokeArgvBytes + 1},
		// The rest sat in the old dead band: past tmux's real ceiling,
		// below the old 64 KiB reroute point.
		{"past tmux command ceiling", 16*1024 + 128},
		{"mid dead band", 32 * 1024},
		{"old threshold boundary", 64 * 1024},
		{"well past old threshold", 128 * 1024},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, _, sinkPath := newPaneSink(t, "test-argv-"+strconv.Itoa(tc.size), false)

			payload := newlineDelimitedPayload(tc.size)
			if err := p.WriteInput(payload, pty.KindKeystroke); err != nil {
				t.Fatalf("WriteInput %d bytes: %v", tc.size, err)
			}

			got := readSink(t, sinkPath, tc.size, 20*time.Second)
			if len(got) != tc.size {
				t.Fatalf("pane received %d bytes, want exactly %d", len(got), tc.size)
			}
			if !bytes.Equal(got, payload) {
				t.Errorf("pane received %d bytes but they differ from what was sent", len(got))
			}
			assertNoLeakedPasteBuffers(t)
		})
	}
}

// TestTmuxPTY_Paste_IsBracketedWhenAppRequestsIt pins the clipboard
// path's semantics: raw text read from navigator.clipboard carries no
// markers of its own, so tmux must supply them via `paste-buffer -p`.
//
// Without this, every newline in a multi-line paste reaches a TUI as a
// separate Enter — an agent submits the first line and treats the rest
// as follow-up prompts.
func TestTmuxPTY_Paste_IsBracketedWhenAppRequestsIt(t *testing.T) {
	requireTmux(t)

	p, _, sinkPath := newPaneSink(t, "test-paste-bracketed", true)

	payload := []byte("first line\nsecond line\nthird line\n")
	if err := p.WriteInput(payload, pty.KindPaste); err != nil {
		t.Fatalf("WriteInput paste: %v", err)
	}
	// The closing marker tmux appends has no newline of its own, and a
	// canonical-mode pane only hands over whole lines — so without a
	// flush the sink would never show it. A real user pressing Enter
	// after pasting does exactly this.
	if err := p.WriteInput([]byte("\n"), pty.KindKeystroke); err != nil {
		t.Fatalf("WriteInput flush newline: %v", err)
	}

	// Wait only for the payload itself, not for the markers: if the
	// markers are missing this should fail on the explicit marker
	// assertions below, not time out waiting for bytes that will never
	// come.
	got := readSink(t, sinkPath, len(payload), 10*time.Second)
	if !bytes.Contains(got, []byte(bracketedPasteStart)) {
		t.Errorf("paste reached the app without a start marker; got %q", got)
	}
	if !bytes.Contains(got, []byte(bracketedPasteEnd)) {
		t.Errorf("paste reached the app without an end marker; got %q", got)
	}
	if !bytes.Contains(got, payload) {
		t.Errorf("paste payload did not arrive intact; got %q", got)
	}
}

// TestTmuxPTY_OversizedKeystroke_IsNotBracketed is the other half of the
// contract: a keystroke payload takes the buffer path only because it is
// too large for a tmux command, and that transport choice must not change
// its semantics.
//
// xterm.js brackets pastes on the browser side before they reach the
// wire, so bracketing again here would double-wrap them and leak a
// literal ESC[200~ into the application's input.
func TestTmuxPTY_OversizedKeystroke_IsNotBracketed(t *testing.T) {
	requireTmux(t)

	// The pane requests bracketed paste, so `-p` would definitely take
	// effect if the keystroke path wrongly passed it.
	p, _, sinkPath := newPaneSink(t, "test-keystroke-unbracketed", true)

	size := maxKeystrokeArgvBytes * 2
	payload := newlineDelimitedPayload(size)
	if err := p.WriteInput(payload, pty.KindKeystroke); err != nil {
		t.Fatalf("WriteInput oversized keystroke: %v", err)
	}

	got := readSink(t, sinkPath, size, 20*time.Second)
	if bytes.Contains(got, []byte(bracketedPasteStart)) {
		t.Error("oversized keystroke payload was bracketed; the buffer path is a transport fallback " +
			"and must not add paste markers the client did not ask for")
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("oversized keystroke payload did not arrive intact: got %d bytes, want %d", len(got), size)
	}
}

// assertNoLeakedPasteBuffers checks that `paste-buffer -d` cleaned up
// after itself, so long-running sessions don't accumulate buffers.
func assertNoLeakedPasteBuffers(t *testing.T) {
	t.Helper()
	out, err := tmuxCmd("list-buffers", "-F", "#{buffer_name}").Output()
	if err != nil {
		return // no buffers at all
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(line, "wc-paste-") {
			t.Errorf("tmux buffer %q leaked after delivery", line)
		}
	}
}
