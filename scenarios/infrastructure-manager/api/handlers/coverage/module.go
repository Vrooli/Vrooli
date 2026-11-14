package coverage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/spacedoc"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage"
	coveragev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage/coverage_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/shared"
	internalcoverage "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/coverage"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/module"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Module(root string) module.Module {
	service := internalcoverage.NewService(root, nil)
	path, handler := coveragev1connect.NewCoverageServiceHandler(&connectHandler{service: service})
	return module.Module{
		Name:      "coverage",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) },
		Endpoints: Endpoints,
	}
}

type connectHandler struct{ service *internalcoverage.Service }

func (h *connectHandler) GetCoverage(ctx context.Context, req *connect.Request[coveragev1.GetCoverageRequest]) (*connect.Response[coveragev1.GetCoverageResponse], error) {
	projections, err := requestedProjections(req.Msg.GetProjections())
	if err != nil {
		return nil, err
	}
	snapshot, err := h.service.Snapshot(ctx, projections)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	response := &coveragev1.GetCoverageResponse{ComputedAt: timestamppb.New(snapshot.ComputedAt)}
	for _, finding := range snapshot.Findings {
		response.IntegrityFindings = append(response.IntegrityFindings, protoFinding(finding))
	}
	keys := make([]spacedoc.Projection, 0, len(snapshot.Projections))
	for projection := range snapshot.Projections {
		keys = append(keys, projection)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, projection := range keys {
		response.Projections = append(response.Projections, projectionCoverage(projection, snapshot.Projections[projection]))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ListCells(ctx context.Context, req *connect.Request[coveragev1.ListCellsRequest]) (*connect.Response[coveragev1.ListCellsResponse], error) {
	requested := []spacedoc.Projection(nil)
	if req.Msg.GetProjection() != coveragev1.Projection_PROJECTION_UNSPECIFIED {
		projection, err := parseProjection(req.Msg.GetProjection())
		if err != nil {
			return nil, err
		}
		requested = []spacedoc.Projection{projection}
	}
	snapshot, err := h.service.Snapshot(ctx, requested)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	response := &coveragev1.ListCellsResponse{}
	keys := make([]spacedoc.Projection, 0, len(snapshot.Projections))
	for projection := range snapshot.Projections {
		keys = append(keys, projection)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, projection := range keys {
		item := snapshot.Projections[projection]
		for _, cell := range item.Definition.Cells {
			if req.Msg.GetStatus() != coveragev1.CellStatus_CELL_STATUS_UNSPECIFIED && sharedv1.CellStatus(req.Msg.GetStatus()) != protoCellStatus(cell.Status) {
				continue
			}
			response.Cells = append(response.Cells, protoCell(projection, item.Definition, cell))
		}
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ListOpenLoopCells(ctx context.Context, req *connect.Request[coveragev1.ListCellsRequest]) (*connect.Response[coveragev1.ListCellsResponse], error) {
	return h.ListCells(ctx, connect.NewRequest(&coveragev1.ListCellsRequest{
		Projection: req.Msg.GetProjection(),
		Status:     coveragev1.CellStatus_CELL_STATUS_MISSING,
	}))
}

func (h *connectHandler) GetProjection(ctx context.Context, req *connect.Request[coveragev1.GetProjectionRequest]) (*connect.Response[coveragev1.GetProjectionResponse], error) {
	projection, err := parseProjection(req.Msg.GetProjection())
	if err != nil {
		return nil, err
	}
	snapshot, err := h.service.Snapshot(ctx, []spacedoc.Projection{projection})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	item, ok := snapshot.Projections[projection]
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("projection %q is unavailable", projection))
	}
	response := &coveragev1.GetProjectionResponse{Projection: protoProjection(projection), Coverage: projectionCoverage(projection, item)}
	for _, cell := range item.Definition.Cells {
		response.Cells = append(response.Cells, protoCell(projection, item.Definition, cell))
	}
	for _, bar := range item.Bars {
		response.Bars = append(response.Bars, protoBar(bar))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ValidateSetpoint(ctx context.Context, _ *connect.Request[coveragev1.ValidateSetpointRequest]) (*connect.Response[coveragev1.ValidateSetpointResponse], error) {
	snapshot, err := h.service.Snapshot(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	response := &coveragev1.ValidateSetpointResponse{Ok: len(snapshot.Findings) == 0}
	for _, finding := range snapshot.Findings {
		response.Findings = append(response.Findings, protoFinding(finding))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) GetDrift(ctx context.Context, req *connect.Request[coveragev1.GetDriftRequest]) (*connect.Response[coveragev1.GetDriftResponse], error) {
	requested := []spacedoc.Projection(nil)
	if req.Msg.GetProjection() != coveragev1.Projection_PROJECTION_UNSPECIFIED {
		projection, err := parseProjection(req.Msg.GetProjection())
		if err != nil {
			return nil, err
		}
		requested = []spacedoc.Projection{projection}
	}
	snapshot, err := h.service.Snapshot(ctx, requested)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	response := &coveragev1.GetDriftResponse{}
	for _, finding := range snapshot.Findings {
		response.Findings = append(response.Findings, &coveragev1.DriftFinding{
			Code:    finding.Code,
			Message: finding.Message,
			CellRef: finding.Location,
			Source:  "coverage",
		})
	}
	return connect.NewResponse(response), nil
}

func parseProjection(projection coveragev1.Projection) (spacedoc.Projection, error) {
	name := strings.TrimPrefix(projection.String(), "PROJECTION_")
	name = strings.ToLower(strings.ReplaceAll(name, "_", "-"))
	p := spacedoc.Projection(name)
	if !p.Valid() {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown projection %q", projection.String()))
	}
	return p, nil
}

func requestedProjections(values []coveragev1.Projection) ([]spacedoc.Projection, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]spacedoc.Projection, 0, len(values))
	for _, value := range values {
		projection, err := parseProjection(value)
		if err != nil {
			return nil, err
		}
		out = append(out, projection)
	}
	return out, nil
}

// protoProjection resolves a projection identifier against the proto enum's own
// value table rather than a hand-maintained parallel list. The parallel list is
// how the `substrate` projection shipped on 2026-08-20 and then rendered as
// `unspecified` on every typed surface: the map had ten entries, a miss returns
// the zero value, and an unlabelled projection is indistinguishable from a
// deliberate one. Deriving the number from the enum means adding a projection to
// the proto is the only step there is.
func protoProjection(projection spacedoc.Projection) coveragev1.Projection {
	name := "PROJECTION_" + strings.ToUpper(strings.ReplaceAll(string(projection), "-", "_"))
	if value, ok := coveragev1.Projection_value[name]; ok {
		return coveragev1.Projection(value)
	}
	return coveragev1.Projection_PROJECTION_UNSPECIFIED
}

func protoCell(projection spacedoc.Projection, def *spacedoc.SpaceDefinition, cell spacedoc.Cell) *sharedv1.Cell {
	gapDays := int32(0)
	if parsed, err := time.Parse("2006-01-02", cell.GapOpenedOn); err == nil {
		gapDays = int32(max(0, int(time.Since(parsed).Hours()/24)))
	}
	return &sharedv1.Cell{Id: cell.ID, Projection: sharedv1.Projection(protoProjection(projection)), Question: cell.Question, Owner: cell.Owner, LegUnit: def.LegUnit, Status: protoCellStatus(cell.Status), GapOpenedOn: cell.GapOpenedOn, GapOpenDays: gapDays, Notes: cell.Notes}
}

func protoCellStatus(status spacedoc.CellStatus) sharedv1.CellStatus {
	switch status {
	case spacedoc.StatusNow:
		return sharedv1.CellStatus_CELL_STATUS_NOW
	case spacedoc.StatusInReach:
		return sharedv1.CellStatus_CELL_STATUS_IN_REACH
	case spacedoc.StatusMissing:
		return sharedv1.CellStatus_CELL_STATUS_MISSING
	default:
		return sharedv1.CellStatus_CELL_STATUS_UNSPECIFIED
	}
}

func projectionCoverage(projection spacedoc.Projection, item internalcoverage.ProjectionSnapshot) *coveragev1.ProjectionCoverage {
	now, inReach, missing := 0, 0, 0
	for _, cell := range item.Definition.Cells {
		switch cell.Status {
		case spacedoc.StatusNow:
			now++
		case spacedoc.StatusInReach:
			inReach++
		case spacedoc.StatusMissing:
			missing++
		}
	}
	total := now + inReach + missing
	ratio := 0.0
	if total > 0 {
		ratio = float64(now) / float64(total)
	}
	confidence := &coveragev1.Confidence{Level: confidenceLevel(item.Definition.DenominatorConfidence), Rationale: item.Definition.ConfidenceRationale}
	return &coveragev1.ProjectionCoverage{Projection: protoProjection(projection), Ratio: &coveragev1.Ratio{Value: ratio, Confidence: confidence, Numerator: int32(now), Denominator: int32(total)}, NowCount: int32(now), InReachCount: int32(inReach), MissingCount: int32(missing), TotalCells: int32(total), Confidence: confidence, ComputedAt: timestamppb.Now(), Available: true}
}

func confidenceLevel(level spacedoc.DenominatorConfidence) coveragev1.ConfidenceLevel {
	switch level {
	case spacedoc.ConfidenceAuthoritative:
		return coveragev1.ConfidenceLevel_CONFIDENCE_LEVEL_AUTHORITATIVE
	case spacedoc.ConfidencePartial:
		return coveragev1.ConfidenceLevel_CONFIDENCE_LEVEL_PARTIAL
	default:
		return coveragev1.ConfidenceLevel_CONFIDENCE_LEVEL_SKETCH
	}
}

func protoBar(bar internalcoverage.Bar) *coveragev1.Bar {
	return &coveragev1.Bar{Id: bar.ID, CellRef: bar.CellRef, Projection: protoProjection(spacedoc.Projection(bar.Projection)), TargetKind: bar.TargetKind, Deadband: bar.Deadband, Sustain: bar.Sustain, Actuator: bar.Actuator, DecisionRef: bar.DecisionRef}
}

func protoFinding(f internalcoverage.IntegrityFinding) *coveragev1.IntegrityFinding {
	return &coveragev1.IntegrityFinding{Code: f.Code, Message: f.Message, Location: f.Location, Severity: f.Severity}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{ID: "coverage_get", Path: coveragev1connect.CoverageServiceGetCoverageProcedure, Method: "POST", Summary: "Read coverage ratios", Category: "coverage"},
	{ID: "coverage_cells", Path: coveragev1connect.CoverageServiceListCellsProcedure, Method: "POST", Summary: "List coverage cells", Category: "coverage"},
	{ID: "coverage_open_loop", Path: coveragev1connect.CoverageServiceListOpenLoopCellsProcedure, Method: "POST", Summary: "List dated open-loop cells", Category: "coverage"},
	{ID: "coverage_projection", Path: coveragev1connect.CoverageServiceGetProjectionProcedure, Method: "POST", Summary: "Read one projection", Category: "coverage"},
	{ID: "coverage_validate_setpoint", Path: coveragev1connect.CoverageServiceValidateSetpointProcedure, Method: "POST", Summary: "Validate the read-only setpoint", Category: "coverage"},
	{ID: "coverage_drift", Path: coveragev1connect.CoverageServiceGetDriftProcedure, Method: "POST", Summary: "Read coverage drift", Category: "coverage"},
}
