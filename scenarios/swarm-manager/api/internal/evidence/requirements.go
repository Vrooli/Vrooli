package evidence

import (
	"context"
	"fmt"
	"strings"
)

// Requirement is a data-defined verification predicate. It deliberately uses
// normalized fact fields rather than mode-specific artifact vocabulary. A
// matching observation must be verification=verified as well as meeting the
// declared confidence threshold; self-reported or unavailable observations
// never complete an operating-mode gate.
type Requirement struct {
	SubjectKind   string
	Action        string
	ProducerID    string
	MinConfidence Confidence
	MinCount      int
	MatchFields   map[string]string
}

type RequirementState string

const (
	RequirementSatisfied RequirementState = "satisfied"
	RequirementPending   RequirementState = "pending_evidence"
	RequirementMissing   RequirementState = "missing_evidence"
)

type RequirementResult struct {
	Requirement Requirement
	State       RequirementState
	Matches     []Record
	Reason      string
}

func (r Requirement) Validate() error {
	if strings.TrimSpace(r.SubjectKind) == "" || strings.TrimSpace(r.Action) == "" {
		return fmt.Errorf("evidence requirement subject kind and action are required")
	}
	switch r.MinConfidence {
	case ConfidenceAuthoritative, ConfidenceObserved, ConfidenceReported, ConfidenceOperator:
	default:
		return fmt.Errorf("evidence requirement minimum confidence %q is invalid", r.MinConfidence)
	}
	if r.MinCount < 0 {
		return fmt.Errorf("evidence requirement minimum count %d is invalid", r.MinCount)
	}
	if len(r.MatchFields) > 24 {
		return fmt.Errorf("evidence requirement has too many match fields")
	}
	for key, value := range r.MatchFields {
		if strings.TrimSpace(key) == "" || len(key) > 96 || len(value) > 512 {
			return fmt.Errorf("evidence requirement match field exceeds bounds")
		}
	}
	return nil
}

func (s *Service) EvaluateRequirement(ctx context.Context, owner Owner, runID string, requirement Requirement) (RequirementResult, error) {
	if err := requirement.Validate(); err != nil {
		return RequirementResult{}, err
	}
	records, err := s.ListByOwnerID(ctx, owner.Kind, owner.ID)
	if err != nil {
		return RequirementResult{}, err
	}
	matches := []Record{}
	for _, record := range records {
		if record.Observation.RunID != strings.TrimSpace(runID) || record.Observation.Subject.Kind != requirement.SubjectKind || record.Observation.Action != requirement.Action || record.Observation.Verification != VerificationVerified || !confidenceMeets(record.Observation.Confidence, requirement.MinConfidence) || !matchesFields(record.Observation.Metadata, requirement.MatchFields) {
			continue
		}
		if requirement.ProducerID != "" && record.Observation.SourceSystem != requirement.ProducerID {
			continue
		}
		matches = append(matches, record)
	}
	minimum := requirement.MinCount
	if minimum == 0 {
		minimum = 1
	}
	if len(matches) >= minimum {
		return RequirementResult{Requirement: requirement, State: RequirementSatisfied, Matches: matches}, nil
	}
	complete, err := s.store.HasTerminalWatermark(ctx, strings.TrimSpace(requirement.ProducerID), strings.TrimSpace(runID), requirement.SubjectKind)
	if err != nil {
		return RequirementResult{}, err
	}
	if complete {
		return RequirementResult{Requirement: requirement, State: RequirementMissing, Reason: "required producer has terminal coverage with no matching fact"}, nil
	}
	return RequirementResult{Requirement: requirement, State: RequirementPending, Reason: "required producer coverage is incomplete"}, nil
}

func matchesFields(metadata, wanted map[string]string) bool {
	for key, value := range wanted {
		if metadata[key] != value {
			return false
		}
	}
	return true
}

func confidenceMeets(actual, minimum Confidence) bool {
	rank := map[Confidence]int{ConfidenceReported: 1, ConfidenceObserved: 2, ConfidenceAuthoritative: 3, ConfidenceOperator: 3}
	return rank[actual] >= rank[minimum]
}
