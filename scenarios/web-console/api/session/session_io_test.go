package session

import (
	"bytes"
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
