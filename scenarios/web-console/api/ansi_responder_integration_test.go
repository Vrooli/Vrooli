package main

import (
	"web-console/session"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"web-console/internal/pty"
)

// da1ProbeScript writes `\x1b[c` to stdout, waits up to 2 s for a reply on
// stdin, and prints a tagged result line containing the bytes it received.
// Each invocation must use a unique tag so we can distinguish probe output
// from the shell's echo of the command that invoked it (which also
// appears in the PTY stream).
//
// The probe puts its TTY into raw mode (ICANON off, ECHO off) before
// reading stdin because line discipline otherwise buffers the reply until
// it sees a newline and echoes it back as output. Real TUIs that emit
// these queries — Claude Code, vim, ncurses apps — all do the same dance.
func da1ProbeScript(tag string) string {
	return fmt.Sprintf(`import sys, select, os, termios, tty
fd = sys.stdin.fileno()
old = termios.tcgetattr(fd)
tty.setraw(fd)
try:
    sys.stdout.write(chr(0x1b)+'[c')
    sys.stdout.flush()
    r,_,_ = select.select([sys.stdin], [], [], 2.0)
    d = os.read(fd, 64) if r else b''
finally:
    termios.tcsetattr(fd, termios.TCSADRAIN, old)
sys.stdout.write("\n" + %q + "=" + repr(d) + "\n")
sys.stdout.flush()
`, tag)
}

// writeProbe writes the given Python source to a temp file and returns the
// path. Tests invoke it with `python3 -u <path>` so the shell echo of the
// invoking command doesn't accidentally contain the expected output
// markers.
func writeProbe(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "probe.py")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("write probe: %v", err)
	}
	return path
}

// readUntil consumes output from sub until `needle` appears or the deadline
// expires. Returns the accumulated buffer.
func readUntil(t *testing.T, sub session.SubscribeResult, needle string, timeout time.Duration) string {
	t.Helper()
	var out bytes.Buffer
	deadline := time.After(timeout)
	for {
		select {
		case chunk, ok := <-sub.OutputCh:
			if !ok {
				t.Fatalf("session output channel closed before %q; got: %q", needle, out.String())
			}
			if chunk == nil {
				continue
			}
			out.Write(chunk)
			if strings.Contains(out.String(), needle) {
				return out.String()
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %q; got (%d bytes): %q", needle, out.Len(), out.String())
			return out.String()
		}
	}
}

// TestStandardBackend_AnswersDA1Probe is a full-stack regression test that
// reproduces the Claude Code hang without needing the Claude binary or the
// browser: it spawns a real standard-backend session, runs a Python probe
// script that emits `\x1b[c` (DA1) and then waits up to 2 s for a reply on
// stdin, and asserts that the reply lands.
//
// Before the server-side ANSI responder was added this test's probe
// receives empty bytes (the `TIMEOUT` branch) because nothing upstream of
// the PTY answers the query. Any regression that removes the responder or
// breaks its wiring into readLoop will fail this test within ~3 s.
func TestStandardBackend_AnswersDA1Probe(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found; skipping DA1 probe integration test")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not found; skipping DA1 probe integration test")
	}

	sm := newSessionManager()
	sess, err := sm.Create("/bin/bash", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(sess.ID) })

	sub := sess.Subscribe()
	t.Cleanup(func() { sess.Unsubscribe(sub.OutputCh) })

	const tag = "DA1_RESULT_d59e72"
	probePath := writeProbe(t, da1ProbeScript(tag))

	if err := sess.WriteInput([]byte("python3 -u "+probePath+"\n"), pty.KindKeystroke); err != nil {
		t.Fatalf("write probe: %v", err)
	}

	out := readUntil(t, sub, tag+"=", 6*time.Second)

	// Extract the result line. The tag is unique so the first occurrence
	// that has an "=" after it is the probe's own output (the shell echo
	// of the invoking command doesn't contain the tag).
	idx := strings.Index(out, tag+"=")
	line := out[idx:]
	if nl := strings.IndexAny(line, "\r\n"); nl >= 0 {
		line = line[:nl]
	}

	// Python's repr(bytes) renders \x1b as literal backslash-x-1-b. If the
	// responder did not fire, the bytes are empty (b'' or b"").
	if strings.Contains(line, "b''") || strings.Contains(line, `b""`) {
		t.Fatalf("probe received no DA1 reply — server-side ANSI responder not wired into standard-backend readLoop. line=%q", line)
	}
	if !strings.Contains(line, `\x1b[?1;2c`) {
		t.Fatalf("probe received wrong DA1 reply; expected \\x1b[?1;2c, got: %q", line)
	}
}

// TestStandardBackend_AnswersClaudeStartupProbeSequence mirrors the exact
// three-query burst Claude Code 2.1.x emits at startup (DA3 + DECRQM 2026
// + DA1) and verifies all three replies arrive. Ordering matters because
// TUIs read them in emission order.
func TestStandardBackend_AnswersClaudeStartupProbeSequence(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not found")
	}

	sm := newSessionManager()
	sess, err := sm.Create("/bin/bash", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(sess.ID) })

	sub := sess.Subscribe()
	t.Cleanup(func() { sess.Unsubscribe(sub.OutputCh) })

	const tag = "CLAUDE_RESULT_8f1a2b"
	src := fmt.Sprintf(`import sys, select, os, time, termios, tty
fd = sys.stdin.fileno()
old = termios.tcgetattr(fd)
tty.setraw(fd)
try:
    esc = chr(0x1b)
    q = esc + "[>0q" + esc + "[?2026$p" + esc + "[c"
    sys.stdout.write(q)
    sys.stdout.flush()
    buf = b""
    end = time.time() + 3.0
    while time.time() < end:
        r, _, _ = select.select([sys.stdin], [], [], 0.2)
        if r:
            buf += os.read(fd, 128)
        if b"\x1b[?1;2c" in buf and b"P!|" in buf and b"2026" in buf:
            break
finally:
    termios.tcsetattr(fd, termios.TCSADRAIN, old)
sys.stdout.write("\n" + %q + "=" + repr(buf) + "\n")
sys.stdout.flush()
`, tag)

	probePath := writeProbe(t, src)

	if err := sess.WriteInput([]byte("python3 -u "+probePath+"\n"), pty.KindKeystroke); err != nil {
		t.Fatalf("write probe: %v", err)
	}

	out := readUntil(t, sub, tag+"=", 8*time.Second)

	idx := strings.Index(out, tag+"=")
	line := out[idx:]
	if nl := strings.IndexAny(line, "\r\n"); nl >= 0 {
		line = line[:nl]
	}

	if !strings.Contains(line, `\x1b[?1;2c`) {
		t.Errorf("missing DA1 reply in: %q", line)
	}
	if !strings.Contains(line, "P!|") {
		t.Errorf("missing DA3 reply in: %q", line)
	}
	if !strings.Contains(line, "2026") {
		t.Errorf("missing DECRQM 2026 reply in: %q", line)
	}
}
