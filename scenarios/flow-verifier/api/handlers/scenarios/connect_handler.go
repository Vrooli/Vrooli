package scenarios

import (
	"context"
	"log"

	"connectrpc.com/connect"

	"flow-verifier/internal/flows"
	"flow-verifier/internal/scenarios"

	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows"
	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"
)

// Service is the in-process surface this handler depends on (kept in
// the same interface shape it had in the REST handler so main.go's
// wiring is unchanged).
type Service interface {
	List() ([]scenarios.Summary, error)
	Detail(id string) (scenarios.Detail, error)
	Root() string
}

// Deps wires the scenarios Connect handler's dependencies.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListScenarios(_ context.Context, _ *connect.Request[scenariosv1.ListScenariosRequest]) (*connect.Response[scenariosv1.ListScenariosResponse], error) {
	rows, err := h.deps.Service.List()
	if err != nil {
		h.deps.Logger.Printf("scenarios.ListScenarios: %v", err)
		return nil, scenarios.ToConnectError(err)
	}
	out := &scenariosv1.ListScenariosResponse{
		VrooliRoot: h.deps.Service.Root(),
		Scenarios:  make([]*scenariosv1.ScenarioSummary, 0, len(rows)),
	}
	for _, s := range rows {
		out.Scenarios = append(out.Scenarios, scenarioSummaryToProto(s))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetScenario(_ context.Context, req *connect.Request[scenariosv1.GetScenarioRequest]) (*connect.Response[scenariosv1.GetScenarioResponse], error) {
	detail, err := h.deps.Service.Detail(req.Msg.GetId())
	if err != nil {
		connectErr := scenarios.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("scenarios.GetScenario(%q): %v", req.Msg.GetId(), err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&scenariosv1.GetScenarioResponse{Scenario: detailToProto(detail)}), nil
}

func scenarioSummaryToProto(s scenarios.Summary) *scenariosv1.ScenarioSummary {
	return &scenariosv1.ScenarioSummary{
		Id:             s.ID,
		DisplayName:    s.DisplayName,
		Description:    s.Description,
		Path:           s.Path,
		FlowCount:      int32(s.FlowCount),
		DiscoveryError: s.DiscoveryErr,
	}
}

func detailToProto(d scenarios.Detail) *scenariosv1.ScenarioDetail {
	out := &scenariosv1.ScenarioDetail{
		Summary: scenarioSummaryToProto(d.Summary),
		Flows:   make([]*flowsv1.FlowSummary, 0, len(d.Flows)),
	}
	for _, f := range d.Flows {
		out.Flows = append(out.Flows, flowSummaryToProto(f))
	}
	return out
}

// flowSummaryToProto translates flows.Summary → proto. Kept here (and
// re-implemented in handlers/flows/connect_handler.go) rather than
// extracted to avoid an inward-pointing import cycle.
func flowSummaryToProto(f flows.Summary) *flowsv1.FlowSummary {
	return &flowsv1.FlowSummary{
		FlowId:        f.FlowID,
		ContractPath:  f.ContractPath,
		Language:      f.Language,
		SchemaVersion: int32(f.SchemaVer),
		ScenarioId:    f.ScenarioID,
	}
}
