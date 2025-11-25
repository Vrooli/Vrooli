package events

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestPerExecutionSequencerIsolation(t *testing.T) {
	seq := NewPerExecutionSequencer()
	a := uuid.New()
	b := uuid.New()

	if got := seq.Next(a); got != 1 {
		t.Fatalf("expected first sequence to be 1, got %d", got)
	}
	if got := seq.Next(a); got != 2 {
		t.Fatalf("expected second sequence to be 2, got %d", got)
	}
	if got := seq.Next(b); got != 1 {
		t.Fatalf("expected first sequence for new execution to be 1, got %d", got)
	}
}

func TestPerExecutionSequencerConcurrent(t *testing.T) {
	seq := NewPerExecutionSequencer()
	executionID := uuid.New()
	const workers = 10
	const increments = 25

	var wg sync.WaitGroup
	var failed bool
	var failMu sync.Mutex

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				if seq.Next(executionID) == 0 {
					failMu.Lock()
					failed = true
					failMu.Unlock()
					return
				}
			}
		}()
	}
	wg.Wait()

	if failed {
		t.Fatal("sequence should never return zero value")
	}

	if got := seq.Next(executionID); got != workers*increments+1 {
		t.Fatalf("expected next sequence to be %d, got %d", workers*increments+1, got)
	}
}
