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
// This method checks for cached results first (from passive background extraction).
// If cached results are available, returns them immediately without LLM call.
// If extraction is in progress, returns a status indicator.
// If status is "none", queues the run for background extraction and returns pending.
func (o *Orchestrator) ExtractRecommendations(ctx context.Context, runID uuid.UUID) (*domain.ExtractionResult, error) {
	// Get the run
	run, err := o.runs.Get(ctx, runID)
	if err != nil {
		return nil, err
	}

	if allowed, reason := domain.CanExtractRecommendations(run, o.investigationTagAllowlist(ctx)); !allowed {
		return nil, domain.NewValidationError("run_id", reason)
	}

	// Check for cached result from passive extraction
	switch run.RecommendationStatus {
	case domain.RecommendationStatusComplete:
		// Return cached result immediately (no LLM call needed)
		if run.RecommendationResult != nil {
			return run.RecommendationResult, nil
		}
		// Cached status but no result - queue for extraction below

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

	// Status is "none" or empty - queue for background extraction
	// This avoids blocking the API request with a synchronous LLM call
	if err := o.queueRecommendationExtraction(ctx, run); err != nil {
		log.Printf("[recommendation] Failed to queue extraction for run %s: %v", run.ID, err)
		// Return error but don't fail - the background worker will eventually pick it up
	}

	return &domain.ExtractionResult{
		Success:       false,
		ExtractedFrom: "pending",
		Error:         "extraction queued",
	}, nil
}

// queueRecommendationExtraction queues a run for background recommendation extraction.
func (o *Orchestrator) queueRecommendationExtraction(ctx context.Context, run *domain.Run) error {
	now := time.Now()
	run.RecommendationStatus = domain.RecommendationStatusPending
	run.RecommendationQueuedAt = &now
	run.UpdatedAt = now

	if err := o.runs.Update(ctx, run); err != nil {
		return err
	}

	// Broadcast status change so UI knows extraction is queued
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(run)
	}

	return nil
}

// RegenerateRecommendations forces re-extraction of recommendations for an investigation run.
// This resets the extraction state and queues the run for background processing.
func (o *Orchestrator) RegenerateRecommendations(ctx context.Context, runID uuid.UUID) error {
	// Get the run
	run, err := o.runs.Get(ctx, runID)
	if err != nil {
		return err
	}

	if allowed, reason := domain.CanRegenerateRecommendations(run, o.investigationTagAllowlist(ctx)); !allowed {
		return domain.NewValidationError("run_id", reason)
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
		AfterSequence: -1,
		EventTypes:    []domain.RunEventType{domain.EventTypeMessage},
		Limit:         100,
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
