// Package branding owns HTTP transport for site-wide branding configuration.
package branding

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Dependencies struct {
	Get        func() any
	Public     func() any
	Update     func(json.RawMessage) (any, error)
	Clear      func(string) error
	DecodeJSON func(http.ResponseWriter, *http.Request, any) bool
	WriteJSON  func(http.ResponseWriter, any)
	WriteError func(http.ResponseWriter, int, string, string)
	Log        func(string, map[string]any)
}

func Get(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { deps.WriteJSON(w, deps.Get()) }
}
func Public(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { deps.WriteJSON(w, deps.Public()) }
}
func Update(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var raw json.RawMessage
		if !deps.DecodeJSON(w, r, &raw) {
			return
		}
		branding, err := deps.Update(raw)
		if err != nil {
			deps.Log("update_branding_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to update branding", "server_error")
			return
		}
		deps.WriteJSON(w, branding)
	}
}
func Clear(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Field string `json:"field"`
		}
		if !deps.DecodeJSON(w, r, &req) {
			return
		}
		field := strings.TrimSpace(req.Field)
		if field == "" {
			deps.WriteError(w, http.StatusBadRequest, "Field name is required", "validation")
			return
		}
		if err := deps.Clear(field); err != nil {
			deps.Log("clear_branding_field_failed", map[string]any{"field": field, "error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to clear field", "server_error")
			return
		}
		deps.WriteJSON(w, deps.Get())
	}
}
