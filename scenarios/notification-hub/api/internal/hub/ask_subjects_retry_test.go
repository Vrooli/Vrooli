package hub

import (
	"context"
	"errors"
	"testing"
)

func TestRetrySubjectStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	attempts := 0
	value, err := retrySubject(ctx, func() (int, bool, error) {
		attempts++
		return 7, true, errors.New("sqlite is locked")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("retrySubject() error = %v, want context canceled", err)
	}
	if value != 0 {
		t.Fatalf("retrySubject() value = %d, want zero result after cancellation", value)
	}
	if attempts != 1 {
		t.Fatalf("operation attempts = %d, want one before cancellation stops retries", attempts)
	}
}
