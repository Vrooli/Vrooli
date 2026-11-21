package convert

import (
	reportspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/reports"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func EnhancedSystemReportToProto(r *models.EnhancedSystemReport) *reportspb.EnhancedSystemReport {
	if r == nil {
		return nil
	}
	pb := &reportspb.EnhancedSystemReport{
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
	pb.TimeRange = &reportspb.ReportTimeRange{
		StartTime: timestamppb.New(r.TimeRange.StartTime),
		EndTime:   timestamppb.New(r.TimeRange.EndTime),
	}
	pb.ExecutiveSummary = &reportspb.EnhancedExecutiveSummary{
		OverallHealth:   r.ExecutiveSummary.OverallHealth,
		KeyFindings:     r.ExecutiveSummary.KeyFindings,
		TimeDescription: r.ExecutiveSummary.TimeDescription,
		MetricsAnalyzed: int32(r.ExecutiveSummary.MetricsAnalyzed),
	}
	pb.Performance = &reportspb.PerformanceAnalysis{
		Cpu:       metricStatsToProto(r.Performance.CPU),
		Memory:    metricStatsToProto(r.Performance.Memory),
		TimeRange: r.Performance.TimeRange,
	}
	pb.OwnerResources = &reportspb.OwnerResourceReport{
		WindowStart:           timestamppb.New(r.OwnerResources.WindowStart),
		WindowEnd:             timestamppb.New(r.OwnerResources.WindowEnd),
		SampleCoverageStart:   timestamppb.New(r.OwnerResources.SampleCoverageStart),
		SampleCoverageEnd:     timestamppb.New(r.OwnerResources.SampleCoverageEnd),
		SwapCoverageStart:     timestamppb.New(r.OwnerResources.SwapCoverageStart),
		SwapCoverageEnd:       timestamppb.New(r.OwnerResources.SwapCoverageEnd),
		Complete:              r.OwnerResources.Complete,
		AttributionComplete:   r.OwnerResources.AttributionComplete,
		SwapCoverageComplete:  r.OwnerResources.SwapCoverageComplete,
		ReturnedProcessGroups: int32(r.OwnerResources.ReturnedProcessGroups),
		SwapMeasuredSamples:   r.OwnerResources.SwapMeasuredSamples,
		SwapUnknownSamples:    r.OwnerResources.SwapUnknownSamples,
		Owners:                make([]*reportspb.OwnerResourceSummary, 0, len(r.OwnerResources.Owners)),
	}
	for _, owner := range r.OwnerResources.Owners {
		pb.OwnerResources.Owners = append(pb.OwnerResources.Owners, &reportspb.OwnerResourceSummary{
			Owner:                    owner.Owner,
			ProcessGroups:            int32(owner.ProcessGroups),
			Samples:                  owner.Samples,
			CoreHours:                owner.CoreHours,
			MeanRssGibProcessSum:     owner.MeanRSSGiBProcessSum,
			MeanSwapGibProcessSum:    owner.MeanSwapGiBProcessSum,
			PeakSingleProcessVramGib: owner.PeakSingleProcessVRAMGiB,
			SwapMeasuredSamples:     owner.SwapMeasuredSamples,
			SwapUnknownSamples:      owner.SwapUnknownSamples,
		})
	}
	if len(r.Trends) > 0 {
		pb.Trends = make([]*reportspb.Trend, len(r.Trends))
		for i, t := range r.Trends {
			pb.Trends[i] = &reportspb.Trend{
				Name:          t.Name,
				Direction:     t.Direction,
				Change:        t.Change,
				ChangePercent: t.ChangePercent,
			}
		}
	}
	return pb
}

func EnhancedSystemReportsToProto(rs []*models.EnhancedSystemReport) []*reportspb.EnhancedSystemReport {
	result := make([]*reportspb.EnhancedSystemReport, len(rs))
	for i, r := range rs {
		result[i] = EnhancedSystemReportToProto(r)
	}
	return result
}

func metricStatsToProto(ms models.MetricStats) *reportspb.MetricStats {
	return &reportspb.MetricStats{
		Average:   ms.Average,
		Min:       ms.Min,
		Max:       ms.Max,
		StdDev:    ms.StdDev,
		PeakValue: ms.PeakValue,
		PeakTime:  timestamppb.New(ms.PeakTime),
		MinTime:   timestamppb.New(ms.MinTime),
	}
}
