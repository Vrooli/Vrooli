package pipeline

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/capabilities"
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

func TestIsBackendTimeout(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"os deadline", os.ErrDeadlineExceeded, true},
		{"net timeout", timeoutErr{}, true},
		{"client cancellation", context.Canceled, false},
		{"backend down", syscall.ECONNREFUSED, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isBackendTimeout(tc.err); got != tc.want {
				t.Errorf("isBackendTimeout(%v) = %v, want %v", tc.err, got, tc.want)
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
	degraded := newBackendDegraded("whisper")
	if !degraded.Transient {
		t.Error("degraded timeout error must be transient")
	}
	if degraded.EffectiveState() != STTBackendStateDegraded {
		t.Errorf("degraded state = %q, want %q", degraded.EffectiveState(), STTBackendStateDegraded)
	}
	if contains(degraded.Error(), "deadline") || contains(degraded.Error(), "timeout") {
		t.Errorf("degraded user message should not expose raw timeout detail: %q", degraded.Error())
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
	mu              sync.Mutex
	calls           int
	failWith        error
	bringUp         *flakeyDoer
	bringCapability *atomic.Bool
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
	if e.bringCapability != nil {
		e.bringCapability.Store(true)
	}
	return nil
}

type toggledCapability struct {
	available *atomic.Bool
}

func (c toggledCapability) Check(context.Context) (capabilities.Status, string) {
	if c.available.Load() {
		return capabilities.StatusAvailable, "available"
	}
	return capabilities.StatusUnavailable, "unavailable"
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

func TestEnsureWhisperAvailableEnsuresAndRefreshesCapability(t *testing.T) {
	available := &atomic.Bool{}
	caps := capabilities.NewRegistry(
		[]capabilities.Def{{ID: "whisper-stt", Name: "Whisper STT"}},
		map[string]capabilities.Checker{"whisper-stt": toggledCapability{available: available}},
		time.Hour,
	)
	ens := &fakeEnsurer{bringCapability: available}
	s := NewService(Config{}, "", nil, "", SpeakerConfig{}, "", nil, caps, nil, "", nil, nil)
	s.SetBackendEnsurer(ens, "whisper")
	s.SetAutoEnsure(true)

	if !s.EnsureWhisperAvailable(context.Background()) {
		t.Fatal("EnsureWhisperAvailable returned false after ensurer made capability available")
	}
	if ens.calls != 1 {
		t.Fatalf("EnsureRunning calls = %d, want 1", ens.calls)
	}
}

func TestEnsureWhisperAvailableRespectsAutoEnsureOff(t *testing.T) {
	available := &atomic.Bool{}
	caps := capabilities.NewRegistry(
		[]capabilities.Def{{ID: "whisper-stt", Name: "Whisper STT"}},
		map[string]capabilities.Checker{"whisper-stt": toggledCapability{available: available}},
		time.Hour,
	)
	ens := &fakeEnsurer{bringCapability: available}
	s := NewService(Config{}, "", nil, "", SpeakerConfig{}, "", nil, caps, nil, "", nil, nil)
	s.SetBackendEnsurer(ens, "whisper")
	s.SetAutoEnsure(false)

	if s.EnsureWhisperAvailable(context.Background()) {
		t.Fatal("EnsureWhisperAvailable returned true with auto ensure disabled")
	}
	if ens.calls != 0 {
		t.Fatalf("EnsureRunning calls = %d, want 0", ens.calls)
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

type timeoutDoer struct {
	calls atomic.Int32
}

func (d *timeoutDoer) Do(*http.Request) (*http.Response, error) {
	d.calls.Add(1)
	return nil, context.DeadlineExceeded
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestTranscribeTimeoutReturnsDegradedWithoutEnsure(t *testing.T) {
	doer := &timeoutDoer{}
	ens := &fakeEnsurer{}
	s := newRecoveryService(doer)
	s.SetBackendEnsurer(ens, "whisper")
	s.SetAutoEnsure(true)

	_, err := s.Transcribe(context.Background(), pcmAudio, "pcm_s16le", "en", "", false)
	var be *STTBackendError
	if !errors.As(err, &be) {
		t.Fatalf("err = %v, want *STTBackendError", err)
	}
	if be.EffectiveState() != STTBackendStateDegraded {
		t.Fatalf("state = %q, want %q", be.EffectiveState(), STTBackendStateDegraded)
	}
	if !be.Transient {
		t.Error("timeout/degraded error should be transient")
	}
	if ens.calls != 0 {
		t.Errorf("EnsureRunning called %d times for timeout, want 0", ens.calls)
	}
	if got := doer.calls.Load(); got != 1 {
		t.Errorf("HTTP attempts = %d, want 1", got)
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
