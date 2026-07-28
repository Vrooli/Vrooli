package forest

import "context"

// CandidateSource lets forest enforce retention and pin policy without owning
// journal or facets storage.
type (
	CandidateSource interface {
		CompactionCandidates(context.Context) ([]Candidate, error)
	}
	Candidate struct {
		EntryID, FacetID, Body string
		Pinned                 bool
	}
	Service struct {
		repo   Repository
		source CandidateSource
	}
)

func NewService(repo Repository, source CandidateSource) *Service {
	return &Service{repo: repo, source: source}
}
