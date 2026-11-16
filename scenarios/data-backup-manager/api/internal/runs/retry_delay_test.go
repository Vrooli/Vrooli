package runs

import (
	"testing"
	"time"

	"data-backup-manager/internal/failures"
)

func TestRetryDelayGrowsThenStopsAtTheCap(t *testing.T) {
	want := []time.Duration{5 * time.Minute, 15 * time.Minute, 45 * time.Minute, 135 * time.Minute, 6 * time.Hour, 6 * time.Hour, 6 * time.Hour}
	for i, w := range want {
		if got := retryDelay(i + 1); got != w {
			t.Errorf("after %d failed retries wait %v, want %v", i+1, got, w)
		}
	}
}

func TestRetryHeldUntilReadsOnlyTheNewestUnbrokenStreak(t *testing.T) {
	end := time.Date(2026, 11, 15, 13, 0, 0, 0, time.UTC)
	failedRetry := func(code string) Run {
		return Run{Trigger: TriggerRetry, Status: RunFailed, FailureCode: failures.Code(code), FinishedAt: end}
	}
	cases := []struct {
		name       string
		history    []Run // newest first
		wantFailed int
	}{
		{"no history", nil, 0},
		{"newest is a scheduled failure: first retry goes ahead", []Run{{Trigger: TriggerScheduler, Status: RunFailed, FinishedAt: end}}, 0},
		{"newest retry completed", []Run{{Trigger: TriggerRetry, Status: RunCompleted, FinishedAt: end}, failedRetry("a")}, 0},
		{"one failed retry", []Run{failedRetry("a"), {Trigger: TriggerScheduler, Status: RunFailed}}, 1},
		{"three in a row", []Run{failedRetry("a"), failedRetry("a"), failedRetry("a")}, 3},
		{"a different failure ends the streak", []Run{failedRetry("a"), failedRetry("b"), failedRetry("b")}, 1},
		{"a completed run in between ends the streak", []Run{failedRetry("a"), {Trigger: TriggerManual, Status: RunCompleted}, failedRetry("a")}, 1},
	}
	for _, c := range cases {
		until, failed := retryHeldUntil(c.history)
		if failed != c.wantFailed {
			t.Errorf("%s: failed = %d, want %d", c.name, failed, c.wantFailed)
		}
		if c.wantFailed == 0 && !until.IsZero() {
			t.Errorf("%s: held until %v, want no hold", c.name, until)
		}
		if c.wantFailed > 0 && !until.Equal(end.Add(retryDelay(c.wantFailed))) {
			t.Errorf("%s: held until %v, want %v", c.name, until, end.Add(retryDelay(c.wantFailed)))
		}
	}
}
