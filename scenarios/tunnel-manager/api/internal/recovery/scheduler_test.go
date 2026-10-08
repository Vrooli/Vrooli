package recovery_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"sync"
	"testing"
	"time"

	"tunnel-manager/internal/recovery"

	"github.com/stretchr/testify/require"
)

type recoverySchedulerService struct {
	mu     sync.Mutex
	calls  int
	errs   []error
	events []recovery.RecoveryEvent
	acted  []bool
	called chan int
}

func newRecoverySchedulerService() *recoverySchedulerService {
	return &recoverySchedulerService{called: make(chan int, 8)}
}

func (s *recoverySchedulerService) GetState(context.Context) (recovery.RecoveryState, error) {
	panic("unused")
}

func (s *recoverySchedulerService) ListEvents(context.Context, int) ([]recovery.RecoveryEvent, error) {
	panic("unused")
}

func (s *recoverySchedulerService) Recover(context.Context, bool) (recovery.EventOutcome, recovery.RecoveryEvent, error) {
	panic("unused")
}

func (s *recoverySchedulerService) Evaluate(context.Context) (recovery.RecoveryEvent, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	call := s.calls
	s.calls++
	s.called <- s.calls
	if call < len(s.errs) && s.errs[call] != nil {
		return recovery.RecoveryEvent{}, false, s.errs[call]
	}
	if call < len(s.events) {
		acted := call < len(s.acted) && s.acted[call]
		return s.events[call], acted, nil
	}
	return recovery.RecoveryEvent{}, false, nil
}

func (s *recoverySchedulerService) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestRecoverySchedulerRunsBootAndPeriodicEvaluation(t *testing.T) {
	ticks := make(chan time.Time)
	svc := newRecoverySchedulerService()
	svc.events = []recovery.RecoveryEvent{
		{Action: recovery.ActionRestart, Outcome: recovery.OutcomeSuccess, Attempt: 1},
		{Action: recovery.ActionRestart, Outcome: recovery.OutcomeSkipped, Attempt: 2},
	}
	svc.acted = []bool{true, true}
	var logs bytes.Buffer
	scheduler, err := recovery.NewScheduler(recovery.SchedulerConfig{
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
	waitRecoveryCall(t, svc, 1)

	ticks <- time.Now()
	waitRecoveryCall(t, svc, 2)

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after context cancellation")
	}
	require.Equal(t, 2, svc.count())
	require.Contains(t, logs.String(), "outcome=success")
	require.Contains(t, logs.String(), "outcome=skipped")
}

func TestRecoverySchedulerContinuesAfterFailure(t *testing.T) {
	ticks := make(chan time.Time)
	svc := newRecoverySchedulerService()
	svc.errs = []error{errors.New("sqlite unavailable")}
	svc.events = []recovery.RecoveryEvent{{}, {Action: recovery.ActionRestart, Outcome: recovery.OutcomeFailure, Attempt: 1}}
	svc.acted = []bool{false, true}
	var logs bytes.Buffer
	scheduler, err := recovery.NewScheduler(recovery.SchedulerConfig{
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
	waitRecoveryCall(t, svc, 1)
	require.Contains(t, logs.String(), "failed (will retry)")

	ticks <- time.Now()
	waitRecoveryCall(t, svc, 2)
	require.Contains(t, logs.String(), "outcome=failure")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after context cancellation")
	}
}

func TestNewRecoverySchedulerRequiresService(t *testing.T) {
	_, err := recovery.NewScheduler(recovery.SchedulerConfig{})
	require.Error(t, err)
}

func waitRecoveryCall(t *testing.T, svc *recoverySchedulerService, want int) {
	t.Helper()
	for {
		select {
		case got := <-svc.called:
			if got >= want {
				return
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for recovery call %d; got %d", want, svc.count())
		}
	}
}
