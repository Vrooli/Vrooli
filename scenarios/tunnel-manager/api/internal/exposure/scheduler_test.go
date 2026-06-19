package exposure_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"sync"
	"testing"
	"time"

	"tunnel-manager/internal/exposure"

	"github.com/stretchr/testify/require"
)

type schedulerService struct {
	mu      sync.Mutex
	calls   int
	errs    []error
	results []struct {
		core   int
		reaped int
	}
	called chan int
}

func newSchedulerService() *schedulerService {
	return &schedulerService{called: make(chan int, 8)}
}

func (s *schedulerService) Expose(context.Context, exposure.ExposeInput) (exposure.Lease, string, error) {
	panic("unused")
}

func (s *schedulerService) ExtendLease(context.Context, string, time.Duration) (exposure.Lease, error) {
	panic("unused")
}

func (s *schedulerService) RevokeLease(context.Context, string) (bool, error) {
	panic("unused")
}

func (s *schedulerService) ListLeases(context.Context, exposure.LeaseStatus) ([]exposure.Lease, error) {
	panic("unused")
}

func (s *schedulerService) ListExposures(context.Context) ([]exposure.Exposure, error) {
	panic("unused")
}

func (s *schedulerService) IsExposed(context.Context, string) (bool, string, error) {
	panic("unused")
}

func (s *schedulerService) Reconcile(context.Context) (int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	call := s.calls
	s.calls++
	s.called <- s.calls
	if call < len(s.errs) && s.errs[call] != nil {
		return 0, 0, s.errs[call]
	}
	if call < len(s.results) {
		return s.results[call].core, s.results[call].reaped, nil
	}
	return 0, 0, nil
}

func (s *schedulerService) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestSchedulerRunsBootAndPeriodicReconcile(t *testing.T) {
	ticks := make(chan time.Time)
	svc := newSchedulerService()
	var logs bytes.Buffer
	scheduler, err := exposure.NewScheduler(exposure.SchedulerConfig{
		Service: svc,
		Ticks:   ticks,
		Logger:  log.New(&logs, "", 0),
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		scheduler.Run(ctx)
	}()
	waitCall(t, svc, 1)

	ticks <- time.Now()
	waitCall(t, svc, 2)

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after context cancellation")
	}
	require.Equal(t, 2, svc.count())
}

func TestSchedulerContinuesAfterReconcileFailure(t *testing.T) {
	ticks := make(chan time.Time)
	svc := newSchedulerService()
	svc.errs = []error{errors.New("remote unavailable")}
	svc.results = []struct {
		core   int
		reaped int
	}{{}, {core: 1, reaped: 2}}
	var logs bytes.Buffer
	scheduler, err := exposure.NewScheduler(exposure.SchedulerConfig{
		Service: svc,
		Ticks:   ticks,
		Logger:  log.New(&logs, "", 0),
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		scheduler.Run(ctx)
	}()
	waitCall(t, svc, 1)
	require.Contains(t, logs.String(), "failed (will retry)")

	ticks <- time.Now()
	waitCall(t, svc, 2)
	require.Contains(t, logs.String(), "core_ensured=1 leases_reaped=2")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after context cancellation")
	}
}

func TestNewSchedulerRequiresService(t *testing.T) {
	_, err := exposure.NewScheduler(exposure.SchedulerConfig{})
	require.Error(t, err)
}

func waitCall(t *testing.T, svc *schedulerService, want int) {
	t.Helper()
	for {
		select {
		case got := <-svc.called:
			if got >= want {
				return
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for reconcile call %d; got %d", want, svc.count())
		}
	}
}
