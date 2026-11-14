package convert

import (
	settingspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

func SettingsToProto(s *services.Settings) *settingspb.SystemSettings {
	if s == nil {
		return nil
	}
	return &settingspb.SystemSettings{
		Active:                        s.Active,
		MetricCollectionInterval:      int32(s.MetricCollectionInterval),
		AnomalyDetectionInterval:      int32(s.AnomalyDetectionInterval),
		ThresholdCheckInterval:        int32(s.ThresholdCheckInterval),
		CooldownPeriodSeconds:         int32(s.CooldownPeriodSeconds),
		CpuThreshold:                  s.CPUThreshold,
		CpuHighPercent:                s.CPUHighPercent,
		CpuCriticalPercent:            s.CPUCriticalPercent,
		CpuEscalationCooldownSeconds:  int32(s.CPUEscalationCooldownSeconds),
		CpuEscalationDebounceTicks:    int32(s.CPUEscalationDebounceTicks),
		CpuSustainedWindowTicks:       int32(s.CPUSustainedWindowTicks),
		CpuPressureThreshold:          s.CPUPressureThreshold,
		MemoryThreshold:               s.MemoryThreshold,
		MemoryCriticalPercent:         s.MemoryCriticalPercent,
		TcpConnectionsWarning:         int32(s.TCPConnectionsWarning),
		TcpConnectionsCritical:        int32(s.TCPConnectionsCritical),
		TcpGrowthWindowSeconds:        int32(s.TCPGrowthWindowSeconds),
		TcpGrowthFloor:                int32(s.TCPGrowthFloor),
		TcpGrowthMinSamples:           int32(s.TCPGrowthMinSamples),
		TcpGrowthFactor:               s.TCPGrowthFactor,
		GpuUsageWarningPercent:        s.GPUUsageWarningPercent,
		MetricsSnapshotStaleIntervals: int32(s.MetricsSnapshotStaleIntervals),
		OrphanedAgentAgeSeconds:       int32(s.OrphanedAgentAgeSeconds),
		OrphanedAgentMemoryBytes:      s.OrphanedAgentMemoryBytes,
		AutoInvestigateCriticalRules:  cloneBoolMap(s.AutoInvestigateCriticalRules),
		DiskThreshold:                 s.DiskThreshold,

		MetricsRetentionDays:          int32(s.MetricsRetentionDays),
		RetentionCheckIntervalSeconds: int32(s.RetentionCheckIntervalSeconds),
		RetentionRunOnStartup:         s.RetentionRunOnStartup,
		CompactAfterRetention:         s.CompactAfterRetention,

		DiskHighPercent:               s.DiskHighPercent,
		DiskCriticalPercent:           s.DiskCriticalPercent,
		DiskEscalationCooldownSeconds: int32(s.DiskEscalationCooldownSeconds),
		DiskEscalationDebounceTicks:   int32(s.DiskEscalationDebounceTicks),
		DiskFastFillJumpPercent:       s.DiskFastFillJumpPercent,

		AlertSustainSeconds:                        int32(s.AlertSustainSeconds),
		MemoryPressureSomeAvg60ElevatedPercent:     s.MemoryPressureSomeAvg60ElevatedPercent,
		MemoryPressureFullAvg60CriticalPercent:     s.MemoryPressureFullAvg60CriticalPercent,
		MemoryPressureSwapInElevatedBytesPerSecond: s.MemoryPressureSwapInElevatedBytesPerSecond,
		MemoryPressureSwapInCriticalBytesPerSecond: s.MemoryPressureSwapInCriticalBytesPerSecond,
	}
}

func ProtoToSettings(pb *settingspb.SystemSettings) *services.Settings {
	if pb == nil {
		return nil
	}
	return &services.Settings{
		Active:                        pb.Active,
		MetricCollectionInterval:      int(pb.MetricCollectionInterval),
		AnomalyDetectionInterval:      int(pb.AnomalyDetectionInterval),
		ThresholdCheckInterval:        int(pb.ThresholdCheckInterval),
		CooldownPeriodSeconds:         int(pb.CooldownPeriodSeconds),
		CPUThreshold:                  pb.CpuThreshold,
		CPUHighPercent:                pb.CpuHighPercent,
		CPUCriticalPercent:            pb.CpuCriticalPercent,
		CPUEscalationCooldownSeconds:  int(pb.CpuEscalationCooldownSeconds),
		CPUEscalationDebounceTicks:    int(pb.CpuEscalationDebounceTicks),
		CPUSustainedWindowTicks:       int(pb.CpuSustainedWindowTicks),
		CPUPressureThreshold:          pb.CpuPressureThreshold,
		MemoryThreshold:               pb.MemoryThreshold,
		MemoryCriticalPercent:         pb.MemoryCriticalPercent,
		TCPConnectionsWarning:         int(pb.TcpConnectionsWarning),
		TCPConnectionsCritical:        int(pb.TcpConnectionsCritical),
		TCPGrowthWindowSeconds:        int(pb.TcpGrowthWindowSeconds),
		TCPGrowthFloor:                int(pb.TcpGrowthFloor),
		TCPGrowthMinSamples:           int(pb.TcpGrowthMinSamples),
		TCPGrowthFactor:               pb.TcpGrowthFactor,
		GPUUsageWarningPercent:        pb.GpuUsageWarningPercent,
		MetricsSnapshotStaleIntervals: int(pb.MetricsSnapshotStaleIntervals),
		OrphanedAgentAgeSeconds:       int(pb.OrphanedAgentAgeSeconds),
		OrphanedAgentMemoryBytes:      pb.OrphanedAgentMemoryBytes,
		AutoInvestigateCriticalRules:  cloneBoolMap(pb.AutoInvestigateCriticalRules),
		DiskThreshold:                 pb.DiskThreshold,

		MetricsRetentionDays:          int(pb.MetricsRetentionDays),
		RetentionCheckIntervalSeconds: int(pb.RetentionCheckIntervalSeconds),
		RetentionRunOnStartup:         pb.RetentionRunOnStartup,
		CompactAfterRetention:         pb.CompactAfterRetention,

		DiskHighPercent:               pb.DiskHighPercent,
		DiskCriticalPercent:           pb.DiskCriticalPercent,
		DiskEscalationCooldownSeconds: int(pb.DiskEscalationCooldownSeconds),
		DiskEscalationDebounceTicks:   int(pb.DiskEscalationDebounceTicks),
		DiskFastFillJumpPercent:       pb.DiskFastFillJumpPercent,

		AlertSustainSeconds:                        int(pb.AlertSustainSeconds),
		MemoryPressureSomeAvg60ElevatedPercent:     pb.MemoryPressureSomeAvg60ElevatedPercent,
		MemoryPressureFullAvg60CriticalPercent:     pb.MemoryPressureFullAvg60CriticalPercent,
		MemoryPressureSwapInElevatedBytesPerSecond: pb.MemoryPressureSwapInElevatedBytesPerSecond,
		MemoryPressureSwapInCriticalBytesPerSecond: pb.MemoryPressureSwapInCriticalBytesPerSecond,
	}
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	if source == nil {
		return nil
	}
	clone := make(map[string]bool, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
