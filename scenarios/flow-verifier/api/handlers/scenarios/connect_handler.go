package scenarios

import (
	"context"
	"log"

	"connectrpc.com/connect"

	"flow-verifier/internal/flows"
	"flow-verifier/internal/lifecycleinference"
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
	ListLifecycles(ctx context.Context, scenarioID string, huntIndexJSON []byte) (lifecycleinference.Inventory, error)
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

func (h *connectHandler) ListLifecycles(ctx context.Context, req *connect.Request[scenariosv1.ListLifecyclesRequest]) (*connect.Response[scenariosv1.ListLifecyclesResponse], error) {
	inventory, err := h.deps.Service.ListLifecycles(ctx, req.Msg.GetScenarioId(), req.Msg.GetHuntIndexJson())
	if err != nil {
		connectErr := scenarios.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("scenarios.ListLifecycles(%q): %v", req.Msg.GetScenarioId(), err)
		}
		return nil, connectErr
	}
	out := &scenariosv1.ListLifecyclesResponse{
		VrooliRoot:                 inventory.Root,
		ScenarioCount:              int32(inventory.ScenarioCount),
		CandidateCount:             int32(inventory.CandidateCount),
		HuntCoverageComplete:       inventory.HuntCoverageComplete,
		HuntCoverageScope:          inventory.HuntCoverageScope,
		PreselectionCandidateCount: int32(inventory.PreselectionCandidateCount),
		Scenarios:                  make([]*scenariosv1.LifecycleScenarioSummary, 0, len(inventory.Scenarios)),
		Candidates:                 make([]*scenariosv1.LifecycleCandidate, 0, len(inventory.Candidates)),
		PreselectionCandidates:     make([]*scenariosv1.LifecycleCandidate, 0, len(inventory.PreselectionCandidates)),
		UnrankedCandidates:         make([]*scenariosv1.LifecycleCandidate, 0, len(inventory.UnrankedCandidates)),
	}
	for _, scenario := range inventory.Scenarios {
		out.ParseErrorCount += int32(scenario.ParseErrors)
		out.FlowContractParseErrorCount += int32(scenario.FlowParseErrors)
		out.Scenarios = append(out.Scenarios, &scenariosv1.LifecycleScenarioSummary{
			Id: scenario.ID, SourceFiles: int32(scenario.SourceFiles), ParseErrors: int32(scenario.ParseErrors),
			CandidateCount: int32(scenario.CandidateCount), ExplicitFlowContracts: int32(scenario.ExplicitFlows),
			FlowContractParseErrors: int32(scenario.FlowParseErrors),
		})
	}
	for _, candidate := range inventory.Candidates {
		out.Candidates = append(out.Candidates, lifecycleCandidateToProto(candidate))
	}
	for _, candidate := range inventory.PreselectionCandidates {
		out.PreselectionCandidates = append(out.PreselectionCandidates, lifecycleCandidateToProto(candidate))
	}
	for _, candidate := range inventory.UnrankedCandidates {
		out.UnrankedCandidates = append(out.UnrankedCandidates, lifecycleCandidateToProto(candidate))
	}
	return connect.NewResponse(out), nil
}

func lifecycleCandidateToProto(candidate lifecycleinference.Candidate) *scenariosv1.LifecycleCandidate {
	out := &scenariosv1.LifecycleCandidate{
		Id: candidate.ID, ScenarioId: candidate.ScenarioID, Package: candidate.Package,
		TypeName: candidate.TypeName, FieldName: candidate.FieldName, FieldType: candidate.FieldType,
		Declaration:  lifecycleEvidenceToProto(candidate.Declaration),
		RelatedFlags: lifecycleEvidenceListToProto(candidate.RelatedFlags),
		TimeFields:   lifecycleEvidenceListToProto(candidate.TimeFields),
		WriteSites:   lifecycleEvidenceListToProto(candidate.WriteSites),
		GuardSites:   lifecycleEvidenceListToProto(candidate.GuardSites),
		TestFiles:    append([]string(nil), candidate.TestFiles...),
		BurdenScore:  int32(candidate.BurdenScore), ScoreBreakdown: append([]string(nil), candidate.ScoreBreakdown...),
		Confidence:              candidate.Confidence,
		States:                  make([]*scenariosv1.LifecycleStateEvidence, 0, len(candidate.States)),
		HuntRecords:             make([]*scenariosv1.LifecycleHuntReference, 0, len(candidate.HuntRecords)),
		ExplicitFlows:           make([]*scenariosv1.LifecycleFlowReference, 0, len(candidate.ExplicitFlows)),
		TransitionOwners:        lifecycleEvidenceListToProto(candidate.TransitionOwners),
		ExplicitlyModeled:       candidate.ExplicitlyModeled,
		PersistenceClass:        string(candidate.Persistence.Class),
		PersistenceReason:       candidate.Persistence.Reason,
		PersistenceEvidence:     lifecycleEvidenceListToProto(candidate.Persistence.Evidence),
		PersistenceScope:        candidate.Persistence.Scope,
		PersistenceFilesScanned: int32(candidate.Persistence.FilesScanned),
		PersistenceScanComplete: candidate.Persistence.ScanComplete,
		PreselectionStatus:      candidate.PreselectionStatus,
	}
	for _, state := range candidate.States {
		out.States = append(out.States, &scenariosv1.LifecycleStateEvidence{Name: state.Name, Value: state.Value, At: lifecycleEvidenceToProto(state.At)})
	}
	for _, record := range candidate.HuntRecords {
		out.HuntRecords = append(out.HuntRecords, &scenariosv1.LifecycleHuntReference{EntryId: record.EntryID, Round: record.Round, Status: record.Status})
	}
	for _, flow := range candidate.ExplicitFlows {
		out.ExplicitFlows = append(out.ExplicitFlows, &scenariosv1.LifecycleFlowReference{
			FlowId: flow.FlowID, Path: flow.Path, StatusType: flow.StatusType,
			MatchedStates:       append([]string(nil), flow.MatchedStates...),
			CandidateStateCount: int32(flow.CandidateStateCount), ContractStateCount: int32(flow.ContractStateCount),
			ExactLifecycleMatch: flow.ExactLifecycleMatch,
		})
	}
	return out
}

func lifecycleEvidenceListToProto(rows []lifecycleinference.Evidence) []*scenariosv1.LifecycleEvidence {
	out := make([]*scenariosv1.LifecycleEvidence, 0, len(rows))
	for _, row := range rows {
		out = append(out, lifecycleEvidenceToProto(row))
	}
	return out
}

func lifecycleEvidenceToProto(row lifecycleinference.Evidence) *scenariosv1.LifecycleEvidence {
	return &scenariosv1.LifecycleEvidence{Path: row.Path, Line: int32(row.Line), Text: row.Text}
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
