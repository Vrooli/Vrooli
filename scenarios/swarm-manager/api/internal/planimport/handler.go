package planimport

import (
	"encoding/json"
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/identity"

	"github.com/gorilla/mux"
)

// Handler serves the plan-import bridge over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler creates a plan-import Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers the Create-Work-From-Plan HTTP surface.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/plan-import/plans", h.ListPlans).Methods("GET")
	r.HandleFunc("/api/v1/plan-import", h.Import).Methods("POST")
}

type importRequest struct {
	PlanID     string                 `json:"plan_id"`
	SourcePath string                 `json:"source_path,omitempty"`
	Markdown   string                 `json:"markdown,omitempty"`
	Title      string                 `json:"title,omitempty"`
	Slug       string                 `json:"slug,omitempty"`
	Container  importContainerRequest `json:"container,omitempty"`
	Mode       string                 `json:"mode,omitempty"`
}

type importContainerRequest struct {
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Mode        string `json:"mode,omitempty"`
}

type listPlansResponse struct {
	Plans []PlanSummary `json:"plans"`
}

// ListPlans serves the small plan picker proxy. The full plan model remains in
// plan-manager; swarm-manager only needs stable ids/slugs for binding work.
func (h *Handler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.ListPlans(r.Context())
	if err != nil {
		apierr.MapError(w, "[plan-import]", err)
		return
	}
	if err := httputil.JSON(w, listPlansResponse{Plans: plans}); err != nil {
		apierr.MapError(w, "[plan-import]", apierr.Internal("failed to encode response"))
	}
}

func (r *importContainerRequest) UnmarshalJSON(data []byte) error {
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		r.Type = asString
		return nil
	}
	type alias importContainerRequest
	var obj alias
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*r = importContainerRequest(obj)
	return nil
}

// Import handles POST /api/v1/plan-import. The legacy shape
// {"plan_id":"<id-or-slug>"} still imports backlog items; callers may also
// provide source_path/markdown for plan-manager adoption and container
// selection for initiative creation.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[plan-import]", apierr.BadRequest("invalid request body"))
		return
	}
	planID := strings.TrimSpace(req.PlanID)
	if planID == "" && strings.TrimSpace(req.SourcePath) == "" && strings.TrimSpace(req.Markdown) == "" {
		apierr.MapError(w, "[plan-import]", apierr.BadRequest("plan_id, source_path, or markdown is required"))
		return
	}
	containerType := strings.TrimSpace(req.Container.Type)
	if containerType == "" && strings.TrimSpace(req.Container.Mode) != "" {
		containerType = "initiative"
	}
	containerMode := strings.TrimSpace(req.Container.Mode)
	if containerMode == "" {
		containerMode = strings.TrimSpace(req.Mode)
	}
	result, err := h.svc.Import(r.Context(), Request{
		PlanID:     planID,
		SourcePath: strings.TrimSpace(req.SourcePath),
		Markdown:   req.Markdown,
		Title:      strings.TrimSpace(req.Title),
		Slug:       strings.TrimSpace(req.Slug),
		Container: ContainerSpec{
			Type:        containerType,
			Name:        strings.TrimSpace(req.Container.Name),
			Title:       strings.TrimSpace(req.Container.Title),
			Description: strings.TrimSpace(req.Container.Description),
			Mode:        containerMode,
		},
	}, identity.FromContext(r.Context()))
	if err != nil {
		apierr.MapError(w, "[plan-import]", err)
		return
	}
	if err := httputil.JSONWithStatus(w, http.StatusCreated, result); err != nil {
		apierr.MapError(w, "[plan-import]", apierr.Internal("failed to encode response"))
	}
}
