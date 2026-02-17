package convert

import (
	"system-monitor-api/internal/models"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func EnhancedSystemReportToProto(r *models.EnhancedSystemReport) *domain.EnhancedSystemReport {
	if r == nil {
		return nil
	}
	pb := &domain.EnhancedSystemReport{
		ReportId:            r.ReportID,
		ReportType:          r.ReportType,
		GeneratedAt:         timestamppb.New(r.GeneratedAt),
		ActualDuration:      r.ActualDuration,
		DateRangeDisplay:    r.DateRangeDisplay,
		Recommendations:     r.Recommendations,
		Highlights:          r.Highlights,
		MetricsCount:        int32(r.MetricsCount),
		AlertsCount:         int32(r.AlertsCount),
		InvestigationsCount: int32(r.InvestigationsCount),
	}
	pb.TimeRange = &domain.ReportTimeRange{
		StartTime: timestamppb.New(r.TimeRange.StartTime),
		EndTime:   timestamppb.New(r.TimeRange.EndTime),
	}
	pb.ExecutiveSummary = &domain.EnhancedExecutiveSummary{
		OverallHealth:   r.ExecutiveSummary.OverallHealth,
		KeyFindings:     r.ExecutiveSummary.KeyFindings,
		TimeDescription: r.ExecutiveSummary.TimeDescription,
		MetricsAnalyzed: int32(r.ExecutiveSummary.MetricsAnalyzed),
	}
	pb.Performance = &domain.PerformanceAnalysis{
		Cpu:       metricStatsToProto(r.Performance.CPU),
		Memory:    metricStatsToProto(r.Performance.Memory),
		TimeRange: r.Performance.TimeRange,
	}
	if len(r.Trends) > 0 {
		pb.Trends = make([]*domain.Trend, len(r.Trends))
		for i, t := range r.Trends {
			pb.Trends[i] = &domain.Trend{
				Name:          t.Name,
				Direction:     t.Direction,
				Change:        t.Change,
				ChangePercent: t.ChangePercent,
			}
		}
	}
	return pb
}

func EnhancedSystemReportsToProto(rs []*models.EnhancedSystemReport) []*domain.EnhancedSystemReport {
	result := make([]*domain.EnhancedSystemReport, len(rs))
	for i, r := range rs {
		result[i] = EnhancedSystemReportToProto(r)
	}
	return result
}

func metricStatsToProto(ms models.MetricStats) *domain.MetricStats {
	return &domain.MetricStats{
		Average:   ms.Average,
		Min:       ms.Min,
		Max:       ms.Max,
		StdDev:    ms.StdDev,
		PeakValue: ms.PeakValue,
		PeakTime:  timestamppb.New(ms.PeakTime),
		MinTime:   timestamppb.New(ms.MinTime),
	}
}
