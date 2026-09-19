// Package content owns transport for variant content section reads.
package content

import "net/http"

type Dependencies struct {
	AllSections func(string) (any, error)
	Path        func(*http.Request, string) (string, bool)
	WriteJSON   func(http.ResponseWriter, any)
	WriteError  func(http.ResponseWriter, int, string, string)
	Log         func(string, map[string]any)
}

func Admin(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug, ok := deps.Path(r, "variant_slug")
		if !ok || slug == "" {
			deps.WriteError(w, http.StatusBadRequest, "Variant slug is required", "validation")
			return
		}
		items, err := deps.AllSections(slug)
		if err != nil {
			deps.Log("sections_get_failed", map[string]any{"slug": slug, "error": err.Error()})
			deps.WriteError(w, http.StatusNotFound, "Variant not found", "not_found")
			return
		}
		deps.WriteJSON(w, map[string]any{"sections": items})
	}
}
