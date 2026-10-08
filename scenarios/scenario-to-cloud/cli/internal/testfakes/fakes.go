// Package testfakes provides in-memory Connect service handlers shaped like
// the scenario-to-cloud API for CLI tests: deployments (selector resolution
// with typed ambiguity), plans (compile/apply) and operations (standing,
// wait, cancel, list). Tests mount them on an httptest server and drive the
// real command code against them.
package testfakes

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"connectrpc.com/connect"
	deploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/errors"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/identity"
	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations/operationsv1connect"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans/plansv1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

// TypedError builds a Connect error carrying the errors.v1.Error detail the
// API attaches, so the CLI decodes the same stable code it would in
// production.
func TypedError(code connect.Code, stable, message string, details map[string]any) *connect.Error {
	var next *errorsv1.NextAction
	if stable == "deployment_selector_ambiguous" {
		next = &errorsv1.NextAction{Owner: "scenario-to-cloud", Kind: "selector", Reference: "id", Label: "Select by deployment id"}
	}
	return TypedErrorWithNextAction(code, stable, message, details, next)
}

// TypedErrorWithNextAction is TypedError with an explicit next action.
func TypedErrorWithNextAction(code connect.Code, stable, message string, details map[string]any, next *errorsv1.NextAction) *connect.Error {
	cerr := connect.NewError(code, errors.New(stable+": "+message))
	wire := &errorsv1.Error{Code: stable, Message: message, NextAction: next}
	if details != nil {
		if s, err := structpb.NewStruct(details); err == nil {
			wire.Details = s
		}
	}
	if detail, err := connect.NewErrorDetail(wire); err == nil {
		cerr.AddDetail(detail)
	}
	return cerr
}

// Deployment is one fake record.
type Deployment struct {
	Ref       *identityv1.DeploymentRef
	Name      string
	Status    string
	Domain    string
	BundleSHA string
	Manifest  map[string]any
}

// Ref builds a deployment reference for tests.
func Ref(id, scenario, environment, host string) *identityv1.DeploymentRef {
	return &identityv1.DeploymentRef{
		Id: id, ScenarioId: scenario, Environment: environment,
		Target: &identityv1.TargetRef{Transport: "ssh", Locator: &identityv1.TargetLocator{Host: host, Port: 22, User: "root", Workdir: "/root/Vrooli"}},
	}
}

// Deployments is the fake DeploymentsService.
type Deployments struct {
	deploymentsv1connect.UnimplementedDeploymentsServiceHandler
	mu    sync.Mutex
	Items []Deployment
	Calls []string
	// ListError, when set, is returned by ListDeployments.
	ListError error
}

func (d *Deployments) record(call string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Calls = append(d.Calls, call)
}

func (d *Deployments) matches(sel *deploymentsv1.DeploymentSelector) []Deployment {
	var out []Deployment
	for _, item := range d.Items {
		ref := item.Ref
		switch {
		case sel.GetId() != "":
			if ref.GetId() == sel.GetId() {
				out = append(out, item)
			}
		case sel.GetScenarioId() != ref.GetScenarioId():
		case sel.GetEnvironment() != "" && sel.GetEnvironment() == ref.GetEnvironment():
			out = append(out, item)
		case sel.GetDomain() != "" && sel.GetDomain() == item.Domain:
			out = append(out, item)
		case sel.GetHost() != "" && sel.GetHost() == ref.GetTarget().GetLocator().GetHost():
			out = append(out, item)
		}
	}
	return out
}

// ResolveDeployment implements the selector grammar of api/identity.
func (d *Deployments) ResolveDeployment(_ context.Context, req *connect.Request[deploymentsv1.ResolveDeploymentRequest]) (*connect.Response[deploymentsv1.ResolveDeploymentResponse], error) {
	d.record("ResolveDeployment")
	sel := req.Msg.GetSelector()
	matches := d.matches(sel)
	switch len(matches) {
	case 0:
		return nil, TypedError(connect.CodeNotFound, "deployment_not_found", "No deployment matches the selector", nil)
	case 1:
		return connect.NewResponse(&deploymentsv1.ResolveDeploymentResponse{SchemaVersion: "1", Ref: matches[0].Ref}), nil
	}
	candidates := make([]any, 0, len(matches))
	for _, m := range matches {
		candidates = append(candidates, map[string]any{"id": m.Ref.GetId(), "scenario_id": m.Ref.GetScenarioId(), "environment": m.Ref.GetEnvironment(), "target_key": "host:" + m.Ref.GetTarget().GetLocator().GetHost()})
	}
	return nil, TypedError(connect.CodeAborted, "deployment_selector_ambiguous", "Selector matches more than one deployment; select by id or add environment", map[string]any{"candidates": candidates})
}

// GetDeployment returns the record.
func (d *Deployments) GetDeployment(_ context.Context, req *connect.Request[deploymentsv1.GetDeploymentRequest]) (*connect.Response[deploymentsv1.GetDeploymentResponse], error) {
	d.record("GetDeployment")
	for _, item := range d.Items {
		if item.Ref.GetId() == req.Msg.GetId() {
			manifest, _ := structpb.NewStruct(item.Manifest)
			return connect.NewResponse(&deploymentsv1.GetDeploymentResponse{SchemaVersion: "1", Deployment: &deploymentsv1.Deployment{
				Ref: item.Ref, Name: item.Name, Status: item.Status, Domain: item.Domain, BundleSha256: item.BundleSHA, Manifest: manifest,
			}}), nil
		}
	}
	return nil, TypedError(connect.CodeNotFound, "deployment_not_found", "Deployment not found", nil)
}

// ListDeployments filters by scenario, environment and status.
func (d *Deployments) ListDeployments(_ context.Context, req *connect.Request[deploymentsv1.ListDeploymentsRequest]) (*connect.Response[deploymentsv1.ListDeploymentsResponse], error) {
	d.record("ListDeployments")
	if d.ListError != nil {
		return nil, d.ListError
	}
	resp := &deploymentsv1.ListDeploymentsResponse{SchemaVersion: "1"}
	for _, item := range d.Items {
		if req.Msg.GetScenarioId() != "" && req.Msg.GetScenarioId() != item.Ref.GetScenarioId() {
			continue
		}
		if req.Msg.GetEnvironment() != "" && req.Msg.GetEnvironment() != item.Ref.GetEnvironment() {
			continue
		}
		if req.Msg.GetStatus() != "" && req.Msg.GetStatus() != item.Status {
			continue
		}
		resp.Deployments = append(resp.Deployments, &deploymentsv1.DeploymentSummary{Ref: item.Ref, Name: item.Name, Status: item.Status, Domain: item.Domain})
	}
	return connect.NewResponse(resp), nil
}

// Plans is the fake PlansService.
type Plans struct {
	plansv1connect.UnimplementedPlansServiceHandler
	mu sync.Mutex
	// Compile answers CompilePlan; a nil function returns CompileError.
	Compile      func(deploymentID, scope string) *plansv1.CompilePlanResponse
	CompileError error
	// Apply answers ApplyPlan; a nil function returns ApplyError.
	Apply      func(req *plansv1.ApplyPlanRequest) *plansv1.ApplyPlanResponse
	ApplyError error
	Applied    []*plansv1.ApplyPlanRequest
	Compiled   []string
}

// CompilePlan implements plansv1connect.PlansServiceHandler.
func (p *Plans) CompilePlan(_ context.Context, req *connect.Request[plansv1.CompilePlanRequest]) (*connect.Response[plansv1.CompilePlanResponse], error) {
	p.mu.Lock()
	p.Compiled = append(p.Compiled, req.Msg.GetDeploymentId())
	p.mu.Unlock()
	if p.Compile == nil {
		if p.CompileError != nil {
			return nil, p.CompileError
		}
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("compile not scripted"))
	}
	return connect.NewResponse(p.Compile(req.Msg.GetDeploymentId(), req.Msg.GetScope())), nil
}

// ApplyPlan implements plansv1connect.PlansServiceHandler.
func (p *Plans) ApplyPlan(_ context.Context, req *connect.Request[plansv1.ApplyPlanRequest]) (*connect.Response[plansv1.ApplyPlanResponse], error) {
	p.mu.Lock()
	p.Applied = append(p.Applied, req.Msg)
	p.mu.Unlock()
	if p.ApplyError != nil {
		return nil, p.ApplyError
	}
	if p.Apply == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("apply not scripted"))
	}
	return connect.NewResponse(p.Apply(req.Msg)), nil
}

// Operations is the fake OperationsService. Standings are answered in
// order: Get and Wait pop from Script when it has more than one entry, so a
// test can make a wait return still_pending first and succeeded afterwards.
type Operations struct {
	operationsv1connect.UnimplementedOperationsServiceHandler
	mu         sync.Mutex
	Script     []*operationsv1.OperationStanding
	WaitCalls  []*operationsv1.WaitOperationRequest
	GetCalls   []string
	Cancelled  []string
	Listed     []string
	ListAnswer *operationsv1.ListDeploymentOperationsResponse
}

func (o *Operations) next() *operationsv1.OperationStanding {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.Script) == 0 {
		return &operationsv1.OperationStanding{SchemaVersion: "1", State: "unknown"}
	}
	st := o.Script[0]
	if len(o.Script) > 1 {
		o.Script = o.Script[1:]
	}
	return st
}

// GetOperation implements the handler.
func (o *Operations) GetOperation(_ context.Context, req *connect.Request[operationsv1.GetOperationRequest]) (*connect.Response[operationsv1.OperationStanding], error) {
	o.mu.Lock()
	o.GetCalls = append(o.GetCalls, req.Msg.GetOperationId())
	o.mu.Unlock()
	return connect.NewResponse(o.next()), nil
}

// WaitOperation implements the handler.
func (o *Operations) WaitOperation(_ context.Context, req *connect.Request[operationsv1.WaitOperationRequest]) (*connect.Response[operationsv1.OperationStanding], error) {
	o.mu.Lock()
	o.WaitCalls = append(o.WaitCalls, req.Msg)
	o.mu.Unlock()
	return connect.NewResponse(o.next()), nil
}

// CancelOperation implements the handler.
func (o *Operations) CancelOperation(_ context.Context, req *connect.Request[operationsv1.CancelOperationRequest]) (*connect.Response[operationsv1.OperationStanding], error) {
	o.mu.Lock()
	o.Cancelled = append(o.Cancelled, req.Msg.GetOperationId())
	o.mu.Unlock()
	return connect.NewResponse(o.next()), nil
}

// ListDeploymentOperations implements the handler.
func (o *Operations) ListDeploymentOperations(_ context.Context, req *connect.Request[operationsv1.ListDeploymentOperationsRequest]) (*connect.Response[operationsv1.ListDeploymentOperationsResponse], error) {
	o.mu.Lock()
	o.Listed = append(o.Listed, req.Msg.GetDeploymentId())
	o.mu.Unlock()
	if o.ListAnswer != nil {
		return connect.NewResponse(o.ListAnswer), nil
	}
	return connect.NewResponse(&operationsv1.ListDeploymentOperationsResponse{SchemaVersion: "1", DeploymentId: req.Msg.GetDeploymentId(), Operations: o.Script}), nil
}

// Standing builds a standing for tests.
func Standing(id, deploymentID, state string, terminal bool) *operationsv1.OperationStanding {
	return &operationsv1.OperationStanding{
		SchemaVersion: "1", OperationId: id, DeploymentId: deploymentID, State: state, Terminal: terminal, Fence: 3,
		PlanDigest: "sha256:" + id, RequestKey: "key-" + id, ActiveStep: "release.activate", CompletedSteps: []string{"release.stage"},
		StepReceipts:    []*operationsv1.StepReceipt{{Step: "release.stage", Outcome: "succeeded", Fence: 3, Source: "worker", CompletedAt: "t"}},
		ReattachCommand: "scenario-to-cloud operation wait " + id,
		NextAction:      &errorsv1.NextAction{Kind: "wait", Reference: "/api/v1/operations/" + id + "/wait", Label: "Wait once"},
		CreatedAt:       "t0", UpdatedAt: "t1",
	}
}

// Server bundles the fakes on one httptest-ready mux. Extra REST routes are
// added with Mux.HandleFunc.
type Server struct {
	Mux         *http.ServeMux
	Deployments *Deployments
	Plans       *Plans
	Operations  *Operations
}

// NewServer mounts the three fakes and a healthy /health.
func NewServer() *Server {
	s := &Server{Mux: http.NewServeMux(), Deployments: &Deployments{}, Plans: &Plans{}, Operations: &Operations{}}
	path, handler := deploymentsv1connect.NewDeploymentsServiceHandler(s.Deployments)
	s.Mux.Handle(path, handler)
	path, handler = plansv1connect.NewPlansServiceHandler(s.Plans)
	s.Mux.Handle(path, handler)
	path, handler = operationsv1connect.NewOperationsServiceHandler(s.Operations)
	s.Mux.Handle(path, handler)
	s.Mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy","readiness":true}`))
	})
	return s
}
