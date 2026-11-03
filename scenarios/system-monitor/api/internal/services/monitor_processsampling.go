package services

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// shouldSampleProcesses gates the /proc sampler to its configured interval.
func (s *MonitorService) shouldSampleProcesses(now time.Time) bool {
	if s.sampler == nil || s.procRepo == nil {
		return false
	}
	s.mu.RLock()
	last := s.lastProcSampledAt
	s.mu.RUnlock()
	if !last.IsZero() && now.Sub(last) < s.procSampleEvery {
		return false
	}
	return true
}

// sampleProcesses walks /proc once, attributes each pid to its owning scenario,
// caps to the configured top-N (logging what was dropped — no silent caps), and
// persists the cycle to the process_samples table.
func (s *MonitorService) sampleProcesses(ctx context.Context, now time.Time, gpuVRAMArgs ...map[int]float64) {
	var gpuVRAM map[int]float64
	if len(gpuVRAMArgs) > 0 {
		gpuVRAM = gpuVRAMArgs[0]
	}
	s.mu.Lock()
	s.lastProcSampledAt = now
	s.mu.Unlock()

	started := time.Now()
	samples, err := s.sampler.Sample(ctx)
	s.recordProcSampleDuration(time.Since(started))
	if err != nil {
		if err == procsampler.ErrUnsupported {
			return // non-Linux: aggregate collectors still work; nothing to persist
		}
		s.logger.Warn("process sampler failed", "error", err)
		return
	}
	if len(samples) == 0 {
		return
	}

	if s.attributor != nil {
		s.attributor.Attribute(ctx, samples)
	}

	// Retain independent CPU and RSS leaders. A CPU-only cap loses the exact
	// low-CPU memory hog evidence needed after OOM; the bounded union is at
	// most 2*topN and still comes from this single /proc walk.
	samples, dropped := selectRankSamples(samples, s.procSampleTopN, gpuVRAM, s.procSampleOwners)
	if dropped > 0 {
		s.logger.Info("process sample cap applied", "retained_rows", len(samples), "dropped_processes", dropped)
	}

	rows := make([]repository.ProcessSample, 0, len(samples))
	for _, ps := range samples {
		rows = append(rows, repository.ProcessSample{
			Timestamp:            now.UTC(),
			PID:                  ps.PID,
			PPID:                 ps.PPID,
			Comm:                 ps.Comm,
			Cmdline:              ps.Cmdline,
			Cwd:                  ps.Cwd,
			Owner:                ps.Owner,
			CPUPct:               ps.CPUPct,
			CPUSeconds:           ps.CPUSeconds,
			CPUSecondsStatus:     ps.CPUSecondsStatus,
			CPUSecondsReason:     ps.CPUSecondsReason,
			RSSKB:                ps.RSSKB,
			SwapKB:               ps.SwapKB,
			MajorFaultsPerSecond: ps.MajorFaultsPerSecond,
			MetricsStatus:        ps.MetricsStatus,
			MetricsReason:        ps.MetricsReason,
			Threads:              ps.Threads,
			GPUVRAMMB:            gpuVRAM[ps.PID],
		})
	}
	if err := s.procRepo.SaveProcessSamples(ctx, rows); err != nil {
		s.logger.Error("failed to persist process samples", "rows", len(rows), "error", err)
	}
}

func selectDualRankSamples(samples []procsampler.ProcessSample, top int) ([]procsampler.ProcessSample, int) {
	return selectRankSamples(samples, top, nil, nil)
}

func selectRankSamples(samples []procsampler.ProcessSample, top int, gpuVRAM map[int]float64, pinnedOwners []string) ([]procsampler.ProcessSample, int) {
	if top <= 0 || len(samples) <= top {
		return samples, 0
	}
	keep := make(map[int]struct{}, top*2)
	retainTopProcesses(keep, samples, top, func(a, b procsampler.ProcessSample) bool { return a.CPUPct > b.CPUPct }, nil)
	retainTopProcesses(keep, samples, top, func(a, b procsampler.ProcessSample) bool { return a.RSSKB > b.RSSKB }, nil)
	retainTopProcesses(keep, samples, top, func(a, b procsampler.ProcessSample) bool {
		if a.MajorFaultsPerSecond != b.MajorFaultsPerSecond {
			return a.MajorFaultsPerSecond > b.MajorFaultsPerSecond
		}
		return a.SwapKB > b.SwapKB
	}, func(sample procsampler.ProcessSample) bool {
		return sample.MajorFaultsPerSecond > 0 || sample.SwapKB > 0
	})
	if len(pinnedOwners) > 0 {
		ownerSet := make(map[string]struct{}, len(pinnedOwners))
		ownerSamples := make(map[string][]procsampler.ProcessSample, len(pinnedOwners))
		for _, owner := range pinnedOwners {
			if owner = strings.TrimSpace(owner); owner != "" {
				ownerSet[owner] = struct{}{}
			}
		}
		for _, sample := range samples {
			if _, ok := ownerSet[sample.Owner]; ok {
				ownerSamples[sample.Owner] = append(ownerSamples[sample.Owner], sample)
			}
		}
		for owner, owned := range ownerSamples {
			retainTopProcesses(keep, owned, top, func(a, b procsampler.ProcessSample) bool { return a.CPUPct > b.CPUPct }, nil)
			retainTopProcesses(keep, owned, top, func(a, b procsampler.ProcessSample) bool { return a.RSSKB > b.RSSKB }, nil)
			delete(ownerSet, owner)
		}
	}
	if len(gpuVRAM) > 0 {
		retainTopProcesses(keep, samples, top, func(a, b procsampler.ProcessSample) bool {
			return gpuVRAM[a.PID] > gpuVRAM[b.PID]
		}, func(sample procsampler.ProcessSample) bool { return gpuVRAM[sample.PID] > 0 })
	}
	selected := make([]procsampler.ProcessSample, 0, len(keep))
	for _, sample := range samples {
		if _, ok := keep[sample.PID]; ok {
			selected = append(selected, sample)
		}
	}
	return selected, len(samples) - len(selected)
}

func retainTopProcesses(
	keep map[int]struct{},
	samples []procsampler.ProcessSample,
	top int,
	ranksBefore func(a, b procsampler.ProcessSample) bool,
	include func(sample procsampler.ProcessSample) bool,
) {
	ranked := append([]procsampler.ProcessSample(nil), samples...)
	sort.SliceStable(ranked, func(i, j int) bool { return ranksBefore(ranked[i], ranked[j]) })
	for i := 0; i < top && i < len(ranked); i++ {
		if include == nil || include(ranked[i]) {
			keep[ranked[i].PID] = struct{}{}
		}
	}
}

func gpuVRAMByPID(metricsData []*collectors.MetricData) map[int]float64 {
	for _, data := range metricsData {
		if data.CollectorName != "gpu" {
			continue
		}
		devices, _ := data.Values["devices"].([]models.GPUDeviceMetrics)
		result := make(map[int]float64)
		for _, device := range devices {
			for _, process := range device.Processes {
				result[process.PID] += process.MemoryUsedMB
			}
		}
		return result
	}
	return nil
}
