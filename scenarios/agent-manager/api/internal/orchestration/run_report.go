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

type orchestratorReportSource struct{ o *Orchestrator }

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
