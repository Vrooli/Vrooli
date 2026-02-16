package agents

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"prompt-manager/store"
	"prompt-manager/validation"

	"github.com/gorilla/mux"
)

// GraphInvalidator allows triggering graph index invalidation.
type GraphInvalidator interface {
	Invalidate()
}

// AIAgentIndexer defines the interface for AI search index operations on agents.
type AIAgentIndexer interface {
	IndexAgent(ctx context.Context, agentID string) error
	DeleteAgentFromIndex(ctx context.Context, agentID string) error
}

// Handlers provides HTTP handlers for agent operations.
type Handlers struct {
	agentStore       store.AgentStore
	indexStore       store.IndexStore
	relationStore    store.RelationStore
	teamStore        store.TeamStore
	graphInvalidator GraphInvalidator
	aiIndexer        AIAgentIndexer
	storeDir         string
}

// NewHandlers creates a new agents handler.
func NewHandlers(agentStore store.AgentStore, indexStore store.IndexStore, storeDir string, relationStore store.RelationStore, teamStore store.TeamStore) *Handlers {
	return &Handlers{
		agentStore:    agentStore,
		indexStore:    indexStore,
		relationStore: relationStore,
		teamStore:     teamStore,
		storeDir:      storeDir,
	}
}

// SetGraphInvalidator sets the graph invalidator.
func (h *Handlers) SetGraphInvalidator(inv GraphInvalidator) {
	h.graphInvalidator = inv
}

// SetAIIndexer sets the AI search indexer for async index updates.
func (h *Handlers) SetAIIndexer(indexer AIAgentIndexer) {
	h.aiIndexer = indexer
}

func (h *Handlers) invalidateGraph() {
	if h.graphInvalidator != nil {
		h.graphInvalidator.Invalidate()
	}
}

func (h *Handlers) asyncIndexAgent(ctx context.Context, agentID string) {
	if h.aiIndexer == nil {
		return
	}
	go func() {
		if err := h.aiIndexer.IndexAgent(ctx, agentID); err != nil {
			log.Printf("[agents] AI index update failed for %s: %v", agentID, err)
		}
	}()
}

func (h *Handlers) asyncDeleteAgentFromIndex(ctx context.Context, agentID string) {
	if h.aiIndexer == nil {
		return
	}
	go func() {
		if err := h.aiIndexer.DeleteAgentFromIndex(ctx, agentID); err != nil {
			log.Printf("[agents] AI index delete failed for %s: %v", agentID, err)
		}
	}()
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
	if req.FileOrder != nil {
		normalized, err := normalizeFileOrder(req.FileOrder)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		agent.FileOrder = normalized
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

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateAgents(ctx)
	}
	h.invalidateGraph()
	h.asyncIndexAgent(context.Background(), agent.ID)

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
	if req.FileOrder != nil {
		normalized, err := normalizeFileOrder(req.FileOrder)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		updates.FileOrder = normalized
	}

	if err := h.agentStore.Update(ctx, id, updates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateAgents(ctx)
	}
	h.asyncIndexAgent(context.Background(), id)

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

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateAgents(ctx)
	}
	h.invalidateGraph()
	h.asyncDeleteAgentFromIndex(context.Background(), id)

	w.WriteHeader(http.StatusNoContent)
}

// ListTeams handles GET /agents/{id}/teams - returns team memberships for an agent.
func (h *Handlers) ListTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	// Verify agent exists
	if _, err := h.agentStore.Get(ctx, id); err != nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	relations, err := h.relationStore.ListAgentTeams(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	memberships := make([]AgentTeamMembershipDTO, 0, len(relations))
	for _, rel := range relations {
		displayName := rel.TeamID
		if h.teamStore != nil {
			if team, err := h.teamStore.Get(ctx, rel.TeamID); err == nil {
				displayName = team.DisplayName
			}
		}
		roles := rel.Roles
		if roles == nil {
			roles = []string{}
		}
		memberships = append(memberships, AgentTeamMembershipDTO{
			TeamID:          rel.TeamID,
			TeamDisplayName: displayName,
			Roles:           roles,
			Status:          rel.Status,
		})
	}

	resp := AgentTeamsResponse{
		AgentID:     id,
		Memberships: memberships,
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
	h.invalidateGraph()

	resp := SoulResponse{
		AgentID: id,
		Content: req.Content,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ListFiles handles GET /agents/{id}/files - lists files in agent folder.
func (h *Handlers) ListFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	fileStore, ok := h.agentStore.(*store.FileAgentStore)
	if !ok {
		http.Error(w, "ListFiles not supported", http.StatusInternalServerError)
		return
	}

	files, err := fileStore.ListFiles(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mapped := make([]AgentFileEntry, 0, len(files))
	for _, entry := range files {
		mapped = append(mapped, AgentFileEntry{
			Path:  entry.Path,
			IsDir: entry.IsDir,
			Size:  entry.Size,
		})
	}

	resp := AgentFileListResponse{
		AgentID: id,
		Files:   mapped,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetFile handles GET /agents/{id}/files/content - returns file content.
func (h *Handlers) GetFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	fileStore, ok := h.agentStore.(*store.FileAgentStore)
	if !ok {
		http.Error(w, "GetFile not supported", http.StatusInternalServerError)
		return
	}

	content, err := fileStore.ReadFile(ctx, id, path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "invalid path") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := AgentFileContentResponse{
		AgentID: id,
		Path:    path,
		Content: content,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SetFile handles PUT /agents/{id}/files/content - writes file content.
func (h *Handlers) SetFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	var req AgentFileWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileStore, ok := h.agentStore.(*store.FileAgentStore)
	if !ok {
		http.Error(w, "SetFile not supported", http.StatusInternalServerError)
		return
	}

	if err := fileStore.WriteFile(ctx, id, path, req.Content); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.invalidateGraph()

	w.WriteHeader(http.StatusNoContent)
}

// CreateFile handles POST /agents/{id}/files - creates a new file or directory.
func (h *Handlers) CreateFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	var req AgentFileCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	fileStore, ok := h.agentStore.(*store.FileAgentStore)
	if !ok {
		http.Error(w, "CreateFile not supported", http.StatusInternalServerError)
		return
	}

	if err := fileStore.CreateFile(ctx, id, req.Path, req.Content, req.IsDir); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.invalidateGraph()

	w.WriteHeader(http.StatusCreated)
}

// RenameFile handles POST /agents/{id}/files/rename - renames a file.
func (h *Handlers) RenameFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	var req AgentFileRenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.From == "" || req.To == "" {
		http.Error(w, "from and to are required", http.StatusBadRequest)
		return
	}

	fileStore, ok := h.agentStore.(*store.FileAgentStore)
	if !ok {
		http.Error(w, "RenameFile not supported", http.StatusInternalServerError)
		return
	}

	if err := fileStore.RenameFile(ctx, id, req.From, req.To); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.invalidateGraph()

	w.WriteHeader(http.StatusNoContent)
}

// DeleteFile handles DELETE /agents/{id}/files - deletes a file or directory.
func (h *Handlers) DeleteFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	fileStore, ok := h.agentStore.(*store.FileAgentStore)
	if !ok {
		http.Error(w, "DeleteFile not supported", http.StatusInternalServerError)
		return
	}

	if err := fileStore.DeleteFile(ctx, id, path); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.invalidateGraph()

	w.WriteHeader(http.StatusNoContent)
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
		FileOrder:         a.FileOrder,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
	}

	if h.storeDir != "" {
		resp.AgentDir = filepath.Join(h.storeDir, "agents", a.ID)
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

	return resp
}

func normalizeFileOrder(fileOrder []string) ([]string, error) {
	if len(fileOrder) == 0 {
		return []string{}, nil
	}

	normalized := make([]string, 0, len(fileOrder))
	seen := make(map[string]struct{}, len(fileOrder))

	for _, entry := range fileOrder {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			return nil, errors.New("fileOrder entries must be non-empty")
		}

		clean := filepath.Clean(filepath.FromSlash(trimmed))
		if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
			return nil, errors.New("fileOrder entries must be relative file paths")
		}
		if !strings.HasSuffix(strings.ToLower(clean), ".md") {
			return nil, errors.New("fileOrder entries must point to .md files")
		}

		key := strings.ToLower(clean)
		if _, exists := seen[key]; exists {
			return nil, errors.New("fileOrder entries must be unique")
		}
		seen[key] = struct{}{}

		normalized = append(normalized, filepath.ToSlash(clean))
	}

	return normalized, nil
}
