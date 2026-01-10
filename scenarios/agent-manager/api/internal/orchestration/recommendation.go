package orchestration

import (
	"context"
	"log"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// ExtractRecommendations extracts structured recommendations from an investigation run.
// This method now checks for cached results first (from passive background extraction).
// If cached results are available, returns them immediately without LLM call.
// If extraction is in progress, returns a status indicator.
// Falls back to sync extraction for backward compatibility if status is "none".
func (o *Orchestrator) ExtractRecommendations(ctx context.Context, runID uuid.UUID) (*domain.ExtractionResult, error) {
	// Get the run
	run, err := o.runs.Get(ctx, runID)
	if err != nil {
		return nil, err
	}

	// Validate it's an investigation run (not an apply run)
	if !run.IsInvestigationRun() {
		return nil, domain.NewValidationError("run_id", "run is not an investigation")
	}

	// Validate status is COMPLETE
	if run.Status != domain.RunStatusComplete {
		return nil, domain.NewValidationError("run_id", "investigation is not complete")
	}

	// Check for cached result from passive extraction
	switch run.RecommendationStatus {
	case domain.RecommendationStatusComplete:
		// Return cached result immediately (no LLM call needed)
		if run.RecommendationResult != nil {
			return run.RecommendationResult, nil
		}
		// Cached status but no result - fall through to sync extraction

	case domain.RecommendationStatusPending, domain.RecommendationStatusExtracting:
		// Extraction is in progress - return status indicator
		// The frontend should show loading state and wait for WebSocket update
		return &domain.ExtractionResult{
			Success:       false,
			ExtractedFrom: "pending",
			Error:         "extraction in progress",
		}, nil

	case domain.RecommendationStatusFailed:
		// Extraction failed after max retries - return error with raw text for fallback
		text, source := o.getInvestigationText(ctx, run)
		return &domain.ExtractionResult{
			Success:       false,
			RawText:       text,
			ExtractedFrom: source,
			Error:         run.RecommendationError,
		}, nil
	}

	// Fallback: sync extraction (for backward compatibility or if status is "none")
	return o.extractRecommendationsSync(ctx, run)
}

// extractRecommendationsSync performs synchronous extraction (the original behavior).
// Used for backward compatibility when passive extraction hasn't run yet.
// After successful extraction, the result is saved to the database for caching.
func (o *Orchestrator) extractRecommendationsSync(ctx context.Context, run *domain.Run) (*domain.ExtractionResult, error) {
	// Check if extractor is available
	if o.recommendationExtractor == nil {
		// No extractor configured - return fallback with raw text
		text, source := o.getInvestigationText(ctx, run)
		return &domain.ExtractionResult{
			Success:       false,
			RawText:       text,
			ExtractedFrom: source,
			Error:         "recommendation extractor not configured",
		}, nil
	}

	// Extract text from the run (prefer summary, fallback to events)
	text, source := o.getInvestigationText(ctx, run)

	// Truncate if too long (10k chars max for LLM context)
	const maxLen = 10000
	if len(text) > maxLen {
		text = text[:maxLen]
	}

	// Call extractor adapter
	result, err := o.recommendationExtractor.Extract(ctx, domain.ExtractionRequest{
		InvestigationText: text,
		MaxLength:         maxLen,
	})
	if err != nil {
		return nil, err
	}

	result.ExtractedFrom = source
	result.RawText = text

	// Save the result to the database for caching (so we don't re-extract on next modal open)
	now := time.Now()
	run.RecommendationStatus = domain.RecommendationStatusComplete
	run.RecommendationResult = result
	run.RecommendationAttempts = 1
	run.RecommendationError = ""
	run.UpdatedAt = now
	if err := o.runs.Update(ctx, run); err != nil {
		// Log but don't fail - the extraction succeeded, just caching failed
		// Next time the modal opens, it will re-extract (not ideal but functional)
		log.Printf("[recommendation] Failed to persist recommendation result for run %s: %v", run.ID, err)
	}

	// Broadcast the update so UI gets real-time notification
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(run)
	}

	return result, nil
}

// RegenerateRecommendations forces re-extraction of recommendations for an investigation run.
// This resets the extraction state and queues the run for background processing.
func (o *Orchestrator) RegenerateRecommendations(ctx context.Context, runID uuid.UUID) error {
	// Get the run
	run, err := o.runs.Get(ctx, runID)
	if err != nil {
		return err
	}

	// Validate it's an investigation run (not an apply run)
	if !run.IsInvestigationRun() {
		return domain.NewValidationError("run_id", "run is not an investigation")
	}

	// Validate status is COMPLETE
	if run.Status != domain.RunStatusComplete {
		return domain.NewValidationError("run_id", "investigation is not complete")
	}

	// Reset extraction state to trigger re-extraction
	now := time.Now()
	run.RecommendationStatus = domain.RecommendationStatusPending
	run.RecommendationAttempts = 0
	run.RecommendationError = ""
	run.RecommendationResult = nil
	run.RecommendationQueuedAt = &now
	run.UpdatedAt = now

	// Save the updated run
	if err := o.runs.Update(ctx, run); err != nil {
		return err
	}

	// Broadcast status change so UI knows extraction is queued
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(run)
	}

	return nil
}

// getInvestigationText extracts text from an investigation run.
// Returns (text, source) where source is "summary" or "events".
func (o *Orchestrator) getInvestigationText(ctx context.Context, run *domain.Run) (string, string) {
	// Try to get summary description from run metadata first
	if run.Summary != nil && run.Summary.Description != "" {
		return run.Summary.Description, "summary"
	}

	// Fallback: get events and concatenate message content
	if o.events == nil {
		return "", "events"
	}

	events, err := o.events.Get(ctx, run.ID, event.GetOptions{
		EventTypes: []domain.RunEventType{domain.EventTypeMessage},
		Limit:      100,
	})
	if err != nil || len(events) == 0 {
		return "", "events"
	}

	// Concatenate assistant messages
	var builder strings.Builder
	for _, evt := range events {
		if evt.EventType == domain.EventTypeMessage {
			if msgData, ok := evt.Data.(*domain.MessageEventData); ok {
				if msgData.Role == "assistant" {
					if builder.Len() > 0 {
						builder.WriteString("\n\n")
					}
					builder.WriteString(msgData.Content)
				}
			}
		}
	}

	return builder.String(), "events"
}
