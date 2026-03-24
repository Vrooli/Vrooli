package main

// CalculateReadiness derives a readiness signal from review dimensions.
//
// Logic mirrors the client-side ScenarioReviewPanel.tsx OverviewTab:
//   - Green: hasScreenshots AND hasTests AND testsPass AND qualityOk (score >= 60, not stale)
//   - Yellow: any one of the above is true
//   - Red: none of the above
func CalculateReadiness(dims ReviewDimensions) Readiness {
	hasScreenshots := dims.Visual != nil && dims.Visual.Available && dims.Visual.ScreenshotCount > 0
	hasTests := dims.Tests != nil && dims.Tests.Available && dims.Tests.Total > 0
	testsPass := hasTests && dims.Tests.Passed
	qualityOk := dims.CodeQuality != nil && dims.CodeQuality.Available &&
		!dims.CodeQuality.Stale && dims.CodeQuality.Score >= 60

	if hasScreenshots && hasTests && testsPass && qualityOk {
		return ReadinessGreen
	}
	if hasScreenshots || hasTests || qualityOk {
		return ReadinessYellow
	}
	return ReadinessRed
}
