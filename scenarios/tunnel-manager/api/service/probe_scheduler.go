package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// ProbeScheduler runs probe cycles at a configurable interval.
type ProbeScheduler struct {
	probeRunner ProbeRunner
	interval    time.Duration
	mu          sync.Mutex
	running     bool
	cancel      context.CancelFunc
	lastRun     time.Time
	lastErr     error
}

func NewProbeScheduler(probeRunner ProbeRunner, interval time.Duration) *ProbeScheduler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &ProbeScheduler{
		probeRunner: probeRunner,
		interval:    interval,
	}
}

// Start begins the periodic probe cycle. It is safe to call multiple times;
// subsequent calls are no-ops if already running.
func (ps *ProbeScheduler) Start(ctx context.Context) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.running {
		return
	}

	innerCtx, cancel := context.WithCancel(ctx)
	ps.cancel = cancel
	ps.running = true

	go ps.loop(innerCtx)
}

// Stop halts the periodic probe cycle.
func (ps *ProbeScheduler) Stop() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if !ps.running {
		return
	}
	ps.cancel()
	ps.running = false
}

// IsRunning returns whether the scheduler is currently active.
func (ps *ProbeScheduler) IsRunning() bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.running
}

// LastRun returns the time of the last completed probe cycle.
func (ps *ProbeScheduler) LastRun() time.Time {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.lastRun
}

// LastError returns the error from the most recent probe cycle, if any.
func (ps *ProbeScheduler) LastError() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.lastErr
}

func (ps *ProbeScheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(ps.interval)
	defer ticker.Stop()

	// Run immediately on start
	ps.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			ps.mu.Lock()
			ps.running = false
			ps.mu.Unlock()
			return
		case <-ticker.C:
			ps.runOnce(ctx)
		}
	}
}

func (ps *ProbeScheduler) runOnce(ctx context.Context) {
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := ps.probeRunner.RunAll(probeCtx)

	ps.mu.Lock()
	ps.lastRun = time.Now()
	ps.lastErr = err
	ps.mu.Unlock()

	if err != nil {
		slog.Error("probe cycle failed", "component", "probe-scheduler", "error", err)
	}
}
