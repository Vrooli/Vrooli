package execution

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"test-genie/internal/orchestrator"
)

const (
	planHistoryWindow         = 90 * 24 * time.Hour
	maxHistoryRows            = 2000
	maxScenarioSamplesPerPhase = 20
	maxGlobalSamplesPerPhase   = 50
	primaryScenarioSamples     = 5
)

// ExecutionPlanner exposes scenario-aware plan previews for API/UI/CLI surfaces.
type ExecutionPlanner interface {
	Preview(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*ExecutionPlanPreview, error)
}

type executionPlanBuilder interface {
	PreviewExecution(req orchestrator.SuiteExecutionRequest) (*orchestrator.ExecutionPlanPreview, error)
}

type phaseSampleReader interface {
	ListPhaseSamples(ctx context.Context, phaseNames []string, since time.Time, limit int) ([]PhaseDurationSample, error)
}

// ExecutionPlanService builds scenario-aware phase plans enriched with historical estimates.
type ExecutionPlanService struct {
	builder executionPlanBuilder
	samples phaseSampleReader
	now     func() time.Time
}

func NewExecutionPlanService(builder executionPlanBuilder, samples phaseSampleReader) *ExecutionPlanService {
	return &ExecutionPlanService{
		builder: builder,
		samples: samples,
		now:     time.Now,
	}
}

// Preview returns the actual selected phase plan plus timing guidance.
func (s *ExecutionPlanService) Preview(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*ExecutionPlanPreview, error) {
	basePlan, err := s.builder.PreviewExecution(req)
	if err != nil {
		return nil, err
	}

	preview := &ExecutionPlanPreview{
		ScenarioName: basePlan.ScenarioName,
		PresetUsed:   basePlan.PresetUsed,
		Warnings:     append([]string(nil), basePlan.Warnings...),
		Phases:       make([]PlannedPhase, 0, len(basePlan.Phases)),
	}

	if len(basePlan.Phases) == 0 {
		return preview, nil
	}

	phaseNames := make([]string, 0, len(basePlan.Phases))
	for _, phase := range basePlan.Phases {
		phaseNames = append(phaseNames, phase.Name)
	}

	since := s.now().UTC().Add(-planHistoryWindow)
	phaseSamples, err := s.samples.ListPhaseSamples(ctx, phaseNames, since, maxHistoryRows)
	if err != nil {
		return nil, err
	}

	scenarioBuckets, globalBuckets := bucketPhaseSamples(basePlan.ScenarioName, phaseSamples)

	for _, phase := range basePlan.Phases {
		timeoutSeconds := clampNonNegative(phase.TimeoutSeconds)
		scenarioDurations := scenarioBuckets[normalizePhaseKey(phase.Name)]
		globalDurations := globalBuckets[normalizePhaseKey(phase.Name)]
		estimate := estimatePhase(timeoutSeconds, scenarioDurations, globalDurations)

		previewPhase := PlannedPhase{
			Name:                     phase.Name,
			Description:              phase.Description,
			Optional:                 phase.Optional,
			EstimatedDurationSeconds: estimate.durationSeconds,
			TimeoutSeconds:           timeoutSeconds,
			EstimateSource:           estimate.source,
			EstimateConfidence:       estimate.confidence,
			EstimateSampleSize:       estimate.sampleSize,
		}
		preview.Phases = append(preview.Phases, previewPhase)
		preview.Summary.PhaseCount++
		preview.Summary.EstimatedDurationSeconds += previewPhase.EstimatedDurationSeconds
		preview.Summary.TimeoutSeconds += previewPhase.TimeoutSeconds
	}

	return preview, nil
}

type phaseEstimate struct {
	durationSeconds int
	source          EstimateSource
	confidence      EstimateConfidence
	sampleSize      int
}

func bucketPhaseSamples(scenarioName string, samples []PhaseDurationSample) (map[string][]int, map[string][]int) {
	scenarioBuckets := make(map[string][]int)
	globalBuckets := make(map[string][]int)

	for _, sample := range samples {
		key := normalizePhaseKey(sample.PhaseName)
		if key == "" {
			continue
		}
		duration := clampNonNegative(sample.DurationSeconds)

		if len(globalBuckets[key]) < maxGlobalSamplesPerPhase {
			globalBuckets[key] = append(globalBuckets[key], duration)
		}
		if strings.EqualFold(sample.ScenarioName, scenarioName) && len(scenarioBuckets[key]) < maxScenarioSamplesPerPhase {
			scenarioBuckets[key] = append(scenarioBuckets[key], duration)
		}
	}

	return scenarioBuckets, globalBuckets
}

func estimatePhase(timeoutSeconds int, scenarioDurations, globalDurations []int) phaseEstimate {
	scenarioCount := len(scenarioDurations)
	globalCount := len(globalDurations)

	switch {
	case scenarioCount >= primaryScenarioSamples:
		return phaseEstimate{
			durationSeconds: percentileSeconds(scenarioDurations, 0.5),
			source:          EstimateSourceScenarioHistory,
			confidence:      confidenceFor(EstimateSourceScenarioHistory, scenarioCount, globalCount),
			sampleSize:      scenarioCount,
		}
	case scenarioCount > 0 && globalCount > 0:
		globalWeight := minInt(globalCount, primaryScenarioSamples)
		blended := weightedBlend(
			percentileSeconds(scenarioDurations, 0.5), scenarioCount,
			percentileSeconds(globalDurations, 0.5), globalWeight,
		)
		return phaseEstimate{
			durationSeconds: blended,
			source:          EstimateSourceBlendedHistory,
			confidence:      confidenceFor(EstimateSourceBlendedHistory, scenarioCount, globalCount),
			sampleSize:      scenarioCount + globalWeight,
		}
	case scenarioCount > 0:
		return phaseEstimate{
			durationSeconds: percentileSeconds(scenarioDurations, 0.5),
			source:          EstimateSourceScenarioHistory,
			confidence:      confidenceFor(EstimateSourceScenarioHistory, scenarioCount, globalCount),
			sampleSize:      scenarioCount,
		}
	case globalCount > 0:
		return phaseEstimate{
			durationSeconds: percentileSeconds(globalDurations, 0.5),
			source:          EstimateSourceGlobalHistory,
			confidence:      confidenceFor(EstimateSourceGlobalHistory, scenarioCount, globalCount),
			sampleSize:      globalCount,
		}
	default:
		return phaseEstimate{
			durationSeconds: timeoutSeconds,
			source:          EstimateSourceTimeoutFallback,
			confidence:      EstimateConfidenceLow,
			sampleSize:      0,
		}
	}
}

func confidenceFor(source EstimateSource, scenarioCount, globalCount int) EstimateConfidence {
	switch source {
	case EstimateSourceScenarioHistory:
		switch {
		case scenarioCount >= 10:
			return EstimateConfidenceHigh
		case scenarioCount >= 3:
			return EstimateConfidenceMedium
		default:
			return EstimateConfidenceLow
		}
	case EstimateSourceBlendedHistory:
		if scenarioCount >= 2 && globalCount >= 5 {
			return EstimateConfidenceMedium
		}
		return EstimateConfidenceLow
	case EstimateSourceGlobalHistory:
		if globalCount >= 12 {
			return EstimateConfidenceMedium
		}
		return EstimateConfidenceLow
	default:
		return EstimateConfidenceLow
	}
}

func percentileSeconds(values []int, percentile float64) int {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	if percentile <= 0 {
		return sorted[0]
	}
	if percentile >= 1 {
		return sorted[len(sorted)-1]
	}
	index := int(math.Round(percentile * float64(len(sorted)-1)))
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func weightedBlend(left, leftWeight, right, rightWeight int) int {
	totalWeight := leftWeight + rightWeight
	if totalWeight <= 0 {
		return 0
	}
	return int(math.Round(float64(left*leftWeight+right*rightWeight) / float64(totalWeight)))
}

func normalizePhaseKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func clampNonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

var _ ExecutionPlanner = (*ExecutionPlanService)(nil)
