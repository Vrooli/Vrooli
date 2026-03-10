package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/repository"
)

// RegisterDomain handles POST /api/v1/domains - P0-002
// [REQ:LD-DOMAIN-REGISTER] Registers or updates a domain scenario.
func (h *Handler) RegisterDomain(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name == "" || req.DisplayName == "" {
		WriteError(w, http.StatusBadRequest, "name and display_name are required")
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
		log.Printf("Error registering domain: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to register domain")
		return
	}

	WriteJSON(w, http.StatusCreated, d)
}

// ListDomains handles GET /api/v1/domains - P0-002
// [REQ:LD-DOMAIN-DISCOVER] Lists all registered domains.
func (h *Handler) ListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := h.Domains.List(r.Context())
	if err != nil {
		log.Printf("Error listing domains: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to list domains")
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
		WriteError(w, http.StatusNotFound, "domain not found")
		return
	}
	if err != nil {
		log.Printf("Error getting domain: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to get domain")
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
		WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	err := h.Domains.Update(r.Context(), name, updates)
	if repository.IsNotFound(err) {
		WriteError(w, http.StatusNotFound, "domain not found")
		return
	}
	if err != nil {
		log.Printf("Error updating domain: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to update domain")
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
		WriteError(w, http.StatusNotFound, "domain not found")
		return
	}
	if err != nil {
		log.Printf("Error getting domain health: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to get domain")
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

	// Check health URL
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", d.HealthURL, nil)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)

	now := time.Now().UTC().Format(time.RFC3339)
	status := "healthy"
	if err != nil || resp.StatusCode >= 300 {
		status = "unhealthy"
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Update domain status via repository
	h.Domains.UpdateStatus(r.Context(), name, status, now)

	WriteJSON(w, http.StatusOK, domain.HealthCheckResponse{
		Domain:    name,
		Status:    status,
		LastCheck: now,
	})
}
