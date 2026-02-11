package xrefs

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for cross-reference operations.
type Handlers struct {
	indexStore *IndexStore
}

// NewHandlers creates new xref handlers.
func NewHandlers(indexStore *IndexStore) *Handlers {
	return &Handlers{indexStore: indexStore}
}

// GetSkillXRefs handles GET /api/v1/skills/{id}/xrefs.
func (h *Handlers) GetSkillXRefs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	refs, err := h.indexStore.GetForSkill(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if refs == nil {
		refs = []Reference{}
	}

	resp := SkillXRefsResponse{
		SkillID:    id,
		References: refs,
		Total:      len(refs),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
