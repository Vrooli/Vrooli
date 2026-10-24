package facets

import "time"

const UnclassifiedFacet = "unclassified"

type (
	Definition struct {
		ID, Label, ClassificationGuidance, RetentionPolicy string
		ClassificationExamples                             []string
		CompactionEligible                                 bool
		ResidentBudget                                     int
	}
	FacetPolicy struct {
		ID                 string
		RetentionPolicy    string
		CompactionEligible bool
		ResidentBudget     int
	}
	Assignment struct {
		ID, EntryID, FacetID, ActorID string
		AssignedAt                    time.Time
	}
	PinProposal struct {
		ID        string
		EntryIDs  []string
		Rationale string
	}
	PinCandidate struct {
		EntryID, Body  string
		RecallCount    int
		CreatedAt      time.Time
		LastRecalledAt *time.Time
	}
)

type ErrPinBudgetExceeded struct {
	ProposalID string
}

func (e ErrPinBudgetExceeded) Error() string {
	return "pin budget exceeded; review proposal " + e.ProposalID
}

type ErrUnknownFacet struct{ ID string }

func (e ErrUnknownFacet) Error() string { return "unknown facet: " + e.ID }
