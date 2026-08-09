// Run-report responsibility: adapt orchestration's durable read seams to the
// shared bounded run-report projection without owning rendering.
package orchestration

import (
	"context"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runreport"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

// BuildRunReport exposes the one bounded diagnostic projection used by every
// investigation surface. It intentionally delegates rendering elsewhere.
func (o *Orchestrator) BuildRunReport(ctx context.Context, runID uuid.UUID) (*runreport.RunReport, error) {
	return runreport.Build(ctx, orchestratorReportSource{o: o}, runID)
}

func (o *Orchestrator) InvocationFacts(ctx context.Context, runID uuid.UUID) ([]runsignal.InvocationFact, error) {
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
			out := make([]runsignal.InvocationFact, 0, len(facts))
			for _, fact := range facts {
				out = append(out, fact.InvocationFact)
			}
			return out, nil
		}
	}
	return []runsignal.InvocationFact{}, nil
}

// Episodes returns the durable episode projection for one run. A report build
// creates it, so callers receive an explicit empty collection before evidence
// has been derived rather than a fabricated aggregate.
func (o *Orchestrator) Episodes(ctx context.Context, runID uuid.UUID) ([]runsignal.FrictionEpisode, error) {
	if projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore); ok {
		episodes, err := projection.EpisodesForRun(ctx, runID.String())
		if err != nil {
			return nil, err
		}
		out := make([]runsignal.FrictionEpisode, 0, len(episodes))
		for _, episode := range episodes {
			out = append(out, episode.FrictionEpisode)
		}
		return out, nil
	}
	return []runsignal.FrictionEpisode{}, nil
}

func (o *Orchestrator) SelfReportSpans(ctx context.Context, runID uuid.UUID) ([]runsignal.SelfReportSpan, error) {
	if projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore); ok {
		spans, err := projection.SelfReportSpansForRun(ctx, runID.String())
		if err != nil {
			return nil, err
		}
		out := make([]runsignal.SelfReportSpan, 0, len(spans))
		for _, span := range spans {
			out = append(out, span.SelfReportSpan)
		}
		return out, nil
	}
	return []runsignal.SelfReportSpan{}, nil
}

func (s orchestratorReportSource) DurableInvocationFacts(ctx context.Context, id uuid.UUID) ([]runsignal.InvocationFact, bool, error) {
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
	_ runreport.DurableInvocationFactSource = orchestratorReportSource{}
	_ runreport.DurableEpisodeSource        = orchestratorReportSource{}
	_ runreport.DurableSelfReportSpanSource = orchestratorReportSource{}
	_ runreport.DurableTimeAccountingSource = orchestratorReportSource{}
	_ runreport.LedgerStore                 = orchestratorReportSource{}
	_ runreport.ReceiptJoinStore            = orchestratorReportSource{}
)

func (s orchestratorReportSource) Run(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	return s.o.GetRun(ctx, id)
}

func (s orchestratorReportSource) DurableTimeAccounting(ctx context.Context, id uuid.UUID) (runsignal.TimeAccounting, bool, error) {
	if projection, ok := s.o.invocationReadModel.(invocationreadmodel.ProjectionStore); ok {
		return projection.TimeAccountingForRun(ctx, id.String())
	}
	return runsignal.TimeAccounting{}, false, nil
}

func (s orchestratorReportSource) Events(ctx context.Context, id uuid.UUID) ([]*domain.RunEvent, error) {
	// The run report derives time accounting and evidence from this stream, so
	// a capped read silently describes part of a long run as the whole run —
	// the same defect the invocation projection carried.
	return s.o.allRunEvents(ctx, id, event.GetOptions{AfterSequence: -1})
}

func (s orchestratorReportSource) Diff(ctx context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
	return s.o.GetRunDiff(ctx, id)
}

func (s orchestratorReportSource) Receipts(ctx context.Context, id uuid.UUID) (runreport.ReceiptSummary, error) {
	if s.o.receipts == nil {
		return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityUnavailable, Reason: "vrooli-events receipt reader is not configured"}}, nil
	}
	return s.o.receipts.ReadReceiptSummary(ctx, id)
}

func (s orchestratorReportSource) DurableEpisodes(ctx context.Context, id uuid.UUID) ([]runsignal.FrictionEpisode, bool, error) {
	if s.o.invocationReadModel == nil {
		return nil, false, nil
	}
	watermark, err := s.o.invocationReadModel.Watermark(ctx, id.String())
	if err != nil || watermark == nil {
		return nil, false, err
	}
	projection, ok := s.o.invocationReadModel.(invocationreadmodel.ProjectionStore)
	if !ok {
		return nil, false, nil
	}
	episodes, err := projection.EpisodesForRun(ctx, id.String())
	if err != nil {
		return nil, false, err
	}
	out := make([]runsignal.FrictionEpisode, 0, len(episodes))
	for _, episode := range episodes {
		out = append(out, episode.FrictionEpisode)
	}
	return out, true, nil
}

func (s orchestratorReportSource) DurableSelfReportSpans(ctx context.Context, id uuid.UUID) ([]runsignal.SelfReportSpan, bool, error) {
	if s.o.invocationReadModel == nil {
		return nil, false, nil
	}
	watermark, err := s.o.invocationReadModel.Watermark(ctx, id.String())
	if err != nil || watermark == nil {
		return nil, false, err
	}
	projection, ok := s.o.invocationReadModel.(invocationreadmodel.ProjectionStore)
	if !ok {
		return nil, false, nil
	}
	spans, err := projection.SelfReportSpansForRun(ctx, id.String())
	if err != nil {
		return nil, false, err
	}
	out := make([]runsignal.SelfReportSpan, 0, len(spans))
	for _, span := range spans {
		out = append(out, span.SelfReportSpan)
	}
	return out, true, nil
}

func (s orchestratorReportSource) ReplaceReceiptEvidence(ctx context.Context, id uuid.UUID, availability string, eventIDs []string) error {
	if s.o.receiptEvidence == nil {
		return nil
	}
	return s.o.receiptEvidence.ReplaceReceiptEvidence(ctx, id, availability, eventIDs)
}

func (s orchestratorReportSource) ReceiptEvidence(ctx context.Context, id uuid.UUID) ([]string, error) {
	if s.o.receiptEvidence == nil {
		return []string{}, nil
	}
	return s.o.receiptEvidence.ReceiptEvidence(ctx, id)
}

func (s orchestratorReportSource) ReplaceCrossScenarioCalls(ctx context.Context, id uuid.UUID, availability string, calls []runreport.CrossScenarioCall) error {
	if s.o.investigationLedger == nil {
		return nil
	}
	return s.o.investigationLedger.ReplaceCrossScenarioCalls(ctx, id, availability, calls)
}

func (s orchestratorReportSource) CrossScenarioCalls(ctx context.Context, id uuid.UUID) ([]runreport.CrossScenarioCall, error) {
	if s.o.investigationLedger == nil {
		return []runreport.CrossScenarioCall{}, nil
	}
	return s.o.investigationLedger.CrossScenarioCalls(ctx, id)
}
