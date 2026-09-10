// This file records terminal run outcomes and result metadata.
package orchestration

import (
	"context"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"

	"github.com/google/uuid"
)

func resolvePersistedRunResult(ctx context.Context, store event.Store, runID uuid.UUID, success bool, exitCode int, terminalReason string) (*domain.RunResult, *domain.RunSummary, error) {
	if store == nil {
		result := domain.ResolveRunResult(nil, success, exitCode, terminalReason)
		return result, domain.SummaryFromRunResult(result, 0, 0, 0, 0), nil
	}
	events, err := store.Get(ctx, runID, event.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	var fallbackTurns int
	for _, evt := range events {
		if data, ok := messageEvent(evt); ok && data.Role == "assistant" && data.Content != "" && !data.EvidenceOnly {
			fallbackTurns++
		}
	}
	// Use the same owner projection as workflow accounting: terminal receipts
	// supersede interim samples and provider turns outrank message counts.
	fact := invocationreadmodel.ProjectRun(&domain.Run{ID: runID, Summary: &domain.RunSummary{TurnsUsed: fallbackTurns}}, events, time.Now())
	result := domain.ResolveRunResult(latestTurnResultEvents(events), success, exitCode, terminalReason)
	return result, domain.SummaryFromRunResult(result, int(fact.Turns), int(fact.TotalTokens), int(fact.InputTokens), fact.TotalCostUSD), nil
}

// latestTurnResultEvents prevents terminal outputs from earlier continuation
// turns competing with the current handoff. Provider turn identity defines the
// boundary; legacy streams without turn IDs retain the conservative whole-run
// resolver behavior.
func latestTurnResultEvents(events []*domain.RunEvent) []*domain.RunEvent {
	latestTurn := ""
	latestSequence := int64(-1)
	turnCandidateIDs := map[string]struct{}{}
	for _, evt := range events {
		message, ok := messageEvent(evt)
		if !ok || message.EvidenceOnly || message.Role != "assistant" || message.TurnID == "" || message.Content == "" {
			continue
		}
		if evt.Sequence > latestSequence {
			latestSequence = evt.Sequence
			latestTurn = message.TurnID
		}
	}
	if latestTurn == "" {
		return events
	}
	for _, evt := range events {
		if message, ok := messageEvent(evt); ok && !message.EvidenceOnly && message.TurnID == latestTurn {
			turnCandidateIDs[evt.ID.String()] = struct{}{}
		}
	}
	selected := make([]*domain.RunEvent, 0)
	for _, evt := range events {
		message, ok := messageEvent(evt)
		if !ok {
			continue
		}
		_, evidenceForCurrentTurn := turnCandidateIDs[message.EvidenceForEventID]
		if message.TurnID == latestTurn || (message.EvidenceOnly && evidenceForCurrentTurn) {
			selected = append(selected, evt)
		}
	}
	return selected
}

func messageEvent(evt *domain.RunEvent) (*domain.MessageEventData, bool) {
	if evt == nil {
		return nil, false
	}
	message, ok := evt.Data.(*domain.MessageEventData)
	return message, ok
}

func (o *Orchestrator) persistedResultBuilder(ctx context.Context, runID uuid.UUID, success bool, exitCode int, terminalReason string) (*domain.RunResult, *domain.RunSummary) {
	result, summary, err := resolvePersistedRunResult(ctx, o.events, runID, success, exitCode, terminalReason)
	if err != nil {
		return nil, nil
	}
	if result != nil && o.structuredResults != nil && o.runs != nil {
		if run, getErr := o.runs.Get(ctx, runID); getErr == nil && run != nil && run.ResolvedConfig != nil {
			result.Structured = o.structuredResults.Resolve(ctx, run.ResolvedConfig.ResultSpec, result)
		}
	}
	return result, summary
}
