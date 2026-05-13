package flows

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"flow-verifier/internal/flows"
	"flow-verifier/internal/flows/kinds/temporal/contract"
	"flow-verifier/internal/flows/kinds/temporal/layout"
	"flow-verifier/internal/scenarios"

	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows"
)

// ScenariosService is the subset of scenarios.Service this handler
// depends on. Declared inline so the dependency arrow points inward:
// handlers/flows → internal/scenarios, not handlers/flows → handlers/scenarios.
type ScenariosService interface {
	List() ([]scenarios.Summary, error)
	Detail(id string) (scenarios.Detail, error)
}

// Deps wires the flows Connect handler's dependencies.
type Deps struct {
	Scenarios ScenariosService
	Logger    *log.Logger
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

func (h *connectHandler) ListFlows(_ context.Context, req *connect.Request[flowsv1.ListFlowsRequest]) (*connect.Response[flowsv1.ListFlowsResponse], error) {
	root := req.Msg.GetRoot()
	flowID := req.Msg.GetFlowId()
	kindFilter := req.Msg.GetKind()
	if root != "" {
		rows, err := flows.List(root, flowID, kindFilter)
		if err != nil {
			return nil, flows.ToConnectError(err)
		}
		return connect.NewResponse(&flowsv1.ListFlowsResponse{Flows: summariesToProto(rows)}), nil
	}
	if h.deps.Scenarios == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("scenarios service not configured"))
	}
	all, err := h.deps.Scenarios.List()
	if err != nil {
		return nil, scenarios.ToConnectError(err)
	}
	out := make([]flows.Summary, 0)
	for _, scenario := range all {
		if scenario.DiscoveryErr != "" || scenario.FlowCount == 0 {
			continue
		}
		detail, err := h.deps.Scenarios.Detail(scenario.ID)
		if err != nil {
			continue
		}
		out = append(out, stampScenarioID(detail.Flows, scenario.ID)...)
	}
	return connect.NewResponse(&flowsv1.ListFlowsResponse{Flows: summariesToProto(out)}), nil
}

func (h *connectHandler) GetFlow(_ context.Context, req *connect.Request[flowsv1.GetFlowRequest]) (*connect.Response[flowsv1.GetFlowResponse], error) {
	root := req.Msg.GetRoot()
	flowID := req.Msg.GetFlowId()
	if root == "" {
		if flowID == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("flow_id is required for GetFlow"))
		}
		if h.deps.Scenarios == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("root is required for GetFlow"))
		}
		resolved, err := h.resolveFlowRoot(flowID)
		if err != nil {
			return nil, err
		}
		root = resolved
	}
	detail, err := flows.Detail(root, flowID)
	if err != nil {
		connectErr := flows.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("flows.GetFlow(%q): %v", req.Msg.GetFlowId(), err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&flowsv1.GetFlowResponse{Flow: detailToProto(detail)}), nil
}

func (h *connectHandler) CreateFlow(_ context.Context, req *connect.Request[flowsv1.CreateFlowRequest]) (*connect.Response[flowsv1.CreateFlowResponse], error) {
	kindName := req.Msg.GetKind()
	var lang layout.Language
	if kindName == "" || kindName == "temporal" {
		l, err := languageFromString(req.Msg.GetLanguage())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		lang = l
	}
	dir, err := flows.New(flows.NewOptions{
		Root:      req.Msg.GetRoot(),
		ParentDir: req.Msg.GetParentDir(),
		FlowID:    req.Msg.GetFlowId(),
		Kind:      kindName,
		Language:  lang,
	})
	if err != nil {
		return nil, flows.ToConnectError(err)
	}
	return connect.NewResponse(&flowsv1.CreateFlowResponse{FlowDir: dir}), nil
}

func (h *connectHandler) ValidateFlow(_ context.Context, req *connect.Request[flowsv1.ValidateFlowRequest]) (*connect.Response[flowsv1.ValidateFlowResponse], error) {
	rows, err := flows.Validate(req.Msg.GetRoot(), req.Msg.GetFlowId(), "")
	if err != nil {
		return nil, flows.ToConnectError(err)
	}
	return connect.NewResponse(&flowsv1.ValidateFlowResponse{Flows: summariesToProto(rows)}), nil
}

func (h *connectHandler) ExplainFlow(_ context.Context, req *connect.Request[flowsv1.ExplainFlowRequest]) (*connect.Response[flowsv1.ExplainFlowResponse], error) {
	report, err := flows.Explain(req.Msg.GetRoot(), req.Msg.GetFlowId())
	if err != nil {
		return nil, flows.ToConnectError(err)
	}
	return connect.NewResponse(&flowsv1.ExplainFlowResponse{Report: report}), nil
}

func (h *connectHandler) CodegenFlow(_ context.Context, req *connect.Request[flowsv1.CodegenFlowRequest]) (*connect.Response[flowsv1.CodegenFlowResponse], error) {
	res, err := flows.Codegen(flows.CodegenOptions{
		Root:     req.Msg.GetRoot(),
		FlowID:   req.Msg.GetFlowId(),
		Language: req.Msg.GetLanguage(),
		Write:    req.Msg.GetWrite(),
	})
	if err != nil {
		return nil, flows.ToConnectError(err)
	}
	out := &flowsv1.CodegenFlowResponse{Written: append([]string(nil), res.Written...)}
	for _, a := range res.Artifacts {
		out.Artifacts = append(out.Artifacts, &flowsv1.CodegenArtifact{Path: a.Path, Content: a.Content})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ReconcileFlow(_ context.Context, req *connect.Request[flowsv1.ReconcileFlowRequest]) (*connect.Response[flowsv1.ReconcileFlowResponse], error) {
	res, err := flows.Reconcile(flows.ReconcileOptions{
		Root:         req.Msg.GetRoot(),
		FlowID:       req.Msg.GetFlowId(),
		ScenarioRoot: req.Msg.GetScenarioRoot(),
	})
	if err != nil {
		return nil, flows.ToConnectError(err)
	}
	out := &flowsv1.ReconcileFlowResponse{
		Passed:       res.Passed,
		FilesScanned: int32(res.FilesScanned),
	}
	for _, f := range res.Findings {
		out.Findings = append(out.Findings, &flowsv1.ReconcileFinding{
			Id:         f.ID,
			Severity:   f.Severity,
			Message:    f.Message,
			SourceFile: f.SourceFile,
			SourceLine: int32(f.SourceLine),
		})
	}
	return connect.NewResponse(out), nil
}

// resolveFlowRoot walks scenarios to find one whose flows include
// flowID, returning that scenario's path. Mirrors the cross-scenario
// aggregation ListFlows performs when root is empty.
func (h *connectHandler) resolveFlowRoot(flowID string) (string, error) {
	all, err := h.deps.Scenarios.List()
	if err != nil {
		return "", scenarios.ToConnectError(err)
	}
	for _, scenario := range all {
		if scenario.DiscoveryErr != "" || scenario.FlowCount == 0 {
			continue
		}
		detail, err := h.deps.Scenarios.Detail(scenario.ID)
		if err != nil {
			continue
		}
		for _, f := range detail.Flows {
			if f.FlowID == flowID {
				return scenario.Path, nil
			}
		}
	}
	return "", connect.NewError(connect.CodeNotFound, errors.New("flow not found: "+flowID))
}

func languageFromString(s string) (layout.Language, error) {
	switch s {
	case "go", "Go":
		return layout.LanguageGo, nil
	case "typescript", "ts", "TypeScript":
		return layout.LanguageTypeScript, nil
	case "":
		return "", errors.New("language is required (go|typescript)")
	}
	return "", errors.New("invalid language: must be one of go|typescript")
}

func summariesToProto(rows []flows.Summary) []*flowsv1.FlowSummary {
	out := make([]*flowsv1.FlowSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, summaryToProto(r))
	}
	return out
}

func summaryToProto(f flows.Summary) *flowsv1.FlowSummary {
	return &flowsv1.FlowSummary{
		FlowId:        f.FlowID,
		Kind:          f.Kind,
		ContractPath:  f.ContractPath,
		Language:      f.Language,
		SchemaVersion: int32(f.SchemaVer),
		ScenarioId:    f.ScenarioID,
	}
}

func detailToProto(d flows.FlowDetail) *flowsv1.FlowDetail {
	states := make([]*flowsv1.FlowState, 0, len(d.States))
	for _, s := range d.States {
		states = append(states, &flowsv1.FlowState{
			Id:       s.ID,
			Quint:    s.Quint,
			Initial:  s.Initial,
			Terminal: s.Terminal,
		})
	}
	events := make([]*flowsv1.FlowEvent, 0, len(d.Events))
	for _, e := range d.Events {
		events = append(events, &flowsv1.FlowEvent{Id: e.ID, Quint: e.Quint})
	}
	transitions := make([]*flowsv1.FlowTransition, 0, len(d.Transitions))
	for _, t := range d.Transitions {
		transitions = append(transitions, transitionToProto(t))
	}
	traces := make([]*flowsv1.FlowTrace, 0, len(d.Traces))
	for _, t := range d.Traces {
		steps := make([]*flowsv1.FlowTraceStep, 0, len(t.Steps))
		for _, s := range t.Steps {
			steps = append(steps, &flowsv1.FlowTraceStep{
				Event:     s.Event,
				Want:      s.Want,
				WantError: s.WantError,
			})
		}
		traces = append(traces, &flowsv1.FlowTrace{Name: t.Name, Initial: t.Initial, Steps: steps})
	}
	invariants := make([]*flowsv1.FlowInvariant, 0, len(d.Invariants))
	for _, inv := range d.Invariants {
		invariants = append(invariants, &flowsv1.FlowInvariant{
			Id:          inv.ID,
			Quint:       inv.Quint,
			Description: inv.Description,
			Expression:  inv.Expression,
		})
	}
	return &flowsv1.FlowDetail{
		FlowId:        d.FlowID,
		Kind:          d.Kind,
		Domain:        d.Domain,
		Description:   d.Description,
		ContractPath:  d.ContractPath,
		Language:      d.Language,
		SchemaVersion: int32(d.SchemaVersion),
		InitialState:  d.InitialState,
		States:        states,
		Events:        events,
		Transitions:   transitions,
		Traces:        traces,
		Invariants:    invariants,
		Model:         modelToProto(d.Model),
		Runtime:       runtimeToProto(d.Runtime),
		Report:        d.Report,
	}
}

// transitionToProto converts a model.Transition (interface{} to keep
// this file's import surface tight) into the proto shape.
func transitionToProto(t any) *flowsv1.FlowTransition {
	tr := concreteTransition(t)
	return &flowsv1.FlowTransition{
		From:         tr.from,
		Event:        tr.event,
		To:           tr.to,
		WantError:    tr.wantError,
		WantErrorSet: tr.wantErrorSet,
	}
}

func modelToProto(m contract.Model) *flowsv1.FlowModel {
	return &flowsv1.FlowModel{
		Module:     m.Module,
		Seed:       m.Seed,
		MaxSteps:   int32(m.MaxSteps),
		TraceCount: int32(m.TraceCount),
		Verify:     &flowsv1.FlowVerify{Invariants: append([]string(nil), m.Verify.Invariants...)},
	}
}

func runtimeToProto(r contract.Runtime) *flowsv1.FlowRuntime {
	out := &flowsv1.FlowRuntime{
		SideEffects:     append([]string(nil), r.SideEffects...),
		StaleCompletion: r.StaleCompletion,
	}
	if r.Go != nil {
		out.Go = &flowsv1.GoRuntime{
			Package:        r.Go.Package,
			StatusType:     r.Go.StatusType,
			EventType:      r.Go.EventType,
			ConstantPrefix: r.Go.ConstantPrefix,
		}
	}
	if r.TypeScript != nil {
		out.Typescript = &flowsv1.TypeScriptRuntime{
			StatusType:             r.TypeScript.StatusType,
			EventType:              r.TypeScript.EventType,
			StatusesConst:          r.TypeScript.StatusesConst,
			EventsConst:            r.TypeScript.EventsConst,
			FormalExpectationConst: r.TypeScript.FormalExpectationConst,
			StateUnionType:         r.TypeScript.StateUnionType,
			EventUnionType:         r.TypeScript.EventUnionType,
			PayloadTypes:           copyStringMap(r.TypeScript.PayloadTypes),
			StateVariants:          wrapVariantMap(r.TypeScript.StateVariants),
			EventVariants:          wrapVariantMap(r.TypeScript.EventVariants),
		}
	}
	return out
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func wrapVariantMap(m map[string]map[string]string) map[string]*flowsv1.VariantFields {
	if m == nil {
		return nil
	}
	out := make(map[string]*flowsv1.VariantFields, len(m))
	for k, inner := range m {
		out[k] = &flowsv1.VariantFields{Fields: copyStringMap(inner)}
	}
	return out
}

// stampScenarioID stamps a scenario id onto every flow summary whose
// own ScenarioID is empty (matches the prior REST aggregator's shape).
func stampScenarioID(rows []flows.Summary, scenarioID string) []flows.Summary {
	out := make([]flows.Summary, len(rows))
	for i, r := range rows {
		out[i] = r
		if out[i].ScenarioID == "" {
			out[i].ScenarioID = scenarioID
		}
	}
	return out
}
