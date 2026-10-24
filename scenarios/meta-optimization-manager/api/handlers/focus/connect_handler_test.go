package focus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	internalfocus "meta-optimization-manager/internal/focus"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/api-core/spacedoc"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/focus"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/shared"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
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

func (f *fakeService) ListConditionReport(_ context.Context) (internalfocus.ConditionReport, error) {
	return internalfocus.ConditionReport{Gaps: f.gaps}, nil
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
			MaturityFindings: []internalfocus.MaturityFinding{{
				Code: "SEARCH_CONFIG_INVALID", Message: "descriptor is invalid", Location: "search.json",
				Remediation: "repair the descriptor", FixClass: "descriptor", RepairCommand: "search-hub maturity fix x --apply",
			}},
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
	findings := its[0].GetGap().GetMaturityFindings()
	if len(findings) != 1 || findings[0].GetRepairCommand() != "search-hub maturity fix x --apply" || findings[0].GetFixClass() != "descriptor" {
		t.Fatalf("maturity evidence not mapped: %v", findings)
	}
}

type fixedScenarioResolver struct{ base string }

func (r fixedScenarioResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.base, nil
}

func TestSearchHubMaturityReaderUsesRegistryAndCarriesFindingEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var message any
		switch {
		case strings.Contains(req.URL.Path, "RegistryService/ListMaturityTargets"):
			message = &registryv1.ListMaturityTargetsResponse{Targets: []*registryv1.MaturityTarget{{
				Scenario: "descriptor-only", ApplicabilityReason: "descriptor",
			}}}
		case strings.Contains(req.URL.Path, "ScenarioValidationService/ValidateScenario"):
			message = &scenariovalidationv1.ValidateScenarioResponse{Assessment: &commonv1.MaturityAssessment{
				Scenario: "descriptor-only",
				Local:    &commonv1.LocalMaturityAssessment{BlockingFindingCodes: []string{"SEARCH_CONFIG_INVALID"}},
				Findings: []*commonv1.AssessmentFinding{{
					Code: "SEARCH_CONFIG_INVALID", Message: "descriptor is invalid", Location: "search.json",
					Remediation: "repair the descriptor",
				}},
			}}
		default:
			http.Error(w, "unexpected endpoint", http.StatusNotFound)
			return
		}
		body, err := proto.Marshal(message.(proto.Message))
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		w.Header().Set("Content-Type", "application/proto")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	observations, err := (searchHubMaturityReader{
		resolver: fixedScenarioResolver{base: server.URL},
		http:     server.Client(),
	}).Maturity(context.Background())
	if err != nil {
		t.Fatalf("Maturity: %v", err)
	}
	if len(observations) != 1 || observations[0].Scenario != "descriptor-only" {
		t.Fatalf("observations = %+v, want descriptor-only scenario from registry", observations)
	}
	findings := observations[0].Findings
	if len(findings) != 1 || findings[0].Message != "descriptor is invalid" || findings[0].Location != "search.json" || findings[0].Remediation != "repair the descriptor" || findings[0].FixClass != "manual" || findings[0].RepairCommand != "search-hub maturity fix descriptor-only --apply" {
		t.Fatalf("finding evidence = %+v", findings)
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
