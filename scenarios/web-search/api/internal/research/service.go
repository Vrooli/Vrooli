package research

import (
	"context"
	"log"

	"web-search/internal/findings"
	"web-search/internal/research/agentmanager"
)

// FindingsService is the subset of the findings application surface the
// research domain depends on: capture (Add), gather (semantic + nearby reads),
// and reconcile (Supersede / Flag). Both internalfindings.Service and a test
// fake satisfy it.
type FindingsService interface {
	Add(ctx context.Context, in findings.NewFinding) (findings.Finding, error)
	Supersede(ctx context.Context, id, replacement, reason string) (findings.Finding, error)
	Flag(ctx context.Context, id, reason string) (findings.Finding, error)
}

// Deps wires the research service's seams. All are optional: a nil Searcher /
// Fetcher / Synthesizer makes L2 abstain (graceful when browserless/ollama are
// down); a nil AgentManager makes L3 unavailable; a nil Findings disables
// capture. main.go constructs the production impls from env; tests inject fakes.
type Deps struct {
	Searcher     Searcher
	Fetcher      Fetcher
	Synthesizer  Synthesizer
	Findings     FindingsService
	AgentManager agentmanager.Service
	Logger       *log.Logger
}

// Service orchestrates the L2 (synchronous fetch/read/synthesize) and L3
// (agent-manager run) research paths over its injected seams.
type Service struct {
	searcher     Searcher
	fetcher      Fetcher
	synthesizer  Synthesizer
	findings     FindingsService
	agentManager agentmanager.Service
	logger       *log.Logger
}

// NewService constructs the research service.
func NewService(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		searcher:     d.Searcher,
		fetcher:      d.Fetcher,
		synthesizer:  d.Synthesizer,
		findings:     d.Findings,
		agentManager: d.AgentManager,
		logger:       logger,
	}
}

// RunL2 runs the synchronous L2 fetch -> read -> single-pass cited synthesis
// pipeline. When the seams needed for L2 (searcher/fetcher/synthesizer) are not
// wired it abstains gracefully rather than erroring.
func (s *Service) RunL2(ctx context.Context, query string, topN int, capture bool) (L2Outcome, error) {
	if s.searcher == nil || s.fetcher == nil || s.synthesizer == nil {
		return L2Outcome{
			Brief:     Brief{Query: query, Level: LevelL2, Summary: abstainNote},
			Abstained: true,
		}, nil
	}
	return s.runL2(ctx, query, topN, capture)
}
