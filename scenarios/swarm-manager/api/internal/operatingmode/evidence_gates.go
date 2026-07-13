package operatingmode

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/evidence"
)

type evidenceGateResult struct {
	Pending bool
	Missing bool
	Reason  string
}

// evaluateEvidenceRequirements applies the pinned phase requirements to the
// canonical ledger. Requirements without a named producer can be satisfied by
// any producer, but remain pending on absence because no bounded producer set
// can establish a definitive negative result.
func (s *Service) evaluateEvidenceRequirements(ctx context.Context, round RoundEnvelope) (evidenceGateResult, error) {
	bundle, _, err := s.definitionBundleForRound(round)
	if err != nil {
		return evidenceGateResult{}, err
	}
	phase, err := bundle.Definition(Mode(round.Mode))
	if err != nil {
		return evidenceGateResult{}, err
	}
	phaseDefinition, err := phase.PhaseDefinition(Phase(round.Phase))
	if err != nil {
		return evidenceGateResult{}, err
	}
	if len(phaseDefinition.EvidenceRequirements) == 0 {
		return evidenceGateResult{}, nil
	}
	if s.evidenceService == nil {
		return evidenceGateResult{Pending: true, Reason: "evidence service is unavailable"}, nil
	}
	if strings.TrimSpace(round.ExecutionID) == "" || strings.TrimSpace(round.RunID) == "" {
		return evidenceGateResult{Pending: true, Reason: "round lacks immutable execution or run identity for evidence lookup"}, nil
	}
	owner := evidence.Owner{Kind: evidence.OwnerOperatingModeExecution, ID: round.ExecutionID, Round: round.Round}
	var pending []string
	for _, requirement := range phaseDefinition.EvidenceRequirements {
		result, err := s.evidenceService.EvaluateRequirement(ctx, owner, round.RunID, requirement.LedgerRequirement())
		if err != nil {
			return evidenceGateResult{}, err
		}
		switch result.State {
		case evidence.RequirementSatisfied:
			continue
		case evidence.RequirementMissing:
			return evidenceGateResult{Missing: true, Reason: evidenceRequirementReason(requirement, result.Reason)}, nil
		case evidence.RequirementPending:
			pending = append(pending, evidenceRequirementReason(requirement, result.Reason))
		default:
			return evidenceGateResult{}, fmt.Errorf("unknown evidence requirement state %q", result.State)
		}
	}
	if len(pending) > 0 {
		return evidenceGateResult{Pending: true, Reason: strings.Join(pending, "; ")}, nil
	}
	return evidenceGateResult{}, nil
}

func evidenceRequirementReason(requirement EvidenceRequirement, detail string) string {
	identity := requirement.SubjectKind + "." + requirement.Action
	if requirement.ProducerID != "" {
		identity += " from " + requirement.ProducerID
	}
	if detail == "" {
		return "evidence requirement " + identity
	}
	return "evidence requirement " + identity + ": " + detail
}

func (s *Service) refreshPendingEvidence(ctx context.Context, initiativeName string, mode Mode, number int, round RoundEnvelope) (RoundEnvelope, error) {
	gate, err := s.evaluateEvidenceRequirements(ctx, round)
	if err != nil {
		round.Error = "evidence evaluation pending: " + err.Error()
		MutableRoundPayload(&round).SetEvidenceGateState("pending")
	} else if gate.Pending {
		round.Error = gate.Reason
		MutableRoundPayload(&round).SetEvidenceGateState("pending")
	} else if gate.Missing {
		round.Status = RoundStatusNeedsAttention
		round.Error = gate.Reason
		MutableRoundPayload(&round).SetEvidenceGateState("missing")
		s.emitPhaseFailed(round, round.Error)
	} else {
		round.Status = RoundStatusCompleted
		round.Error = ""
		MutableRoundPayload(&round).ClearEvidenceGateState()
		s.emitPhaseCompleted(round)
		s.emitParsedPhaseSignals(round)
	}
	if err := s.store.SaveRound(round); err != nil {
		return RoundEnvelope{}, err
	}
	if err := s.syncExecutionStatus(round); err != nil {
		return RoundEnvelope{}, err
	}
	if round.Status == RoundStatusCompleted {
		s.maybeAutoStartNext(ctx, round)
	}
	return round, nil
}
