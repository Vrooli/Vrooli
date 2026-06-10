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

// FindingsGatherer is the bounded semantic-gather seam: given a query it returns
// the findings semantically NEAR it, already capped by the caller's limit. The
// production impl wraps the findings semantic index + store; a test fake
// satisfies it. A nil Gatherer makes GatherRelatedFindings unavailable.
type FindingsGatherer interface {
	Gather(ctx context.Context, query string, limit int) ([]GatheredFinding, error)
}

// Deps wires the research service's seams. All are optional: a nil Searcher /
// Fetcher / Synthesizer makes L2 abstain (graceful when the fetch stack/ollama are
// down); a nil AgentManager makes L3 unavailable; a nil Findings disables
// capture. main.go constructs the production impls from env; tests inject fakes.
type Deps struct {
	Searcher     Searcher
	Fetcher      Fetcher
	Synthesizer  Synthesizer
	Findings     FindingsService
	Gatherer     FindingsGatherer
	AgentManager agentmanager.Service
	Logger       *log.Logger

	// ConfidenceGate overrides HighConfidenceThreshold for the L3 reconcile
	// confidence gate. Zero (or any value outside (0,1]) keeps the default.
	ConfidenceGate float64
	// GatherCap overrides MaxGatherFindings, the hard cap on the bounded
	// GATHER sweep. Zero or negative keeps the default.
	GatherCap int
	// MaxResearchLoops overrides DefaultMaxResearchLoops, the iteration
	// budget written into the L3 task contract. Zero or negative keeps the
	// default.
	MaxResearchLoops int
}

// Service orchestrates the L2 (synchronous fetch/read/synthesize) and L3
// (agent-manager run) research paths over its injected seams.
type Service struct {
	searcher       Searcher
	fetcher        Fetcher
	synthesizer    Synthesizer
	findings       FindingsService
	gatherer       FindingsGatherer
	agentManager   agentmanager.Service
	logger         *log.Logger
	confidenceGate float64
	gatherCap      int
	maxLoops       int
}

// NewService constructs the research service.
func NewService(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = log.Default()
	}
	gate := d.ConfidenceGate
	if gate <= 0 || gate > 1 {
		gate = HighConfidenceThreshold
	}
	gatherCap := d.GatherCap
	if gatherCap <= 0 {
		gatherCap = MaxGatherFindings
	}
	maxLoops := d.MaxResearchLoops
	if maxLoops <= 0 {
		maxLoops = DefaultMaxResearchLoops
	}
	return &Service{
		searcher:       d.Searcher,
		fetcher:        d.Fetcher,
		synthesizer:    d.Synthesizer,
		findings:       d.Findings,
		gatherer:       d.Gatherer,
		agentManager:   d.AgentManager,
		logger:         logger,
		confidenceGate: gate,
		gatherCap:      gatherCap,
		maxLoops:       maxLoops,
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
