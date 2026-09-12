// Package plansvc is the Connect implementation of
// vrooli.scenario_to_cloud.v1.plans.PlansService. It mirrors the REST plan
// endpoints exactly: the same compile path, the same digest comparison and
// the same typed errors, carried as an errors.v1.Error detail.
package plansvc

import (
	"context"
	"encoding/json"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/deploymentsvc"
	"scenario-to-cloud/execplan"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans/plansv1connect"
)

// SchemaVersion is the response schema version clients negotiate on.
const SchemaVersion = execplan.SchemaVersion

// Compiled is the compile result the planner returns.
type Compiled struct {
	Plan          *execplan.Plan
	PlanDigest    string
	Preview       execplan.Preview
	ClosureStatus string
}

// Applied is the apply result the planner returns.
type Applied struct {
	OperationID string
	PlanDigest  string
	State       string
}

// Planner is the seam the REST server implements; the Connect service is a
// transport adapter over it so both surfaces share one compile path.
type Planner interface {
	CompilePlan(ctx context.Context, deploymentID, scope string, forceBundleBuild bool) (*Compiled, error)
	ApplyPlan(ctx context.Context, deploymentID, planDigest, requestKey, scope string, runPreflight bool) (*Applied, error)
}

// Service implements plansv1connect.PlansServiceHandler.
type Service struct {
	planner Planner
}

// New builds the service over a planner.
func New(planner Planner) *Service {
	return &Service{planner: planner}
}

// Handler returns the Connect mount path and handler.
func (s *Service) Handler(opts ...connect.HandlerOption) (string, http.Handler) {
	return plansv1connect.NewPlansServiceHandler(s, opts...)
}

// CompilePlan compiles and previews the plan for a deployment.
func (s *Service) CompilePlan(ctx context.Context, req *connect.Request[plansv1.CompilePlanRequest]) (*connect.Response[plansv1.CompilePlanResponse], error) {
	if req.Msg.GetDeploymentId() == "" {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "deployment_id is required"))
	}
	compiled, err := s.planner.CompilePlan(ctx, req.Msg.GetDeploymentId(), req.Msg.GetScope(), req.Msg.GetForceBundleBuild())
	if err != nil {
		return nil, deploymentsvc.ConnectError(err)
	}
	plan, err := PlanProto(compiled.Plan)
	if err != nil {
		return nil, deploymentsvc.ConnectError(apierrors.Internal("Failed to encode plan", err))
	}
	preview, err := PreviewProto(compiled.Preview)
	if err != nil {
		return nil, deploymentsvc.ConnectError(apierrors.Internal("Failed to encode preview", err))
	}
	return connect.NewResponse(&plansv1.CompilePlanResponse{
		SchemaVersion: SchemaVersion,
		Plan:          plan,
		PlanDigest:    compiled.PlanDigest,
		Preview:       preview,
		ClosureStatus: compiled.ClosureStatus,
	}), nil
}

// ApplyPlan admits the reviewed plan digest.
func (s *Service) ApplyPlan(ctx context.Context, req *connect.Request[plansv1.ApplyPlanRequest]) (*connect.Response[plansv1.ApplyPlanResponse], error) {
	if req.Msg.GetDeploymentId() == "" {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "deployment_id is required"))
	}
	applied, err := s.planner.ApplyPlan(ctx, req.Msg.GetDeploymentId(), req.Msg.GetPlanDigest(), req.Msg.GetRequestKey(), req.Msg.GetScope(), req.Msg.GetRunPreflight())
	if err != nil {
		return nil, deploymentsvc.ConnectError(err)
	}
	return connect.NewResponse(&plansv1.ApplyPlanResponse{
		SchemaVersion: SchemaVersion,
		OperationId:   applied.OperationID,
		PlanDigest:    applied.PlanDigest,
		State:         applied.State,
	}), nil
}

// PlanProto converts the domain plan to its wire message. The domain JSON
// uses the proto field names, so the conversion is a protojson round trip
// and cannot drift from the schema silently.
func PlanProto(plan *execplan.Plan) (*plansv1.ExecutablePlan, error) {
	raw, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	out := &plansv1.ExecutablePlan{}
	if err := protojson.Unmarshal(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// PlanFromProto converts a wire plan back to the domain plan.
func PlanFromProto(msg *plansv1.ExecutablePlan) (*execplan.Plan, error) {
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		return nil, err
	}
	// protojson encodes 64-bit integers as strings; the domain JSON carries
	// numbers. Normalise the two uint64 fields before decoding.
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	numberField(generic, "desired_revision")
	if target, ok := generic["target"].(map[string]any); ok {
		numberField(target, "enrollment_generation")
	}
	normalised, err := json.Marshal(generic)
	if err != nil {
		return nil, err
	}
	var plan execplan.Plan
	if err := json.Unmarshal(normalised, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func numberField(obj map[string]any, key string) {
	if s, ok := obj[key].(string); ok {
		obj[key] = json.Number(s)
	}
}

// PreviewProto converts the rendered preview to its wire message.
func PreviewProto(preview execplan.Preview) (*plansv1.Preview, error) {
	raw, err := json.Marshal(preview)
	if err != nil {
		return nil, err
	}
	out := &plansv1.Preview{}
	if err := protojson.Unmarshal(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}
