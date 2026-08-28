package session

import (
	"bytes"
	"errors"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"web-console/internal/ptyfake"
	"web-console/terminal"
)

type echoCountingPTY struct {
	*ptyfake.FakePTYWithOutput
	samples atomic.Int32
	state   EchoState
}

func (p *echoCountingPTY) TerminalEchoState() (EchoState, error) {
	p.samples.Add(1)
	return p.state, nil
}

func TestSplitCompleteUTF8_MatchesStdlib(t *testing.T) {
	cases := [][]byte{
		[]byte("plain text"),
		{0xe2},
		{0xe2, 0x82},
		{0xe2, 0x82, 0xac},
		{0xf0, 0x9f, 0x8c},
		{0xf0, 0x9f, 0x8c, 0x8d},
		{0x80, 0x81}, // orphaned continuation bytes pass through
	}
	for _, input := range cases {
		wantAt := len(input)
		for i := len(input) - 1; i >= 0 && i >= max(0, len(input)-4); i-- {
			if utf8.RuneStart(input[i]) && !utf8.FullRune(input[i:]) {
				wantAt = i
				break
			}
		}
		got, remainder := splitCompleteUTF8(input)
		if !bytes.Equal(got, input[:wantAt]) || !bytes.Equal(remainder, input[wantAt:]) {
			t.Fatalf("splitCompleteUTF8(%#v) = (%#v, %#v), want (%#v, %#v)", input, got, remainder, input[:wantAt], input[wantAt:])
		}
	}
}

func TestEchoStateUsesCachedBackendSample(t *testing.T) {
	p := &echoCountingPTY{
		FakePTYWithOutput: ptyfake.NewFakePTYWithOutput(),
		state:             EchoState{Known: true, EchoEnabled: true},
	}
	s := &Session{
		pty: p,
		emu: terminal.New(terminal.Options{Cols: 80, Rows: 24}),
	}

	s.RefreshEchoState(true)
	for i := 0; i < 100; i++ {
		if _, err := s.EchoState(); err != nil {
			t.Fatalf("EchoState() returned error: %v", err)
		}
	}
	if got := p.samples.Load(); got != 1 {
		t.Fatalf("backend echo samples after cached reads = %d, want 1", got)
	}

	p.state.EchoEnabled = false
	s.echoSampleMu.Lock()
	s.lastEchoSampleAt = time.Time{}
	s.echoSampleMu.Unlock()
	if !s.RefreshEchoState(false) {
		t.Fatal("RefreshEchoState did not report the changed backend state")
	}
	state, err := s.EchoState()
	if err != nil {
		t.Fatalf("EchoState() after refresh returned error: %v", err)
	}
	if state.EchoEnabled {
		t.Fatal("EchoState returned stale cached echo state")
	}
}

// paneAltScreenPTY reports the pane's own alternate-screen state, the way a
// tmux-backed PTY does. samples counts probes so the test can prove the
// backend is read on the bounded cadence rather than on every read.
type paneAltScreenPTY struct {
	*ptyfake.FakePTYWithOutput
	samples  atomic.Int32
	altState atomic.Bool
	probeErr error
}

func (p *paneAltScreenPTY) TerminalEchoState() (EchoState, error) {
	return EchoState{Known: true, EchoEnabled: true}, nil
}

func (p *paneAltScreenPTY) PaneInAltScreen() (bool, error) {
	p.samples.Add(1)
	if p.probeErr != nil {
		return false, p.probeErr
	}
	return p.altState.Load(), nil
}

// TestEchoStatePrefersPaneAltScreenOverEmulator locks in which artifact is the
// source of truth for "is the program full-screen".
//
// The emulator reads the tmux attach stream, and a tmux client emits
// `\x1b[?1049h` the moment it attaches. The emulator therefore answers
// "alternate buffer" for every persistent pane, whether it runs a shell or a
// full-screen TUI. Predictive echo is gated on this flag, so trusting the
// emulator silently disabled prediction for every tmux-backed session.
func TestEchoStatePrefersPaneAltScreenOverEmulator(t *testing.T) {
	p := &paneAltScreenPTY{FakePTYWithOutput: ptyfake.NewFakePTYWithOutput()}
	s := &Session{
		pty: p,
		emu: terminal.New(terminal.Options{Cols: 80, Rows: 24}),
	}

	// Put the emulator in the alternate buffer, exactly as a tmux attach does.
	s.emuMu.Lock()
	_, _ = s.emu.Feed([]byte("\x1b[?1049h"))
	emulatorSaysAlt := s.emu.InAltBuffer()
	s.emuMu.Unlock()
	if !emulatorSaysAlt {
		t.Fatal("precondition: emulator did not enter the alternate buffer")
	}

	// The pane runs an ordinary shell, so the answer must be false despite
	// the emulator, because the emulator is describing tmux, not the program.
	s.RefreshEchoState(true)
	state, err := s.EchoState()
	if err != nil {
		t.Fatalf("EchoState(): %v", err)
	}
	if state.InAltBuffer {
		t.Fatal("InAltBuffer reported the tmux client's screen instead of the pane's")
	}

	// When the pane's program really does go full-screen, the flag follows it.
	p.altState.Store(true)
	s.echoSampleMu.Lock()
	s.lastEchoSampleAt = time.Time{}
	s.echoSampleMu.Unlock()
	if !s.RefreshEchoState(false) {
		t.Fatal("RefreshEchoState did not report the pane entering the alternate screen")
	}
	state, err = s.EchoState()
	if err != nil {
		t.Fatalf("EchoState() after the pane went full-screen: %v", err)
	}
	if !state.InAltBuffer {
		t.Fatal("InAltBuffer did not follow the pane into the alternate screen")
	}
}

// TestEchoStateFallsBackToEmulatorWithoutPaneProbe keeps non-tmux backends
// working: a PTY that cannot report its pane state must leave the emulator in
// charge rather than defaulting the answer to false.
func TestEchoStateFallsBackToEmulatorWithoutPaneProbe(t *testing.T) {
	p := &echoCountingPTY{
		FakePTYWithOutput: ptyfake.NewFakePTYWithOutput(),
		state:             EchoState{Known: true, EchoEnabled: true},
	}
	s := &Session{
		pty: p,
		emu: terminal.New(terminal.Options{Cols: 80, Rows: 24}),
	}
	s.emuMu.Lock()
	_, _ = s.emu.Feed([]byte("\x1b[?1049h"))
	s.emuMu.Unlock()

	s.RefreshEchoState(true)
	state, err := s.EchoState()
	if err != nil {
		t.Fatalf("EchoState(): %v", err)
	}
	if !state.InAltBuffer {
		t.Fatal("a backend without a pane probe must keep the emulator's answer")
	}
}

// TestEchoStateIgnoresFailedPaneProbe asserts a failing probe does not get
// mistaken for "the pane is not full-screen". A permission or timeout failure
// and a genuine false must not share a value.
func TestEchoStateIgnoresFailedPaneProbe(t *testing.T) {
	p := &paneAltScreenPTY{
		FakePTYWithOutput: ptyfake.NewFakePTYWithOutput(),
		probeErr:          errors.New("tmux unavailable"),
	}
	s := &Session{
		pty: p,
		emu: terminal.New(terminal.Options{Cols: 80, Rows: 24}),
	}
	s.emuMu.Lock()
	_, _ = s.emu.Feed([]byte("\x1b[?1049h"))
	s.emuMu.Unlock()

	s.RefreshEchoState(true)
	state, err := s.EchoState()
	if err != nil {
		t.Fatalf("EchoState(): %v", err)
	}
	if !state.InAltBuffer {
		t.Fatal("a failed pane probe was treated as a confident 'not full-screen'")
	}
}
