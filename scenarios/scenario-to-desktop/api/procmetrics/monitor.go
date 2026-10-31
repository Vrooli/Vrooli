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
	// to be considered the main app window (not a splash screen). Set high
	// (90%) because on our dedicated virtual displays, the main app window
	// fills the display while splash screens are much smaller.
	readySizeRatio = 0.9
)

// DefaultMonitor implements Monitor using ProcReader and WindowDetector.
type DefaultMonitor struct {
	proc   ProcReader
	tree   ProcessTreeReader
	window WindowDetector
	logger *slog.Logger

	cancel context.CancelFunc
	done   chan struct{}

	mu      sync.Mutex
	report  Report
	stopped bool

	// Previous sample state for CPU delta calculation.
	prevUtime          int64
	prevStime          int64
	prevSampleT        time.Time
	treePrev           map[int]int64
	treeAt             time.Time
	treeStartedAt      time.Time
	treeSeen           map[ProcessRole]map[int]struct{}
	treeCPUAccum       map[ProcessRole]float64
	treeCPUSamples     map[ProcessRole]int
	treeWindowPoints   []treeWindowPoint
	processTreeOptions ProcessTreeOptions
}

type treeWindowPoint struct {
	at              time.Time
	cpuPercent      float64
	cpuAvailable    bool
	pssBytes        int64
	ussBytes        int64
	memoryAvailable bool
	roles           map[ProcessRole]RoleSummary
}

// ProcessTreeWindow is a bounded aggregate for a requested interval.
type ProcessTreeWindow struct {
	AverageCPUPercent float64
	PSSBytes          int64
	USSBytes          int64
	Samples           int
	Available         bool
	CPUAvailable      bool
	MemoryAvailable   bool
	Reason            string
}

// NewDefaultMonitor creates a monitor with the given dependencies.
func NewDefaultMonitor(proc ProcReader, window WindowDetector, logger *slog.Logger) *DefaultMonitor {
	return NewDefaultMonitorWithTree(proc, nil, window, logger)
}

func NewDefaultMonitorWithTree(proc ProcReader, tree ProcessTreeReader, window WindowDetector, logger *slog.Logger) *DefaultMonitor {
	return &DefaultMonitor{
		proc:           proc,
		tree:           tree,
		window:         window,
		logger:         logger,
		done:           make(chan struct{}),
		treePrev:       make(map[int]int64),
		treeSeen:       make(map[ProcessRole]map[int]struct{}),
		treeCPUAccum:   make(map[ProcessRole]float64),
		treeCPUSamples: make(map[ProcessRole]int),
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
	m.report.ProcessTree = newProcessTreeReport(m.tree != nil)
	m.collectTreeSample(pid)

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
		Startup:     m.report.Startup,
		Summary:     m.report.Summary,
		ProcessTree: m.report.ProcessTree,
	}
	if len(m.report.Samples) > 0 {
		r.Samples = make([]Sample, len(m.report.Samples))
		copy(r.Samples, m.report.Samples)
	}
	if m.report.ProcessTree != nil {
		r.ProcessTree = &ProcessTreeReport{Supported: m.report.ProcessTree.Supported, Scope: m.report.ProcessTree.Scope, Roles: make(map[ProcessRole]RoleSummary, len(m.report.ProcessTree.Roles))}
		for role, summary := range m.report.ProcessTree.Roles {
			r.ProcessTree.Roles[role] = summary
		}
	}
	return &r
}

// Window returns idle CPU averaged over the requested interval and the final
// process-tree PSS/USS point captured inside it. Sparse or missing observations
// remain unavailable rather than being represented as zero.
func (m *DefaultMonitor) Window(start, end time.Time) ProcessTreeWindow {
	if m == nil || !end.After(start) {
		return ProcessTreeWindow{Reason: "a positive process-tree window is required"}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var cpuTotal float64
	cpuSamples := 0
	var final treeWindowPoint
	var firstAt time.Time
	var lastAt time.Time
	for _, point := range m.treeWindowPoints {
		if point.at.Before(start) || point.at.After(end) {
			continue
		}
		if point.cpuAvailable {
			cpuTotal += point.cpuPercent
			cpuSamples++
			if firstAt.IsZero() {
				firstAt = point.at
			}
			lastAt = point.at
		}
		if point.memoryAvailable {
			final = point
		}
	}
	cpuAvailable := cpuSamples >= 2 && firstAt.Sub(start) <= 2*resourcePollInterval && end.Sub(lastAt) <= 2*resourcePollInterval
	memoryAvailable := !final.at.IsZero() && end.Sub(final.at) <= 2*resourcePollInterval
	if !cpuAvailable && !memoryAvailable {
		return ProcessTreeWindow{Samples: cpuSamples, Reason: "process-tree observations do not cover the requested window"}
	}
	return ProcessTreeWindow{AverageCPUPercent: cpuTotal / float64(max(cpuSamples, 1)), PSSBytes: final.pssBytes, USSBytes: final.ussBytes, Samples: cpuSamples, Available: cpuAvailable && memoryAvailable, CPUAvailable: cpuAvailable, MemoryAvailable: memoryAvailable, Reason: "process-tree observations cover the requested window"}
}

// SnapshotAt returns the closest process-tree observation to the requested
// mark when the observation is within two resource polling intervals.
func (m *DefaultMonitor) SnapshotAt(at time.Time) (*ProcessTreeReport, bool) {
	if m == nil || at.IsZero() {
		return nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var closest *treeWindowPoint
	var distance time.Duration
	for i := range m.treeWindowPoints {
		point := &m.treeWindowPoints[i]
		candidate := point.at.Sub(at)
		if candidate < 0 {
			candidate = -candidate
		}
		if closest == nil || candidate < distance {
			closest, distance = point, candidate
		}
	}
	if closest == nil || distance > 2*resourcePollInterval || !closest.memoryAvailable {
		return nil, false
	}
	report := &ProcessTreeReport{Supported: true, Scope: "linux:/proc", Roles: make(map[ProcessRole]RoleSummary, len(closest.roles))}
	for role, summary := range closest.roles {
		report.Roles[role] = summary
	}
	return report, true
}

// Done returns a channel that closes when monitoring has finished.
func (m *DefaultMonitor) Done() <-chan struct{} {
	return m.done
}

// pollResources samples /proc stats at regular intervals.
func (m *DefaultMonitor) pollResources(ctx context.Context, pid int) {
	samplePID := pid
	identity, hasIdentity := m.tree.(ProcessIdentityReader)
	if hasIdentity {
		samplePID = 0 // the launcher is not an app-resource sample
	}
	setBaseline := func(candidate int) {
		utime, stime, err := m.proc.ReadStat(candidate)
		if err == nil {
			m.prevUtime = utime
			m.prevStime = stime
			m.prevSampleT = time.Now()
		}
	}
	if !hasIdentity {
		setBaseline(samplePID)
	}
	// Record a t0 sample so short-lived launches still have a truthful sample
	// series and duration. The first CPU value remains zero because it has no
	// prior interval.
	if !hasIdentity {
		if sample := m.collectSample(samplePID); sample != nil {
			m.mu.Lock()
			m.report.Samples = append(m.report.Samples, *sample)
			m.mu.Unlock()
		}
	}
	ticker := time.NewTicker(resourcePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if hasIdentity {
				if resolved, err := identity.ApplicationPID(pid, m.processTreeOptions); err == nil && resolved != 0 {
					samplePID = resolved
					if m.prevSampleT.IsZero() {
						setBaseline(samplePID)
					}
				}
				// The launcher tree remains the attribution boundary even when
				// no single application PID can be resolved (or that process exits).
				m.collectTreeSample(pid)
			}
			if samplePID == 0 {
				continue
			}
			if !m.proc.IsAlive(samplePID) {
				if hasIdentity {
					continue
				}
				return
			}

			sample := m.collectSample(samplePID)
			if sample != nil {
				m.mu.Lock()
				m.report.Samples = append(m.report.Samples, *sample)
				m.mu.Unlock()
			}
			if !hasIdentity {
				m.collectTreeSample(pid)
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

func newProcessTreeReport(supported bool) *ProcessTreeReport {
	result := &ProcessTreeReport{Supported: supported, Scope: "unavailable", Roles: make(map[ProcessRole]RoleSummary)}
	if supported {
		result.Scope = "linux:/proc"
	}
	for _, role := range []ProcessRole{RoleElectronMain, RoleElectronRender, RoleElectronGPU, RoleElectronUtility, RoleElectronCrashpad, RoleBundledRuntime, RoleScenarioService, RoleLauncher, RoleShellMain, RoleWebviewRenderer, RoleWebviewGPU, RoleWebviewOther, RoleBackend} {
		result.Roles[role] = RoleSummary{Role: role, Unsupported: !supported}
	}
	return result
}

func (m *DefaultMonitor) collectTreeSample(rootPID int) {
	if m.tree == nil {
		return
	}
	var processes []ProcessInfo
	var err error
	if reader, ok := m.tree.(ProcessTreeReaderWithOptions); ok {
		processes, err = reader.ProcessTreeWithOptions(rootPID, m.processTreeOptions)
	} else {
		processes, err = m.tree.ProcessTree(rootPID)
	}
	if err != nil {
		m.logger.Debug("process tree unavailable", "pid", rootPID, "error", err)
		return
	}
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.report.ProcessTree == nil {
		m.report.ProcessTree = newProcessTreeReport(true)
	}
	if m.treeStartedAt.IsZero() {
		m.treeStartedAt = now
	}
	currentRSS := make(map[ProcessRole]int64)
	currentPSS := make(map[ProcessRole]int64)
	currentUSS := make(map[ProcessRole]int64)
	currentMemoryComplete := make(map[ProcessRole]bool)
	currentThreads := make(map[ProcessRole]int)
	currentCPU := make(map[ProcessRole]float64)
	currentCPUSamples := make(map[ProcessRole]int)
	for _, process := range processes {
		role := process.Role
		if role == "" {
			role = RoleUnknown
		}
		if m.treeSeen[role] == nil {
			m.treeSeen[role] = make(map[int]struct{})
		}
		summary := m.report.ProcessTree.Roles[role]
		summary.Role = role
		summary.Available = true
		summary.Unsupported = false
		if _, seen := m.treeSeen[role][process.PID]; !seen {
			m.treeSeen[role][process.PID] = struct{}{}
			summary.ProcessCount++
		}
		currentRSS[role] += process.RSSBytes
		if _, initialized := currentMemoryComplete[role]; !initialized {
			currentMemoryComplete[role] = true
		}
		if process.MemoryStatsAvailable {
			currentPSS[role] += process.PSSBytes
			currentUSS[role] += process.USSBytes
		} else {
			currentMemoryComplete[role] = false
		}
		currentThreads[role] += process.Threads
		if previous, ok := m.treePrev[process.PID]; ok {
			elapsed := now.Sub(m.treeAt).Seconds()
			if elapsed > 0 {
				cpu := float64(process.CPUJiffies-previous) / float64(clockTicksPerSec) / elapsed * 100
				if cpu >= 0 {
					currentCPU[role] += cpu
					currentCPUSamples[role]++
				}
			}
		}
		m.treePrev[process.PID] = process.CPUJiffies
		m.report.ProcessTree.Roles[role] = summary
	}
	for role, rss := range currentRSS {
		summary := m.report.ProcessTree.Roles[role]
		summary.RSSBytes = rss
		summary.MemoryStatsAvailable = currentMemoryComplete[role]
		if summary.MemoryStatsAvailable {
			summary.PSSBytes = currentPSS[role]
			summary.USSBytes = currentUSS[role]
		} else {
			summary.PSSBytes = 0
			summary.USSBytes = 0
		}
		if rss > summary.PeakRSSBytes {
			summary.PeakRSSBytes = rss
		}
		summary.Threads = currentThreads[role]
		summary.SampleCount++
		if samples := currentCPUSamples[role]; samples > 0 {
			m.treeCPUAccum[role] += currentCPU[role]
			m.treeCPUSamples[role] += samples
			summary.CPUPercent = m.treeCPUAccum[role] / float64(m.treeCPUSamples[role])
			if current := currentCPU[role] / float64(samples); current > summary.PeakCPU {
				summary.PeakCPU = current
			}
		}
		m.report.ProcessTree.Roles[role] = summary
	}
	var totalCPU float64
	cpuAvailable := false
	for role, value := range currentCPU {
		totalCPU += value
		cpuAvailable = cpuAvailable || currentCPUSamples[role] > 0
	}
	var totalPSS, totalUSS int64
	memoryAvailable := len(currentRSS) > 0
	for role := range currentRSS {
		if !currentMemoryComplete[role] {
			memoryAvailable = false
			break
		}
		totalPSS += currentPSS[role]
		totalUSS += currentUSS[role]
	}
	m.treeWindowPoints = append(m.treeWindowPoints, treeWindowPoint{
		at: now, cpuPercent: totalCPU, cpuAvailable: cpuAvailable,
		pssBytes: totalPSS, ussBytes: totalUSS, memoryAvailable: memoryAvailable,
		roles: currentRoleSummaries(m.report.ProcessTree.Roles),
	})
	if !m.treeStartedAt.IsZero() {
		for role, summary := range m.report.ProcessTree.Roles {
			summary.DurationMs = now.Sub(m.treeStartedAt).Milliseconds()
			m.report.ProcessTree.Roles[role] = summary
		}
	}
	m.treeAt = now
}

func currentRoleSummaries(roles map[ProcessRole]RoleSummary) map[ProcessRole]RoleSummary {
	result := make(map[ProcessRole]RoleSummary)
	for role, summary := range roles {
		if summary.ProcessCount > 0 {
			result[role] = summary
		}
	}
	return result
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

				// Skip to next tick — don't check size on the same tick as
				// splash detection, since the splash window itself would match.
				continue
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
