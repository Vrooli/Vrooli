package facets

import "time"

const UnclassifiedFacet = "unclassified"

type (
	Definition struct {
		ID, Label, RetentionPolicy string
		CompactionEligible         bool
	}
	Assignment struct {
		ID, EntryID, FacetID, ActorID string
		AssignedAt                    time.Time
	}
)

type ErrUnknownFacet struct{ ID string }

func (e ErrUnknownFacet) Error() string { return "unknown facet: " + e.ID }
