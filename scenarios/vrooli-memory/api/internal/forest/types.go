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
)
