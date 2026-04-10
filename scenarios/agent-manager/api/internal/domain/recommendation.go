// Package domain contains pure domain types with no infrastructure dependencies.
package domain

// Recommendation represents a single actionable item from an investigation.
type Recommendation struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Selected bool   `json:"selected"`
}

// RecommendationCategory groups related recommendations.
type RecommendationCategory struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Recommendations []Recommendation `json:"recommendations"`
}

// ExtractionResult contains the outcome of recommendation extraction.
type ExtractionResult struct {
	Success       bool                     `json:"success"`
	Categories    []RecommendationCategory `json:"categories"`
	RawText       string                   `json:"rawText"`       // Original text for fallback
	ExtractedFrom string                   `json:"extractedFrom"` // "summary" or "events"
	Error         string                   `json:"error,omitempty"`
}

// ExtractionRequest contains parameters for extraction.
type ExtractionRequest struct {
	InvestigationText string
	MaxLength         int // Truncate input if exceeds
}
