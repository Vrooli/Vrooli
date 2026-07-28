// Package variant_space owns verbatim variant-space HTTP transport.
package variant_space

import "net/http"

type Dependencies struct {
	JSON func() []byte
	Log  func(string, map[string]any)
}

func Get(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(deps.JSON()); err != nil {
			deps.Log("variant_space_write_failed", map[string]any{"error": err.Error()})
		}
	}
}
