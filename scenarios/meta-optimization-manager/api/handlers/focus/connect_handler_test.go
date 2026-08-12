package focus

import (
	"context"
	"testing"

	internalfocus "meta-optimization-manager/internal/focus"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/spacedoc"
	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/focus"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/shared"
)

// fakeService is a hand fake of internalfocus.Service for handler tests.
type fakeService struct {
	result      internalfocus.FocusResult
	gaps        []internalfocus.Gap
	gap         internalfocus.Gap
	gapErr      error
	noteErr     error
	lastID      string
	lastApproac string
	lastFilter  internalfocus.GapFilter
}

func (f *fakeService) GetFocus(_ context.Context, _ int, _ internalfocus.Projection) (internalfocus.FocusResult, error) {
	return f.result, nil
}

func (f *fakeService) ListGaps(_ context.Context, filter internalfocus.GapFilter) ([]internalfocus.Gap, error) {
	f.lastFilter = filter
	return f.gaps, nil
}

func (f *fakeService) GetGap(_ context.Context, id string) (internalfocus.Gap, error) {
	f.lastID = id
	return f.gap, f.gapErr
}

func (f *fakeService) AddGapNote(_ context.Context, id, approach string) (internalfocus.Gap, error) {
	f.lastID = id
	f.lastApproac = approach
	return f.gap, f.noteErr
}

func (f *fakeService) ListCondition(_ context.Context) ([]internalfocus.Gap, error) {
	return f.gaps, nil
}

func (f *fakeService) ExplainCondition(_ context.Context, providerID string) (internalfocus.Gap, error) {
	f.lastID = providerID
	return f.gap, f.gapErr
}

func TestHandlerGetFocus(t *testing.T) {
	svc := &fakeService{result: internalfocus.FocusResult{Items: []internalfocus.FocusItem{{
		Gap: internalfocus.Gap{
			ID:              "answer/1",
			Axis:            internalfocus.AxisCoverage,
			Projection:      internalfocus.ProjectionAnswer,
			Title:           "x",
			Status:          spacedoc.StatusMissing,
			Recurrence:      3,
			EvidenceSource:  "trials",
			EvidenceLocator: "trial-task:x/run:1",
		},
		Impact:     1.0,
		Importance: 1.0,
		Priority:   1.0,
		Rationale:  "top",
	}}}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.GetFocus(context.Background(), connect.NewRequest(&focusv1.GetFocusRequest{Limit: 5}))
	if err != nil {
		t.Fatal(err)
	}
	its := resp.Msg.GetItems()
	if len(its) != 1 || its[0].GetGap().GetId() != "answer/1" || its[0].GetPriorityScore() != 1.0 {
		t.Fatalf("items = %+v", its)
	}
	if its[0].GetGap().GetStatus() != sharedv1.CellStatus_CELL_STATUS_MISSING {
		t.Fatalf("status not mapped: %v", its[0].GetGap().GetStatus())
	}
	if its[0].GetGap().GetAxis() != sharedv1.GapAxis_GAP_AXIS_COVERAGE || its[0].GetGap().GetRecurrence() != 3 || its[0].GetGap().GetEvidenceLocator() == "" {
		t.Fatalf("provenance not mapped: %v", its[0].GetGap())
	}
}

func TestHandlerGetFocusCarriesDegradedMetadata(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{result: internalfocus.FocusResult{
		Degraded:       true,
		DegradedReason: "search-hub insights unavailable",
	}}})
	resp, err := h.GetFocus(context.Background(), connect.NewRequest(&focusv1.GetFocusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetDegraded() || resp.Msg.GetDegradedReason() == "" {
		t.Fatalf("degraded metadata lost: %+v", resp.Msg)
	}
}

func TestHandlerListGapsThreadsFilter(t *testing.T) {
	svc := &fakeService{gaps: []internalfocus.Gap{{ID: "validate/2"}}}
	h := NewConnectHandler(Deps{Service: svc})
	_, err := h.ListGaps(context.Background(), connect.NewRequest(&focusv1.ListGapsRequest{
		Projection: sharedv1.Projection_PROJECTION_VALIDATE,
		CellId:     "2",
		Status:     sharedv1.CellStatus_CELL_STATUS_IN_REACH,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if svc.lastFilter.Projection != internalfocus.ProjectionValidate || svc.lastFilter.CellID != "2" || svc.lastFilter.Status != spacedoc.StatusInReach {
		t.Fatalf("filter not threaded: %+v", svc.lastFilter)
	}
}

func TestHandlerGetGapNotFound(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{gapErr: context.DeadlineExceeded}})
	_, err := h.GetGap(context.Background(), connect.NewRequest(&focusv1.GetGapRequest{Id: "x/9"}))
	if err == nil || connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestHandlerAddGapNote(t *testing.T) {
	svc := &fakeService{gap: internalfocus.Gap{ID: "answer/1", Approaches: []string{"idea"}}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.AddGapNote(context.Background(), connect.NewRequest(&focusv1.AddGapNoteRequest{Id: "answer/1", Approach: "idea"}))
	if err != nil {
		t.Fatal(err)
	}
	if svc.lastID != "answer/1" || svc.lastApproac != "idea" {
		t.Fatalf("args not threaded: id=%q approach=%q", svc.lastID, svc.lastApproac)
	}
	if len(resp.Msg.GetGap().GetApproaches()) != 1 {
		t.Fatalf("gap not returned: %+v", resp.Msg.GetGap())
	}
}
