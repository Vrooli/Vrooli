package duplex

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"audio-tools/internal/session"
	"github.com/gorilla/websocket"
)

type testKey struct{ value string }

func (k testKey) ResolveOpenAIKey(context.Context) (string, error) { return k.value, nil }

type keyResolverFunc func(context.Context) (string, error)

func (f keyResolverFunc) ResolveOpenAIKey(ctx context.Context) (string, error) { return f(ctx) }

type eventSink struct{ events chan Event }

func (s *eventSink) Emit(e Event) { s.events <- e }

type blockingSink struct {
	entered chan struct{}
	release chan struct{}
}

func (s blockingSink) Emit(Event) {
	close(s.entered)
	<-s.release
}

func TestOpenAILiveOutputDeliveryIsSerializedWithInterruption(t *testing.T) {
	sink := blockingSink{entered: make(chan struct{}), release: make(chan struct{})}
	conn := &openAILiveConn{sink: sink, closing: true}
	outputDone := make(chan struct{})
	go func() {
		conn.mapEvent(liveEvent{Type: "session.output_transcript.delta", Delta: "before interruption"})
		close(outputDone)
	}()
	select {
	case <-sink.entered:
	case <-time.After(time.Second):
		t.Fatal("output event did not reach sink")
	}
	cancelDone := make(chan error, 1)
	go func() { cancelDone <- conn.Cancel(context.Background(), session.BargeInExplicit, "") }()
	select {
	case err := <-cancelDone:
		t.Fatalf("Cancel crossed an in-flight output delivery: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(sink.release)
	select {
	case <-outputDone:
	case <-time.After(time.Second):
		t.Fatal("output delivery did not finish")
	}
	select {
	case err := <-cancelDone:
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("Cancel error = %v, want closed connection after acquiring interruption boundary", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Cancel remained blocked after output delivery finished")
	}
}

func TestOpenAILiveProviderRequiresRegistrySelectedModel(t *testing.T) {
	var keyLookups atomic.Int32
	p := &OpenAILiveProvider{Keys: keyResolverFunc(func(context.Context) (string, error) {
		keyLookups.Add(1)
		return "secret", nil
	})}
	_, err := p.Open(context.Background(), Request{}, &eventSink{events: make(chan Event, 1)})
	if !errors.Is(err, ErrUnsupportedLiveModel) {
		t.Fatalf("Open with no selected model error = %v, want %v", err, ErrUnsupportedLiveModel)
	}
	if got := keyLookups.Load(); got != 0 {
		t.Fatalf("credential lookup count = %d, want 0 when model selection is missing", got)
	}
}

func TestOpenAILiveContractAndFinalCost(t *testing.T) {
	const secret = "do-not-print-this-key"
	commands := make(chan map[string]any, 8)
	fixtureReady := make(chan map[string]any, 1)
	fixtureAck := make(chan struct{})
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+secret {
			t.Errorf("authorization header missing")
		}
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		read := func() map[string]any {
			_, b, e := ws.ReadMessage()
			if e != nil {
				t.Errorf("read command: %v", e)
				return nil
			}
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			commands <- m
			return m
		}
		if read()["type"] != "session.start" {
			t.Error("start was not first")
		}
		_ = ws.WriteJSON(map[string]any{"type": "session.started", "event_id": "started"})
		var play map[string]any
		for i := 0; i < 2; i++ {
			got := read()
			if got != nil && got["type"] == "session.output_audio.fixture.play" {
				play = got
			}
		}
		if play == nil {
			t.Error("documented interruption fixture command missing")
		} else {
			if play["audio"] != base64.StdEncoding.EncodeToString(make([]byte, 480)) {
				t.Error("fixture was not 10 ms mono PCM16LE silence")
			}
			fixtureReady <- play
			_ = ws.WriteJSON(map[string]any{"type": "session.output_audio.fixture.played", "event_id": "wrong", "client_event_id": "unrelated"})
			_ = ws.WriteJSON(map[string]any{"type": "session.output_transcript.delta", "event_id": "old-output", "delta": "must stop"})
			_ = ws.WriteJSON(map[string]any{"type": "session.output_audio.delta", "event_id": "old-audio", "delta": base64.StdEncoding.EncodeToString([]byte{9, 0})})
			<-fixtureAck
			_ = ws.WriteJSON(map[string]any{"type": "session.output_audio.fixture.played", "event_id": "played", "client_event_id": play["event_id"]})
		}
		_ = ws.WriteJSON(map[string]any{"type": "session.output_transcript.delta", "event_id": "d1", "delta": "hello"})
		_ = ws.WriteJSON(map[string]any{"type": "session.input_transcript.delta", "event_id": "t1", "delta": "hi"})
		_ = ws.WriteJSON(map[string]any{"type": "session.output_audio.delta", "event_id": "new-audio", "delta": base64.StdEncoding.EncodeToString([]byte{1, 0})})
		if read()["type"] != "session.input_audio.append" {
			t.Error("audio did not resume after fixture.played")
		}
		if read()["type"] != "session.close" {
			t.Error("close command missing")
		}
		_ = ws.WriteJSON(map[string]any{"type": "session.closed", "event_id": "closed", "reason": "close_requested", "session": map[string]any{"id": "live-test-session"}, "usage": map[string]any{"seconds": 3, "active_seconds": 2, "paused_seconds": 1}})
		for {
			if _, _, e := ws.ReadMessage(); e != nil {
				return
			}
		}
	}))
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	sink := &eventSink{events: make(chan Event, 8)}
	p := &OpenAILiveProvider{Keys: testKey{secret}, URL: url}
	conn, err := p.Open(context.Background(), Request{Model: "gpt-live-1.1", Voice: "marin"}, sink)
	if err != nil {
		t.Fatal(err)
	}
	if err = conn.SendAudio(context.Background(), session.AudioFrame{Bytes: []byte{1, 0}, Format: "pcm_s16le", SampleRate: 24000}); err != nil {
		t.Fatal(err)
	}
	if err = conn.SendText(context.Background(), "test input"); !errors.Is(err, ErrTextInputUnsupported) {
		t.Fatalf("text input error=%v, want explicit unsupported", err)
	}
	cancelResult := make(chan error, 1)
	go func() { cancelResult <- conn.Cancel(context.Background(), session.BargeInExplicit, "response-1") }()
	select {
	case <-fixtureReady:
	case <-time.After(time.Second):
		t.Fatal("fixture command was not received")
	}
	select {
	case err = <-cancelResult:
		t.Fatalf("unmatched fixture acknowledgment ended cancel: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	close(fixtureAck)
	select {
	case err = <-cancelResult:
		if err != nil {
			t.Fatalf("cancel fixture handshake failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not wait for correlated fixture.played")
	}
	if err = conn.SendAudio(context.Background(), session.AudioFrame{Bytes: []byte{3, 0}, Format: "pcm_s16le", SampleRate: 24000}); err != nil {
		t.Fatalf("audio did not resume after fixture: %v", err)
	}
	audioIndex := 0
	for _, want := range []string{"session.start", "session.input_audio.append", "session.output_audio.fixture.play", "session.input_audio.append"} {
		select {
		case got := <-commands:
			if got["type"] != want {
				t.Fatalf("command=%v want %s", got["type"], want)
			}
			if want == "session.input_audio.append" {
				wantBytes := []byte{1, 0}
				if audioIndex == 1 {
					wantBytes = []byte{3, 0}
				}
				if got["audio"] != base64.StdEncoding.EncodeToString(wantBytes) {
					t.Fatalf("audio command %d was not encoded as raw PCM", audioIndex)
				}
				audioIndex++
			}
		case <-time.After(time.Second):
			t.Fatalf("missing %s", want)
		}
	}
	if err = conn.Close(); err != nil {
		t.Fatal(err)
	}
	usage, ok := conn.(FinalUsage).FinalProviderUsage()
	if !ok || usage.CostMicros != 2000 || usage.ActiveSeconds != 2 || usage.PausedSeconds != 1 || usage.ProviderReceiptID != "live-test-session" || usage.Model != "gpt-live-1.1" || usage.Provider != openAILiveProviderID {
		t.Fatalf("usage=%+v confirmed=%v", usage, ok)
	}
	gotTypes := map[string]bool{}
	var assistantText string
	var audioBytes []byte
	for len(sink.events) > 0 {
		e := <-sink.events
		if e.Err != nil {
			t.Fatal(e.Err)
		}
		gotTypes[string(e.Value.Type)] = true
		if e.Value.AssistantDelta != nil {
			assistantText += e.Value.AssistantDelta.Text
		}
		if e.Value.AudioOutput != nil {
			audioBytes = append(audioBytes, e.Value.AudioOutput.Bytes...)
		}
	}
	if !gotTypes[string(session.EventAssistantDelta)] || !gotTypes[string(session.EventTranscriptDelta)] {
		t.Fatalf("missing mapped events %v", gotTypes)
	}
	if assistantText != "hello" {
		t.Fatalf("assistant output during interruption was not suppressed or did not resume: %q", assistantText)
	}
	if !bytes.Equal(audioBytes, []byte{1, 0}) {
		t.Fatalf("audio during interruption was not suppressed or did not resume: %v", audioBytes)
	}
}

type countingKeyResolver struct{ calls atomic.Int32 }

func (r *countingKeyResolver) ResolveOpenAIKey(context.Context) (string, error) {
	r.calls.Add(1)
	return "", nil
}

func TestOpenAILiveRejectsUnknownModelBeforeCredentialOrDial(t *testing.T) {
	var dials atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { dials.Add(1) }))
	defer srv.Close()
	keys := &countingKeyResolver{}
	p := &OpenAILiveProvider{Keys: keys, URL: "ws" + strings.TrimPrefix(srv.URL, "http")}
	_, err := p.Open(context.Background(), Request{Model: "not-in-model-catalog"}, &eventSink{events: make(chan Event, 1)})
	if !errors.Is(err, ErrUnsupportedLiveModel) {
		t.Fatalf("Open error=%v, want unsupported model", err)
	}
	if keys.calls.Load() != 0 || dials.Load() != 0 {
		t.Fatalf("rejected model still resolved credentials or dialed: key_calls=%d dials=%d", keys.calls.Load(), dials.Load())
	}
}

func TestOpenAILiveUnsolicitedClosedCleansUpAndClosesSocket(t *testing.T) {
	peerClosed := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = ws.Close(); close(peerClosed) }()
		_, _, _ = ws.ReadMessage()
		_ = ws.WriteJSON(map[string]any{"type": "session.started"})
		_ = ws.WriteJSON(map[string]any{"type": "session.closed", "event_id": "expired", "reason": "expired", "session": map[string]any{"id": "live-expired-session"}, "usage": map[string]any{"seconds": 2}})
		_, _, _ = ws.ReadMessage()
	}))
	defer srv.Close()
	sink := &eventSink{events: make(chan Event, 2)}
	p := &OpenAILiveProvider{Keys: testKey{"unused"}, URL: "ws" + strings.TrimPrefix(srv.URL, "http")}
	conn, err := p.Open(context.Background(), Request{Model: "gpt-live-1.1"}, sink)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-sink.events:
		if ev.Err == nil || !strings.Contains(ev.Err.Error(), "expired") {
			t.Fatalf("unexpected remote close event: %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("remote close did not clean up")
	}
	usage, confirmed := conn.(FinalUsage).FinalProviderUsage()
	if !confirmed || usage.CostMicros != 1667 || usage.ProviderReceiptID != "live-expired-session" {
		t.Fatalf("final usage=%+v confirmed=%v", usage, confirmed)
	}
	select {
	case <-peerClosed:
	case <-time.After(time.Second):
		t.Fatal("remote terminal event did not close socket")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close after remote finalization: %v", err)
	}
}

func TestOpenAILiveTextInputIsExplicitlyUnsupported(t *testing.T) {
	c := &openAILiveConn{}
	if err := c.SendText(context.Background(), "user utterance"); !errors.Is(err, ErrTextInputUnsupported) {
		t.Fatalf("text input error=%v, want explicit unsupported", err)
	}
}

func TestOpenAILiveDoesNotConfirmUsageWithoutSessionReference(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		_, _, _ = ws.ReadMessage()
		_ = ws.WriteJSON(map[string]any{"type": "session.started"})
		if _, _, err := ws.ReadMessage(); err != nil {
			return
		}
		_ = ws.WriteJSON(map[string]any{"type": "session.closed", "event_id": "closed-without-session-id", "reason": "close_requested", "usage": map[string]any{"seconds": 2}})
	}))
	defer srv.Close()
	p := &OpenAILiveProvider{Keys: testKey{"unused"}, URL: "ws" + strings.TrimPrefix(srv.URL, "http")}
	conn, err := p.Open(context.Background(), Request{Model: "gpt-live-1.1"}, &eventSink{events: make(chan Event, 1)})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err == nil {
		t.Fatal("terminal usage without provider session reference was accepted")
	}
	if _, ok := conn.(FinalUsage).FinalProviderUsage(); ok {
		t.Fatal("usage without provider session reference was confirmed")
	}
}

func TestOpenAILiveInputIsDroppedWhileFixtureIsActive(t *testing.T) {
	c := &openAILiveConn{fixtureActive: true}
	err := c.SendAudio(context.Background(), session.AudioFrame{Bytes: []byte{0, 0}, Format: "pcm_s16le", SampleRate: 24000})
	if !errors.Is(err, ErrLiveInputDropped) {
		t.Fatalf("audio during fixture returned %v, want ErrLiveInputDropped", err)
	}
}

type finalUsageConn struct {
	Connection
	cost      uint64
	confirmed bool
}

func (c finalUsageConn) FinalProviderUsage() (ProviderUsage, bool) {
	return ProviderUsage{ProviderReceiptID: "fixture-session", Model: "gpt-live-1.1", Provider: openAILiveProviderID, CostMicros: c.cost}, c.confirmed
}

type finalUsageProvider struct {
	cost      uint64
	confirmed bool
}

func (p finalUsageProvider) Open(_ context.Context, _ Request, _ Sink) (Connection, error) {
	return finalUsageConn{Connection: &fakeConnection{}, cost: p.cost, confirmed: p.confirmed}, nil
}

func TestCoordinatorSettlesOnlyConfirmedFinalProviderUsage(t *testing.T) {
	budget := NewBudget(5000)
	reservation, err := budget.Reserve(4000)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewCoordinator(finalUsageProvider{cost: 2000, confirmed: true}, reserveAdmissionStub{reservation: reservation})
	if err != nil {
		t.Fatal(err)
	}
	h, err := coordinator.Open(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	if err = h.Close("test"); err != nil {
		t.Fatal(err)
	}
	if got := budget.Snapshot(); got.SpentMicros != 2000 || got.ReservedMicros != 0 {
		t.Fatalf("confirmed cost not settled: %+v", got)
	}
}

func TestCoordinatorRetainsUnconfirmedUsageReservation(t *testing.T) {
	budget := NewBudget(5000)
	reservation, err := budget.Reserve(4000)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewCoordinator(finalUsageProvider{cost: 0, confirmed: false}, reserveAdmissionStub{reservation: reservation})
	if err != nil {
		t.Fatal(err)
	}
	h, err := coordinator.Open(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	_ = h.Close("disconnect")
	if got := budget.Snapshot(); got.SpentMicros != 0 || got.ReservedMicros != 4000 {
		t.Fatalf("uncertain usage did not retain its full hold: %+v", got)
	}
	if _, err := budget.Reserve(1001); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("uncertain hold was available for reuse: %v", err)
	}
}

func TestCoordinatorUnconfirmedUsageCloseIsConcurrentAndIdempotent(t *testing.T) {
	budget := NewBudget(5000)
	reservation, err := budget.Reserve(4000)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewCoordinator(finalUsageProvider{confirmed: false}, reserveAdmissionStub{reservation: reservation})
	if err != nil {
		t.Fatal(err)
	}
	h, err := coordinator.Open(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = h.Close("disconnect")
		}()
	}
	wg.Wait()
	if got := budget.Snapshot(); got.SpentMicros != 0 || got.ReservedMicros != 4000 {
		t.Fatalf("concurrent close released or duplicated uncertain hold: %+v", got)
	}
}

func TestOpenAILiveDisconnectHasNoFinalUsageAndSanitizedErrors(t *testing.T) {
	const secret = "top-secret"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+secret {
			t.Error("missing auth")
		}
		up := websocket.Upgrader{}
		ws, _ := up.Upgrade(w, r, nil)
		if ws == nil {
			return
		}
		defer ws.Close()
		_, _, _ = ws.ReadMessage()
		_ = ws.WriteJSON(map[string]any{"type": "session.started"})
		_, _, _ = ws.ReadMessage()
	}))
	defer srv.Close()
	p := &OpenAILiveProvider{Keys: testKey{secret}, URL: "ws" + strings.TrimPrefix(srv.URL, "http")}
	sink := &eventSink{events: make(chan Event, 2)}
	conn, err := p.Open(context.Background(), Request{Model: "gpt-live-1.1"}, sink)
	if err != nil {
		t.Fatal(err)
	}
	if err = conn.SendAudio(context.Background(), session.AudioFrame{Bytes: []byte{0, 0}, Format: "pcm_s16le", SampleRate: 24000}); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-sink.events:
		if e.Err == nil || strings.Contains(e.Err.Error(), secret) {
			t.Fatalf("unsafe or missing disconnect error: %v", e.Err)
		}
	case <-time.After(time.Second):
		t.Fatal("disconnect not reported")
	}
	_, ok := conn.(FinalUsage).FinalProviderUsage()
	if ok {
		t.Fatal("unexpected disconnect confirmed usage")
	}
	_ = conn.Close()
}

func TestLiveCostConservativeRoundingAndAudioBounds(t *testing.T) {
	if got, ok := liveCostMicros("gpt-live-1.1", 1, 0); !ok || got != 834 {
		t.Fatalf("rounding=%d", got)
	}
	if got, ok := liveCostMicros("gpt-live-1.1", 0, 1); !ok || got != 334 {
		t.Fatalf("standby rounding=%d", got)
	}
	c := &openAILiveConn{}
	if err := c.SendAudio(context.Background(), session.AudioFrame{Bytes: []byte{0}, Format: "pcm_s16le", SampleRate: 24000}); err == nil {
		t.Fatal("odd PCM frame accepted")
	}
}
