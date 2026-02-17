package convert

import (
	"system-monitor-api/internal/services"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/domain"
)

func SettingsToProto(s *services.Settings) *domain.SystemSettings {
	if s == nil {
		return nil
	}
	return &domain.SystemSettings{
		Active:                   s.Active,
		MetricCollectionInterval: int32(s.MetricCollectionInterval),
		AnomalyDetectionInterval: int32(s.AnomalyDetectionInterval),
		ThresholdCheckInterval:   int32(s.ThresholdCheckInterval),
		CooldownPeriodSeconds:    int32(s.CooldownPeriodSeconds),
		CpuThreshold:             s.CPUThreshold,
		MemoryThreshold:          s.MemoryThreshold,
		DiskThreshold:            s.DiskThreshold,
	}
}

func ProtoToSettings(pb *domain.SystemSettings) *services.Settings {
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
	}
}
