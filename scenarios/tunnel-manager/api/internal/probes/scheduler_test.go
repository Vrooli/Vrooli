package probes_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"sync"
	"testing"
	"time"

	"tunnel-manager/internal/probes"

	"github.com/stretchr/testify/require"
)

type probeSchedulerService struct {
	mu      sync.Mutex
	calls   int
	errs    []error
	results [][]probes.ProbeResult
	called  chan int
}

func newProbeSchedulerService() *probeSchedulerService {
	return &probeSchedulerService{called: make(chan int, 8)}
}

func (s *probeSchedulerService) RunProbes(context.Context) ([]probes.ProbeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	call := s.calls
	s.calls++
	s.called <- s.calls
	if call < len(s.errs) && s.errs[call] != nil {
		return nil, s.errs[call]
	}
	if call < len(s.results) {
		return s.results[call], nil
	}
	return nil, nil
}

func (s *probeSchedulerService) ListProbes(context.Context, string, int) ([]probes.ProbeResult, error) {
	panic("unused")
}

func (s *probeSchedulerService) Classify(context.Context) ([]probes.RouteClassification, error) {
	panic("unused")
}

func (s *probeSchedulerService) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestProbeSchedulerRunsBootAndPeriodicProbes(t *testing.T) {
	ticks := make(chan time.Time)
	svc := newProbeSchedulerService()
	svc.results = [][]probes.ProbeResult{{{Subdomain: "core"}}, {{Subdomain: "core"}}}
	var logs bytes.Buffer
	scheduler, err := probes.NewScheduler(probes.SchedulerConfig{
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
	waitProbeCall(t, svc, 1)

	ticks <- time.Now()
	waitProbeCall(t, svc, 2)

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after context cancellation")
	}
	require.Equal(t, 2, svc.count())
	require.Contains(t, logs.String(), "recorded results=1")
}

func TestProbeSchedulerContinuesAfterFailure(t *testing.T) {
	ticks := make(chan time.Time)
	svc := newProbeSchedulerService()
	svc.errs = []error{errors.New("routes unavailable")}
	svc.results = [][]probes.ProbeResult{{}, {{Subdomain: "core"}}}
	var logs bytes.Buffer
	scheduler, err := probes.NewScheduler(probes.SchedulerConfig{
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
	waitProbeCall(t, svc, 1)
	require.Contains(t, logs.String(), "failed (will retry)")

	ticks <- time.Now()
	waitProbeCall(t, svc, 2)
	require.Contains(t, logs.String(), "recorded results=1")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after context cancellation")
	}
}

func TestNewProbeSchedulerRequiresService(t *testing.T) {
	_, err := probes.NewScheduler(probes.SchedulerConfig{})
	require.Error(t, err)
}

func waitProbeCall(t *testing.T, svc *probeSchedulerService, want int) {
	t.Helper()
	for {
		select {
		case got := <-svc.called:
			if got >= want {
				return
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for probe call %d; got %d", want, svc.count())
		}
	}
}
