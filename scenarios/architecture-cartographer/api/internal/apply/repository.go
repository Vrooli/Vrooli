package apply

import "context"

// Repository is the persistence seam for apply plans + runs. v0.1
// persists planned operations so v0.2's RunApply can pick them up
// without re-deriving.
type Repository interface {
	SavePlan(ctx context.Context, p Plan) (Plan, error)
	GetPlan(ctx context.Context, id string) (Plan, error)
	ListRuns(ctx context.Context, f ListRunsFilter) (RunPage, error)
	GetBaseline(ctx context.Context, scenario string) (BuildBaseline, error)
}

// ListRunsFilter scopes ListRuns.
type ListRunsFilter struct {
	Scenario  string
	Domain    string
	PageSize  int
	PageToken string
}

type RunPage struct {
	Runs          []ApplyRun
	NextPageToken string
}
