package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
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
	processSamples  []repository.ProcessSample
	processRollups  []processRollup
}

// processRollup is the in-memory analog of the SQLite process_sample_rollups
// row (one per owner+comm+minute).
type processRollup struct {
	Minute                  time.Time
	Owner                   string
	Comm                    string
	AvgCPUPct               float64
	MaxCPUPct               float64
	CPUSeconds              float64
	AvgRSSKB                int64
	MaxRSSKB                int64
	AvgMajorFaultsPerSecond float64
	MaxMajorFaultsPerSecond float64
	SampleCount             int64
}

type metricEntry struct {
	CycleID       string
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

func (r *MemoryRepository) SaveMetricCycle(ctx context.Context, cycleID string, observedAt time.Time, observations []repository.MetricObservation) error {
	if cycleID == "" {
		return fmt.Errorf("cycle id is required")
	}
	if observedAt.IsZero() {
		return fmt.Errorf("observation time is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, entry := range r.metrics {
		if entry.CycleID == cycleID {
			return fmt.Errorf("metric cycle %q already exists", cycleID)
		}
	}

	for _, observation := range observations {
		values := make(map[string]interface{}, len(observation.Values))
		for key, value := range observation.Values {
			values[key] = value
		}
		r.metrics = append(r.metrics, metricEntry{CycleID: cycleID, CollectorName: observation.CollectorName, Timestamp: observedAt.UTC(), Values: values})
	}

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
	metricsMap := make(map[string]*models.MetricsResponse)

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
		key := entry.CycleID
		if key == "" {
			key = entry.Timestamp.UTC().Format(time.RFC3339Nano)
		}
		response, exists := metricsMap[key]
		if !exists {
			response = &models.MetricsResponse{
				CycleID:   entry.CycleID,
				Timestamp: entry.Timestamp,
			}
			metricsMap[key] = response
		}

		// CPU metrics - check for the correct field name based on collector
		if entry.CollectorName == "cpu" {
			if memoryMetricMeasured(entry.Values) {
				if cpu, ok := entry.Values["usage_percent"].(float64); ok {
					response.CPUUsage = cpu
				}
			}
			response.CPUState = memoryMetricState(entry.CycleID, entry.Timestamp, entry.CollectorName, entry.Values, "usage_percent")
			response.CPUContextSwitchesPerSecond = cpuMetricState(entry, "context_switches_per_second")
			response.CPUInterruptsPerSecond = cpuMetricState(entry, "interrupts_per_second")
			response.CPUNormalizedLoad1 = cpuMetricState(entry, "normalized_load_1")
			response.CPUNormalizedLoad5 = cpuMetricState(entry, "normalized_load_5")
			response.CPURunQueueDepth = cpuMetricState(entry, "run_queue_depth")
			response.CPUCoreImbalanceIndex = cpuMetricState(entry, "core_imbalance_index")
			response.CPUModeIowait = cpuModeMetricState(entry, "iowait")
			response.CPUModeSteal = cpuModeMetricState(entry, "steal")
		}

		// Memory metrics
		if entry.CollectorName == "memory" {
			if memoryMetricMeasured(entry.Values) {
				if mem, ok := entry.Values["usage_percent"].(float64); ok {
					response.MemoryUsage = mem
				}
			}
			response.MemoryState = memoryMetricState(entry.CycleID, entry.Timestamp, entry.CollectorName, entry.Values, "usage_percent")
		}

		// Network metrics
		if entry.CollectorName == "network" {
			if memoryMetricMeasured(entry.Values) {
				if tcp, ok := entry.Values["tcp_connections"].(int); ok {
					response.TCPConnections = tcp
				}
				if tcp, ok := entry.Values["tcp_connections"].(float64); ok {
					response.TCPConnections = int(tcp)
				}
			}
			response.ConnectionsState = memoryMetricState(entry.CycleID, entry.Timestamp, entry.CollectorName, entry.Values, "tcp_connections")
		}

		if entry.CollectorName == "gpu" {
			if memoryMetricMeasured(entry.Values) {
				if usage, ok := entry.Values["total_usage_percent"].(float64); ok {
					value := usage
					response.GPUUsage = &value
				}
			}
			response.GPUState = memoryMetricState(entry.CycleID, entry.Timestamp, entry.CollectorName, entry.Values, "total_usage_percent")
		}
		if entry.CollectorName == "pressure" {
			response.CPUStallSomeAvg10 = pressureMetricStateMemory(entry, "cpu_psi_some_avg10", "cpu_psi_status", "cpu_psi_reason")
			response.CPUStallFullAvg10 = pressureMetricStateMemory(entry, "cpu_psi_full_avg10", "cpu_psi_status", "cpu_psi_reason")
			response.SwapTrafficState = pressureMetricStateMemory(entry, "swap_traffic_pages_per_second", "swap_traffic_rate_status", "swap_traffic_rate_reason")
			response.MajorFaultsState = pressureMetricStateMemory(entry, "pgmajfault_per_second", "pgmajfault_rate_status", "pgmajfault_rate_reason")
			response.FragmentationIndexState = pressureMetricStateMemory(entry, "fragmentation_max_free_order", "fragmentation_status", "fragmentation_reason")
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

func cpuMetricState(entry metricEntry, key string) models.MetricState {
	state := models.MetricState{Status: "not_yet_sampled", Reason: "CPU signal has not been sampled", Provenance: "system-monitor/cpu", CycleID: entry.CycleID, ObservedAt: entry.Timestamp}
	if status, ok := entry.Values[key+"_status"].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := entry.Values[key+"_reason"].(string); ok && reason != "" {
		state.Reason = reason
	}
	if provenance, ok := entry.Values[key+"_provenance"].(string); ok && provenance != "" {
		state.Provenance = provenance
	}
	if value, ok := entry.Values[key].(float64); ok {
		state.Status, state.Value, state.Reason = "measured", value, ""
	}
	return state
}

func cpuModeMetricState(entry metricEntry, mode string) models.MetricState {
	state := models.MetricState{Status: "not_yet_sampled", Reason: "CPU mode breakdown has not been sampled", Provenance: "system-monitor/cpu", Units: "percent", CycleID: entry.CycleID, ObservedAt: entry.Timestamp}
	if raw, ok := entry.Values["mode_breakdown"].(map[string]float64); ok {
		if value, exists := raw[mode]; exists {
			state.Status, state.Value, state.Reason = "measured", value, ""
		}
	}
	return state
}

func pressureMetricStateMemory(entry metricEntry, valueKey, statusKey, reasonKey string) models.MetricState {
	state := models.MetricState{Status: "not_yet_sampled", Reason: "rate has not been sampled", Provenance: "system-monitor/pressure", CycleID: entry.CycleID, ObservedAt: entry.Timestamp}
	if _, hasSignalStatus := entry.Values[valueKey+"_status"]; hasSignalStatus {
		statusKey = valueKey + "_status"
		reasonKey = valueKey + "_reason"
	}
	if status, ok := entry.Values[statusKey].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := entry.Values[reasonKey].(string); ok && reason != "" {
		state.Reason = reason
	}
	if value, ok := entry.Values[valueKey].(float64); ok {
		state.Status, state.Value, state.Reason = "measured", value, ""
	}
	return state
}

func memoryMetricMeasured(values map[string]interface{}) bool {
	status, explicit := values["status"].(string)
	return !explicit || status == "measured"
}

func memoryMetricState(cycleID string, observedAt time.Time, collector string, values map[string]interface{}, key string) models.MetricState {
	state := models.MetricState{Status: "failed", Reason: "collector did not return a measurement", Provenance: "system-monitor/" + collector, Units: metricUnitsForCollector(collector), CycleID: cycleID, ObservedAt: observedAt}
	if status, ok := values["status"].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := values["reason"].(string); ok && reason != "" {
		state.Reason = reason
	}
	if source, ok := values["source"].(string); ok && source != "" {
		state.Provenance = source
	}
	if memoryMetricMeasured(values) {
		if key == "usage" {
			if usage, ok := values[key].(map[string]interface{}); ok {
				if percent, ok := usage["percent"].(float64); ok {
					state.Status, state.Reason, state.Value = "measured", "", percent
				}
			}
			return state
		}
		switch values[key].(type) {
		case float64:
			state.Status, state.Reason, state.Value = "measured", "", values[key].(float64)
		case int:
			state.Status, state.Reason, state.Value = "measured", "", float64(values[key].(int))
		case int64:
			state.Status, state.Reason, state.Value = "measured", "", float64(values[key].(int64))
		}
	}
	return state
}

func metricUnitsForCollector(collector string) string {
	if collector == "network" {
		return "count"
	}
	return "percent"
}

func (r *MemoryRepository) GetLatestMetrics(ctx context.Context) (*models.MetricsResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.metrics) == 0 {
		return nil, fmt.Errorf("no metrics available")
	}

	latest := make(map[string]metricEntry)
	for i := len(r.metrics) - 1; i >= 0; i-- {
		entry := r.metrics[i]
		if _, exists := latest[entry.CollectorName]; !exists {
			latest[entry.CollectorName] = entry
		}
	}
	if len(latest) == 0 {
		return nil, fmt.Errorf("no metrics available")
	}
	response := &models.MetricsResponse{}
	for collector, entry := range latest {
		if entry.Timestamp.After(response.Timestamp) {
			response.CycleID, response.Timestamp = entry.CycleID, entry.Timestamp
		}
		switch collector {
		case "cpu":
			response.CPUState = memoryMetricState(entry.CycleID, entry.Timestamp, collector, entry.Values, "usage_percent")
			response.CPUUsage, _ = entry.Values["usage_percent"].(float64)
		case "memory":
			response.MemoryState = memoryMetricState(entry.CycleID, entry.Timestamp, collector, entry.Values, "usage_percent")
			response.MemoryUsage, _ = entry.Values["usage_percent"].(float64)
		case "network":
			response.ConnectionsState = memoryMetricState(entry.CycleID, entry.Timestamp, collector, entry.Values, "tcp_connections")
			response.TCPConnections = metricInt(entry.Values["tcp_connections"])
		case "gpu":
			response.GPUState = memoryMetricState(entry.CycleID, entry.Timestamp, collector, entry.Values, "total_usage_percent")
			if usage, ok := entry.Values["total_usage_percent"].(float64); ok {
				response.GPUUsage = &usage
			}
		case "disk":
			response.DiskState = memoryMetricState(entry.CycleID, entry.Timestamp, collector, entry.Values, "usage")
		}
	}
	return response, nil
}

func metricInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
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
