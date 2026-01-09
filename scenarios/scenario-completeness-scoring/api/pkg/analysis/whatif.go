// Package analysis provides what-if analysis and bulk operations for scenario scoring.
// [REQ:SCS-ANALYSIS-001] What-if analysis implementation
package analysis

import (
	"fmt"

	"scenario-completeness-scoring/pkg/collectors"
	"scenario-completeness-scoring/pkg/scoring"
)

// WhatIfRequest represents a hypothetical change to simulate
// [REQ:SCS-ANALYSIS-001] What-if analysis request structure
type WhatIfRequest struct {
	Changes []MetricChange `json:"changes"`
}

// MetricChange represents a single metric change to simulate
type MetricChange struct {
	Component string  `json:"component"` // e.g., "quality.test_pass_rate", "ui.template", "quantity.tests"
	NewValue  float64 `json:"new_value"` // New value for the metric
}

// WhatIfResult represents the result of a what-if analysis
type WhatIfResult struct {
	CurrentScore      int                    `json:"current_score"`
	ProjectedScore    int                    `json:"projected_score"`
	Delta             int                    `json:"delta"`
	CurrentClass      string                 `json:"current_classification"`
	ProjectedClass    string                 `json:"projected_classification"`
	ClassificationChanged bool               `json:"classification_changed"`
	ChangesApplied    []AppliedChange        `json:"changes_applied"`
	Breakdown         scoring.ScoreBreakdown `json:"breakdown"`
}

// AppliedChange describes a change that was applied in simulation
type AppliedChange struct {
	Component  string  `json:"component"`
	OldValue   float64 `json:"old_value"`
	NewValue   float64 `json:"new_value"`
	Impact     int     `json:"impact"`
}

// WhatIfAnalyzer performs what-if analysis on scenario scores
type WhatIfAnalyzer struct {
	collector *collectors.MetricsCollector
}

// NewWhatIfAnalyzer creates a new what-if analyzer
func NewWhatIfAnalyzer(collector *collectors.MetricsCollector) *WhatIfAnalyzer {
	return &WhatIfAnalyzer{
		collector: collector,
	}
}

// Analyze performs what-if analysis for a scenario
// [REQ:SCS-ANALYSIS-001] What-if analysis: simulate score changes
func (w *WhatIfAnalyzer) Analyze(scenarioName string, changes []MetricChange) (*WhatIfResult, error) {
	// Get current metrics
	metrics, err := w.collector.Collect(scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics for %s: %w", scenarioName, err)
	}

	thresholds := scoring.GetThresholds(metrics.Category)

	// Calculate current score (no validation penalty for what-if analysis)
	currentBreakdown := scoring.CalculateCompletenessScore(*metrics, thresholds, 0)

	// Create a copy of metrics to simulate changes
	simulatedMetrics := *metrics
	if metrics.UI != nil {
		uiCopy := *metrics.UI
		simulatedMetrics.UI = &uiCopy
	}

	// Track applied changes
	var appliedChanges []AppliedChange

	// Apply each change
	for _, change := range changes {
		oldValue, newValue, applied := w.applyChange(&simulatedMetrics, change)
		if applied {
			appliedChanges = append(appliedChanges, AppliedChange{
				Component: change.Component,
				OldValue:  oldValue,
				NewValue:  newValue,
			})
		}
	}

	// Calculate projected score (no validation penalty for what-if analysis)
	projectedBreakdown := scoring.CalculateCompletenessScore(simulatedMetrics, thresholds, 0)

	// Calculate impact for each change
	for i := range appliedChanges {
		// Rough estimate of impact based on component weights
		appliedChanges[i].Impact = estimateImpact(appliedChanges[i].Component,
			appliedChanges[i].OldValue, appliedChanges[i].NewValue)
	}

	return &WhatIfResult{
		CurrentScore:         currentBreakdown.Score,
		ProjectedScore:       projectedBreakdown.Score,
		Delta:                projectedBreakdown.Score - currentBreakdown.Score,
		CurrentClass:         currentBreakdown.Classification,
		ProjectedClass:       projectedBreakdown.Classification,
		ClassificationChanged: currentBreakdown.Classification != projectedBreakdown.Classification,
		ChangesApplied:       appliedChanges,
		Breakdown:            projectedBreakdown,
	}, nil
}

// applyChange applies a single metric change and returns (oldValue, newValue, wasApplied)
func (w *WhatIfAnalyzer) applyChange(metrics *scoring.Metrics, change MetricChange) (float64, float64, bool) {
	switch change.Component {
	// Quality changes
	case "quality.requirement_pass_rate":
		if metrics.Requirements.Total > 0 {
			oldRate := float64(metrics.Requirements.Passing) / float64(metrics.Requirements.Total)
			metrics.Requirements.Passing = int(change.NewValue * float64(metrics.Requirements.Total))
			return oldRate, change.NewValue, true
		}
	case "quality.target_pass_rate":
		if metrics.Targets.Total > 0 {
			oldRate := float64(metrics.Targets.Passing) / float64(metrics.Targets.Total)
			metrics.Targets.Passing = int(change.NewValue * float64(metrics.Targets.Total))
			return oldRate, change.NewValue, true
		}
	case "quality.test_pass_rate":
		if metrics.Tests.Total > 0 {
			oldRate := float64(metrics.Tests.Passing) / float64(metrics.Tests.Total)
			metrics.Tests.Passing = int(change.NewValue * float64(metrics.Tests.Total))
			return oldRate, change.NewValue, true
		}

	// Quantity changes (add more)
	case "quantity.requirements":
		oldCount := float64(metrics.Requirements.Total)
		metrics.Requirements.Total = int(change.NewValue)
		// Maintain pass rate by adjusting passing count
		if oldCount > 0 {
			passRate := float64(metrics.Requirements.Passing) / oldCount
			metrics.Requirements.Passing = int(passRate * change.NewValue)
		}
		return oldCount, change.NewValue, true
	case "quantity.targets":
		oldCount := float64(metrics.Targets.Total)
		metrics.Targets.Total = int(change.NewValue)
		if oldCount > 0 {
			passRate := float64(metrics.Targets.Passing) / oldCount
			metrics.Targets.Passing = int(passRate * change.NewValue)
		}
		return oldCount, change.NewValue, true
	case "quantity.tests":
		oldCount := float64(metrics.Tests.Total)
		metrics.Tests.Total = int(change.NewValue)
		if oldCount > 0 {
			passRate := float64(metrics.Tests.Passing) / oldCount
			metrics.Tests.Passing = int(passRate * change.NewValue)
		}
		return oldCount, change.NewValue, true

	// UI changes
	case "ui.template":
		if metrics.UI != nil {
			oldValue := 0.0
			if metrics.UI.IsTemplate {
				oldValue = 1.0
			}
			metrics.UI.IsTemplate = change.NewValue > 0.5
			return oldValue, change.NewValue, true
		}
	case "ui.file_count":
		if metrics.UI != nil {
			oldCount := float64(metrics.UI.FileCount)
			metrics.UI.FileCount = int(change.NewValue)
			return oldCount, change.NewValue, true
		}
	case "ui.api_endpoints":
		if metrics.UI != nil {
			oldCount := float64(metrics.UI.APIBeyondHealth)
			metrics.UI.APIBeyondHealth = int(change.NewValue)
			return oldCount, change.NewValue, true
		}
	case "ui.route_count":
		if metrics.UI != nil {
			oldCount := float64(metrics.UI.RouteCount)
			metrics.UI.RouteCount = int(change.NewValue)
			metrics.UI.HasRouting = change.NewValue > 0
			return oldCount, change.NewValue, true
		}
	case "ui.loc":
		if metrics.UI != nil {
			oldLOC := float64(metrics.UI.TotalLOC)
			metrics.UI.TotalLOC = int(change.NewValue)
			return oldLOC, change.NewValue, true
		}
	}

	return 0, 0, false
}

// estimateImpact estimates the point impact of a change
func estimateImpact(component string, oldValue, newValue float64) int {
	switch component {
	case "ui.template":
		if oldValue > 0.5 && newValue <= 0.5 {
			return 10 // Template to non-template is 10 points
		}
		return 0
	case "quality.requirement_pass_rate":
		return int((newValue-oldValue)*20 + 0.5) // 20 points max, round properly
	case "quality.target_pass_rate":
		return int((newValue-oldValue)*15 + 0.5) // 15 points max, round properly
	case "quality.test_pass_rate":
		return int((newValue-oldValue)*15 + 0.5) // 15 points max, round properly
	case "quantity.tests", "quantity.requirements", "quantity.targets":
		// Rough estimate based on threshold changes
		return int((newValue-oldValue)/10 + 0.5)
	default:
		return 0
	}
}

// AvailableComponents returns the list of components that can be changed
func AvailableComponents() []ComponentInfo {
	return []ComponentInfo{
		{Name: "quality.requirement_pass_rate", Description: "Requirement pass rate (0.0-1.0)", MaxPoints: 20, Category: "quality"},
		{Name: "quality.target_pass_rate", Description: "Operational target pass rate (0.0-1.0)", MaxPoints: 15, Category: "quality"},
		{Name: "quality.test_pass_rate", Description: "Test pass rate (0.0-1.0)", MaxPoints: 15, Category: "quality"},
		{Name: "quantity.requirements", Description: "Total requirement count", MaxPoints: 4, Category: "quantity"},
		{Name: "quantity.targets", Description: "Total target count", MaxPoints: 3, Category: "quantity"},
		{Name: "quantity.tests", Description: "Total test count", MaxPoints: 3, Category: "quantity"},
		{Name: "ui.template", Description: "Is template UI (0=no, 1=yes)", MaxPoints: 10, Category: "ui"},
		{Name: "ui.file_count", Description: "UI file count", MaxPoints: 5, Category: "ui"},
		{Name: "ui.api_endpoints", Description: "API endpoints beyond /health", MaxPoints: 6, Category: "ui"},
		{Name: "ui.route_count", Description: "Number of UI routes", MaxPoints: 2, Category: "ui"},
		{Name: "ui.loc", Description: "Total lines of code in UI", MaxPoints: 3, Category: "ui"},
	}
}

// ComponentInfo describes a component that can be changed in what-if analysis
type ComponentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MaxPoints   int    `json:"max_points"`
	Category    string `json:"category"`
}
