package orchestration

import (
	"context"
	"strings"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// ExtractRecommendations extracts structured recommendations from an investigation run.
// This method:
// 1. Validates the run is a complete investigation run
// 2. Extracts text from the run (summary or events)
// 3. Calls the recommendation extractor adapter
// 4. Returns categorized recommendations for the UI
func (o *Orchestrator) ExtractRecommendations(ctx context.Context, runID uuid.UUID) (*domain.ExtractionResult, error) {
	// Get the run
	run, err := o.runs.Get(ctx, runID)
	if err != nil {
		return nil, err
	}

	// Validate it's an investigation run
	if !strings.HasPrefix(run.Tag, "agent-manager-investigation") {
		return nil, domain.NewValidationError("run_id", "run is not an investigation")
	}

	// Validate status is COMPLETE
	if run.Status != domain.RunStatusComplete {
		return nil, domain.NewValidationError("run_id", "investigation is not complete")
	}

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
	return result, nil
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
