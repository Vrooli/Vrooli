package session

import (
	"bytes"
	"testing"

	"web-console/terminal"
)

func TestAnsiReplyFor_DA1(t *testing.T) {
	ev := terminal.ControlEvent{Kind: terminal.EventCSIQuery, Final: 'c'}
	want := []byte("\x1b[?1;2c")
	if got := ansiReplyFor(ev); !bytes.Equal(got, want) {
		t.Errorf("DA1: got %q want %q", got, want)
	}
}

func TestAnsiReplyFor_DA1ExplicitZero(t *testing.T) {
	ev := terminal.ControlEvent{Kind: terminal.EventCSIQuery, Final: 'c', Params: []int{0}}
	want := []byte("\x1b[?1;2c")
	if got := ansiReplyFor(ev); !bytes.Equal(got, want) {
		t.Errorf("DA1(0): got %q want %q", got, want)
	}
}

func TestAnsiReplyFor_DA3(t *testing.T) {
	// CSI = c — private '=', final 'c'.
	ev := terminal.ControlEvent{Kind: terminal.EventCSIQuery, Private: true, Final: 'c'}
	want := []byte("\x1bP!|00000000\x1b\\")
	if got := ansiReplyFor(ev); !bytes.Equal(got, want) {
		t.Errorf("DA3: got %q want %q", got, want)
	}
}

func TestAnsiReplyFor_XTVersion(t *testing.T) {
	// CSI > 0 q — private '>', params [0], final 'q'.
	ev := terminal.ControlEvent{Kind: terminal.EventCSIQuery, Private: true, Final: 'q', Params: []int{0}}
	want := []byte("\x1bP!|00000000\x1b\\")
	if got := ansiReplyFor(ev); !bytes.Equal(got, want) {
		t.Errorf("XTVERSION: got %q want %q", got, want)
	}
}

func TestAnsiReplyFor_DECRQM2026(t *testing.T) {
	// CSI ? 2026 $ p — private '?', params [2026], final 'p'.
	ev := terminal.ControlEvent{Kind: terminal.EventCSIQuery, Private: true, Final: 'p', Params: []int{2026}}
	want := []byte("\x1b[?2026;0$y")
	if got := ansiReplyFor(ev); !bytes.Equal(got, want) {
		t.Errorf("DECRQM 2026: got %q want %q", got, want)
	}
}

func TestAnsiReplyFor_UnknownEventKind(t *testing.T) {
	ev := terminal.ControlEvent{Kind: terminal.EventAltBufferEnter}
	if got := ansiReplyFor(ev); got != nil {
		t.Errorf("non-query event should yield nil; got %q", got)
	}
}

func TestAnsiReplyFor_UnknownDECRQM(t *testing.T) {
	// Same shape as DECRQM 2026 but mode 25 — not handled.
	ev := terminal.ControlEvent{Kind: terminal.EventCSIQuery, Private: true, Final: 'p', Params: []int{25}}
	if got := ansiReplyFor(ev); got != nil {
		t.Errorf("non-2026 DECRQM should yield nil; got %q", got)
	}
}

func TestAnsiReplyFor_CUPNotAQuery(t *testing.T) {
	// CUP `\x1b[1;1H` is final 'H' — should never produce a reply even if
	// somehow surfaced as an EventCSIQuery (defensive guard).
	ev := terminal.ControlEvent{Kind: terminal.EventCSIQuery, Final: 'H', Params: []int{1, 1}}
	if got := ansiReplyFor(ev); got != nil {
		t.Errorf("CUP should yield nil; got %q", got)
	}
}

// End-to-end through the emulator parser: feeding the literal query bytes
// must produce events that ansiReplyFor maps to the expected replies. This
// is the "byte-equivalent to the pre-Phase-3 inline scanner" guarantee.
func TestAnsiReplyFor_FromEmulatorParser_ClaudeStartup(t *testing.T) {
	// Claude Code 2.1.x emits DA3 (XTVERSION) + DECRQM + DA1 back-to-back
	// during init. The emulator must parse all three; the responder must
	// produce all three replies.
	emu := terminal.New(terminal.Options{Cols: 80, Rows: 24})
	events := emu.ControlEvents() // allocate channel BEFORE Feed
	chunk := []byte("\x1b[?25l\x1b[?2004h\x1b[?1004h\x1b[?2031h\x1b[>0q\x1b[?2026$p\x1b[c")
	_, _ = emu.Feed(chunk)

	var combined []byte
	// Drain at most a handful of events; we only care about the queries.
	for i := 0; i < 32; i++ {
		select {
		case ev := <-events:
			if r := ansiReplyFor(ev); len(r) > 0 {
				combined = append(combined, r...)
			}
		default:
			i = 32 // exit early
		}
	}

	if !bytes.Contains(combined, []byte("\x1bP!|00000000\x1b\\")) {
		t.Errorf("missing DA3/XTVERSION reply in %q", combined)
	}
	if !bytes.Contains(combined, []byte("\x1b[?2026;0$y")) {
		t.Errorf("missing DECRQM 2026 reply in %q", combined)
	}
	if !bytes.Contains(combined, []byte("\x1b[?1;2c")) {
		t.Errorf("missing DA1 reply in %q", combined)
	}
}
