// DOC: docs/QUICKSTART.md#Register-a-Domain
// DOC: PRD.md#OT-P0-002
// DOC: README.md#Domains
// DOC: docs/internal/ERROR_SEMANTICS.md
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"lifestyle-dashboard/config"
	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/errors"
	"lifestyle-dashboard/repository"
)

// RegisterDomain handles POST /api/v1/domains - P0-002
// [REQ:LD-DOMAIN-REGISTER] Registers or updates a domain scenario.
func (h *Handler) RegisterDomain(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, errors.ErrInvalidJSON)
		return
	}

	// Validate required fields with specific error messages
	if req.Name == "" {
		WriteAPIError(w, errors.ErrMissingName)
		return
	}
	if req.DisplayName == "" {
		WriteAPIError(w, errors.ErrMissingDisplayName)
		return
	}

	// Build domain from request
	d := &domain.Domain{
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Capabilities: req.Capabilities,
		HealthURL:    req.HealthURL,
	}

	// Delegate to repository (handles timestamps and defaults)
	if err := h.Domains.Upsert(r.Context(), d); err != nil {
		log.Printf("[ERROR] RegisterDomain(%s): database error: %v", req.Name, err)
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to register domain. Please try again."))
		return
	}

	WriteJSON(w, http.StatusCreated, d)
}

// ListDomains handles GET /api/v1/domains - P0-002
// [REQ:LD-DOMAIN-DISCOVER] Lists all registered domains.
func (h *Handler) ListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := h.Domains.List(r.Context())
	if err != nil {
		log.Printf("[ERROR] ListDomains: database error: %v", err)
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to list domains. Please try again."))
		return
	}

	WriteJSON(w, http.StatusOK, domain.DomainsResponse{
		Domains: domains,
		Count:   len(domains),
	})
}

// GetDomain handles GET /api/v1/domains/{name} - P0-002
// [REQ:LD-DOMAIN-DISCOVER] Retrieves a single domain by name.
func (h *Handler) GetDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	d, err := h.Domains.GetByName(r.Context(), name)
	if repository.IsNotFound(err) {
		WriteAPIError(w, errors.NewNotFoundError(errors.CodeDomainNotFound, "domain", name))
		return
	}
	if err != nil {
		log.Printf("[ERROR] GetDomain(%s): database error: %v", name, err)
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to retrieve domain. Please try again."))
		return
	}

	WriteJSON(w, http.StatusOK, d)
}

// UpdateDomain handles PATCH /api/v1/domains/{name}
// [REQ:LD-DOMAIN-REGISTER] Updates domain attributes.
func (h *Handler) UpdateDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		WriteAPIError(w, errors.ErrInvalidJSON)
		return
	}

	err := h.Domains.Update(r.Context(), name, updates)
	if repository.IsNotFound(err) {
		WriteAPIError(w, errors.NewNotFoundError(errors.CodeDomainNotFound, "domain", name))
		return
	}
	if err != nil {
		log.Printf("[ERROR] UpdateDomain(%s): database error: %v", name, err)
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to update domain. Please try again."))
		return
	}

	// Fetch and return updated domain
	h.GetDomain(w, r)
}

// GetDomainHealth handles GET /api/v1/domains/{name}/health
// [REQ:LD-DOMAIN-HEALTH] Checks domain health via configured health URL.
func (h *Handler) GetDomainHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	d, err := h.Domains.GetByName(r.Context(), name)
	if repository.IsNotFound(err) {
		WriteAPIError(w, errors.NewNotFoundError(errors.CodeDomainNotFound, "domain", name))
		return
	}
	if err != nil {
		log.Printf("[ERROR] GetDomainHealth(%s): database error: %v", name, err)
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to retrieve domain. Please try again."))
		return
	}

	// If no health URL, just return current status
	if d.HealthURL == "" {
		lastCheck := ""
		if d.LastHealthAt != nil {
			lastCheck = *d.LastHealthAt
		}
		WriteJSON(w, http.StatusOK, domain.HealthCheckResponse{
			Domain:    name,
			Status:    d.Status,
			LastCheck: lastCheck,
			Message:   "no health URL configured",
		})
		return
	}

	// Check health URL using configured timeout
	cfg := config.DefaultHealthCheckConfig()
	ctx, cancel := context.WithTimeout(r.Context(), cfg.Timeout)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", d.HealthURL, nil)
	client := &http.Client{Timeout: cfg.Timeout}
	resp, healthErr := client.Do(req)

	now := time.Now().UTC().Format(time.RFC3339)

	// Determine status using centralized decision helper
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
		resp.Body.Close()
	}
	result := domain.DetermineHealthCheckResult(healthErr, statusCode, cfg.UnhealthyThreshold)

	if healthErr != nil {
		log.Printf("[WARN] GetDomainHealth(%s): health check failed: %v", name, healthErr)
	}

	// Update domain status via repository (uses DomainStatus for storage)
	if updateErr := h.Domains.UpdateStatus(r.Context(), name, string(result.DomainStatus), now); updateErr != nil {
		log.Printf("[WARN] GetDomainHealth(%s): failed to update status: %v", name, updateErr)
	}

	// API response uses HealthStatus (healthy/unhealthy)
	WriteJSON(w, http.StatusOK, domain.HealthCheckResponse{
		Domain:    name,
		Status:    string(result.ResponseStatus),
		LastCheck: now,
		Message:   result.Message,
	})
}
