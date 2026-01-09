package handlers

import (
	"encoding/json"
	"net/http"
)

// HandleTemplateOnly makes clear that specific capabilities belong to generated landing scenarios, not the factory.
func HandleTemplateOnly(feature string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		response := map[string]string{
			"status":  "template_only",
			"feature": feature,
			"message": "Use a generated landing scenario to access this capability; the landing-manager factory only creates templates and scenarios.",
		}
		_ = json.NewEncoder(w).Encode(response)
	}
}
