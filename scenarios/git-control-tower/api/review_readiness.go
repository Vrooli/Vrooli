package main

// ReadinessThresholds configures the criteria used by CalculateReadiness and
// CalculateDimensionStatuses to determine readiness signals.
type ReadinessThresholds struct {
	CodeQualityMinScore   float64 `json:"codeQualityMinScore"`
	TestMinPassRate       float64 `json:"testMinPassRate"` // 0.0-1.0
	MaxBlockingViolations int     `json:"maxBlockingViolations"`
	MaxWarnings           int     `json:"maxWarnings"` // -1 = unlimited
	RequireScreenshots    bool    `json:"requireScreenshots"`
	RequireTests          bool    `json:"requireTests"`
}

// DefaultReadinessThresholds returns the baseline thresholds that match the
// original hardcoded behavior.
func DefaultReadinessThresholds() ReadinessThresholds {
	return ReadinessThresholds{
		CodeQualityMinScore:   60,
		TestMinPassRate:       1.0,
		MaxBlockingViolations: 0,
		MaxWarnings:           -1,
		RequireScreenshots:    true,
		RequireTests:          true,
	}
}

// CalculateReadiness derives a readiness signal from review dimensions using
// the provided thresholds.
//
// Logic:
//   - Green: all configured requirements are met
//   - Yellow: at least partial progress on required dimensions
//   - Red: no meaningful progress
func CalculateReadiness(dims ReviewDimensions, thresholds ReadinessThresholds) Readiness {
	screenshotsOk := !thresholds.RequireScreenshots ||
		(dims.Visual != nil && dims.Visual.Available && dims.Visual.ScreenshotCount > 0)

	hasTests := dims.Tests != nil && dims.Tests.Available && dims.Tests.Total > 0
	testsOk := !thresholds.RequireTests || (hasTests && meetsPassRate(dims.Tests, thresholds.TestMinPassRate))

	qualityOk := dims.CodeQuality != nil && dims.CodeQuality.Available &&
		!dims.CodeQuality.Stale && dims.CodeQuality.Score >= thresholds.CodeQualityMinScore

	standardsOk := dims.Standards == nil || !dims.Standards.Available ||
		(dims.Standards.BlockingViolations <= thresholds.MaxBlockingViolations &&
			(thresholds.MaxWarnings < 0 || dims.Standards.Warnings <= thresholds.MaxWarnings))

	if screenshotsOk && testsOk && qualityOk && standardsOk {
		return ReadinessGreen
	}

	// Yellow: at least some positive signal exists.
	hasScreenshots := dims.Visual != nil && dims.Visual.Available && dims.Visual.ScreenshotCount > 0
	hasPartialTests := hasTests
	if hasScreenshots || hasPartialTests || qualityOk {
		return ReadinessYellow
	}
	return ReadinessRed
}

// CalculateDimensionStatuses returns per-dimension status (green/yellow/red/skipped)
// based on the provided thresholds. This is the single source of truth for
// dimension coloring — consumers should use these statuses directly.
func CalculateDimensionStatuses(dims ReviewDimensions, thresholds ReadinessThresholds) map[string]string {
	statuses := make(map[string]string)

	// Code quality.
	if dims.CodeQuality == nil || !dims.CodeQuality.Available {
		statuses["codeQuality"] = "skipped"
	} else if dims.CodeQuality.Stale {
		statuses["codeQuality"] = "yellow"
	} else if dims.CodeQuality.Score >= thresholds.CodeQualityMinScore && dims.CodeQuality.Violations == 0 {
		statuses["codeQuality"] = "green"
	} else if dims.CodeQuality.Score >= thresholds.CodeQualityMinScore {
		statuses["codeQuality"] = "yellow"
	} else {
		statuses["codeQuality"] = "red"
	}

	// Tests.
	if dims.Tests == nil || !dims.Tests.Available {
		statuses["tests"] = "skipped"
	} else if dims.Tests.Total == 0 {
		statuses["tests"] = "yellow"
	} else if meetsPassRate(dims.Tests, thresholds.TestMinPassRate) {
		statuses["tests"] = "green"
	} else if dims.Tests.PassedCount > 0 {
		statuses["tests"] = "yellow"
	} else {
		statuses["tests"] = "red"
	}

	// Standards.
	if dims.Standards == nil || !dims.Standards.Available {
		statuses["standards"] = "skipped"
	} else if dims.Standards.BlockingViolations > thresholds.MaxBlockingViolations {
		statuses["standards"] = "red"
	} else if thresholds.MaxWarnings >= 0 && dims.Standards.Warnings > thresholds.MaxWarnings {
		statuses["standards"] = "yellow"
	} else if dims.Standards.Warnings > 0 {
		statuses["standards"] = "yellow"
	} else {
		statuses["standards"] = "green"
	}

	// Visual.
	if dims.Visual == nil || !dims.Visual.Available {
		statuses["visual"] = "skipped"
	} else if dims.Visual.ScreenshotCount > 0 && !dims.Visual.Stale {
		statuses["visual"] = "green"
	} else if dims.Visual.ScreenshotCount > 0 {
		statuses["visual"] = "yellow"
	} else {
		statuses["visual"] = "red"
	}

	// Provenance (informational — does not affect overall readiness).
	if dims.Provenance == nil || !dims.Provenance.Available {
		statuses["provenance"] = "skipped"
	} else {
		statuses["provenance"] = "green"
	}

	return statuses
}

// meetsPassRate checks if the test pass rate meets the required threshold.
func meetsPassRate(tests *TestsDimension, minRate float64) bool {
	if tests.Total == 0 {
		return false
	}
	rate := float64(tests.PassedCount) / float64(tests.Total)
	return rate >= minRate
}
