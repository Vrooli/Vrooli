// Run-report responsibility: adapt orchestration's durable read seams to the
// shared bounded run-report projection without owning rendering.
package orchestration

import (
	"context"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/runreport"

	"github.com/google/uuid"
)

// BuildRunReport exposes the one bounded diagnostic projection used by every
// investigation surface. It intentionally delegates rendering elsewhere.
func (o *Orchestrator) BuildRunReport(ctx context.Context, runID uuid.UUID) (*runreport.RunReport, error) {
	return runreport.Build(ctx, orchestratorReportSource{o: o}, runID)
}

func (o *Orchestrator) InvocationFacts(ctx context.Context, runID uuid.UUID) ([]runreport.InvocationFact, error) {
	if o.invocationReadModel != nil {
		watermark, err := o.invocationReadModel.Watermark(ctx, runID.String())
		if err != nil {
			return nil, err
		}
		if watermark != nil {
			facts, err := o.invocationReadModel.Facts(ctx, runID.String())
			if err != nil {
				return nil, err
			}
			out := make([]runreport.InvocationFact, 0, len(facts))
			for _, fact := range facts {
				out = append(out, fact.InvocationFact)
			}
			return out, nil
		}
	}
	if o.invocationFacts == nil {
		return []runreport.InvocationFact{}, nil
	}
	return o.invocationFacts.InvocationFacts(ctx, runID)
}

// Episodes returns the durable episode projection for one run. A report build
// creates it, so callers receive an explicit empty collection before evidence
// has been derived rather than a fabricated aggregate.
func (o *Orchestrator) Episodes(ctx context.Context, runID uuid.UUID) ([]runreport.FrictionEpisode, error) {
	store, ok := o.invocationFacts.(runreport.EpisodeStore)
	if !ok {
		return []runreport.FrictionEpisode{}, nil
	}
	return store.Episodes(ctx, runID)
}

func (o *Orchestrator) SelfReportSpans(ctx context.Context, runID uuid.UUID) ([]runreport.SelfReportSpan, error) {
	store, ok := o.invocationFacts.(runreport.SelfReportStore)
	if !ok {
		return []runreport.SelfReportSpan{}, nil
	}
	return store.SelfReportSpans(ctx, runID)
}

func (s orchestratorReportSource) DurableInvocationFacts(ctx context.Context, id uuid.UUID) ([]runreport.InvocationFact, bool, error) {
	if s.o.invocationReadModel == nil {
		return nil, false, nil
	}
	watermark, err := s.o.invocationReadModel.Watermark(ctx, id.String())
	if err != nil || watermark == nil {
		return nil, false, err
	}
	facts, err := s.o.InvocationFacts(ctx, id)
	return facts, true, err
}

type orchestratorReportSource struct{ o *Orchestrator }

var (
	_ runreport.InvocationFactStore = orchestratorReportSource{}
	_ runreport.EpisodeStore        = orchestratorReportSource{}
	_ runreport.SelfReportStore     = orchestratorReportSource{}
	_ runreport.LedgerStore         = orchestratorReportSource{}
	_ runreport.ReceiptJoinStore    = orchestratorReportSource{}
)

func (s orchestratorReportSource) Run(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	return s.o.GetRun(ctx, id)
}

func (s orchestratorReportSource) Events(ctx context.Context, id uuid.UUID) ([]*domain.RunEvent, error) {
	return s.o.GetRunEvents(ctx, id, event.GetOptions{AfterSequence: -1, Limit: 10000})
}

func (s orchestratorReportSource) Diff(ctx context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
	return s.o.GetRunDiff(ctx, id)
}

func (s orchestratorReportSource) Receipts(ctx context.Context, id uuid.UUID) (runreport.ReceiptSummary, error) {
	if s.o.receipts == nil {
		return runreport.ReceiptSummary{State: "unavailable", Detail: "vrooli-events receipt reader is not configured"}, nil
	}
	return s.o.receipts.ReadReceiptSummary(ctx, id)
}

func (s orchestratorReportSource) ReplaceInvocationFacts(ctx context.Context, id uuid.UUID, facts []runreport.InvocationFact) error {
	if s.o.invocationFacts == nil {
		return nil
	}
	return s.o.invocationFacts.ReplaceInvocationFacts(ctx, id, facts)
}

func (s orchestratorReportSource) InvocationFacts(ctx context.Context, id uuid.UUID) ([]runreport.InvocationFact, error) {
	if s.o.invocationFacts == nil {
		return []runreport.InvocationFact{}, nil
	}
	return s.o.invocationFacts.InvocationFacts(ctx, id)
}

func (s orchestratorReportSource) ReplaceEpisodes(ctx context.Context, id uuid.UUID, episodes []runreport.FrictionEpisode) error {
	store, ok := s.o.invocationFacts.(runreport.EpisodeStore)
	if !ok {
		return nil
	}
	return store.ReplaceEpisodes(ctx, id, episodes)
}

func (s orchestratorReportSource) Episodes(ctx context.Context, id uuid.UUID) ([]runreport.FrictionEpisode, error) {
	store, ok := s.o.invocationFacts.(runreport.EpisodeStore)
	if !ok {
		return []runreport.FrictionEpisode{}, nil
	}
	return store.Episodes(ctx, id)
}

func (s orchestratorReportSource) ReplaceSelfReportSpans(ctx context.Context, id uuid.UUID, spans []runreport.SelfReportSpan) error {
	store, ok := s.o.invocationFacts.(runreport.SelfReportStore)
	if !ok {
		return nil
	}
	return store.ReplaceSelfReportSpans(ctx, id, spans)
}

func (s orchestratorReportSource) SelfReportSpans(ctx context.Context, id uuid.UUID) ([]runreport.SelfReportSpan, error) {
	store, ok := s.o.invocationFacts.(runreport.SelfReportStore)
	if !ok {
		return []runreport.SelfReportSpan{}, nil
	}
	return store.SelfReportSpans(ctx, id)
}

func (s orchestratorReportSource) ReplaceReceiptEvidence(ctx context.Context, id uuid.UUID, availability string, eventIDs []string) error {
	if s.o.invocationFacts == nil {
		return nil
	}
	store, ok := s.o.invocationFacts.(runreport.ReceiptJoinStore)
	if !ok {
		return nil
	}
	return store.ReplaceReceiptEvidence(ctx, id, availability, eventIDs)
}

func (s orchestratorReportSource) ReceiptEvidence(ctx context.Context, id uuid.UUID) ([]string, error) {
	if s.o.invocationFacts == nil {
		return []string{}, nil
	}
	store, ok := s.o.invocationFacts.(runreport.ReceiptJoinStore)
	if !ok {
		return []string{}, nil
	}
	return store.ReceiptEvidence(ctx, id)
}

func (s orchestratorReportSource) ReplaceCrossScenarioCalls(ctx context.Context, id uuid.UUID, availability string, calls []runreport.CrossScenarioCall) error {
	store, ok := s.o.invocationFacts.(runreport.LedgerStore)
	if !ok {
		return nil
	}
	return store.ReplaceCrossScenarioCalls(ctx, id, availability, calls)
}

func (s orchestratorReportSource) CrossScenarioCalls(ctx context.Context, id uuid.UUID) ([]runreport.CrossScenarioCall, error) {
	store, ok := s.o.invocationFacts.(runreport.LedgerStore)
	if !ok {
		return []runreport.CrossScenarioCall{}, nil
	}
	return store.CrossScenarioCalls(ctx, id)
}
