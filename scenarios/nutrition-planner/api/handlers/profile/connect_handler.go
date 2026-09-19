package profile

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/profile"
	"nutrition-planner/internal/eligibility"
	internal "nutrition-planner/internal/profile"
	recipe "nutrition-planner/internal/recipe"
	workspace "nutrition-planner/internal/workspace"
)

type connectHandler struct {
	s       internal.Service
	ws      workspace.Service
	recipes recipe.Service
	logger  *log.Logger
}

func NewConnectHandler(s internal.Service, ws workspace.Service, recipes recipe.Service, l *log.Logger) *connectHandler {
	if l == nil {
		l = log.Default()
	}
	return &connectHandler{s: s, ws: ws, recipes: recipes, logger: l}
}

func workspaceID(ctx context.Context, id string, ws workspace.Service) (string, error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if id == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := ws.Get(ctx, id, p.Subject); err != nil {
		var nf workspace.ErrNotFound
		if errors.As(err, &nf) {
			return "", connect.NewError(connect.CodeNotFound, err)
		}
		var forbidden workspace.ErrForbidden
		if errors.As(err, &forbidden) {
			return "", connect.NewError(connect.CodePermissionDenied, err)
		}
		return "", connect.NewError(connect.CodeInternal, err)
	}
	return id, nil
}

func (h *connectHandler) GetProfile(ctx context.Context, q *connect.Request[v1.GetProfileRequest]) (*connect.Response[v1.GetProfileResponse], error) {
	id, e := workspaceID(ctx, q.Msg.WorkspaceId, h.ws)
	if e != nil {
		return nil, e
	}
	p, e := h.s.Get(ctx, id)
	if e != nil {
		var nf internal.ErrNotFound
		if errors.As(e, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, e)
		}
		return nil, connect.NewError(connect.CodeInternal, e)
	}
	return connect.NewResponse(&v1.GetProfileResponse{Profile: toProto(p)}), nil
}

func (h *connectHandler) SaveProfileDraft(ctx context.Context, q *connect.Request[v1.SaveProfileDraftRequest]) (*connect.Response[v1.SaveProfileDraftResponse], error) {
	id, e := workspaceID(ctx, q.Msg.WorkspaceId, h.ws)
	if e != nil {
		return nil, e
	}
	p, e := h.s.SaveDraft(ctx, id, q.Msg.DraftJson)
	if e != nil {
		return nil, connect.NewError(connect.CodeInternal, e)
	}
	return connect.NewResponse(&v1.SaveProfileDraftResponse{Profile: toProto(p)}), nil
}

func (h *connectHandler) ApplyProfile(ctx context.Context, q *connect.Request[v1.ApplyProfileRequest]) (*connect.Response[v1.ApplyProfileResponse], error) {
	id, e := workspaceID(ctx, q.Msg.WorkspaceId, h.ws)
	if e != nil {
		return nil, e
	}
	p, e := h.s.Apply(ctx, internal.ApplyInput{WorkspaceID: id, Preset: q.Msg.Preset, ExcludedGroups: q.Msg.ExcludedGroups, Allergies: q.Msg.Allergies, Appliances: q.Msg.Appliances, CostWeight: q.Msg.CostWeight, EffortWeight: q.Msg.EffortWeight, VarietyWeight: q.Msg.VarietyWeight})
	if e != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, e)
	}
	matching := int64(0)
	needsReview := int64(0)
	excluded := int64(0)
	if h.recipes != nil {
		items, listErr := h.recipes.List(ctx, id)
		if listErr != nil {
			return nil, connect.NewError(connect.CodeInternal, listErr)
		}
		for _, item := range items {
			decision := eligibility.Evaluate(eligibility.Profile{Revision: p.Revision, ExcludedGroups: append(append([]string(nil), p.ActiveRules...), p.ExcludedGroups...), Allergies: p.Allergies, Appliances: p.Appliances}, eligibility.Candidate{Revision: item.Revision, Groups: item.Groups, RequiredAppliances: item.RequiredAppliances, AllergenEvidence: mapEvidence(item.AllergenEvidence), MethodIDs: methodIDs(item.Methods)})
			switch decision.Status {
			case eligibility.Eligible:
				matching++
			case eligibility.NeedsInformation:
				needsReview++
			case eligibility.Ineligible:
				excluded++
			}
		}
	}
	return connect.NewResponse(&v1.ApplyProfileResponse{Profile: toProto(p), MatchingMeals: matching, NeedsReviewMeals: needsReview, ExcludedMeals: excluded}), nil
}

func methodIDs(methods []recipe.Method) []string {
	out := make([]string, 0, len(methods))
	for _, method := range methods {
		out = append(out, method.ID)
	}
	return out
}

func mapEvidence(in map[string]string) map[string]eligibility.EvidenceState {
	out := make(map[string]eligibility.EvidenceState, len(in))
	for key, value := range in {
		out[key] = eligibility.EvidenceState(value)
	}
	return out
}

func toProto(p internal.Profile) *v1.Profile {
	return &v1.Profile{WorkspaceId: p.WorkspaceID, Revision: p.Revision, Preset: p.Preset, PresetVersion: p.PresetVersion, ActiveRules: p.ActiveRules, ExcludedGroups: p.ExcludedGroups, Allergies: p.Allergies, Appliances: p.Appliances, CostWeight: p.CostWeight, EffortWeight: p.EffortWeight, VarietyWeight: p.VarietyWeight, DraftJson: p.DraftJSON}
}
