package forest

import "time"

type (
	Summary struct {
		ID, Body, FacetID string
		Depth, Generation int
		CreatedAt         time.Time
	}
	Edge struct{ ParentID, ChildID, ChildKind string }
)
