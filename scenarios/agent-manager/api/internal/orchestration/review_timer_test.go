package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

func TestReviewTimerValidation(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	at := now.Add(10 * time.Minute)
	if err := validateReviewTimer(at.Format(time.RFC3339), now, at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		key      string
		deadline time.Time
	}{
		{"tomorrow", at.Add(time.Minute)},
		{now.Format(time.RFC3339), at.Add(time.Minute)},
		{at.Format(time.RFC3339), at},
	} {
		if err := validateReviewTimer(test.key, now, test.deadline); err == nil {
			t.Fatalf("accepted key=%q deadline=%v", test.key, test.deadline)
		}
	}
}

func TestReviewTimerWakeMessages(t *testing.T) {
	handle := &domain.AwaitHandle{Producer: ProducerReviewTimer, Key: "2026-01-02T03:04:05Z"}
	message := formatWakeMessage(uuid.New(), handle, "review due; health unknown", false)
	if !strings.HasPrefix(message, domain.WakeTimerPrefix) || !strings.Contains(message, "health unknown") || strings.Contains(message, "Do NOT re-run") {
		t.Fatalf("review wake message: %q", message)
	}
	message = formatWakeMessage(uuid.New(), handle, "", true)
	if !strings.HasPrefix(message, domain.WakeTimerPrefix) || !strings.Contains(message, "unknown") {
		t.Fatalf("review deadline message: %q", message)
	}
}

func TestReviewTimerWaitsAndHonoursCancellation(t *testing.T) {
	w := NewReviewTimerWaiter()
	if w.Producer() != ProducerReviewTimer {
		t.Fatalf("producer=%q", w.Producer())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := w.Wait(ctx, time.Now().Add(time.Hour).Format(time.RFC3339Nano)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled wait: %v", err)
	}
	// Recovery after the chosen time must wake promptly, not wait a fresh interval.
	result, err := w.Wait(context.Background(), time.Now().Add(-time.Minute).Format(time.RFC3339Nano))
	if err != nil || !strings.Contains(result, "does not establish worker health or completion") {
		t.Fatalf("elapsed wait=%q err=%v", result, err)
	}
	if _, err := w.Wait(context.Background(), "invalid"); err == nil {
		t.Fatal("invalid time was accepted")
	}
}
