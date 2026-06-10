package scoring

import (
	"fmt"
	"math"

	"scenario-completeness-scoring/internal/signals"
)

// Group weights (sum 100), ported from the legacy implementation.
const (
	maxQualityPoints  = 50.0
	maxCoveragePoints = 15.0
	maxQuantityPoints = 10.0
	maxUIPoints       = 25.0
)

// Quality sub-weights.
const (
	maxReqPassPoints    = 20.0
	maxTargetPassPoints = 15.0
	maxPhasePassPoints  = 15.0
)

// Coverage decisions, ported: depth target 3.0 levels; 8+7 split. The legacy
// test-to-requirement ratio signal is re-based onto declared validation
// coverage (requirements carrying at least one validation entry) — the
// cached registry knows validations, not individual test counts.
const (
	maxValidationCoveragePoints = 8.0
	maxDepthPoints              = 7.0
	optimalRequirementDepth     = 3.0
)

// Quantity sub-weights, ported.
const (
	maxReqCountPoints    = 4.0
	maxTargetCountPoints = 3.0
	maxPhaseCountPoints  = 3.0
)

// UI decisions, ported verbatim from the legacy decisions module.
const (
	templateUIPoints   = 10.0
	maxComponentPoints = 5.0
	maxAPIPoints       = 6.0
	maxRoutingPoints   = 1.5
	maxVolumePoints    = 2.5

	uiFileCountMinimum = 5
	uiFileCountLow     = 10
	uiLOCMinimum       = 100
)

// thresholdLevels is the ported ok/good/excellent band set.
type thresholdLevels struct{ OK, Good, Excellent int }

// thresholdConfig selects bands by scenario category (ported tables).
type thresholdConfig struct {
	Requirements thresholdLevels
	Targets      thresholdLevels
	Phases       thresholdLevels
	UIFileCount  thresholdLevels
	UITotalLOC   thresholdLevels
	UIEndpoints  thresholdLevels
}

// phaseCountLevels is the one new band set: distinct test-genie phases with
// recorded evidence. The quick preset runs 5 phases; a comprehensive suite
// run records ~18.
var phaseCountLevels = thresholdLevels{OK: 5, Good: 10, Excellent: 14}

func defaultThresholds() thresholdConfig {
	return thresholdConfig{
		Requirements: thresholdLevels{OK: 10, Good: 15, Excellent: 25},
		Targets:      thresholdLevels{OK: 8, Good: 12, Excellent: 20},
		Phases:       phaseCountLevels,
		UIFileCount:  thresholdLevels{OK: 15, Good: 25, Excellent: 40},
		UITotalLOC:   thresholdLevels{OK: 300, Good: 600, Excellent: 1200},
		UIEndpoints:  thresholdLevels{OK: 2, Good: 4, Excellent: 8},
	}
}

// categoryThresholds ports the legacy per-category tables (test-count bands
// dropped; the phase-count band is category-independent).
var categoryThresholds = map[string]thresholdConfig{
	"utility": defaultThresholds(),
	"business-application": {
		Requirements: thresholdLevels{OK: 25, Good: 40, Excellent: 60},
		Targets:      thresholdLevels{OK: 15, Good: 25, Excellent: 35},
		Phases:       phaseCountLevels,
		UIFileCount:  thresholdLevels{OK: 30, Good: 50, Excellent: 80},
		UITotalLOC:   thresholdLevels{OK: 1500, Good: 3000, Excellent: 6000},
		UIEndpoints:  thresholdLevels{OK: 5, Good: 10, Excellent: 20},
	},
	"automation": {
		Requirements: thresholdLevels{OK: 15, Good: 25, Excellent: 40},
		Targets:      thresholdLevels{OK: 12, Good: 18, Excellent: 28},
		Phases:       phaseCountLevels,
		UIFileCount:  thresholdLevels{OK: 20, Good: 35, Excellent: 60},
		UITotalLOC:   thresholdLevels{OK: 800, Good: 1800, Excellent: 3500},
		UIEndpoints:  thresholdLevels{OK: 4, Good: 8, Excellent: 15},
	},
	"platform": {
		Requirements: thresholdLevels{OK: 30, Good: 50, Excellent: 80},
		Targets:      thresholdLevels{OK: 20, Good: 30, Excellent: 45},
		Phases:       phaseCountLevels,
		UIFileCount:  thresholdLevels{OK: 40, Good: 70, Excellent: 120},
		UITotalLOC:   thresholdLevels{OK: 2000, Good: 4500, Excellent: 8000},
		UIEndpoints:  thresholdLevels{OK: 8, Good: 15, Excellent: 30},
	},
	"developer_tools": {
		Requirements: thresholdLevels{OK: 20, Good: 30, Excellent: 50},
		Targets:      thresholdLevels{OK: 12, Good: 20, Excellent: 30},
		Phases:       phaseCountLevels,
		UIFileCount:  thresholdLevels{OK: 25, Good: 45, Excellent: 75},
		UITotalLOC:   thresholdLevels{OK: 1000, Good: 2500, Excellent: 5000},
		UIEndpoints:  thresholdLevels{OK: 4, Good: 8, Excellent: 15},
	},
}

func thresholdsFor(category string) thresholdConfig {
	if cfg, ok := categoryThresholds[category]; ok {
		return cfg
	}
	return defaultThresholds()
}

// thresholdLevel reports the band a count falls in (ported).
func thresholdLevel(count int, t thresholdLevels) string {
	switch {
	case count >= t.Excellent:
		return "excellent"
	case count >= t.Good:
		return "good"
	case count >= t.OK:
		return "ok"
	default:
		return "below"
	}
}

// classification bands, ported verbatim.
func classify(score int) (string, string) {
	switch {
	case score >= 96:
		return "production_ready", "Production ready, excellent validation coverage"
	case score >= 81:
		return "nearly_ready", "Nearly ready, final polish and edge cases"
	case score >= 61:
		return "mostly_complete", "Mostly complete, needs refinement and validation"
	case score >= 41:
		return "functional_incomplete", "Functional but incomplete, needs more features/tests"
	case score >= 21:
		return "foundation_laid", "Foundation laid, core features in progress"
	default:
		return "early_stage", "Just starting, needs significant development"
	}
}

// computeComposite assembles the four signal groups from the snapshot.
func computeComposite(snap signals.Snapshot) Composite {
	th := thresholdsFor(snap.Category)

	quality := qualityGroup(snap)
	coverage := coverageGroup(snap)
	quantity := quantityGroup(snap, th)
	ui := uiGroup(snap, th)

	total := quality.Score + coverage.Score + quantity.Score + ui.Score
	score := int(math.Round(math.Max(0, math.Min(100, total))))
	id, label := classify(score)

	return Composite{
		Score:               score,
		Classification:      id,
		ClassificationLabel: label,
		Groups:              []Group{quality, coverage, quantity, ui},
	}
}

// phaseTally counts pass/fail evidence (skipped phases are excluded — they
// prove nothing either way).
func phaseTally(snap signals.Snapshot) (passed, counted int) {
	for _, pr := range snap.Phases.Phases {
		switch pr.Status {
		case "passed":
			passed++
			counted++
		case "failed":
			counted++
		}
	}
	return passed, counted
}

func qualityGroup(snap signals.Snapshot) Group {
	req := snap.Requirements

	reqRate := rate(req.Passing, req.Total)
	targetRate := rate(req.TargetsPassing, req.TargetsTotal)
	phasesPassed, phasesCounted := phaseTally(snap)
	phaseRate := rate(phasesPassed, phasesCounted)

	metrics := []Metric{
		{
			ID:        "requirement_pass_rate",
			Label:     "Requirements",
			Observed:  fmt.Sprintf("%d total, %d passing (%d%%)", req.Total, req.Passing, pct(reqRate)),
			Points:    round1(reqRate * maxReqPassPoints),
			MaxPoints: maxReqPassPoints,
		},
		{
			ID:        "target_pass_rate",
			Label:     "Op Targets",
			Observed:  fmt.Sprintf("%d total, %d passing (%d%%)", req.TargetsTotal, req.TargetsPassing, pct(targetRate)),
			Points:    round1(targetRate * maxTargetPassPoints),
			MaxPoints: maxTargetPassPoints,
		},
		{
			ID:        "phase_pass_rate",
			Label:     "Test Phases",
			Observed:  fmt.Sprintf("%d recorded, %d passing (%d%%)", phasesCounted, phasesPassed, pct(phaseRate)),
			Points:    round1(phaseRate * maxPhasePassPoints),
			MaxPoints: maxPhasePassPoints,
		},
	}
	return finishGroup("quality", "Quality", maxQualityPoints, metrics)
}

func coverageGroup(snap signals.Snapshot) Group {
	req := snap.Requirements

	valRate := rate(req.WithValidation, req.Total)
	depthRatio := 0.0
	if req.AvgDepth > 0 {
		depthRatio = math.Min(req.AvgDepth/optimalRequirementDepth, 1.0)
	}

	metrics := []Metric{
		{
			ID:        "validation_coverage",
			Label:     "Validation Coverage",
			Observed:  fmt.Sprintf("%d/%d requirements declare validations", req.WithValidation, req.Total),
			Points:    round1(valRate * maxValidationCoveragePoints),
			MaxPoints: maxValidationCoveragePoints,
		},
		{
			ID:        "requirement_depth",
			Label:     "Depth Score",
			Observed:  fmt.Sprintf("%.1f avg levels (target %.1f+)", req.AvgDepth, optimalRequirementDepth),
			Points:    round1(depthRatio * maxDepthPoints),
			MaxPoints: maxDepthPoints,
		},
	}
	return finishGroup("coverage", "Coverage", maxCoveragePoints, metrics)
}

func quantityGroup(snap signals.Snapshot, th thresholdConfig) Group {
	req := snap.Requirements
	_, phasesCounted := phaseTally(snap)

	metrics := []Metric{
		quantityMetric("requirements_count", "Requirements", req.Total, th.Requirements, maxReqCountPoints),
		quantityMetric("targets_count", "Targets", req.TargetsTotal, th.Targets, maxTargetCountPoints),
		quantityMetric("phases_count", "Test Phases", phasesCounted, th.Phases, maxPhaseCountPoints),
	}
	return finishGroup("quantity", "Quantity", maxQuantityPoints, metrics)
}

// quantityMetric ports the legacy pattern: points scale linearly to the
// "good" band, capped at 1.0.
func quantityMetric(id, label string, count int, t thresholdLevels, maxPts float64) Metric {
	ratio := 0.0
	if t.Good > 0 {
		ratio = math.Min(float64(count)/float64(t.Good), 1.0)
	}
	return Metric{
		ID:        id,
		Label:     label,
		Observed:  fmt.Sprintf("%d", count),
		Points:    round1(ratio * maxPts),
		MaxPoints: maxPts,
		Threshold: thresholdLevel(count, t),
	}
}

func uiGroup(snap signals.Snapshot, th thresholdConfig) Group {
	ui := snap.UI

	templatePts := templateUIPoints
	templateObserved := "custom UI"
	if ui.IsTemplate {
		templatePts = 0
		templateObserved = "TEMPLATE (replace the starter UI)"
	}
	if !ui.Collected {
		templatePts = 0
		templateObserved = "no UI sources found"
	}

	metrics := []Metric{
		{
			ID:        "template_check",
			Label:     "Template",
			Observed:  templateObserved,
			Points:    templatePts,
			MaxPoints: templateUIPoints,
		},
		{
			ID:        "component_complexity",
			Label:     "Files",
			Observed:  fmt.Sprintf("%d files", ui.FileCount),
			Points:    componentComplexityPoints(ui.FileCount, th.UIFileCount),
			MaxPoints: maxComponentPoints,
			Threshold: thresholdLevel(ui.FileCount, th.UIFileCount),
		},
		{
			ID:        "api_integration",
			Label:     "API Integration",
			Observed:  fmt.Sprintf("%d endpoints beyond /health", ui.APIBeyondHealth),
			Points:    apiIntegrationPoints(ui.APIBeyondHealth, th.UIEndpoints),
			MaxPoints: maxAPIPoints,
			Threshold: thresholdLevel(ui.APIBeyondHealth, th.UIEndpoints),
		},
		{
			ID:        "routing",
			Label:     "Routing",
			Observed:  fmt.Sprintf("%d routes", ui.RouteCount),
			Points:    routingPoints(ui.RouteCount),
			MaxPoints: maxRoutingPoints,
		},
		{
			ID:        "code_volume",
			Label:     "LOC",
			Observed:  fmt.Sprintf("%d lines", ui.TotalLOC),
			Points:    volumePoints(ui.TotalLOC, th.UITotalLOC),
			MaxPoints: maxVolumePoints,
			Threshold: thresholdLevel(ui.TotalLOC, th.UITotalLOC),
		},
	}
	return finishGroup("ui", "UI", maxUIPoints, metrics)
}

// componentComplexityPoints ports the legacy decision tree.
func componentComplexityPoints(fileCount int, t thresholdLevels) float64 {
	switch {
	case fileCount >= t.Excellent:
		return maxComponentPoints
	case fileCount >= t.Good:
		return 4
	case fileCount >= t.OK:
		return 3
	case fileCount >= uiFileCountLow:
		return 2
	case fileCount >= uiFileCountMinimum:
		return 1
	default:
		return 0
	}
}

// apiIntegrationPoints ports the legacy decision tree.
func apiIntegrationPoints(beyondHealth int, t thresholdLevels) float64 {
	switch {
	case beyondHealth >= t.Excellent:
		return maxAPIPoints
	case beyondHealth >= t.Good:
		return 5
	case beyondHealth >= t.OK:
		return 4
	case beyondHealth >= 2:
		return 3
	case beyondHealth >= 1:
		return 2
	default:
		return 0
	}
}

// routingPoints ports the legacy decision tree (1/3/5 route bands).
func routingPoints(routes int) float64 {
	switch {
	case routes >= 5:
		return maxRoutingPoints
	case routes >= 3:
		return 1.0
	case routes >= 1:
		return 0.5
	default:
		return 0
	}
}

// volumePoints ports the legacy decision tree.
func volumePoints(loc int, t thresholdLevels) float64 {
	switch {
	case loc >= t.Excellent:
		return maxVolumePoints
	case loc >= t.Good:
		return 2.0
	case loc >= t.OK:
		return 1.5
	case loc >= uiLOCMinimum:
		return 0.5
	default:
		return 0
	}
}

func finishGroup(id, label string, maxPts float64, metrics []Metric) Group {
	total := 0.0
	for _, m := range metrics {
		total += m.Points
	}
	return Group{
		ID:      id,
		Label:   label,
		Score:   math.Min(round1(total), maxPts),
		Max:     maxPts,
		Metrics: metrics,
	}
}

func rate(passing, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(passing) / float64(total)
}

func pct(r float64) int { return int(math.Round(r * 100)) }

func round1(v float64) float64 { return math.Round(v*10) / 10 }
