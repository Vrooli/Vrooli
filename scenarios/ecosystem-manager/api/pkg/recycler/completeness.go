package recycler

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CompletenessResult represents the output from the completeness scoring system
type CompletenessResult struct {
	Scenario        string              `json:"scenario"`
	Category        string              `json:"category"`
	Score           int                 `json:"score"`
	Classification  string              `json:"classification"`
	Breakdown       CompletenessDetails `json:"breakdown"`
	Warnings        []Warning           `json:"warnings"`
	Recommendations []string            `json:"recommendations"`
}

// CompletenessDetails provides structured breakdown of the score
type CompletenessDetails struct {
	Quality  QualityMetrics  `json:"quality"`
	Coverage CoverageMetrics `json:"coverage"`
	Quantity QuantityMetrics `json:"quantity"`
	UI       UIMetrics       `json:"ui"`
}

// QualityMetrics tracks pass rates
type QualityMetrics struct {
	Score               int      `json:"score"`
	Max                 int      `json:"max"`
	RequirementPassRate PassRate `json:"requirement_pass_rate"`
	TargetPassRate      PassRate `json:"target_pass_rate"`
	TestPassRate        PassRate `json:"test_pass_rate"`
}

// PassRate represents a pass/fail ratio
type PassRate struct {
	Passing int     `json:"passing"`
	Total   int     `json:"total"`
	Rate    float64 `json:"rate"`
	Points  int     `json:"points"`
}

// CoverageMetrics tracks test coverage and depth
type CoverageMetrics struct {
	Score             int               `json:"score"`
	Max               int               `json:"max"`
	TestCoverageRatio TestCoverageRatio `json:"test_coverage_ratio"`
	DepthScore        DepthScore        `json:"depth_score"`
}

// TestCoverageRatio represents test-to-requirement ratio
type TestCoverageRatio struct {
	Ratio  float64 `json:"ratio"`
	Points int     `json:"points"`
}

// DepthScore represents requirement tree depth
type DepthScore struct {
	AvgDepth float64 `json:"avg_depth"`
	Points   int     `json:"points"`
}

// QuantityMetrics tracks absolute counts
type QuantityMetrics struct {
	Score        int          `json:"score"`
	Max          int          `json:"max"`
	Requirements QuantityItem `json:"requirements"`
	Targets      QuantityItem `json:"targets"`
	Tests        QuantityItem `json:"tests"`
}

// QuantityItem represents a single quantity metric
type QuantityItem struct {
	Count     int    `json:"count"`
	Threshold string `json:"threshold"`
	Points    int    `json:"points"`
}

// UIMetrics tracks user interface completeness
type UIMetrics struct {
	Score               int                 `json:"score"`
	Max                 int                 `json:"max"`
	TemplateCheck       TemplateCheck       `json:"template_check"`
	ComponentComplexity ComponentComplexity `json:"component_complexity"`
	APIIntegration      APIIntegration      `json:"api_integration"`
	Routing             Routing             `json:"routing"`
	CodeVolume          CodeVolume          `json:"code_volume"`
}

// TemplateCheck verifies UI is not template boilerplate
type TemplateCheck struct {
	IsTemplate bool `json:"is_template"`
	Penalty    int  `json:"penalty"`
	Points     int  `json:"points"`
}

// ComponentComplexity tracks UI component depth
type ComponentComplexity struct {
	FileCount      int    `json:"file_count"`
	ComponentCount int    `json:"component_count"`
	PageCount      int    `json:"page_count"`
	Threshold      string `json:"threshold"`
	Points         int    `json:"points"`
}

// APIIntegration tracks UI-to-API integration
type APIIntegration struct {
	EndpointCount  int `json:"endpoint_count"`
	TotalEndpoints int `json:"total_endpoints"`
	Points         int `json:"points"`
}

// Routing tracks UI routing complexity
type Routing struct {
	HasRouting bool `json:"has_routing"`
	RouteCount int  `json:"route_count"`
	Points     int  `json:"points"`
}

// CodeVolume tracks total lines of UI code
type CodeVolume struct {
	TotalLOC int     `json:"total_loc"`
	Points   float64 `json:"points"`
}

// Warning represents a completeness warning
type Warning struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Action  string `json:"action,omitempty"`
}

// getCompletenessClassification executes the completeness CLI and returns the result
func getCompletenessClassification(scenarioName string) (CompletenessResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "completeness", scenarioName, "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return CompletenessResult{}, fmt.Errorf("completeness command failed: %w (output: %s)", err, string(output))
	}

	var result CompletenessResult
	if err := json.Unmarshal(output, &result); err != nil {
		return CompletenessResult{}, fmt.Errorf("parse completeness JSON: %w", err)
	}

	return result, nil
}

// buildCompositeNote combines metrics classification with AI notes
func buildCompositeNote(metricsResult CompletenessResult, aiNote string) string {
	var builder strings.Builder

	// Metrics-driven classification prefix (replaces old AI prefix)
	builder.WriteString(getClassificationPrefix(metricsResult.Classification, metricsResult.Score))
	builder.WriteString("\n\n")

	// Completeness breakdown
	builder.WriteString(formatBreakdown(metricsResult))
	builder.WriteString("\n\n")

	// Warnings (if any)
	if len(metricsResult.Warnings) > 0 {
		builder.WriteString("**Warnings:**\n")
		for _, w := range metricsResult.Warnings {
			builder.WriteString(fmt.Sprintf("- ⚠️  %s", w.Message))
			if w.Action != "" {
				builder.WriteString(fmt.Sprintf(". %s", w.Action))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// Recommendations (actionable next steps from metrics)
	if len(metricsResult.Recommendations) > 0 {
		builder.WriteString("**Priority Actions:**\n")
		for _, rec := range metricsResult.Recommendations {
			builder.WriteString(fmt.Sprintf("- %s\n", rec))
		}
		builder.WriteString("\n")
	}

	// AI notes from last run (factual summary only, no completion judgment)
	if strings.TrimSpace(aiNote) != "" {
		builder.WriteString("**Notes from Last Run:**\n")
		builder.WriteString(aiNote)
	}

	return builder.String()
}

// getClassificationPrefix returns human-readable prefix for a classification
func getClassificationPrefix(classification string, score int) string {
	switch classification {
	case "production_ready":
		return fmt.Sprintf("**Score: %d/100** - Production ready, excellent validation coverage", score)
	case "nearly_ready":
		return fmt.Sprintf("**Score: %d/100** - Nearly ready, final polish and edge cases", score)
	case "mostly_complete":
		return fmt.Sprintf("**Score: %d/100** - Mostly complete, needs refinement and validation", score)
	case "functional_incomplete":
		return fmt.Sprintf("**Score: %d/100** - Functional but incomplete, needs more features/tests", score)
	case "foundation_laid":
		return fmt.Sprintf("**Score: %d/100** - Foundation laid, core features in progress", score)
	case "early_stage":
		return fmt.Sprintf("**Score: %d/100** - Just starting, needs significant development", score)
	default:
		return fmt.Sprintf("**Score: %d/100** - Status unclear", score)
	}
}

// formatBreakdown creates a compact breakdown summary
func formatBreakdown(result CompletenessResult) string {
	var lines []string

	// Quality summary
	lines = append(lines, fmt.Sprintf("**Quality:** %d/%d pts (Req: %.0f%%, Targets: %.0f%%, Tests: %.0f%%)",
		result.Breakdown.Quality.Score,
		result.Breakdown.Quality.Max,
		result.Breakdown.Quality.RequirementPassRate.Rate*100,
		result.Breakdown.Quality.TargetPassRate.Rate*100,
		result.Breakdown.Quality.TestPassRate.Rate*100,
	))

	// Coverage summary
	lines = append(lines, fmt.Sprintf("**Coverage:** %d/%d pts (Test ratio: %.1fx, Depth: %.1f levels)",
		result.Breakdown.Coverage.Score,
		result.Breakdown.Coverage.Max,
		result.Breakdown.Coverage.TestCoverageRatio.Ratio,
		result.Breakdown.Coverage.DepthScore.AvgDepth,
	))

	// Quantity summary
	lines = append(lines, fmt.Sprintf("**Quantity:** %d/%d pts (%d reqs [%s], %d targets [%s], %d tests [%s])",
		result.Breakdown.Quantity.Score,
		result.Breakdown.Quantity.Max,
		result.Breakdown.Quantity.Requirements.Count,
		result.Breakdown.Quantity.Requirements.Threshold,
		result.Breakdown.Quantity.Targets.Count,
		result.Breakdown.Quantity.Targets.Threshold,
		result.Breakdown.Quantity.Tests.Count,
		result.Breakdown.Quantity.Tests.Threshold,
	))

	// UI summary
	templateWarning := ""
	if result.Breakdown.UI.TemplateCheck.IsTemplate {
		templateWarning = fmt.Sprintf(", ⚠️ Template penalty: -%dpts", result.Breakdown.UI.TemplateCheck.Penalty)
	}
	lines = append(lines, fmt.Sprintf("**UI:** %d/%d pts (%d files [%s], %d components, %d routes, %d/%d API endpoints, %d LOC%s)",
		result.Breakdown.UI.Score,
		result.Breakdown.UI.Max,
		result.Breakdown.UI.ComponentComplexity.FileCount,
		result.Breakdown.UI.ComponentComplexity.Threshold,
		result.Breakdown.UI.ComponentComplexity.ComponentCount,
		result.Breakdown.UI.Routing.RouteCount,
		result.Breakdown.UI.APIIntegration.EndpointCount,
		result.Breakdown.UI.APIIntegration.TotalEndpoints,
		result.Breakdown.UI.CodeVolume.TotalLOC,
		templateWarning,
	))

	return strings.Join(lines, "\n")
}
