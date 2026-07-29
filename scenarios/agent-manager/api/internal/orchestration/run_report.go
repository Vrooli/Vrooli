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
	if o.invocationFacts == nil {
		return []runreport.InvocationFact{}, nil
	}
	return o.invocationFacts.InvocationFacts(ctx, runID)
}

type orchestratorReportSource struct{ o *Orchestrator }

var (
	_ runreport.InvocationFactStore = orchestratorReportSource{}
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
