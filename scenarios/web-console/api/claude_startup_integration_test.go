package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"web-console/session"
)

// TestStandardBackend_ClaudeCodeActuallyStarts is the end-to-end regression
// test for the "Claude Code hangs in standard backend" bug. It spawns a
// real standard-backend session with /bin/bash, sends the literal shortcut
// command the UI sends (`claude --dangerously-skip-permissions\n`), and
// waits for Claude to emit evidence that it has rendered its UI far enough
// to be usable: the input prompt box.
//
// Pass condition: within 30 s we see the Unicode input-box top border
// ("╭" used by Claude Code's prompt) after the banner. If this doesn't
// arrive, Claude is stuck waiting on something server-side.
//
// Skip conditions: claude binary missing, or no auth credentials (we look
// for ~/.claude/.credentials.json as the cheap proxy for "Claude is
// logged in on this machine" — nothing else distinguishes "hang because
// the server broke it" from "hang on OAuth flow because no auth").
func TestStandardBackend_ClaudeCodeActuallyStarts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow e2e test in -short mode")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude binary not on PATH")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot resolve home dir: %v", err)
	}
	credPath := home + "/.claude/.credentials.json"
	if _, err := os.Stat(credPath); err != nil {
		t.Skipf("no Claude credentials at %s — skipping e2e start test", credPath)
	}

	sm := newSessionManager()
	sess, err := sm.Create(context.Background(), "/bin/bash", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(context.Background(), sess.ID) })

	sub := sess.Subscribe()
	t.Cleanup(func() { sess.Unsubscribe(sub.OutputCh) })

	// Exactly what the UI's Claude Code shortcut sends.
	cmd := "claude --dangerously-skip-permissions\n"
	if err := sess.SendInput(session.InputText(cmd)); err != nil {
		t.Fatalf("write shortcut: %v", err)
	}

	// Collect output for up to 30 s. The broken path emits only the banner
	// heading ("Claude Code v…") and then stalls; the working path proceeds
	// to render the input prompt row (`❯` — U+276F — followed by a block
	// cursor in reverse video) and the permission-mode footer
	// ("bypass permissions on"). We require at least one of those, since
	// their exact wording varies by account, but neither appears until
	// Claude's startup pipeline clears whatever was blocking it on the
	// probe replies.
	var out bytes.Buffer
	const (
		bannerMarker = "Claude Code v"
		// These signal a fully rendered interactive UI. They come from
		// Claude's post-init render pass, after the stall point. Claude
		// renders words with `\x1b[1C` (cursor-forward) between them to
		// save bytes, so we match individual words rather than phrases.
		promptMarker  = "❯"      // input prompt glyph
		footerMarker1 = "bypass" // part of "bypass permissions on"
		footerMarker2 = "shift+tab"
	)
	deadline := time.After(30 * time.Second)
	sawBanner := false
	for {
		select {
		case chunk, ok := <-sub.OutputCh:
			if !ok {
				t.Fatalf("session channel closed before Claude UI rendered; output=%q", out.String())
			}
			if chunk == nil {
				continue
			}
			out.Write(chunk)
			if !sawBanner && bytes.Contains(out.Bytes(), []byte(bannerMarker)) {
				sawBanner = true
				t.Logf("saw banner after %d bytes — waiting for interactive UI", out.Len())
			}
			if bytes.Contains(out.Bytes(), []byte(promptMarker)) &&
				bytes.Contains(out.Bytes(), []byte(footerMarker1)) &&
				bytes.Contains(out.Bytes(), []byte(footerMarker2)) {
				// Render reached an apparent prompt. Now verify Claude is
				// actually interactive: type a printable character and
				// confirm the prompt box updates with it. Without this
				// check, a static "frozen" render would pass.
				sawBytes := out.Len()
				t.Logf("interactive-UI render reached at %d bytes; probing input responsiveness", sawBytes)
				if err := sess.SendInput(session.InputText("z")); err != nil {
					t.Fatalf("write probe keystroke: %v", err)
				}
				probeDeadline := time.After(5 * time.Second)
				for {
					select {
					case chunk, ok := <-sub.OutputCh:
						if !ok {
							t.Fatalf("session channel closed while probing interactivity; last output=%q", tailString(out.String(), 2048))
						}
						if chunk == nil {
							continue
						}
						out.Write(chunk)
						// Claude re-renders the prompt box after each
						// keystroke; the letter 'z' appears inside the box.
						if idx := bytes.LastIndex(out.Bytes(), []byte{'z'}); idx >= sawBytes {
							return
						}
					case <-probeDeadline:
						t.Fatalf("Claude rendered UI but did not echo a typed keystroke within 5s — the UI is frozen. Last 2 KB:\n%s", tailString(out.String(), 2048))
					}
				}
			}
		case <-deadline:
			if sawBanner {
				t.Fatalf(
					"Claude Code rendered its banner but never reached an interactive prompt within 30s — the server-side ANSI responder or some other part of the standard-backend pipeline is still leaving Claude stuck. last 2 KB of output:\n%s",
					tailString(out.String(), 2048),
				)
			}
			t.Fatalf("Claude Code did not even render its banner within 30s. last 2 KB of output:\n%s", tailString(out.String(), 2048))
		}
	}
}

func tailString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + strings.ReplaceAll(s[len(s)-n:], "\x1b", "ESC")
}

// TestStandardBackend_StripsSyncModeFromClientStream is the regression
// test for "Claude Code banner never appears in mobile Safari". xterm.js
// has a 1 s auto-flush timer on DECSET 2026 (synchronized output); mobile
// Safari throttles JS timers in a way that starves that timer, so unclosed
// sync-mode segments from Claude hide the banner indefinitely. The server
// sanitizer strips both DECSET and DECRST for mode 2026 from the stream
// before it reaches the WS client. This test asserts that the client
// never sees either escape even when Claude Code is actually running and
// emitting them.
func TestStandardBackend_StripsSyncModeFromClientStream(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude binary not on PATH")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot resolve home: %v", err)
	}
	if _, err := os.Stat(home + "/.claude/.credentials.json"); err != nil {
		t.Skip("no Claude credentials — skipping")
	}

	sm := newSessionManager()
	sess, err := sm.Create(context.Background(), "/bin/bash", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(context.Background(), sess.ID) })

	sub := sess.Subscribe()
	t.Cleanup(func() { sess.Unsubscribe(sub.OutputCh) })

	if err := sess.SendInput(session.InputText("claude --dangerously-skip-permissions\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Collect 3 s of output. That's more than enough for Claude to emit
	// its initial banner which in the log evidence always contains 2026
	// toggles.
	var out bytes.Buffer
	deadline := time.After(3 * time.Second)
collect:
	for {
		select {
		case chunk, ok := <-sub.OutputCh:
			if !ok {
				break collect
			}
			if chunk == nil {
				continue
			}
			out.Write(chunk)
		case <-deadline:
			break collect
		}
	}

	// Must NOT contain either sync-mode toggle — if it does, mobile
	// Safari will starve the flush timer and the banner will stay hidden.
	if bytes.Contains(out.Bytes(), []byte("\x1b[?2026h")) {
		t.Errorf("client stream contained DECSET 2026 (\\x1b[?2026h) — mobile xterm.js will hide buffered frames. Sanitize before broadcast.")
	}
	if bytes.Contains(out.Bytes(), []byte("\x1b[?2026l")) {
		t.Errorf("client stream contained DECRST 2026 (\\x1b[?2026l). Sanitize before broadcast.")
	}

	// Sanity: we did capture Claude output, not an empty stream.
	if !bytes.Contains(out.Bytes(), []byte("Claude Code")) {
		t.Errorf("did not see any Claude Code output in 3 s; test cannot tell whether sanitize ran. Got %d bytes.", out.Len())
	}
}
