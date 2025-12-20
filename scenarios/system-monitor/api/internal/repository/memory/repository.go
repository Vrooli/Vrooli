package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"system-monitor-api/internal/models"
	"system-monitor-api/internal/repository"
)

// MemoryRepository provides an in-memory implementation of all repositories
type MemoryRepository struct {
	mu              sync.RWMutex
	metrics         []metricEntry
	investigations  map[string]*models.Investigation
	reports         map[string]*models.Report
	enhancedReports map[string]*models.EnhancedSystemReport
	alerts          map[string]*models.Alert
	anomalies       map[string]*models.Anomaly
	thresholds      map[string]*models.Threshold
	violations      []models.ThresholdViolation
}

type metricEntry struct {
	CollectorName string
	Timestamp     time.Time
	Values        map[string]interface{}
}

// NewRepository creates a new in-memory repository.
func NewRepository() *MemoryRepository {
	return &MemoryRepository{
		metrics:         make([]metricEntry, 0),
		investigations:  make(map[string]*models.Investigation),
		reports:         make(map[string]*models.Report),
		enhancedReports: make(map[string]*models.EnhancedSystemReport),
		alerts:          make(map[string]*models.Alert),
		anomalies:       make(map[string]*models.Anomaly),
		thresholds:      make(map[string]*models.Threshold),
		violations:      make([]models.ThresholdViolation, 0),
	}
}

// MetricsRepository implementation

func (r *MemoryRepository) SaveMetrics(ctx context.Context, collectorName string, metrics map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics = append(r.metrics, metricEntry{
		CollectorName: collectorName,
		Timestamp:     time.Now(),
		Values:        metrics,
	})

	// Keep only last 1000 entries
	if len(r.metrics) > 1000 {
		r.metrics = r.metrics[len(r.metrics)-1000:]
	}

	return nil
}

func (r *MemoryRepository) GetMetrics(ctx context.Context, filter repository.MetricsFilter) ([]*models.MetricsResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Group metrics by timestamp to combine collectors
	metricsMap := make(map[time.Time]*models.MetricsResponse)

	for _, entry := range r.metrics {
		if filter.CollectorName != "" && entry.CollectorName != filter.CollectorName {
			continue
		}

		// Check if within time range
		if !filter.TimeRange.StartTime.IsZero() && entry.Timestamp.Before(filter.TimeRange.StartTime) {
			continue
		}
		if !filter.TimeRange.EndTime.IsZero() && entry.Timestamp.After(filter.TimeRange.EndTime) {
			continue
		}

		// Get or create response for this timestamp
		response, exists := metricsMap[entry.Timestamp]
		if !exists {
			response = &models.MetricsResponse{
				Timestamp: entry.Timestamp,
			}
			metricsMap[entry.Timestamp] = response
		}

		// CPU metrics - check for the correct field name based on collector
		if entry.CollectorName == "cpu" {
			if cpu, ok := entry.Values["usage_percent"].(float64); ok {
				response.CPUUsage = cpu
			}
		}

		// Memory metrics
		if entry.CollectorName == "memory" {
			if mem, ok := entry.Values["usage_percent"].(float64); ok {
				response.MemoryUsage = mem
			}
		}

		// Network metrics
		if entry.CollectorName == "network" {
			if tcp, ok := entry.Values["tcp_connections"].(int); ok {
				response.TCPConnections = tcp
			}
		}

		if entry.CollectorName == "gpu" {
			if usage, ok := entry.Values["total_usage_percent"].(float64); ok {
				value := usage
				response.GPUUsage = &value
			}
		}
	}

	// Convert map to slice
	var results []*models.MetricsResponse
	for _, response := range metricsMap {
		results = append(results, response)
	}

	// Sort by timestamp
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.Before(results[j].Timestamp)
	})

	// Apply limit if specified
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[len(results)-filter.Limit:]
	}

	return results, nil
}

func (r *MemoryRepository) GetLatestMetrics(ctx context.Context) (*models.MetricsResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.metrics) == 0 {
		return nil, fmt.Errorf("no metrics available")
	}

	// Combine latest metrics from different collectors
	response := &models.MetricsResponse{
		Timestamp: time.Now(),
	}

	// Get latest CPU metrics
	for i := len(r.metrics) - 1; i >= 0; i-- {
		entry := r.metrics[i]
		if entry.CollectorName == "cpu" {
			if cpu, ok := entry.Values["usage_percent"].(float64); ok {
				response.CPUUsage = cpu
				break
			}
		}
	}

	// Get latest Memory metrics
	for i := len(r.metrics) - 1; i >= 0; i-- {
		entry := r.metrics[i]
		if entry.CollectorName == "memory" {
			if mem, ok := entry.Values["usage_percent"].(float64); ok {
				response.MemoryUsage = mem
				break
			}
		}
	}

	// Get latest Network metrics
	for i := len(r.metrics) - 1; i >= 0; i-- {
		entry := r.metrics[i]
		if entry.CollectorName == "network" {
			if tcp, ok := entry.Values["tcp_connections"].(int); ok {
				response.TCPConnections = tcp
				break
			}
		}
	}

	// Get latest GPU metrics
	for i := len(r.metrics) - 1; i >= 0; i-- {
		entry := r.metrics[i]
		if entry.CollectorName == "gpu" {
			if usage, ok := entry.Values["total_usage_percent"].(float64); ok {
				value := usage
				response.GPUUsage = &value
				break
			}
		}
	}

	return response, nil
}

func (r *MemoryRepository) GetDetailedMetrics(ctx context.Context, timeRange repository.TimeRange) (*models.DetailedMetrics, error) {
	// Return mock detailed metrics
	return &models.DetailedMetrics{
		Timestamp: time.Now(),
	}, nil
}

func (r *MemoryRepository) GetHistoricalMetrics(ctx context.Context, metricName string, timeRange repository.TimeRange) ([]repository.MetricDataPoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var points []repository.MetricDataPoint
	for _, entry := range r.metrics {
		if entry.Timestamp.Before(timeRange.StartTime) || entry.Timestamp.After(timeRange.EndTime) {
			continue
		}

		if val, ok := entry.Values[metricName].(float64); ok {
			points = append(points, repository.MetricDataPoint{
				Timestamp: entry.Timestamp,
				Value:     val,
			})
		}
	}

	return points, nil
}

func (r *MemoryRepository) GetAggregatedMetrics(ctx context.Context, aggregation repository.AggregationQuery) (map[string]interface{}, error) {
	return map[string]interface{}{
		"average": 50.0,
		"max":     95.0,
		"min":     10.0,
		"count":   100,
	}, nil
}

// GetEarliestMetricTime returns the timestamp of the earliest stored metric
func (r *MemoryRepository) GetEarliestMetricTime(ctx context.Context) (time.Time, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.metrics) == 0 {
		return time.Time{}, fmt.Errorf("no metrics available")
	}

	return r.metrics[0].Timestamp, nil
}

// InvestigationRepository implementation

func (r *MemoryRepository) CreateInvestigation(ctx context.Context, investigation *models.Investigation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.investigations[investigation.ID] = investigation
	return nil
}

func (r *MemoryRepository) GetInvestigation(ctx context.Context, id string) (*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if inv, exists := r.investigations[id]; exists {
		return inv, nil
	}
	return nil, fmt.Errorf("investigation not found: %s", id)
}

func (r *MemoryRepository) UpdateInvestigation(ctx context.Context, investigation *models.Investigation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.investigations[investigation.ID] = investigation
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
		results = append(results, inv)
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
			latest = inv
			latestTime = inv.StartTime
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no investigations found")
	}

	return latest, nil
}

func (r *MemoryRepository) SaveInvestigationStep(ctx context.Context, investigationID string, step *models.InvestigationStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if inv, exists := r.investigations[investigationID]; exists {
		inv.Steps = append(inv.Steps, *step)
		return nil
	}

	return fmt.Errorf("investigation not found: %s", investigationID)
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
	return nil, fmt.Errorf("report not found: %s", id)
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

	return nil, fmt.Errorf("detailed report not found")
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

	return nil, fmt.Errorf("enhanced report not found")
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
		results = []*models.Threshold{
			{
				MetricName:        "cpu_usage",
				Min:               0,
				Max:               100,
				WarningThreshold:  80,
				CriticalThreshold: 95,
				CheckInterval:     60,
				Enabled:           true,
			},
			{
				MetricName:        "memory_usage",
				Min:               0,
				Max:               100,
				WarningThreshold:  85,
				CriticalThreshold: 95,
				CheckInterval:     60,
				Enabled:           true,
			},
		}
	}

	return results, nil
}

func (r *MemoryRepository) GetThreshold(ctx context.Context, metricName string) (*models.Threshold, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if threshold, exists := r.thresholds[metricName]; exists {
		return threshold, nil
	}
	return nil, fmt.Errorf("threshold not found: %s", metricName)
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
	return nil, fmt.Errorf("alert not found: %s", id)
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
		results = append(results, alert)
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

	return fmt.Errorf("alert not found: %s", id)
}

func (r *MemoryRepository) ResolveAlert(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if alert, exists := r.alerts[id]; exists {
		now := time.Now()
		alert.ResolvedAt = &now
		return nil
	}

	return fmt.Errorf("alert not found: %s", id)
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
