package coverage

import (
	"context"
	"testing"
	"time"

	internalcoverage "meta-optimization-manager/internal/coverage"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/spacedoc"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/coverage"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/shared"
)

// fakeService is a hand fake of internalcoverage.Service for handler tests.
type fakeService struct {
	status   internalcoverage.Status
	cells    []internalcoverage.Cell
	cell     internalcoverage.Cell
	cellErr  error
	report   internalcoverage.BaseDocReport
	lastProj internalcoverage.Projection
}

func (f *fakeService) GetStatus(_ context.Context, p internalcoverage.Projection) (internalcoverage.Status, error) {
	f.lastProj = p
	return f.status, nil
}

func (f *fakeService) ListCells(_ context.Context, p internalcoverage.Projection, _ spacedoc.CellStatus) ([]internalcoverage.Cell, error) {
	f.lastProj = p
	return f.cells, nil
}

func (f *fakeService) ExplainCell(_ context.Context, _ string) (internalcoverage.Cell, error) {
	return f.cell, f.cellErr
}

func (f *fakeService) ValidateBaseDocs(_ context.Context, _ internalcoverage.Projection) (internalcoverage.BaseDocReport, error) {
	return f.report, nil
}

func TestHandlerGetStatus(t *testing.T) {
	svc := &fakeService{status: internalcoverage.Status{
		ComputedAt: time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC),
		Projections: []internalcoverage.ProjectionCoverage{{
			Projection: internalcoverage.ProjectionAnswer, NowCount: 3, InReachCount: 26, MissingCount: 7,
			TotalCells: 36, CoverageRatio: 0.0833, DenominatorConfidence: spacedoc.ConfidencePartial, Available: true,
		}},
		LatestTrialTrend: &internalcoverage.EmpiricalTrendPoint{SuccessRate: 0.5, MedianTokens: 1200, At: time.Now()},
	}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.GetStatus(context.Background(), connect.NewRequest(&coveragev1.GetStatusRequest{Projection: sharedv1.Projection_PROJECTION_ANSWER}))
	if err != nil {
		t.Fatal(err)
	}
	if svc.lastProj != internalcoverage.ProjectionAnswer {
		t.Errorf("projection not threaded: %q", svc.lastProj)
	}
	pcs := resp.Msg.GetProjections()
	if len(pcs) != 1 || pcs[0].GetNowCount() != 3 || pcs[0].GetProjection() != sharedv1.Projection_PROJECTION_ANSWER {
		t.Errorf("projections = %+v", pcs)
	}
	if pcs[0].GetDenominatorConfidence() != sharedv1.DenominatorConfidence_DENOMINATOR_CONFIDENCE_PARTIAL {
		t.Errorf("confidence = %v", pcs[0].GetDenominatorConfidence())
	}
	if resp.Msg.GetLatestTrialTrend().GetMedianTokens() != 1200 {
		t.Errorf("trend not mapped: %+v", resp.Msg.GetLatestTrialTrend())
	}
}

func TestHandlerExplainCellNotFound(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{cellErr: context.DeadlineExceeded}})
	_, err := h.ExplainCell(context.Background(), connect.NewRequest(&coveragev1.ExplainCellRequest{CellId: "answer/999"}))
	if err == nil || connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", err)
	}
}

func TestHandlerValidateBaseDocs(t *testing.T) {
	svc := &fakeService{report: internalcoverage.BaseDocReport{
		OK: false,
		Issues: []internalcoverage.BaseDocIssue{
			{Projection: internalcoverage.ProjectionGuide, Code: "guide_row_no_skill", Message: "x", Location: "y", Severity: internalcoverage.SeverityError},
		},
	}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.ValidateBaseDocs(context.Background(), connect.NewRequest(&coveragev1.ValidateBaseDocsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetOk() {
		t.Error("expected ok=false")
	}
	if len(resp.Msg.GetIssues()) != 1 || resp.Msg.GetIssues()[0].GetSeverity() != sharedv1.Severity_SEVERITY_ERROR {
		t.Errorf("issues = %+v", resp.Msg.GetIssues())
	}
}
