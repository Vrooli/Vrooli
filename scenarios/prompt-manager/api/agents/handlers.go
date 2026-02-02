package agents

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"prompt-manager/store"
	"prompt-manager/validation"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for agent operations.
type Handlers struct {
	agentStore    store.AgentStore
	relationStore store.RelationStore
	indexStore    store.IndexStore
}

// NewHandlers creates a new agents handler.
func NewHandlers(agentStore store.AgentStore, relationStore store.RelationStore, indexStore store.IndexStore) *Handlers {
	return &Handlers{
		agentStore:    agentStore,
		relationStore: relationStore,
		indexStore:    indexStore,
	}
}

// List handles GET /agents - returns all agents.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	agents, err := h.agentStore.List(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responses := make([]Response, 0, len(agents))
	for _, a := range agents {
		resp := h.toResponse(ctx, &a)
		responses = append(responses, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responses)
}

// Get handles GET /agents/{id} - returns a single agent.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	agent, err := h.agentStore.Get(ctx, id)
	if err != nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, agent))
}

// Create handles POST /agents - creates a new agent.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.DisplayName == "" {
		http.Error(w, "displayName is required", http.StatusBadRequest)
		return
	}

	// Generate ID if not provided
	id := req.ID
	if id == "" {
		id = validation.Slugify(req.DisplayName)
	}

	// Validate appearance colors if provided
	if req.Appearance != nil {
		if !validation.IsValidHexColor(req.Appearance.Body) {
			http.Error(w, "Invalid body color format (expected hex color)", http.StatusBadRequest)
			return
		}
		if !validation.IsValidHexColor(req.Appearance.Head) {
			http.Error(w, "Invalid head color format (expected hex color)", http.StatusBadRequest)
			return
		}
		if !validation.IsValidHexColor(req.Appearance.Accent) {
			http.Error(w, "Invalid accent color format (expected hex color)", http.StatusBadRequest)
			return
		}
	}

	agent := &store.Agent{
		ID:                id,
		DisplayName:       req.DisplayName,
		Description:       req.Description,
		Status:            store.AgentStatusActive,
		DefaultProfileRef: req.DefaultProfileRef,
		Tags:              req.Tags,
	}

	if req.Appearance != nil {
		agent.Appearance = &store.AgentAppearance{
			Body:   req.Appearance.Body,
			Head:   req.Appearance.Head,
			Accent: req.Appearance.Accent,
		}
	}

	if req.Capabilities != nil {
		agent.Capabilities = &store.AgentCapabilities{}
		if req.Capabilities.Provides != nil {
			for _, p := range req.Capabilities.Provides {
				agent.Capabilities.Provides = append(agent.Capabilities.Provides, store.AgentCapability{
					CapabilityID: p.CapabilityID,
					Verbs:        p.Verbs,
				})
			}
		}
		if req.Capabilities.Requires != nil {
			for _, r := range req.Capabilities.Requires {
				agent.Capabilities.Requires = append(agent.Capabilities.Requires, store.AgentCapability{
					CapabilityID: r.CapabilityID,
					Verbs:        r.Verbs,
				})
			}
		}
	}

	if req.Connectors != nil {
		for _, c := range req.Connectors {
			agent.Connectors = append(agent.Connectors, store.AgentConnector{
				Type:    c.Type,
				ID:      c.ID,
				Enabled: c.Enabled,
			})
		}
	}

	if req.Heartbeat != nil {
		agent.Heartbeat = &store.AgentHeartbeat{
			IntervalSeconds: req.Heartbeat.IntervalSeconds,
			TimeoutSeconds:  req.Heartbeat.TimeoutSeconds,
			MaxMissedBeats:  req.Heartbeat.MaxMissedBeats,
		}
	}

	if err := h.agentStore.Create(ctx, agent); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create skill relations if skills provided
	if len(req.Skills) > 0 && h.relationStore != nil {
		for _, skillID := range req.Skills {
			rel := &store.AgentSkillRelation{
				AgentID: id,
				SkillID: skillID,
				Pin:     "latest",
				Enabled: true,
			}
			_ = h.relationStore.SetAgentSkill(ctx, rel)
		}
	}

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateAgents(ctx)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, agent))
}

// Update handles PUT /agents/{id} - updates an existing agent.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := h.agentStore.Get(ctx, id); err != nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Build updates
	updates := &store.Agent{}
	if req.DisplayName != nil {
		updates.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		updates.Description = *req.Description
	}
	if req.Status != nil {
		updates.Status = *req.Status
	}
	if req.Appearance != nil {
		if !validation.IsValidHexColor(req.Appearance.Body) {
			http.Error(w, "Invalid body color format (expected hex color)", http.StatusBadRequest)
			return
		}
		if !validation.IsValidHexColor(req.Appearance.Head) {
			http.Error(w, "Invalid head color format (expected hex color)", http.StatusBadRequest)
			return
		}
		if !validation.IsValidHexColor(req.Appearance.Accent) {
			http.Error(w, "Invalid accent color format (expected hex color)", http.StatusBadRequest)
			return
		}
		updates.Appearance = &store.AgentAppearance{
			Body:   req.Appearance.Body,
			Head:   req.Appearance.Head,
			Accent: req.Appearance.Accent,
		}
	}
	if req.Capabilities != nil {
		updates.Capabilities = &store.AgentCapabilities{}
		if req.Capabilities.Provides != nil {
			for _, p := range req.Capabilities.Provides {
				updates.Capabilities.Provides = append(updates.Capabilities.Provides, store.AgentCapability{
					CapabilityID: p.CapabilityID,
					Verbs:        p.Verbs,
				})
			}
		}
		if req.Capabilities.Requires != nil {
			for _, r := range req.Capabilities.Requires {
				updates.Capabilities.Requires = append(updates.Capabilities.Requires, store.AgentCapability{
					CapabilityID: r.CapabilityID,
					Verbs:        r.Verbs,
				})
			}
		}
	}
	if req.Connectors != nil {
		for _, c := range req.Connectors {
			updates.Connectors = append(updates.Connectors, store.AgentConnector{
				Type:    c.Type,
				ID:      c.ID,
				Enabled: c.Enabled,
			})
		}
	}
	if req.DefaultProfileRef != nil {
		updates.DefaultProfileRef = *req.DefaultProfileRef
	}
	if req.Heartbeat != nil {
		updates.Heartbeat = &store.AgentHeartbeat{
			IntervalSeconds: req.Heartbeat.IntervalSeconds,
			TimeoutSeconds:  req.Heartbeat.TimeoutSeconds,
			MaxMissedBeats:  req.Heartbeat.MaxMissedBeats,
		}
	}
	if req.Tags != nil {
		updates.Tags = req.Tags
	}

	if err := h.agentStore.Update(ctx, id, updates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update skill relations if skills provided
	if req.Skills != nil && h.relationStore != nil {
		// Get existing relations
		existingRels, _ := h.relationStore.ListAgentSkills(ctx, id)
		existingMap := make(map[string]bool)
		for _, rel := range existingRels {
			existingMap[rel.SkillID] = true
		}

		// Create set of desired skills
		desiredMap := make(map[string]bool)
		for _, skillID := range req.Skills {
			desiredMap[skillID] = true
		}

		// Add new relations
		for _, skillID := range req.Skills {
			if !existingMap[skillID] {
				rel := &store.AgentSkillRelation{
					AgentID: id,
					SkillID: skillID,
					Pin:     "latest",
					Enabled: true,
				}
				_ = h.relationStore.SetAgentSkill(ctx, rel)
			}
		}

		// Remove old relations
		for _, rel := range existingRels {
			if !desiredMap[rel.SkillID] {
				_ = h.relationStore.DeleteAgentSkill(ctx, id, rel.SkillID)
			}
		}
	}

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateAgents(ctx)
	}

	// Get updated agent
	agent, _ := h.agentStore.Get(ctx, id)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, agent))
}

// Delete handles DELETE /agents/{id} - deletes an agent.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.agentStore.Delete(ctx, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete all skill relations for this agent
	if h.relationStore != nil {
		rels, _ := h.relationStore.ListAgentSkills(ctx, id)
		for _, rel := range rels {
			_ = h.relationStore.DeleteAgentSkill(ctx, id, rel.SkillID)
		}
	}

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateAgents(ctx)
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetEffectiveSkills handles GET /agents/{id}/effective-skills
func (h *Handlers) GetEffectiveSkills(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	// Get optional team context from query param
	var teamID *string
	if t := r.URL.Query().Get("teamId"); t != "" {
		teamID = &t
	}

	skills, err := h.agentStore.GetEffectiveSkills(ctx, id, teamID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := EffectiveSkillsResponse{
		AgentID: id,
		TeamID:  teamID,
		Skills:  skills,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetSoul handles GET /agents/{id}/soul - returns the SOUL.md content for an agent.
func (h *Handlers) GetSoul(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	// Type assert to FileAgentStore to access GetSoul
	fileStore, ok := h.agentStore.(*store.FileAgentStore)
	if !ok {
		http.Error(w, "GetSoul not supported", http.StatusInternalServerError)
		return
	}

	content, err := fileStore.GetSoul(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := SoulResponse{
		AgentID: id,
		Content: content,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SetSoul handles PUT /agents/{id}/soul - sets the SOUL.md content for an agent.
func (h *Handlers) SetSoul(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	var req SoulRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Type assert to FileAgentStore to access SetSoul
	fileStore, ok := h.agentStore.(*store.FileAgentStore)
	if !ok {
		http.Error(w, "SetSoul not supported", http.StatusInternalServerError)
		return
	}

	if err := fileStore.SetSoul(ctx, id, req.Content); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := SoulResponse{
		AgentID: id,
		Content: req.Content,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Helper functions

func (h *Handlers) toResponse(ctx context.Context, a *store.Agent) Response {
	resp := Response{
		ID:                a.ID,
		DisplayName:       a.DisplayName,
		Description:       a.Description,
		Status:            a.Status,
		DefaultProfileRef: a.DefaultProfileRef,
		Tags:              a.Tags,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
	}

	if a.Appearance != nil {
		resp.Appearance = &AppearanceDTO{
			Body:   a.Appearance.Body,
			Head:   a.Appearance.Head,
			Accent: a.Appearance.Accent,
		}
	}

	if a.Capabilities != nil {
		resp.Capabilities = &CapabilitiesDTO{}
		if a.Capabilities.Provides != nil {
			for _, p := range a.Capabilities.Provides {
				resp.Capabilities.Provides = append(resp.Capabilities.Provides, CapabilityDTO{
					CapabilityID: p.CapabilityID,
					Verbs:        p.Verbs,
				})
			}
		}
		if a.Capabilities.Requires != nil {
			for _, r := range a.Capabilities.Requires {
				resp.Capabilities.Requires = append(resp.Capabilities.Requires, CapabilityDTO{
					CapabilityID: r.CapabilityID,
					Verbs:        r.Verbs,
				})
			}
		}
	}

	if a.Connectors != nil {
		for _, c := range a.Connectors {
			resp.Connectors = append(resp.Connectors, ConnectorDTO{
				Type:    c.Type,
				ID:      c.ID,
				Enabled: c.Enabled,
			})
		}
	}

	if a.Heartbeat != nil {
		resp.Heartbeat = &HeartbeatDTO{
			IntervalSeconds: a.Heartbeat.IntervalSeconds,
			TimeoutSeconds:  a.Heartbeat.TimeoutSeconds,
			MaxMissedBeats:  a.Heartbeat.MaxMissedBeats,
		}
	}

	// Get skills from relations
	if h.relationStore != nil {
		rels, err := h.relationStore.ListAgentSkills(ctx, a.ID)
		if err == nil {
			skills := make([]string, 0, len(rels))
			for _, rel := range rels {
				if rel.Enabled {
					skills = append(skills, rel.SkillID)
				}
			}
			resp.Skills = skills
		}
	}

	if resp.Skills == nil {
		resp.Skills = []string{}
	}

	return resp
}
