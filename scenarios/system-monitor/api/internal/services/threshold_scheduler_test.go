package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/cleanupmanager"
)

type capturedDomainEvent struct{ events []eventbus.DomainEvent }

func (c *capturedDomainEvent) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	c.events = append(c.events, event)
	return nil
}

func TestPublishWriterEventsUsesStructSafeTimestamp(t *testing.T) {
	capture := &capturedDomainEvent{}
	clock := NewStubClock(time.Date(2026, 9, 3, 3, 0, 0, 0, time.UTC))
	scheduler := &ThresholdScheduler{clock: clock, events: capture, hotEpisodes: make(map[string]bool)}
	scheduler.publishWriterEvents(context.Background(), []WriterSnapshot{{
		Root: "/tmp/proof", RootID: "proof", Mount: "/", Bytes: 10,
		BytesPerHour: 20, DeltaHours: 1, Hot: true,
		ObservedAt: time.Date(2026, 9, 3, 2, 59, 0, 0, time.UTC),
	}})
	if len(capture.events) != 1 || capture.events[0].EventType != "storage.writer.hot" {
		t.Fatalf("events = %#v, want one hot event", capture.events)
	}
	if _, ok := capture.events[0].Payload["sampled_at"].(string); !ok {
		t.Fatalf("sampled_at = %#v, want RFC3339 string", capture.events[0].Payload["sampled_at"])
	}
}

// fakeDiskSource is the fault-injection seam. Filling a real filesystem is not
// an option in a unit test, so pressure is simulated here.
type fakeDiskSource struct {
	usage collectors.DiskUsage
	err   error
	reads int
}

func (f *fakeDiskSource) ReadDiskUsage(context.Context) (collectors.DiskUsage, error) {
	f.reads++
	if f.err != nil {
		return collectors.DiskUsage{}, f.err
	}
	return f.usage, nil
}

func (f *fakeDiskSource) setPercent(pct float64) {
	f.usage = collectors.DiskUsage{
		TotalBytes:     1000,
		UsedBytes:      int64(pct * 10),
		AvailableBytes: int64((100 - pct) * 10),
		UsedPercent:    pct,
	}
}

// thresholdHarness wires a scheduler against in-memory storage and a stub
// clock so a whole pressure episode can be replayed deterministically.
type thresholdHarness struct {
	scheduler *ThresholdScheduler
	source    *fakeDiskSource
	repo      *memory.MemoryRepository
	clock     *StubClock
	settings  *SettingsManager
	evaluated chan struct{}
}

func newThresholdHarness(t *testing.T, interval int, threshold float64, mutate ...func(*Settings)) *thresholdHarness {
	t.Helper()

	settings := NewSettingsManager(
		WithSettingsConfigStore(NewMemoryConfigStore()),
		WithSettingsClock(NewStubClock(time.Unix(0, 0))),
	)
	next := settings.GetSettings()
	next.Active = true
	next.ThresholdCheckInterval = interval
	next.DiskThreshold = threshold
	// Most tests exercise one mechanism at a time, so debounce and cooldown
	// are off by default and switched on explicitly by the tests that own them.
	next.DiskEscalationDebounceTicks = 1
	next.DiskEscalationCooldownSeconds = 1
	for _, m := range mutate {
		m(&next)
	}
	if err := settings.UpdateSettings(next); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	repo := memory.NewRepository()
	source := &fakeDiskSource{}
	source.setPercent(10)
	clock := NewStubClock(time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC))
	evaluated := make(chan struct{}, 64)

	alerts := NewAlertService(&config.Config{}, repo, WithAlertClock(clock))
	scheduler := NewThresholdScheduler(settings, alerts, repo, nil,
		WithThresholdClock(clock),
		WithDiskUsageSource(source),
		WithEvaluationSignal(evaluated),
	)

	return &thresholdHarness{
		scheduler: scheduler,
		source:    source,
		repo:      repo,
		clock:     clock,
		settings:  settings,
		evaluated: evaluated,
	}
}

// tick advances the clock by one interval and blocks until the loop has
// finished the resulting evaluation. No real time passes.
func (h *thresholdHarness) tick(t *testing.T, interval time.Duration) {
	t.Helper()
	if !h.clock.WaitUntilArmed(2 * time.Second) {
		t.Fatal("scheduler never armed its timer")
	}
	h.clock.Advance(interval)
	select {
	case <-h.evaluated:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not evaluate after the clock advanced")
	}
}

func (h *thresholdHarness) violations(t *testing.T) []*models.ThresholdViolation {
	t.Helper()
	got, err := h.repo.GetThresholdViolations(context.Background(), repository.TimeRange{
		StartTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GetThresholdViolations: %v", err)
	}
	return got
}

// TestThresholdScheduler_FiresOnConfiguredInterval proves the loop actually
// runs. The 2026-07-31 incident happened with a fully tested alerting stack
// that no loop ever called, so "the code is correct" is not the property under
// test here — "the code is reached, on a schedule" is.
func TestThresholdScheduler_FiresOnConfiguredInterval(t *testing.T) {
	h := newThresholdHarness(t, 20, 85)
	h.scheduler.Start()
	defer h.scheduler.Stop()

	const interval = 20 * time.Second
	for i := 1; i <= 3; i++ {
		h.tick(t, interval)
		if h.source.reads != i {
			t.Fatalf("after %d ticks the source was read %d times, want %d", i, h.source.reads, i)
		}
	}

	if status := h.scheduler.Status(); !status.HasRun {
		t.Error("Status().HasRun is false after three evaluations")
	}
}

// TestThresholdScheduler_DoesNotEvaluateWhenInactive asserts a paused monitor
// stays silent.
func TestThresholdScheduler_DoesNotEvaluateWhenInactive(t *testing.T) {
	h := newThresholdHarness(t, 20, 85)

	next := h.settings.GetSettings()
	next.Active = false
	if err := h.settings.UpdateSettings(next); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	h.source.setPercent(99)

	h.scheduler.Start()
	defer h.scheduler.Stop()

	h.tick(t, 20*time.Second)

	if h.source.reads != 0 {
		t.Errorf("inactive monitor read the filesystem %d times, want 0", h.source.reads)
	}
	if got := h.violations(t); len(got) != 0 {
		t.Errorf("inactive monitor persisted %d violations, want 0", len(got))
	}
}

// TestThresholdScheduler_ZeroIntervalFallsBackToDefault asserts a
// zero/nonsensical interval cannot spin the loop or disable it silently.
func TestThresholdScheduler_ZeroIntervalFallsBackToDefault(t *testing.T) {
	h := newThresholdHarness(t, 20, 85)

	// Write the interval straight into the manager, bypassing sanitizeSettings,
	// to simulate a hand-edited settings file.
	h.settings.mutex.Lock()
	h.settings.settings.ThresholdCheckInterval = 0
	h.settings.mutex.Unlock()

	want := time.Duration(defaultSettings.ThresholdCheckInterval) * time.Second
	if got := h.scheduler.currentInterval(); got != want {
		t.Errorf("currentInterval() = %s, want the default %s", got, want)
	}
}

// TestThresholdScheduler_PersistsViolationOnCrossing is the end-to-end proof:
// simulated pressure produces a record that survives in the repository and is
// readable back.
func TestThresholdScheduler_PersistsViolationOnCrossing(t *testing.T) {
	h := newThresholdHarness(t, 20, 85)
	h.scheduler.Start()
	defer h.scheduler.Stop()

	const interval = 20 * time.Second

	// Below the threshold: nothing is recorded.
	h.source.setPercent(70)
	h.tick(t, interval)
	if got := h.violations(t); len(got) != 0 {
		t.Fatalf("usage below threshold produced %d violations, want 0", len(got))
	}

	// Cross the threshold: a violation must exist and be readable.
	h.source.setPercent(92)
	h.tick(t, interval)

	got := h.violations(t)
	if len(got) != 1 {
		t.Fatalf("threshold crossing produced %d violations, want 1", len(got))
	}
	v := got[0]
	if v.MetricName != "disk_used_percent" {
		t.Errorf("MetricName = %q, want disk_used_percent", v.MetricName)
	}
	if v.CurrentValue != 92 {
		t.Errorf("CurrentValue = %v, want 92", v.CurrentValue)
	}
	// 92 percent sits in the high band, so the recorded boundary is the high
	// boundary that was actually crossed, not the warning threshold.
	if v.ThresholdValue != 90 {
		t.Errorf("ThresholdValue = %v, want the high boundary 90", v.ThresholdValue)
	}
	if v.ViolationType != "high" {
		t.Errorf("ViolationType = %q, want high", v.ViolationType)
	}
	if v.PreviousValue != 70 {
		t.Errorf("PreviousValue = %v, want 70", v.PreviousValue)
	}
	if v.Trend != "increasing" {
		t.Errorf("Trend = %q, want increasing", v.Trend)
	}
	if v.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}

	// The alert path must have been exercised too, not just persistence.
	alerts, err := h.repo.GetActiveAlerts(context.Background())
	if err != nil {
		t.Fatalf("GetActiveAlerts: %v", err)
	}
	if len(alerts) == 0 {
		t.Fatal("threshold crossing produced no alert; the alert service is not wired")
	}
	if alerts[0].Type != "threshold_violation" {
		t.Errorf("alert Type = %q, want threshold_violation", alerts[0].Type)
	}
}

// TestThresholdScheduler_ReportsTrendDirection walks usage up and back down.
func TestThresholdScheduler_ReportsTrendDirection(t *testing.T) {
	h := newThresholdHarness(t, 20, 50)
	h.scheduler.Start()
	defer h.scheduler.Stop()

	const interval = 20 * time.Second
	for _, pct := range []float64{60, 80, 70} {
		h.source.setPercent(pct)
		h.tick(t, interval)
	}

	got := h.violations(t)
	if len(got) != 3 {
		t.Fatalf("got %d violations, want 3", len(got))
	}
	// The repository returns newest-first ordering guarantees nowhere, so index
	// by current value instead.
	byValue := map[float64]string{}
	for _, v := range got {
		byValue[v.CurrentValue] = v.Trend
	}
	if byValue[60] != "stable" {
		t.Errorf("first sample trend = %q, want stable (no predecessor)", byValue[60])
	}
	if byValue[80] != "increasing" {
		t.Errorf("rising sample trend = %q, want increasing", byValue[80])
	}
	if byValue[70] != "decreasing" {
		t.Errorf("falling sample trend = %q, want decreasing", byValue[70])
	}
}

// TestThresholdScheduler_SurvivesMeasurementFailure asserts a failing
// filesystem read is recorded rather than crashing or silently succeeding.
func TestThresholdScheduler_SurvivesMeasurementFailure(t *testing.T) {
	h := newThresholdHarness(t, 20, 85)
	h.source.err = errors.New("statfs: permission denied")

	h.scheduler.Start()
	defer h.scheduler.Stop()
	h.tick(t, 20*time.Second)

	status := h.scheduler.Status()
	if status.LastError == "" {
		t.Error("a failed measurement left LastError empty")
	}
	if got := h.violations(t); len(got) != 0 {
		t.Errorf("a failed measurement produced %d violations, want 0", len(got))
	}
}

// TestThresholdScheduler_StopHaltsTheLoop asserts shutdown works and does not
// hang, so the server can exit cleanly.
func TestThresholdScheduler_StopHaltsTheLoop(t *testing.T) {
	h := newThresholdHarness(t, 20, 85)
	h.scheduler.Start()
	h.tick(t, 20*time.Second)

	done := make(chan struct{})
	go func() {
		h.scheduler.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return")
	}
}

// fakeReporter records escalations instead of reaching a live storage-manager.
type fakeReporter struct {
	mu      sync.Mutex
	reports []cleanupmanager.Report
	outcome cleanupmanager.Outcome
	err     error
}

func (f *fakeReporter) ReportPressure(_ context.Context, report cleanupmanager.Report) (cleanupmanager.Outcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reports = append(f.reports, report)
	if f.err != nil {
		return cleanupmanager.Outcome{}, f.err
	}
	return f.outcome, nil
}

func (f *fakeReporter) snapshot() []cleanupmanager.Report {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]cleanupmanager.Report(nil), f.reports...)
}

// TestThresholdScheduler_ForwardsWarningHighAndCritical asserts every emitted
// pressure band reaches storage-manager, which owns the safe action policy.
func TestThresholdScheduler_ForwardsWarningHighAndCritical(t *testing.T) {
	reporter := &fakeReporter{outcome: cleanupmanager.Outcome{Action: "applied", ReclaimedBytes: 4096}}

	h := newThresholdHarness(t, 20, 80)
	h.scheduler.reporter = reporter
	h.scheduler.Start()
	defer h.scheduler.Stop()

	const interval = 20 * time.Second
	// normal, warning, high, critical
	for _, pct := range []float64{70, 82, 91, 96} {
		h.source.setPercent(pct)
		h.tick(t, interval)
	}

	reports := reporter.snapshot()
	if len(reports) != 3 {
		t.Fatalf("forwarded %d times, want warning, high, and critical: %+v", len(reports), reports)
	}
	if reports[0].Band != cleanupmanager.BandWarning {
		t.Errorf("first escalation band = %s, want warning", reports[0].Band)
	}
	if reports[1].Band != cleanupmanager.BandHigh {
		t.Errorf("second escalation band = %s, want high", reports[1].Band)
	}
	if reports[2].Band != cleanupmanager.BandCritical {
		t.Errorf("third escalation band = %s, want critical", reports[2].Band)
	}
	if reports[2].UsedPercent != 96 {
		t.Errorf("escalation reported %v percent, want the observed 96", reports[2].UsedPercent)
	}
	if reports[1].SourceScenario != "system-monitor" {
		t.Errorf("escalation source = %q, want system-monitor so the audit can attribute it", reports[1].SourceScenario)
	}

	// The remediation outcome must be readable from status.
	status := h.scheduler.Status()
	if status.LastRemediation == nil {
		t.Fatal("Status() reports no remediation after two escalations")
	}
	if status.LastRemediation.ReclaimedBytes != 4096 {
		t.Errorf("recorded reclaimed = %d, want 4096", status.LastRemediation.ReclaimedBytes)
	}
}

// TestThresholdScheduler_SurvivesUnreachableCleanupManager asserts that a
// remediation peer being down does not stop the monitor observing.
//
// This is the failure mode the incident's own design notes warn about: two
// safeguards that both depend on one mediator share a point of failure. The
// monitor must keep recording violations even when it cannot escalate them.
func TestThresholdScheduler_SurvivesUnreachableCleanupManager(t *testing.T) {
	reporter := &fakeReporter{err: errors.New("storage-manager unreachable: connection refused")}

	h := newThresholdHarness(t, 20, 80)
	h.scheduler.reporter = reporter
	h.scheduler.Start()
	defer h.scheduler.Stop()

	h.source.setPercent(96)
	h.tick(t, 20*time.Second)

	// The violation is still recorded even though escalation failed.
	if got := h.violations(t); len(got) != 1 {
		t.Errorf("persisted %d violations while storage-manager was down, want 1", len(got))
	}

	status := h.scheduler.Status()
	if status.LastRemediation == nil || status.LastRemediation.Error == "" {
		t.Error("the failed escalation was not recorded, so an operator cannot tell remediation is broken")
	}

	// And the loop keeps running.
	h.source.setPercent(70)
	h.tick(t, 20*time.Second)
	if !h.scheduler.Status().HasRun {
		t.Error("the loop stopped after a failed escalation")
	}
}

func TestThresholdSchedulerReportsPositiveFillRateFromSamples(t *testing.T) {
	scheduler := &ThresholdScheduler{}
	firstAt := time.Unix(100, 0)
	first := scheduler.withFillRate(collectors.DiskUsage{UsedBytes: 10_000}, firstAt)
	if first.FillRateBytesPerHour != 0 {
		t.Fatalf("first sample rate = %d, want 0 without a predecessor", first.FillRateBytesPerHour)
	}
	second := scheduler.withFillRate(collectors.DiskUsage{UsedBytes: 11_000}, firstAt.Add(10*time.Minute))
	if second.FillRateBytesPerHour != 6_000 {
		t.Fatalf("second sample rate = %d, want 6000 bytes/hour", second.FillRateBytesPerHour)
	}
	decrease := scheduler.withFillRate(collectors.DiskUsage{UsedBytes: 9_000}, firstAt.Add(20*time.Minute))
	if decrease.FillRateBytesPerHour != 0 {
		t.Fatalf("decreasing sample rate = %d, want 0", decrease.FillRateBytesPerHour)
	}
}
