// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/SEAMS.md

package main

import (
	"net/http"
)

// AIProviderConfigResponse wraps config + health for the API response.
type AIProviderConfigResponse struct {
	Providers []ProviderConfig `json:"providers"`
	Health    []ProviderHealth `json:"health"`
}

// ProviderConfigUpdateRequest is the JSON body for updating a provider config.
type ProviderConfigUpdateRequest struct {
	Name       string `json:"name"`
	Enabled    *bool  `json:"enabled,omitempty"`
	Priority   *int   `json:"priority,omitempty"`
	TimeoutSec *int   `json:"timeout_sec,omitempty"`
	MaxRetries *int   `json:"max_retries,omitempty"`
}

// handleGetAIConfig returns provider configuration and health.
// GET /api/v1/ai/config
// [REQ:P1-003a] Provider Configuration Storage
// [REQ:P1-003b] Provider Health Dashboard
func (s *Server) handleGetAIConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, AIProviderConfigResponse{
		Providers: s.aiConfig.GetConfigs(),
		Health:    s.aiConfig.GetHealth(),
	})
}

// handleUpdateAIConfig updates a provider's configuration.
// PUT /api/v1/ai/config
// [REQ:P1-003a] Provider Configuration Storage
func (s *Server) handleUpdateAIConfig(w http.ResponseWriter, r *http.Request) {
	var req ProviderConfigUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Name == "" {
		writeCatalogError(w, "invalid_body", "Provider name is required")
		return
	}

	// Read current config and apply partial updates
	configs := s.aiConfig.GetConfigs()
	var current *ProviderConfig
	for i := range configs {
		if configs[i].Name == req.Name {
			current = &configs[i]
			break
		}
	}
	if current == nil {
		writeCatalogError(w, "invalid_body", "Unknown provider: "+req.Name)
		return
	}

	enabled := current.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	priority := current.Priority
	if req.Priority != nil {
		priority = *req.Priority
	}
	timeoutSec := current.TimeoutSec
	if req.TimeoutSec != nil {
		if *req.TimeoutSec < 1 || *req.TimeoutSec > 120 {
			writeCatalogError(w, "invalid_body", "Timeout must be between 1 and 120 seconds")
			return
		}
		timeoutSec = *req.TimeoutSec
	}
	maxRetries := current.MaxRetries
	if req.MaxRetries != nil {
		if *req.MaxRetries < 0 || *req.MaxRetries > 5 {
			writeCatalogError(w, "invalid_body", "Max retries must be between 0 and 5")
			return
		}
		maxRetries = *req.MaxRetries
	}

	s.aiConfig.UpdateConfig(req.Name, enabled, priority, timeoutSec, maxRetries)

	writeJSON(w, http.StatusOK, AIProviderConfigResponse{
		Providers: s.aiConfig.GetConfigs(),
		Health:    s.aiConfig.GetHealth(),
	})
}

// handleGetAIHealth returns provider health status only.
// GET /api/v1/ai/health
// [REQ:P1-003b] Provider Health Dashboard
func (s *Server) handleGetAIHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.aiConfig.GetHealth())
}
