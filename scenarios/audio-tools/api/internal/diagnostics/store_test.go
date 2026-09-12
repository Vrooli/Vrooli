package diagnostics_test

import (
	"sync"
	"testing"

	"audio-tools/internal/diagnostics"
)

func TestLastRunStore_RingNewestFirst(t *testing.T) {
	s := diagnostics.NewLastRunStore(3)
	for i := 1; i <= 5; i++ {
		s.Record(diagnostics.Run{ID: idOf(i)})
	}
	recent := s.Recent()
	if len(recent) != 3 {
		t.Fatalf("want 3 retained runs, got %d", len(recent))
	}
	if recent[0].ID != "id-5" || recent[1].ID != "id-4" || recent[2].ID != "id-3" {
		t.Fatalf("unexpected order: %+v", recent)
	}
	if got := s.Latest(); got.ID != "id-5" {
		t.Fatalf("Latest = %s, want id-5", got.ID)
	}
}

func TestLastRunStore_ConcurrentRecord(t *testing.T) {
	s := diagnostics.NewLastRunStore(10)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Record(diagnostics.Run{ID: idOf(i)})
		}(i)
	}
	wg.Wait()
	if len(s.Recent()) != 10 {
		t.Fatalf("want 10 recent runs after concurrent writes, got %d", len(s.Recent()))
	}
}

func idOf(i int) string { return "id-" + intToString(i) }

func intToString(i int) string {
	// Avoid pulling in strconv just for tests.
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	digits := make([]byte, 0, 4)
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
