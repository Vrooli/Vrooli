package sqlite

import (
	"context"
	"testing"
	"time"

	"system-monitor-api/internal/models"
	"system-monitor-api/internal/repository"
)

func newTestRepo(t *testing.T) *Repository {
	t.Helper()
	repo, err := NewInMemoryRepository()
	if err != nil {
		t.Fatalf("NewInMemoryRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

// ---------------------------------------------------------------------------
// Metrics
// ---------------------------------------------------------------------------

func TestSaveAndGetMetrics(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.SaveMetrics(ctx, "cpu", map[string]interface{}{"usage_percent": 42.5}); err != nil {
		t.Fatalf("SaveMetrics: %v", err)
	}
	if err := repo.SaveMetrics(ctx, "memory", map[string]interface{}{"usage_percent": 60.0}); err != nil {
		t.Fatalf("SaveMetrics: %v", err)
	}

	results, err := repo.GetMetrics(ctx, repository.MetricsFilter{})
	if err != nil {
		t.Fatalf("GetMetrics: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected metrics, got none")
	}
}

func TestGetMetricsWithFilter(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.SaveMetrics(ctx, "cpu", map[string]interface{}{"usage_percent": 10.0}); err != nil {
		t.Fatalf("SaveMetrics(cpu): %v", err)
	}
	if err := repo.SaveMetrics(ctx, "memory", map[string]interface{}{"usage_percent": 20.0}); err != nil {
		t.Fatalf("SaveMetrics(memory): %v", err)
	}

	results, err := repo.GetMetrics(ctx, repository.MetricsFilter{CollectorName: "cpu"})
	if err != nil {
		t.Fatalf("GetMetrics: %v", err)
	}

	for _, r := range results {
		if r.CPUUsage == 0 {
			t.Error("expected non-zero CPU usage for cpu filter")
		}
	}
}

func TestGetMetricsWithTimeRange(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.SaveMetrics(ctx, "cpu", map[string]interface{}{"usage_percent": 50.0}); err != nil {
		t.Fatalf("SaveMetrics: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	now := time.Now()
	results, err := repo.GetMetrics(ctx, repository.MetricsFilter{
		TimeRange: repository.TimeRange{
			StartTime: now.Add(-1 * time.Second),
			EndTime:   now.Add(1 * time.Second),
		},
	})
	if err != nil {
		t.Fatalf("GetMetrics: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected metrics within time range")
	}
}

func TestGetMetricsWithLimit(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := repo.SaveMetrics(ctx, "cpu", map[string]interface{}{"usage_percent": float64(i * 10)}); err != nil {
			t.Fatalf("SaveMetrics[%d]: %v", i, err)
		}
	}

	results, err := repo.GetMetrics(ctx, repository.MetricsFilter{Limit: 2})
	if err != nil {
		t.Fatalf("GetMetrics: %v", err)
	}
	if len(results) > 2 {
		t.Fatalf("expected at most 2 results, got %d", len(results))
	}
}

func TestGetLatestMetrics(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.SaveMetrics(ctx, "cpu", map[string]interface{}{"usage_percent": 75.0}); err != nil {
		t.Fatalf("SaveMetrics(cpu): %v", err)
	}
	if err := repo.SaveMetrics(ctx, "memory", map[string]interface{}{"usage_percent": 80.0}); err != nil {
		t.Fatalf("SaveMetrics(memory): %v", err)
	}

	latest, err := repo.GetLatestMetrics(ctx)
	if err != nil {
		t.Fatalf("GetLatestMetrics: %v", err)
	}
	if latest.CPUUsage != 75.0 {
		t.Errorf("expected CPU 75.0, got %f", latest.CPUUsage)
	}
	if latest.MemoryUsage != 80.0 {
		t.Errorf("expected Memory 80.0, got %f", latest.MemoryUsage)
	}
}

func TestGetLatestMetricsEmpty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetLatestMetrics(ctx)
	if err == nil {
		t.Fatal("expected error for empty metrics")
	}
}

func TestGetHistoricalMetrics(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.SaveMetrics(ctx, "cpu", map[string]interface{}{"usage_percent": 50.0}); err != nil {
		t.Fatalf("SaveMetrics: %v", err)
	}

	points, err := repo.GetHistoricalMetrics(ctx, "usage_percent", repository.TimeRange{
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now().Add(1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("GetHistoricalMetrics: %v", err)
	}
	if len(points) == 0 {
		t.Fatal("expected historical data points")
	}
	if points[0].Value != 50.0 {
		t.Errorf("expected value 50.0, got %f", points[0].Value)
	}
}

func TestGetEarliestMetricTime(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetEarliestMetricTime(ctx)
	if err == nil {
		t.Fatal("expected error for empty metrics")
	}

	if err := repo.SaveMetrics(ctx, "cpu", map[string]interface{}{"usage_percent": 10.0}); err != nil {
		t.Fatalf("SaveMetrics: %v", err)
	}

	ts, err := repo.GetEarliestMetricTime(ctx)
	if err != nil {
		t.Fatalf("GetEarliestMetricTime: %v", err)
	}
	if ts.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
}

// ---------------------------------------------------------------------------
// Investigations
// ---------------------------------------------------------------------------

func TestInvestigationCRUD(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	inv := &models.Investigation{
		ID:        "inv-1",
		Status:    "in_progress",
		AnomalyID: "anom-1",
		StartTime: time.Now(),
		Progress:  50,
		Findings:  "initial findings",
	}

	if err := repo.CreateInvestigation(ctx, inv); err != nil {
		t.Fatalf("CreateInvestigation: %v", err)
	}

	got, err := repo.GetInvestigation(ctx, "inv-1")
	if err != nil {
		t.Fatalf("GetInvestigation: %v", err)
	}
	if got.Status != "in_progress" {
		t.Errorf("expected in_progress, got %s", got.Status)
	}
	if got.Findings != "initial findings" {
		t.Errorf("expected 'initial findings', got %s", got.Findings)
	}

	// Update
	inv.Status = "completed"
	inv.Findings = "updated findings"
	if err := repo.UpdateInvestigation(ctx, inv); err != nil {
		t.Fatalf("UpdateInvestigation: %v", err)
	}

	got, err = repo.GetInvestigation(ctx, "inv-1")
	if err != nil {
		t.Fatalf("GetInvestigation after update: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("expected completed, got %s", got.Status)
	}
}

func TestListInvestigations(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.CreateInvestigation(ctx, &models.Investigation{ID: "inv-1", Status: "completed", StartTime: time.Now()}); err != nil {
		t.Fatalf("CreateInvestigation(inv-1): %v", err)
	}
	if err := repo.CreateInvestigation(ctx, &models.Investigation{ID: "inv-2", Status: "in_progress", StartTime: time.Now()}); err != nil {
		t.Fatalf("CreateInvestigation(inv-2): %v", err)
	}

	all, err := repo.ListInvestigations(ctx, repository.InvestigationFilter{})
	if err != nil {
		t.Fatalf("ListInvestigations: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}

	filtered, err := repo.ListInvestigations(ctx, repository.InvestigationFilter{Status: "completed"})
	if err != nil {
		t.Fatalf("ListInvestigations with filter: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("expected 1, got %d", len(filtered))
	}
}

func TestGetLatestInvestigation(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.CreateInvestigation(ctx, &models.Investigation{ID: "inv-old", Status: "completed", StartTime: time.Now().Add(-1 * time.Hour)}); err != nil {
		t.Fatalf("CreateInvestigation(inv-old): %v", err)
	}
	if err := repo.CreateInvestigation(ctx, &models.Investigation{ID: "inv-new", Status: "in_progress", StartTime: time.Now()}); err != nil {
		t.Fatalf("CreateInvestigation(inv-new): %v", err)
	}

	latest, err := repo.GetLatestInvestigation(ctx)
	if err != nil {
		t.Fatalf("GetLatestInvestigation: %v", err)
	}
	if latest.ID != "inv-new" {
		t.Errorf("expected inv-new, got %s", latest.ID)
	}
}

func TestSaveInvestigationStep(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.CreateInvestigation(ctx, &models.Investigation{ID: "inv-1", Status: "in_progress", StartTime: time.Now()}); err != nil {
		t.Fatalf("CreateInvestigation: %v", err)
	}

	step := &models.InvestigationStep{
		Name:      "collect-metrics",
		Status:    "completed",
		StartTime: time.Now(),
		Findings:  "metrics collected",
	}
	if err := repo.SaveInvestigationStep(ctx, "inv-1", step); err != nil {
		t.Fatalf("SaveInvestigationStep: %v", err)
	}

	inv, _ := repo.GetInvestigation(ctx, "inv-1")
	if len(inv.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(inv.Steps))
	}
	if inv.Steps[0].Name != "collect-metrics" {
		t.Errorf("expected collect-metrics, got %s", inv.Steps[0].Name)
	}
}

func TestSaveInvestigationStepNotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.SaveInvestigationStep(ctx, "nonexistent", &models.InvestigationStep{Name: "test"})
	if err == nil {
		t.Fatal("expected error for nonexistent investigation")
	}
}

// ---------------------------------------------------------------------------
// Reports
// ---------------------------------------------------------------------------

func TestReportCRUD(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	report := &models.Report{
		ID:          "rpt-1",
		Type:        "daily",
		GeneratedAt: time.Now(),
		Data:        map[string]interface{}{"key": "value"},
		Format:      "json",
	}

	if err := repo.CreateReport(ctx, report); err != nil {
		t.Fatalf("CreateReport: %v", err)
	}

	got, err := repo.GetReport(ctx, "rpt-1")
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if got.Type != "daily" {
		t.Errorf("expected daily, got %s", got.Type)
	}
}

func TestListReports(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.CreateReport(ctx, &models.Report{ID: "rpt-1", Type: "daily", GeneratedAt: time.Now()}); err != nil {
		t.Fatalf("CreateReport(rpt-1): %v", err)
	}
	if err := repo.CreateReport(ctx, &models.Report{ID: "rpt-2", Type: "weekly", GeneratedAt: time.Now()}); err != nil {
		t.Fatalf("CreateReport(rpt-2): %v", err)
	}

	all, err := repo.ListReports(ctx, repository.ReportFilter{})
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}

	filtered, err := repo.ListReports(ctx, repository.ReportFilter{Type: "daily"})
	if err != nil {
		t.Fatalf("ListReports with filter: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("expected 1, got %d", len(filtered))
	}
}

func TestEnhancedReportCRUD(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	report := &models.EnhancedSystemReport{
		ReportID:    "erpt-1",
		ReportType:  "comprehensive",
		GeneratedAt: time.Now(),
	}

	if err := repo.SaveEnhancedReport(ctx, report); err != nil {
		t.Fatalf("SaveEnhancedReport: %v", err)
	}

	got, err := repo.GetEnhancedReport(ctx, "erpt-1")
	if err != nil {
		t.Fatalf("GetEnhancedReport: %v", err)
	}
	if got.ReportType != "comprehensive" {
		t.Errorf("expected comprehensive, got %s", got.ReportType)
	}

	list, err := repo.ListEnhancedReports(ctx)
	if err != nil {
		t.Fatalf("ListEnhancedReports: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1, got %d", len(list))
	}
}

// ---------------------------------------------------------------------------
// Thresholds (in-memory)
// ---------------------------------------------------------------------------

func TestThresholdDefaults(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	thresholds, err := repo.GetActiveThresholds(ctx)
	if err != nil {
		t.Fatalf("GetActiveThresholds: %v", err)
	}
	if len(thresholds) < 2 {
		t.Fatalf("expected at least 2 default thresholds, got %d", len(thresholds))
	}
}

func TestThresholdCRUD(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	th := &models.Threshold{
		MetricName:        "disk_usage",
		WarningThreshold:  80,
		CriticalThreshold: 90,
		Enabled:           true,
	}

	if err := repo.SaveThreshold(ctx, th); err != nil {
		t.Fatalf("SaveThreshold: %v", err)
	}

	got, err := repo.GetThreshold(ctx, "disk_usage")
	if err != nil {
		t.Fatalf("GetThreshold: %v", err)
	}
	if got.WarningThreshold != 80 {
		t.Errorf("expected 80, got %f", got.WarningThreshold)
	}

	if err := repo.DeleteThreshold(ctx, "disk_usage"); err != nil {
		t.Fatalf("DeleteThreshold: %v", err)
	}

	_, err = repo.GetThreshold(ctx, "disk_usage")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestThresholdViolation(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	violation := &models.ThresholdViolation{
		MetricName:     "cpu_usage",
		CurrentValue:   96.5,
		ThresholdValue: 95.0,
		Severity:       "critical",
		ViolationType:  "critical",
		Timestamp:      time.Now(),
	}

	if err := repo.SaveThresholdViolation(ctx, violation); err != nil {
		t.Fatalf("SaveThresholdViolation: %v", err)
	}

	violations, err := repo.GetThresholdViolations(ctx, repository.TimeRange{
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now().Add(1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("GetThresholdViolations: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].CurrentValue != 96.5 {
		t.Errorf("expected 96.5, got %f", violations[0].CurrentValue)
	}
}

// ---------------------------------------------------------------------------
// Alerts
// ---------------------------------------------------------------------------

func TestAlertCRUD(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	alert := &models.Alert{
		ID:          "alert-1",
		Type:        "threshold",
		Severity:    "critical",
		Message:     "CPU usage exceeded threshold",
		MetricName:  "cpu_usage",
		MetricValue: 96.5,
		Timestamp:   time.Now(),
	}

	if err := repo.CreateAlert(ctx, alert); err != nil {
		t.Fatalf("CreateAlert: %v", err)
	}

	got, err := repo.GetAlert(ctx, "alert-1")
	if err != nil {
		t.Fatalf("GetAlert: %v", err)
	}
	if got.Severity != "critical" {
		t.Errorf("expected critical, got %s", got.Severity)
	}

	// Update
	alert.Message = "updated message"
	if err := repo.UpdateAlert(ctx, alert); err != nil {
		t.Fatalf("UpdateAlert: %v", err)
	}

	got, _ = repo.GetAlert(ctx, "alert-1")
	if got.Message != "updated message" {
		t.Errorf("expected 'updated message', got %s", got.Message)
	}
}

func TestAlertAcknowledgeAndResolve(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.CreateAlert(ctx, &models.Alert{
		ID:        "alert-1",
		Type:      "threshold",
		Severity:  "warning",
		Message:   "test",
		Timestamp: time.Now(),
	}); err != nil {
		t.Fatalf("CreateAlert: %v", err)
	}

	// Acknowledge
	if err := repo.AcknowledgeAlert(ctx, "alert-1", "admin"); err != nil {
		t.Fatalf("AcknowledgeAlert: %v", err)
	}

	got, _ := repo.GetAlert(ctx, "alert-1")
	if got.AckedAt == nil {
		t.Fatal("expected AckedAt to be set")
	}
	if got.AckedBy != "admin" {
		t.Errorf("expected admin, got %s", got.AckedBy)
	}

	// Resolve
	if err := repo.ResolveAlert(ctx, "alert-1"); err != nil {
		t.Fatalf("ResolveAlert: %v", err)
	}

	got, _ = repo.GetAlert(ctx, "alert-1")
	if got.ResolvedAt == nil {
		t.Fatal("expected ResolvedAt to be set")
	}
}

func TestGetActiveAlerts(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.CreateAlert(ctx, &models.Alert{ID: "alert-1", Type: "threshold", Severity: "warning", Message: "t1", Timestamp: time.Now()}); err != nil {
		t.Fatalf("CreateAlert(alert-1): %v", err)
	}
	if err := repo.CreateAlert(ctx, &models.Alert{ID: "alert-2", Type: "threshold", Severity: "critical", Message: "t2", Timestamp: time.Now()}); err != nil {
		t.Fatalf("CreateAlert(alert-2): %v", err)
	}

	if err := repo.ResolveAlert(ctx, "alert-1"); err != nil {
		t.Fatalf("ResolveAlert: %v", err)
	}

	active, err := repo.GetActiveAlerts(ctx)
	if err != nil {
		t.Fatalf("GetActiveAlerts: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active alert, got %d", len(active))
	}
	if active[0].ID != "alert-2" {
		t.Errorf("expected alert-2, got %s", active[0].ID)
	}
}

func TestListAlerts(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if err := repo.CreateAlert(ctx, &models.Alert{ID: "a1", Type: "threshold", Severity: "warning", Message: "t1", Timestamp: time.Now()}); err != nil {
		t.Fatalf("CreateAlert(a1): %v", err)
	}
	if err := repo.CreateAlert(ctx, &models.Alert{ID: "a2", Type: "anomaly", Severity: "critical", Message: "t2", Timestamp: time.Now()}); err != nil {
		t.Fatalf("CreateAlert(a2): %v", err)
	}

	byType, err := repo.ListAlerts(ctx, repository.AlertFilter{Type: "threshold"})
	if err != nil {
		t.Fatalf("ListAlerts by type: %v", err)
	}
	if len(byType) != 1 {
		t.Errorf("expected 1, got %d", len(byType))
	}

	bySev, err := repo.ListAlerts(ctx, repository.AlertFilter{Severity: "critical"})
	if err != nil {
		t.Fatalf("ListAlerts by severity: %v", err)
	}
	if len(bySev) != 1 {
		t.Errorf("expected 1, got %d", len(bySev))
	}
}

func TestAlertNotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetAlert(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent alert")
	}

	err = repo.AcknowledgeAlert(ctx, "nonexistent", "admin")
	if err == nil {
		t.Fatal("expected error for nonexistent alert acknowledge")
	}

	err = repo.ResolveAlert(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent alert resolve")
	}
}
