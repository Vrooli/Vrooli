package main

import (
	"sync"
	"testing"
	"time"
)

func TestSubscribeTTS_SendTTS_Delivered(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan TTSCandidate]struct{}),
	}

	ch := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ch)

	candidate := TTSCandidate{EventID: "evt-1", Source: "test", SessionID: "s1", Text: "speak this"}
	sess.SendTTS(candidate)

	select {
	case msg := <-ch:
		if msg != candidate {
			t.Fatalf("expected %+v, got %+v", candidate, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TTS candidate")
	}
}

func TestUnsubscribeTTS_ChannelClosed(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan TTSCandidate]struct{}),
	}

	ch := sess.SubscribeTTS()
	sess.UnsubscribeTTS(ch)

	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after UnsubscribeTTS")
	}
}

func TestSendTTS_ConcurrentUnsubscribe(t *testing.T) {
	t.Parallel()
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan TTSCandidate]struct{}),
	}

	const numGoroutines = 100
	const numSends = 1000

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < numSends; i++ {
			sess.SendTTS(TTSCandidate{EventID: "evt", Source: "test", SessionID: "s1", Text: "msg"})
		}
	}()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ch := sess.SubscribeTTS()
			select {
			case <-ch:
			default:
			}
			sess.UnsubscribeTTS(ch)
		}()
	}

	wg.Wait()
	<-done
}

func TestSendTTS_DropsLoggedOnce(t *testing.T) {
	sess := &Session{
		ID:         "drop-test",
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan TTSCandidate]struct{}),
	}

	ch := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ch)

	candidate := TTSCandidate{EventID: "evt", Source: "test", SessionID: "s1", Text: "fill"}
	for i := 0; i < 8; i++ {
		sess.SendTTS(candidate)
	}

	if sess.ttsDropLogged {
		t.Fatal("ttsDropLogged should be false before overflow")
	}

	sess.SendTTS(candidate)
	sess.SendTTS(candidate)
	sess.SendTTS(candidate)

	if !sess.ttsDropLogged {
		t.Error("expected ttsDropLogged to be true after channel overflow")
	}
}

func TestSendTTS_NoSubscribers(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan TTSCandidate]struct{}),
	}
	sess.SendTTS(TTSCandidate{EventID: "evt", Source: "test", SessionID: "s1", Text: "nobody listening"})
}
