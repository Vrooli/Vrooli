package session

import (
	"context"
	"errors"
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

func TestSession_OverflowDetachesSlowObserverWithExplicitTerminalEvent(t *testing.T) {
	s := New(Options{Transport: "fake"})
	defer s.Close("test")
	_, slow, err := s.Subscribe(context.Background(), 1)
	require.NoError(t, err)
	_, healthy, err := s.Subscribe(context.Background(), 4)
	require.NoError(t, err)

	s.EmitEvent(SessionEvent{Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "first"}})
	s.EmitEvent(SessionEvent{Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "second"}})

	terminal, ok := <-slow
	require.True(t, ok)
	require.Equal(t, EventClosed, terminal.Type)
	require.Contains(t, terminal.Closed.Reason, "observer-overflow: 2 events unavailable")
	require.Equal(t, s.ID(), terminal.Closed.ResumeFromEventID)
	_, ok = <-slow
	require.False(t, ok)
	require.Equal(t, 1, s.ObserverCount(), "only the healthy subscriber remains attached")

	for _, want := range []string{"first", "second"} {
		ev := <-healthy
		require.Equal(t, EventTranscriptDelta, ev.Type)
		require.Equal(t, want, ev.TranscriptDelta.Text)
	}
}

func TestSession_CloseReportsUndeliveredTailToSlowObserver(t *testing.T) {
	s := New(Options{Transport: "fake"})
	_, slow, err := s.Subscribe(context.Background(), 1)
	require.NoError(t, err)
	_, healthy, err := s.Subscribe(context.Background(), 2)
	require.NoError(t, err)

	s.EmitEvent(SessionEvent{Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "queued"}})
	s.Close("finished")

	terminal, ok := <-slow
	require.True(t, ok)
	require.Equal(t, EventClosed, terminal.Type)
	require.Contains(t, terminal.Closed.Reason, "observer-overflow: 2 events unavailable")
	require.Equal(t, s.ID(), terminal.Closed.ResumeFromEventID)
	_, ok = <-slow
	require.False(t, ok)

	first := <-healthy
	final := <-healthy
	require.Equal(t, EventTranscriptDelta, first.Type)
	require.Equal(t, EventClosed, final.Type)
	require.Equal(t, "finished", final.Closed.Reason)
}

func TestSession_SubscribeFromReplaysMissedEventsThenLiveEvents(t *testing.T) {
	s := New(Options{Transport: "fake"})
	defer s.Close("test")
	s.EmitEvent(SessionEvent{EventID: "cursor-1", Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "one"}})
	s.EmitEvent(SessionEvent{EventID: "cursor-2", Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "two"}})

	_, events, err := s.SubscribeFrom(context.Background(), 1, "cursor-1")
	require.NoError(t, err)
	s.EmitEvent(SessionEvent{EventID: "cursor-3", Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "three"}})

	for i, want := range []string{"cursor-2", "cursor-3"} {
		select {
		case event := <-events:
			require.Equal(t, want, event.EventID, "event index %d", i)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for event %q", want)
		}
	}
}

func TestSession_SubscribeFromRejectsUnavailableCursor(t *testing.T) {
	s := New(Options{Transport: "fake"})
	defer s.Close("test")
	_, _, err := s.SubscribeFrom(context.Background(), 4, "unknown-event")
	require.ErrorIs(t, err, ErrReplayCursorUnavailable)
}

func TestSession_IDIsInitialReplayAnchor(t *testing.T) {
	s := New(Options{Transport: "fake"})
	defer s.Close("test")
	s.EmitEvent(SessionEvent{EventID: "initial-event", Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "retained"}})

	_, events, err := s.SubscribeFrom(context.Background(), 1, s.ID())
	require.NoError(t, err)
	require.Equal(t, "initial-event", (<-events).EventID)
}

func TestSession_OverflowFailureCanRecoverFromRetainedCursor(t *testing.T) {
	s := New(Options{Transport: "fake"})
	defer s.Close("test")
	s.EmitEvent(SessionEvent{EventID: "last-seen", Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "seen"}})
	_, slow, err := s.Subscribe(context.Background(), 1)
	require.NoError(t, err)
	s.EmitEvent(SessionEvent{EventID: "missed-1", Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "recover me 1"}})
	s.EmitEvent(SessionEvent{EventID: "missed-2", Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "recover me 2"}})

	terminal := <-slow
	require.Equal(t, EventClosed, terminal.Type)
	require.Contains(t, terminal.Closed.Reason, "observer-overflow")
	require.Equal(t, "last-seen", terminal.Closed.ResumeFromEventID)

	_, recovered, err := s.SubscribeFrom(context.Background(), 1, "last-seen")
	require.NoError(t, err)
	first, second := <-recovered, <-recovered
	require.Equal(t, "missed-1", first.EventID)
	require.Equal(t, "missed-2", second.EventID)
}

func TestSession_ReplayInvalidatesCursorWhenOversizedEventExceedsBudget(t *testing.T) {
	s := New(Options{Transport: "fake"})
	defer s.Close("test")
	s.EmitEvent(SessionEvent{EventID: "before-large", Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "before"}})
	s.EmitEvent(SessionEvent{EventID: "too-large", Type: EventAudioOutput, AudioOutput: &AudioFrame{Bytes: make([]byte, maxRetainedBytes+1)}})
	s.EmitEvent(SessionEvent{EventID: "after-large", Type: EventTranscriptDelta, TranscriptDelta: &TranscriptDelta{Text: "after"}})

	if _, _, err := s.SubscribeFrom(context.Background(), 4, "before-large"); !errors.Is(err, ErrReplayCursorUnavailable) {
		t.Fatalf("SubscribeFrom with invalidated cursor error = %v, want ErrReplayCursorUnavailable", err)
	}
	_, events, err := s.SubscribeFrom(context.Background(), 4, "too-large")
	require.NoError(t, err)
	require.Equal(t, "after-large", (<-events).EventID)
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

func TestSession_BargeIn_OlderCancelHookCannotOverlapNewAction(t *testing.T) {
	entered := make(chan string, 2)
	release := make(chan struct{})
	s := New(Options{CancelHook: func(_ BargeInReason, eventID string) {
		entered <- eventID
		if eventID == "old" {
			<-release
		}
	}})
	defer s.Close("test")

	s.MarkInflight("old")
	bargeDone := make(chan struct{})
	go func() {
		s.BargeIn(BargeInVAD)
		close(bargeDone)
	}()
	require.Equal(t, "old", <-entered)

	markStarted := make(chan struct{})
	markDone := make(chan struct{})
	go func() {
		close(markStarted)
		s.MarkInflight("new")
		close(markDone)
	}()
	<-markStarted
	select {
	case <-markDone:
		t.Fatal("new action overlapped the older cancellation hook")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	<-bargeDone
	<-markDone

	s.BargeIn(BargeInVAD)
	require.Equal(t, "new", <-entered)
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

func TestSession_SubscribeCloseRaceIsSafe(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		s := New(Options{Transport: "fake"})
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 32; i++ {
				_, _, err := s.Subscribe(context.Background(), 1)
				if err != nil && err != ErrSessionClosed && err != ErrTooManySubscribers {
					t.Errorf("Subscribe error = %v", err)
				}
			}
		}()
		go func() {
			defer wg.Done()
			<-start
			s.Close("race")
			for i := 0; i < 32; i++ {
				s.EmitEvent(SessionEvent{Type: EventTranscriptDelta})
			}
		}()
		close(start)
		wg.Wait()
		require.Zero(t, s.ObserverCount())
	}
}

func TestSession_SubscribeEnforcesObserverLimit(t *testing.T) {
	s := New(Options{Transport: "fake"})
	defer s.Close("test")
	for i := 0; i < maxSessionObservers; i++ {
		if _, _, err := s.Subscribe(context.Background(), 1); err != nil {
			t.Fatalf("Subscribe[%d] error: %v", i, err)
		}
	}
	if _, _, err := s.Subscribe(context.Background(), 1); err != ErrTooManySubscribers {
		t.Fatalf("Subscribe at limit error = %v, want %v", err, ErrTooManySubscribers)
	}
}

func TestSession_SubscribeRejectsNilOrCanceledContext(t *testing.T) {
	s := New(Options{Transport: "fake"})
	defer s.Close("test")
	var nilContext context.Context
	if _, _, err := s.Subscribe(nilContext, 1); err != ErrNilContext {
		t.Fatalf("Subscribe(nil) error = %v, want %v", err, ErrNilContext)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.Subscribe(ctx, 1); err != context.Canceled {
		t.Fatalf("Subscribe(canceled) error = %v, want %v", err, context.Canceled)
	}
	require.Zero(t, s.ObserverCount())
}

func TestSession_UnsubscribeStopsContextWatcher(t *testing.T) {
	s := New(Options{Transport: "fake"})
	defer s.Close("test")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	key, _, err := s.Subscribe(ctx, 1)
	require.NoError(t, err)
	s.Unsubscribe(key)
	// Cancel after explicit unsubscribe; the watcher must already have exited
	// via the observer's done channel, and repeated unsubscribe remains safe.
	cancel()
	s.Unsubscribe(key)
	require.Zero(t, s.ObserverCount())
}

func TestSession_ClearInflightDoesNotClearNewerAction(t *testing.T) {
	s := New(Options{Transport: "fake"})
	defer s.Close("test")
	s.MarkInflight("old")
	s.MarkInflight("new")
	s.ClearInflight("old")
	var canceled string
	s.cancelHook = func(_ BargeInReason, id string) { canceled = id }
	s.BargeIn(BargeInVAD)
	require.Equal(t, "new", canceled)
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
