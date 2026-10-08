package planning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/planning"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/eligibility"
	feedback "nutrition-planner/internal/feedback"
	"nutrition-planner/internal/inventory"
	internal "nutrition-planner/internal/planning"
	profile "nutrition-planner/internal/profile"
	recipe "nutrition-planner/internal/recipe"
	shopping "nutrition-planner/internal/shopping"
	workspace "nutrition-planner/internal/workspace"
)

type connectHandler struct {
	workspaces       workspace.Service
	recipes          recipe.Service
	profiles         profile.Service
	plans            internal.Repository
	shopping         shopping.Repository
	shoppingEvidence shopping.Evidence
	feedback         feedback.Repository
	logger           *log.Logger
}

func NewConnectHandler(workspaces workspace.Service, recipes recipe.Service, profiles profile.Service, plans internal.Repository, shoppingRepo shopping.Repository, feedbackRepo feedback.Repository, logger *log.Logger, evidence ...shopping.Evidence) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	h := &connectHandler{workspaces: workspaces, recipes: recipes, profiles: profiles, plans: plans, shopping: shoppingRepo, feedback: feedbackRepo, logger: logger}
	if len(evidence) > 0 {
		h.shoppingEvidence = evidence[0]
	}
	return h
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
			active = eligibility.Profile{Revision: saved.Revision, ExcludedGroups: append(append([]string(nil), saved.ActiveRules...), saved.ExcludedGroups...), Allergies: saved.Allergies, Appliances: saved.Appliances}
			profileRevision = saved.Revision
		} else {
			var notFound profile.ErrNotFound
			if !errors.As(profileErr, &notFound) {
				return nil, connect.NewError(connect.CodeInternal, profileErr)
			}
		}
	}
	candidates := make([]internal.Candidate, 0, len(items))
	recipeRevisions := make(map[string]int64, len(items))
	for _, item := range items {
		recipeRevisions[item.ID] = item.Revision
		decision := eligibility.Evaluate(active, eligibility.Candidate{Revision: item.Revision, Groups: item.Groups, RequiredAppliances: item.RequiredAppliances, AllergenEvidence: mapEvidence(item.AllergenEvidence), MethodIDs: methodIDs(item.Methods)})
		candidates = append(candidates, internal.Candidate{ID: item.ID, Name: item.Name, Eligible: decision.Status == eligibility.Eligible, InputRevision: fmt.Sprint(item.Revision), Cost: 0, Effort: float64(len(item.Methods))})
	}
	currentRevision, currentPlan, revisionErr := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if revisionErr != nil {
		return nil, connect.NewError(connect.CodeInternal, revisionErr)
	}
	slots := make([]internal.Slot, 0, len(req.Msg.Dates)+len(req.Msg.MealSlots))
	if len(req.Msg.MealSlots) > 0 {
		for _, slot := range req.Msg.MealSlots {
			slots = append(slots, internal.Slot{Date: slot.Date, SlotName: slot.SlotName, Mode: slot.Mode, Quantity: slot.Quantity, Locked: slot.LockedRecipeId != "", RecipeID: slot.LockedRecipeId})
		}
	} else {
		lockedRecipeIDs := make(map[string]string, len(req.Msg.LockedRecipeIds))
		for date, recipeID := range req.Msg.LockedRecipeIds {
			lockedRecipeIDs[date] = recipeID
		}
		openDates := map[string]bool{}
		var saved internal.Draft
		if currentPlan != "" && json.Unmarshal([]byte(currentPlan), &saved) == nil {
			for _, occurrence := range saved.Occurrences {
				if occurrence.Mode == "open" && containsDate(req.Msg.Dates, occurrence.Date) {
					openDates[occurrence.Date] = true
				}
				if occurrence.Locked && occurrence.RecipeID != "" && containsDate(req.Msg.Dates, occurrence.Date) {
					lockedRecipeIDs[occurrence.Date] = occurrence.RecipeID
				}
			}
		}
		for _, date := range req.Msg.Dates {
			if openDates[date] {
				slots = append(slots, internal.Slot{Date: date, SlotName: "dinner", Mode: "open", Quantity: "1"})
				continue
			}
			recipeID := lockedRecipeIDs[date]
			slots = append(slots, internal.Slot{Date: date, SlotName: "dinner", Mode: "fixed", Locked: recipeID != "", RecipeID: recipeID})
		}
	}
	inputReferences := []string{"workspace:" + req.Msg.WorkspaceId, fmt.Sprintf("profile:%d", profileRevision)}
	for _, item := range items {
		inputReferences = append(inputReferences, fmt.Sprintf("recipe:%s:%d", item.ID, item.Revision))
	}
	draft := internal.Generate(internal.GenerateInput{Slots: slots, Candidates: candidates, Seed: req.Msg.Seed, CostWeight: req.Msg.CostWeight, EffortWeight: req.Msg.EffortWeight, RepetitionWeight: req.Msg.RepetitionWeight, InputReferences: inputReferences})
	for i := range draft.Occurrences {
		draft.Occurrences[i].RecipeRevision = recipeRevisions[draft.Occurrences[i].RecipeID]
	}
	raw, err := json.Marshal(draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	unresolved := make([]string, 0, len(draft.Unresolved))
	for _, item := range draft.Unresolved {
		unresolved = append(unresolved, item.Date)
	}
	return connect.NewResponse(&v1.GeneratePlanResponse{RunId: draft.RunID, DraftJson: string(raw), UnresolvedDates: unresolved, InputReferences: draft.InputReferences, CurrentRevision: currentRevision}), nil
}

// ExploreRecipes is the read-only discovery query. It shares the exact eligibility
// evaluator and saved workspace recipe source used by planning; it never invents
// catalog content or claims facts that have not been evaluated.
func (h *connectHandler) ExploreRecipes(ctx context.Context, req *connect.Request[v1.ExploreRecipesRequest]) (*connect.Response[v1.ExploreRecipesResponse], error) {
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
	items, err := h.recipes.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	profileState := eligibility.Profile{}
	profileConfigured := false
	profileRevision := int64(0)
	if h.profiles != nil {
		if saved, profileErr := h.profiles.Get(ctx, req.Msg.WorkspaceId); profileErr == nil {
			profileConfigured = true
			profileRevision = saved.Revision
			profileState = eligibility.Profile{Revision: saved.Revision, ExcludedGroups: append(append([]string(nil), saved.ActiveRules...), saved.ExcludedGroups...), Allergies: saved.Allergies, Appliances: saved.Appliances}
		} else {
			var notFound profile.ErrNotFound
			if !errors.As(profileErr, &notFound) {
				return nil, connect.NewError(connect.CodeInternal, profileErr)
			}
		}
	}
	planRevision, _, err := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &v1.ExploreRecipesResponse{PlanRevision: planRevision, ProfileRevision: profileRevision, ProfileConfigured: profileConfigured, Candidates: make([]*v1.ExploreRecipeCandidate, 0, len(items)), BlockingReasons: make([]*v1.ExploreFitReason, 0), SavedRecipeCount: int32(len(items))}
	blockingReasonSet := make(map[string]bool)
	for _, item := range items {
		candidate := eligibility.Candidate{Revision: item.Revision, Groups: item.Groups, RequiredAppliances: item.RequiredAppliances, AllergenEvidence: mapEvidence(item.AllergenEvidence), MethodIDs: methodIDs(item.Methods)}
		decision := eligibility.Evaluate(profileState, candidate)
		if decision.Status != eligibility.Eligible {
			for _, reason := range decision.Reasons {
				key := string(reason.Code) + "\x00" + reason.Rule + "\x00" + reason.Reference
				if blockingReasonSet[key] {
					continue
				}
				blockingReasonSet[key] = true
				response.BlockingReasons = append(response.BlockingReasons, &v1.ExploreFitReason{Code: string(reason.Code), Rule: reason.Rule, Reference: reason.Reference, Message: reason.Message})
			}
			continue
		}
		summary := strings.TrimSpace(item.Notes)
		if summary == "" {
			summary = strings.TrimSpace(item.OriginalText)
		}
		response.Candidates = append(response.Candidates, &v1.ExploreRecipeCandidate{RecipeId: item.ID, Name: item.Name, RecipeRevision: item.Revision, FitReasons: exploreFitReasons(profileState, candidate, profileConfigured), Summary: summary})
	}
	return connect.NewResponse(response), nil
}

func exploreFitReasons(profileState eligibility.Profile, candidate eligibility.Candidate, configured bool) []*v1.ExploreFitReason {
	if !configured {
		return []*v1.ExploreFitReason{{Code: "setup_unconfigured", Rule: "setup", Reference: "workspace.profile", Message: "No setup rules are saved yet; this result is checked only against declared recipe evidence."}}
	}
	var reasons []*v1.ExploreFitReason
	if len(profileState.ExcludedGroups) > 0 {
		reasons = append(reasons, &v1.ExploreFitReason{Code: "excluded_groups_clear", Rule: strings.Join(profileState.ExcludedGroups, ", "), Reference: "recipe.groups", Message: "No excluded food group is declared for this recipe."})
	}
	for _, allergen := range profileState.Allergies {
		if candidate.AllergenEvidence[allergen] == eligibility.DeclaredAbsent {
			reasons = append(reasons, &v1.ExploreFitReason{Code: "allergen_declared_absent", Rule: allergen, Reference: "recipe.allergens." + allergen, Message: "Recipe evidence declares this allergen absent within its stated scope."})
		}
	}
	for _, appliance := range candidate.RequiredAppliances {
		if containsString(profileState.Appliances, appliance) {
			reasons = append(reasons, &v1.ExploreFitReason{Code: "required_appliance_available", Rule: appliance, Reference: "recipe.method.appliances", Message: "Your saved kitchen includes a required appliance."})
		}
	}
	if len(reasons) == 0 {
		reasons = append(reasons, &v1.ExploreFitReason{Code: "no_configured_conflicts", Rule: "active setup", Reference: "eligibility-v1", Message: "No conflict was found between the configured food rules and this recipe's declared requirements."})
	}
	return reasons
}

func containsString(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func (h *connectHandler) GetPlan(ctx context.Context, req *connect.Request[v1.GetPlanRequest]) (*connect.Response[v1.GetPlanResponse], error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	from, fromErr := time.Parse("2006-01-02", req.Msg.FromDate)
	to, toErr := time.Parse("2006-01-02", req.Msg.ToDate)
	if fromErr != nil || toErr != nil || to.Before(from) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("from_date and to_date must be valid ordered YYYY-MM-DD dates"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, p.Subject); err != nil {
		return nil, planningWorkspaceError(err)
	}
	revision, raw, err := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	draft := internal.Draft{Occurrences: []internal.Occurrence{}, Unresolved: []internal.Unresolved{}}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &draft); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode saved plan: %w", err))
		}
	}
	draft.Occurrences = filterOccurrences(draft.Occurrences, req.Msg.FromDate, req.Msg.ToDate)
	draft.Unresolved = filterUnresolved(draft.Unresolved, req.Msg.FromDate, req.Msg.ToDate)
	encoded, err := json.Marshal(draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	hasPlan := len(draft.Occurrences) > 0 || len(draft.Unresolved) > 0
	return connect.NewResponse(&v1.GetPlanResponse{CurrentRevision: revision, DraftJson: string(encoded), HasPlan: hasPlan}), nil
}

func filterOccurrences(in []internal.Occurrence, from, to string) []internal.Occurrence {
	out := make([]internal.Occurrence, 0, len(in))
	for _, occurrence := range in {
		if occurrence.Date >= from && occurrence.Date <= to {
			out = append(out, occurrence)
		}
	}
	return out
}

func filterUnresolved(in []internal.Unresolved, from, to string) []internal.Unresolved {
	out := make([]internal.Unresolved, 0, len(in))
	for _, unresolved := range in {
		if unresolved.Date >= from && unresolved.Date <= to {
			out = append(out, unresolved)
		}
	}
	return out
}

func containsDate(dates []string, target string) bool {
	for _, date := range dates {
		if date == target {
			return true
		}
	}
	return false
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
	currentRevision, currentRaw, err := h.plans.Get(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if currentRevision != req.Msg.ExpectedRevision {
		return nil, connect.NewError(connect.CodeAborted, internal.ErrStaleInputs{WorkspaceID: req.Msg.WorkspaceId, Expected: req.Msg.ExpectedRevision, Actual: currentRevision})
	}
	var currentDraft, proposedDraft internal.Draft
	if currentRaw != "" {
		if err := json.Unmarshal([]byte(currentRaw), &currentDraft); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if err := json.Unmarshal([]byte(req.Msg.DraftJson), &proposedDraft); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := internal.EnsureLockedOccurrencesRetained(currentDraft, proposedDraft); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err := h.validateReviewedImpactInputs(ctx, req.Msg.WorkspaceId, currentDraft, proposedDraft); err != nil {
		var stale staleReviewInputError
		if errors.As(err, &stale) {
			return nil, connect.NewError(connect.CodeAborted, err)
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
	if req.Msg.WorkspaceId == "" || req.Msg.Date == "" || req.Msg.SlotName == "" || req.Msg.ReplacementRecipeId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id, date, slot_name, and replacement_recipe_id are required"))
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
	if err := h.validateDraftInputs(ctx, req.Msg.WorkspaceId, raw); err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	replacement, err := h.recipes.Get(ctx, req.Msg.ReplacementRecipeId, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.profiles != nil {
		if saved, profileErr := h.profiles.Get(ctx, req.Msg.WorkspaceId); profileErr == nil {
			decision := eligibility.Evaluate(eligibility.Profile{Revision: saved.Revision, ExcludedGroups: append(append([]string(nil), saved.ActiveRules...), saved.ExcludedGroups...), Allergies: saved.Allergies, Appliances: saved.Appliances}, eligibility.Candidate{Revision: replacement.Revision, Groups: replacement.Groups, RequiredAppliances: replacement.RequiredAppliances, AllergenEvidence: mapEvidence(replacement.AllergenEvidence), MethodIDs: methodIDs(replacement.Methods)})
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
	preview, err := internal.PreviewSwap(draft, internal.SwapRequest{Date: req.Msg.Date, SlotName: req.Msg.SlotName, ReplacementID: replacement.ID, ReplacementRevision: replacement.Revision, ReplacementName: replacement.Name, ReplaceMatchingFuture: req.Msg.ReplaceMatchingFuture})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := addPreparedBatchImpacts(ctx, h.shoppingEvidence.Inventory, req.Msg.WorkspaceId, &preview); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	appendPlanInputReference(&preview.Draft, fmt.Sprintf("recipe:%s:%d", replacement.ID, replacement.Revision))
	beforeLines, err := h.shoppingLines(ctx, req.Msg.WorkspaceId, draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	afterLines, err := h.shoppingLines(ctx, req.Msg.WorkspaceId, preview.Draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if h.shoppingEvidence.Inventory != nil && h.shoppingEvidence.Costs != nil && h.shoppingEvidence.Now != nil {
		reference, refErr := shoppingImpactReference(afterLines)
		if refErr != nil {
			return nil, connect.NewError(connect.CodeInternal, refErr)
		}
		appendPlanInputReference(&preview.Draft, reference)
	}
	if reference, refErr := batchImpactReference(ctx, h.shoppingEvidence.Inventory, req.Msg.WorkspaceId, draft, preview.Draft); refErr != nil {
		return nil, connect.NewError(connect.CodeInternal, refErr)
	} else if reference != "" {
		appendPlanInputReference(&preview.Draft, reference)
	}
	shoppingChanges := shopping.Diff(beforeLines, afterLines)
	encoded, err := json.Marshal(struct {
		internal.SwapPreview
		ShoppingChanges []shopping.Change `json:"shoppingChanges"`
	}{SwapPreview: preview, ShoppingChanges: shoppingChanges})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	affected := make([]string, 0, len(preview.Changes))
	for _, change := range preview.Changes {
		affected = append(affected, change.Date)
	}
	return connect.NewResponse(&v1.PreviewSwapResponse{Revision: current, PreviewJson: string(encoded), AffectedDates: affected}), nil
}

func addPreparedBatchImpacts(ctx context.Context, stock inventory.Repository, workspaceID string, preview *internal.SwapPreview) error {
	sources := make(map[recipeVersion]string)
	for _, change := range preview.Changes {
		sources[recipeVersion{id: change.BeforeID, revision: change.BeforeRevision}] = change.BeforeName
	}
	impacts, err := preparedBatchImpacts(ctx, stock, workspaceID, sources)
	if err != nil {
		return err
	}
	preview.PreparedBatchImpacts = impacts
	return nil
}

type recipeVersion struct {
	id       string
	revision int64
}

type staleReviewInputError struct {
	kind, expected, actual string
}

func (e staleReviewInputError) Error() string {
	return fmt.Sprintf("%s impact input changed: expected %s, actual %s; refresh the review", e.kind, e.expected, e.actual)
}

func (h *connectHandler) shoppingLines(ctx context.Context, workspaceID string, draft internal.Draft) ([]shopping.Line, error) {
	checked := map[string]bool{}
	if h.shopping != nil {
		var err error
		checked, err = h.shopping.Checked(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
	}
	recipes, err := shopping.LoadSelectedRecipes(ctx, h.recipes, workspaceID, draft)
	if err != nil {
		return nil, err
	}
	lines := shopping.Derive(draft, recipes, checked)
	if h.shoppingEvidence.Inventory != nil && h.shoppingEvidence.Costs != nil && h.shoppingEvidence.Now != nil {
		lines, err = shopping.Enrich(ctx, workspaceID, lines, h.shoppingEvidence.Inventory, h.shoppingEvidence.Costs, h.shoppingEvidence.Now())
		if err != nil {
			return nil, err
		}
	}
	return lines, nil
}

func shoppingImpactReference(lines []shopping.Line) (string, error) {
	snapshot := append([]shopping.Line(nil), lines...)
	for index := range snapshot {
		snapshot[index].Checked = false
	}
	return jsonImpactReference("shopping-impact", snapshot)
}

func (h *connectHandler) validateReviewedImpactInputs(ctx context.Context, workspaceID string, before, proposed internal.Draft) error {
	for _, reference := range proposed.InputReferences {
		switch {
		case strings.HasPrefix(reference, "shopping-impact:"):
			lines, err := h.shoppingLines(ctx, workspaceID, proposed)
			if err != nil {
				return err
			}
			actual, err := shoppingImpactReference(lines)
			if err != nil {
				return err
			}
			if actual != reference {
				return staleReviewInputError{kind: "shopping", expected: reference, actual: actual}
			}
		case strings.HasPrefix(reference, "batch-impact:"):
			actual, err := batchImpactReference(ctx, h.shoppingEvidence.Inventory, workspaceID, before, proposed)
			if err != nil {
				return err
			}
			if actual != reference {
				return staleReviewInputError{kind: "prepared batch", expected: reference, actual: actual}
			}
		}
	}
	return nil
}

func batchImpactReference(ctx context.Context, stock inventory.Repository, workspaceID string, before, after internal.Draft) (string, error) {
	if _, ok := stock.(inventory.BatchRepository); !ok {
		return "", nil
	}
	bySlot := make(map[string]internal.Occurrence, len(after.Occurrences))
	for _, occurrence := range after.Occurrences {
		bySlot[occurrence.Date+"\x00"+occurrence.SlotName] = occurrence
	}
	sources := make(map[recipeVersion]string)
	for _, original := range before.Occurrences {
		current, exists := bySlot[original.Date+"\x00"+original.SlotName]
		if original.RecipeID == "" || !exists || current.RecipeID != original.RecipeID || current.RecipeRevision != original.RecipeRevision || current.Quantity != original.Quantity {
			sources[recipeVersion{id: original.RecipeID, revision: original.RecipeRevision}] = original.RecipeName
		}
	}
	impacts, err := preparedBatchImpacts(ctx, stock, workspaceID, sources)
	if err != nil {
		return "", err
	}
	return jsonImpactReference("batch-impact", impacts)
}

func preparedBatchImpacts(ctx context.Context, stock inventory.Repository, workspaceID string, sources map[recipeVersion]string) ([]internal.PreparedBatchImpact, error) {
	batchRepo, ok := stock.(inventory.BatchRepository)
	if !ok || len(sources) == 0 {
		return []internal.PreparedBatchImpact{}, nil
	}
	batches, err := batchRepo.ListBatches(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	impacts := make([]internal.PreparedBatchImpact, 0)
	for _, batch := range batches {
		name, relevant := sources[recipeVersion{id: batch.RecipeID, revision: batch.RecipeRevision}]
		if !relevant {
			continue
		}
		if !batch.Available.IsUnknown() {
			if comparison, compareErr := decimalx.Compare(batch.Available, decimalx.KnownInt(0)); compareErr != nil || comparison <= 0 {
				continue
			}
		}
		impacts = append(impacts, internal.PreparedBatchImpact{BatchID: batch.ID, RecipeID: batch.RecipeID, RecipeName: name, RecipeRevision: batch.RecipeRevision, Available: batch.Available.String(), Unit: batch.Unit})
	}
	for i := 0; i < len(impacts); i++ {
		for j := i + 1; j < len(impacts); j++ {
			if impacts[j].RecipeID < impacts[i].RecipeID || (impacts[j].RecipeID == impacts[i].RecipeID && (impacts[j].RecipeRevision < impacts[i].RecipeRevision || (impacts[j].RecipeRevision == impacts[i].RecipeRevision && impacts[j].BatchID < impacts[i].BatchID))) {
				impacts[i], impacts[j] = impacts[j], impacts[i]
			}
		}
	}
	return impacts, nil
}

func jsonImpactReference(kind string, value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return kind + ":" + hex.EncodeToString(digest[:]), nil
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
	recipes, err := shopping.LoadSelectedRecipes(ctx, h.recipes, req.Msg.WorkspaceId, draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	checked := map[string]bool{}
	haveThis := map[string]bool{}
	actuals := map[string]shopping.PurchaseLine{}
	if h.shopping != nil {
		checked, err = h.shopping.Checked(ctx, req.Msg.WorkspaceId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		haveThis, err = h.shopping.HaveThis(ctx, req.Msg.WorkspaceId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		actuals, err = h.shopping.LatestPurchaseReview(ctx, req.Msg.WorkspaceId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	lines := shopping.Derive(draft, recipes, checked)
	for i := range lines {
		lines[i].HaveThis = haveThis[lines[i].Key]
		if actual, ok := actuals[lines[i].Key]; ok {
			if !actual.Omitted {
				lines[i].ActualQuantity = actual.Amount.String()
			}
			lines[i].ActualUnit, lines[i].ActualPrice, lines[i].PurchaseOmitted = actual.Unit, actual.Price, actual.Omitted
		}
	}
	if h.shoppingEvidence.Inventory != nil && h.shoppingEvidence.Costs != nil && h.shoppingEvidence.Now != nil {
		lines, err = shopping.Enrich(ctx, req.Msg.WorkspaceId, lines, h.shoppingEvidence.Inventory, h.shoppingEvidence.Costs, h.shoppingEvidence.Now())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	out := &v1.GetShoppingPreviewResponse{Revision: revision, Lines: make([]*v1.ShoppingLine, 0, len(lines))}
	for _, line := range lines {
		out.Lines = append(out.Lines, &v1.ShoppingLine{Key: line.Key, Label: line.Label, Need: line.Need, Stock: line.Stock, Missing: line.Missing, PackageCount: line.PackageCount, Price: line.Price, PortionCost: line.PortionCost, CheckoutTotal: line.CheckoutTotal, ActualSpend: line.ActualSpend, SourceRecipeIds: line.SourceRecipes, Checked: line.Checked, HaveThis: line.HaveThis, ActualQuantity: line.ActualQuantity, ActualUnit: line.ActualUnit, ActualPrice: line.ActualPrice, PurchaseOmitted: line.PurchaseOmitted})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SetShoppingHaveThis(ctx context.Context, req *connect.Request[v1.SetShoppingHaveThisRequest]) (*connect.Response[v1.SetShoppingHaveThisResponse], error) {
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
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("shopping storage is unavailable"))
	}
	if err := h.shopping.SetHaveThis(ctx, req.Msg.WorkspaceId, req.Msg.LineKey, req.Msg.HaveThis); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.SetShoppingHaveThisResponse{HaveThis: req.Msg.HaveThis}), nil
}

func (h *connectHandler) ConfirmShoppingPurchases(ctx context.Context, req *connect.Request[v1.ConfirmShoppingPurchasesRequest]) (*connect.Response[v1.ConfirmShoppingPurchasesResponse], error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if req.Msg.WorkspaceId == "" || req.Msg.ReviewId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id and review_id are required"))
	}
	if _, err := h.workspaces.Get(ctx, req.Msg.WorkspaceId, p.Subject); err != nil {
		return nil, planningWorkspaceError(err)
	}
	if h.shopping == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("shopping storage is unavailable"))
	}
	lines := make([]shopping.PurchaseLine, 0, len(req.Msg.Lines))
	for _, actual := range req.Msg.Lines {
		if actual == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("purchase rows cannot be empty"))
		}
		amount := decimalx.Unknown
		if !actual.Omitted {
			var err error
			amount, err = decimalx.Parse(actual.Amount)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("purchase amount for %q: %w", actual.LineKey, err))
			}
		}
		lines = append(lines, shopping.PurchaseLine{Key: actual.LineKey, ItemID: actual.ItemId, Amount: amount, Unit: actual.Unit, Price: actual.Price, Omitted: actual.Omitted})
	}
	if err := h.shopping.ConfirmPurchases(ctx, req.Msg.WorkspaceId, req.Msg.ReviewId, lines); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.ConfirmShoppingPurchasesResponse{Confirmed: true}), nil
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
	for _, occurrence := range draft.Occurrences {
		if occurrence.RecipeID == "" || occurrence.RecipeRevision == 0 {
			continue // legacy drafts without a pinned occurrence revision remain readable
		}
		current, ok := actual[occurrence.RecipeID]
		if !ok || current != occurrence.RecipeRevision {
			return fmt.Errorf("recipe occurrence input changed: %s expected revision %d, actual %d", occurrence.RecipeID, occurrence.RecipeRevision, current)
		}
	}
	return nil
}

func appendPlanInputReference(draft *internal.Draft, reference string) {
	for _, existing := range draft.InputReferences {
		if existing == reference {
			return
		}
	}
	draft.InputReferences = append(draft.InputReferences, reference)
}
