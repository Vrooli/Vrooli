package asyncpoll

import (
	"errors"
	"testing"
	"time"

	"github.com/vrooli/api-core/scheduletest"
)

func TestUntilTerminalPollsImmediatelyAndWaitsBetweenPendingResults(t *testing.T) {
	clock := scheduletest.New(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	statuses := []struct {
		terminal bool
		status   string
	}{{false, "running"}, {true, "completed"}}
	calls := 0
	got, err := UntilTerminal(clock, 2*time.Second, 6*time.Hour,
		func() (string, bool, string, error) {
			step := statuses[calls]
			calls++
			return step.status, step.terminal, step.status, nil
		}, "audit", "a-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "completed" || calls != 2 || !clock.Now().Equal(time.Date(2026, 1, 2, 3, 4, 7, 0, time.UTC)) {
		t.Fatalf("got %q after %d calls at %s", got, calls, clock.Now())
	}
}

func TestUntilTerminalPreservesFetchErrors(t *testing.T) {
	want := errors.New("transport failed")
	_, err := UntilTerminal(scheduletest.New(time.Unix(100, 0)), 2*time.Second, time.Hour,
		func() (string, bool, string, error) { return "", false, "", want }, "restore", "r-1")
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want fetch error %v", err, want)
	}
}

func TestUntilTerminalReportsOperationAndLastStatusAtDeadline(t *testing.T) {
	clock := scheduletest.New(time.Unix(100, 0))
	_, err := UntilTerminal(clock, 2*time.Second, 3*time.Second,
		func() (string, bool, string, error) { return "", false, "restoring", nil }, "restore", "r-2")
	want := `restore "r-2" did not reach a terminal state within 3s (still restoring)`
	if err == nil || err.Error() != want {
		t.Fatalf("unexpected timeout error: %v", err)
	}
	if !clock.Now().Equal(time.Unix(104, 0)) {
		t.Fatalf("deadline checked at %s, want 104s", clock.Now())
	}
}
