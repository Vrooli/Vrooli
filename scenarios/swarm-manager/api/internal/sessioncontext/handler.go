package sessioncontext

import (
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

type Handler struct {
	resolver *Resolver
}

type briefResponse struct {
	Brief agentsessions.ContextItem `json:"brief"`
}

func NewHandler(resolver *Resolver) *Handler {
	return &Handler{resolver: resolver}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/portfolio/brief", h.PortfolioBrief).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/operating-mode/brief", h.OperatingModeBrief).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/initiative-candidates", h.InitiativeCandidates).Methods(http.MethodGet)
}

func (h *Handler) PortfolioBrief(w http.ResponseWriter, r *http.Request) {
	h.writeStartupBrief(w, r, agentsessions.KindMetaOrchestration, "[session-context] portfolio brief")
}

func (h *Handler) OperatingModeBrief(w http.ResponseWriter, r *http.Request) {
	h.writeStartupBrief(w, r, agentsessions.KindOperatingModeAuthoring, "[session-context] operating-mode brief")
}

func (h *Handler) InitiativeCandidates(w http.ResponseWriter, r *http.Request) {
	purpose := strings.TrimSpace(r.URL.Query().Get("purpose"))
	if purpose != "" && purpose != "next-action" {
		apierr.MapError(w, "[session-context] initiative candidates", apierr.BadRequest("unsupported purpose %q", purpose))
		return
	}
	h.writeStartupBrief(w, r, agentsessions.KindMetaOrchestration, "[session-context] initiative candidates")
}

func (h *Handler) writeStartupBrief(w http.ResponseWriter, r *http.Request, kind agentsessions.Kind, label string) {
	if h.resolver == nil {
		apierr.MapError(w, label, apierr.Unavailable("startup brief resolver is unavailable"))
		return
	}
	brief, err := h.resolver.ResolveSessionStartupBrief(r.Context(), kind, agentsessions.ContextLimits{
		Kind:            kind,
		MaxSummaryRunes: 1200,
	})
	if err != nil {
		apierr.MapError(w, label, err)
		return
	}
	if brief.SelectedAt == "" {
		brief.SelectedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := httputil.JSON(w, briefResponse{Brief: brief}); err != nil {
		apierr.MapError(w, label, apierr.Internal("failed to encode brief"))
	}
}
