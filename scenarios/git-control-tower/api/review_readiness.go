package main

// CalculateReadiness derives a readiness signal from review dimensions.
//
// Logic mirrors the client-side ScenarioReviewPanel.tsx OverviewTab:
//   - Green: hasScreenshots AND hasTests AND testsPass AND qualityOk AND noBlockingViolations
//   - Yellow: any one of [hasScreenshots, hasTests, qualityOk] is true
//   - Red: none of the above
func CalculateReadiness(dims ReviewDimensions) Readiness {
	hasScreenshots := dims.Visual != nil && dims.Visual.Available && dims.Visual.ScreenshotCount > 0
	hasTests := dims.Tests != nil && dims.Tests.Available && dims.Tests.Total > 0
	testsPass := hasTests && dims.Tests.Passed
	qualityOk := dims.CodeQuality != nil && dims.CodeQuality.Available &&
		!dims.CodeQuality.Stale && dims.CodeQuality.Score >= 60
	noBlockingViolations := dims.Standards == nil || !dims.Standards.Available ||
		dims.Standards.BlockingViolations == 0

	if hasScreenshots && hasTests && testsPass && qualityOk && noBlockingViolations {
		return ReadinessGreen
	}
	if hasScreenshots || hasTests || qualityOk {
		return ReadinessYellow
	}
	return ReadinessRed
}
