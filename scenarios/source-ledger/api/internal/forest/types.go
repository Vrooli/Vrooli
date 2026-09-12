package forest

import "time"

type (
	Summary struct {
		ID, Scope, Body, FacetID string
		Vector                   []float64
		Depth, Generation        int
		CreatedAt                time.Time
	}
	Edge struct{ ParentID, ChildID, ChildKind string }
	Node struct {
		ID, EntryID, FacetID, Body string
		Vectors                    [][]float64
		CompactionScore            float64
		Depth, Generation          int
		CreatedAt                  time.Time
		Summary                    bool
	}
	CompactionResult struct {
		CompactedCount, EligibleFrontierBefore, EligibleFrontierAfter int
		// Target is the frontier target this pass resolved for its scope. It
		// travels with the result so a reporting surface never has to re-resolve
		// scope policy to interpret the frontier number.
		Target int
	}
	FrontierResult struct {
		Nodes         []Node
		EligibleCount int
		Target        int
	}
)
