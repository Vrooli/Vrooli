package diagnostics

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"
	internal "nutrition-planner/internal/diagnostics"
	"nutrition-planner/internal/module"
	"nutrition-planner/internal/workspace"
)

type moduleHandler struct {
	db         internal.SQLExecutor
	workspaces workspace.Service
}

func Module(db internal.SQLExecutor, workspaces workspace.Service) module.Module {
	h := &moduleHandler{db: db, workspaces: workspaces}
	return module.Module{Name: "diagnostics", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/diagnostics", h.handle).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func (h *moduleHandler) handle(w http.ResponseWriter, req *http.Request) {
	principal, ok := identity.PrincipalFromContext(req.Context())
	if !ok {
		http.Error(w, "verified actor required", http.StatusUnauthorized)
		return
	}
	workspaceID := req.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		http.Error(w, "workspace_id is required", http.StatusBadRequest)
		return
	}
	if _, err := h.workspaces.Get(req.Context(), workspaceID, principal.Subject); err != nil {
		if _, forbidden := err.(workspace.ErrForbidden); forbidden {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	report, err := internal.Build(req.Context(), h.db, workspaceID, now(), providerStatuses())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	report.ImageGeneration = imageGenerationStatus(req.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

// imageGenerationStatus calls Image Tools' read-only operation-candidate
// surface. It never selects a model, submits work, or contacts a model provider.
func imageGenerationStatus(ctx context.Context) internal.ImageGenerationStatus {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "image-tools")
	if err != nil || baseURL == "" {
		return internal.ImageGenerationFromCandidates(false, false, nil)
	}
	client := modelsconnect.NewModelsServiceClient(&http.Client{Timeout: 2 * time.Second}, baseURL)
	response, err := client.ListOperationModels(ctx, connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "text_to_image"}))
	if err != nil {
		return internal.ImageGenerationFromCandidates(true, false, nil)
	}
	readyStates := make([]string, 0, len(response.Msg.GetCandidates()))
	for _, candidate := range response.Msg.GetCandidates() {
		readyStates = append(readyStates, candidate.GetReadyState())
	}
	return internal.ImageGenerationFromCandidates(true, true, readyStates)
}

var now = func() time.Time { return time.Now().UTC() }

func providerStatuses() []internal.ProviderStatus {
	return []internal.ProviderStatus{
		configuredStatus("nutrition_fdc", os.Getenv("NUTRITION_FDC_API_KEY"), "Set NUTRITION_FDC_API_KEY to enable server-side nutrition lookup."),
		configuredStatus("ai_gateway", os.Getenv("AI_GATEWAY_URL"), "Configure AI_GATEWAY_URL to enable optional extraction assistance."),
		{Name: "price", State: "manual", Reason: "Manual price entry remains available; no checkout adapter is enabled."},
		{Name: "receipt", State: "manual", Reason: "Receipt proposals require an explicitly configured adapter and review."},
	}
}

func configuredStatus(name, value, reason string) internal.ProviderStatus {
	if value != "" {
		return internal.ProviderStatus{Name: name, State: "configured", Reason: "Configuration is present; provider calls remain optional."}
	}
	return internal.ProviderStatus{Name: name, State: "not_configured", Reason: reason}
}
