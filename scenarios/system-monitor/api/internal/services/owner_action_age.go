package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// The alert evaluator adds its ordinary five-minute condition sustain. Trigger
// the age finding at ten minutes so its notification opens by minute fifteen.
const ownerActionResponseWindow = 10 * time.Minute

type ownerActionAgeRule struct{ alerts repository.AlertRepository }

func NewOwnerActionAgeRule(alerts repository.AlertRepository) AlertRule {
	return ownerActionAgeRule{alerts: alerts}
}

func (ownerActionAgeRule) ID() string { return RuleOwnerActionAge }

func (r ownerActionAgeRule) Evaluate(ctx context.Context, snap AlertSnapshot) RuleEvaluation {
	if r.alerts == nil {
		return unsupportedRule("alert repository is not configured")
	}
	active, err := r.alerts.GetActiveAlerts(ctx)
	if err != nil {
		return failedRule("read active owner alerts: " + err.Error())
	}
	conditions := make([]AlertCondition, 0)
	for _, alert := range active {
		if !eligibleOwnerActionSource(alert) || alert.AckedAt != nil || alert.Timestamp.IsZero() || alert.Timestamp.After(snap.Now) {
			continue
		}
		age := snap.Now.Sub(alert.Timestamp)
		if age < ownerActionResponseWindow {
			continue
		}
		owner := ownerFromAlert(alert)
		conditions = append(conditions, AlertCondition{
			Key: owner + ":" + alert.ID, Severity: SeverityWarning,
			Message:    fmt.Sprintf("Owner %s has not acknowledged alert %s after %s; route the source alert to its owner for a fix or decision.", owner, alert.ID, age.Round(time.Minute)),
			MetricName: RuleOwnerActionAge, Value: age.Minutes(), Threshold: ownerActionResponseWindow.Minutes(),
			Details: map[string]interface{}{
				"rule_id": RuleOwnerActionAge, "owner_id": owner, "source_alert_id": alert.ID,
				"source_rule_id": alertRuleID(alert), "source_metric": alert.MetricName,
				"source_opened_at": alert.Timestamp.UTC().Format(time.RFC3339Nano),
				"age_seconds":      age.Seconds(), "response_window_seconds": ownerActionResponseWindow.Seconds(),
				"report_only": true,
			},
		})
	}
	if len(conditions) == 0 {
		return clearRule()
	}
	return activeRule(conditions...)
}

func eligibleOwnerActionSource(alert *models.Alert) bool {
	if alert == nil || alert.Type != ConditionAlertType || alert.ID == "" || alert.ResolvedAt != nil || alertRuleID(alert) == RuleOwnerActionAge || alert.MetricName == RuleOwnerActionAge {
		return false
	}
	return ownerFromAlert(alert) != ""
}

func alertRuleID(alert *models.Alert) string {
	if alert == nil || alert.Details == nil {
		return ""
	}
	if id, ok := alert.Details["rule_id"].(string); ok {
		return id
	}
	return ""
}

func ownerFromAlert(alert *models.Alert) string {
	if alert == nil || alert.Details == nil {
		return ""
	}
	for _, key := range []string{"owner_id", "owner"} {
		if owner, ok := alert.Details[key].(string); ok && owner != "" && owner != "unknown" {
			return owner
		}
	}
	return ""
}
