package pipeline

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/testutil/mocks"
)

func TestAutoEnsureEnabledFromEnv(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"", true}, {"1", true}, {"true", true}, {"TRUE", true}, {"yes", true},
		{"0", false}, {"false", false}, {"FALSE", false}, {"off", false}, {"OFF", false}, {"no", false}, {"NO", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv(EnvAutoEnsure, tc.value)
			if got := AutoEnsureEnabledFromEnv(); got != tc.want {
				t.Errorf("AutoEnsureEnabledFromEnv() with %q = %t, want %t", tc.value, got, tc.want)
			}
		})
	}
}

func TestWhisperAvailableNilRegistryAndCachedState(t *testing.T) {
	if (newRecoveryService(nil)).WhisperAvailable(context.Background()) {
		t.Fatal("WhisperAvailable with no registry = true, want false")
	}
	for _, want := range []bool{false, true} {
		t.Run(map[bool]string{false: "unavailable", true: "available"}[want], func(t *testing.T) {
			available := &atomic.Bool{}
			available.Store(want)
			caps := capabilities.NewRegistry(
				[]capabilities.Def{{ID: "whisper-stt", Name: "Whisper STT"}},
				map[string]capabilities.Checker{"whisper-stt": toggledCapability{available: available}},
				time.Hour,
			)
			s := NewWhisperTranscriber(caps, "", nil, nil)
			if got := s.WhisperAvailable(context.Background()); got != want {
				t.Fatalf("WhisperAvailable() = %t, want %t", got, want)
			}
		})
	}
}

func TestSetWhisperConcurrencyNonPositiveDisablesBound(t *testing.T) {
	s := newRecoveryService(nil)
	for _, n := range []int{0, -1} {
		s.SetWhisperConcurrency(n)
		if got := s.snapshotDeps().whisperSem; got != nil {
			t.Errorf("SetWhisperConcurrency(%d) left semaphore enabled", n)
		}
	}
}

func TestEnsureWhisperAvailableRecoveryOutcomes(t *testing.T) {
	newCaps := func(available *atomic.Bool) *capabilities.Registry {
		return capabilities.NewRegistry(
			[]capabilities.Def{{ID: "whisper-stt", Name: "Whisper STT"}},
			map[string]capabilities.Checker{"whisper-stt": toggledCapability{available: available}},
			time.Hour,
		)
	}
	t.Run("already available skips ensure", func(t *testing.T) {
		available := &atomic.Bool{}
		available.Store(true)
		ens := &fakeEnsurer{}
		s := NewWhisperTranscriber(newCaps(available), "", nil, nil)
		s.SetBackendEnsurer(ens, "whisper")
		s.SetAutoEnsure(true)
		got := s.EnsureWhisperAvailable(context.Background())
		if !got || ens.calls != 0 {
			t.Fatalf("available probe: result/ensure calls = %t/%d, want true/0", got, ens.calls)
		}
	})
	t.Run("nil registry cannot be recovered", func(t *testing.T) {
		ens := &fakeEnsurer{}
		s := NewWhisperTranscriber(nil, "", nil, nil)
		s.SetBackendEnsurer(ens, "whisper")
		s.SetAutoEnsure(true)
		if s.EnsureWhisperAvailable(context.Background()) || ens.calls != 0 {
			t.Fatalf("nil registry should fail without ensure; calls=%d", ens.calls)
		}
	})
	t.Run("missing resource skips ensure", func(t *testing.T) {
		ens := &fakeEnsurer{}
		s := NewWhisperTranscriber(newCaps(&atomic.Bool{}), "", nil, nil)
		s.SetBackendEnsurer(ens, "")
		s.SetAutoEnsure(true)
		if s.EnsureWhisperAvailable(context.Background()) || ens.calls != 0 {
			t.Fatalf("missing resource should fail without ensure; calls=%d", ens.calls)
		}
	})
	t.Run("ensure error returns false", func(t *testing.T) {
		ens := &fakeEnsurer{failWith: errors.New("start failed")}
		s := NewWhisperTranscriber(newCaps(&atomic.Bool{}), "", nil, nil)
		s.SetBackendEnsurer(ens, "whisper")
		s.SetAutoEnsure(true)
		if s.EnsureWhisperAvailable(context.Background()) || ens.calls != 1 {
			t.Fatalf("ensure error should fail after one attempt; calls=%d", ens.calls)
		}
	})
	t.Run("successful ensure but unavailable capability returns false", func(t *testing.T) {
		available := &atomic.Bool{}
		ens := &fakeEnsurer{}
		s := NewWhisperTranscriber(newCaps(available), "", nil, nil)
		s.SetBackendEnsurer(ens, "whisper")
		s.SetAutoEnsure(true)
		if s.EnsureWhisperAvailable(context.Background()) || ens.calls != 1 {
			t.Fatalf("still unavailable should fail after one attempt; calls=%d", ens.calls)
		}
	})
	t.Run("registry without whisper capability returns false after ensure", func(t *testing.T) {
		caps := capabilities.NewRegistry(nil, nil, time.Hour)
		ens := &fakeEnsurer{}
		s := NewWhisperTranscriber(caps, "", nil, nil)
		s.SetBackendEnsurer(ens, "whisper")
		s.SetAutoEnsure(true)
		if s.EnsureWhisperAvailable(context.Background()) || ens.calls != 1 {
			t.Fatalf("missing whisper state should fail after ensure; calls=%d", ens.calls)
		}
	})
}

func TestSetLoggerCapturesRecoveryDecision(t *testing.T) {
	doer := &flakeyDoer{}
	s := newRecoveryService(doer)
	s.SetBackendEnsurer(&fakeEnsurer{}, "whisper")
	s.SetAutoEnsure(false)
	logger := mocks.NewFakeLogger()
	s.SetLogger(logger)
	_, err := s.Transcribe(context.Background(), pcmAudio, "pcm_s16le", "en", "", false)
	var backendErr *STTBackendError
	if !errors.As(err, &backendErr) || backendErr.State != STTBackendStateUnavailable {
		t.Fatalf("Transcribe error = %v, want typed unavailable error", err)
	}
	if entries := logger.Entries(); len(entries) != 1 || !strings.Contains(entries[0], "event=stt_backend_down") {
		t.Fatalf("logger entries = %v, want one backend-down event", entries)
	}
}

func TestQueuedTranscriptionCancellationDoesNotReachWhisper(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		entered <- struct{}{}
		select {
		case <-release:
			_, _ = w.Write([]byte(`{"text":"ok"}`))
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	s := NewWhisperTranscriber(nil, srv.URL+"/asr", srv.Client(), audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false })))
	s.SetWhisperConcurrency(1)
	first := make(chan error, 1)
	go func() {
		_, err := s.Transcribe(context.Background(), pcmAudio, "pcm_s16le", "", "", false)
		first <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first request did not enter Whisper")
	}
	ctx, cancel := context.WithCancel(context.Background())
	second := make(chan error, 1)
	go func() {
		_, err := s.Transcribe(ctx, pcmAudio, "pcm_s16le", "", "", false)
		second <- err
	}()
	cancel()
	select {
	case err := <-second:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("queued request error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled queued request did not return")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("Whisper request count = %d, want 1; cancelled waiter must not reach backend", got)
	}
	close(release)
	if err := <-first; err != nil {
		t.Fatalf("first request: %v", err)
	}
}
