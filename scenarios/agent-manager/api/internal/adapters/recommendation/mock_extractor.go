package recommendation

import (
	"context"

	"agent-manager/internal/domain"
)

// MockExtractor provides controllable behavior for testing.
type MockExtractor struct {
	ExtractFunc     func(ctx context.Context, req domain.ExtractionRequest) (*domain.ExtractionResult, error)
	IsAvailableFunc func(ctx context.Context) bool
}

// NewMockExtractor creates a new mock extractor with default behavior.
func NewMockExtractor() *MockExtractor {
	return &MockExtractor{
		IsAvailableFunc: func(ctx context.Context) bool { return true },
	}
}

// Extract delegates to ExtractFunc or returns a default mock result.
func (m *MockExtractor) Extract(ctx context.Context, req domain.ExtractionRequest) (*domain.ExtractionResult, error) {
	if m.ExtractFunc != nil {
		return m.ExtractFunc(ctx, req)
	}
	// Default mock: return single category
	return &domain.ExtractionResult{
		Success: true,
		Categories: []domain.RecommendationCategory{{
			ID:   "mock-cat-1",
			Name: "Mock Recommendations",
			Recommendations: []domain.Recommendation{{
				ID:       "mock-rec-1",
				Text:     "Mock recommendation",
				Selected: true,
			}},
		}},
		ExtractedFrom: "summary",
	}, nil
}

// IsAvailable delegates to IsAvailableFunc or returns true.
func (m *MockExtractor) IsAvailable(ctx context.Context) bool {
	if m.IsAvailableFunc != nil {
		return m.IsAvailableFunc(ctx)
	}
	return true
}
