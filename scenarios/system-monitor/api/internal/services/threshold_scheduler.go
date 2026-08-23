package services

// DOC: docs/concepts/ARCHITECTURE.md#alerting

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/cleanupmanager"
)

// minThresholdIntervalSeconds bounds how often the loop may evaluate. A
// misconfigured one-second interval would sample the filesystem constantly and
// add its own pressure.
const minThresholdIntervalSeconds = 5

// PressureReporter forwards a band escalation to a remediation service.
// It is an interface so the threshold loop can be tested without a live
// storage-manager, and so a failure to report can be asserted as non-fatal.
type PressureReporter interface {
	ReportPressure(ctx context.Context, report cleanupmanager.Report) (cleanupmanager.Outcome, error)
}

// DiskUsageSource supplies the disk measurement the threshold loop evaluates.
// Production reads a real filesystem; tests inject a controllable source, which
// is the only way to prove the loop fires under pressure without filling a real
// disk.
type DiskUsageSource interface {
	ReadDiskUsage(ctx context.Context) (collectors.DiskUsage, error)
}

// CPUObservationSource supplies the latest scheduler-owned CPU observation.
// Refused states are deliberately preserved so alerting can skip them rather
// than treating an unavailable host as healthy.
type CPUObservationSource interface {
	ReadCPUObservation(ctx context.Context) (models.CPUMetrics, error)
}

// RootDiskUsageSource measures the host root filesystem.
type RootDiskUsageSource struct {
	Path string
}

// ReadDiskUsage measures the configured path, defaulting to the host root.
func (s RootDiskUsageSource) ReadDiskUsage(_ context.Context) (collectors.DiskUsage, error) {
	path := s.Path
	if path == "" {
		path = collectors.RootMountPath()
	}
	return collectors.ReadDiskUsage(path)
}

// ThresholdStatus captures the most recent evaluation for status views.
type ThresholdStatus struct {
	HasRun        bool
	LastRunAt     time.Time
	LastError     string
	LastUsage     collectors.DiskUsage
	LastViolation *models.ThresholdViolation
	Violations    int64
	NextInterval  time.Duration

	// Band is the escalation band the most recent sample resolved to, and
	// LastTransition is the observation that caused the most recent band
	// change — the evidence behind whatever the system did next.
	Band           PressureBand
	LastTransition *BandObservation

	// LastRemediation is what the remediation service last did about this
	// pressure, so one status read answers "did anything actually happen".
	LastRemediation *RemediationResult
}

// RemediationResult records the outcome of escalating to storage-manager.
type RemediationResult struct {
	At                time.Time `json:"at"`
	Band              string    `json:"band"`
	Action            string    `json:"action"`
	PlanID            string    `json:"plan_id,omitempty"`
	EstimatedBytes    int64     `json:"estimated_bytes,omitempty"`
	ReclaimedBytes    int64     `json:"reclaimed_bytes,omitempty"`
	ProvidersApplied  []string  `json:"providers_applied,omitempty"`
	ProvidersWithheld []string  `json:"providers_withheld,omitempty"`
	Error             string    `json:"error,omitempty"`
}

// ThresholdScheduler evaluates disk pressure against the configured threshold
// on a settings-driven interval and turns a crossing into a durable record.
//
// This loop is the piece that did not exist on 2026-07-31. Every component it
// uses — the threshold setting, the violation model, the alert service, the
// SQLite alert repository — was already present, tested, and unreachable:
// server/runtime.go constructed the alert service and immediately discarded
// it. An observation that reaches nobody is not a safeguard, so this type owns
// exactly one responsibility: making the observation arrive somewhere durable.
//
// It deliberately mirrors RetentionScheduler's shape (injected clock, settings
// re-read every tick, optional run-on-startup, stop channel) rather than
// inventing a second scheduling idiom in the same package.
type ThresholdScheduler struct {
	settings  *SettingsManager
	alerts    *AlertService
	repo      repository.ThresholdRepository
	source    DiskUsageSource
	cpuSource CPUObservationSource
	reporter  PressureReporter
	log       *slog.Logger
	clock     Clock

	mu             sync.Mutex
	bands          bandTracker
	hasRun         bool
	lastRunAt      time.Time
	lastErr        error
	lastUsage      collectors.DiskUsage
	lastViolation  *models.ThresholdViolation
	violations     int64
	previousValue  float64
	hasPrevious    bool
	lastBand       PressureBand
	lastRemedy     *RemediationResult
	cpuConsecutive int
	cpuLastBand    PressureBand
	cpuLastEmit    time.Time

	// evaluated is signalled after every completed evaluation when non-nil.
	// Tests use it to synchronise with the loop rather than sleeping.
	evaluated chan struct{}

	cancel context.CancelFunc
	done   chan struct{}
}

// ThresholdSchedulerOption configures a ThresholdScheduler.
type ThresholdSchedulerOption func(*ThresholdScheduler)

// WithThresholdClock sets the clock the loop waits and timestamps through.
func WithThresholdClock(c Clock) ThresholdSchedulerOption {
	return func(s *ThresholdScheduler) {
		if c != nil {
			s.clock = c
		}
	}
}

// WithDiskUsageSource sets the measurement source.
func WithDiskUsageSource(src DiskUsageSource) ThresholdSchedulerOption {
	return func(s *ThresholdScheduler) {
		if src != nil {
			s.source = src
		}
	}
}

func WithCPUObservationSource(src CPUObservationSource) ThresholdSchedulerOption {
	return func(s *ThresholdScheduler) {
		if src != nil {
			s.cpuSource = src
		}
	}
}

// WithPressureReporter sets the remediation service the loop escalates to.
// A nil reporter leaves the loop observation-only.
func WithPressureReporter(r PressureReporter) ThresholdSchedulerOption {
	return func(s *ThresholdScheduler) {
		if r != nil {
			s.reporter = r
		}
	}
}

// WithEvaluationSignal sets a channel signalled after each evaluation.
func WithEvaluationSignal(ch chan struct{}) ThresholdSchedulerOption {
	return func(s *ThresholdScheduler) { s.evaluated = ch }
}

// NewThresholdScheduler creates the evaluation loop. A nil logger defaults to
// slog.Default().
func NewThresholdScheduler(
	settings *SettingsManager,
	alerts *AlertService,
	repo repository.ThresholdRepository,
	log *slog.Logger,
	opts ...ThresholdSchedulerOption,
) *ThresholdScheduler {
	if log == nil {
		log = slog.Default()
	}
	s := &ThresholdScheduler{
		settings: settings,
		alerts:   alerts,
		repo:     repo,
		source:   RootDiskUsageSource{},
		log:      log,
		clock:    RealClock{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start launches the loop. It returns immediately; Stop halts it.
func (s *ThresholdScheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})

	go func() {
		defer close(s.done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.clock.After(s.currentInterval()):
				s.RunOnce(ctx)
			}
		}
	}()
}

// Stop halts the loop and waits for it to exit.
func (s *ThresholdScheduler) Stop() {
	if s.cancel == nil {
		return
	}
	s.cancel()
	<-s.done
}

// currentInterval reads the live evaluation interval from settings.
func (s *ThresholdScheduler) currentInterval() time.Duration {
	seconds := s.settings.GetSettings().ThresholdCheckInterval
	if seconds < minThresholdIntervalSeconds {
		seconds = defaultSettings.ThresholdCheckInterval
	}
	return time.Duration(seconds) * time.Second
}

// RunOnce performs a single evaluation. It is exported so an operator command
// can force an evaluation without waiting for the next tick.
func (s *ThresholdScheduler) RunOnce(ctx context.Context) {
	defer s.signalEvaluated()

	settings := s.settings.GetSettings()

	// An inactive monitor observes nothing. This mirrors MonitorService and
	// keeps a paused monitor from emitting alerts.
	if !settings.Active {
		return
	}

	usage, err := s.source.ReadDiskUsage(ctx)
	if err != nil {
		s.recordRun(usage, err, nil, bandDecision{})
		s.log.Error("disk threshold evaluation failed", "error", err)
		s.evaluateCPU(ctx, settings)
		return
	}

	violation, decision := s.evaluate(settings, usage)
	s.recordRun(usage, nil, violation, decision)
	if violation == nil {
		// CPU is evaluated independently of disk. A missing/refused CPU source
		// is not a passing sample and therefore cannot clear its window.
		s.evaluateCPU(ctx, settings)
		return
	}

	if err := s.repo.SaveThresholdViolation(ctx, violation); err != nil {
		s.log.Error("persist threshold violation failed", "error", err)
	}
	if err := s.alerts.SendThresholdViolation(ctx, violation); err != nil {
		s.log.Error("send threshold violation failed", "error", err)
	}

	s.escalate(ctx, decision, usage)

	s.log.Warn("disk pressure band recorded",
		"band", decision.Band.String(),
		"used_percent", violation.CurrentValue,
		"boundary", violation.ThresholdValue,
		"severity", violation.Severity,
		"trend", violation.Trend,
		"available_bytes", usage.AvailableBytes,
	)
	s.evaluateCPU(ctx, settings)
}

func (s *ThresholdScheduler) evaluateCPU(ctx context.Context, settings Settings) {
	if s.cpuSource == nil {
		return
	}
	cpu, err := s.cpuSource.ReadCPUObservation(ctx)
	if err != nil || cpu.UsageState.Status != "measured" {
		return
	}
	value := cpu.UsageState.Value
	band := classifyCPUBand(value, settings)
	if band == BandNormal {
		s.mu.Lock()
		s.cpuConsecutive = 0
		s.cpuLastBand = BandNormal
		s.mu.Unlock()
		return
	}
	s.mu.Lock()
	s.cpuConsecutive++
	consecutive := s.cpuConsecutive
	prior := s.cpuLastBand
	last := s.cpuLastEmit
	now := s.clock.Now()
	s.cpuLastBand = band
	s.mu.Unlock()
	window := settings.CPUSustainedWindowTicks
	if window < 1 {
		window = 1
	}
	debounce := settings.CPUEscalationDebounceTicks
	if debounce < 1 {
		debounce = 1
	}
	if consecutive < window || consecutive < debounce {
		return
	}
	if prior == band && !last.IsZero() && now.Sub(last) < time.Duration(settings.CPUEscalationCooldownSeconds)*time.Second {
		return
	}
	violation := &models.ThresholdViolation{MetricName: "cpu_usage", CurrentValue: value, ThresholdValue: cpuBandBoundary(band, settings), Severity: band.Severity(), ViolationType: band.String(), Timestamp: now, Details: map[string]interface{}{"mode_breakdown": cpu.ModeBreakdown, "stall": map[string]interface{}{"some_avg10": cpu.StallSomeAvg10, "full_avg10": cpu.StallFullAvg10}, "top_consumers": cpu.TopProcesses}}
	s.mu.Lock()
	s.cpuLastEmit = now
	s.mu.Unlock()
	if err := s.repo.SaveThresholdViolation(ctx, violation); err != nil {
		s.log.Error("persist CPU threshold violation failed", "error", err)
		return
	}
	if s.alerts != nil {
		if err := s.alerts.SendThresholdViolation(ctx, violation); err != nil {
			s.log.Error("send CPU threshold violation failed", "error", err)
		}
	}
}

func classifyCPUBand(value float64, settings Settings) PressureBand {
	switch {
	case value >= settings.CPUCriticalPercent:
		return BandCritical
	case value >= settings.CPUHighPercent:
		return BandHigh
	case value >= settings.CPUThreshold:
		return BandWarning
	default:
		return BandNormal
	}
}
func cpuBandBoundary(band PressureBand, settings Settings) float64 {
	switch band {
	case BandCritical:
		return settings.CPUCriticalPercent
	case BandHigh:
		return settings.CPUHighPercent
	default:
		return settings.CPUThreshold
	}
}

// escalate forwards a high or critical band to the remediation service.
//
// A failed report is logged and recorded, never fatal: storage-manager being
// unreachable must not stop the monitor from continuing to observe. The
// warning band is deliberately not escalated — it exists to record pressure,
// not to act on it.
func (s *ThresholdScheduler) escalate(ctx context.Context, decision bandDecision, usage collectors.DiskUsage) {
	if s.reporter == nil || decision.Band < BandHigh {
		return
	}

	band := cleanupmanager.BandHigh
	if decision.Band == BandCritical {
		band = cleanupmanager.BandCritical
	}

	result := &RemediationResult{
		At:   s.clock.Now(),
		Band: decision.Band.String(),
	}

	outcome, err := s.reporter.ReportPressure(ctx, cleanupmanager.Report{
		SourceScenario: "system-monitor",
		Partition:      collectors.RootMountPath(),
		UsedPercent:    usage.UsedPercent,
		Band:           band,
		AvailableBytes: usage.AvailableBytes,
	})
	if err != nil {
		result.Error = err.Error()
		s.log.Error("report disk pressure to storage-manager failed", "band", decision.Band.String(), "error", err)
	} else {
		result.Action = outcome.Action
		result.PlanID = outcome.PlanID
		result.EstimatedBytes = outcome.EstimatedBytes
		result.ReclaimedBytes = outcome.ReclaimedBytes
		result.ProvidersApplied = outcome.ProvidersApplied
		result.ProvidersWithheld = outcome.ProvidersWithheld
		s.log.Info("disk pressure escalated to storage-manager",
			"band", decision.Band.String(),
			"action", outcome.Action,
			"reclaimed_bytes", outcome.ReclaimedBytes,
			"withheld", outcome.ProvidersWithheld,
		)
	}

	s.mu.Lock()
	s.lastRemedy = result
	s.mu.Unlock()
}

// evaluate classifies the sample into a band and decides whether it warrants a
// durable record. It returns nil when the band model says this observation is
// not worth recording — because usage is normal, because the band has not
// changed and the cooldown has not expired, or because a candidate escalation
// is still being debounced.
func (s *ThresholdScheduler) evaluate(settings Settings, usage collectors.DiskUsage) (*models.ThresholdViolation, bandDecision) {
	s.mu.Lock()
	previous := s.previousValue
	hasPrevious := s.hasPrevious
	decision := s.bands.evaluate(usage, previous, hasPrevious, settings, s.clock.Now())
	s.mu.Unlock()

	if !decision.Emit {
		return nil, decision
	}

	return &models.ThresholdViolation{
		MetricName:     "disk_used_percent",
		CurrentValue:   usage.UsedPercent,
		ThresholdValue: bandBoundary(decision.Band, settings),
		Severity:       decision.Band.Severity(),
		ViolationType:  decision.Band.String(),
		Timestamp:      s.clock.Now(),
		PreviousValue:  previous,
		Trend:          trendBetween(previous, usage.UsedPercent, hasPrevious),
		Duration:       fmt.Sprintf("%.0fs", s.currentInterval().Seconds()),
	}, decision
}

// bandBoundary reports the configured boundary a band was entered at, so a
// persisted violation records the threshold it actually crossed.
func bandBoundary(band PressureBand, settings Settings) float64 {
	switch band {
	case BandCritical:
		return settings.DiskCriticalPercent
	case BandHigh:
		return settings.DiskHighPercent
	default:
		return settings.DiskThreshold
	}
}

// trendBetween describes movement between two samples. The first sample has no
// predecessor, so it reports "stable" rather than inventing an increase.
func trendBetween(previous, current float64, hasPrevious bool) string {
	if !hasPrevious {
		return "stable"
	}
	switch {
	case current > previous:
		return "increasing"
	case current < previous:
		return "decreasing"
	default:
		return "stable"
	}
}

func (s *ThresholdScheduler) recordRun(usage collectors.DiskUsage, err error, violation *models.ThresholdViolation, decision bandDecision) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastBand = decision.Band
	s.hasRun = true
	s.lastRunAt = s.clock.Now()
	s.lastErr = err
	if err == nil {
		s.lastUsage = usage
		s.previousValue = usage.UsedPercent
		s.hasPrevious = true
	}
	if violation != nil {
		s.lastViolation = violation
		s.violations++
	}
}

func (s *ThresholdScheduler) signalEvaluated() {
	if s.evaluated == nil {
		return
	}
	s.evaluated <- struct{}{}
}

// Status returns the outcome of the most recent evaluation.
func (s *ThresholdScheduler) Status() ThresholdStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := ThresholdStatus{
		HasRun:          s.hasRun,
		LastRunAt:       s.lastRunAt,
		LastUsage:       s.lastUsage,
		LastViolation:   s.lastViolation,
		Violations:      s.violations,
		NextInterval:    s.currentInterval(),
		Band:            s.lastBand,
		LastTransition:  s.bands.lastObservation,
		LastRemediation: s.lastRemedy,
	}
	if s.lastErr != nil {
		st.LastError = s.lastErr.Error()
	}
	return st
}
