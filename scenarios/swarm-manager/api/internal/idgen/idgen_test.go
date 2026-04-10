package idgen

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestGenerateUsesRandomHex(t *testing.T) {
	originalRandRead := randRead
	t.Cleanup(func() { randRead = originalRandRead })

	randRead = func(b []byte) (int, error) {
		for i := range b {
			b[i] = byte(i)
		}
		return len(b), nil
	}

	id := Generate()
	if id != "0001020304050607" {
		t.Fatalf("expected hex id to match stubbed bytes, got %q", id)
	}
	if len(id) != 16 {
		t.Fatalf("expected hex id length 16, got %d", len(id))
	}
}

func TestGenerateFallsBackOnRandError(t *testing.T) {
	originalRandRead := randRead
	t.Cleanup(func() { randRead = originalRandRead })

	randRead = func([]byte) (int, error) {
		return 0, errors.New("entropy failed")
	}

	id := Generate()
	if len(id) == 0 {
		t.Fatalf("expected fallback id, got empty string")
	}
	if strings.Contains(id, " ") {
		t.Fatalf("expected fallback id without spaces, got %q", id)
	}
	if _, err := time.Parse("20060102T150405.000000000", id); err != nil {
		t.Fatalf("expected fallback id to match time format: %v", err)
	}
}
