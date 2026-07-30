package backlog

import (
	"encoding/json"
	"net/http"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

// ReviewDecide is a test-only adapter retained while legacy handler tests are
// converted to the Connect contract. Production registers no REST review
// decision route; the behavior under test is the transport-neutral mutation.
func (h *Handler) ReviewDecide(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "review-decide")
	if !ok {
		return
	}
	var req ReviewDecideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[backlog] review-decide", apierr.BadRequest("invalid request body: %s", err.Error()))
		return
	}
	if req.DecidedBy == "" {
		req.DecidedBy = "test-legacy-actor"
	}
	resp, err := h.DecideReview(r.Context(), kind, name, req)
	if err != nil {
		apierr.MapError(w, "[backlog] review-decide", err)
		return
	}
	if err := httputil.JSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] review-decide", apierr.Internal("failed to encode response"))
	}
}
