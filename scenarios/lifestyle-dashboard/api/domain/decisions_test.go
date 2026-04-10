package domain

import (
	"testing"
)

// =============================================================================
// Direction Decision Tests
// =============================================================================

func TestDetermineDirection(t *testing.T) {
	tests := []struct {
		name          string
		percentChange float64
		expected      Direction
	}{
		{"positive above threshold", 10.0, DirectionUp},
		{"positive at threshold", 5.0, DirectionUp}, // At threshold (>=5) is up
		{"negative above threshold", -10.0, DirectionDown},
		{"negative at threshold", -5.0, DirectionDown}, // At threshold (<=-5) is down
		{"small positive", 3.0, DirectionStable},
		{"small negative", -3.0, DirectionStable},
		{"zero", 0.0, DirectionStable},
		{"large positive", 100.0, DirectionUp},
		{"large negative", -100.0, DirectionDown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineDirection(tt.percentChange)
			if result != tt.expected {
				t.Errorf("DetermineDirection(%v) = %v, want %v", tt.percentChange, result, tt.expected)
			}
		})
	}
}

func TestDetermineDirectionWithThreshold(t *testing.T) {
	tests := []struct {
		name          string
		percentChange float64
		threshold     float64
		expected      Direction
	}{
		{"above custom threshold", 15.0, 10.0, DirectionUp},
		{"below custom threshold", 5.0, 10.0, DirectionStable},
		{"negative above custom threshold", -15.0, 10.0, DirectionDown},
		{"custom threshold boundary up", 10.1, 10.0, DirectionUp},
		{"custom threshold boundary down", -10.1, 10.0, DirectionDown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineDirectionWithThreshold(tt.percentChange, tt.threshold)
			if result != tt.expected {
				t.Errorf("DetermineDirectionWithThreshold(%v, %v) = %v, want %v",
					tt.percentChange, tt.threshold, result, tt.expected)
			}
		})
	}
}

func TestDetermineDomainDirection(t *testing.T) {
	// Domain threshold is 10%, higher than the default 5%
	tests := []struct {
		name          string
		percentChange float64
		expected      Direction
	}{
		{"above domain threshold", 15.0, DirectionUp},
		{"at domain threshold", 10.0, DirectionUp}, // At threshold (>=10) is up
		{"below domain threshold", 8.0, DirectionStable},
		{"negative above domain threshold", -15.0, DirectionDown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineDomainDirection(tt.percentChange)
			if result != tt.expected {
				t.Errorf("DetermineDomainDirection(%v) = %v, want %v", tt.percentChange, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Percent Change Calculation Tests
// =============================================================================

func TestCalculatePercentChange(t *testing.T) {
	tests := []struct {
		name     string
		current  float64
		baseline float64
		expected float64
	}{
		{"50% increase", 150.0, 100.0, 50.0},
		{"50% decrease", 50.0, 100.0, -50.0},
		{"no change", 100.0, 100.0, 0.0},
		{"new activity (baseline 0)", 50.0, 0.0, 100.0},
		{"no activity at all", 0.0, 0.0, 0.0},
		{"double", 200.0, 100.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePercentChange(tt.current, tt.baseline)
			if result != tt.expected {
				t.Errorf("CalculatePercentChange(%v, %v) = %v, want %v",
					tt.current, tt.baseline, result, tt.expected)
			}
		})
	}
}

func TestCalculatePercentChangeInt(t *testing.T) {
	result := CalculatePercentChangeInt(150, 100.0)
	if result != 50.0 {
		t.Errorf("CalculatePercentChangeInt(150, 100.0) = %v, want 50.0", result)
	}
}

// =============================================================================
// Notable Change Tests
// =============================================================================

func TestIsNotableChange(t *testing.T) {
	tests := []struct {
		name          string
		percentChange float64
		expected      bool
	}{
		{"notable positive", 25.0, true},
		{"notable negative", -25.0, true},
		{"not notable positive", 15.0, false},
		{"not notable negative", -15.0, false},
		{"at threshold", 20.0, false}, // Exactly at threshold is not notable
		{"just above threshold", 20.1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotableChange(tt.percentChange)
			if result != tt.expected {
				t.Errorf("IsNotableChange(%v) = %v, want %v", tt.percentChange, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Data Quality Tests
// =============================================================================

func TestDetermineDataQuality(t *testing.T) {
	tests := []struct {
		name          string
		activeDomains int
		expected      DataQuality
	}{
		{"good quality (3+ domains)", 3, DataQualityGood},
		{"good quality (many domains)", 10, DataQualityGood},
		{"limited quality (2 domains)", 2, DataQualityLimited},
		{"limited quality (1 domain)", 1, DataQualityLimited},
		{"insufficient (no domains)", 0, DataQualityInsufficient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineDataQuality(tt.activeDomains)
			if result != tt.expected {
				t.Errorf("DetermineDataQuality(%v) = %v, want %v", tt.activeDomains, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Score Calculation Tests
// =============================================================================

func TestCalculateDomainScore(t *testing.T) {
	tests := []struct {
		name       string
		eventCount int
		expected   int
	}{
		{"1 event = 20 points", 1, 20},
		{"2 events = 40 points", 2, 40},
		{"5 events = 100 points (max)", 5, 100},
		{"10 events = 100 points (capped)", 10, 100},
		{"0 events = 0 points", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateDomainScore(tt.eventCount)
			if result != tt.expected {
				t.Errorf("CalculateDomainScore(%v) = %v, want %v", tt.eventCount, result, tt.expected)
			}
		})
	}
}

func TestCalculateDomainScoreWithParams(t *testing.T) {
	// Custom: 10 points per event, max 50
	result := CalculateDomainScoreWithParams(3, 10, 50)
	if result != 30 {
		t.Errorf("CalculateDomainScoreWithParams(3, 10, 50) = %v, want 30", result)
	}

	// Custom: exceeds max
	result = CalculateDomainScoreWithParams(10, 10, 50)
	if result != 50 {
		t.Errorf("CalculateDomainScoreWithParams(10, 10, 50) = %v, want 50", result)
	}
}

// =============================================================================
// Score Level Tests
// =============================================================================

func TestDetermineScoreLevel(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		expected ScoreLevel
	}{
		{"excellent (80+)", 85, ScoreLevelExcellent},
		{"excellent boundary", 80, ScoreLevelExcellent},
		{"good (60-79)", 65, ScoreLevelGood},
		{"good boundary", 60, ScoreLevelGood},
		{"moderate (40-59)", 50, ScoreLevelModerate},
		{"moderate boundary", 40, ScoreLevelModerate},
		{"light (below 40)", 30, ScoreLevelLight},
		{"light (zero)", 0, ScoreLevelLight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineScoreLevel(tt.score)
			if result != tt.expected {
				t.Errorf("DetermineScoreLevel(%v) = %v, want %v", tt.score, result, tt.expected)
			}
		})
	}
}

func TestScoreLevelMessage(t *testing.T) {
	tests := []struct {
		level    ScoreLevel
		contains string
	}{
		{ScoreLevelExcellent, "Excellent"},
		{ScoreLevelGood, "Good"},
		{ScoreLevelModerate, "Moderate"},
		{ScoreLevelLight, "Light"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			result := ScoreLevelMessage(tt.level)
			if result == "" {
				t.Errorf("ScoreLevelMessage(%v) returned empty string", tt.level)
			}
		})
	}
}

func TestTrendMessage(t *testing.T) {
	tests := []struct {
		direction Direction
		contains  string
	}{
		{DirectionUp, "up"},
		{DirectionDown, "Down"},
		{DirectionStable, "Steady"},
	}

	for _, tt := range tests {
		t.Run(string(tt.direction), func(t *testing.T) {
			result := TrendMessage(tt.direction)
			if result == "" {
				t.Errorf("TrendMessage(%v) returned empty string", tt.direction)
			}
		})
	}
}

// =============================================================================
// Health Check Result Tests
// =============================================================================

func TestDetermineHealthCheckResult(t *testing.T) {
	tests := []struct {
		name               string
		healthError        error
		statusCode         int
		unhealthyThreshold int
		wantResponse       HealthStatus
		wantDomain         DomainStatus
		wantMessage        bool
	}{
		{
			name:               "healthy - 200 OK",
			healthError:        nil,
			statusCode:         200,
			unhealthyThreshold: 300,
			wantResponse:       HealthStatusHealthy,
			wantDomain:         DomainStatusActive,
			wantMessage:        false,
		},
		{
			name:               "healthy - 299",
			healthError:        nil,
			statusCode:         299,
			unhealthyThreshold: 300,
			wantResponse:       HealthStatusHealthy,
			wantDomain:         DomainStatusActive,
			wantMessage:        false,
		},
		{
			name:               "unhealthy - 300 redirect",
			healthError:        nil,
			statusCode:         300,
			unhealthyThreshold: 300,
			wantResponse:       HealthStatusUnhealthy,
			wantDomain:         DomainStatusUnhealthy,
			wantMessage:        true,
		},
		{
			name:               "unhealthy - 500 server error",
			healthError:        nil,
			statusCode:         500,
			unhealthyThreshold: 300,
			wantResponse:       HealthStatusUnhealthy,
			wantDomain:         DomainStatusUnhealthy,
			wantMessage:        true,
		},
		{
			name:               "unhealthy - connection error",
			healthError:        errTestConnection{},
			statusCode:         0,
			unhealthyThreshold: 300,
			wantResponse:       HealthStatusUnhealthy,
			wantDomain:         DomainStatusUnhealthy,
			wantMessage:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineHealthCheckResult(tt.healthError, tt.statusCode, tt.unhealthyThreshold)
			if result.ResponseStatus != tt.wantResponse {
				t.Errorf("ResponseStatus = %v, want %v", result.ResponseStatus, tt.wantResponse)
			}
			if result.DomainStatus != tt.wantDomain {
				t.Errorf("DomainStatus = %v, want %v", result.DomainStatus, tt.wantDomain)
			}
			hasMessage := result.Message != ""
			if hasMessage != tt.wantMessage {
				t.Errorf("Message present = %v, want %v (msg: %q)", hasMessage, tt.wantMessage, result.Message)
			}
		})
	}
}

// errTestConnection is a test error for connection failures
type errTestConnection struct{}

func (e errTestConnection) Error() string { return "connection refused" }

func TestDetermineHealthCheckResultWithDefaults(t *testing.T) {
	// Test with default threshold (300)
	result := DetermineHealthCheckResultWithDefaults(nil, 200)
	if result.ResponseStatus != HealthStatusHealthy {
		t.Errorf("Expected healthy for 200, got %v", result.ResponseStatus)
	}

	result = DetermineHealthCheckResultWithDefaults(nil, 300)
	if result.ResponseStatus != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy for 300, got %v", result.ResponseStatus)
	}
}

func TestHealthCheckMessageForInactive(t *testing.T) {
	msg := HealthCheckMessageForInactive()
	if msg != "no health URL configured" {
		t.Errorf("Expected 'no health URL configured', got %q", msg)
	}
}

// =============================================================================
// Highlight Decision Tests
// =============================================================================

func TestShouldHighlightDomainChange(t *testing.T) {
	tests := []struct {
		name          string
		percentChange float64
		direction     Direction
		shouldHigh    bool
		expectedType  HighlightType
	}{
		{"notable increase", 25.0, DirectionUp, true, HighlightTypePositive},
		{"notable decrease", -25.0, DirectionDown, true, HighlightTypeWarning},
		{"small increase", 10.0, DirectionUp, false, ""},
		{"small decrease", -10.0, DirectionDown, false, ""},
		{"notable but stable", 25.0, DirectionStable, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldHighlight, highlightType := ShouldHighlightDomainChange(tt.percentChange, tt.direction)
			if shouldHighlight != tt.shouldHigh {
				t.Errorf("shouldHighlight = %v, want %v", shouldHighlight, tt.shouldHigh)
			}
			if highlightType != tt.expectedType {
				t.Errorf("highlightType = %v, want %v", highlightType, tt.expectedType)
			}
		})
	}
}

func TestShouldHighlightScoreImprovement(t *testing.T) {
	tests := []struct {
		name          string
		percentChange float64
		direction     Direction
		expected      bool
	}{
		{"large improvement", 15.0, DirectionUp, true},
		{"at threshold", 10.0, DirectionUp, false},
		{"small improvement", 5.0, DirectionUp, false},
		{"decline", -15.0, DirectionDown, false},
		{"stable", 0.0, DirectionStable, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldHighlightScoreImprovement(tt.percentChange, tt.direction)
			if result != tt.expected {
				t.Errorf("ShouldHighlightScoreImprovement(%v, %v) = %v, want %v",
					tt.percentChange, tt.direction, result, tt.expected)
			}
		})
	}
}

func TestGenerateFocusRecommendation(t *testing.T) {
	tests := []struct {
		name          string
		displayName   string
		percentChange float64
		direction     Direction
		wantRec       bool
		wantContains  string
	}{
		{"declining domain", "Sleep", -25.0, DirectionDown, true, "Focus on Sleep"},
		{"improving domain", "Exercise", 25.0, DirectionUp, true, "Keep up Exercise"},
		{"small change", "Diet", 10.0, DirectionUp, false, ""},
		{"stable", "Mood", 5.0, DirectionStable, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, shouldAdd := GenerateFocusRecommendation(tt.displayName, tt.percentChange, tt.direction)
			if shouldAdd != tt.wantRec {
				t.Errorf("shouldAdd = %v, want %v", shouldAdd, tt.wantRec)
			}
			if tt.wantContains != "" && rec == "" {
				t.Errorf("Expected recommendation containing %q, got empty", tt.wantContains)
			}
		})
	}
}
