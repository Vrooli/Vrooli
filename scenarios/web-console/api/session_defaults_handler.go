package main

import "net/http"

// SessionDefaultsResponse is the JSON shape for session defaults settings.
type SessionDefaultsResponse struct {
	DefaultBackend string            `json:"default_backend"`
	DefaultPolicy  *ExpirationPolicy `json:"default_policy"`
}

// handleGetSessionDefaults returns the current session defaults.
// GET /api/v1/settings/session-defaults
func (s *Server) handleGetSessionDefaults(w http.ResponseWriter, _ *http.Request) {
	cfg := s.sessions.GetConfig()
	resp := SessionDefaultsResponse{
		DefaultBackend: cfg.DefaultBackend,
		DefaultPolicy: &ExpirationPolicy{
			Mode:     PolicyMode(cfg.DefaultPolicyMode),
			Duration: cfg.DefaultPolicyDuration,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

// UpdateSessionDefaultsRequest is the JSON body for updating session defaults.
type UpdateSessionDefaultsRequest struct {
	DefaultBackend *string           `json:"default_backend,omitempty"`
	DefaultPolicy  *ExpirationPolicy `json:"default_policy,omitempty"`
}

// handleUpdateSessionDefaults updates the session defaults.
// PUT /api/v1/settings/session-defaults
func (s *Server) handleUpdateSessionDefaults(w http.ResponseWriter, r *http.Request) {
	var req UpdateSessionDefaultsRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.DefaultBackend != nil {
		bid := BackendID(*req.DefaultBackend)
		if _, ok := s.backendRegistry.Get(bid); !ok {
			writeCatalogError(w, "backend_unknown", "Unknown backend: "+*req.DefaultBackend)
			return
		}
		s.sessions.SetConfigField(func(cfg *Config) {
			cfg.DefaultBackend = *req.DefaultBackend
		})
	}

	if req.DefaultPolicy != nil {
		if err := ValidatePolicy(*req.DefaultPolicy); err != nil {
			writeCatalogError(w, "invalid_policy", err.Error())
			return
		}
		s.sessions.SetConfigField(func(cfg *Config) {
			cfg.DefaultPolicyMode = string(req.DefaultPolicy.Mode)
			cfg.DefaultPolicyDuration = req.DefaultPolicy.Duration
		})
	}

	// Return updated state
	s.handleGetSessionDefaults(w, r)
}
