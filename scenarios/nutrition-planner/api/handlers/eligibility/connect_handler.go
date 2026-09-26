package eligibility

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/identity"
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
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, principal.Subject); err != nil {
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
	decision := internal.Evaluate(internal.Profile{Revision: req.Msg.ProfileRevision, ExcludedGroups: req.Msg.ExcludedGroups, Allergies: req.Msg.Allergies, Appliances: req.Msg.Appliances}, internal.Candidate{Revision: req.Msg.RecipeRevision, Groups: req.Msg.RecipeGroups, RequiredAppliances: req.Msg.RequiredAppliances, AllergenEvidence: evidence, MethodIDs: req.Msg.MethodIds})
	reasons := make([]*v1.EligibilityReason, 0, len(decision.Reasons))
	for _, reason := range decision.Reasons {
		reasons = append(reasons, &v1.EligibilityReason{Code: reason.Code, Rule: reason.Rule, Reference: reason.Reference, Message: reason.Message, ViableMethodIds: reason.ViableMethodIDs})
	}
	return connect.NewResponse(&v1.EvaluateResponse{Status: string(decision.Status), Reasons: reasons, ProfileRevision: decision.ProfileRevision, RecipeRevision: decision.RecipeRevision, EvaluatorVersion: decision.EvaluatorVersion}), nil
}
