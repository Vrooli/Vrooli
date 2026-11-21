package services

// DOC: docs/concepts/ARCHITECTURE.md#alerting

import (
	"context"
	"encoding/json"
	"fmt"

	catalogpkg "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/investigations"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// AnomalyRunRecorder runs a catalog investigation for an anomaly and
// persists the run with the anomaly identifier. *ScriptService satisfies it.
type AnomalyRunRecorder interface {
	ExecuteForAnomaly(ctx context.Context, entryID, anomalyID string, findings []catalogpkg.Finding) (ScriptExecution, error)
}

// escalationInvestigation names the native catalog investigation that gathers
// evidence for each rule. Rules without a specific entry use the general
// resource-leak sweep.
var escalationInvestigation = map[string]string{
	RuleMemoryPressure: "memory-pressure",
	RuleMemoryUsage:    "memory-pressure",
}

const defaultEscalationInvestigation = "resource-leak-detector"

// AnomalyEscalator turns a sustained critical condition alert into an
// anomaly record and a persisted native investigation run carrying the
// anomaly identifier and a finding. It never spawns an AI agent; an
// agent-manager investigation remains an operator decision.
type AnomalyEscalator struct {
	alerts *AlertService
	runs   AnomalyRunRecorder
	clock  Clock
}

// NewAnomalyEscalator builds the escalator. A nil runs recorder records the
// anomaly only.
func NewAnomalyEscalator(alerts *AlertService, runs AnomalyRunRecorder, clock Clock) *AnomalyEscalator {
	if clock == nil {
		clock = RealClock{}
	}
	return &AnomalyEscalator{alerts: alerts, runs: runs, clock: clock}
}

// Escalate implements AlertEscalator.
func (e *AnomalyEscalator) Escalate(ctx context.Context, alert *models.Alert) (string, error) {
	ruleID, _ := alert.Details["rule_id"].(string)
	if ruleID == "" {
		ruleID = alert.MetricName
	}
	now := e.clock.Now()
	anomalyID := fmt.Sprintf("anomaly_%s_%d", ruleID, now.UnixNano())
	evidence := map[string]interface{}{"alert_id": alert.ID, "rule_id": ruleID, "metric_value": alert.MetricValue}
	for k, v := range alert.Details {
		evidence[k] = v
	}
	anomaly := &models.Anomaly{
		ID:          anomalyID,
		Type:        ruleID,
		Severity:    SeverityCritical,
		Description: "Sustained critical condition: " + alert.Message,
		MetricData:  evidence,
		DetectedAt:  now,
		Status:      "investigating",
	}
	if err := e.alerts.SendAnomaly(ctx, anomaly); err != nil {
		return "", fmt.Errorf("record anomaly: %w", err)
	}
	if e.runs == nil {
		return anomalyID, nil
	}
	entry := escalationInvestigation[ruleID]
	if entry == "" {
		entry = defaultEscalationInvestigation
	}
	detail, _ := json.Marshal(evidence)
	finding := catalogpkg.Finding{Severity: SeverityCritical, Code: "sustained_" + ruleID, Summary: alert.Message, DetailJSON: string(detail)}
	if _, err := e.runs.ExecuteForAnomaly(ctx, entry, anomalyID, []catalogpkg.Finding{finding}); err != nil {
		return anomalyID, fmt.Errorf("run %s investigation: %w", entry, err)
	}
	return anomalyID, nil
}
