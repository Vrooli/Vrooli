package main

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

const indexLockStderr = "fatal: Unable to create '/repo/.git/index.lock': File exists.\n\nAnother git process seems to be running in this repository."

// dummyCmd returns a harmless *exec.Cmd; the fake run closures never execute it.
func dummyCmd() *exec.Cmd { return exec.Command("true") }

func TestRetryOnIndexLock_RetriesThenSucceeds(t *testing.T) {
	attempts := 0
	build := func() *exec.Cmd { return dummyCmd() }
	run := func(_ *exec.Cmd) ([]byte, error) {
		attempts++
		if attempts < 3 {
			return []byte(indexLockStderr), &exec.ExitError{}
		}
		return []byte("ok"), nil
	}

	out, err := retryOnIndexLock(context.Background(), 3, time.Millisecond, build, run)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if string(out) != "ok" {
		t.Fatalf("expected final output 'ok', got %q", string(out))
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts (2 lock failures + 1 success), got %d", attempts)
	}
}

func TestRetryOnIndexLock_ReturnsOriginalErrorAfterCap(t *testing.T) {
	attempts := 0
	sentinel := errors.New("boom")
	build := func() *exec.Cmd { return dummyCmd() }
	run := func(_ *exec.Cmd) ([]byte, error) {
		attempts++
		return []byte(indexLockStderr), sentinel
	}

	out, err := retryOnIndexLock(context.Background(), 3, time.Millisecond, build, run)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected the original lock error after the cap, got %v", err)
	}
	if !outputIndicatesIndexLock(out) {
		t.Fatalf("expected the original lock output to be surfaced unchanged")
	}
	if attempts != 3 {
		t.Fatalf("expected exactly 3 attempts at the cap, got %d", attempts)
	}
}

func TestRetryOnIndexLock_DoesNotRetryNonLockErrors(t *testing.T) {
	attempts := 0
	sentinel := errors.New("fatal: pathspec 'x' did not match any files")
	build := func() *exec.Cmd { return dummyCmd() }
	run := func(_ *exec.Cmd) ([]byte, error) {
		attempts++
		return []byte("fatal: pathspec 'x' did not match any files"), sentinel
	}

	_, err := retryOnIndexLock(context.Background(), 3, time.Millisecond, build, run)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected the genuine error, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("genuine git failures must not be retried; got %d attempts", attempts)
	}
}

func TestRetryOnIndexLock_ContextCancelStopsRetries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled up front

	attempts := 0
	sentinel := errors.New("locked")
	build := func() *exec.Cmd { return dummyCmd() }
	run := func(_ *exec.Cmd) ([]byte, error) {
		attempts++
		return []byte(indexLockStderr), sentinel
	}

	_, err := retryOnIndexLock(ctx, 3, time.Second, build, run)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected the original error surfaced on cancel, got %v", err)
	}
	// First attempt runs, backoff observes cancellation, no second attempt.
	if attempts != 1 {
		t.Fatalf("expected 1 attempt before cancellation aborts backoff, got %d", attempts)
	}
}

func TestOutputIndicatesIndexLock(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"canonical lock", indexLockStderr, true},
		{"file exists only", "index.lock: File exists", true},
		{"unable to create only", "Unable to create index.lock", true},
		{"unrelated error", "fatal: pathspec did not match", false},
		{"lock word without markers", "index.lock removed", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := outputIndicatesIndexLock([]byte(tc.in)); got != tc.want {
				t.Fatalf("outputIndicatesIndexLock(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
