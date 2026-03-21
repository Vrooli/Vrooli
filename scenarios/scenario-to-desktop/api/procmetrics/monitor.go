package procmetrics

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	// resourcePollInterval is how often resource samples are collected.
	resourcePollInterval = 1 * time.Second
	// windowPollInterval is how often we check for a visible window.
	windowPollInterval = 250 * time.Millisecond
	// windowDetectTimeout is how long to wait for a window before giving up.
	windowDetectTimeout = 60 * time.Second
	// clockTicksPerSec is the standard Linux clock ticks per second (sysconf _SC_CLK_TCK).
	clockTicksPerSec = 100
	// readySizeRatio is the fraction of expected dimensions a window must meet
	// to be considered the main app window (not a splash screen).
	readySizeRatio = 0.5
)

// DefaultMonitor implements Monitor using ProcReader and WindowDetector.
type DefaultMonitor struct {
	proc   ProcReader
	window WindowDetector
	logger *slog.Logger

	cancel context.CancelFunc
	done   chan struct{}

	mu      sync.Mutex
	report  Report
	stopped bool

	// Previous sample state for CPU delta calculation.
	prevUtime   int64
	prevStime   int64
	prevSampleT time.Time
}

// NewDefaultMonitor creates a monitor with the given dependencies.
func NewDefaultMonitor(proc ProcReader, window WindowDetector, logger *slog.Logger) *DefaultMonitor {
	return &DefaultMonitor{
		proc:   proc,
		window: window,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start begins monitoring the given PID on the given X display.
// expectedWidth/expectedHeight define the main window size threshold:
//   - Phase 1 (splash): any visible window on the display
//   - Phase 2 (ready): a window at least readySizeRatio of expected dimensions
//
// If expectedWidth/expectedHeight are 0, any visible window counts as ready immediately.
func (m *DefaultMonitor) Start(ctx context.Context, pid int, display string, expectedWidth, expectedHeight int) error {
	if !m.proc.IsAlive(pid) {
		return fmt.Errorf("process %d is not alive", pid)
	}

	ctx, m.cancel = context.WithCancel(ctx)
	m.report.Startup.LaunchAt = time.Now()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		m.pollResources(ctx, pid)
	}()

	go func() {
		defer wg.Done()
		m.detectWindow(ctx, pid, display, expectedWidth, expectedHeight)
	}()

	go func() {
		wg.Wait()
		m.computeSummary()
		close(m.done)
	}()

	return nil
}

// Stop halts monitoring and computes the summary. Safe to call multiple times.
func (m *DefaultMonitor) Stop() {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return
	}
	m.stopped = true
	m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}
	<-m.done
}

// Report returns the current metrics report. Safe to call while running.
func (m *DefaultMonitor) Report() *Report {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a snapshot copy
	r := Report{
		Startup: m.report.Startup,
		Summary: m.report.Summary,
	}
	if len(m.report.Samples) > 0 {
		r.Samples = make([]Sample, len(m.report.Samples))
		copy(r.Samples, m.report.Samples)
	}
	return &r
}

// Done returns a channel that closes when monitoring has finished.
func (m *DefaultMonitor) Done() <-chan struct{} {
	return m.done
}

// pollResources samples /proc stats at regular intervals.
func (m *DefaultMonitor) pollResources(ctx context.Context, pid int) {
	// Take an initial baseline reading for CPU delta calculation.
	utime, stime, err := m.proc.ReadStat(pid)
	if err == nil {
		m.prevUtime = utime
		m.prevStime = stime
		m.prevSampleT = time.Now()
	}

	ticker := time.NewTicker(resourcePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !m.proc.IsAlive(pid) {
				return
			}

			sample := m.collectSample(pid)
			if sample != nil {
				m.mu.Lock()
				m.report.Samples = append(m.report.Samples, *sample)
				m.mu.Unlock()
			}
		}
	}
}

// collectSample reads /proc and computes a resource sample.
func (m *DefaultMonitor) collectSample(pid int) *Sample {
	now := time.Now()

	rssBytes, peakBytes, threads, err := m.proc.ReadStatus(pid)
	if err != nil {
		m.logger.Debug("failed to read /proc/pid/status", "error", err)
		return nil
	}

	utime, stime, err := m.proc.ReadStat(pid)
	if err != nil {
		m.logger.Debug("failed to read /proc/pid/stat", "error", err)
		return nil
	}

	// Compute CPU% from tick deltas since last sample.
	var cpuPercent float64
	elapsed := now.Sub(m.prevSampleT).Seconds()
	if elapsed > 0 && m.prevSampleT != (time.Time{}) {
		deltaTicks := float64((utime - m.prevUtime) + (stime - m.prevStime))
		cpuPercent = (deltaTicks / clockTicksPerSec) / elapsed * 100
	}

	m.prevUtime = utime
	m.prevStime = stime
	m.prevSampleT = now

	return &Sample{
		Timestamp:  now,
		CPUPercent: cpuPercent,
		RSSBytes:   rssBytes,
		PeakBytes:  peakBytes,
		Threads:    threads,
	}
}

// detectWindow polls for visible X11 windows in two phases:
//
//  1. Splash — records the instant any visible window appears.
//  2. Ready  — continues polling until a window meets the size threshold
//     (readySizeRatio of expectedWidth × expectedHeight). If no expected
//     dimensions are given, phase 1 counts as ready too.
func (m *DefaultMonitor) detectWindow(ctx context.Context, pid int, display string, expectedWidth, expectedHeight int) {
	if display == "" {
		return
	}

	minWidth := int(float64(expectedWidth) * readySizeRatio)
	minHeight := int(float64(expectedHeight) * readySizeRatio)
	needSizeCheck := minWidth > 0 && minHeight > 0

	splashDetected := false

	deadline := time.After(windowDetectTimeout)
	ticker := time.NewTicker(windowPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline:
			m.logger.Debug("window detection timed out", "pid", pid, "timeout", windowDetectTimeout)
			return
		case <-ticker.C:
			// Phase 1: detect any visible window (splash).
			if !splashDetected {
				visible, err := m.window.HasVisibleWindow(ctx, pid, display)
				if err != nil {
					m.logger.Debug("window detection error", "error", err)
					continue
				}
				if !visible {
					continue
				}

				now := time.Now()
				dur := now.Sub(m.report.Startup.LaunchAt).Milliseconds()
				m.mu.Lock()
				m.report.Startup.SplashVisibleAt = &now
				m.report.Startup.SplashDurationMs = &dur
				m.mu.Unlock()
				splashDetected = true
				m.logger.Info("splash window detected", "pid", pid, "splash_ms", dur)

				// If no size threshold, splash = ready.
				if !needSizeCheck {
					m.setReady(now, dur, pid)
					return
				}
			}

			// Phase 2: wait for a window that meets the size threshold.
			geo, err := m.window.LargestVisibleWindow(ctx, pid, display)
			if err != nil {
				m.logger.Debug("window geometry error", "error", err)
				continue
			}
			if geo != nil && geo.Width >= minWidth && geo.Height >= minHeight {
				now := time.Now()
				dur := now.Sub(m.report.Startup.LaunchAt).Milliseconds()
				m.setReady(now, dur, pid)
				return
			}
		}
	}
}

// setReady records the ready timing under lock.
func (m *DefaultMonitor) setReady(now time.Time, durMs int64, pid int) {
	m.mu.Lock()
	m.report.Startup.ReadyAt = &now
	m.report.Startup.ReadyMs = &durMs
	m.mu.Unlock()
	m.logger.Info("app window ready", "pid", pid, "ready_ms", durMs)
}

// computeSummary calculates aggregate stats from collected samples.
func (m *DefaultMonitor) computeSummary() {
	m.mu.Lock()
	defer m.mu.Unlock()

	samples := m.report.Samples
	if len(samples) == 0 {
		return
	}

	summary := &Summary{
		SampleCount: len(samples),
	}

	var totalRSS int64
	var totalCPU float64
	// Skip the first sample for CPU averages (no baseline delta).
	cpuSampleCount := 0

	for i, s := range samples {
		if s.RSSBytes > summary.PeakRSSBytes {
			summary.PeakRSSBytes = s.RSSBytes
		}
		totalRSS += s.RSSBytes

		if s.CPUPercent > summary.PeakCPU {
			summary.PeakCPU = s.CPUPercent
		}
		// Skip first sample for CPU average since it has no meaningful delta.
		if i > 0 {
			totalCPU += s.CPUPercent
			cpuSampleCount++
		}

		if s.Threads > summary.MaxThreads {
			summary.MaxThreads = s.Threads
		}
	}

	if len(samples) > 0 {
		summary.AvgRSSBytes = totalRSS / int64(len(samples))
	}
	if cpuSampleCount > 0 {
		summary.AvgCPU = totalCPU / float64(cpuSampleCount)
	}

	first := samples[0].Timestamp
	last := samples[len(samples)-1].Timestamp
	summary.DurationMs = last.Sub(first).Milliseconds()

	m.report.Summary = summary
}
