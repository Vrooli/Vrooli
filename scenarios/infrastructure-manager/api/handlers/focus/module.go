package focus

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/focus"
	focusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/focus/focus_v1connect"
	internalfocus "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/focus"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/module"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Module(root string) module.Module { return ModuleWithDeps(root, nil, nil) }

func Schema() string { return internalfocus.Schema() }

// ModuleWithDeps composes the ranked surface from both joins. The condition
// reader is passed in rather than constructed here so focus reads the same
// configured condition service the condition domain serves, instead of a
// second one that could drift from it.
func ModuleWithDeps(root string, db *database.RoutedDB, condition internalfocus.ConditionReader) module.Module {
	merged := internalfocus.MergedSource{Sources: []internalfocus.Source{internalfocus.NewCoverageSource(root)}}
	if condition != nil {
		merged.Sources = append(merged.Sources, internalfocus.ConditionSource{Condition: condition})
	} else {
		merged.Sources = append(merged.Sources, internalfocus.ConditionSource{})
	}
	service := &internalfocus.Service{Source: merged}
	if db != nil {
		service.Repository = internalfocus.NewSQLiteRepository(db)
	}
	path, handler := focusconnect.NewFocusServiceHandler(&connectHandler{service: service})
	return module.Module{Name: "focus", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

type connectHandler struct{ service *internalfocus.Service }

func (h *connectHandler) GetNext(ctx context.Context, req *connect.Request[focusv1.GetNextRequest]) (*connect.Response[focusv1.GetNextResponse], error) {
	findings, sources, noFindings, allUnavailable := h.service.Next(ctx, int(req.Msg.GetLimit()))
	response := &focusv1.GetNextResponse{NoFindings: noFindings, AllSourcesUnavailable: allUnavailable, ComputedAt: timestamppb.Now()}
	for _, finding := range findings {
		response.Findings = append(response.Findings, protoFinding(finding))
	}
	for _, source := range sources {
		response.Sources = append(response.Sources, protoSource(source))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) GetFinding(ctx context.Context, req *connect.Request[focusv1.GetFindingRequest]) (*connect.Response[focusv1.GetFindingResponse], error) {
	findings, _, _, _ := h.service.Next(ctx, 0)
	for _, finding := range findings {
		if finding.ID == req.Msg.GetFindingId() {
			return connect.NewResponse(&focusv1.GetFindingResponse{Finding: protoFinding(finding)}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("finding %q was not found", req.Msg.GetFindingId()))
}

func (h *connectHandler) ListSources(ctx context.Context, _ *connect.Request[focusv1.ListSourcesRequest]) (*connect.Response[focusv1.ListSourcesResponse], error) {
	_, sources, _, _ := h.service.Next(ctx, 0)
	response := &focusv1.ListSourcesResponse{}
	for _, source := range sources {
		response.Sources = append(response.Sources, protoSource(source))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) GetEfficacy(ctx context.Context, req *connect.Request[focusv1.GetEfficacyRequest]) (*connect.Response[focusv1.GetEfficacyResponse], error) {
	records, err := h.service.Efficacy(ctx, req.Msg.GetFindingId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &focusv1.GetEfficacyResponse{}
	for _, record := range records {
		response.Records = append(response.Records, protoEfficacy(record))
	}
	return connect.NewResponse(response), nil
}

func protoFinding(f internalfocus.RankedFinding) *focusv1.Finding {
	return &focusv1.Finding{Id: f.ID, Source: f.Source, CellRef: f.CellRef, Title: f.Title, Message: f.Message, SensorRef: f.CellRef, ExpectedReturn: f.ExpectedReturn, Rationale: &focusv1.RankingRationale{Rank: int32(f.Rank), CascadeStage: stageName(f.Stage), Explanation: f.RankExplanation}, ObservedAt: timestamppb.Now()}
}

func protoSource(source internalfocus.GapSource) *focusv1.GapSource {
	return &focusv1.GapSource{Id: source.ID, Label: source.Label, Available: source.Available, Reason: source.Reason, FindingCount: int32(source.FindingCount)}
}

func protoEfficacy(record internalfocus.EfficacyRecord) *focusv1.EfficacyRecord {
	verdict := focusv1.EfficacyVerdict_EFFICACY_VERDICT_UNMEASURABLE
	switch record.Verdict {
	case internalfocus.EfficacyMoved:
		verdict = focusv1.EfficacyVerdict_EFFICACY_VERDICT_MOVED
	case internalfocus.EfficacyDidNotMove:
		verdict = focusv1.EfficacyVerdict_EFFICACY_VERDICT_DID_NOT_MOVE
	case internalfocus.EfficacyAwaitingWork:
		verdict = focusv1.EfficacyVerdict_EFFICACY_VERDICT_AWAITING_WORK
	}
	return &focusv1.EfficacyRecord{FindingId: record.FindingID, SensorRef: record.SensorRef, ExpectedReturn: record.ExpectedReturn, ObservedReturn: record.ObservedReturn, Verdict: verdict, ObservedAt: timestamppb.New(record.ObservedAt)}
}

func stageName(stage internalfocus.Stage) string {
	switch stage {
	case internalfocus.StageIntegrity:
		return "sensor-channel-integrity"
	case internalfocus.StageSubstrate:
		return "host-process-substrate"
	case internalfocus.StageAvailability:
		return "capability-availability"
	case internalfocus.StageEfficiency:
		return "efficiency-performance"
	default:
		return "measurement-improvement"
	}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "focus_next", Path: focusconnect.FocusServiceGetNextProcedure, Method: "POST", Summary: "Read the ranked next findings", Category: "focus"},
	{ID: "focus_finding", Path: focusconnect.FocusServiceGetFindingProcedure, Method: "POST", Summary: "Read one finding", Category: "focus"},
	{ID: "focus_sources", Path: focusconnect.FocusServiceListSourcesProcedure, Method: "POST", Summary: "Read finding sources", Category: "focus"},
	{ID: "focus_efficacy", Path: focusconnect.FocusServiceGetEfficacyProcedure, Method: "POST", Summary: "Read efficacy evidence", Category: "focus"},
}
