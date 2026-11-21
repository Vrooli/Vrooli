package services

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

func TestOwnerActionAgeRuleFlagsOwnerAlertAndCarriesSourceReceiptKeys(t *testing.T) {
	now := time.Date(2026, 11, 15, 20, 0, 0, 0, time.UTC)
	repo := memory.NewRepository()
	source := &models.Alert{
		ID: "cond_sustained_process_cpu_image_tools_sd_cli", Type: ConditionAlertType,
		Severity: SeverityCritical, MetricName: RuleSustainedProcessCPU,
		Message: "process CPU is sustained", Timestamp: now.Add(-16 * time.Minute),
		Details: map[string]interface{}{"rule_id": RuleSustainedProcessCPU, "owner": "image-tools", "command": "sd-cli"},
	}
	if err := repo.CreateAlert(context.Background(), source); err != nil {
		t.Fatalf("create source alert: %v", err)
	}

	got := NewOwnerActionAgeRule(repo).Evaluate(context.Background(), AlertSnapshot{Now: now})
	if got.Status != RuleSupported || len(got.Conditions) != 1 {
		t.Fatalf("evaluation = %+v, want one supported overdue-owner finding", got)
	}
	condition := got.Conditions[0]
	if condition.Key != "image-tools:"+source.ID || condition.Severity != SeverityWarning || condition.MetricName != RuleOwnerActionAge {
		t.Fatalf("condition = %+v, want owner/source key and warning rule", condition)
	}
	for key, want := range map[string]interface{}{
		"rule_id": RuleOwnerActionAge, "owner_id": "image-tools", "source_alert_id": source.ID,
		"source_rule_id": RuleSustainedProcessCPU, "report_only": true,
	} {
		if condition.Details[key] != want {
			t.Errorf("details[%q] = %#v, want %#v", key, condition.Details[key], want)
		}
	}
}

func TestOwnerActionAgeRuleWaitsForOwnerAndAcknowledgementWindow(t *testing.T) {
	now := time.Date(2026, 11, 15, 20, 0, 0, 0, time.UTC)
	repo := memory.NewRepository()
	rows := []*models.Alert{
		{ID: "young", Type: ConditionAlertType, Timestamp: now.Add(-9 * time.Minute), Details: map[string]interface{}{"owner_id": "image-tools"}},
		{ID: "unknown-owner", Type: ConditionAlertType, Timestamp: now.Add(-time.Hour), Details: map[string]interface{}{"owner_id": "unknown"}},
		{ID: "no-owner", Type: ConditionAlertType, Timestamp: now.Add(-time.Hour)},
		{ID: "legacy-event", Type: "threshold", Timestamp: now.Add(-time.Hour), Details: map[string]interface{}{"owner_id": "image-tools"}},
		{ID: "self", Type: ConditionAlertType, MetricName: RuleOwnerActionAge, Timestamp: now.Add(-time.Hour), Details: map[string]interface{}{"owner_id": "image-tools"}},
	}
	acked := now.Add(-time.Minute)
	rows = append(rows, &models.Alert{ID: "acked", Type: ConditionAlertType, Timestamp: now.Add(-time.Hour), AckedAt: &acked, Details: map[string]interface{}{"owner_id": "image-tools"}})
	for _, row := range rows {
		if err := repo.CreateAlert(context.Background(), row); err != nil {
			t.Fatalf("create alert %s: %v", row.ID, err)
		}
	}

	got := NewOwnerActionAgeRule(repo).Evaluate(context.Background(), AlertSnapshot{Now: now})
	if got.Status != RuleSupported || len(got.Conditions) != 0 {
		t.Fatalf("evaluation = %+v, want no finding for young, ownerless, non-condition, self, or acknowledged rows", got)
	}
}

func TestOwnerActionAgeRuleFailsClosedOnRepositoryError(t *testing.T) {
	got := NewOwnerActionAgeRule(&failingAlertRepository{}).Evaluate(context.Background(), AlertSnapshot{Now: time.Now()})
	if got.Status != RuleFailed {
		t.Fatalf("status = %q, want failed when active alert inventory is unavailable", got.Status)
	}
}

func TestOwnerActionAgeAlertOpensByFifteenMinutesIncludingEvaluatorSustain(t *testing.T) {
	h := newEvaluatorHarness(t)
	h.eval.rules = []AlertRule{NewOwnerActionAgeRule(h.repo)}
	source := &models.Alert{
		ID: "cond_source_1", Type: ConditionAlertType, Severity: SeverityCritical,
		MetricName: RuleSustainedProcessCPU, Timestamp: h.clock.Now(),
		Details: map[string]interface{}{"rule_id": RuleSustainedProcessCPU, "owner_id": "image-tools"},
	}
	if err := h.repo.CreateAlert(context.Background(), source); err != nil {
		t.Fatalf("create source alert: %v", err)
	}

	h.eval.RunOnce(context.Background())
	h.pass(t, 10*time.Minute)
	if hasOwnerActionAlert(t, h.repo) {
		t.Fatal("age warning opened before ten-minute response threshold plus five-minute evaluator sustain")
	}
	h.pass(t, 5*time.Minute-time.Second)
	if hasOwnerActionAlert(t, h.repo) {
		t.Fatal("age warning opened before the complete 15-minute response bound")
	}
	h.pass(t, time.Second)
	if !hasOwnerActionAlert(t, h.repo) {
		t.Fatal("age warning did not open at the 15-minute response bound")
	}
}

func hasOwnerActionAlert(t *testing.T, repo *memory.MemoryRepository) bool {
	t.Helper()
	alerts, err := repo.GetActiveAlerts(context.Background())
	if err != nil {
		t.Fatalf("get active alerts: %v", err)
	}
	for _, alert := range alerts {
		if alertRuleID(alert) == RuleOwnerActionAge {
			return true
		}
	}
	return false
}

type failingAlertRepository struct{ memory.MemoryRepository }

func (failingAlertRepository) GetActiveAlerts(context.Context) ([]*models.Alert, error) {
	return nil, context.DeadlineExceeded
}
