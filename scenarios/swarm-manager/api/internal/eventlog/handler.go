package eventlog

import (
	"net/http"
	"strconv"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// Handler exposes bounded, entity-scoped event history. The event log remains
// append-only; this is a read projection for operator timelines.
type Handler struct{ repo Repository }

func NewHandler(repo Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/events", h.QueryByEntity).Methods(http.MethodGet)
}

// QueryByEntity handles GET /api/v1/events?entity=backlog/<kind>/<name>.
func (h *Handler) QueryByEntity(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.repo == nil {
		apierr.MapError(w, "[events] query", apierr.Unavailable("event log is unavailable"))
		return
	}
	entityType, entityID, err := parseEntity(strings.TrimSpace(r.URL.Query().Get("entity")))
	if err != nil {
		apierr.MapError(w, "[events] query", apierr.BadRequest("%s", err.Error()))
		return
	}
	after, err := parseNonNegativeInt64(r.URL.Query().Get("since_id"))
	if err != nil {
		apierr.MapError(w, "[events] query", apierr.BadRequest("since_id must be a non-negative integer"))
		return
	}
	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		apierr.MapError(w, "[events] query", apierr.BadRequest("%s", err.Error()))
		return
	}
	events, err := h.repo.QueryByEntity(r.Context(), entityType, entityID, after, limit)
	if err != nil {
		apierr.MapError(w, "[events] query", err)
		return
	}
	if err := httputil.JSON(w, map[string]any{"items": events}); err != nil {
		apierr.MapError(w, "[events] query", apierr.Internal("encode event history"))
	}
}

func parseEntity(raw string) (EntityType, string, error) {
	parts := strings.Split(raw, "/")
	if len(parts) != 3 || parts[0] != "backlog" || strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[2]) == "" {
		return "", "", apierr.BadRequest("entity must be backlog/<kind>/<name>")
	}
	return EntityBacklogItem, parts[1] + "/" + parts[2], nil
}

func parseNonNegativeInt64(raw string) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, apierr.BadRequest("invalid integer")
	}
	return value, nil
}

func parseLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 100, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 250 {
		return 0, apierr.BadRequest("limit must be between 1 and 250")
	}
	return value, nil
}
