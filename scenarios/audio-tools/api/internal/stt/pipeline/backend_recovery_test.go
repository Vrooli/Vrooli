package pipeline

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"

	"audio-tools/internal/audioformat"
)

// pcmAudio is a tiny valid PCM payload the audioformat substrate wraps as WAV
// before the (faked) Whisper call — enough to reach the HTTP transport.
var pcmAudio = []byte{0x01, 0x00, 0x02, 0x00}

func TestIsBackendDown(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"econnrefused", syscall.ECONNREFUSED, true},
		{"dial op error", &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}, true},
		{"wrapped dial", errors.New("Post \"http://x\": " + (&net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}).Error()), false}, // string-only: must NOT match (no typed chain)
		{"host unreachable", syscall.EHOSTUNREACH, true},
		{"deadline exceeded", context.DeadlineExceeded, false},
		{"nil", nil, false},
		{"unrelated", errors.New("boom"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isBackendDown(tc.err); got != tc.want {
				t.Errorf("isBackendDown(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestBackendErrorTypingAndMessages(t *testing.T) {
	starting := newBackendStarting("whisper")
	if !errors.Is(starting, ErrSTTBackendUnavailable) {
		t.Error("starting error must match ErrSTTBackendUnavailable via Unwrap")
	}
	if !starting.Transient {
		t.Error("starting error must be Transient")
	}
	var be *STTBackendError
	if !errors.As(starting, &be) {
		t.Fatal("errors.As(*STTBackendError) must succeed")
	}
	if contains(starting.Error(), "dial") || contains(starting.Error(), "tcp") {
		t.Errorf("user message leaked transport detail: %q", starting.Error())
	}
	needsOp := newBackendNeedsOperator("whisper")
	if needsOp.Transient {
		t.Error("operator-action error must NOT be Transient")
	}
	if !contains(needsOp.Error(), "vrooli resource start whisper") {
		t.Errorf("operator message must carry the remediation command: %q", needsOp.Error())
	}
}

// flakeyDoer fails with a dial/connection-refused error until `up` is set, then
// returns a 200 OK Whisper response. It records how many requests it served.
type flakeyDoer struct {
	up    atomic.Bool
	calls atomic.Int32
}

func (d *flakeyDoer) Do(req *http.Request) (*http.Response, error) {
	d.calls.Add(1)
	if !d.up.Load() {
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"text":"recovered"}`)),
		Header:     http.Header{},
		Request:    req,
	}, nil
}

// fakeEnsurer records EnsureRunning calls and (optionally) brings the doer "up".
type fakeEnsurer struct {
	mu       sync.Mutex
	calls    int
	failWith error
	bringUp  *flakeyDoer
}

func (e *fakeEnsurer) EnsureRunning(_ context.Context, _ string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	if e.failWith != nil {
		return e.failWith
	}
	if e.bringUp != nil {
		e.bringUp.up.Store(true)
	}
	return nil
}

func newRecoveryService(doer HTTPDoer) *Service {
	s := NewService(Config{}, "", nil, "", SpeakerConfig{}, "", nil, nil, nil,
		"http://127.0.0.1:8090/asr?output=json", doer,
		audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false })))
	return s
}

// Down backend -> EnsureRunning -> retry succeeds. Exactly one ensure, two HTTP
// attempts, and a clean transcript.
func TestTranscribeEnsuresAndRetriesOnce(t *testing.T) {
	doer := &flakeyDoer{}
	ens := &fakeEnsurer{bringUp: doer}
	s := newRecoveryService(doer)
	s.SetBackendEnsurer(ens, "whisper")
	s.SetAutoEnsure(true)

	_, err := s.Transcribe(context.Background(), pcmAudio, "pcm_s16le", "en", "", false)
	if err != nil {
		t.Fatalf("Transcribe after recovery err = %v, want nil", err)
	}
	if ens.calls != 1 {
		t.Errorf("EnsureRunning called %d times, want 1", ens.calls)
	}
	if got := doer.calls.Load(); got != 2 {
		t.Errorf("HTTP attempts = %d, want 2 (fail then retry)", got)
	}
}

// Ensure fails -> typed non-transient operator-action error (FailedPrecondition
// downstream), and only one HTTP attempt (no retry against a backend we failed to
// start).
func TestTranscribeEnsureFailureReturnsTypedError(t *testing.T) {
	doer := &flakeyDoer{} // stays down
	ens := &fakeEnsurer{failWith: errors.New("start timed out")}
	s := newRecoveryService(doer)
	s.SetBackendEnsurer(ens, "whisper")
	s.SetAutoEnsure(true)

	_, err := s.Transcribe(context.Background(), pcmAudio, "pcm_s16le", "en", "", false)
	var be *STTBackendError
	if !errors.As(err, &be) {
		t.Fatalf("err = %v, want *STTBackendError", err)
	}
	if be.Transient {
		t.Error("ensure-failure error must be non-transient (operator action)")
	}
	if got := doer.calls.Load(); got != 1 {
		t.Errorf("HTTP attempts = %d, want 1 (no retry after start failed)", got)
	}
}

// Ensure succeeds but the backend is still not serving -> transient "starting"
// error (Unavailable downstream).
func TestTranscribeEnsuredButStillDownIsTransient(t *testing.T) {
	doer := &flakeyDoer{} // ensure does NOT bring it up
	ens := &fakeEnsurer{} // bringUp nil -> stays down
	s := newRecoveryService(doer)
	s.SetBackendEnsurer(ens, "whisper")
	s.SetAutoEnsure(true)

	_, err := s.Transcribe(context.Background(), pcmAudio, "pcm_s16le", "en", "", false)
	var be *STTBackendError
	if !errors.As(err, &be) {
		t.Fatalf("err = %v, want *STTBackendError", err)
	}
	if !be.Transient {
		t.Error("ensured-but-still-down error must be transient (starting)")
	}
	if ens.calls != 1 || doer.calls.Load() != 2 {
		t.Errorf("calls: ensure=%d http=%d, want ensure=1 http=2", ens.calls, doer.calls.Load())
	}
}

// Auto-ensure disabled -> no ensure attempt, immediate operator-action error.
func TestTranscribeAutoEnsureDisabled(t *testing.T) {
	doer := &flakeyDoer{}
	ens := &fakeEnsurer{}
	s := newRecoveryService(doer)
	s.SetBackendEnsurer(ens, "whisper")
	s.SetAutoEnsure(false)

	_, err := s.Transcribe(context.Background(), pcmAudio, "pcm_s16le", "en", "", false)
	var be *STTBackendError
	if !errors.As(err, &be) {
		t.Fatalf("err = %v, want *STTBackendError", err)
	}
	if be.Transient {
		t.Error("disabled auto-ensure must yield the operator-action error")
	}
	if ens.calls != 0 {
		t.Errorf("EnsureRunning called %d times with auto-ensure off, want 0", ens.calls)
	}
}

func TestAutoEnsureEnabledToggle(t *testing.T) {
	on := func(string) string { return "" }
	if !autoEnsureEnabled(on) {
		t.Error("empty env must default to ON")
	}
	for _, v := range []string{"0", "false", "off", "no"} {
		if autoEnsureEnabled(func(string) string { return v }) {
			t.Errorf("STT_AUTO_ENSURE=%q must disable", v)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
