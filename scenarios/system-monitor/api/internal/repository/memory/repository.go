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

		repository.HydrateMetricObservation(response, entry.CycleID, entry.Timestamp, entry.CollectorName, entry.Values)
	}

	// Convert map to slice
	var results []*models.MetricsResponse
	for _, response := range metricsMap {
		results = append(results, response)
	}

	// Sort by timestamp
	sort.Slice(results, func(i, j int) bool {
		if results[i].Timestamp.Equal(results[j].Timestamp) {
			return results[i].CycleID < results[j].CycleID
		}
		return results[i].Timestamp.Before(results[j].Timestamp)
	})
	if len(filter.SampleBuckets) > 0 {
		buckets := filter.SampleBuckets
		if len(buckets) > filter.EffectiveLimit() {
			buckets = buckets[len(buckets)-filter.EffectiveLimit():]
		}
		selected := make([]*models.MetricsResponse, 0, len(buckets))
		for _, bucket := range buckets {
			index := sort.Search(len(results), func(i int) bool { return results[i].Timestamp.After(bucket.EndTime) }) - 1
			if index >= 0 && results[index].Timestamp.After(bucket.StartTime) {
				selected = append(selected, results[index])
			}
		}
		results = selected
	}

	// Keep the newest cycles; the limit is always bounded.
	if limit := filter.EffectiveLimit(); len(results) > limit {
		results = results[len(results)-limit:]
	}

	return results, nil
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
		repository.HydrateMetricObservation(response, entry.CycleID, entry.Timestamp, collector, entry.Values)
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
