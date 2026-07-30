package forest

import "time"

type (
	Summary struct {
		ID, Body, FacetID string
		Vector            []float64
		Depth, Generation int
		CreatedAt         time.Time
	}
	Edge struct{ ParentID, ChildID, ChildKind string }
	Node struct {
		ID, EntryID, FacetID, Body string
		Vector                     []float64
		Depth, Generation          int
		CreatedAt                  time.Time
		Summary                    bool
	}
	CompactionResult struct {
		CompactedCount, EligibleFrontierBefore, EligibleFrontierAfter int
	}
)
