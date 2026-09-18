package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/planning"
	"nutrition-planner/internal/eligibility"
	feedback "nutrition-planner/internal/feedback"
	internal "nutrition-planner/internal/planning"
	profile "nutrition-planner/internal/profile"
	recipe "nutrition-planner/internal/recipe"
	shopping "nutrition-planner/internal/shopping"
	workspace "nutrition-planner/internal/workspace"
)

type connectHandler struct {
	workspaces workspace.Service
	recipes    recipe.Service
	profiles   profile.Service
	plans      internal.Repository
	shopping   shopping.Repository
	feedback   feedback.Repository
	logger     *log.Logger
}

func NewConnectHandler(workspaces workspace.Service, recipes recipe.Service, profiles profile.Service, plans internal.Repository, shoppingRepo shopping.Repository, feedbackRepo feedback.Repository, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{workspaces: workspaces, recipes: recipes, profiles: profiles, plans: plans, shopping: shoppingRepo, feedback: feedbackRepo, logger: logger}
}
func (h *connectHandler) GeneratePlan(ctx context.Context, req *connect.Request[v1.GeneratePlanRequest]) (*connect.Response[v1.GeneratePlanResponse], error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, p.Subject); err != nil {
		var nf workspace.ErrNotFound
		if errors.As(err, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		var f workspace.ErrForbidden
		if errors.As(err, &f) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items, err := h.recipes.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	active := eligibility.Profile{}
	profileRevision := int64(0)
	if h.profiles != nil {
		if saved, profileErr := h.profiles.Get(ctx, req.Msg.WorkspaceId); profileErr == nil {
			active = eligibility.Profile{Revision: saved.Revision, ExcludedGroups: saved.ActiveRules, Allergies: saved.Allergies, Appliances: saved.Appliances}
			profileRevision = saved.Revision
		} else {
			var notFound profile.ErrNotFound
			if !errors.As(profileErr, &notFound) {
				return nil, connect.NewError(connect.CodeInternal, profileErr)
			}
		}
	}
	candidates := make([]internal.Candidate, 0, len(items))
	for _, item := range items {
		decision := eligibility.Evaluate(active, eligibility.Candidate{Revision: item.Revision, Groups: item.Groups, RequiredAppliances: item.RequiredAppliances, AllergenEvidence: mapEvidence(item.AllergenEvidence), MethodIDs: methodIDs(item.Methods)})
		candidates = append(candidates, internal.Candidate{ID: item.ID, Name: item.Name, Eligible: decision.Status == eligibility.Eligible, InputRevision: fmt.Sprint(item.Revision), Cost: 0, Effort: float64(len(item.Methods))})
	}
	slots := make([]internal.Slot, 0, len(req.Msg.Dates)+len(req.Msg.MealSlots))
	if len(req.Msg.MealSlots) > 0 {
		for _, slot := range req.Msg.MealSlots {
			slots = append(slots, internal.Slot{Date: slot.Date, SlotName: slot.SlotName, Mode: slot.Mode, Quantity: slot.Quantity, Locked: slot.LockedRecipeId != "", RecipeID: slot.LockedRecipeId})
		}
	} else {
		for _, date := range req.Msg.Dates {
			recipeID := req.Msg.LockedRecipeIds[date]
			slots = append(slots, internal.Slot{Date: date, SlotName: "dinner", Mode: "fixed", Locked: recipeID != "", RecipeID: recipeID})
		}
	}
	inputReferences := []string{"workspace:" + req.Msg.WorkspaceId, fmt.Sprintf("profile:%d", profileRevision)}
	for _, item := range items {
		inputReferences = append(inputReferences, fmt.Sprintf("recipe:%s:%d", item.ID, item.Revision))
	}
	draft := internal.Generate(internal.GenerateInput{Slots: slots, Candidates: candidates, Seed: req.Msg.Seed, CostWeight: req.Msg.CostWeight, EffortWeight: req.Msg.EffortWeight, RepetitionWeight: req.Msg.RepetitionWeight, InputReferences: inputReferences})
	raw, err := json.Marshal(draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	unresolved := make([]string, 0, len(draft.Unresolved))
	for _, item := range draft.Unresolved {
		unresolved = append(unresolved, item.Date)
	}
	currentRevision, _, revisionErr := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if revisionErr != nil {
		return nil, connect.NewError(connect.CodeInternal, revisionErr)
	}
	return connect.NewResponse(&v1.GeneratePlanResponse{RunId: draft.RunID, DraftJson: string(raw), UnresolvedDates: unresolved, InputReferences: draft.InputReferences, CurrentRevision: currentRevision}), nil
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

func (h *connectHandler) ApplyPlan(ctx context.Context, req *connect.Request[v1.ApplyPlanRequest]) (*connect.Response[v1.ApplyPlanResponse], error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.DraftJson == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id and draft_json are required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, p.Subject); err != nil {
		var nf workspace.ErrNotFound
		if errors.As(err, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		var f workspace.ErrForbidden
		if errors.As(err, &f) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.validateDraftInputs(ctx, req.Msg.WorkspaceId, req.Msg.DraftJson); err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	revision, err := h.plans.Apply(ctx, req.Msg.WorkspaceId, req.Msg.ExpectedRevision, req.Msg.DraftJson)
	if err != nil {
		var stale internal.ErrStaleInputs
		if errors.As(err, &stale) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ApplyPlanResponse{Revision: revision, PlanJson: req.Msg.DraftJson}), nil
}

func (h *connectHandler) PreviewSwap(ctx context.Context, req *connect.Request[v1.PreviewSwapRequest]) (*connect.Response[v1.PreviewSwapResponse], error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.Date == "" || req.Msg.ReplacementRecipeId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id, date, and replacement_recipe_id are required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, p.Subject); err != nil {
		return nil, planningWorkspaceError(err)
	}
	current, raw, err := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if current != req.Msg.ExpectedRevision {
		return nil, connect.NewError(connect.CodeAborted, internal.ErrStaleInputs{WorkspaceID: req.Msg.WorkspaceId, Expected: req.Msg.ExpectedRevision, Actual: current})
	}
	var draft internal.Draft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	replacement, err := h.recipes.Get(ctx, req.Msg.ReplacementRecipeId, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.profiles != nil {
		if saved, profileErr := h.profiles.Get(ctx, req.Msg.WorkspaceId); profileErr == nil {
			decision := eligibility.Evaluate(eligibility.Profile{Revision: saved.Revision, ExcludedGroups: saved.ActiveRules, Allergies: saved.Allergies, Appliances: saved.Appliances}, eligibility.Candidate{Revision: replacement.Revision, Groups: replacement.Groups, RequiredAppliances: replacement.RequiredAppliances, AllergenEvidence: mapEvidence(replacement.AllergenEvidence), MethodIDs: methodIDs(replacement.Methods)})
			if decision.Status != eligibility.Eligible {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("replacement recipe does not satisfy active rules"))
			}
		} else {
			var nf profile.ErrNotFound
			if !errors.As(profileErr, &nf) {
				return nil, connect.NewError(connect.CodeInternal, profileErr)
			}
		}
	}
	preview, err := internal.PreviewSwap(draft, internal.SwapRequest{Date: req.Msg.Date, ReplacementID: replacement.ID, ReplacementName: replacement.Name, ReplaceMatchingFuture: req.Msg.ReplaceMatchingFuture})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	encoded, err := json.Marshal(preview)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	affected := make([]string, 0, len(preview.Changes))
	for _, change := range preview.Changes {
		affected = append(affected, change.Date)
	}
	return connect.NewResponse(&v1.PreviewSwapResponse{Revision: current, PreviewJson: string(encoded), AffectedDates: affected}), nil
}

func planningWorkspaceError(err error) error {
	var nf workspace.ErrNotFound
	if errors.As(err, &nf) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var f workspace.ErrForbidden
	if errors.As(err, &f) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (h *connectHandler) GetShoppingPreview(ctx context.Context, req *connect.Request[v1.GetShoppingPreviewRequest]) (*connect.Response[v1.GetShoppingPreviewResponse], error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, p.Subject); err != nil {
		return nil, planningWorkspaceError(err)
	}
	revision, raw, err := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if req.Msg.ExpectedRevision >= 0 && revision != req.Msg.ExpectedRevision {
		return nil, connect.NewError(connect.CodeAborted, internal.ErrStaleInputs{WorkspaceID: req.Msg.WorkspaceId, Expected: req.Msg.ExpectedRevision, Actual: revision})
	}
	var draft internal.Draft
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &draft); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	recipes, err := h.recipes.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	checked := map[string]bool{}
	if h.shopping != nil {
		checked, err = h.shopping.Checked(ctx, req.Msg.WorkspaceId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	lines := shopping.Derive(draft, recipes, checked)
	out := &v1.GetShoppingPreviewResponse{Revision: revision, Lines: make([]*v1.ShoppingLine, 0, len(lines))}
	for _, line := range lines {
		out.Lines = append(out.Lines, &v1.ShoppingLine{Key: line.Key, Label: line.Label, Need: line.Need, Stock: line.Stock, Missing: line.Missing, PackageCount: line.PackageCount, Price: line.Price, SourceRecipeIds: line.SourceRecipes, Checked: line.Checked})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SetShoppingChecked(ctx context.Context, req *connect.Request[v1.SetShoppingCheckedRequest]) (*connect.Response[v1.SetShoppingCheckedResponse], error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.LineKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id and line_key are required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, p.Subject); err != nil {
		return nil, planningWorkspaceError(err)
	}
	if h.shopping == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("shopping checklist is unavailable"))
	}
	if err := h.shopping.SetChecked(ctx, req.Msg.WorkspaceId, req.Msg.LineKey, req.Msg.Checked); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.SetShoppingCheckedResponse{Checked: req.Msg.Checked}), nil
}

func (h *connectHandler) RecordFeedback(ctx context.Context, req *connect.Request[v1.RecordFeedbackRequest]) (*connect.Response[v1.RecordFeedbackResponse], error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.Date == "" || req.Msg.RecipeId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id, date, and recipe_id are required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, p.Subject); err != nil {
		return nil, planningWorkspaceError(err)
	}
	revision, raw, err := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if revision != req.Msg.ExpectedRevision {
		return nil, connect.NewError(connect.CodeAborted, internal.ErrStaleInputs{WorkspaceID: req.Msg.WorkspaceId, Expected: req.Msg.ExpectedRevision, Actual: revision})
	}
	var draft internal.Draft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	found := false
	for _, occurrence := range draft.Occurrences {
		if occurrence.Date == req.Msg.Date && occurrence.RecipeID == req.Msg.RecipeId {
			found = true
			break
		}
	}
	if !found {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("feedback must reference the planned recipe for that date"))
	}
	if h.feedback == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("feedback is unavailable"))
	}
	saved, err := h.feedback.Record(ctx, feedback.Feedback{WorkspaceID: req.Msg.WorkspaceId, Date: req.Msg.Date, RecipeID: req.Msg.RecipeId, Portion: req.Msg.Portion, Minutes: int(req.Msg.Minutes)})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.RecordFeedbackResponse{Date: saved.Date, RecipeId: saved.RecipeID, Portion: saved.Portion, Minutes: int32(saved.Minutes)}), nil
}

func (h *connectHandler) UndoFeedback(ctx context.Context, req *connect.Request[v1.UndoFeedbackRequest]) (*connect.Response[v1.UndoFeedbackResponse], error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.Date == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id and date are required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, p.Subject); err != nil {
		return nil, planningWorkspaceError(err)
	}
	if h.feedback == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("feedback is unavailable"))
	}
	if err := h.feedback.Undo(ctx, req.Msg.WorkspaceId, req.Msg.Date); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&v1.UndoFeedbackResponse{Undone: true}), nil
}

func (h *connectHandler) validateDraftInputs(ctx context.Context, workspaceID, raw string) error {
	var draft internal.Draft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return fmt.Errorf("draft_json: invalid plan draft: %w", err)
	}
	var saved profile.Profile
	var err error
	if h.profiles != nil {
		saved, err = h.profiles.Get(ctx, workspaceID)
	}
	if h.profiles != nil && err == nil {
		for _, ref := range draft.InputReferences {
			if strings.HasPrefix(ref, "profile:") {
				expected, parseErr := strconv.ParseInt(strings.TrimPrefix(ref, "profile:"), 10, 64)
				if parseErr == nil && saved.Revision != expected {
					return fmt.Errorf("profile input changed: expected revision %d, actual %d", expected, saved.Revision)
				}
			}
		}
	} else if h.profiles != nil {
		var nf profile.ErrNotFound
		if !errors.As(err, &nf) {
			return err
		}
	}
	items, err := h.recipes.List(ctx, workspaceID)
	if err != nil {
		return err
	}
	actual := make(map[string]int64, len(items))
	for _, item := range items {
		actual[item.ID] = item.Revision
	}
	for _, ref := range draft.InputReferences {
		parts := strings.Split(ref, ":")
		if len(parts) != 3 || parts[0] != "recipe" {
			continue
		}
		expected, parseErr := strconv.ParseInt(parts[2], 10, 64)
		if parseErr != nil {
			continue
		}
		current, ok := actual[parts[1]]
		if !ok || current != expected {
			return fmt.Errorf("recipe input changed: %s expected revision %d, actual %d", parts[1], expected, current)
		}
	}
	return nil
}
