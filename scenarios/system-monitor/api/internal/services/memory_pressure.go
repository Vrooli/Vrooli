package services

// DOC: docs/concepts/ARCHITECTURE.md#memory-pressure

import (
	"context"
	"errors"
	"runtime"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
)

// memoryPressureContributorLimit bounds attribution to the few processes an
// operator can act on.
const memoryPressureContributorLimit = 5

// GetMemoryPressure returns the scheduler-owned memory-pressure reading. When
// the level is elevated or critical it attaches the top paging contributors
// from the shared process sampler, with scenario ownership attributed.
func (s *MonitorService) GetMemoryPressure(ctx context.Context) (*models.MemoryPressure, error) {
	data := s.latestCollectorSnapshot()["pressure"]
	if data == nil {
		return &models.MemoryPressure{
			Level:              collectors.MemoryPressureUnknown,
			LevelSource:        "system-monitor/pressure",
			LevelReason:        "the pressure collector has not sampled yet",
			Platform:           runtime.GOOS,
			Signals:            map[string]models.MetricState{},
			ContributorsStatus: "not_requested",
			ObservedAt:         s.clock.Now(),
		}, nil
	}
	reading, ok := data.Values["memory_pressure"].(collectors.MemoryPressureReading)
	if !ok {
		return nil, errors.New("pressure collector payload has no memory-pressure reading")
	}
	result := MemoryPressureFromReading(reading)
	if collectors.LevelRank(reading.Level) < collectors.LevelRank(collectors.MemoryPressureElevated) {
		result.ContributorsStatus = "not_requested"
		result.ContributorsReason = "contributors are attributed only at elevated or critical pressure"
		return result, nil
	}
	if s.sampler == nil {
		result.ContributorsStatus = "unsupported"
		result.ContributorsReason = "no process sampler is configured"
		return result, nil
	}
	samples, err := s.sampler.Sample(ctx)
	switch {
	case errors.Is(err, procsampler.ErrUnsupported):
		result.ContributorsStatus = "unsupported"
		result.ContributorsReason = "process sampling is unsupported on " + runtime.GOOS
		return result, nil
	case err != nil:
		result.ContributorsStatus = "failed"
		result.ContributorsReason = err.Error()
		return result, nil
	}
	if s.attributor != nil {
		s.attributor.Attribute(ctx, samples)
	}
	result.Contributors = MemoryPressureContributors(samples, memoryPressureContributorLimit)
	result.ContributorsStatus = "measured"
	if len(result.Contributors) == 0 {
		result.ContributorsReason = "no process has a measured fault rate or swapped memory yet"
	}
	return result, nil
}

// MemoryPressureFromReading projects a collector reading onto the API model.
func MemoryPressureFromReading(reading collectors.MemoryPressureReading) *models.MemoryPressure {
	result := &models.MemoryPressure{
		Level:       reading.Level,
		LevelSource: reading.LevelSource,
		LevelReason: reading.LevelReason,
		Platform:    reading.Platform,
		Signals:     make(map[string]models.MetricState, len(reading.Signals)),
		ObservedAt:  reading.ObservedAt,
	}
	for name, signal := range reading.Signals {
		result.Signals[name] = models.MetricState{
			Status:     signal.Status,
			Value:      signal.Value,
			Reason:     signal.Reason,
			Provenance: signal.Provenance,
			Units:      signal.Units,
			ObservedAt: reading.ObservedAt,
		}
	}
	return result
}

// MemoryPressureContributors ranks samples by measured fault rate, then
// swapped memory, and maps them to the API model with owner attribution.
func MemoryPressureContributors(samples []procsampler.ProcessSample, limit int) []models.MemoryPressureContributor {
	ranked := procsampler.RankPagingContributors(samples, limit)
	out := make([]models.MemoryPressureContributor, 0, len(ranked))
	for _, sample := range ranked {
		owner := sample.Owner
		if owner == "" {
			owner = procsampler.OwnerUnknown
		}
		faultStatus := sample.FaultStatus
		if faultStatus == "" {
			faultStatus = procsampler.FaultUnsupported
		}
		swapStatus := sample.SwapStatus
		if swapStatus == "" {
			swapStatus = "unsupported"
		}
		out = append(out, models.MemoryPressureContributor{
			PID:                  sample.PID,
			Name:                 sample.Comm,
			Owner:                owner,
			MajorFaultsPerSecond: sample.MajorFaultsPerSecond,
			FaultStatus:          faultStatus,
			FaultCounter:         sample.FaultCounter,
			SwapKB:               sample.SwapKB,
			SwapStatus:           swapStatus,
			RSSKB:                sample.RSSKB,
		})
	}
	return out
}
