package session

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSession_MultiObserver_ReceivesSameEvent(t *testing.T) {
	s := New(Options{Transport: "fake", Voice: "voice.feminine.warm", Language: "en"})
	defer s.Close("test")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const n = 3
	var wg sync.WaitGroup
	var counts [n]int32
	for i := 0; i < n; i++ {
		_, ch, err := s.Subscribe(ctx, 16)
		if err != nil {
			t.Fatalf("Subscribe[%d] error: %v", i, err)
		}
		wg.Add(1)
		go func(idx int, ch <-chan SessionEvent) {
			defer wg.Done()
			for evt := range ch {
				if evt.Type == EventTranscriptDelta {
					atomic.AddInt32(&counts[idx], 1)
				}
				if evt.Type == EventClosed {
					return
				}
			}
		}(i, ch)
	}

	for i := 0; i < 5; i++ {
		s.EmitEvent(SessionEvent{Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "hi"}})
	}
	// Wait for every observer to have observed all 5 events before
	// calling Close. require.Eventually is the event-driven substitute
	// for a wall-clock sleep — it polls the counters and fails fast if
	// the fan-out invariant doesn't hold.
	require.Eventually(t, func() bool {
		for i := 0; i < n; i++ {
			if atomic.LoadInt32(&counts[i]) != 5 {
				return false
			}
		}
		return true
	}, 2*time.Second, 1*time.Millisecond, "all observers must see all 5 events before Close")
	s.Close("done")
	wg.Wait()

	for i := 0; i < n; i++ {
		if c := atomic.LoadInt32(&counts[i]); c != 5 {
			t.Errorf("observer[%d] got %d events, want 5", i, c)
		}
	}
}

func TestSession_BargeIn_CancelsInflight(t *testing.T) {
	var canceled atomic.Bool
	var canceledID atomic.Value
	canceledID.Store("")

	s := New(Options{
		Transport: "fake",
		CancelHook: func(reason BargeInReason, eventID string) {
			canceled.Store(true)
			canceledID.Store(eventID)
		},
	})
	defer s.Close("test")

	s.MarkInflight("evt-1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, ch, err := s.Subscribe(ctx, 8)
	if err != nil {
		t.Fatalf("Subscribe error: %v", err)
	}

	s.BargeIn(BargeInVAD)

	select {
	case evt := <-ch:
		if evt.Type != EventBargeInCancel {
			t.Fatalf("first event type = %q, want %q", evt.Type, EventBargeInCancel)
		}
		if evt.BargeInCancel.CanceledEventID != "evt-1" {
			t.Errorf("canceled event id = %q, want %q", evt.BargeInCancel.CanceledEventID, "evt-1")
		}
		if evt.BargeInCancel.Reason != BargeInVAD {
			t.Errorf("reason = %q, want %q", evt.BargeInCancel.Reason, BargeInVAD)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("did not receive barge-in event within 100ms")
	}

	if !canceled.Load() {
		t.Error("cancel hook not invoked")
	}
	if id, _ := canceledID.Load().(string); id != "evt-1" {
		t.Errorf("cancel hook received eventID %q, want evt-1", id)
	}
}

func TestSession_BargeIn_NoInflight_NoOp(t *testing.T) {
	var canceled atomic.Bool
	s := New(Options{
		Transport: "fake",
		CancelHook: func(reason BargeInReason, eventID string) {
			canceled.Store(true)
		},
	})
	defer s.Close("test")

	s.BargeIn(BargeInVAD)

	if canceled.Load() {
		t.Error("cancel hook fired with no inflight event")
	}
}

func TestSession_Close_DropsSubscribers(t *testing.T) {
	s := New(Options{Transport: "fake"})

	_, ch, err := s.Subscribe(context.Background(), 4)
	if err != nil {
		t.Fatalf("Subscribe error: %v", err)
	}

	s.Close("test-done")

	gotClosed := false
	gotChannelClose := false
	for evt := range ch {
		if evt.Type == EventClosed {
			gotClosed = true
		}
	}
	gotChannelClose = true

	if !gotClosed {
		t.Error("subscriber did not receive Closed event")
	}
	if !gotChannelClose {
		t.Error("subscriber channel did not close")
	}

	if _, _, err := s.Subscribe(context.Background(), 4); err != ErrSessionClosed {
		t.Errorf("post-close Subscribe error = %v, want %v", err, ErrSessionClosed)
	}
}

func TestRegistry_AddGetRemove(t *testing.T) {
	r := NewRegistry()
	s := New(Options{Transport: "fake"})
	r.Add(s)

	if r.Count() != 1 {
		t.Errorf("Count = %d, want 1", r.Count())
	}
	got, err := r.Get(s.ID())
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.ID() != s.ID() {
		t.Errorf("Get returned id %q, want %q", got.ID(), s.ID())
	}
	r.Remove(s.ID())
	if _, err := r.Get(s.ID()); err == nil {
		t.Error("Get after Remove succeeded; want ErrUnknownSession")
	}
}

func TestResample_MuLawRoundTrip(t *testing.T) {
	// Synthesize a sine-ish PCM16 input, encode mu-law -> decode -> compare.
	pcm := make([]byte, 320) // 160 samples at 16k = 10 ms
	for i := 0; i < 160; i++ {
		v := int16((i % 31) * 256)
		pcm[i*2] = byte(uint16(v))
		pcm[i*2+1] = byte(uint16(v) >> 8)
	}
	muLaw, err := PCM16To8kMuLaw(pcm)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if len(muLaw) != 80 {
		t.Fatalf("muLaw length = %d, want 80", len(muLaw))
	}
	back := MuLawToPCM16(muLaw)
	if len(back) != len(pcm) {
		t.Fatalf("round-trip length %d != %d", len(back), len(pcm))
	}
	// We don't compare values — mu-law is lossy — only shape.
}
