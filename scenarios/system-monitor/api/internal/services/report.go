package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/models"
	"system-monitor-api/internal/repository"
)

// ReportService handles report generation and analysis
type ReportService struct {
	config *config.Config
	repo   repository.Repository
}

// NewReportService creates a new report service
func NewReportService(cfg *config.Config, repo repository.Repository) *ReportService {
	return &ReportService{
		config: cfg,
		repo:   repo,
	}
}

// GenerateReport creates comprehensive daily or weekly reports based on historical data
func (s *ReportService) GenerateReport(ctx context.Context, reportType string) (*models.EnhancedSystemReport, error) {
	reportID := fmt.Sprintf("report_%s_%d", reportType, time.Now().Unix())
	
	// Determine time range based on report type
	timeRange := s.calculateTimeRange(reportType)
	
	// Gather historical metrics from repository
	historicalMetrics, err := s.gatherHistoricalMetrics(ctx, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to gather historical metrics: %w", err)
	}
	
	// Get recent investigations for context
	investigations, err := s.repo.ListInvestigations(ctx, repository.InvestigationFilter{
		TimeRange: timeRange,
		Limit:     10,
	})
	if err != nil {
		// Non-fatal error, continue without investigations
		investigations = []*models.Investigation{}
	}
	
	// Get alerts from the time period
	alerts, err := s.repo.ListAlerts(ctx, repository.AlertFilter{
		TimeRange: timeRange,
		Status:    "active",
		Limit:     50,
	})
	if err != nil {
		// Non-fatal error, continue without alerts
		alerts = []*models.Alert{}
	}
	
	// Analyze the data
	executiveSummary := s.generateExecutiveSummary(historicalMetrics, investigations, alerts, reportType)
	performanceAnalysis := s.analyzePerformance(historicalMetrics, timeRange)
	trends := s.analyzeTrends(historicalMetrics)
	recommendations := s.generateRecommendations(performanceAnalysis, trends, alerts)
	highlights := s.generateHighlights(historicalMetrics, investigations, alerts, reportType)
	
	// Calculate actual duration and date range display
	duration := timeRange.EndTime.Sub(timeRange.StartTime)
	actualDuration := formatDuration(duration)
	dateRangeDisplay := fmt.Sprintf("%s to %s",
		timeRange.StartTime.Format("January 2, 2006 3:04 PM MST"),
		timeRange.EndTime.Format("January 2, 2006 3:04 PM MST"))
	
	report := &models.EnhancedSystemReport{
		ReportID:     reportID,
		ReportType:   reportType,
		GeneratedAt:  time.Now(),
		TimeRange:    models.TimeRange{StartTime: timeRange.StartTime, EndTime: timeRange.EndTime},
		ActualDuration: actualDuration,
		DateRangeDisplay: dateRangeDisplay,
		ExecutiveSummary: executiveSummary,
		Performance:  performanceAnalysis,
		Trends:       trends,
		Recommendations: recommendations,
		Highlights:   highlights,
		MetricsCount: len(historicalMetrics),
		AlertsCount:  len(alerts),
		InvestigationsCount: len(investigations),
	}
	
	// Store the report
	err = s.repo.SaveEnhancedReport(ctx, report)
	if err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}
	
	return report, nil
}

// calculateTimeRange determines the time range for the report
func (s *ReportService) calculateTimeRange(reportType string) repository.TimeRange {
	now := time.Now()
	
	// Get earliest metrics time to ensure we don't request data before collection started
	if memRepo, ok := s.repo.(*repository.MemoryRepository); ok {
		earliest, err := memRepo.GetEarliestMetricTime(context.Background())
		if err == nil {
			// Determine the intended start time
			var intendedStart time.Time
			switch reportType {
			case "daily":
				intendedStart = now.AddDate(0, 0, -1) // Last 24 hours
			case "weekly":
				intendedStart = now.AddDate(0, 0, -7) // Last 7 days
			default:
				intendedStart = now.AddDate(0, 0, -1) // Default to daily
			}
			
			// Use the later of intended start or earliest available data
			if earliest.After(intendedStart) {
				return repository.TimeRange{StartTime: earliest, EndTime: now}
			}
			return repository.TimeRange{StartTime: intendedStart, EndTime: now}
		}
	}
	
	// Fallback to standard time ranges if we can't get earliest time
	switch reportType {
	case "daily":
		start := now.AddDate(0, 0, -1) // Last 24 hours
		return repository.TimeRange{StartTime: start, EndTime: now}
	case "weekly":
		start := now.AddDate(0, 0, -7) // Last 7 days
		return repository.TimeRange{StartTime: start, EndTime: now}
	default:
		// Default to daily
		start := now.AddDate(0, 0, -1)
		return repository.TimeRange{StartTime: start, EndTime: now}
	}
}

// gatherHistoricalMetrics retrieves all metrics within the time range
func (s *ReportService) gatherHistoricalMetrics(ctx context.Context, timeRange repository.TimeRange) ([]*models.MetricsResponse, error) {
	filter := repository.MetricsFilter{
		TimeRange: timeRange,
		Limit:     1000, // Get up to 1000 data points
	}
	
	return s.repo.GetMetrics(ctx, filter)
}

// generateExecutiveSummary creates a high-level summary of system health
func (s *ReportService) generateExecutiveSummary(metrics []*models.MetricsResponse, investigations []*models.Investigation, alerts []*models.Alert, reportType string) models.EnhancedExecutiveSummary {
	if len(metrics) == 0 {
		return models.EnhancedExecutiveSummary{
			OverallHealth: "unknown",
			KeyFindings:   []string{"No metrics data available for this time period"},
		}
	}
	
	// Calculate averages
	var totalCPU, totalMemory float64
	var totalTCP int
	var highCPUCount, highMemoryCount int
	
	for _, metric := range metrics {
		totalCPU += metric.CPUUsage
		totalMemory += metric.MemoryUsage
		totalTCP += metric.TCPConnections
		
		if metric.CPUUsage > 80 {
			highCPUCount++
		}
		if metric.MemoryUsage > 85 {
			highMemoryCount++
		}
	}
	
	avgCPU := totalCPU / float64(len(metrics))
	avgMemory := totalMemory / float64(len(metrics))
	avgTCP := float64(totalTCP) / float64(len(metrics))
	
	// Determine overall health
	health := "healthy"
	if avgCPU > 80 || avgMemory > 85 || len(alerts) > 5 {
		health = "warning"
	}
	if avgCPU > 95 || avgMemory > 95 || len(alerts) > 10 {
		health = "critical"
	}
	
	// Generate key findings
	findings := []string{
		fmt.Sprintf("Average CPU usage: %.1f%%", avgCPU),
		fmt.Sprintf("Average memory usage: %.1f%%", avgMemory),
		fmt.Sprintf("Average TCP connections: %.0f", avgTCP),
	}
	
	if highCPUCount > 0 {
		findings = append(findings, fmt.Sprintf("High CPU usage detected %d times (>80%%)", highCPUCount))
	}
	if highMemoryCount > 0 {
		findings = append(findings, fmt.Sprintf("High memory usage detected %d times (>85%%)", highMemoryCount))
	}
	if len(investigations) > 0 {
		findings = append(findings, fmt.Sprintf("%d investigations conducted", len(investigations)))
	}
	if len(alerts) > 0 {
		findings = append(findings, fmt.Sprintf("%d alerts generated", len(alerts)))
	}
	
	timeDesc := "24 hours"
	if reportType == "weekly" {
		timeDesc = "7 days"
	}
	
	return models.EnhancedExecutiveSummary{
		OverallHealth: health,
		KeyFindings:   findings,
		TimeDescription: timeDesc,
		MetricsAnalyzed: len(metrics),
	}
}

// analyzePerformance provides detailed performance analysis
func (s *ReportService) analyzePerformance(metrics []*models.MetricsResponse, timeRange repository.TimeRange) models.PerformanceAnalysis {
	if len(metrics) == 0 {
		return models.PerformanceAnalysis{}
	}
	
	// Calculate performance statistics with timestamps
	cpuStats := calculateStatsWithTime(metrics, func(m *models.MetricsResponse) float64 { return m.CPUUsage })
	memoryStats := calculateStatsWithTime(metrics, func(m *models.MetricsResponse) float64 { return m.MemoryUsage })
	
	return models.PerformanceAnalysis{
		CPU: models.MetricStats{
			Average:   cpuStats.avg,
			Min:       cpuStats.min,
			Max:       cpuStats.max,
			StdDev:    cpuStats.stddev,
			PeakValue: cpuStats.max,
			PeakTime:  cpuStats.maxTime,
			MinTime:   cpuStats.minTime,
		},
		Memory: models.MetricStats{
			Average:   memoryStats.avg,
			Min:       memoryStats.min,
			Max:       memoryStats.max,
			StdDev:    memoryStats.stddev,
			PeakValue: memoryStats.max,
			PeakTime:  memoryStats.maxTime,
			MinTime:   memoryStats.minTime,
		},
		TimeRange: fmt.Sprintf("%s to %s", 
			timeRange.StartTime.Format("2006-01-02 15:04"), 
			timeRange.EndTime.Format("2006-01-02 15:04")),
	}
}

type statsWithTime struct {
	avg, min, max, stddev float64
	minTime, maxTime      time.Time
}

func calculateStatsWithTime(metrics []*models.MetricsResponse, valueExtractor func(*models.MetricsResponse) float64) statsWithTime {
	if len(metrics) == 0 {
		return statsWithTime{}
	}
	
	// Extract values and calculate min, max with timestamps
	values := make([]float64, len(metrics))
	min, max := valueExtractor(metrics[0]), valueExtractor(metrics[0])
	minTime, maxTime := metrics[0].Timestamp, metrics[0].Timestamp
	var sum float64
	
	for i, m := range metrics {
		v := valueExtractor(m)
		values[i] = v
		sum += v
		
		if v < min {
			min = v
			minTime = m.Timestamp
		}
		if v > max {
			max = v
			maxTime = m.Timestamp
		}
	}
	
	avg := sum / float64(len(values))
	
	// Calculate sample standard deviation
	var varianceSum float64
	for _, v := range values {
		varianceSum += math.Pow(v-avg, 2)
	}
	// Use sample standard deviation formula (n-1) for better statistical accuracy
	var stddev float64
	if len(values) > 1 {
		stddev = math.Sqrt(varianceSum / float64(len(values)-1))
	} else {
		stddev = 0 // Single data point has no standard deviation
	}
	
	return statsWithTime{
		avg:     avg,
		min:     min,
		max:     max,
		stddev:  stddev,
		minTime: minTime,
		maxTime: maxTime,
	}
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
		}
		return fmt.Sprintf("%d hours", hours)
	} else {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		if hours > 0 {
			return fmt.Sprintf("%d days, %d hours", days, hours)
		}
		return fmt.Sprintf("%d days", days)
	}
}

// analyzeTrends identifies trends in the metrics data
func (s *ReportService) analyzeTrends(metrics []*models.MetricsResponse) []models.Trend {
	if len(metrics) < 2 {
		return []models.Trend{}
	}
	
	// Sort metrics by timestamp
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Timestamp.Before(metrics[j].Timestamp)
	})
	
	trends := []models.Trend{}
	
	// Analyze CPU trend
	cpuTrend := analyzeTrend("CPU Usage", metrics, func(m *models.MetricsResponse) float64 { return m.CPUUsage })
	if cpuTrend.Direction != "stable" {
		trends = append(trends, cpuTrend)
	}
	
	// Analyze Memory trend
	memoryTrend := analyzeTrend("Memory Usage", metrics, func(m *models.MetricsResponse) float64 { return m.MemoryUsage })
	if memoryTrend.Direction != "stable" {
		trends = append(trends, memoryTrend)
	}
	
	return trends
}

func analyzeTrend(name string, metrics []*models.MetricsResponse, valueExtractor func(*models.MetricsResponse) float64) models.Trend {
	if len(metrics) < 2 {
		return models.Trend{Name: name, Direction: "stable"}
	}
	
	// Calculate average of first and last 10% of data points for more stable trend analysis
	n := len(metrics)
	tenth := n / 10
	if tenth < 1 {
		tenth = 1
	}
	
	// Calculate average of first 10% of data points
	var firstSum float64
	for i := 0; i < tenth; i++ {
		firstSum += valueExtractor(metrics[i])
	}
	firstAvg := firstSum / float64(tenth)
	
	// Calculate average of last 10% of data points
	var lastSum float64
	for i := n - tenth; i < n; i++ {
		lastSum += valueExtractor(metrics[i])
	}
	lastAvg := lastSum / float64(tenth)
	
	change := lastAvg - firstAvg
	
	// Calculate percentage change with protection against division by zero
	var changePercent float64
	if math.Abs(firstAvg) > 0.001 { // Avoid division by very small numbers
		changePercent = (change / math.Abs(firstAvg)) * 100
	} else if math.Abs(lastAvg) > 0.001 {
		// If first is near zero, calculate relative to last value instead
		changePercent = (change / math.Abs(lastAvg)) * 100
	} else {
		changePercent = 0 // Both values near zero
	}
	
	// Determine direction based on absolute change relative to average
	avgValue := (firstAvg + lastAvg) / 2
	direction := "stable"
	if avgValue > 0 {
		relativeChange := math.Abs(change) / avgValue
		if relativeChange > 0.1 { // 10% relative change threshold
			if change > 0 {
				direction = "increasing"
			} else {
				direction = "decreasing"
			}
		}
	} else if math.Abs(change) > 5 { // For zero-baseline metrics, use absolute threshold
		if change > 0 {
			direction = "increasing"
		} else {
			direction = "decreasing"
		}
	}
	
	return models.Trend{
		Name:      name,
		Direction: direction,
		Change:    change,
		ChangePercent: changePercent,
	}
}

// generateRecommendations provides actionable recommendations
func (s *ReportService) generateRecommendations(performance models.PerformanceAnalysis, trends []models.Trend, alerts []*models.Alert) []string {
	recommendations := []string{}
	
	// Performance-based recommendations
	if performance.CPU.Average > 80 {
		recommendations = append(recommendations, "Consider scaling CPU resources or optimizing high-CPU processes")
	}
	if performance.Memory.Average > 85 {
		recommendations = append(recommendations, "Monitor memory usage closely and consider increasing available memory")
	}
	if performance.CPU.Max > 95 {
		recommendations = append(recommendations, "Investigate processes causing CPU spikes - check top CPU consumers")
	}
	if performance.Memory.Max > 95 {
		recommendations = append(recommendations, "Memory usage reached critical levels - review memory leaks or increase capacity")
	}
	
	// Trend-based recommendations
	for _, trend := range trends {
		if trend.Direction == "increasing" && trend.ChangePercent > 20 {
			recommendations = append(recommendations, fmt.Sprintf("%s is trending upward (+%.1f%%) - monitor closely", trend.Name, trend.ChangePercent))
		}
	}
	
	// Alert-based recommendations
	if len(alerts) > 10 {
		recommendations = append(recommendations, "High alert volume detected - review alert thresholds and investigate root causes")
	}
	if len(alerts) == 0 {
		recommendations = append(recommendations, "System appears stable with no active alerts")
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System performance appears normal - continue regular monitoring")
	}
	
	return recommendations
}

// generateHighlights creates notable events and insights
func (s *ReportService) generateHighlights(metrics []*models.MetricsResponse, investigations []*models.Investigation, alerts []*models.Alert, reportType string) []string {
	highlights := []string{}
	
	if len(metrics) == 0 {
		highlights = append(highlights, "No metrics data collected during this period")
		return highlights
	}
	
	// Find peak resource usage
	var maxCPU, maxMemory float64
	var maxCPUTime, maxMemoryTime time.Time
	
	for _, metric := range metrics {
		if metric.CPUUsage > maxCPU {
			maxCPU = metric.CPUUsage
			maxCPUTime = metric.Timestamp
		}
		if metric.MemoryUsage > maxMemory {
			maxMemory = metric.MemoryUsage
			maxMemoryTime = metric.Timestamp
		}
	}
	
	// Add highlights for peak usage
	if maxCPU > 90 {
		highlights = append(highlights, fmt.Sprintf("Peak CPU usage: %.1f%% at %s", maxCPU, maxCPUTime.Format("Jan 02 15:04")))
	}
	if maxMemory > 90 {
		highlights = append(highlights, fmt.Sprintf("Peak memory usage: %.1f%% at %s", maxMemory, maxMemoryTime.Format("Jan 02 15:04")))
	}
	
	// Add investigation highlights
	for _, inv := range investigations {
		if inv.Status == "completed" {
			highlights = append(highlights, fmt.Sprintf("Investigation %s completed: %s", inv.ID[:8], inv.Findings))
		}
	}
	
	// Add alert highlights (most recent)
	if len(alerts) > 0 {
		recentAlert := alerts[0] // Assuming alerts are ordered by recency
		highlights = append(highlights, fmt.Sprintf("Latest alert: %s (severity: %s)", recentAlert.Message, recentAlert.Severity))
	}
	
	// Time period summary
	timeDesc := "last 24 hours"
	if reportType == "weekly" {
		timeDesc = "last 7 days"
	}
	highlights = append(highlights, fmt.Sprintf("Analyzed %d metric data points over %s", len(metrics), timeDesc))
	
	return highlights
}

// ListReports retrieves all generated reports with summary information
func (s *ReportService) ListReports(ctx context.Context) ([]*models.EnhancedSystemReport, error) {
	// Get all enhanced reports from repository
	reports, err := s.repo.ListEnhancedReports(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list enhanced reports: %w", err)
	}
	
	// Sort by generation time (newest first)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].GeneratedAt.After(reports[j].GeneratedAt)
	})
	
	return reports, nil
}

// GetReport retrieves a specific report by ID
func (s *ReportService) GetReport(ctx context.Context, reportID string) (*models.EnhancedSystemReport, error) {
	return s.repo.GetEnhancedReport(ctx, reportID)
}