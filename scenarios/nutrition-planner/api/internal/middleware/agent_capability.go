package middleware

import (
	"net/http"
	"strings"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// AgentCapability preserves the delegated scope ceiling without requiring
// Nutrition-specific human consent. Workspace handlers still bind every
// private object to the verified owner account.
func AgentCapability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := identity.PrincipalFromContext(r.Context())
		if !ok || p.Kind != identity.ActorAgent {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/health" || r.URL.Path == "/api/v1/capabilities" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/diagnostics" {
			if _, err := authn.RequireCapability(r.Context(), "nutrition-planner:read"); err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		_, method, found := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
		effect, classified := nutritionMethodEffect(method)
		if !found || !strings.HasPrefix(r.URL.Path, "/vrooli.nutrition_planner.v1.") || !classified || p.OwnerSubject() == "" {
			http.Error(w, "verified owner and classified Nutrition operation required", http.StatusForbidden)
			return
		}
		if _, err := authn.RequireCapability(r.Context(), "nutrition-planner:"+effect); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Unknown operations are refused until their owner classifies the effect.
func nutritionMethodEffect(method string) (string, bool) {
	switch method {
	case "Evaluate", "EvaluateScope", "ExploreRecipes", "ExportGroceriesCSV", "ExportRecipePDF", "ExportRecipes", "ExportWeeklyPDF", "ExportWorkspace", "GetPlan", "GetProfile", "GetRecipe", "GetRecipeRevision", "GetRestoreCheckpoint", "GetRevision", "GetSchedule", "GetSession", "GetShoppingPreview", "GetTarget", "GetWorkspace", "ListBatches", "ListCatalog", "ListEvents", "ListIntakes", "ListJobs", "ListPriceObservations", "ListReceiptProposals", "ListRecipes", "ListSchedules", "ListSessions", "ListTargets", "ListTemplates", "ListWorkspaces", "PreviewRecipesImport", "PreviewSwap", "PreviewWorkspaceImport":
		return "read", true
	case "ApplyWorkspaceImport", "RecoverRestoreCheckpoint":
		return "destructive", true
	case "ApplyPlan", "ApplyProfile", "ApplyReceiptProposal", "ApplyRecipesImport", "CancelJob", "ConfirmShoppingPurchases", "ConsumeBatchPortion", "CorrectBatchYield", "CreateJob", "CreatePriceObservation", "CreateRecipe", "CreateRevision", "CreateSchedule", "CreateTarget", "CreateTemplate", "CreateWorkspace", "GenerateOccurrences", "GeneratePlan", "PrepareBatch", "RecordEvent", "RecordFeedback", "RecordIntake", "SaveProfileDraft", "SaveSession", "SetShoppingChecked", "SetShoppingHaveThis", "StageReceiptProposal", "StartSession", "TransitionJob", "UndoBatchPortion", "UndoFeedback", "UpdateRecipe", "UpdateSchedule", "UpdateTemplate":
		return "write", true
	default:
		return "", false
	}
}
