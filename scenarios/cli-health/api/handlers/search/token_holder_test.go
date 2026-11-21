package search

import (
	"sync"
	"testing"
)

func TestTokenHolderSetGet(t *testing.T) {
	h := NewTokenHolder()
	if h.Get() != "" {
		t.Fatal("a fresh holder must return an empty token")
	}
	h.Set("tok-1")
	if got := h.Get(); got != "tok-1" {
		t.Fatalf("Get after Set = %q, want tok-1", got)
	}
	h.Set("tok-2")
	if got := h.Get(); got != "tok-2" {
		t.Fatalf("Set must overwrite; got %q, want tok-2", got)
	}
}

// TestTokenHolderConcurrent runs the registration-goroutine writer against the
// handler reader to surface a data race under -race.
func TestTokenHolderConcurrent(t *testing.T) {
	h := NewTokenHolder()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); h.Set("tok") }()
		go func() { defer wg.Done(); _ = h.Get() }()
	}
	wg.Wait()
}
