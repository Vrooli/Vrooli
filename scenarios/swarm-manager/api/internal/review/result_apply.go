package review

import (
	"encoding/json"
	"strings"
)

type reviewHandoff struct {
	Verdict                string                  `json:"verdict"`
	AgentAssessment        string                  `json:"agent_assessment"`
	Evidence               []EvidenceItem          `json:"evidence"`
	ImprovementSuggestions []ImprovementSuggestion `json:"improvement_suggestions"`
	RegressionIntroduced   bool                    `json:"regression_introduced"`
	Notes                  []string                `json:"notes"`
	Summary                string                  `json:"summary"`
	Disposition            *Disposition            `json:"disposition,omitempty"`
}

type reviewResultEnvelope struct {
	Verdict string          `json:"verdict"`
	Handoff json.RawMessage `json:"handoff"`
}

func FinalizeRoundFromResult(round *Round, result json.RawMessage, outcome string) {
	applyReviewHandoff(round, parseReviewHandoff(result))
	if isReviewSuccessOutcome(outcome) {
		round.Status = RoundStatusComplete
		*round = normalizeRound(*round)
	} else {
		round.Status = RoundStatusFailed
		if strings.TrimSpace(round.FailureReason) == "" {
			round.FailureReason = reviewAbstainReason(outcome)
		}
	}
	round.CurrentRunStatus = ""
}

func applyReviewHandoff(round *Round, handoff reviewHandoff) {
	round.Classification = strings.TrimSpace(handoff.Verdict)
	round.AgentAssessment = strings.TrimSpace(handoff.AgentAssessment)
	round.RegressionIntroduced = handoff.RegressionIntroduced
	if len(handoff.Evidence) > 0 {
		round.Evidence = handoff.Evidence
	}
	if round.Evidence == nil {
		round.Evidence = []EvidenceItem{}
	}
	if len(handoff.ImprovementSuggestions) > 0 {
		round.ImprovementSuggestions = handoff.ImprovementSuggestions
	}
	if len(handoff.Notes) > 0 {
		round.Notes = handoff.Notes
	}
	if handoff.Disposition != nil {
		round.Disposition = handoff.Disposition
	}
}

func parseReviewHandoff(raw json.RawMessage) reviewHandoff {
	if len(raw) == 0 {
		return reviewHandoff{}
	}
	var envelope reviewResultEnvelope
	if json.Unmarshal(raw, &envelope) == nil && len(envelope.Handoff) > 0 {
		var handoff reviewHandoff
		if json.Unmarshal(envelope.Handoff, &handoff) == nil {
			if strings.TrimSpace(handoff.Verdict) == "" {
				handoff.Verdict = strings.TrimSpace(envelope.Verdict)
			}
			return handoff
		}
	}
	var direct struct {
		Handoff json.RawMessage `json:"handoff"`
	}
	if json.Unmarshal(raw, &direct) == nil && len(direct.Handoff) > 0 {
		var handoff reviewHandoff
		if json.Unmarshal(direct.Handoff, &handoff) == nil {
			return handoff
		}
	}
	var handoff reviewHandoff
	_ = json.Unmarshal(raw, &handoff)
	return handoff
}

func isReviewSuccessOutcome(outcome string) bool {
	return outcome == "accepted" || outcome == "changes-requested" || outcome == "inconclusive"
}

func reviewAbstainReason(outcome string) string {
	if outcome == "failed" {
		return "review round failed"
	}
	return "review agent could not derive an honest verdict; round abstained to operator attention"
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func FindGatheringRoundByRunID(itemDir, runID string) (*Round, error) {
	rounds, err := LoadRounds(itemDir)
	if err != nil {
		return nil, err
	}
	var fallback *Round
	for i := range rounds {
		round := rounds[i]
		if runID != "" && round.RunID == runID {
			return &round, nil
		}
		if round.Status == RoundStatusGathering && (fallback == nil || round.RoundNum > fallback.RoundNum) {
			fallback = &round
		}
	}
	if runID != "" {
		return nil, nil
	}
	return fallback, nil
}
