package fetch

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
)

// scriptedLease answers Heartbeat from a script; every other lease method is
// unused by the guard.
type scriptedLease struct {
	capacity.DiskJobLease
	mu     sync.Mutex
	script []error
	calls  int
}

func (l *scriptedLease) Heartbeat(context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	if len(l.script) == 0 {
		return nil
	}
	err := l.script[0]
	l.script = l.script[1:]
	return err
}

func shrinkDiskLeaseTiming(t *testing.T, stale time.Duration) {
	t.Helper()
	oldStale, oldRetry, oldAttempt := diskLeaseStaleBudget, diskLeaseRetryInterval, diskLeaseAttemptTimeout
	diskLeaseStaleBudget, diskLeaseRetryInterval, diskLeaseAttemptTimeout = stale, time.Millisecond, 50*time.Millisecond
	t.Cleanup(func() {
		diskLeaseStaleBudget, diskLeaseRetryInterval, diskLeaseAttemptTimeout = oldStale, oldRetry, oldAttempt
	})
}

func TestDiskLeaseGuardRetriesTimedOutHeartbeatsWithinTheStaleBudget(t *testing.T) {
	shrinkDiskLeaseTiming(t, time.Minute)
	lease := &scriptedLease{script: []error{context.DeadlineExceeded, context.DeadlineExceeded, nil}}
	ctx, guard := MaintainDiskLease(context.Background(), lease, time.Millisecond)
	deadline := time.After(2 * time.Second)
	for {
		lease.mu.Lock()
		calls := lease.calls
		lease.mu.Unlock()
		if calls >= 4 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("guard stopped renewing after %d heartbeats", calls)
		case <-time.After(time.Millisecond):
		}
	}
	if ctx.Err() != nil {
		t.Fatal("transient ledger timeouts within the stale budget must not cancel the install")
	}
	if err := guard.Stop(); err != nil {
		t.Fatalf("recovered guard reported %v", err)
	}
}

func TestDiskLeaseGuardCancelsOnDefinitiveRefusalOrExhaustedBudget(t *testing.T) {
	shrinkDiskLeaseTiming(t, time.Minute)
	refused := &scriptedLease{script: []error{errors.New("disk job lease-1 is not active")}}
	ctx, guard := MaintainDiskLease(context.Background(), refused, time.Millisecond)
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("a definitive refusal must cancel the install immediately")
	}
	if err := guard.Stop(); err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("guard error = %v", err)
	}

	shrinkDiskLeaseTiming(t, 0)
	stale := &scriptedLease{script: []error{context.DeadlineExceeded}}
	ctx, guard = MaintainDiskLease(context.Background(), stale, time.Millisecond)
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("timeouts past the stale budget must cancel the install")
	}
	if err := guard.Stop(); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("guard error = %v", err)
	}
}
