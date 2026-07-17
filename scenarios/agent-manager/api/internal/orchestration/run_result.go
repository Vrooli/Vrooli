package orchestration

import (
	"context"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

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
	var turns, tokens, contextTokens int
	var cost float64
	for _, evt := range events {
		switch data := evt.Data.(type) {
		case *domain.MessageEventData:
			if data.Role == "assistant" && data.Content != "" {
				turns++
			}
		case *domain.CostEventData:
			tokens = data.InputTokens + data.OutputTokens + data.CacheReadTokens + data.CacheCreationTokens
			contextTokens = data.InputTokens
			cost = data.TotalCostUSD
		}
	}
	result := domain.ResolveRunResult(latestTurnResultEvents(events), success, exitCode, terminalReason)
	return result, domain.SummaryFromRunResult(result, turns, tokens, contextTokens, cost), nil
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
