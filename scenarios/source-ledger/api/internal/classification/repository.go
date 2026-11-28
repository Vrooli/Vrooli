package classification

import (
	"context"
	"time"
)

type repository interface {
	prepare(context.Context, definition, time.Time, int) (work, error)
	save(context.Context, definition, work, []decision, time.Time) error
	matchingEntries(context.Context, string, definition) ([]string, error)
	coverage(context.Context, string, definition) (bool, error)
	notices(context.Context, string, definition, int64, int) ([]Notice, error)
}

type work struct {
	items                    []observation
	historyStart, historyEnd int64
	liveStart, liveEnd       int64
}

type observation struct {
	position  int64
	attempts  int
	id        string
	scope     string
	body      string
	kind      string
	evidence  string
	hint      string
	importKey string
}

type decision struct {
	match    bool
	method   string
	provider string
	model    string
	failure  string
}

// Notice identifies a classified observation. Consumers own their cursor and
// use recall and zoom to determine its current significance.
type Notice struct {
	Sequence      int64  `json:"sequence"`
	ObservationID string `json:"observation_id"`
	CategoryID    string `json:"category_id"`
	Scope         string `json:"scope"`
}
