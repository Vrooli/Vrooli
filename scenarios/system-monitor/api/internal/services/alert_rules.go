package services

// DOC: docs/concepts/ARCHITECTURE.md#alerting

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// Alert-condition severities. Condition alerts use two levels; legacy
// threshold-violation events keep their low/medium/high/critical scale.
const (
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Rule evaluation states. An unsupported or failed rule neither opens nor
// resolves alerts: absence of evidence is not a clear condition.
const (
	RuleSupported   = "supported"
	RuleUnsupported = "unsupported"
	RuleFailed      = "failed"
)

// Built-in rule identifiers.
const (
	RuleMetricsSnapshot = "metrics_snapshot"
	RuleCPUUsage        = "cpu_usage"
	RuleMemoryUsage     = "memory_usage"
	RuleTCPConnections  = "tcp_connections"
	RuleGPUUsage        = "gpu_usage"
	RuleMemoryPressure  = "memory_pressure"
	RuleOrphanedAgents  = "orphaned_agents"
)

// hostKey is the dedup key for host-wide rules.
const hostKey = "host"

// GPUUsageWarning is the fixed GPU boundary the rule carried when it lived in
// the CLI.
const GPUUsageWarning = 90.0

// Connection-count boundaries. A fixed low level (the CLI's former 300)
// fires permanently on hosts where hundreds of connections are normal, which
// trains readers to ignore alerts; these are the dashboard's baseline-aware
// boundaries, now owned by the server.
const (
	// TCPConnectionsCritical is unambiguously wrong for a single host.
	TCPConnectionsCritical = 20_000
	// TCPConnectionsWarning is plausible under real load but worth a look.
	TCPConnectionsWarning = 5_000
	// tcpGrowthFloor: below it, doubling (20 -> 40) is ordinary activity.
	tcpGrowthFloor = 500
	// tcpGrowthMinSamples before a trend claim is meaningful.
	tcpGrowthMinSamples = 5
	// tcpGrowthFactor: multiple of the window's opening value that counts as
	// sustained growth.
	tcpGrowthFactor = 2
	// tcpGrowthWindow is how much evaluator history the growth rule reads.
	tcpGrowthWindow = 10 * time.Minute
)

// AlertSnapshot is everything one evaluation pass reads. It is gathered once
// per pass so every rule sees the same instant.
type AlertSnapshot struct {
	Now            time.Time
	Settings       Settings
	Metrics        *models.MetricsResponse
	MetricsErr     error
	MemoryPressure *models.MemoryPressure
	PressureErr    error
}

// AlertCondition is one rule key whose condition currently holds.
type AlertCondition struct {
	Key        string
	Severity   string
	Message    string
	MetricName string
	Value      float64
	Threshold  float64
	Details    map[string]interface{}
}

// RuleEvaluation is a rule's verdict for one pass. Conditions lists only the
// keys whose condition currently holds; a supported rule with no conditions
// is a clear observation.
type RuleEvaluation struct {
	Status     string
	Reason     string
	Conditions []AlertCondition
}

// AlertRule evaluates one condition family against a snapshot.
type AlertRule interface {
	ID() string
	Evaluate(ctx context.Context, snapshot AlertSnapshot) RuleEvaluation
}

func unsupportedRule(reason string) RuleEvaluation {
	return RuleEvaluation{Status: RuleUnsupported, Reason: reason}
}

func failedRule(reason string) RuleEvaluation {
	return RuleEvaluation{Status: RuleFailed, Reason: reason}
}

func clearRule() RuleEvaluation { return RuleEvaluation{Status: RuleSupported} }

func activeRule(conditions ...AlertCondition) RuleEvaluation {
	return RuleEvaluation{Status: RuleSupported, Conditions: conditions}
}

// DefaultAlertRules returns the built-in host rules. The CPU, memory and GPU
// boundaries are the ones the CLI previously applied client-side; the TCP rule
// is the dashboard's former level-plus-growth rule.
func DefaultAlertRules() []AlertRule {
	return []AlertRule{
		metricsSnapshotRule{},
		cpuUsageRule{},
		memoryUsageRule{},
		&tcpConnectionsRule{},
		gpuUsageRule{},
		memoryPressureRule{},
	}
}

// measuredMetric returns a primary metric's value only when it was measured.
// A state with no status predates typed states and carries the legacy scalar.
func ruleMetricValue(state models.MetricState, legacy float64) (float64, RuleEvaluation, bool) {
	switch state.Status {
	case "", "measured":
		if state.Status == "" {
			return legacy, RuleEvaluation{}, true
		}
		return state.Value, RuleEvaluation{}, true
	case "unsupported":
		return 0, unsupportedRule(state.Reason), false
	default:
		reason := state.Reason
		if reason == "" {
			reason = "metric state is " + state.Status
		}
		return 0, failedRule(reason), false
	}
}

// metricsSnapshotRule raises a critical alert when the monitor has no usable
// snapshot: none at all, a read error, or one frozen for longer than ten
// collection intervals (the monitor has stopped collecting).
type metricsSnapshotRule struct{}

func (metricsSnapshotRule) ID() string { return RuleMetricsSnapshot }

func (metricsSnapshotRule) Evaluate(_ context.Context, snap AlertSnapshot) RuleEvaluation {
	condition := AlertCondition{Key: hostKey, Severity: SeverityCritical, MetricName: RuleMetricsSnapshot}
	switch {
	case snap.MetricsErr != nil:
		condition.Message = "No metrics snapshot is available: " + snap.MetricsErr.Error()
		return activeRule(condition)
	case snap.Metrics == nil:
		condition.Message = "No metrics snapshot is available."
		return activeRule(condition)
	}
	interval := time.Duration(snap.Settings.MetricCollectionInterval) * time.Second
	if interval <= 0 {
		interval = time.Duration(defaultSettings.MetricCollectionInterval) * time.Second
	}
	if !snap.Metrics.Timestamp.IsZero() && snap.Now.Sub(snap.Metrics.Timestamp) > 10*interval {
		age := snap.Now.Sub(snap.Metrics.Timestamp).Round(time.Second)
		condition.Message = fmt.Sprintf("The latest metrics snapshot is %s old; collection has stopped.", age)
		condition.Value = age.Seconds()
		return activeRule(condition)
	}
	return clearRule()
}

type cpuUsageRule struct{}

func (cpuUsageRule) ID() string { return RuleCPUUsage }

func (cpuUsageRule) Evaluate(_ context.Context, snap AlertSnapshot) RuleEvaluation {
	if snap.Metrics == nil {
		return failedRule("no metrics snapshot")
	}
	value, refused, ok := ruleMetricValue(snap.Metrics.CPUState, snap.Metrics.CPUUsage)
	if !ok {
		return refused
	}
	threshold := snap.Settings.CPUThreshold
	if threshold <= 0 {
		threshold = defaultSettings.CPUThreshold
	}
	message := fmt.Sprintf("CPU usage is %.1f%% against a %.1f%% threshold.", value, threshold)
	switch {
	case value >= threshold+10:
		return activeRule(AlertCondition{Key: hostKey, Severity: SeverityCritical, Message: message, MetricName: "cpu_usage_percent", Value: value, Threshold: threshold + 10})
	case value >= threshold:
		return activeRule(AlertCondition{Key: hostKey, Severity: SeverityWarning, Message: message, MetricName: "cpu_usage_percent", Value: value, Threshold: threshold})
	}
	return clearRule()
}

type memoryUsageRule struct{}

func (memoryUsageRule) ID() string { return RuleMemoryUsage }

func (memoryUsageRule) Evaluate(_ context.Context, snap AlertSnapshot) RuleEvaluation {
	if snap.Metrics == nil {
		return failedRule("no metrics snapshot")
	}
	value, refused, ok := ruleMetricValue(snap.Metrics.MemoryState, snap.Metrics.MemoryUsage)
	if !ok {
		return refused
	}
	threshold := snap.Settings.MemoryThreshold
	if threshold <= 0 {
		threshold = defaultSettings.MemoryThreshold
	}
	message := fmt.Sprintf("Memory usage is %.1f%% against a %.1f%% threshold.", value, threshold)
	switch {
	case value >= threshold+5:
		return activeRule(AlertCondition{Key: hostKey, Severity: SeverityCritical, Message: message, MetricName: "memory_usage_percent", Value: value, Threshold: threshold + 5})
	case value >= threshold:
		return activeRule(AlertCondition{Key: hostKey, Severity: SeverityWarning, Message: message, MetricName: "memory_usage_percent", Value: value, Threshold: threshold})
	}
	return clearRule()
}

// tcpConnectionsRule flags a high connection level and a leak-shaped climb.
// A leak is a shape, not a level: it climbs monotonically from a
// normal-looking baseline (2026-08-21: 57,706 connections, 93% held by one
// leaking process), so sustained growth warns before the level does.
type tcpConnectionsRule struct {
	mu      sync.Mutex
	samples []tcpSample
}

type tcpSample struct {
	at    time.Time
	value float64
}

func (*tcpConnectionsRule) ID() string { return RuleTCPConnections }

func (r *tcpConnectionsRule) Evaluate(_ context.Context, snap AlertSnapshot) RuleEvaluation {
	if snap.Metrics == nil {
		return failedRule("no metrics snapshot")
	}
	value, refused, ok := ruleMetricValue(snap.Metrics.ConnectionsState, float64(snap.Metrics.TCPConnections))
	if !ok {
		// An unread count is not a sample; the growth window keeps only
		// measured points.
		return refused
	}
	r.mu.Lock()
	r.samples = append(r.samples, tcpSample{at: snap.Now, value: value})
	cutoff := snap.Now.Add(-tcpGrowthWindow)
	kept := r.samples[:0]
	for _, sample := range r.samples {
		if !sample.at.Before(cutoff) {
			kept = append(kept, sample)
		}
	}
	r.samples = kept
	history := make([]float64, len(kept))
	for i, sample := range kept {
		history[i] = sample.value
	}
	r.mu.Unlock()

	var severity string
	var reasons []string
	boundary := 0.0
	switch {
	case value >= TCPConnectionsCritical:
		severity, boundary = SeverityCritical, TCPConnectionsCritical
		reasons = append(reasons, fmt.Sprintf("%d established connections (>= %d)", int(value), TCPConnectionsCritical))
	case value >= TCPConnectionsWarning:
		severity, boundary = SeverityWarning, TCPConnectionsWarning
		reasons = append(reasons, fmt.Sprintf("%d established connections (>= %d)", int(value), TCPConnectionsWarning))
	}
	growing := SustainedConnectionGrowth(history)
	if growing {
		if severity == "" {
			severity = SeverityWarning
		}
		reasons = append(reasons, fmt.Sprintf("connection count climbing steadily (%d -> %d over %d samples)", int(history[0]), int(value), len(history)))
	}
	if severity == "" {
		return clearRule()
	}
	return activeRule(AlertCondition{
		Key:        hostKey,
		Severity:   severity,
		Message:    "TCP: " + strings.Join(reasons, "; ") + ".",
		MetricName: "tcp_connections",
		Value:      value,
		Threshold:  boundary,
		Details:    map[string]interface{}{"growth": growing, "window_samples": len(history)},
	})
}

// SustainedConnectionGrowth detects a monotone-enough climb. Strict
// monotonicity would be defeated by one sample of jitter, so the series must
// rise across the window to at least tcpGrowthFactor times its opening value,
// end at or near its own peak (not recovering from a spike), and never give
// back more than half of the rise in a single step.
func SustainedConnectionGrowth(values []float64) bool {
	if len(values) < tcpGrowthMinSamples {
		return false
	}
	first, last := values[0], values[len(values)-1]
	if last < tcpGrowthFloor {
		return false
	}
	if first <= 0 || last < first*tcpGrowthFactor {
		return false
	}
	peak := 0.0
	deepestDrop := 0.0
	for i, value := range values {
		if value > peak {
			peak = value
		}
		if i > 0 && values[i-1]-value > deepestDrop {
			deepestDrop = values[i-1] - value
		}
	}
	if last < peak*0.9 {
		return false
	}
	return deepestDrop <= (last-first)*0.5
}

type gpuUsageRule struct{}

func (gpuUsageRule) ID() string { return RuleGPUUsage }

func (gpuUsageRule) Evaluate(_ context.Context, snap AlertSnapshot) RuleEvaluation {
	if snap.Metrics == nil {
		return failedRule("no metrics snapshot")
	}
	state := snap.Metrics.GPUState
	if state.Status == "" {
		if snap.Metrics.GPUUsage == nil {
			return unsupportedRule("no GPU usage is reported on this host")
		}
		state = models.MetricState{Status: "measured", Value: *snap.Metrics.GPUUsage}
	}
	value, refused, ok := ruleMetricValue(state, 0)
	if !ok {
		return refused
	}
	if value >= GPUUsageWarning {
		return activeRule(AlertCondition{Key: hostKey, Severity: SeverityWarning, Message: fmt.Sprintf("GPU usage is %.1f%%.", value), MetricName: "gpu_usage_percent", Value: value, Threshold: GPUUsageWarning})
	}
	return clearRule()
}

// memoryPressureRule alerts on the platform-neutral memory-pressure level:
// elevated -> warning, critical -> critical. An unknown level means no
// supported signal exists, which is reported as unsupported, not healthy.
type memoryPressureRule struct{}

func (memoryPressureRule) ID() string { return RuleMemoryPressure }

func (memoryPressureRule) Evaluate(_ context.Context, snap AlertSnapshot) RuleEvaluation {
	if snap.PressureErr != nil {
		return failedRule(snap.PressureErr.Error())
	}
	reading := snap.MemoryPressure
	if reading == nil {
		return failedRule("no memory-pressure reading")
	}
	var severity string
	switch reading.Level {
	case collectors.MemoryPressureCritical:
		severity = SeverityCritical
	case collectors.MemoryPressureElevated:
		severity = SeverityWarning
	case collectors.MemoryPressureNormal:
		return clearRule()
	default:
		return unsupportedRule("memory-pressure level is unknown: " + reading.LevelReason)
	}
	signals := map[string]interface{}{}
	for name, state := range reading.Signals {
		if state.Status == "measured" {
			signals[name] = state.Value
		}
	}
	contributors := make([]interface{}, 0, len(reading.Contributors))
	for _, c := range reading.Contributors {
		contributors = append(contributors, map[string]interface{}{
			"pid": c.PID, "name": c.Name, "owner": c.Owner,
			"major_faults_per_second": c.MajorFaultsPerSecond, "fault_status": c.FaultStatus, "fault_counter": c.FaultCounter,
			"swap_kb": c.SwapKB, "swap_status": c.SwapStatus, "rss_kb": c.RSSKB,
		})
	}
	message := fmt.Sprintf("Memory pressure is %s: %s.", reading.Level, reading.LevelReason)
	if len(reading.Contributors) > 0 {
		top := reading.Contributors[0]
		message += fmt.Sprintf(" Top contributor: %s (pid %d, owner %s).", top.Name, top.PID, top.Owner)
	}
	return activeRule(AlertCondition{
		Key:        hostKey,
		Severity:   severity,
		Message:    message,
		MetricName: "memory_pressure_level",
		Value:      float64(collectors.LevelRank(reading.Level)),
		Details: map[string]interface{}{
			"level":               reading.Level,
			"level_source":        reading.LevelSource,
			"level_reason":        reading.LevelReason,
			"platform":            reading.Platform,
			"signals":             signals,
			"contributors":        contributors,
			"contributors_status": reading.ContributorsStatus,
			"contributors_reason": reading.ContributorsReason,
		},
	})
}
