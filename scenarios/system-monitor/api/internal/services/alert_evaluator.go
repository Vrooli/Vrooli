package services

// DOC: docs/concepts/ARCHITECTURE.md#alerting

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/logx"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// ConditionAlertType marks alerts whose lifecycle the evaluator owns.
const ConditionAlertType = "condition"

// AlertSnapshotSource supplies the scheduler-owned readings the evaluator
// judges. It never triggers collection.
type AlertSnapshotSource interface {
	GetCurrentMetricsFresh(ctx context.Context) (*models.MetricsResponse, error)
	GetMemoryPressure(ctx context.Context) (*models.MemoryPressure, error)
}

// AlertEscalator hands a sustained critical alert to the anomaly →
// investigation pipeline and returns the anomaly identifier it recorded.
type AlertEscalator interface {
	Escalate(ctx context.Context, alert *models.Alert) (string, error)
}

// conditionState tracks one rule/key across passes.
type conditionState struct {
	phase         conditionPhase
	activeSince   time.Time
	criticalSince time.Time
	clearSince    time.Time
	open          *models.Alert
	last          AlertCondition
}

// conditionPhase is the evaluator's explicit state machine for one rule/key.
// Unsupported evidence resets pending/clearing time but never resolves an
// already-open alert.
type conditionPhase uint8

const (
	conditionPending conditionPhase = iota
	conditionOpenWarning
	conditionOpenCritical
	conditionClearing
	conditionResolved
)

type conditionActionKind uint8

const (
	conditionActionOpen conditionActionKind = iota + 1
	conditionActionSeverity
	conditionActionEscalate
	conditionActionResolve
)

type conditionAction struct {
	kind      conditionActionKind
	state     *conditionState
	condition AlertCondition
}

func (p conditionPhase) String() string {
	switch p {
	case conditionPending:
		return "pending"
	case conditionOpenWarning:
		return "open_warning"
	case conditionOpenCritical:
		return "open_critical"
	case conditionClearing:
		return "clearing"
	case conditionResolved:
		return "resolved"
	default:
		return "unknown"
	}
}

func phaseForSeverity(severity string) conditionPhase {
	if severity == SeverityCritical {
		return conditionOpenCritical
	}
	return conditionOpenWarning
}

// RuleStatus is the evaluator's latest view of one rule.
type RuleStatus struct {
	RuleID          string
	Status          string
	Reason          string
	ConditionActive bool
	PendingSince    time.Time
	EvaluatedAt     time.Time
}

// AlertEvaluator is the single server-side source of alert state.
//
// A condition must hold continuously for the sustain duration before an
// alert opens; there is at most one open alert per rule/key; an open alert
// escalates to critical only after the critical condition itself has held
// for the sustain duration, and resolves only after the condition has stayed
// clear for the same duration. A sustained critical alert is escalated once
// into the anomaly → investigation pipeline. Unsupported or failed rules
// never open or resolve anything.
type AlertEvaluator struct {
	settings  *SettingsManager
	source    AlertSnapshotSource
	repo      repository.AlertRepository
	alerts    *AlertService
	escalator AlertEscalator
	rules     []AlertRule
	clock     Clock
	log       logx.Logger

	mu          sync.Mutex
	states      map[string]*conditionState
	ruleStatus  map[string]RuleStatus
	adopted     bool
	evaluatedAt time.Time

	evaluated chan struct{}
	cancel    context.CancelFunc
	done      chan struct{}
}

// AlertEvaluatorOption configures an AlertEvaluator.
type AlertEvaluatorOption func(*AlertEvaluator)

// WithAlertEvaluatorClock injects the clock (tests use StubClock).
func WithAlertEvaluatorClock(c Clock) AlertEvaluatorOption {
	return func(e *AlertEvaluator) {
		if c != nil {
			e.clock = c
		}
	}
}

// WithAlertRules replaces the rule set.
func WithAlertRules(rules ...AlertRule) AlertEvaluatorOption {
	return func(e *AlertEvaluator) { e.rules = rules }
}

// WithAdditionalAlertRules appends rules to the default set.
func WithAdditionalAlertRules(rules ...AlertRule) AlertEvaluatorOption {
	return func(e *AlertEvaluator) { e.rules = append(e.rules, rules...) }
}

// WithAlertEscalator sets the sustained-critical escalation target.
func WithAlertEscalator(esc AlertEscalator) AlertEvaluatorOption {
	return func(e *AlertEvaluator) { e.escalator = esc }
}

// WithAlertEvaluationSignal sets a channel signalled after each pass so tests
// synchronise with the loop instead of sleeping.
func WithAlertEvaluationSignal(ch chan struct{}) AlertEvaluatorOption {
	return func(e *AlertEvaluator) { e.evaluated = ch }
}

// NewAlertEvaluator builds the evaluator with the default rules.
func NewAlertEvaluator(settings *SettingsManager, source AlertSnapshotSource, repo repository.AlertRepository, alerts *AlertService, log logx.Logger, opts ...AlertEvaluatorOption) *AlertEvaluator {
	if log == nil {
		log = logx.Default()
	}
	e := &AlertEvaluator{
		settings:   settings,
		source:     source,
		repo:       repo,
		alerts:     alerts,
		rules:      DefaultAlertRules(),
		clock:      RealClock{},
		log:        log,
		states:     make(map[string]*conditionState),
		ruleStatus: make(map[string]RuleStatus),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// SustainDuration is the live sustain/clear duration from settings.
func (e *AlertEvaluator) SustainDuration() time.Duration {
	seconds := e.settings.GetSettings().AlertSustainSeconds
	if seconds <= 0 {
		seconds = defaultSettings.AlertSustainSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (e *AlertEvaluator) interval() time.Duration {
	seconds := e.settings.GetSettings().ThresholdCheckInterval
	if seconds < minThresholdIntervalSeconds {
		seconds = defaultSettings.ThresholdCheckInterval
	}
	return time.Duration(seconds) * time.Second
}

// Start launches the evaluation loop on the threshold-check interval.
func (e *AlertEvaluator) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.done = make(chan struct{})
	go func() {
		defer close(e.done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-e.clock.After(e.interval()):
				e.RunOnce(ctx)
			}
		}
	}()
}

// Stop halts the loop and waits for it to exit.
func (e *AlertEvaluator) Stop() {
	if e.cancel == nil {
		return
	}
	e.cancel()
	<-e.done
}

func stateKey(ruleID, key string) string { return ruleID + "\x00" + key }

// RunOnce performs one evaluation pass.
func (e *AlertEvaluator) RunOnce(ctx context.Context) {
	defer e.signal()
	settings := e.settings.GetSettings()
	// A paused monitor observes nothing; open alerts stay as they are rather
	// than resolving on the absence of evidence.
	if !settings.Active {
		return
	}
	now := e.clock.Now()
	e.adoptOpenAlerts(ctx)

	snapshot := AlertSnapshot{Now: now, Settings: settings}
	snapshot.Metrics, snapshot.MetricsErr = e.source.GetCurrentMetricsFresh(ctx)
	snapshot.MemoryPressure, snapshot.PressureErr = e.source.GetMemoryPressure(ctx)
	sustain := e.SustainDuration()

	for _, rule := range e.rules {
		eval := rule.Evaluate(ctx, snapshot)
		e.apply(ctx, rule.ID(), eval, now, sustain, settings.AutoInvestigateCriticalRules[rule.ID()])
	}
	e.mu.Lock()
	e.evaluatedAt = now
	e.mu.Unlock()
}

func (e *AlertEvaluator) apply(ctx context.Context, ruleID string, eval RuleEvaluation, now time.Time, sustain time.Duration, investigateCritical bool) {
	status := RuleStatus{RuleID: ruleID, Status: eval.Status, Reason: eval.Reason, EvaluatedAt: now}
	if status.Status == "" {
		status.Status = RuleSupported
	}

	e.mu.Lock()
	if status.Status != RuleSupported {
		e.resetUnsupportedRuleLocked(ruleID)
		e.ruleStatus[ruleID] = status
		e.mu.Unlock()
		return
	}

	var actions []conditionAction
	active := keyedConditions(ruleID, eval.Conditions)
	actions = append(actions, e.updateActiveConditionsLocked(active, now, sustain, investigateCritical, &status)...)
	actions = append(actions, e.resolveMissingConditionsLocked(ruleID, active, now, sustain)...)
	e.ruleStatus[ruleID] = status
	e.mu.Unlock()
	e.executeConditionActions(ctx, ruleID, actions, now, sustain)
}

func keyedConditions(ruleID string, conditions []AlertCondition) map[string]AlertCondition {
	active := make(map[string]AlertCondition, len(conditions))
	for _, condition := range conditions {
		if condition.Key == "" {
			condition.Key = hostKey
		}
		active[stateKey(ruleID, condition.Key)] = condition
	}
	return active
}

// resetUnsupportedRuleLocked breaks pending/clearing continuity while
// retaining open alerts until fresh evidence can resolve them.
func (e *AlertEvaluator) resetUnsupportedRuleLocked(ruleID string) {
	for id, state := range e.states {
		if !strings.HasPrefix(id, ruleID+"\x00") {
			continue
		}
		state.activeSince, state.criticalSince, state.clearSince = time.Time{}, time.Time{}, time.Time{}
		if state.open == nil {
			delete(e.states, id)
			continue
		}
		state.phase = phaseForSeverity(state.open.Severity)
	}
}

// updateActiveConditionsLocked advances each condition present in this pass.
// The caller holds e.mu so state and status move as one evaluator transition.
func (e *AlertEvaluator) updateActiveConditionsLocked(
	active map[string]AlertCondition,
	now time.Time,
	sustain time.Duration,
	investigateCritical bool,
	status *RuleStatus,
) []conditionAction {
	var actions []conditionAction
	for id, condition := range active {
		state := e.states[id]
		if state == nil {
			state = &conditionState{}
			e.states[id] = state
		}
		state.clearSince = time.Time{}
		state.last = condition
		if state.activeSince.IsZero() {
			state.activeSince = now
		}
		if condition.Severity == SeverityCritical {
			if state.criticalSince.IsZero() {
				state.criticalSince = now
			}
		} else {
			state.criticalSince = time.Time{}
		}
		if status.PendingSince.IsZero() || state.activeSince.Before(status.PendingSince) {
			status.PendingSince = state.activeSince
		}
		status.ConditionActive = true
		severity := sustainedSeverity(state, now, sustain)
		if state.open == nil {
			state.phase = conditionPending
			if now.Sub(state.activeSince) >= sustain {
				state.phase = phaseForSeverity(severity)
			}
		} else {
			state.phase = phaseForSeverity(severity)
		}
		switch {
		case state.open == nil && now.Sub(state.activeSince) >= sustain:
			actions = append(actions, conditionAction{kind: conditionActionOpen, state: state, condition: condition})
		case state.open != nil && state.open.Severity != severity:
			actions = append(actions, conditionAction{kind: conditionActionSeverity, state: state, condition: condition})
		}
		if investigateCritical && (state.open != nil || (state.open == nil && now.Sub(state.activeSince) >= sustain)) {
			if severity == SeverityCritical && !alertEscalated(state.open) {
				actions = append(actions, conditionAction{kind: conditionActionEscalate, state: state, condition: condition})
			}
		}
	}
	return actions
}

// resolveMissingConditionsLocked starts or completes clear windows for
// previously known conditions absent from this supported evaluation.
func (e *AlertEvaluator) resolveMissingConditionsLocked(ruleID string, active map[string]AlertCondition, now time.Time, sustain time.Duration) []conditionAction {
	var actions []conditionAction
	for id, st := range e.states {
		if !strings.HasPrefix(id, ruleID+"\x00") {
			continue
		}
		if _, still := active[id]; still {
			continue
		}
		st.activeSince, st.criticalSince = time.Time{}, time.Time{}
		if st.open == nil {
			delete(e.states, id)
			continue
		}
		if st.clearSince.IsZero() {
			st.clearSince = now
		}
		st.phase = conditionClearing
		if now.Sub(st.clearSince) >= sustain {
			actions = append(actions, conditionAction{kind: conditionActionResolve, state: st})
			delete(e.states, id)
		}
	}
	return actions
}

func (e *AlertEvaluator) executeConditionActions(ctx context.Context, ruleID string, actions []conditionAction, now time.Time, sustain time.Duration) {
	// Persistence happens outside the lock; each action owns its state.
	for _, action := range actions {
		switch action.kind {
		case conditionActionOpen:
			e.open(ctx, ruleID, action.state, action.condition, now, sustain)
		case conditionActionSeverity:
			e.updateSeverity(ctx, action.state, action.condition, now, sustain)
		case conditionActionEscalate:
			e.escalate(ctx, action.state, now)
		case conditionActionResolve:
			e.resolve(ctx, action.state, now)
		}
	}
}

// sustainedSeverity is critical only once the critical condition itself has
// held for the sustain duration; otherwise a held condition is a warning.
func sustainedSeverity(st *conditionState, now time.Time, sustain time.Duration) string {
	if !st.criticalSince.IsZero() && now.Sub(st.criticalSince) >= sustain {
		return SeverityCritical
	}
	return SeverityWarning
}

func alertEscalated(alert *models.Alert) bool {
	if alert == nil {
		return false
	}
	escalated, _ := alert.Details["escalated"].(bool)
	return escalated
}

func (e *AlertEvaluator) conditionDetails(ruleID string, condition AlertCondition, st *conditionState) map[string]interface{} {
	details := map[string]interface{}{}
	for k, v := range condition.Details {
		details[k] = v
	}
	details["rule_id"] = ruleID
	details["key"] = condition.Key
	details["threshold"] = condition.Threshold
	details["condition_since"] = st.activeSince.UTC().Format(time.RFC3339Nano)
	details["lifecycle_phase"] = st.phase.String()
	return details
}

func (e *AlertEvaluator) open(ctx context.Context, ruleID string, st *conditionState, condition AlertCondition, now time.Time, sustain time.Duration) {
	alert := &models.Alert{
		ID:          fmt.Sprintf("cond_%s_%s_%d", ruleID, condition.Key, now.UnixNano()),
		Type:        ConditionAlertType,
		Severity:    sustainedSeverity(st, now, sustain),
		Message:     condition.Message,
		MetricName:  condition.MetricName,
		MetricValue: condition.Value,
		Details:     e.conditionDetails(ruleID, condition, st),
		Timestamp:   now,
	}
	alert.Details["escalated"] = false
	if err := e.alerts.OpenConditionAlert(ctx, alert); err != nil {
		e.log.Error("persist condition alert failed", "rule", ruleID, "key", condition.Key, "error", err)
		return
	}
	e.mu.Lock()
	st.open = alert
	st.phase = phaseForSeverity(alert.Severity)
	e.mu.Unlock()
	e.log.Warn("alert opened", "rule", ruleID, "key", condition.Key, "severity", alert.Severity, "message", alert.Message)
}

func (e *AlertEvaluator) updateSeverity(ctx context.Context, st *conditionState, condition AlertCondition, now time.Time, sustain time.Duration) {
	e.mu.Lock()
	updated := *st.open
	ruleID, _ := updated.Details["rule_id"].(string)
	details := e.conditionDetails(ruleID, condition, st)
	for _, keep := range []string{"escalated", "anomaly_id", "escalated_at"} {
		if value, ok := st.open.Details[keep]; ok {
			details[keep] = value
		}
	}
	updated.Details = details
	updated.Severity = sustainedSeverity(st, now, sustain)
	updated.Message = condition.Message
	updated.MetricValue = condition.Value
	e.mu.Unlock()
	if err := e.repo.UpdateAlert(ctx, &updated); err != nil {
		e.log.Error("update condition alert failed", "alert_id", updated.ID, "error", err)
		return
	}
	e.mu.Lock()
	st.open = &updated
	st.phase = phaseForSeverity(updated.Severity)
	e.mu.Unlock()
	e.alerts.NotifyTransition(ctx, &updated, AlertTransitionSeverityChange)
}

func (e *AlertEvaluator) escalate(ctx context.Context, st *conditionState, now time.Time) {
	e.mu.Lock()
	if st.open == nil || alertEscalated(st.open) {
		e.mu.Unlock()
		return
	}
	snapshot := *st.open
	e.mu.Unlock()
	if e.escalator == nil {
		return
	}
	anomalyID, err := e.escalator.Escalate(ctx, &snapshot)
	if err != nil {
		e.log.Error("escalate sustained critical alert failed", "alert_id", snapshot.ID, "error", err)
		return
	}
	e.mu.Lock()
	updated := *st.open
	details := make(map[string]interface{}, len(updated.Details)+3)
	for k, v := range updated.Details {
		details[k] = v
	}
	details["escalated"] = true
	details["anomaly_id"] = anomalyID
	details["escalated_at"] = now.UTC().Format(time.RFC3339Nano)
	updated.Details = details
	st.open = &updated
	e.mu.Unlock()
	if err := e.repo.UpdateAlert(ctx, &updated); err != nil {
		e.log.Error("record alert escalation failed", "alert_id", updated.ID, "error", err)
		return
	}
	e.alerts.NotifyTransition(ctx, &updated, AlertTransitionEscalated)
}

func (e *AlertEvaluator) resolve(ctx context.Context, st *conditionState, now time.Time) {
	resolved := *st.open
	resolvedAt := now
	resolved.ResolvedAt = &resolvedAt
	resolved.Details = make(map[string]interface{}, len(st.open.Details)+1)
	for key, value := range st.open.Details {
		resolved.Details[key] = value
	}
	resolved.Details["lifecycle_phase"] = conditionResolved.String()
	if err := e.repo.UpdateAlert(ctx, &resolved); err != nil {
		e.log.Error("resolve condition alert failed", "alert_id", resolved.ID, "error", err)
		return
	}
	e.alerts.NotifyTransition(ctx, &resolved, AlertTransitionResolved)
	e.log.Info("alert resolved", "alert_id", resolved.ID)
}

// adoptOpenAlerts restores open condition alerts after a restart so dedup
// holds across process lifetimes. An adopted alert is treated as a condition
// that has already been sustained.
func (e *AlertEvaluator) adoptOpenAlerts(ctx context.Context) {
	e.mu.Lock()
	if e.adopted {
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()
	open, err := e.repo.ListAlerts(ctx, repository.AlertFilter{Type: ConditionAlertType, Status: "active"})
	if err != nil {
		e.log.Error("load open condition alerts failed", "error", err)
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.adopted = true
	for _, alert := range open {
		ruleID, _ := alert.Details["rule_id"].(string)
		key, _ := alert.Details["key"].(string)
		if ruleID == "" {
			continue
		}
		if key == "" {
			key = hostKey
		}
		id := stateKey(ruleID, key)
		if existing, ok := e.states[id]; ok && existing.open != nil {
			continue
		}
		copied := *alert
		st := &conditionState{open: &copied, activeSince: alert.Timestamp, phase: phaseForSeverity(alert.Severity)}
		if alert.Severity == SeverityCritical {
			st.criticalSince = alert.Timestamp
		}
		e.states[id] = st
	}
}

func (e *AlertEvaluator) signal() {
	if e.evaluated != nil {
		e.evaluated <- struct{}{}
	}
}

// RuleStatuses returns the latest rule states in rule order.
func (e *AlertEvaluator) RuleStatuses() []RuleStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]RuleStatus, 0, len(e.rules))
	for _, rule := range e.rules {
		if status, ok := e.ruleStatus[rule.ID()]; ok {
			out = append(out, status)
		}
	}
	return out
}

// EvaluatedAt is the time of the most recent completed pass.
func (e *AlertEvaluator) EvaluatedAt() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.evaluatedAt
}

// MonitoringActive reports whether the evaluator is running passes.
func (e *AlertEvaluator) MonitoringActive() bool { return e.settings.IsActive() }

// AlertQuery selects alerts for ListAlerts.
type AlertQuery struct {
	Status        string // active (default) | resolved | all
	IncludeEvents bool
	Limit         int
}

// ListAlerts returns persisted alerts. Condition alerts only, unless events
// (threshold violations, anomalies) are explicitly requested.
func (e *AlertEvaluator) ListAlerts(ctx context.Context, query AlertQuery) ([]*models.Alert, error) {
	status := query.Status
	switch status {
	case "", "active":
		status = "active"
	case "resolved", "all":
	default:
		return nil, fmt.Errorf("unknown alert status %q (want active, resolved, or all)", query.Status)
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	filter := repository.AlertFilter{Limit: limit}
	if status != "all" {
		filter.Status = status
	}
	if !query.IncludeEvents {
		filter.Type = ConditionAlertType
	}
	alerts, err := e.repo.ListAlerts(ctx, filter)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(alerts, func(i, j int) bool {
		if severityRank(alerts[i].Severity) != severityRank(alerts[j].Severity) {
			return severityRank(alerts[i].Severity) > severityRank(alerts[j].Severity)
		}
		return alerts[i].Timestamp.After(alerts[j].Timestamp)
	})
	return alerts, nil
}

func severityRank(severity string) int {
	switch severity {
	case SeverityCritical:
		return 4
	case "high":
		return 3
	case SeverityWarning, "medium":
		return 2
	default:
		return 1
	}
}
