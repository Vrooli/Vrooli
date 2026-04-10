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
	screenshotsOk := screenshotsMet(dims, thresholds)
	hasTests := testsAvailable(dims)
	testsOk := !thresholds.RequireTests || (hasTests && meetsPassRate(dims.Tests, thresholds.TestMinPassRate))
	qualityOk := codeQualityMet(dims, thresholds)
	standardsOk := standardsMet(dims, thresholds)

	if screenshotsOk && testsOk && qualityOk && standardsOk {
		return ReadinessGreen
	}

	// Yellow: at least some positive signal exists.
	hasScreenshots := hasVisualScreenshots(dims)
	if hasScreenshots || hasTests || qualityOk {
		return ReadinessYellow
	}
	return ReadinessRed
}

// screenshotsMet checks if the screenshots requirement is satisfied.
func screenshotsMet(dims ReviewDimensions, thresholds ReadinessThresholds) bool {
	return !thresholds.RequireScreenshots || hasVisualScreenshots(dims)
}

// hasVisualScreenshots returns true if visual captures are available with screenshots.
func hasVisualScreenshots(dims ReviewDimensions) bool {
	return dims.Visual != nil && dims.Visual.Available && dims.Visual.ScreenshotCount > 0
}

// testsAvailable returns true if tests are available with at least one test.
func testsAvailable(dims ReviewDimensions) bool {
	return dims.Tests != nil && dims.Tests.Available && dims.Tests.Total > 0
}

// codeQualityMet checks if code quality meets the threshold.
func codeQualityMet(dims ReviewDimensions, thresholds ReadinessThresholds) bool {
	return dims.CodeQuality != nil && dims.CodeQuality.Available &&
		!dims.CodeQuality.Stale && dims.CodeQuality.Score >= thresholds.CodeQualityMinScore
}

// standardsMet checks if standards violations are within thresholds.
func standardsMet(dims ReviewDimensions, thresholds ReadinessThresholds) bool {
	if dims.Standards == nil || !dims.Standards.Available {
		return true
	}
	return dims.Standards.BlockingViolations <= thresholds.MaxBlockingViolations &&
		(thresholds.MaxWarnings < 0 || dims.Standards.Warnings <= thresholds.MaxWarnings)
}

// CalculateDimensionStatuses returns per-dimension status (green/yellow/red/skipped)
// based on the provided thresholds. This is the single source of truth for
// dimension coloring — consumers should use these statuses directly.
func CalculateDimensionStatuses(dims ReviewDimensions, thresholds ReadinessThresholds) map[string]string {
	statuses := make(map[string]string)
	statuses["codeQuality"] = codeQualityStatus(dims.CodeQuality, thresholds)
	statuses["tests"] = testsStatus(dims.Tests, thresholds)
	statuses["standards"] = standardsStatus(dims.Standards, thresholds)
	statuses["visual"] = visualStatus(dims.Visual)
	statuses["provenance"] = provenanceStatus(dims.Provenance)
	return statuses
}

// codeQualityStatus returns the status for the code quality dimension.
func codeQualityStatus(cq *CodeQualityDimension, thresholds ReadinessThresholds) string {
	if cq == nil || !cq.Available {
		return "skipped"
	}
	if cq.Stale {
		return "yellow"
	}
	if cq.Score >= thresholds.CodeQualityMinScore && cq.Violations == 0 {
		return "green"
	}
	if cq.Score >= thresholds.CodeQualityMinScore {
		return "yellow"
	}
	return "red"
}

// testsStatus returns the status for the tests dimension.
func testsStatus(tests *TestsDimension, thresholds ReadinessThresholds) string {
	if tests == nil || !tests.Available {
		return "skipped"
	}
	if tests.Total == 0 {
		return "yellow"
	}
	if meetsPassRate(tests, thresholds.TestMinPassRate) {
		return "green"
	}
	if tests.PassedCount > 0 {
		return "yellow"
	}
	return "red"
}

// standardsStatus returns the status for the standards dimension.
func standardsStatus(std *StandardsDimension, thresholds ReadinessThresholds) string {
	if std == nil || !std.Available {
		return "skipped"
	}
	if std.BlockingViolations > thresholds.MaxBlockingViolations {
		return "red"
	}
	if (thresholds.MaxWarnings >= 0 && std.Warnings > thresholds.MaxWarnings) || std.Warnings > 0 {
		return "yellow"
	}
	return "green"
}

// visualStatus returns the status for the visual dimension.
func visualStatus(vis *VisualDimension) string {
	if vis == nil || !vis.Available {
		return "skipped"
	}
	if vis.ScreenshotCount > 0 && !vis.Stale {
		return "green"
	}
	if vis.ScreenshotCount > 0 {
		return "yellow"
	}
	return "red"
}

// provenanceStatus returns the status for the provenance dimension.
func provenanceStatus(prov *ProvenanceDimension) string {
	if prov == nil || !prov.Available {
		return "skipped"
	}
	return "green"
}

// meetsPassRate checks if the test pass rate meets the required threshold.
func meetsPassRate(tests *TestsDimension, minRate float64) bool {
	if tests.Total == 0 {
		return false
	}
	rate := float64(tests.PassedCount) / float64(tests.Total)
	return rate >= minRate
}
