// DOC: docs/internal/SEAMS.md#Decision-Points
//
// Package domain provides decision helpers for lifestyle score and trend calculations.
// These functions encapsulate key business decisions with configurable thresholds,
// making them easy to test, document, and modify.
//
// Decision Boundaries:
// - Direction determination (up/down/stable) for scores and domain activity
// - Data quality assessment based on active domain count
// - Score-to-message mapping for user feedback
// - Notable change detection for highlights
package domain

import "math"

// =============================================================================
// Threshold Constants (defaults, can be overridden via config)
// =============================================================================

// Default thresholds - these match the config defaults and are used
// when config is not available (e.g., in tests or simpler contexts).
const (
	// DefaultTrendThreshold is the minimum change for up/down trend (composite score)
	DefaultTrendThreshold = 5.0

	// DefaultDomainTrendThreshold is the minimum change for domain direction
	DefaultDomainTrendThreshold = 10.0

	// DefaultNotableChangeThreshold marks significant changes for highlights
	DefaultNotableChangeThreshold = 20.0

	// DefaultDataQualityGoodThreshold is minimum domains for "good" quality
	DefaultDataQualityGoodThreshold = 3

	// DefaultDataQualityLimitedThreshold is minimum domains for "limited" quality
	DefaultDataQualityLimitedThreshold = 1

	// DefaultScoreExcellent is the threshold for excellent score message
	DefaultScoreExcellent = 80

	// DefaultScoreGood is the threshold for good score message
	DefaultScoreGood = 60

	// DefaultScoreModerate is the threshold for moderate score message
	DefaultScoreModerate = 40

	// DefaultPointsPerEvent is score points per event
	DefaultPointsPerEvent = 20

	// DefaultMaxDomainScore caps per-domain score
	DefaultMaxDomainScore = 100
)

// =============================================================================
// Direction Decisions
// =============================================================================

// Direction represents a trend direction.
type Direction string

const (
	DirectionUp     Direction = "up"
	DirectionDown   Direction = "down"
	DirectionStable Direction = "stable"
)

// DetermineDirection calculates trend direction from percent change.
// Uses DefaultTrendThreshold for composite scores.
func DetermineDirection(percentChange float64) Direction {
	return DetermineDirectionWithThreshold(percentChange, DefaultTrendThreshold)
}

// DetermineDirectionWithThreshold calculates trend direction with custom threshold.
// Changes within ±threshold are considered stable.
func DetermineDirectionWithThreshold(percentChange, threshold float64) Direction {
	if math.Abs(percentChange) < threshold {
		return DirectionStable
	}
	if percentChange > 0 {
		return DirectionUp
	}
	return DirectionDown
}

// DetermineDomainDirection calculates direction for domain activity changes.
// Uses DefaultDomainTrendThreshold which is typically higher than score threshold.
func DetermineDomainDirection(percentChange float64) Direction {
	return DetermineDirectionWithThreshold(percentChange, DefaultDomainTrendThreshold)
}

// =============================================================================
// Percent Change Calculation
// =============================================================================

// CalculatePercentChange computes percent change from baseline to current.
// Returns 100 for new activity (baseline=0, current>0) and 0 for no activity.
func CalculatePercentChange(current, baseline float64) float64 {
	if baseline > 0 {
		return ((current - baseline) / baseline) * 100
	}
	if current > 0 {
		return 100 // New activity
	}
	return 0
}

// CalculatePercentChangeInt is a convenience wrapper for integer values.
func CalculatePercentChangeInt(current int, baseline float64) float64 {
	return CalculatePercentChange(float64(current), baseline)
}

// =============================================================================
// Notable Change Detection
// =============================================================================

// IsNotableChange determines if a percent change is significant enough
// to appear in highlights and recommendations.
func IsNotableChange(percentChange float64) bool {
	return IsNotableChangeWithThreshold(percentChange, DefaultNotableChangeThreshold)
}

// IsNotableChangeWithThreshold checks against a custom threshold.
func IsNotableChangeWithThreshold(percentChange, threshold float64) bool {
	return math.Abs(percentChange) > threshold
}

// =============================================================================
// Data Quality Assessment
// =============================================================================

// DataQuality represents the quality level of score data.
type DataQuality string

const (
	DataQualityGood         DataQuality = "good"
	DataQualityLimited      DataQuality = "limited"
	DataQualityInsufficient DataQuality = "insufficient"
)

// DetermineDataQuality assesses data quality based on active domain count.
func DetermineDataQuality(activeDomains int) DataQuality {
	return DetermineDataQualityWithThresholds(
		activeDomains,
		DefaultDataQualityGoodThreshold,
		DefaultDataQualityLimitedThreshold,
	)
}

// DetermineDataQualityWithThresholds allows custom thresholds.
func DetermineDataQualityWithThresholds(activeDomains, goodThreshold, limitedThreshold int) DataQuality {
	if activeDomains >= goodThreshold {
		return DataQualityGood
	}
	if activeDomains >= limitedThreshold {
		return DataQualityLimited
	}
	return DataQualityInsufficient
}

// =============================================================================
// Score Calculation
// =============================================================================

// CalculateDomainScore converts event count to a capped score.
// Each event adds PointsPerEvent points, capped at MaxDomainScore.
func CalculateDomainScore(eventCount int) int {
	return CalculateDomainScoreWithParams(eventCount, DefaultPointsPerEvent, DefaultMaxDomainScore)
}

// CalculateDomainScoreWithParams allows custom scoring parameters.
func CalculateDomainScoreWithParams(eventCount, pointsPerEvent, maxScore int) int {
	score := eventCount * pointsPerEvent
	if score > maxScore {
		return maxScore
	}
	return score
}

// =============================================================================
// Score Message Generation
// =============================================================================

// ScoreLevel represents a score tier for messaging.
type ScoreLevel string

const (
	ScoreLevelExcellent ScoreLevel = "excellent"
	ScoreLevelGood      ScoreLevel = "good"
	ScoreLevelModerate  ScoreLevel = "moderate"
	ScoreLevelLight     ScoreLevel = "light"
)

// DetermineScoreLevel categorizes a score for messaging purposes.
func DetermineScoreLevel(score int) ScoreLevel {
	return DetermineScoreLevelWithThresholds(
		score,
		DefaultScoreExcellent,
		DefaultScoreGood,
		DefaultScoreModerate,
	)
}

// DetermineScoreLevelWithThresholds allows custom tier thresholds.
func DetermineScoreLevelWithThresholds(score, excellent, good, moderate int) ScoreLevel {
	switch {
	case score >= excellent:
		return ScoreLevelExcellent
	case score >= good:
		return ScoreLevelGood
	case score >= moderate:
		return ScoreLevelModerate
	default:
		return ScoreLevelLight
	}
}

// ScoreLevelMessage returns a user-friendly message for a score level.
func ScoreLevelMessage(level ScoreLevel) string {
	switch level {
	case ScoreLevelExcellent:
		return "Excellent day!"
	case ScoreLevelGood:
		return "Good progress today."
	case ScoreLevelModerate:
		return "Moderate activity."
	default:
		return "Light activity today."
	}
}

// TrendMessage returns a user-friendly message for a trend direction.
func TrendMessage(direction Direction) string {
	switch direction {
	case DirectionUp:
		return " Trending up from yesterday!"
	case DirectionDown:
		return " Down from yesterday."
	default:
		return " Steady from yesterday."
	}
}

// =============================================================================
// Domain Status Decisions
// =============================================================================

// DomainStatus represents the stored status of a registered domain.
type DomainStatus string

const (
	// DomainStatusActive indicates the domain is responding and operational.
	DomainStatusActive DomainStatus = "active"

	// DomainStatusInactive indicates the domain has never been checked or has no health URL.
	DomainStatusInactive DomainStatus = "inactive"

	// DomainStatusUnhealthy indicates the most recent health check failed.
	DomainStatusUnhealthy DomainStatus = "unhealthy"
)

// HealthStatus represents the result of a health check (for API responses).
// Note: This differs from DomainStatus in naming to match API contract expectations.
type HealthStatus string

const (
	// HealthStatusHealthy indicates the health check succeeded.
	HealthStatusHealthy HealthStatus = "healthy"

	// HealthStatusUnhealthy indicates the health check failed.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheckResult contains both the API response status and internal domain status.
// These may differ: API returns "healthy"/"unhealthy", but domain is stored as "active"/"unhealthy".
type HealthCheckResult struct {
	// ResponseStatus is for API responses ("healthy" or "unhealthy")
	ResponseStatus HealthStatus
	// DomainStatus is for storage ("active" or "unhealthy")
	DomainStatus DomainStatus
	// Message is a human-readable explanation
	Message string
}

// DetermineHealthCheckResult determines the result of a health check.
// This is the single source of truth for health check → status mapping.
//
// Decision criteria:
// - healthError != nil → unhealthy
// - statusCode >= unhealthyThreshold (default 300) → unhealthy
// - otherwise → healthy
func DetermineHealthCheckResult(healthError error, statusCode, unhealthyThreshold int) HealthCheckResult {
	if healthError != nil {
		return HealthCheckResult{
			ResponseStatus: HealthStatusUnhealthy,
			DomainStatus:   DomainStatusUnhealthy,
			Message:        "Health check failed: " + healthError.Error(),
		}
	}
	if statusCode >= unhealthyThreshold {
		return HealthCheckResult{
			ResponseStatus: HealthStatusUnhealthy,
			DomainStatus:   DomainStatusUnhealthy,
			Message:        "Health check returned unhealthy status code",
		}
	}
	return HealthCheckResult{
		ResponseStatus: HealthStatusHealthy,
		DomainStatus:   DomainStatusActive,
		Message:        "",
	}
}

// DetermineHealthCheckResultWithDefaults uses the default unhealthy threshold (300).
func DetermineHealthCheckResultWithDefaults(healthError error, statusCode int) HealthCheckResult {
	return DetermineHealthCheckResult(healthError, statusCode, DefaultUnhealthyThreshold)
}

// DefaultUnhealthyThreshold is the HTTP status code threshold for unhealthy (300+).
const DefaultUnhealthyThreshold = 300

// HealthCheckMessageForInactive returns the message when no health URL is configured.
func HealthCheckMessageForInactive() string {
	return "no health URL configured"
}

// =============================================================================
// Highlight Generation Decisions
// =============================================================================

// HighlightType categorizes the type of highlight for UI display.
type HighlightType string

const (
	HighlightTypePositive HighlightType = "positive"
	HighlightTypeWarning  HighlightType = "warning"
	HighlightTypeInfo     HighlightType = "info"
)

// Highlight represents a notable item for the weekly digest.
type Highlight struct {
	Type    HighlightType `json:"type"`
	Message string        `json:"message"`
}

// ShouldHighlightDomainChange determines if a domain change warrants highlighting.
// Notable increases and decreases are highlighted with appropriate type.
func ShouldHighlightDomainChange(percentChange float64, direction Direction) (bool, HighlightType) {
	if !IsNotableChange(percentChange) {
		return false, ""
	}
	if direction == DirectionUp {
		return true, HighlightTypePositive
	}
	if direction == DirectionDown {
		return true, HighlightTypeWarning
	}
	return false, ""
}

// ShouldHighlightScoreImprovement determines if a score improvement is significant.
// Score improvements over 10% are highlighted.
const ScoreHighlightThreshold = 10.0

func ShouldHighlightScoreImprovement(percentChange float64, direction Direction) bool {
	return direction == DirectionUp && percentChange > ScoreHighlightThreshold
}

// GenerateFocusRecommendation creates a recommendation based on domain change.
// Declining notable domains get recovery focus, improving ones get momentum focus.
func GenerateFocusRecommendation(displayName string, percentChange float64, direction Direction) (string, bool) {
	if !IsNotableChange(percentChange) {
		return "", false
	}

	if direction == DirectionDown {
		return "Focus on " + displayName + " to recover from drop", true
	}
	if direction == DirectionUp {
		return "Keep up " + displayName + " momentum", true
	}
	return "", false
}
