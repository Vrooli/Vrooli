package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

func (s *MonitorService) GetCurrentMetrics(ctx context.Context) (*models.MetricsResponse, error) {
	// Get latest metrics from repository
	metrics, err := s.repo.GetLatestMetrics(ctx)
	if err != nil {
		// Fallback to real-time collection
		return s.GetCurrentMetricsFresh(ctx)
	}
	s.markStale(metrics)
	return metrics, nil
}

func (s *MonitorService) markStale(metrics *models.MetricsResponse) {
	if metrics == nil {
		return
	}
	cutoff := s.clock.Now().Add(-2 * s.EffectiveCollectionInterval())
	stale := 0
	for _, state := range []*models.MetricState{&metrics.CPUState, &metrics.MemoryState, &metrics.ConnectionsState, &metrics.GPUState, &metrics.DiskState} {
		if state.Status == "measured" && !state.ObservedAt.IsZero() && state.ObservedAt.Before(cutoff) {
			state.Status = "stale"
			state.Reason = fmt.Sprintf("last observation is older than %s", 2*s.EffectiveCollectionInterval())
			stale++
		}
	}
	s.mu.Lock()
	s.lastStaleMetricCount = stale
	s.mu.Unlock()
}

// GetCurrentMetricsFresh performs on-demand metric collection using existing collectors.
func (s *MonitorService) GetCurrentMetricsFresh(ctx context.Context) (*models.MetricsResponse, error) {
	_ = ctx // retained for interface compatibility; sampling is scheduler-owned
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.latestSnapshot != nil {
		copy := *s.latestSnapshot
		return &copy, nil
	}
	now := s.clock.Now()
	return &models.MetricsResponse{
		Timestamp:        now,
		CPUState:         notYetSampledState("CPU has not been sampled yet", "percent", now),
		MemoryState:      notYetSampledState("memory has not been sampled yet", "percent", now),
		ConnectionsState: notYetSampledState("network has not been sampled yet", "count", now),
		GPUState:         notYetSampledState("GPU has not been sampled yet", "percent", now),
		DiskState:        notYetSampledState("disk has not been sampled yet", "percent", now),
	}, nil
}

func (s *MonitorService) collectFromRegistry(ctx context.Context, name string) (*collectors.MetricData, error) {
	collector, ok := s.collectors.Get(name)
	if !ok || !collector.IsEnabled() {
		return nil, nil
	}
	return collector.Collect(ctx)
}

// GetPressureSnapshot performs a bounded fresh read of the pressure collector.
// Unsupported/degraded hosts return an explicit unavailable snapshot instead of
// an error so callers can make a safe fail-closed recovery decision.
func (s *MonitorService) GetPressureSnapshot(ctx context.Context) (*models.PressureSnapshot, error) {
	data, err := s.collectFromRegistry(ctx, "pressure")
	if err != nil {
		return nil, err
	}
	snapshot := &models.PressureSnapshot{Timestamp: s.clock.Now(), Memory: map[string]map[string]float64{}}
	if data == nil {
		snapshot.DegradedReason = "pressure collector is disabled"
		return snapshot, nil
	}
	snapshot.Timestamp = data.Timestamp
	if available, ok := data.Values["available"].(bool); ok {
		snapshot.Available = available
	}
	if reason, ok := data.Values["degraded_reason"].(string); ok {
		snapshot.DegradedReason = reason
	}
	if memory, ok := data.Values["memory"].(map[string]map[string]float64); ok {
		snapshot.Memory = memory
	}
	snapshot.OOMKillCount = pressureInt64(data.Values["oom_kill_count"])
	snapshot.OOMCount = pressureInt64(data.Values["oom_count"])
	return snapshot, nil
}

// GetGPUHistory returns the persisted low-cardinality GPU summary timeline.
func (s *MonitorService) GetGPUHistory(ctx context.Context, window time.Duration) (*models.GPUHistory, error) {
	if window <= 0 {
		window = time.Hour
	}
	end := s.clock.Now()
	start := end.Add(-window)
	utilization, err := s.repo.GetHistoricalMetrics(ctx, "total_usage_percent", repository.TimeRange{StartTime: start, EndTime: end})
	if err != nil {
		return nil, err
	}
	vram, err := s.repo.GetHistoricalMetrics(ctx, "used_memory_mb", repository.TimeRange{StartTime: start, EndTime: end})
	if err != nil {
		return nil, err
	}
	return &models.GPUHistory{Start: start, End: end, Utilization: gpuHistoryPoints(utilization), VRAMUsedMB: gpuHistoryPoints(vram)}, nil
}

// GetPressureHistory returns retained PSI and OOM counter evidence over a
// bounded time window. It is read-only and does not collect or scan processes.
func (s *MonitorService) GetPressureHistory(ctx context.Context, window time.Duration) (*models.PressureHistory, error) {
	if window <= 0 {
		window = time.Hour
	}
	end := s.clock.Now()
	start := end.Add(-window)
	rangeFilter := repository.TimeRange{StartTime: start, EndTime: end}
	some, err := s.repo.GetHistoricalMetrics(ctx, "memory_psi_some_avg10", rangeFilter)
	if err != nil {
		return nil, err
	}
	full, err := s.repo.GetHistoricalMetrics(ctx, "memory_psi_full_avg10", rangeFilter)
	if err != nil {
		return nil, err
	}
	oomKills, err := s.repo.GetHistoricalMetrics(ctx, "oom_kill_count", rangeFilter)
	if err != nil {
		return nil, err
	}
	return &models.PressureHistory{Start: start, End: end, SomeAvg10: gpuHistoryPoints(some), FullAvg10: gpuHistoryPoints(full), OOMKillCount: gpuHistoryPoints(oomKills)}, nil
}

func gpuHistoryPoints(points []repository.MetricDataPoint) []models.GPUHistoryPoint {
	out := make([]models.GPUHistoryPoint, 0, len(points))
	for _, point := range points {
		out = append(out, models.GPUHistoryPoint{Timestamp: point.Timestamp, Value: point.Value})
	}
	return out
}

func pressureInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// GetMetricsTimeline retrieves a windowed timeline of metric samples.
func (s *MonitorService) GetMetricsTimeline(ctx context.Context, windowSeconds, sampleIntervalSeconds int) (*models.MetricsTimelineResponse, error) {
	if windowSeconds <= 0 {
		windowSeconds = 120
	}
	if sampleIntervalSeconds <= 0 {
		sampleIntervalSeconds = 5
	}

	now := s.clock.Now()
	start := now.Add(-time.Duration(windowSeconds) * time.Second)
	interval := time.Duration(sampleIntervalSeconds) * time.Second
	// Return at most the newest 1000 buckets, and report the effective window
	// when a requested interval would exceed that bound.
	if int64(windowSeconds) > int64(sampleIntervalSeconds)*repository.DefaultMetricsLimit {
		maxWindow := time.Duration(int64(sampleIntervalSeconds)*repository.DefaultMetricsLimit) * time.Second
		start = now.Add(-maxWindow)
		windowSeconds = int(maxWindow / time.Second)
	}
	var buckets []repository.TimeRange
	for end := now; end.After(start); {
		begin := end.Add(-interval)
		if begin.Before(start) {
			begin = start
		}
		buckets = append(buckets, repository.TimeRange{StartTime: begin, EndTime: end})
		end = begin
	}
	for i, j := 0, len(buckets)-1; i < j; i, j = i+1, j-1 {
		buckets[i], buckets[j] = buckets[j], buckets[i]
	}

	results, err := s.repo.GetMetrics(ctx, repository.MetricsFilter{
		TimeRange: repository.TimeRange{
			StartTime: start,
			EndTime:   now,
		},
		SampleBuckets: buckets,
	})
	if err != nil {
		return nil, err
	}

	samples := make([]models.MetricTimelineSample, 0, len(results))
	for _, m := range results {
		samples = append(samples, models.MetricTimelineSample{
			CycleID:                     m.CycleID,
			Timestamp:                   m.Timestamp,
			CPUUsage:                    m.CPUUsage,
			MemoryUsage:                 m.MemoryUsage,
			TCPConnections:              m.TCPConnections,
			GPUUsage:                    m.GPUUsage,
			SwapUsage:                   m.SwapUsage,
			CPUState:                    m.CPUState,
			CPUContextSwitchesPerSecond: m.CPUContextSwitchesPerSecond,
			CPUInterruptsPerSecond:      m.CPUInterruptsPerSecond,
			CPUNormalizedLoad1:          m.CPUNormalizedLoad1,
			CPUNormalizedLoad5:          m.CPUNormalizedLoad5,
			CPURunQueueDepth:            m.CPURunQueueDepth,
			CPUStallSomeAvg10:           m.CPUStallSomeAvg10,
			CPUStallFullAvg10:           m.CPUStallFullAvg10,
			CPUCoreImbalanceIndex:       m.CPUCoreImbalanceIndex,
			CPUModeIowait:               m.CPUModeIowait,
			CPUModeSteal:                m.CPUModeSteal,
			MemoryState:                 m.MemoryState,
			KernelMemory:                m.KernelMemory,
			ConnectionsState:            m.ConnectionsState,
			GPUState:                    m.GPUState,
			SwapState:                   m.SwapState,
			SwapTrafficState:            m.SwapTrafficState,
			MajorFaultsState:            m.MajorFaultsState,
			FragmentationIndexState:     m.FragmentationIndexState,
			NetworkEstablishedRate:      m.NetworkEstablishedRate,
			NetworkTimeWaitRate:         m.NetworkTimeWaitRate,
			NetworkCloseWaitRate:        m.NetworkCloseWaitRate,
		})
	}

	return &models.MetricsTimelineResponse{
		WindowStart:           start,
		WindowEnd:             now,
		WindowSeconds:         windowSeconds,
		SampleIntervalSeconds: sampleIntervalSeconds,
		Samples:               samples,
	}, nil
}

// GetDetailedMetrics retrieves comprehensive system metrics
