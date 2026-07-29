// Package variant owns HTTP transport for variant selection and reads.
package variant

import "net/http"

type Dependencies struct {
	Select     func() (any, error)
	Get        func(string) (any, error)
	List       func() any
	Slug       func(*http.Request) string
	WriteJSON  func(http.ResponseWriter, any)
	WriteError func(http.ResponseWriter, int, string, string)
	Log        func(string, map[string]any)
}

func Select(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			deps.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}
		selected, err := deps.Select()
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "No variants available.", "server_error")
			return
		}
		deps.WriteJSON(w, selected)
	}
}

func PublicGet(deps Dependencies) http.HandlerFunc {
	return get(deps, "public_variant_fetch_failed", true)
}
func AdminGet(deps Dependencies) http.HandlerFunc { return get(deps, "variant_fetch_failed", false) }
func List(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			deps.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
			return
		}
		deps.WriteJSON(w, map[string]any{"variants": deps.List()})
	}
}

func get(deps Dependencies, event string, public bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			deps.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}
		slug := deps.Slug(r)
		if slug == "" || (!public && slug == "select") {
			deps.WriteError(w, http.StatusBadRequest, "Variant slug is required.", "validation")
			return
		}
		item, err := deps.Get(slug)
		if err != nil {
			deps.Log(event, map[string]any{"slug": slug, "error": err.Error()})
			deps.WriteError(w, http.StatusNotFound, "Variant not found.", "not_found")
			return
		}
		deps.WriteJSON(w, item)
	}
}
