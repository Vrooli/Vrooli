package scoring

import (
	"testing"
)

// [REQ:SCS-CORE-001D] Test UI dimension scoring
func TestCalculateUIScore(t *testing.T) {
	thresholds := DefaultThresholds()

	tests := []struct {
		name     string
		ui       *UIMetrics
		minScore int
		maxScore int
	}{
		{
			name:     "nil UI",
			ui:       nil,
			minScore: 0,
			maxScore: 0,
		},
		{
			name: "template UI",
			ui: &UIMetrics{
				IsTemplate:      true,
				FileCount:       10,
				APIBeyondHealth: 2,
			},
			minScore: 0,
			maxScore: 15, // Template loses 10 points
		},
		{
			name: "excellent UI",
			ui: &UIMetrics{
				IsTemplate:      false,
				FileCount:       50,
				ComponentCount:  20,
				PageCount:       5,
				APIEndpoints:    10,
				APIBeyondHealth: 10,
				HasRouting:      true,
				RouteCount:      5,
				TotalLOC:        1500,
			},
			minScore: 20,
			maxScore: 25,
		},
		{
			name: "minimal UI",
			ui: &UIMetrics{
				IsTemplate:      false,
				FileCount:       3,
				APIBeyondHealth: 0,
				RouteCount:      0,
				TotalLOC:        50,
			},
			minScore: 10,
			maxScore: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateUIScore(tt.ui, thresholds)
			if result.Score < tt.minScore || result.Score > tt.maxScore {
				t.Errorf("Expected score between %d and %d, got %d", tt.minScore, tt.maxScore, result.Score)
			}
			if result.Max != 25 {
				t.Errorf("Expected max 25, got %d", result.Max)
			}
		})
	}
}

// [REQ:SCS-CORE-001D] Test UI template detection penalty
func TestUITemplateDetection(t *testing.T) {
	thresholds := DefaultThresholds()

	templateUI := &UIMetrics{
		IsTemplate:      true,
		FileCount:       10,
		APIBeyondHealth: 2,
	}

	result := CalculateUIScore(templateUI, thresholds)

	if result.TemplateCheck.IsTemplate != true {
		t.Error("Expected IsTemplate to be true")
	}

	// Template should get 0 points for template check (10 points lost)
	if result.TemplateCheck.Points != 0 {
		t.Errorf("Expected 0 template points for template UI, got %d", result.TemplateCheck.Points)
	}

	// Non-template UI
	realUI := &UIMetrics{
		IsTemplate:      false,
		FileCount:       10,
		APIBeyondHealth: 2,
	}

	result2 := CalculateUIScore(realUI, thresholds)

	if result2.TemplateCheck.Points != 10 {
		t.Errorf("Expected 10 template points for non-template UI, got %d", result2.TemplateCheck.Points)
	}
}
