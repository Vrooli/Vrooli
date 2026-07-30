package administration

import (
	"encoding/json"
	"net/http"

	admin "landing-page-business-suite-api/internal/administration"
)

type APIKeyDependencies struct {
	Service    *admin.APIKeyService
	WriteError func(http.ResponseWriter, int, string, string)
	LogError   func(string, map[string]any)
}

func ListAPIKeys(deps APIKeyDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys, err := deps.Service.List(r.Context())
		if err != nil {
			deps.LogError("list_api_keys_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to list API keys", "server_error")
			return
		}
		if keys == nil {
			keys = []admin.APIKey{}
		}
		writeAPIKeyJSON(w, map[string]any{"keys": keys}, deps)
	}
}

func CreateAPIKey(deps APIKeyDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request admin.APIKeyCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		key, err := deps.Service.Store(r.Context(), request.Provider, request.Key)
		if err != nil {
			deps.LogError("create_api_key_failed", map[string]any{"error": err.Error(), "provider": request.Provider})
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(key); err != nil {
			deps.LogError("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}

func DeleteAPIKey(deps APIKeyDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := r.URL.Query().Get("provider")
		if provider == "" {
			deps.WriteError(w, http.StatusBadRequest, "Provider is required", "validation")
			return
		}
		if err := deps.Service.Delete(r.Context(), provider); err != nil {
			deps.LogError("delete_api_key_failed", map[string]any{"error": err.Error(), "provider": provider})
			deps.WriteError(w, http.StatusNotFound, err.Error(), "not_found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func TestAPIKey(deps APIKeyDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := r.URL.Query().Get("provider")
		if provider == "" {
			deps.WriteError(w, http.StatusBadRequest, "Provider is required", "validation")
			return
		}
		success, message, err := deps.Service.Test(r.Context(), provider)
		if err != nil {
			deps.LogError("test_api_key_failed", map[string]any{"error": err.Error(), "provider": provider})
			deps.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error")
			return
		}
		writeAPIKeyJSON(w, map[string]any{"success": success, "message": message, "provider": provider}, deps)
	}
}

func ToggleAPIKey(deps APIKeyDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Provider string `json:"provider"`
			Active   bool   `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		if err := deps.Service.SetActive(r.Context(), request.Provider, request.Active); err != nil {
			deps.LogError("toggle_api_key_failed", map[string]any{"error": err.Error(), "provider": request.Provider})
			deps.WriteError(w, http.StatusNotFound, err.Error(), "not_found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeAPIKeyJSON(w http.ResponseWriter, value any, deps APIKeyDependencies) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		deps.LogError("encode_response_failed", map[string]any{"error": err.Error()})
	}
}
