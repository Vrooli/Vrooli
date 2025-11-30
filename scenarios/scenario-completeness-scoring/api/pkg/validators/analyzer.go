package validators

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// ValidationInputCounts provides the aggregate counts needed for validation analysis.
// This is distinct from scoring.MetricCounts which tracks passing/total for score calculation.
// ValidationInputCounts captures the total counts of requirements and tests to analyze
// ratios and patterns in the validation approach (e.g., suspicious 1:1 test-to-requirement ratios).
type ValidationInputCounts struct {
	RequirementsTotal int
	TestsTotal        int
}

// AnalyzeValidationQuality performs comprehensive validation quality analysis
// This is the main entry point that detects all 7 anti-patterns
func AnalyzeValidationQuality(
	metrics ValidationInputCounts,
	requirements []Requirement,
	targets []OperationalTarget,
	scenarioRoot string,
) ValidationQualityAnalysis {
	var issues []ValidationIssue
	patterns := make(map[string]interface{})
	totalPenalty := 0

	penaltyParams := DefaultPenaltyParameters()
	validationConfig := DefaultValidationConfig()
	components := DetectScenarioComponents(scenarioRoot)

	// Issue 1: Insufficient test coverage (suspicious 1:1 ratio)
	issue1, penalty1 := analyzeInsufficientTestCoverage(metrics, requirements, penaltyParams, validationConfig)
	if issue1 != nil {
		issues = append(issues, *issue1)
		patterns["insufficient_test_coverage"] = issue1
		totalPenalty += penalty1
	}

	// Issue 2: Invalid test location
	issue2, penalty2 := analyzeInvalidTestLocations(requirements, metrics, scenarioRoot, components, penaltyParams)
	if issue2 != nil {
		issues = append(issues, *issue2)
		patterns["invalid_test_location"] = issue2
		totalPenalty += penalty2
	}

	// Issue 3: Monolithic test files
	issue3, penalty3 := analyzeMonolithicTestFiles(requirements, penaltyParams)
	if issue3 != nil {
		issues = append(issues, *issue3)
		patterns["monolithic_test_files"] = issue3
		totalPenalty += penalty3
	}

	// Issue 4: Ungrouped operational targets
	issue4, penalty4 := analyzeUngroupedTargets(targets, requirements, penaltyParams, validationConfig)
	if issue4 != nil {
		issues = append(issues, *issue4)
		patterns["ungrouped_operational_targets"] = issue4
		totalPenalty += penalty4
	}

	// Issue 5: Insufficient validation layers
	issue5, penalty5 := analyzeInsufficientLayers(requirements, scenarioRoot, components, penaltyParams, validationConfig)
	if issue5 != nil {
		issues = append(issues, *issue5)
		patterns["insufficient_validation_layers"] = issue5
		totalPenalty += penalty5
	}

	// Issue 6: Superficial test implementation
	issue6, penalty6 := analyzeSuperficialTests(requirements, scenarioRoot, penaltyParams)
	if issue6 != nil {
		issues = append(issues, *issue6)
		patterns["superficial_test_implementation"] = issue6
		totalPenalty += penalty6
	}

	// Issue 7: Missing test automation
	issue7, penalty7 := analyzeMissingAutomation(requirements, penaltyParams, validationConfig)
	if issue7 != nil {
		issues = append(issues, *issue7)
		patterns["missing_test_automation"] = issue7
		totalPenalty += penalty7
	}

	// Determine overall severity
	overallSeverity := "medium"
	for _, issue := range issues {
		if issue.Severity == "high" {
			overallSeverity = "high"
			break
		}
	}

	return ValidationQualityAnalysis{
		HasIssues:       len(issues) > 0,
		IssueCount:      len(issues),
		Issues:          issues,
		Patterns:        patterns,
		TotalPenalty:    totalPenalty,
		OverallSeverity: overallSeverity,
	}
}

// analyzeInsufficientTestCoverage detects suspicious 1:1 test-to-requirement ratio
func analyzeInsufficientTestCoverage(
	metrics ValidationInputCounts,
	requirements []Requirement,
	penaltyParams map[string]PenaltyParameters,
	validationConfig ValidationConfig,
) (*ValidationIssue, int) {
	config := penaltyParams["insufficient_test_coverage"]

	if metrics.RequirementsTotal == 0 {
		return nil, 0
	}

	testReqRatio := float64(metrics.TestsTotal) / float64(metrics.RequirementsTotal)
	ratioTolerance := validationConfig.AcceptableRatios.TestToRequirementTolerance

	if math.Abs(testReqRatio-1.0) < ratioTolerance {
		penalty := config.BasePenalty

		return &ValidationIssue{
			Type:     "insufficient_test_coverage",
			Severity: config.Severity,
			Detected: true,
			Penalty:  penalty,
			Ratio:    testReqRatio,
			Message: fmt.Sprintf("Suspicious 1:1 test-to-requirement ratio (%d tests for %d requirements). Expected: 1.5-2.0x ratio with diverse test sources.",
				metrics.TestsTotal, metrics.RequirementsTotal),
			Recommendation: "Add multiple test types per requirement (API tests, UI tests, e2e automation)",
			WhyItMatters:   "Each requirement should be validated by multiple test types (unit, integration, e2e) to ensure comprehensive coverage and catch different types of issues.",
			Description:    "Perfect 1:1 test-to-requirement ratios often indicate that requirements are being validated by single monolithic tests rather than comprehensive, layered testing.",
		}, penalty
	}

	return nil, 0
}

// analyzeInvalidTestLocations detects requirements referencing unsupported test/ directories
func analyzeInvalidTestLocations(
	requirements []Requirement,
	metrics ValidationInputCounts,
	scenarioRoot string,
	components ScenarioComponents,
	penaltyParams map[string]PenaltyParameters,
) (*ValidationIssue, int) {
	config := penaltyParams["invalid_test_location"]

	// Count requirements with unsupported test refs
	var invalidPaths []InvalidPathInfo
	invalidPathsMap := make(map[string][]string)

	for _, req := range requirements {
		for _, v := range req.Validation {
			ref := v.Ref
			if ref == "" {
				continue
			}
			refLower := strings.ToLower(ref)

			// Reject anything in test/ that's not test/playbooks/*.{json,yaml}
			if strings.HasPrefix(refLower, "test/") {
				isPlaybook := strings.HasPrefix(refLower, "test/playbooks/") &&
					(strings.HasSuffix(refLower, ".json") || strings.HasSuffix(refLower, ".yaml"))
				if !isPlaybook {
					invalidPathsMap[ref] = append(invalidPathsMap[ref], req.ID)
				}
			}
		}
	}

	// Convert map to slice
	for path, reqIDs := range invalidPathsMap {
		invalidPaths = append(invalidPaths, InvalidPathInfo{
			Path:           path,
			RequirementIDs: reqIDs,
		})
	}

	unsupportedRefCount := len(invalidPaths)
	if unsupportedRefCount == 0 {
		return nil, 0
	}

	ratio := float64(unsupportedRefCount) / float64(max(metrics.RequirementsTotal, 1))
	penalty := int(math.Min(math.Round(ratio*float64(config.Multiplier)), float64(config.MaxPenalty)))

	// Build component-aware recommendation
	var validSources []string
	if components.HasAPI {
		validSources = append(validSources, "api/**/*_test.go (API unit tests)")
	}
	if components.HasUI {
		validSources = append(validSources, "ui/src/**/*.test.tsx (UI unit tests)")
	}
	validSources = append(validSources, "test/playbooks/**/*.{json,yaml} (e2e automation)")

	severity := "medium"
	if ratio > config.SeverityThreshold {
		severity = "high"
	}

	return &ValidationIssue{
		Type:           "invalid_test_location",
		Severity:       severity,
		Detected:       true,
		Penalty:        penalty,
		Count:          unsupportedRefCount,
		Total:          metrics.RequirementsTotal,
		Ratio:          ratio,
		ValidSources:   validSources,
		InvalidPaths:   invalidPaths,
		Message:        fmt.Sprintf("%d/%d requirements reference unsupported test/ directories", unsupportedRefCount, metrics.RequirementsTotal),
		Recommendation: "Move validation refs to supported test sources listed above",
		WhyItMatters:   "Requirements must be validated by actual tests (API unit tests, UI component tests, or e2e automation playbooks). Infrastructure test files like test/phases/ or CLI wrapper tests in test/cli/ are not acceptable validation sources.",
		Description:    "Test files must live in recognized test locations where they can be executed as part of the test suite. Files in unsupported directories may not run during testing or may be infrastructure scripts rather than actual tests.",
	}, penalty
}

// analyzeMonolithicTestFiles detects test files that validate too many requirements
func analyzeMonolithicTestFiles(
	requirements []Requirement,
	penaltyParams map[string]PenaltyParameters,
) (*ValidationIssue, int) {
	config := penaltyParams["monolithic_test_files"]

	analysis := AnalyzeTestRefUsage(requirements)

	if len(analysis.Violations) == 0 {
		return nil, 0
	}

	penalty := int(math.Min(float64(len(analysis.Violations)*config.PerViolation), float64(config.MaxPenalty)))

	hasSevereViolations := false
	for _, v := range analysis.Violations {
		if v.Severity == "high" {
			hasSevereViolations = true
			break
		}
	}

	severity := "medium"
	if hasSevereViolations {
		severity = "high"
	}

	var worstOffender *MonolithicTestInfo
	if len(analysis.Violations) > 0 {
		worstOffender = &analysis.Violations[0]
	}

	return &ValidationIssue{
		Type:          "monolithic_test_files",
		Severity:      severity,
		Detected:      true,
		Penalty:       penalty,
		Violations:    len(analysis.Violations),
		WorstOffender: worstOffender,
		Message:       fmt.Sprintf("%d test files validate ≥4 requirements each", len(analysis.Violations)),
		Recommendation: "Break monolithic test files into focused tests for each requirement",
		WhyItMatters:   "Monolithic test files create several problems: (1) Hard to trace which requirement failed, (2) Reduces test isolation and makes debugging difficult, (3) Encourages gaming by linking many requirements to one superficial test, (4) Poor code organization and maintainability. Each requirement deserves focused validation.",
		Description:    "Gaming Prevention: This penalty detects when test files validate many requirements (≥4), which often indicates a single broad test being reused to check off multiple requirements without genuine validation. While some grouping is natural (e.g., CRUD operations), excessive sharing suggests superficial testing. The threshold of 4 requirements encourages smaller, more focused test files and better organization.",
	}, penalty
}

// analyzeUngroupedTargets detects excessive 1:1 target-to-requirement mappings
func analyzeUngroupedTargets(
	targets []OperationalTarget,
	requirements []Requirement,
	penaltyParams map[string]PenaltyParameters,
	validationConfig ValidationConfig,
) (*ValidationIssue, int) {
	config := penaltyParams["ungrouped_operational_targets"]

	analysis := AnalyzeTargetGrouping(targets, requirements)

	if len(analysis.Violations) == 0 {
		return nil, 0
	}

	penalty := int(math.Min(math.Round(analysis.OneToOneRatio*float64(config.Multiplier)), float64(config.MaxPenalty)))

	severity := "medium"
	if analysis.OneToOneRatio > config.SeverityThreshold {
		severity = "high"
	}

	acceptableRatio := validationConfig.AcceptableRatios.OneToOneTargetMapping

	return &ValidationIssue{
		Type:           "ungrouped_operational_targets",
		Severity:       severity,
		Detected:       true,
		Penalty:        penalty,
		Ratio:          analysis.OneToOneRatio,
		Count:          analysis.OneToOneCount,
		Message:        fmt.Sprintf("%d%% of operational targets have 1:1 requirement mapping (max %d%% recommended)", int(analysis.OneToOneRatio*100), int(acceptableRatio*100)),
		Recommendation: "Group related requirements under shared operational targets from the PRD",
		WhyItMatters:   "Operational targets from the PRD typically represent high-level business capabilities that encompass multiple related requirements. A 1:1 mapping suggests requirements were auto-generated from targets rather than properly decomposed.",
		Description:    "Operational targets should group related requirements under cohesive business objectives. Excessive 1:1 mappings indicate lack of proper requirement decomposition and PRD understanding.",
	}, penalty
}

// analyzeInsufficientLayers detects requirements lacking multi-layer validation
func analyzeInsufficientLayers(
	requirements []Requirement,
	scenarioRoot string,
	components ScenarioComponents,
	penaltyParams map[string]PenaltyParameters,
	validationConfig ValidationConfig,
) (*ValidationIssue, int) {
	config := penaltyParams["insufficient_validation_layers"]
	applicableLayers := GetApplicableLayers(components)

	var diversityIssues []AffectedRequirement

	for _, req := range requirements {
		criticality := DeriveRequirementCriticality(req)
		layerAnalysis := DetectValidationLayers(req, scenarioRoot)

		// Filter to only applicable AUTOMATED layers
		var applicable []string
		for _, layer := range layerAnalysis.AutomatedLayers {
			if applicableLayers[layer] {
				applicable = append(applicable, layer)
			}
		}

		minLayers := 1
		if criticality == "P0" {
			minLayers = validationConfig.MinLayers.P0Requirements
		} else if criticality == "P1" {
			minLayers = validationConfig.MinLayers.P1Requirements
		}

		if len(applicable) < minLayers {
			// Determine needed layers
			var neededLayers []string
			if components.HasAPI {
				neededLayers = append(neededLayers, "API")
			}
			reqCategory := strings.ToLower(req.Category)
			if components.HasUI && (strings.Contains(reqCategory, "ui") || strings.Contains(reqCategory, "interface")) {
				neededLayers = append(neededLayers, "UI")
			}
			if len(neededLayers) < 2 {
				neededLayers = append(neededLayers, "E2E")
			}

			diversityIssues = append(diversityIssues, AffectedRequirement{
				ID:            req.ID,
				Title:         req.Title,
				Priority:      criticality,
				CurrentLayers: applicable,
				NeededLayers:  neededLayers[:min(2, len(neededLayers))],
			})
		}
	}

	if len(diversityIssues) == 0 {
		return nil, 0
	}

	ratio := float64(len(diversityIssues)) / float64(max(len(requirements), 1))
	penalty := int(math.Min(math.Round(ratio*float64(config.Multiplier)), float64(config.MaxPenalty)))

	// Limit affected requirements to first 5
	affectedToShow := diversityIssues
	if len(affectedToShow) > 5 {
		affectedToShow = affectedToShow[:5]
	}

	hasAPIStr := "no API"
	if components.HasAPI {
		hasAPIStr = "API"
	}
	hasUIStr := "no UI"
	if components.HasUI {
		hasUIStr = "UI"
	}

	description := fmt.Sprintf("Determining applicable layers: This scenario has %s and %s components. P0/P1 requirements need 2+ AUTOMATED layers from applicable types.", hasAPIStr, hasUIStr)
	if components.HasAPI && components.HasUI {
		description += " For full-stack scenarios: validate at API + UI, API + e2e, or all three layers."
	} else if components.HasAPI {
		description += " For API-only scenarios: validate at API + e2e layers."
	} else if components.HasUI {
		description += " For UI-only scenarios: validate at UI + e2e layers."
	} else {
		description += " Add e2e validation."
	}
	description += " Manual validations are excluded as they don't provide automated regression protection."

	return &ValidationIssue{
		Type:           "insufficient_validation_layers",
		Severity:       config.Severity,
		Detected:       true,
		Penalty:        penalty,
		Count:          len(diversityIssues),
		Total:          len(requirements),
		AffectedReqs:   affectedToShow,
		Message:        fmt.Sprintf("%d critical requirements (P0/P1) lack multi-layer AUTOMATED validation", len(diversityIssues)),
		Recommendation: "Add automated validations across API, UI, and e2e layers for critical requirements (manual validations don't count toward diversity)",
		WhyItMatters:   "Multi-layer validation catches different bug types: API tests verify business logic, UI tests verify user interface behavior, and e2e tests verify complete workflows. Single-layer validation misses integration issues and edge cases. Gaming Prevention: This ensures agents don't just link requirements to the easiest/most basic tests.",
		Description:    description,
		ExtraData: map[string]interface{}{
			"has_api": components.HasAPI,
			"has_ui":  components.HasUI,
		},
	}, penalty
}

// analyzeSuperficialTests detects low-quality test files
func analyzeSuperficialTests(
	requirements []Requirement,
	scenarioRoot string,
	penaltyParams map[string]PenaltyParameters,
) (*ValidationIssue, int) {
	config := penaltyParams["superficial_test_implementation"]

	lowQualityTests := FindLowQualityTests(requirements, scenarioRoot)

	if len(lowQualityTests) == 0 {
		return nil, 0
	}

	penalty := int(math.Min(float64(len(lowQualityTests)*config.PerFile), float64(config.MaxPenalty)))

	files := make([]map[string]interface{}, len(lowQualityTests))
	for i, t := range lowQualityTests {
		files[i] = map[string]interface{}{
			"requirement":   t.Requirement,
			"ref":           t.Ref,
			"quality_score": t.QualityScore,
			"reason":        t.Reason,
		}
	}

	return &ValidationIssue{
		Type:           "superficial_test_implementation",
		Severity:       config.Severity,
		Detected:       true,
		Penalty:        penalty,
		Count:          len(lowQualityTests),
		Message:        fmt.Sprintf("%d test file(s) appear superficial (< 20 LOC, missing assertions, or no test functions)", len(lowQualityTests)),
		Recommendation: "Add meaningful test logic with assertions and edge case coverage",
		WhyItMatters:   "Tests must actually verify behavior through assertions and cover edge cases. Superficial tests that merely exist without meaningful validation provide false confidence and won't catch real bugs.",
		Description:    "Test quality matters as much as test quantity. Tests should have sufficient logic (>20 LOC typically), include assertions that verify expected behavior, and cover both happy paths and edge cases.",
		ExtraData: map[string]interface{}{
			"files": files,
		},
	}, penalty
}

// analyzeMissingAutomation detects excessive manual validation usage
func analyzeMissingAutomation(
	requirements []Requirement,
	penaltyParams map[string]PenaltyParameters,
	validationConfig ValidationConfig,
) (*ValidationIssue, int) {
	config := penaltyParams["missing_test_automation"]

	// Count total and manual validations
	totalValidations := 0
	manualValidations := 0
	for _, r := range requirements {
		totalValidations += len(r.Validation)
		for _, v := range r.Validation {
			if v.Type == "manual" {
				manualValidations++
			}
		}
	}

	// Count complete requirements with ONLY manual validation
	var completeWithManual []string
	completePattern := regexp.MustCompile(`(?i)^(complete|validated|implemented|done)$`)
	for _, req := range requirements {
		status := strings.ToLower(req.Status)
		if completePattern.MatchString(status) {
			hasManual := false
			hasAutomated := false
			for _, v := range req.Validation {
				if v.Type == "manual" {
					hasManual = true
				} else {
					hasAutomated = true
				}
			}
			if hasManual && !hasAutomated {
				completeWithManual = append(completeWithManual, req.ID)
			}
		}
	}

	manualRatio := 0.0
	if totalValidations > 0 {
		manualRatio = float64(manualValidations) / float64(totalValidations)
	}

	acceptableManualRatio := validationConfig.AcceptableRatios.ManualValidation
	completeThreshold := validationConfig.CompleteWithManualThreshold

	// Flag if: (a) >10% manual overall OR (b) ≥5 complete requirements with ONLY manual validation
	if manualRatio <= acceptableManualRatio && len(completeWithManual) < completeThreshold {
		return nil, 0
	}

	penalty := int(math.Min(
		math.Round(manualRatio*float64(config.RatioMultiplier))+
			math.Min(float64(len(completeWithManual)*config.PerCompleteReq), float64(config.MaxCompleteCount)),
		float64(config.MaxPenalty),
	))

	severity := "medium"
	if len(completeWithManual) >= int(config.SeverityThreshold) {
		severity = "high"
	}

	message := ""
	if manualRatio > acceptableManualRatio {
		message = fmt.Sprintf("%d%% of validations are manual (max %d%% recommended). Manual validations are intended as temporary measures before automated tests are implemented.",
			int(manualRatio*100), int(acceptableManualRatio*100))
	} else {
		message = fmt.Sprintf("%d requirements marked complete with ONLY manual validations. Automated tests should replace manual validations for completed requirements.",
			len(completeWithManual))
	}

	return &ValidationIssue{
		Type:           "missing_test_automation",
		Severity:       severity,
		Detected:       true,
		Penalty:        penalty,
		Ratio:          manualRatio,
		Count:          manualValidations,
		Message:        message,
		Recommendation: "Replace manual validations with automated tests (API tests, UI tests, or e2e automation). Manual validations should be temporary measures for pending/in_progress requirements.",
		WhyItMatters:   "Manual validations are not repeatable, not run in CI/CD, and provide no regression protection. Completed requirements must have automated tests to ensure they stay working as the codebase evolves.",
		Description:    "Manual validations are acceptable for in-progress work, but completed requirements must be validated by automated tests. Manual testing doesn't scale and can't catch regressions.",
		ExtraData: map[string]interface{}{
			"complete_with_manual": len(completeWithManual),
		},
	}, penalty
}

// NOTE: Go 1.21+ has built-in min() and max() functions, so we no longer need
// custom implementations. The module specifies go 1.21 in go.mod.
