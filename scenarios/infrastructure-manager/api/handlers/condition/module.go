package condition

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/spacedoc"
	conditionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/condition"
	conditionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/condition/condition_v1connect"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/shared"
	internalcondition "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/condition"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/module"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Module() module.Module { return ModuleWithDeps("", nil, nil) }

func Schema() string { return internalcondition.Schema() }

func ModuleWithDeps(root string, db *database.RoutedDB, clk schedule.Clock) module.Module {
	mod, _ := ModuleWithService(root, db, clk)
	return mod
}

// ModuleWithService returns the module alongside the condition service it
// serves, so the focus domain ranks the same readings this domain reports
// rather than constructing a second service that could drift from it.
func ModuleWithService(root string, db *database.RoutedDB, clk schedule.Clock) (module.Module, *internalcondition.Service) {
	service := internalcondition.NewConfiguredService(root, db, clk)
	path, handler := conditionconnect.NewConditionServiceHandler(&connectHandler{service: service})
	return module.Module{Name: "condition", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}, service
}

type connectHandler struct{ service *internalcondition.Service }

func (h *connectHandler) GetCondition(ctx context.Context, req *connect.Request[conditionv1.GetConditionRequest]) (*connect.Response[conditionv1.GetConditionResponse], error) {
	projection, snapshot := h.readProjection(ctx, req.Msg.GetProjection())
	response := &conditionv1.GetConditionResponse{ComputedAt: timestamppb.New(snapshot.At)}
	for _, reading := range snapshot.Readings {
		if cell := strings.TrimSpace(req.Msg.GetCellRef()); cell != "" && cell != reading.CellRef {
			continue
		}
		response.Readings = append(response.Readings, protoReading(reading))
		response.Legs = append(response.Legs, &conditionv1.Leg{CellRef: reading.CellRef, Projection: coverageProjection(projectionOf(reading.CellRef)), Owner: "vrooli-autoheal", Unit: reading.Unit, Source: reading.Source})
	}
	for _, source := range snapshot.Sources {
		response.Sources = append(response.Sources, protoSource(source))
	}
	if projection != "" {
		for _, source := range internalcondition.PeerSourceAvailability(projection, snapshot.At) {
			if source.Source == "vrooli-autoheal" {
				continue
			}
			response.Sources = append(response.Sources, protoSource(source))
		}
	}
	return connect.NewResponse(response), nil
}

// readProjection resolves an unspecified projection to the whole readable
// surface. Reading nothing for an unspecified request made the trust triple
// report an empty distribution — a denominator of zero that looks like a clean
// sheet rather than an unasked question.
func (h *connectHandler) readProjection(ctx context.Context, projection coveragev1.Projection) (string, internalcondition.Snapshot) {
	if projection == coveragev1.Projection_PROJECTION_UNSPECIFIED {
		return "", h.service.ReadAll(ctx)
	}
	name := projectionName(projection)
	return name, h.service.Read(ctx, name)
}

// projectionOf recovers the projection from a cell reference so a merged
// multi-projection read labels each leg with its own projection.
func projectionOf(cellRef string) string {
	if index := strings.Index(cellRef, "/"); index > 0 {
		return cellRef[:index]
	}
	return cellRef
}

func (h *connectHandler) GetTrustDistribution(ctx context.Context, req *connect.Request[conditionv1.GetTrustDistributionRequest]) (*connect.Response[conditionv1.GetTrustDistributionResponse], error) {
	_, snapshot := h.readProjection(ctx, req.Msg.GetProjection())
	trust := &conditionv1.TrustTriple{CheckedDenominator: int32(snapshot.Trust.CheckedDenominator), Total: int32(snapshot.Trust.Total), CheckedAt: timestamppb.New(snapshot.Trust.CheckedAt)}
	verdicts := make([]internalcondition.TrustVerdict, 0, len(snapshot.Trust.Distribution))
	for verdict := range snapshot.Trust.Distribution {
		verdicts = append(verdicts, verdict)
	}
	sort.Slice(verdicts, func(i, j int) bool { return verdicts[i] < verdicts[j] })
	for _, verdict := range verdicts {
		count := snapshot.Trust.Distribution[verdict]
		trust.Distribution = append(trust.Distribution, &conditionv1.TrustCount{Verdict: protoTrust(verdict), Count: int32(count)})
	}
	return connect.NewResponse(&conditionv1.GetTrustDistributionResponse{Trust: trust}), nil
}

func (h *connectHandler) ExplainCell(ctx context.Context, req *connect.Request[conditionv1.ExplainCellRequest]) (*connect.Response[conditionv1.ExplainCellResponse], error) {
	snapshot := h.service.ReadAll(ctx)
	for _, reading := range snapshot.Readings {
		if reading.CellRef == req.Msg.GetCellRef() || reading.ID == req.Msg.GetCellRef() {
			return connect.NewResponse(&conditionv1.ExplainCellResponse{Cell: &sharedv1.Cell{Id: reading.CellRef, Projection: sharedProjection(projectionOf(reading.CellRef)), Question: explainQuestion(reading.CellRef), Owner: "vrooli-autoheal", LegUnit: reading.Unit}, Reading: protoReading(reading)}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("condition cell %q was not found", req.Msg.GetCellRef()))
}

func (h *connectHandler) GetHistory(ctx context.Context, req *connect.Request[conditionv1.GetHistoryRequest]) (*connect.Response[conditionv1.GetHistoryResponse], error) {
	history := h.service.History(ctx, req.Msg.GetCellRef(), int(req.Msg.GetLimit()))
	response := &conditionv1.GetHistoryResponse{Measurable: history.Measurable, UnmeasurableReason: history.UnmeasurableReason}
	for _, reading := range history.Readings {
		response.Readings = append(response.Readings, protoReading(reading))
	}
	return connect.NewResponse(response), nil
}

func projectionName(projection coveragev1.Projection) string {
	name := strings.ToLower(strings.TrimPrefix(projection.String(), "PROJECTION_"))
	return strings.ReplaceAll(name, "_", "-")
}

// coverageProjection maps a projection name onto the generated enum by the
// shared naming rule, so a projection added to the space set does not silently
// render as UNSPECIFIED here.
func coverageProjection(projection string) coveragev1.Projection {
	name := "PROJECTION_" + strings.ToUpper(strings.ReplaceAll(projection, "-", "_"))
	if value, ok := coveragev1.Projection_value[name]; ok {
		return coveragev1.Projection(value)
	}
	return coveragev1.Projection_PROJECTION_UNSPECIFIED
}

func sharedProjection(projection string) sharedv1.Projection {
	name := "PROJECTION_" + strings.ToUpper(strings.ReplaceAll(projection, "-", "_"))
	if value, ok := sharedv1.Projection_value[name]; ok {
		return sharedv1.Projection(value)
	}
	return sharedv1.Projection_PROJECTION_UNSPECIFIED
}

// explainQuestion names what a cell answers. The availability cell is the one
// per-check aggregate; substrate cells are per-subsystem and are named by the
// space document that authors them.
func explainQuestion(cellRef string) string {
	if projectionOf(cellRef) == string(spacedoc.ProjectionSubstrate) {
		return "Host, kernel and device condition for " + cellRef
	}
	return "Per-check uptime trend"
}

func protoTrust(verdict internalcondition.TrustVerdict) conditionv1.TrustVerdict {
	switch verdict {
	case internalcondition.TrustValid:
		return conditionv1.TrustVerdict_TRUST_VERDICT_VALID
	case internalcondition.TrustGhost:
		return conditionv1.TrustVerdict_TRUST_VERDICT_GHOST
	case internalcondition.TrustSaturated:
		return conditionv1.TrustVerdict_TRUST_VERDICT_SATURATED
	case internalcondition.TrustShelved:
		return conditionv1.TrustVerdict_TRUST_VERDICT_SHELVED
	case internalcondition.TrustUnitMismatch:
		return conditionv1.TrustVerdict_TRUST_VERDICT_UNIT_MISMATCH
	case internalcondition.TrustUnavailable:
		return conditionv1.TrustVerdict_TRUST_VERDICT_UNAVAILABLE
	default:
		return conditionv1.TrustVerdict_TRUST_VERDICT_UNTRUSTED
	}
}

func protoBand(verdict internalcondition.BandVerdict) conditionv1.BandVerdict {
	switch verdict {
	case internalcondition.BandInBand:
		return conditionv1.BandVerdict_BAND_VERDICT_IN_BAND
	case internalcondition.BandOutOfBand:
		return conditionv1.BandVerdict_BAND_VERDICT_OUT_OF_BAND
	case internalcondition.BandPendingSustain:
		return conditionv1.BandVerdict_BAND_VERDICT_PENDING_SUSTAIN
	case internalcondition.BandNeedsBaseline:
		return conditionv1.BandVerdict_BAND_VERDICT_NEEDS_BASELINE
	case internalcondition.BandNotGradeable:
		return conditionv1.BandVerdict_BAND_VERDICT_NOT_GRADEABLE
	default:
		return conditionv1.BandVerdict_BAND_VERDICT_NOT_EVALUATED
	}
}

func protoReading(reading internalcondition.Observation) *conditionv1.Reading {
	verdict := reading.BandVerdict
	if verdict == "" {
		verdict = internalcondition.EvaluateBand(reading.Value, reading.Trust, reading.Band)
	}
	return &conditionv1.Reading{
		Id: reading.ID, CellRef: reading.CellRef, Value: reading.Value, Unit: reading.Unit,
		Source: reading.Source, ObservedAt: timestamppb.New(reading.ObservedAt),
		TrustVerdict: protoTrust(reading.Trust), BandVerdict: protoBand(verdict),
		UnavailableReason: reading.UnavailableReason,
		BandExplanation:   reading.Band.NotGradeableReason,
		OutOfScope:        reading.OutOfScope,
	}
}

func protoSource(source internalcondition.SourceAvailability) *conditionv1.SourceAvailability {
	return &conditionv1.SourceAvailability{Source: source.Source, Available: source.Available, Reason: source.Reason, CheckedAt: timestamppb.New(source.CheckedAt)}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "condition_get", Path: conditionconnect.ConditionServiceGetConditionProcedure, Method: "POST", Summary: "Read condition", Category: "condition"},
	{ID: "condition_trust", Path: conditionconnect.ConditionServiceGetTrustDistributionProcedure, Method: "POST", Summary: "Read the trust triple", Category: "condition"},
	{ID: "condition_explain", Path: conditionconnect.ConditionServiceExplainCellProcedure, Method: "POST", Summary: "Explain one condition cell", Category: "condition"},
	{ID: "condition_history", Path: conditionconnect.ConditionServiceGetHistoryProcedure, Method: "POST", Summary: "Read condition history", Category: "condition"},
}

func _unused(_ *timestamppb.Timestamp) {}
