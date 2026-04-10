// Package recommendation provides adapters for extracting structured recommendations
// from investigation outputs.
package recommendation

import (
	"context"

	"agent-manager/internal/domain"
)

// Extractor extracts structured recommendations from investigation text.
type Extractor interface {
	// Extract parses investigation text and returns categorized recommendations.
	Extract(ctx context.Context, req domain.ExtractionRequest) (*domain.ExtractionResult, error)

	// IsAvailable checks if the extraction service is available.
	IsAvailable(ctx context.Context) bool
}
