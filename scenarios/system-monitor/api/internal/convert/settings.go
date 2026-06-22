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
		Active:                   s.Active,
		MetricCollectionInterval: int32(s.MetricCollectionInterval),
		AnomalyDetectionInterval: int32(s.AnomalyDetectionInterval),
		ThresholdCheckInterval:   int32(s.ThresholdCheckInterval),
		CooldownPeriodSeconds:    int32(s.CooldownPeriodSeconds),
		CpuThreshold:             s.CPUThreshold,
		MemoryThreshold:          s.MemoryThreshold,
		DiskThreshold:            s.DiskThreshold,

		MetricsRetentionDays:          int32(s.MetricsRetentionDays),
		RetentionCheckIntervalSeconds: int32(s.RetentionCheckIntervalSeconds),
		RetentionRunOnStartup:         s.RetentionRunOnStartup,
		CompactAfterRetention:         s.CompactAfterRetention,
	}
}

func ProtoToSettings(pb *settingspb.SystemSettings) *services.Settings {
	if pb == nil {
		return nil
	}
	return &services.Settings{
		Active:                   pb.Active,
		MetricCollectionInterval: int(pb.MetricCollectionInterval),
		AnomalyDetectionInterval: int(pb.AnomalyDetectionInterval),
		ThresholdCheckInterval:   int(pb.ThresholdCheckInterval),
		CooldownPeriodSeconds:    int(pb.CooldownPeriodSeconds),
		CPUThreshold:             pb.CpuThreshold,
		MemoryThreshold:          pb.MemoryThreshold,
		DiskThreshold:            pb.DiskThreshold,

		MetricsRetentionDays:          int(pb.MetricsRetentionDays),
		RetentionCheckIntervalSeconds: int(pb.RetentionCheckIntervalSeconds),
		RetentionRunOnStartup:         pb.RetentionRunOnStartup,
		CompactAfterRetention:         pb.CompactAfterRetention,
	}
}
