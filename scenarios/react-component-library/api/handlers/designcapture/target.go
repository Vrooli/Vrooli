package designcapture

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"react-component-library/internal/preview"
	"strings"

	"github.com/gorilla/mux"
	domain "react-component-library/internal/designcapture"
)

// TargetHandler serves only persisted, hash-verified renderer output. The
// repository factory must bind the caller's routed storage context.
type TargetHandler struct {
	RepositoryFor func(context.Context) (domain.Repository, error)
}

func (h TargetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.RepositoryFor == nil {
		http.Error(w, "capture repository unavailable", http.StatusServiceUnavailable)
		return
	}
	repo, err := h.RepositoryFor(r.Context())
	if err != nil || repo == nil {
		http.Error(w, "capture repository unavailable", http.StatusServiceUnavailable)
		return
	}
	op, err := repo.Get(r.Context(), mux.Vars(r)["id"])
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "capture target is unverifiable", http.StatusConflict)
		return
	}
	// Preserve exact HTML bytes. The sandbox header isolates direct browser
	// navigation while allowing the existing versioned runtime module loader.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	policy := "sandbox allow-scripts"
	if strings.Contains(op.Request.HTML, preview.LocalSubmitPolicyMarker) {
		policy = "sandbox allow-scripts allow-forms; form-action 'none'"
	}
	w.Header().Set("Content-Security-Policy", policy)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("ETag", `"`+op.Request.Target.HTMLSHA256+`"`)
	w.Header().Set("X-RCL-Render-Hash", op.Request.Target.RenderHash)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(op.Request.HTML))
}
func MountTarget(router *mux.Router, factory func(context.Context) (domain.Repository, error)) {
	router.Handle("/design-captures/{id}/target.html", TargetHandler{RepositoryFor: factory}).Methods(http.MethodGet, http.MethodHead)
}
