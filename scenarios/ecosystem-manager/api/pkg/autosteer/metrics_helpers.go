package autosteer

// calculateMetricDeltas computes the difference between end and start metrics.
// Returns a map of metric names to their delta values.
func calculateMetricDeltas(start, end MetricsSnapshot) map[string]float64 {
	deltas := make(map[string]float64)

	// Universal metrics
	deltas["phase_loops"] = float64(end.PhaseLoops - start.PhaseLoops)
	deltas["total_loops"] = float64(end.TotalLoops - start.TotalLoops)
	deltas["build_status"] = float64(end.BuildStatus - start.BuildStatus)
	deltas["operational_targets_total"] = float64(end.OperationalTargetsTotal - start.OperationalTargetsTotal)
	deltas["operational_targets_passing"] = float64(end.OperationalTargetsPassing - start.OperationalTargetsPassing)
	deltas["operational_targets_percentage"] = end.OperationalTargetsPercentage - start.OperationalTargetsPercentage

	// UX metrics
	if end.UX != nil && start.UX != nil {
		deltas["accessibility_score"] = end.UX.AccessibilityScore - start.UX.AccessibilityScore
		deltas["ui_test_coverage"] = end.UX.UITestCoverage - start.UX.UITestCoverage
		deltas["responsive_breakpoints"] = float64(end.UX.ResponsiveBreakpoints - start.UX.ResponsiveBreakpoints)
		deltas["user_flows_implemented"] = float64(end.UX.UserFlowsImplemented - start.UX.UserFlowsImplemented)
		deltas["loading_states_count"] = float64(end.UX.LoadingStatesCount - start.UX.LoadingStatesCount)
		deltas["error_handling_coverage"] = end.UX.ErrorHandlingCoverage - start.UX.ErrorHandlingCoverage
	}

	// Test metrics
	if end.Test != nil && start.Test != nil {
		deltas["unit_test_coverage"] = end.Test.UnitTestCoverage - start.Test.UnitTestCoverage
		deltas["integration_test_coverage"] = end.Test.IntegrationTestCoverage - start.Test.IntegrationTestCoverage
		deltas["ui_test_coverage_test"] = end.Test.UITestCoverage - start.Test.UITestCoverage
		deltas["edge_cases_covered"] = float64(end.Test.EdgeCasesCovered - start.Test.EdgeCasesCovered)
		deltas["flaky_tests"] = float64(end.Test.FlakyTests - start.Test.FlakyTests)
		deltas["test_quality_score"] = end.Test.TestQualityScore - start.Test.TestQualityScore
	}

	// Refactor metrics
	if end.Refactor != nil && start.Refactor != nil {
		deltas["cyclomatic_complexity_avg"] = end.Refactor.CyclomaticComplexityAvg - start.Refactor.CyclomaticComplexityAvg
		deltas["duplication_percentage"] = end.Refactor.DuplicationPercentage - start.Refactor.DuplicationPercentage
		deltas["standards_violations"] = float64(end.Refactor.StandardsViolations - start.Refactor.StandardsViolations)
		deltas["tidiness_score"] = end.Refactor.TidinessScore - start.Refactor.TidinessScore
		deltas["tech_debt_items"] = float64(end.Refactor.TechDebtItems - start.Refactor.TechDebtItems)
	}

	// Performance metrics
	if end.Performance != nil && start.Performance != nil {
		deltas["bundle_size_kb"] = end.Performance.BundleSizeKB - start.Performance.BundleSizeKB
		deltas["initial_load_time_ms"] = float64(end.Performance.InitialLoadTimeMS - start.Performance.InitialLoadTimeMS)
		deltas["lcp_ms"] = float64(end.Performance.LCPMS - start.Performance.LCPMS)
		deltas["fid_ms"] = float64(end.Performance.FIDMS - start.Performance.FIDMS)
		deltas["cls_score"] = end.Performance.CLSScore - start.Performance.CLSScore
	}

	// Security metrics
	if end.Security != nil && start.Security != nil {
		deltas["vulnerability_count"] = float64(end.Security.VulnerabilityCount - start.Security.VulnerabilityCount)
		deltas["input_validation_coverage"] = end.Security.InputValidationCoverage - start.Security.InputValidationCoverage
		deltas["auth_implementation_score"] = end.Security.AuthImplementationScore - start.Security.AuthImplementationScore
		deltas["security_scan_score"] = end.Security.SecurityScanScore - start.Security.SecurityScanScore
	}

	return deltas
}

// calculateEffectiveness computes an effectiveness score for a phase execution.
// Score is based on: metrics improvement, iterations used vs max, and stop reason.
func calculateEffectiveness(phase PhaseExecution) float64 {
	if phase.Iterations == 0 {
		return 0.0
	}

	// Base effectiveness from metrics improvement
	deltas := calculateMetricDeltas(phase.StartMetrics, phase.EndMetrics)

	// Count positive improvements
	positiveChanges := 0
	totalMetrics := 0
	for metric, delta := range deltas {
		// Skip loop counters
		if metric == "phase_loops" || metric == "total_loops" {
			continue
		}

		totalMetrics++

		// For most metrics, positive is good
		// Exceptions: complexity, duplication, tech_debt, flaky_tests, vulnerability_count, bundle_size, load times
		isNegativeGood := metric == "cyclomatic_complexity_avg" ||
			metric == "duplication_percentage" ||
			metric == "standards_violations" ||
			metric == "tech_debt_items" ||
			metric == "flaky_tests" ||
			metric == "vulnerability_count" ||
			metric == "bundle_size_kb" ||
			metric == "initial_load_time_ms" ||
			metric == "lcp_ms" ||
			metric == "fid_ms" ||
			metric == "cls_score"

		if isNegativeGood {
			if delta < 0 {
				positiveChanges++
			}
		} else {
			if delta > 0 {
				positiveChanges++
			}
		}
	}

	if totalMetrics == 0 {
		return 0.5 // Neutral if no metrics to compare
	}

	// Calculate improvement ratio (0.0 to 1.0)
	improvementRatio := float64(positiveChanges) / float64(totalMetrics)

	// Bonus for stopping due to condition being met (efficient execution)
	stopBonus := 0.0
	if phase.StopReason == "condition_met" {
		stopBonus = 0.2
	}

	// Final effectiveness score (capped at 1.0)
	effectiveness := improvementRatio + stopBonus
	if effectiveness > 1.0 {
		effectiveness = 1.0
	}

	return effectiveness
}
