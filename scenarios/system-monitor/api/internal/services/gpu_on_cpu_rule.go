package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// GPU-capable workload identities are deliberately exact. A broad match such
// as image-tools/python would conflate model inference with CPU preparation,
// package setup and unrelated operations.
const (
	gpuOnCPUOwner          = "image-tools"
	gpuOnCPUWindow         = 15 * time.Minute
	gpuOnCPUTop            = repository.MaxProcessTimelineGroups
	gpuOnCPUMinimumAge     = time.Minute
	gpuOnCPUMaximumStale   = time.Minute
	gpuOnCPUMinimumSamples = 3
	gpuOnCPUMinimumCPU     = 25.0
)

var gpuCapableProcessNames = map[string]map[string]struct{}{
	"image-tools": {"sd-cli": {}, "sd-gpu": {}},
}

type gpuOnCPURule struct {
	source ProcessTimelineSource
}

// NewGPUOnCPURule detects a known GPU-capable workload that remains CPU-bound
// while the host has measured GPU telemetry. It reports only; remediation and
// capacity changes remain with the owner/control plane.
func NewGPUOnCPURule(source ProcessTimelineSource) AlertRule {
	return gpuOnCPURule{source: source}
}

func (gpuOnCPURule) ID() string { return RuleGPUCapableWorkOnCPU }

func (r gpuOnCPURule) Evaluate(ctx context.Context, snap AlertSnapshot) RuleEvaluation {
	if r.source == nil {
		return unsupportedRule("owner-attributed process timeline is not configured")
	}
	if snap.Metrics == nil || snap.Metrics.GPUState.Status != "measured" || snap.Metrics.GPUUsage == nil {
		return unsupportedRule("GPU telemetry is not measured on this host")
	}
	processState := snap.Metrics.GPUProcessesState
	if processState.Status != "measured" || processState.ObservedAt.IsZero() || snap.Now.Sub(processState.ObservedAt) < 0 || snap.Now.Sub(processState.ObservedAt) > gpuOnCPUMaximumStale {
		return unsupportedRule("fresh, complete GPU process inventory is unavailable")
	}
	entries, err := r.source.GetProcessTimelineRanked(ctx, gpuOnCPUWindow, gpuOnCPUOwner, gpuOnCPUTop, "cpu_seconds")
	if err != nil {
		return failedRule("read GPU-capable process timeline: " + err.Error())
	}
	if len(entries) == 0 {
		return unsupportedRule("no owner-attributed process samples are available")
	}
	possiblyTruncated := len(entries) == gpuOnCPUTop

	var conditions []AlertCondition
	for _, entry := range entries {
		knownNames, knownOwner := gpuCapableProcessNames[entry.Owner]
		if !knownOwner {
			continue
		}
		if _, knownProcess := knownNames[entry.Comm]; !knownProcess {
			continue
		}
		if entry.FirstSeen.IsZero() || entry.LastSeen.IsZero() || entry.FirstSeen.After(entry.LastSeen) || entry.LastSeen.After(snap.Now) {
			continue
		}
		if snap.Now.Sub(entry.FirstSeen) < gpuOnCPUMinimumAge || snap.Now.Sub(entry.LastSeen) > gpuOnCPUMaximumStale || entry.SampleCount < gpuOnCPUMinimumSamples || entry.CPUPct < gpuOnCPUMinimumCPU || entry.GPUVRAMMB > 0 {
			continue
		}

		severity := SeverityWarning
		if entry.CPUPct >= 100 {
			severity = SeverityCritical
		}
		message := fmt.Sprintf("GPU-capable %s process %s averaged %.1f%% CPU with no process GPU memory observed; last seen %s ago.", entry.Owner, entry.Comm, entry.CPUPct, snap.Now.Sub(entry.LastSeen).Round(time.Second))
		conditions = append(conditions, AlertCondition{
			Key:        entry.Owner + ":" + entry.Comm,
			Severity:   severity,
			Message:    message,
			MetricName: RuleGPUCapableWorkOnCPU,
			Value:      entry.CPUPct,
			Threshold:  gpuOnCPUMinimumCPU,
			Details: map[string]interface{}{
				"owner":              entry.Owner,
				"command":            entry.Comm,
				"pid":                entry.PID,
				"cpu_percent":        entry.CPUPct,
				"max_cpu_percent":    entry.MaxCPUPct,
				"gpu_vram_mb":        entry.GPUVRAMMB,
				"sample_count":       entry.SampleCount,
				"first_seen":         entry.FirstSeen.UTC().Format(time.RFC3339Nano),
				"last_seen":          entry.LastSeen.UTC().Format(time.RFC3339Nano),
				"host_gpu_percent":   *snap.Metrics.GPUUsage,
				"possibly_truncated": possiblyTruncated,
			},
		})
	}
	if len(conditions) == 0 {
		if possiblyTruncated {
			return unsupportedRule("owner-attributed process timeline reached its bounded result limit")
		}
		return clearRule()
	}
	return activeRule(conditions...)
}
