package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	catalogpkg "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/investigations"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

// fakeSnapshotSource is the evaluator's metric boundary.
type fakeSnapshotSource struct {
	mu       sync.Mutex
	metrics  *models.MetricsResponse
	pressure *models.MemoryPressure
}

func (f *fakeSnapshotSource) GetCurrentMetricsFresh(context.Context) (*models.MetricsResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.metrics == nil {
		return nil, nil
	}
	copied := *f.metrics
	return &copied, nil
}

func (f *fakeSnapshotSource) GetMemoryPressure(context.Context) (*models.MemoryPressure, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pressure, nil
}

func (f *fakeSnapshotSource) setCPU(percent float64, at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.metrics = &models.MetricsResponse{Timestamp: at, CPUUsage: percent, CPUState: models.MetricState{Status: "measured", Value: percent}}
}

func (f *fakeSnapshotSource) setPressure(level string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pressure = &models.MemoryPressure{Level: level, LevelSource: "test", LevelReason: "fixture says " + level, Platform: "linux"}
}

type fakeEscalator struct {
	mu    sync.Mutex
	calls []*models.Alert
}

func (f *fakeEscalator) Escalate(_ context.Context, alert *models.Alert) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, alert)
	return "anomaly-test-1", nil
}

func (f *fakeEscalator) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

type evaluatorHarness struct {
	eval      *AlertEvaluator
	source    *fakeSnapshotSource
	repo      *memory.MemoryRepository
	clock     *StubClock
	escalator *fakeEscalator
	settings  *SettingsManager
}

func newEvaluatorHarness(t *testing.T, rules ...AlertRule) *evaluatorHarness {
	t.Helper()
	settings := NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore()), WithSettingsClock(NewStubClock(time.Unix(0, 0))))
	next := settings.GetSettings()
	next.Active = true
	next.AlertSustainSeconds = 300
	if err := settings.UpdateSettings(next); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	clock := NewStubClock(time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))
	repo := memory.NewRepository()
	source := &fakeSnapshotSource{}
	escalator := &fakeEscalator{}
	alerts := NewAlertService(&config.Config{}, repo, WithAlertClock(clock))
	opts := []AlertEvaluatorOption{WithAlertEvaluatorClock(clock), WithAlertEscalator(escalator)}
	if len(rules) > 0 {
		opts = append(opts, WithAlertRules(rules...))
	}
	return &evaluatorHarness{
		eval:      NewAlertEvaluator(settings, source, repo, alerts, nil, opts...),
		source:    source,
		repo:      repo,
		clock:     clock,
		escalator: escalator,
		settings:  settings,
	}
}

// pass advances the clock and runs one evaluation, keeping the metric
// snapshot fresh so the snapshot rule stays clear.
func (h *evaluatorHarness) pass(t *testing.T, advance time.Duration) {
	t.Helper()
	h.clock.Advance(advance)
	h.source.mu.Lock()
	if h.source.metrics != nil {
		h.source.metrics.Timestamp = h.clock.Now()
	}
	h.source.mu.Unlock()
	h.eval.RunOnce(context.Background())
}

func (h *evaluatorHarness) open(t *testing.T, ruleID string) []*models.Alert {
	t.Helper()
	all, err := h.repo.ListAlerts(context.Background(), repository.AlertFilter{Type: ConditionAlertType, Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	var out []*models.Alert
	for _, alert := range all {
		if alert.Details["rule_id"] == ruleID {
			out = append(out, alert)
		}
	}
	return out
}

func TestAlertEvaluator_OpensOnlyAfterConditionIsSustained(t *testing.T) {
	h := newEvaluatorHarness(t, cpuUsageRule{})
	h.source.setCPU(90, h.clock.Now()) // >= 85 threshold, < 95: warning

	h.pass(t, 0)
	if got := h.open(t, RuleCPUUsage); len(got) != 0 {
		t.Fatalf("alert opened on first observation: %+v", got[0])
	}
	status := h.eval.RuleStatuses()[0]
	if !status.ConditionActive || !status.PendingSince.Equal(h.clock.Now()) {
		t.Fatalf("rule status = %+v, want an active pending condition", status)
	}

	h.pass(t, 4*time.Minute+59*time.Second)
	if got := h.open(t, RuleCPUUsage); len(got) != 0 {
		t.Fatal("alert opened before the sustain duration elapsed")
	}

	h.pass(t, time.Second)
	opened := h.open(t, RuleCPUUsage)
	if len(opened) != 1 {
		t.Fatalf("open alerts = %d after 5 minutes, want 1", len(opened))
	}
	if opened[0].Severity != SeverityWarning || opened[0].Type != ConditionAlertType || opened[0].MetricValue != 90 {
		t.Fatalf("opened alert = %+v", opened[0])
	}

	for i := 0; i < 5; i++ {
		h.pass(t, 20*time.Second)
	}
	if got := h.open(t, RuleCPUUsage); len(got) != 1 {
		t.Fatalf("open alerts = %d after repeated passes, want exactly 1 (dedup)", len(got))
	}
}

func TestAlertEvaluator_BriefSpikeNeverAlerts(t *testing.T) {
	h := newEvaluatorHarness(t, cpuUsageRule{})
	h.source.setCPU(99, h.clock.Now())
	h.pass(t, 0)
	h.pass(t, 2*time.Minute)
	h.source.setCPU(20, h.clock.Now())
	h.pass(t, 20*time.Second)
	h.source.setCPU(99, h.clock.Now())
	h.pass(t, 20*time.Second)
	h.pass(t, 4*time.Minute)
	if got := h.open(t, RuleCPUUsage); len(got) != 0 {
		t.Fatal("an interrupted condition opened an alert; the sustain window must restart")
	}
}

func TestAlertEvaluator_CriticalNeedsItsOwnSustainAndEscalatesOnce(t *testing.T) {
	h := newEvaluatorHarness(t, memoryPressureRule{})
	h.source.setPressure(collectors.MemoryPressureElevated)
	h.pass(t, 0)
	h.pass(t, 5*time.Minute)
	opened := h.open(t, RuleMemoryPressure)
	if len(opened) != 1 || opened[0].Severity != SeverityWarning {
		t.Fatalf("open = %+v, want one warning", opened)
	}

	h.source.setPressure(collectors.MemoryPressureCritical)
	h.pass(t, 20*time.Second)
	h.pass(t, 4*time.Minute)
	if got := h.open(t, RuleMemoryPressure); got[0].Severity != SeverityWarning {
		t.Fatalf("severity = %s before critical was sustained, want warning", got[0].Severity)
	}
	if h.escalator.count() != 0 {
		t.Fatal("escalated before critical was sustained")
	}

	// Critical was first observed at the 20 s pass; 4 m + 1 m later it has
	// held for exactly the sustain duration.
	h.pass(t, time.Minute)
	got := h.open(t, RuleMemoryPressure)
	if len(got) != 1 || got[0].Severity != SeverityCritical {
		t.Fatalf("open = %+v, want the same alert upgraded to critical", got)
	}
	if got[0].ID != opened[0].ID {
		t.Fatalf("critical opened a second alert %s instead of upgrading %s", got[0].ID, opened[0].ID)
	}
	if h.escalator.count() != 1 {
		t.Fatalf("escalations = %d, want 1", h.escalator.count())
	}
	if got[0].Details["escalated"] != true || got[0].Details["anomaly_id"] != "anomaly-test-1" {
		t.Fatalf("escalation not recorded on the alert: %+v", got[0].Details)
	}

	for i := 0; i < 10; i++ {
		h.pass(t, time.Minute)
	}
	if h.escalator.count() != 1 {
		t.Fatalf("escalations = %d after the condition persisted, want exactly 1", h.escalator.count())
	}
}

func TestAlertEvaluator_ResolvesOnlyAfterClearIsSustained(t *testing.T) {
	h := newEvaluatorHarness(t, cpuUsageRule{})
	h.source.setCPU(90, h.clock.Now())
	h.pass(t, 0)
	h.pass(t, 5*time.Minute)
	opened := h.open(t, RuleCPUUsage)
	if len(opened) != 1 {
		t.Fatal("setup: alert did not open")
	}

	h.source.setCPU(10, h.clock.Now())
	h.pass(t, 20*time.Second)
	h.pass(t, 3*time.Minute)
	// A relapse inside the clear window restarts it.
	h.source.setCPU(90, h.clock.Now())
	h.pass(t, 20*time.Second)
	h.source.setCPU(10, h.clock.Now())
	h.pass(t, 20*time.Second)
	h.pass(t, 4*time.Minute+30*time.Second)
	if got := h.open(t, RuleCPUUsage); len(got) != 1 {
		t.Fatal("alert resolved before the condition stayed clear for the sustain duration")
	}

	h.pass(t, 30*time.Second)
	if got := h.open(t, RuleCPUUsage); len(got) != 0 {
		t.Fatalf("alert still open after 5 minutes clear: %+v", got[0])
	}
	resolved, err := h.repo.GetAlert(context.Background(), opened[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ResolvedAt == nil || !resolved.ResolvedAt.Equal(h.clock.Now()) {
		t.Fatalf("ResolvedAt = %v, want the evaluator clock time %v", resolved.ResolvedAt, h.clock.Now())
	}
}

func TestAlertEvaluator_UnsupportedRuleNeitherOpensNorResolves(t *testing.T) {
	h := newEvaluatorHarness(t, memoryPressureRule{})
	h.source.setPressure(collectors.MemoryPressureElevated)
	h.pass(t, 0)
	h.pass(t, 5*time.Minute)
	if len(h.open(t, RuleMemoryPressure)) != 1 {
		t.Fatal("setup: alert did not open")
	}

	h.source.setPressure(collectors.MemoryPressureUnknown)
	for i := 0; i < 20; i++ {
		h.pass(t, time.Minute)
	}
	if got := h.open(t, RuleMemoryPressure); len(got) != 1 {
		t.Fatal("an unknown level resolved the alert; absence of evidence is not a clear condition")
	}
	status := h.eval.RuleStatuses()[0]
	if status.Status != RuleUnsupported || !strings.Contains(status.Reason, "unknown") {
		t.Fatalf("rule status = %+v, want unsupported naming the unknown level", status)
	}
}

func TestAlertEvaluator_AdoptsOpenAlertsAcrossRestart(t *testing.T) {
	h := newEvaluatorHarness(t, cpuUsageRule{})
	h.source.setCPU(90, h.clock.Now())
	h.pass(t, 0)
	h.pass(t, 5*time.Minute)
	if len(h.open(t, RuleCPUUsage)) != 1 {
		t.Fatal("setup: alert did not open")
	}

	restarted := NewAlertEvaluator(h.settings, h.source, h.repo, NewAlertService(&config.Config{}, h.repo, WithAlertClock(h.clock)), nil,
		WithAlertEvaluatorClock(h.clock), WithAlertRules(cpuUsageRule{}))
	h.eval = restarted
	h.pass(t, 20*time.Second)
	h.pass(t, 6*time.Minute)
	if got := h.open(t, RuleCPUUsage); len(got) != 1 {
		t.Fatalf("open alerts after restart = %d, want the adopted one only", len(got))
	}

	h.source.setCPU(10, h.clock.Now())
	h.pass(t, 20*time.Second)
	h.pass(t, 5*time.Minute)
	if got := h.open(t, RuleCPUUsage); len(got) != 0 {
		t.Fatal("restarted evaluator did not resolve the adopted alert")
	}
}

func TestAlertEvaluator_InactiveMonitoringEvaluatesNothing(t *testing.T) {
	h := newEvaluatorHarness(t, cpuUsageRule{})
	if err := h.settings.SetActive(false); err != nil {
		t.Fatal(err)
	}
	h.source.setCPU(99, h.clock.Now())
	h.pass(t, 0)
	h.pass(t, 10*time.Minute)
	if got := h.open(t, RuleCPUUsage); len(got) != 0 {
		t.Fatal("a paused monitor opened an alert")
	}
	if !h.eval.EvaluatedAt().IsZero() {
		t.Fatal("a paused monitor recorded an evaluation pass")
	}
}

func TestAlertEvaluator_ListAlertsSeparatesEventsFromConditions(t *testing.T) {
	h := newEvaluatorHarness(t, cpuUsageRule{})
	alerts := NewAlertService(&config.Config{}, h.repo, WithAlertClock(h.clock))
	if err := alerts.SendThresholdViolation(context.Background(), &models.ThresholdViolation{MetricName: "disk_used_percent", CurrentValue: 91, Severity: "high", Timestamp: h.clock.Now()}); err != nil {
		t.Fatal(err)
	}
	h.source.setCPU(90, h.clock.Now())
	h.pass(t, 0)
	h.pass(t, 5*time.Minute)

	active, err := h.eval.ListAlerts(context.Background(), AlertQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].Type != ConditionAlertType {
		t.Fatalf("default listing = %d alerts, want only the open condition", len(active))
	}
	all, err := h.eval.ListAlerts(context.Background(), AlertQuery{Status: "all", IncludeEvents: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("listing with events = %d, want 2", len(all))
	}
	if _, err := h.eval.ListAlerts(context.Background(), AlertQuery{Status: "bogus"}); err == nil {
		t.Fatal("unknown status was accepted")
	}
}

func TestPortedHostRules(t *testing.T) {
	settings := defaultSettings
	gpu := func(v float64) *float64 { return &v }
	measured := func(v float64) models.MetricState { return models.MetricState{Status: "measured", Value: v} }
	cases := []struct {
		name     string
		rule     AlertRule
		metrics  *models.MetricsResponse
		status   string
		severity string // "" = clear
	}{
		{"cpu below threshold", cpuUsageRule{}, &models.MetricsResponse{CPUState: measured(84.9)}, RuleSupported, ""},
		{"cpu at threshold", cpuUsageRule{}, &models.MetricsResponse{CPUState: measured(85)}, RuleSupported, SeverityWarning},
		{"cpu at threshold+10", cpuUsageRule{}, &models.MetricsResponse{CPUState: measured(95)}, RuleSupported, SeverityCritical},
		{"cpu legacy scalar", cpuUsageRule{}, &models.MetricsResponse{CPUUsage: 96}, RuleSupported, SeverityCritical},
		{"cpu unsupported", cpuUsageRule{}, &models.MetricsResponse{CPUState: models.MetricState{Status: "unsupported", Reason: "no backend"}}, RuleUnsupported, ""},
		{"cpu stale", cpuUsageRule{}, &models.MetricsResponse{CPUState: models.MetricState{Status: "stale", Reason: "old"}}, RuleFailed, ""},
		{"memory at threshold", memoryUsageRule{}, &models.MetricsResponse{MemoryState: measured(90)}, RuleSupported, SeverityWarning},
		{"memory at threshold+5", memoryUsageRule{}, &models.MetricsResponse{MemoryState: measured(95)}, RuleSupported, SeverityCritical},
		{"memory healthy while thrashing", memoryUsageRule{}, &models.MetricsResponse{MemoryState: measured(53)}, RuleSupported, ""},
		{"tcp normal busy host", &tcpConnectionsRule{}, &models.MetricsResponse{ConnectionsState: measured(650)}, RuleSupported, ""},
		{"tcp at warning level", &tcpConnectionsRule{}, &models.MetricsResponse{ConnectionsState: measured(5000)}, RuleSupported, SeverityWarning},
		{"tcp at critical level", &tcpConnectionsRule{}, &models.MetricsResponse{ConnectionsState: measured(20000)}, RuleSupported, SeverityCritical},
		{"tcp leak incident level", &tcpConnectionsRule{}, &models.MetricsResponse{ConnectionsState: measured(57706)}, RuleSupported, SeverityCritical},
		{"tcp unread", &tcpConnectionsRule{}, &models.MetricsResponse{ConnectionsState: models.MetricState{Status: "unsupported", Reason: "no backend"}}, RuleUnsupported, ""},
		{"gpu at 90", gpuUsageRule{}, &models.MetricsResponse{GPUState: measured(90)}, RuleSupported, SeverityWarning},
		{"gpu legacy scalar", gpuUsageRule{}, &models.MetricsResponse{GPUUsage: gpu(91)}, RuleSupported, SeverityWarning},
		{"gpu absent", gpuUsageRule{}, &models.MetricsResponse{}, RuleUnsupported, ""},
		{"snapshot missing", metricsSnapshotRule{}, nil, RuleSupported, SeverityCritical},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval := tc.rule.Evaluate(context.Background(), AlertSnapshot{Now: time.Unix(1000, 0), Settings: settings, Metrics: tc.metrics})
			if eval.Status != tc.status {
				t.Fatalf("status = %s (%s), want %s", eval.Status, eval.Reason, tc.status)
			}
			switch {
			case tc.severity == "" && len(eval.Conditions) != 0:
				t.Fatalf("conditions = %+v, want clear", eval.Conditions)
			case tc.severity != "" && (len(eval.Conditions) != 1 || eval.Conditions[0].Severity != tc.severity):
				t.Fatalf("conditions = %+v, want one %s", eval.Conditions, tc.severity)
			}
		})
	}
}

func TestMetricsSnapshotRule_FrozenSnapshotIsCritical(t *testing.T) {
	now := time.Unix(10_000, 0)
	settings := defaultSettings // 20 s collection interval
	fresh := metricsSnapshotRule{}.Evaluate(context.Background(), AlertSnapshot{Now: now, Settings: settings, Metrics: &models.MetricsResponse{Timestamp: now.Add(-199 * time.Second)}})
	if len(fresh.Conditions) != 0 {
		t.Fatal("a snapshot younger than ten intervals raised an alert")
	}
	frozen := metricsSnapshotRule{}.Evaluate(context.Background(), AlertSnapshot{Now: now, Settings: settings, Metrics: &models.MetricsResponse{Timestamp: now.Add(-201 * time.Second)}})
	if len(frozen.Conditions) != 1 || frozen.Conditions[0].Severity != SeverityCritical {
		t.Fatalf("frozen snapshot = %+v, want critical", frozen)
	}
}

func TestMemoryPressureRule_CarriesContributorsAndSignals(t *testing.T) {
	reading := &models.MemoryPressure{
		Level: collectors.MemoryPressureCritical, LevelSource: "linux psi + swap-in rate", LevelReason: "memory PSI full avg60 19.77% >= 5.00%", Platform: "linux",
		Signals: map[string]models.MetricState{
			collectors.SignalPSIFullAvg60:       {Status: "measured", Value: 19.77},
			collectors.SignalPageReadsPerSecond: {Status: "unsupported", Reason: "Windows counter"},
		},
		Contributors:       []models.MemoryPressureContributor{{PID: 4242, Name: "node", Owner: "agent-manager", MajorFaultsPerSecond: 3100, FaultStatus: "measured", SwapKB: 6_000_000, SwapStatus: "measured"}},
		ContributorsStatus: "measured",
	}
	eval := memoryPressureRule{}.Evaluate(context.Background(), AlertSnapshot{MemoryPressure: reading})
	if len(eval.Conditions) != 1 || eval.Conditions[0].Severity != SeverityCritical {
		t.Fatalf("evaluation = %+v, want one critical condition", eval)
	}
	condition := eval.Conditions[0]
	if !strings.Contains(condition.Message, "node (pid 4242, owner agent-manager)") {
		t.Fatalf("message %q does not name the top contributor", condition.Message)
	}
	signals := condition.Details["signals"].(map[string]interface{})
	if signals[collectors.SignalPSIFullAvg60] != 19.77 {
		t.Fatalf("signals = %v, want the measured PSI value", signals)
	}
	if _, leaked := signals[collectors.SignalPageReadsPerSecond]; leaked {
		t.Fatal("an unsupported signal was copied into alert evidence as a value")
	}
	if contributors := condition.Details["contributors"].([]interface{}); len(contributors) != 1 {
		t.Fatalf("contributors = %v", contributors)
	}
}

type countingRunner struct {
	calls  int
	stdout string
}

func (r *countingRunner) Run(_ context.Context, name string, args []string, _ string) (string, string, int, error) {
	r.calls++
	if name != "vrooli" || strings.Join(args, " ") != "agent list --json" {
		return "", "unexpected command", 2, errors.New("exit 2")
	}
	return r.stdout, "", 0, nil
}

func TestCLIAgentInventory_CachesBetweenRefreshes(t *testing.T) {
	runner := &countingRunner{stdout: string(capturedAgentList(t))}
	clock := NewStubClock(time.Unix(0, 0))
	inventory := NewCLIAgentInventory(runner, clock)
	for i := 0; i < 3; i++ {
		got, err := inventory.ListAgentSessions(context.Background())
		if err != nil || len(got.Sessions) != 42 {
			t.Fatalf("ListAgentSessions = %d sessions, %v", len(got.Sessions), err)
		}
		clock.Advance(time.Minute)
	}
	if runner.calls != 1 {
		t.Fatalf("control plane read %d times within 3 minutes, want 1", runner.calls)
	}
	clock.Advance(3 * time.Minute)
	if _, err := inventory.ListAgentSessions(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 2 {
		t.Fatalf("control plane read %d times after the refresh interval, want 2", runner.calls)
	}
}

type recordingRuns struct {
	entryID, anomalyID string
	findings           []catalogpkg.Finding
}

func (r *recordingRuns) ExecuteForAnomaly(_ context.Context, entryID, anomalyID string, findings []catalogpkg.Finding) (ScriptExecution, error) {
	r.entryID, r.anomalyID, r.findings = entryID, anomalyID, findings
	return ScriptExecution{Status: "completed"}, nil
}

func TestAnomalyEscalator_RecordsAnomalyAndTaggedInvestigationRun(t *testing.T) {
	repo := memory.NewRepository()
	clock := NewStubClock(time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))
	runs := &recordingRuns{}
	escalator := NewAnomalyEscalator(NewAlertService(&config.Config{}, repo, WithAlertClock(clock)), runs, clock)
	alert := &models.Alert{ID: "cond_memory_pressure_host_1", Message: "Memory pressure is critical", MetricName: "memory_pressure_level", Details: map[string]interface{}{"rule_id": RuleMemoryPressure, "level": "critical"}}

	anomalyID, err := escalator.Escalate(context.Background(), alert)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(anomalyID, "anomaly_memory_pressure_") {
		t.Fatalf("anomaly id = %q", anomalyID)
	}
	if runs.entryID != "memory-pressure" || runs.anomalyID != anomalyID {
		t.Fatalf("investigation run = %s/%s, want memory-pressure tagged with %s", runs.entryID, runs.anomalyID, anomalyID)
	}
	if len(runs.findings) != 1 || runs.findings[0].Code != "sustained_memory_pressure" || runs.findings[0].Severity != SeverityCritical {
		t.Fatalf("findings = %+v", runs.findings)
	}
	events, err := repo.ListAlerts(context.Background(), repository.AlertFilter{Type: "anomaly"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ResolvedAt == nil || events[0].Details["alert_id"] != alert.ID {
		t.Fatalf("anomaly events = %+v, want one resolved event linked to the alert", events)
	}
}
