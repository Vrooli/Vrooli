package main

import "math"

// calculateHealthScore calculates a health score (0-100) based on issue severity counts.
// Weighted scoring: critical=-20, high=-10, medium=-5, low=-1
func calculateHealthScore(critical, high, medium, low int) float64 {
	totalIssues := critical + high + medium + low
	if totalIssues == 0 {
		return 100.0
	}

	weightedScore := 100.0 - (float64(critical)*20 + float64(high)*10 + float64(medium)*5 + float64(low)*1)

	if weightedScore < 0 {
		return 0.0
	}
	// Round to 1 decimal place
	return math.Round(weightedScore*10) / 10
}

// calculateSystemHealthScore calculates overall system health based on scenario and vulnerability counts.
// System health is based on the ratio of critical scenarios and total critical vulnerabilities.
func calculateSystemHealthScore(totalScenarios, criticalScenarios, criticalVulns int) float64 {
	if totalScenarios == 0 {
		return 100.0
	}

	// System health based on critical scenarios and vulnerabilities
	criticalRatio := float64(criticalScenarios) / float64(totalScenarios)
	healthScore := 100.0 - (criticalRatio*50 + float64(criticalVulns)*5)

	if healthScore < 0 {
		return 0.0
	}
	// Round to 1 decimal place
	return math.Round(healthScore*10) / 10
}
