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
		Active:                       s.Active,
		MetricCollectionInterval:     int32(s.MetricCollectionInterval),
		AnomalyDetectionInterval:     int32(s.AnomalyDetectionInterval),
		ThresholdCheckInterval:       int32(s.ThresholdCheckInterval),
		CooldownPeriodSeconds:        int32(s.CooldownPeriodSeconds),
		CpuThreshold:                 s.CPUThreshold,
		CpuHighPercent:               s.CPUHighPercent,
		CpuCriticalPercent:           s.CPUCriticalPercent,
		CpuEscalationCooldownSeconds: int32(s.CPUEscalationCooldownSeconds),
		CpuEscalationDebounceTicks:   int32(s.CPUEscalationDebounceTicks),
		CpuSustainedWindowTicks:      int32(s.CPUSustainedWindowTicks),
		CpuPressureThreshold:         s.CPUPressureThreshold,
		MemoryThreshold:              s.MemoryThreshold,
		DiskThreshold:                s.DiskThreshold,

		MetricsRetentionDays:          int32(s.MetricsRetentionDays),
		RetentionCheckIntervalSeconds: int32(s.RetentionCheckIntervalSeconds),
		RetentionRunOnStartup:         s.RetentionRunOnStartup,
		CompactAfterRetention:         s.CompactAfterRetention,

		DiskHighPercent:               s.DiskHighPercent,
		DiskCriticalPercent:           s.DiskCriticalPercent,
		DiskEscalationCooldownSeconds: int32(s.DiskEscalationCooldownSeconds),
		DiskEscalationDebounceTicks:   int32(s.DiskEscalationDebounceTicks),
		DiskFastFillJumpPercent:       s.DiskFastFillJumpPercent,
	}
}

func ProtoToSettings(pb *settingspb.SystemSettings) *services.Settings {
	if pb == nil {
		return nil
	}
	return &services.Settings{
		Active:                       pb.Active,
		MetricCollectionInterval:     int(pb.MetricCollectionInterval),
		AnomalyDetectionInterval:     int(pb.AnomalyDetectionInterval),
		ThresholdCheckInterval:       int(pb.ThresholdCheckInterval),
		CooldownPeriodSeconds:        int(pb.CooldownPeriodSeconds),
		CPUThreshold:                 pb.CpuThreshold,
		CPUHighPercent:               pb.CpuHighPercent,
		CPUCriticalPercent:           pb.CpuCriticalPercent,
		CPUEscalationCooldownSeconds: int(pb.CpuEscalationCooldownSeconds),
		CPUEscalationDebounceTicks:   int(pb.CpuEscalationDebounceTicks),
		CPUSustainedWindowTicks:      int(pb.CpuSustainedWindowTicks),
		CPUPressureThreshold:         pb.CpuPressureThreshold,
		MemoryThreshold:              pb.MemoryThreshold,
		DiskThreshold:                pb.DiskThreshold,

		MetricsRetentionDays:          int(pb.MetricsRetentionDays),
		RetentionCheckIntervalSeconds: int(pb.RetentionCheckIntervalSeconds),
		RetentionRunOnStartup:         pb.RetentionRunOnStartup,
		CompactAfterRetention:         pb.CompactAfterRetention,

		DiskHighPercent:               pb.DiskHighPercent,
		DiskCriticalPercent:           pb.DiskCriticalPercent,
		DiskEscalationCooldownSeconds: int(pb.DiskEscalationCooldownSeconds),
		DiskEscalationDebounceTicks:   int(pb.DiskEscalationDebounceTicks),
		DiskFastFillJumpPercent:       pb.DiskFastFillJumpPercent,
	}
}
