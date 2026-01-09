// Package analyzer implements UX metrics computation from collected interaction data.
// It provides algorithms for detecting friction signals and computing aggregate metrics.
package analyzer

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// Config holds tunable parameters for friction detection.
type Config struct {
	ExcessiveStepDurationMs  int64   // Default: 10000 (10s)
	ZigZagThreshold          float64 // Default: 0.3 (directness < 0.7)
	HesitationThresholdMs    int64   // Default: 3000 (3s)
	RapidClickWindowMs       int64   // Default: 500
	RapidClickCountThreshold int     // Default: 3
}

// DefaultConfig provides sensible defaults for friction detection.
var DefaultConfig = Config{
	ExcessiveStepDurationMs:  10000,
	ZigZagThreshold:          0.3,
	HesitationThresholdMs:    3000,
	RapidClickWindowMs:       500,
	RapidClickCountThreshold: 3,
}

// Analyzer computes UX metrics from collected interaction data.
type Analyzer struct {
	repo   uxmetrics.Repository
	config Config
}

// NewAnalyzer creates an analyzer with the given repository and config.
func NewAnalyzer(repo uxmetrics.Repository, config *Config) *Analyzer {
	cfg := DefaultConfig
	if config != nil {
		cfg = *config
	}
	return &Analyzer{repo: repo, config: cfg}
}

// AnalyzeExecution computes full metrics for a completed execution.
func (a *Analyzer) AnalyzeExecution(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
	traces, err := a.repo.ListInteractionTraces(ctx, executionID)
	if err != nil {
		return nil, err
	}

	stepMetrics := make([]contracts.StepMetrics, 0)
	allSignals := make([]contracts.FrictionSignal, 0)

	// Group traces by step
	stepTraces := groupByStep(traces)

	// Get sorted step indices for consistent ordering
	stepIndices := make([]int, 0, len(stepTraces))
	for idx := range stepTraces {
		stepIndices = append(stepIndices, idx)
	}
	sort.Ints(stepIndices)

	for _, stepIndex := range stepIndices {
		sTraces := stepTraces[stepIndex]
		cursorPath, _ := a.repo.GetCursorPath(ctx, executionID, stepIndex)

		sm := a.analyzeStepTraces(stepIndex, sTraces, cursorPath)
		stepMetrics = append(stepMetrics, sm)
		allSignals = append(allSignals, sm.FrictionSignals...)
	}

	// Compute aggregate metrics
	totalDuration := int64(0)
	totalRetries := 0
	successCount := 0
	failCount := 0
	totalCursorDist := 0.0
	frictionSum := 0.0

	for _, sm := range stepMetrics {
		totalDuration += sm.TotalDurationMs
		totalRetries += sm.RetryCount
		if sm.CursorPath != nil {
			totalCursorDist += sm.CursorPath.TotalDistancePx
		}
		frictionSum += sm.FrictionScore
	}

	// Count successes/failures from traces
	seenSteps := make(map[int]bool)
	for _, t := range traces {
		if !seenSteps[t.StepIndex] {
			seenSteps[t.StepIndex] = true
			if t.Success {
				successCount++
			} else {
				failCount++
			}
		}
	}

	avgDuration := 0.0
	overallFriction := 0.0
	if len(stepMetrics) > 0 {
		avgDuration = float64(totalDuration) / float64(len(stepMetrics))
		overallFriction = frictionSum / float64(len(stepMetrics))
	}

	// Extract workflow ID from first trace (if available)
	var workflowID uuid.UUID
	// Note: WorkflowID is not in InteractionTrace, would need to be passed in
	// or looked up separately. For now we leave it as zero UUID.

	return &contracts.ExecutionMetrics{
		ExecutionID:       executionID,
		WorkflowID:        workflowID,
		ComputedAt:        time.Now().UTC(),
		TotalDurationMs:   totalDuration,
		StepCount:         len(stepMetrics),
		SuccessfulSteps:   successCount,
		FailedSteps:       failCount,
		TotalRetries:      totalRetries,
		AvgStepDurationMs: avgDuration,
		TotalCursorDist:   totalCursorDist,
		OverallFriction:   overallFriction,
		FrictionSignals:   allSignals,
		StepMetrics:       stepMetrics,
		Summary:           a.generateSummary(stepMetrics, allSignals),
	}, nil
}

// AnalyzeStep computes metrics for a single step.
func (a *Analyzer) AnalyzeStep(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error) {
	traces, err := a.repo.ListInteractionTraces(ctx, executionID)
	if err != nil {
		return nil, err
	}

	// Filter to just this step
	stepTraces := make([]contracts.InteractionTrace, 0)
	for _, t := range traces {
		if t.StepIndex == stepIndex {
			stepTraces = append(stepTraces, t)
		}
	}

	cursorPath, _ := a.repo.GetCursorPath(ctx, executionID, stepIndex)
	sm := a.analyzeStepTraces(stepIndex, stepTraces, cursorPath)
	return &sm, nil
}

func (a *Analyzer) analyzeStepTraces(stepIndex int, traces []contracts.InteractionTrace, cursorPath *contracts.CursorPath) contracts.StepMetrics {
	signals := make([]contracts.FrictionSignal, 0)

	totalDuration := int64(0)
	var nodeID, stepType string
	retryCount := 0

	for i, t := range traces {
		totalDuration += t.DurationMs
		if nodeID == "" {
			nodeID = t.ElementID
		}
		if stepType == "" {
			stepType = string(t.ActionType)
		}
		// Count retries (traces beyond the first for same step)
		if i > 0 {
			retryCount++
		}
	}

	// Detect friction signals
	signals = append(signals, a.detectExcessiveTime(stepIndex, totalDuration)...)
	if cursorPath != nil {
		signals = append(signals, a.detectZigZag(stepIndex, cursorPath)...)
		signals = append(signals, a.detectHesitations(stepIndex, cursorPath)...)
	}
	signals = append(signals, a.detectRapidClicks(stepIndex, traces)...)
	signals = append(signals, a.detectMultipleRetries(stepIndex, retryCount)...)

	// Calculate friction score (weighted average of signal severities)
	frictionScore := a.calculateFrictionScore(signals)

	return contracts.StepMetrics{
		StepIndex:       stepIndex,
		NodeID:          nodeID,
		StepType:        stepType,
		TotalDurationMs: totalDuration,
		CursorPath:      cursorPath,
		RetryCount:      retryCount,
		FrictionSignals: signals,
		FrictionScore:   frictionScore,
	}
}

func (a *Analyzer) generateSummary(stepMetrics []contracts.StepMetrics, signals []contracts.FrictionSignal) *contracts.MetricsSummary {
	summary := &contracts.MetricsSummary{
		HighFrictionSteps:  make([]int, 0),
		SlowestSteps:       make([]int, 0),
		TopFrictionTypes:   make([]string, 0),
		RecommendedActions: make([]string, 0),
	}

	// Find high friction steps (score > 50)
	for _, sm := range stepMetrics {
		if sm.FrictionScore > 50 {
			summary.HighFrictionSteps = append(summary.HighFrictionSteps, sm.StepIndex)
		}
	}

	// Find slowest steps (top 3)
	if len(stepMetrics) > 0 {
		sorted := make([]contracts.StepMetrics, len(stepMetrics))
		copy(sorted, stepMetrics)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].TotalDurationMs > sorted[j].TotalDurationMs
		})
		limit := 3
		if len(sorted) < limit {
			limit = len(sorted)
		}
		for i := 0; i < limit; i++ {
			summary.SlowestSteps = append(summary.SlowestSteps, sorted[i].StepIndex)
		}
	}

	// Count friction types
	typeCounts := make(map[contracts.FrictionType]int)
	for _, s := range signals {
		typeCounts[s.Type]++
	}

	// Get top friction types (sorted by count)
	type typeCount struct {
		ft    contracts.FrictionType
		count int
	}
	counts := make([]typeCount, 0, len(typeCounts))
	for ft, count := range typeCounts {
		counts = append(counts, typeCount{ft, count})
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})
	for _, tc := range counts {
		summary.TopFrictionTypes = append(summary.TopFrictionTypes, string(tc.ft))
	}

	// Generate recommendations based on friction types
	if typeCounts[contracts.FrictionZigZagPath] > 0 {
		summary.RecommendedActions = append(summary.RecommendedActions,
			"Consider improving element visibility or placement to reduce cursor wandering")
	}
	if typeCounts[contracts.FrictionExcessiveTime] > 0 {
		summary.RecommendedActions = append(summary.RecommendedActions,
			"Review slow steps for performance optimization opportunities")
	}
	if typeCounts[contracts.FrictionRapidClicks] > 0 {
		summary.RecommendedActions = append(summary.RecommendedActions,
			"Check for unresponsive UI elements that may frustrate users")
	}
	if typeCounts[contracts.FrictionMultipleRetries] > 0 {
		summary.RecommendedActions = append(summary.RecommendedActions,
			"Investigate steps with multiple retries for reliability issues")
	}
	if typeCounts[contracts.FrictionLongHesitation] > 0 {
		summary.RecommendedActions = append(summary.RecommendedActions,
			"Simplify UI or add visual cues to reduce user hesitation")
	}

	return summary
}

func groupByStep(traces []contracts.InteractionTrace) map[int][]contracts.InteractionTrace {
	groups := make(map[int][]contracts.InteractionTrace)
	for _, t := range traces {
		groups[t.StepIndex] = append(groups[t.StepIndex], t)
	}
	return groups
}

// Compile-time interface check
var _ uxmetrics.Analyzer = (*Analyzer)(nil)
