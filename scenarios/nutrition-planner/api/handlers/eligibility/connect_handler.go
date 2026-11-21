package eligibility

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/eligibility"

	internal "nutrition-planner/internal/eligibility"
	workspace "nutrition-planner/internal/workspace"
)

type connectHandler struct {
	workspaces workspace.Service
	logger     *log.Logger
}

func NewConnectHandler(workspaces workspace.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{workspaces: workspaces, logger: logger}
}

func (h *connectHandler) Evaluate(ctx context.Context, req *connect.Request[v1.EvaluateRequest]) (*connect.Response[v1.EvaluateResponse], error) {
	ownerSubject := workspace.LocalOwnerSubject
	if req.Msg.WorkspaceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, ownerSubject); err != nil {
		var nf workspace.ErrNotFound
		if errors.As(err, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		var forbidden workspace.ErrForbidden
		if errors.As(err, &forbidden) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	evidence := make(map[string]internal.EvidenceState, len(req.Msg.AllergenEvidence))
	for key, value := range req.Msg.AllergenEvidence {
		evidence[key] = internal.EvidenceState(value)
	}
	methods := make([]internal.Method, 0, len(req.Msg.Methods))
	for _, method := range req.Msg.Methods {
		if method == nil {
			continue
		}
		methods = append(methods, internal.Method{ID: method.Id, Name: method.Name, PreparationMode: method.PreparationMode, RequiredAppliances: method.RequiredAppliances, RequiredCookware: method.RequiredCookware, RequiredTools: method.RequiredTools, RequiredDeviceFunctions: method.RequiredDeviceFunctions})
	}
	decision := internal.Evaluate(internal.Profile{Revision: req.Msg.ProfileRevision, ExcludedGroups: req.Msg.ExcludedGroups, Allergies: req.Msg.Allergies, Appliances: req.Msg.Appliances, Cookware: req.Msg.Cookware, Tools: req.Msg.Tools, DeviceFunctions: req.Msg.DeviceFunctions}, internal.Candidate{Revision: req.Msg.RecipeRevision, Groups: req.Msg.RecipeGroups, AllergenEvidence: evidence, Methods: methods})
	reasons := make([]*v1.EligibilityReason, 0, len(decision.Reasons))
	for _, reason := range decision.Reasons {
		reasons = append(reasons, &v1.EligibilityReason{Code: reason.Code, Rule: reason.Rule, Reference: reason.Reference, Message: reason.Message, ViableMethodIds: reason.ViableMethodIDs, MethodId: reason.MethodID})
	}
	assessments := make([]*v1.EligibilityMethodAssessment, 0, len(decision.Methods))
	for _, method := range decision.Methods {
		methodReasons := make([]*v1.EligibilityReason, 0, len(method.Reasons))
		for _, reason := range method.Reasons {
			methodReasons = append(methodReasons, &v1.EligibilityReason{Code: reason.Code, Rule: reason.Rule, Reference: reason.Reference, Message: reason.Message, MethodId: reason.MethodID})
		}
		assessments = append(assessments, &v1.EligibilityMethodAssessment{MethodId: method.MethodID, MethodName: method.MethodName, Status: string(method.Status), Reasons: methodReasons})
	}
	return connect.NewResponse(&v1.EvaluateResponse{Status: string(decision.Status), Reasons: reasons, ProfileRevision: decision.ProfileRevision, RecipeRevision: decision.RecipeRevision, EvaluatorVersion: decision.EvaluatorVersion, MethodAssessments: assessments, ViableMethodIds: decision.ViableMethodIDs}), nil
}
