package domain

import (
	"fmt"
	"strings"
)

const FinalOutputResolverVersion = "final-output/v1"

const (
	finalOutputRuleTerminal    = "unique_terminal_main_assistant"
	finalOutputRuleComplete    = "unique_completed_main_assistant"
	finalOutputRuleFallback    = "unique_main_assistant_fallback"
	finalOutputRuleAmbiguous   = "multiple_equally_supported_candidates"
	finalOutputRuleUnavailable = "no_viable_assistant_candidate"
)

// ResolveRunResult is the single pure final-output resolver used by live and
// recovery paths. It never selects by transcript tail position: candidates are
// ranked only by persisted provider evidence, and equal best evidence abstains.
func ResolveRunResult(events []*RunEvent, success bool, exitCode int, terminalReason string) *RunResult {
	candidates := make([]FinalOutputCandidate, 0)
	candidateByEventID := make(map[string]int)
	for _, event := range events {
		if event == nil {
			continue
		}
		message, ok := event.Data.(*MessageEventData)
		if !ok || message.EvidenceOnly || message.Role != "assistant" || strings.TrimSpace(message.Content) == "" {
			continue
		}
		candidate := FinalOutputCandidate{
			ID:                event.ID.String(),
			EventID:           event.ID.String(),
			Sequence:          event.Sequence,
			Content:           message.Content,
			MessageID:         message.MessageID,
			ConversationID:    message.ConversationID,
			TurnID:            message.TurnID,
			ProviderOrigin:    message.ProviderOrigin,
			CompletionReason:  message.CompletionReason,
			Terminal:          message.Terminal,
			ParentMessageID:   message.ParentMessageID,
			ProviderEventType: message.ProviderEventType,
			RawEvidenceRef:    message.RawEvidenceRef,
		}
		candidate.EvidenceTier = finalOutputEvidenceTier(candidate)
		candidates = append(candidates, candidate)
		candidateByEventID[candidate.EventID] = len(candidates) - 1
	}
	for _, event := range events {
		if event == nil {
			continue
		}
		evidence, ok := event.Data.(*MessageEventData)
		if !ok || !evidence.EvidenceOnly || evidence.EvidenceForEventID == "" {
			continue
		}
		index, ok := candidateByEventID[evidence.EvidenceForEventID]
		if !ok {
			continue
		}
		candidate := &candidates[index]
		candidate.Terminal = candidate.Terminal || evidence.Terminal
		if evidence.CompletionReason != "" {
			candidate.CompletionReason = evidence.CompletionReason
		}
		if evidence.ProviderEventType != "" {
			candidate.ProviderEventType = evidence.ProviderEventType
		}
		if evidence.RawEvidenceRef != "" {
			candidate.RawEvidenceRef = evidence.RawEvidenceRef
		}
		if evidence.TurnID != "" {
			candidate.TurnID = evidence.TurnID
		}
		candidate.EvidenceTier = finalOutputEvidenceTier(*candidate)
	}

	result := &RunResult{
		Candidates:     candidates,
		Success:        success,
		ExitCode:       exitCode,
		TerminalReason: terminalReason,
		Selection: FinalOutputSelection{
			Status:           FinalOutputSelectionUnavailable,
			Rule:             finalOutputRuleUnavailable,
			AlgorithmVersion: FinalOutputResolverVersion,
		},
	}
	bestTier := -1
	for _, candidate := range candidates {
		if candidate.EvidenceTier > bestTier {
			bestTier = candidate.EvidenceTier
		}
	}
	if len(candidates) == 0 || bestTier < 0 {
		return result
	}

	best := make([]FinalOutputCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.EvidenceTier == bestTier {
			best = append(best, candidate)
		}
	}
	if bestTier == 0 && len(best) == 1 {
		result.Selection.Evidence = []string{"provider message lacks terminal or completion evidence"}
		return result
	}
	if len(best) != 1 {
		result.Selection.Status = FinalOutputSelectionAmbiguous
		result.Selection.Rule = finalOutputRuleAmbiguous
		result.Selection.Evidence = []string{fmt.Sprintf("%d candidates at evidence tier %d", len(best), bestTier)}
		return result
	}

	selected := best[0]
	result.FinalOutput = selected.Content
	result.Selection.Status = FinalOutputSelectionSelected
	result.Selection.SelectedCandidateID = selected.ID
	switch bestTier {
	case 3:
		result.Selection.Rule = finalOutputRuleTerminal
	case 2:
		result.Selection.Rule = finalOutputRuleComplete
	default:
		result.Selection.Rule = finalOutputRuleFallback
	}
	result.Selection.Evidence = candidateEvidence(selected)
	return result
}

func finalOutputEvidenceTier(candidate FinalOutputCandidate) int {
	// A parent relation identifies nested/subagent output. Preserve it for
	// diagnostics but never elevate it as the main run's handoff.
	if candidate.ParentMessageID != "" || strings.EqualFold(candidate.ProviderOrigin, "child") || strings.EqualFold(candidate.ProviderOrigin, "subagent") {
		return -1
	}
	if candidate.Terminal {
		return 3
	}
	switch strings.ToLower(candidate.CompletionReason) {
	case "end_turn", "stop", "completed", "task_complete", "turn_completed", "step_finish", "response.completed":
		return 2
	default:
		if candidate.ProviderOrigin == "" {
			return 1
		}
		return 0
	}
}

func candidateEvidence(candidate FinalOutputCandidate) []string {
	evidence := make([]string, 0, 5)
	if candidate.ProviderEventType != "" {
		evidence = append(evidence, "event="+candidate.ProviderEventType)
	}
	if candidate.CompletionReason != "" {
		evidence = append(evidence, "completion_reason="+candidate.CompletionReason)
	}
	if candidate.MessageID != "" {
		evidence = append(evidence, "message_id="+candidate.MessageID)
	}
	if candidate.TurnID != "" {
		evidence = append(evidence, "turn_id="+candidate.TurnID)
	}
	if candidate.RawEvidenceRef != "" {
		evidence = append(evidence, "raw_ref="+candidate.RawEvidenceRef)
	}
	return evidence
}

// SummaryFromRunResult is the only compatibility projection for new terminal
// results. Metrics remain available even when final-output selection abstains.
func SummaryFromRunResult(result *RunResult, turns, tokens, contextTokens int, cost float64) *RunSummary {
	if result == nil && turns == 0 && tokens == 0 && contextTokens == 0 && cost == 0 {
		return nil
	}
	summary := &RunSummary{TurnsUsed: turns, TokensUsed: tokens, ContextTokens: contextTokens, CostEstimate: cost}
	if result != nil && result.Selection.Status == FinalOutputSelectionSelected {
		summary.Description = result.FinalOutput
	}
	return summary
}
