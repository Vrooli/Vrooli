package memory

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

func (r *MemoryRepository) CreateInvestigation(ctx context.Context, investigation *models.Investigation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.investigations[investigation.ID] = cloneInvestigation(investigation)
	return nil
}

func (r *MemoryRepository) GetInvestigation(ctx context.Context, id string) (*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if inv, exists := r.investigations[id]; exists {
		return cloneInvestigation(inv), nil
	}
	return nil, fmt.Errorf("investigation %s: %w", id, repository.ErrNotFound)
}

func (r *MemoryRepository) UpdateInvestigation(ctx context.Context, investigation *models.Investigation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.investigations[investigation.ID] = cloneInvestigation(investigation)
	return nil
}

func (r *MemoryRepository) ListInvestigations(ctx context.Context, filter repository.InvestigationFilter) ([]*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.Investigation
	for _, inv := range r.investigations {
		if filter.Status != "" && inv.Status != filter.Status {
			continue
		}
		results = append(results, cloneInvestigation(inv))
	}

	return results, nil
}

func (r *MemoryRepository) GetLatestInvestigation(ctx context.Context) (*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest *models.Investigation
	var latestTime time.Time

	for _, inv := range r.investigations {
		if inv.StartTime.After(latestTime) {
			latest = cloneInvestigation(inv)
			latestTime = inv.StartTime
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no investigations: %w", repository.ErrNotFound)
	}

	return latest, nil
}

func cloneInvestigation(investigation *models.Investigation) *models.Investigation {
	if investigation == nil {
		return nil
	}
	clone := *investigation
	if investigation.EndTime != nil {
		end := *investigation.EndTime
		clone.EndTime = &end
	}
	clone.Steps = append([]models.InvestigationStep(nil), investigation.Steps...)
	for i := range clone.Steps {
		if investigation.Steps[i].EndTime != nil {
			end := *investigation.Steps[i].EndTime
			clone.Steps[i].EndTime = &end
		}
	}
	if investigation.Details != nil {
		clone.Details = make(map[string]interface{}, len(investigation.Details))
		for key, value := range investigation.Details {
			clone.Details[key] = value
		}
	}
	return &clone
}

func (r *MemoryRepository) SaveInvestigationStep(ctx context.Context, investigationID string, step *models.InvestigationStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if inv, exists := r.investigations[investigationID]; exists {
		inv.Steps = append(inv.Steps, *step)
		return nil
	}

	return fmt.Errorf("investigation %s: %w", investigationID, repository.ErrNotFound)
}

// ReportRepository implementation

func (r *MemoryRepository) CreateReport(ctx context.Context, report *models.Report) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reports[report.ID] = report
	return nil
}

func (r *MemoryRepository) GetReport(ctx context.Context, id string) (*models.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if report, exists := r.reports[id]; exists {
		return report, nil
	}
	return nil, fmt.Errorf("report %s: %w", id, repository.ErrNotFound)
}

func (r *MemoryRepository) ListReports(ctx context.Context, filter repository.ReportFilter) ([]*models.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.Report
	for _, report := range r.reports {
		if filter.Type != "" && report.Type != filter.Type {
			continue
		}
		results = append(results, report)
	}

	return results, nil
}

func (r *MemoryRepository) SaveDetailedReport(ctx context.Context, report *models.DetailedSystemReport) error {
	// Convert to basic report for storage
	basicReport := &models.Report{
		ID:          report.ReportID,
		Type:        report.ReportType,
		GeneratedAt: report.GeneratedAt,
		TimeRange:   report.TimeRange,
		Data:        map[string]interface{}{"detailed": report},
	}

	return r.CreateReport(ctx, basicReport)
}

func (r *MemoryRepository) GetDetailedReport(ctx context.Context, id string) (*models.DetailedSystemReport, error) {
	report, err := r.GetReport(ctx, id)
	if err != nil {
		return nil, err
	}

	if detailed, ok := report.Data["detailed"].(*models.DetailedSystemReport); ok {
		return detailed, nil
	}

	return nil, fmt.Errorf("detailed report: %w", repository.ErrNotFound)
}

func (r *MemoryRepository) SaveEnhancedReport(ctx context.Context, report *models.EnhancedSystemReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.enhancedReports[report.ReportID] = report
	return nil
}

func (r *MemoryRepository) GetEnhancedReport(ctx context.Context, id string) (*models.EnhancedSystemReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if report, exists := r.enhancedReports[id]; exists {
		return report, nil
	}

	return nil, fmt.Errorf("enhanced report: %w", repository.ErrNotFound)
}

func (r *MemoryRepository) ListEnhancedReports(ctx context.Context) ([]*models.EnhancedSystemReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var reports []*models.EnhancedSystemReport
	for _, report := range r.enhancedReports {
		reports = append(reports, report)
	}

	return reports, nil
}

// ThresholdRepository implementation

func (r *MemoryRepository) GetActiveThresholds(ctx context.Context) ([]*models.Threshold, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.Threshold
	for _, threshold := range r.thresholds {
		if threshold.Enabled {
			results = append(results, threshold)
		}
	}

	// Return default thresholds if none configured
	if len(results) == 0 {
		results = repository.DefaultThresholds()
	}

	return results, nil
}

func (r *MemoryRepository) GetThreshold(ctx context.Context, metricName string) (*models.Threshold, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if threshold, exists := r.thresholds[metricName]; exists {
		return threshold, nil
	}
	return nil, fmt.Errorf("threshold %s: %w", metricName, repository.ErrNotFound)
}

func (r *MemoryRepository) SaveThreshold(ctx context.Context, threshold *models.Threshold) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.thresholds[threshold.MetricName] = threshold
	return nil
}

func (r *MemoryRepository) DeleteThreshold(ctx context.Context, metricName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.thresholds, metricName)
	return nil
}

func (r *MemoryRepository) SaveThresholdViolation(ctx context.Context, violation *models.ThresholdViolation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.violations = append(r.violations, *violation)

	// Keep only last 100 violations
	if len(r.violations) > 100 {
		r.violations = r.violations[len(r.violations)-100:]
	}

	return nil
}

func (r *MemoryRepository) GetThresholdViolations(ctx context.Context, timeRange repository.TimeRange) ([]*models.ThresholdViolation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.ThresholdViolation
	for _, violation := range r.violations {
		if violation.Timestamp.After(timeRange.StartTime) && violation.Timestamp.Before(timeRange.EndTime) {
			v := violation // Create copy
			results = append(results, &v)
		}
	}

	return results, nil
}

// AlertRepository implementation

func (r *MemoryRepository) CreateAlert(ctx context.Context, alert *models.Alert) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.alerts[alert.ID] = alert
	return nil
}

func (r *MemoryRepository) GetAlert(ctx context.Context, id string) (*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if alert, exists := r.alerts[id]; exists {
		return alert, nil
	}
	return nil, fmt.Errorf("alert %s: %w", id, repository.ErrNotFound)
}

func (r *MemoryRepository) UpdateAlert(ctx context.Context, alert *models.Alert) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.alerts[alert.ID] = alert
	return nil
}

func (r *MemoryRepository) ListAlerts(ctx context.Context, filter repository.AlertFilter) ([]*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.Alert
	for _, alert := range r.alerts {
		if filter.Type != "" && alert.Type != filter.Type {
			continue
		}
		if filter.Severity != "" && alert.Severity != filter.Severity {
			continue
		}
		if filter.MetricName != "" && alert.MetricName != filter.MetricName {
			continue
		}
		switch filter.Status {
		case "active":
			if alert.ResolvedAt != nil {
				continue
			}
		case "resolved":
			if alert.ResolvedAt == nil {
				continue
			}
		case "acknowledged":
			if alert.AckedAt == nil {
				continue
			}
		}
		results = append(results, alert)
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].Timestamp.After(results[j].Timestamp) })
	if filter.Offset > 0 {
		if filter.Offset >= len(results) {
			return nil, nil
		}
		results = results[filter.Offset:]
	}
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}
	return results, nil
}

func (r *MemoryRepository) AcknowledgeAlert(ctx context.Context, id string, ackedBy string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if alert, exists := r.alerts[id]; exists {
		now := time.Now()
		alert.AckedAt = &now
		alert.AckedBy = ackedBy
		return nil
	}

	return fmt.Errorf("alert %s: %w", id, repository.ErrNotFound)
}

func (r *MemoryRepository) ResolveAlert(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if alert, exists := r.alerts[id]; exists {
		now := time.Now()
		alert.ResolvedAt = &now
		return nil
	}

	return fmt.Errorf("alert %s: %w", id, repository.ErrNotFound)
}

func (r *MemoryRepository) GetActiveAlerts(ctx context.Context) ([]*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.Alert
	for _, alert := range r.alerts {
		if alert.ResolvedAt == nil {
			results = append(results, alert)
		}
	}

	return results, nil
}
