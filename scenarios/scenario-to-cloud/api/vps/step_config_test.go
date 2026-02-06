package vps

import (
	"context"
	"errors"
	"testing"
	"time"

	"scenario-to-cloud/ssh"
)

func TestRunOptionsForStep_KnownStep(t *testing.T) {
	t.Parallel()

	opts := RunOptionsForStep("bootstrap")
	if opts.CommandTimeout != 2*time.Minute {
		t.Errorf("bootstrap CommandTimeout = %v, want %v", opts.CommandTimeout, 2*time.Minute)
	}
	// Should still have default connection settings
	if opts.ConnectTimeout != 5*time.Second {
		t.Errorf("ConnectTimeout = %v, want %v", opts.ConnectTimeout, 5*time.Second)
	}
}

func TestRunOptionsForStep_UnknownStep(t *testing.T) {
	t.Parallel()

	opts := RunOptionsForStep("nonexistent_step")
	defaults := ssh.DefaultRunOptions()
	if opts.CommandTimeout != defaults.CommandTimeout {
		t.Errorf("unknown step CommandTimeout = %v, want %v", opts.CommandTimeout, defaults.CommandTimeout)
	}
	if opts.ConnectTimeout != defaults.ConnectTimeout {
		t.Errorf("unknown step ConnectTimeout = %v, want %v", opts.ConnectTimeout, defaults.ConnectTimeout)
	}
}

// fakeRunner implements ssh.Runner for testing retry logic.
type fakeRunner struct {
	calls    int
	results  []fakeResult
	commands []string
}

type fakeResult struct {
	res ssh.Result
	err error
}

func (f *fakeRunner) Run(_ context.Context, _ ssh.Config, cmd string, _ ssh.RunOptions) (ssh.Result, error) {
	f.commands = append(f.commands, cmd)
	idx := f.calls
	f.calls++
	if idx < len(f.results) {
		return f.results[idx].res, f.results[idx].err
	}
	return ssh.Result{}, nil
}

func TestRunStepWithRetry_SuccessFirstAttempt(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{
		results: []fakeResult{
			{res: ssh.Result{ExitCode: 0}},
		},
	}

	err := RunStepWithRetry(context.Background(), runner, ssh.Config{Host: "test"}, "resource_start", "echo ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.calls != 1 {
		t.Errorf("calls = %d, want 1", runner.calls)
	}
}

func TestRunStepWithRetry_RetriesOnRetryableError(t *testing.T) {
	t.Parallel()

	retryableErr := &ssh.SSHError{
		Category:  ssh.ErrTimeout,
		Message:   "connection timed out",
		Retryable: true,
	}
	runner := &fakeRunner{
		results: []fakeResult{
			{err: retryableErr},
			{res: ssh.Result{ExitCode: 0}},
		},
	}

	err := RunStepWithRetry(context.Background(), runner, ssh.Config{Host: "test"}, "resource_start", "start postgres")
	if err != nil {
		t.Fatalf("unexpected error after retry: %v", err)
	}
	if runner.calls != 2 {
		t.Errorf("calls = %d, want 2 (initial + 1 retry)", runner.calls)
	}
}

func TestRunStepWithRetry_NoRetryOnNonRetryableError(t *testing.T) {
	t.Parallel()

	nonRetryableErr := &ssh.SSHError{
		Category:  ssh.ErrAuth,
		Message:   "auth failed",
		Retryable: false,
	}
	runner := &fakeRunner{
		results: []fakeResult{
			{err: nonRetryableErr},
		},
	}

	err := RunStepWithRetry(context.Background(), runner, ssh.Config{Host: "test"}, "resource_start", "start postgres")
	if err == nil {
		t.Fatal("expected error for non-retryable failure")
	}
	if runner.calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry for non-retryable)", runner.calls)
	}
}

func TestRunStepWithRetry_RespectsMaxRetries(t *testing.T) {
	t.Parallel()

	retryableErr := &ssh.SSHError{
		Category:  ssh.ErrTimeout,
		Message:   "connection timed out",
		Retryable: true,
	}
	runner := &fakeRunner{
		results: []fakeResult{
			{err: retryableErr},
			{err: retryableErr},
		},
	}

	err := RunStepWithRetry(context.Background(), runner, ssh.Config{Host: "test"}, "resource_start", "start postgres")
	if err == nil {
		t.Fatal("expected error when all retries exhausted")
	}
	// resource_start has MaxRetries=1, so 2 total attempts (initial + 1 retry)
	if runner.calls != 2 {
		t.Errorf("calls = %d, want 2", runner.calls)
	}
}

func TestRunStepWithRetry_NoRetryForStepWithoutConfig(t *testing.T) {
	t.Parallel()

	retryableErr := &ssh.SSHError{
		Category:  ssh.ErrTimeout,
		Message:   "timed out",
		Retryable: true,
	}
	runner := &fakeRunner{
		results: []fakeResult{
			{err: retryableErr},
		},
	}

	err := RunStepWithRetry(context.Background(), runner, ssh.Config{Host: "test"}, "unknown_step", "echo test")
	if err == nil {
		t.Fatal("expected error")
	}
	if runner.calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry for unknown step)", runner.calls)
	}
}

func TestRunStepWithRetry_ContextCancellation(t *testing.T) {
	t.Parallel()

	retryableErr := &ssh.SSHError{
		Category:  ssh.ErrTimeout,
		Message:   "timed out",
		Retryable: true,
	}
	runner := &fakeRunner{
		results: []fakeResult{
			{err: retryableErr},
			{err: retryableErr},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := RunStepWithRetry(ctx, runner, ssh.Config{Host: "test"}, "resource_start", "start postgres")
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		// Either context.Canceled or the retryable error - both are acceptable
		var sshErr *ssh.SSHError
		if !errors.As(err, &sshErr) {
			t.Errorf("expected context.Canceled or SSHError, got %T: %v", err, err)
		}
	}
}
