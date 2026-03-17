package main

import (
	"sync"
	"testing"
	"time"
)

func TestContainsRecentText_Match(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan string]struct{}),
	}
	sess.outputHistory = []byte("hello world, this is some output")

	if !sess.ContainsRecentText("hello world", 200) {
		t.Error("expected match for text present in buffer")
	}
}

func TestContainsRecentText_NoMatch(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan string]struct{}),
	}
	sess.outputHistory = []byte("hello world")

	if sess.ContainsRecentText("goodbye", 200) {
		t.Error("expected no match for text not in buffer")
	}
}

func TestContainsRecentText_WithANSI(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan string]struct{}),
	}
	sess.outputHistory = []byte("\x1b[31mhello\x1b[0m world")

	if !sess.ContainsRecentText("hello world", 200) {
		t.Error("expected match after stripping ANSI from buffer")
	}
}

func TestContainsRecentText_MinMatchLen(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan string]struct{}),
	}
	sess.outputHistory = []byte("hello world, this is a long response")

	// Only the first 5 chars should be used as needle
	if !sess.ContainsRecentText("hello DIFFERENT STUFF", 5) {
		t.Error("expected match using only first minMatchLen characters")
	}
}

func TestContainsRecentText_EmptyHistory(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan string]struct{}),
	}
	sess.outputHistory = nil

	if sess.ContainsRecentText("anything", 200) {
		t.Error("expected false for empty history")
	}
}

func TestContainsRecentText_EmptyText(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan string]struct{}),
	}
	sess.outputHistory = []byte("some output")

	// Empty needle: bytes.Contains(x, "") is always true in Go,
	// so ContainsRecentText("", n) returns true. Document this behavior.
	if !sess.ContainsRecentText("", 200) {
		t.Error("expected true for empty needle (bytes.Contains matches empty)")
	}
}

func TestSubscribeTTS_SendTTS_Delivered(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan string]struct{}),
	}

	ch := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ch)

	sess.SendTTS("speak this")

	select {
	case msg := <-ch:
		if msg != "speak this" {
			t.Errorf("expected %q, got %q", "speak this", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TTS message")
	}
}

func TestUnsubscribeTTS_ChannelClosed(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan string]struct{}),
	}

	ch := sess.SubscribeTTS()
	sess.UnsubscribeTTS(ch)

	// Reading from a closed channel should return the zero value immediately.
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after UnsubscribeTTS")
	}
}

// TestSendTTS_ConcurrentUnsubscribe verifies that SendTTS does not panic when
// subscribers are concurrently unsubscribing. Before the fix, close(ch) happened
// outside the lock, allowing SendTTS to write to a closed channel.
func TestSendTTS_ConcurrentUnsubscribe(t *testing.T) {
	t.Parallel()
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan string]struct{}),
	}

	const numGoroutines = 100
	const numSends = 1000

	// Sender goroutine: blast SendTTS calls.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < numSends; i++ {
			sess.SendTTS("msg")
		}
	}()

	// Subscriber goroutines: subscribe, drain one message, then unsubscribe.
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ch := sess.SubscribeTTS()
			// Drain at most one message to reduce buffer pressure.
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
		ttsClients: make(map[chan string]struct{}),
	}

	ch := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ch)

	// Fill the channel to capacity (buffer size is 8).
	for i := 0; i < 8; i++ {
		sess.SendTTS("fill")
	}

	if sess.ttsDropLogged {
		t.Fatal("ttsDropLogged should be false before overflow")
	}

	// These calls should trigger the drop path.
	sess.SendTTS("overflow1")
	sess.SendTTS("overflow2")
	sess.SendTTS("overflow3")

	if !sess.ttsDropLogged {
		t.Error("expected ttsDropLogged to be true after channel overflow")
	}
}

func TestSendTTS_NoSubscribers(t *testing.T) {
	sess := &Session{
		clients:    make(map[chan []byte]*ClientInfo),
		ttsClients: make(map[chan string]struct{}),
	}
	// Should not panic with no subscribers.
	sess.SendTTS("nobody listening")
}
