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
	if _, err := ws.Get(ctx, id, p.OwnerSubject()); err != nil {
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
	devices := []internal.KitchenDevice(nil)
	if q.Msg.ReplaceKitchenDevices {
		devices = make([]internal.KitchenDevice, 0, len(q.Msg.KitchenDevices))
		for _, device := range q.Msg.KitchenDevices {
			if device == nil || device.Id == "" || device.Type == "" || device.Count < 1 || device.Capacity == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("each kitchen device needs an identity, type, positive count, and capacity"))
			}
			devices = append(devices, internal.KitchenDevice{ID: device.Id, Type: device.Type, Count: device.Count, Capacity: device.Capacity, Capabilities: device.Capabilities})
		}
	}
	p, e := h.s.Apply(ctx, internal.ApplyInput{WorkspaceID: id, Preset: q.Msg.Preset, ExcludedGroups: q.Msg.ExcludedGroups, Allergies: q.Msg.Allergies, Appliances: q.Msg.Appliances, Cookware: q.Msg.Cookware, Tools: q.Msg.Tools, KitchenDevices: devices, CostWeight: q.Msg.CostWeight, EffortWeight: q.Msg.EffortWeight, VarietyWeight: q.Msg.VarietyWeight})
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
			decision := eligibility.Evaluate(eligibilityProfile(p), eligibility.Candidate{Revision: item.Revision, Groups: item.Groups, AllergenEvidence: mapEvidence(item.AllergenEvidence), Methods: eligibilityMethods(item.Methods)})
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

func eligibilityProfile(p internal.Profile) eligibility.Profile {
	functions := make([]string, 0)
	for _, device := range p.KitchenDevices {
		functions = append(functions, device.Capabilities...)
	}
	return eligibility.Profile{Revision: p.Revision, ExcludedGroups: append(append([]string(nil), p.ActiveRules...), p.ExcludedGroups...), Allergies: p.Allergies, Appliances: p.Appliances, Cookware: p.Cookware, Tools: p.Tools, DeviceFunctions: functions}
}

func eligibilityMethods(methods []recipe.Method) []eligibility.Method {
	out := make([]eligibility.Method, 0, len(methods))
	for _, method := range methods {
		out = append(out, eligibility.Method{ID: method.ID, Name: method.Name, PreparationMode: method.PreparationMode, RequiredAppliances: method.RequiredAppliances, RequiredCookware: method.RequiredCookware, RequiredTools: method.RequiredTools, RequiredDeviceFunctions: method.RequiredDeviceFunctions})
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
	devices := make([]*v1.KitchenDevice, 0, len(p.KitchenDevices))
	for _, device := range p.KitchenDevices {
		devices = append(devices, &v1.KitchenDevice{Id: device.ID, Type: device.Type, Count: device.Count, Capacity: device.Capacity, Capabilities: device.Capabilities})
	}
	return &v1.Profile{WorkspaceId: p.WorkspaceID, Revision: p.Revision, Preset: p.Preset, PresetVersion: p.PresetVersion, ActiveRules: p.ActiveRules, ExcludedGroups: p.ExcludedGroups, Allergies: p.Allergies, Appliances: p.Appliances, Cookware: p.Cookware, Tools: p.Tools, KitchenDevices: devices, CostWeight: p.CostWeight, EffortWeight: p.EffortWeight, VarietyWeight: p.VarietyWeight, DraftJson: p.DraftJSON}
}
