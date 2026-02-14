// Package services provides business logic orchestration.
// This file implements the prompt sync service that fetches skills from prompt-manager.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"agent-inbox/config"
	"agent-inbox/resilience"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
)

// Skill represents a knowledge module that provides methodology and expertise.
// Skills are injected into the agent's context to enhance specific tasks.
type Skill struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Icon         string   `json:"icon,omitempty"`
	Modes        []string `json:"modes,omitempty"` // Hierarchical path like ["Architecture", "Audits"]
	Content      string   `json:"content"`
	Tags         []string `json:"tags,omitempty"`
	TargetToolID string   `json:"targetToolId,omitempty"` // Optional tool this skill targets
	Draft        bool     `json:"draft,omitempty"`        // Indicates skill may not be fully working
}

// SkillResponse is a skill with additional metadata for API responses.
type SkillResponse struct {
	Skill
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// SkillListResponse is the response for listing skills.
type SkillListResponse struct {
	Skills []SkillResponse `json:"skills"`
	Count  int             `json:"count"`
}

// PromptResponse is the response from prompt-manager for a single prompt.
type PromptResponse struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Content             string   `json:"content"`
	Modes               []string `json:"modes"`
	Tags                []string `json:"tags"`
	Icon                string   `json:"icon,omitempty"`
	TargetToolID        *string  `json:"targetToolId,omitempty"`
	Draft               bool     `json:"draft"`
	Folder              string   `json:"folder"`
	CreatedAt           string   `json:"createdAt"`
	UpdatedAt           string   `json:"updatedAt"`
	UsageCount          int      `json:"usageCount"`
	LastUsed            *string  `json:"lastUsed,omitempty"`
	EffectivenessRating *int     `json:"effectivenessRating,omitempty"`
}

// SyncResponse is the response from prompt-manager's sync endpoint.
type SyncResponse struct {
	Skills      []PromptResponse `json:"skills"`
	LastUpdated string           `json:"lastUpdated"`
	Hash        string           `json:"hash"`
}

// SkillOverride represents a local override for a prompt's skill properties.
type SkillOverride struct {
	PromptID     string  `json:"promptId"`
	Icon         string  `json:"icon,omitempty"`
	TargetToolID *string `json:"targetToolId,omitempty"`
}

// SkillsConfigFile represents the config/skills.json file structure.
type SkillsConfigFile struct {
	PromptManagerURL    string          `json:"promptManagerUrl"`
	SyncIntervalSeconds int             `json:"syncIntervalSeconds"`
	SkillOverrides      []SkillOverride `json:"skillOverrides"`
}

// PromptSyncService syncs skills from prompt-manager and provides read access.
type PromptSyncService struct {
	cfg          *config.PromptSyncConfig
	skillsCfg    *config.SkillsConfig
	client       *http.Client
	mu           sync.RWMutex
	skills       map[string]*SkillResponse // Synced skills from prompt-manager
	localSkills  map[string]*SkillResponse // Local user-created skills
	lastSyncHash string
	overrides    map[string]SkillOverride
	stopChan     chan struct{}
	retryCfg     resilience.RetryConfig
	cb           *resilience.CircuitBreaker
}

// NewPromptSyncService creates a new prompt sync service.
func NewPromptSyncService(cfg *config.PromptSyncConfig, skillsCfg *config.SkillsConfig) *PromptSyncService {
	appCfg := config.Default()
	retryCfg := resilience.RetryConfig{
		MaxAttempts: appCfg.Resilience.RetryAttempts,
		BaseDelay:   appCfg.Resilience.RetryBaseDelay,
		MaxDelay:    appCfg.Resilience.RetryMaxDelay,
		Jitter:      0.1,
	}
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		FailureThreshold: appCfg.Resilience.CircuitBreakerThreshold,
		Cooldown:         appCfg.Resilience.CircuitBreakerCooldown,
	})

	svc := &PromptSyncService{
		cfg:       cfg,
		skillsCfg: skillsCfg,
		client: &http.Client{
			Timeout: cfg.SyncTimeout,
		},
		skills:      make(map[string]*SkillResponse),
		localSkills: make(map[string]*SkillResponse),
		overrides:   make(map[string]SkillOverride),
		stopChan:    make(chan struct{}),
		retryCfg:    retryCfg,
		cb:          cb,
	}

	// Load overrides from config file
	svc.loadOverrides()

	return svc
}

// Start begins the background sync process.
func (s *PromptSyncService) Start() {
	if !s.cfg.Enabled {
		log.Println("Prompt sync disabled, local skills only")
		return
	}

	// Initial sync - don't fail if unavailable
	if s.cfg.PromptManagerURL == "" {
		log.Println("Prompt sync: prompt-manager not available, will retry in background")
	} else if err := s.Sync(); err != nil {
		log.Printf("Initial prompt sync failed: %v", err)
	}

	// Start background sync (will retry periodically)
	go s.backgroundSync()
}

// Stop stops the background sync process.
func (s *PromptSyncService) Stop() {
	close(s.stopChan)
}

// backgroundSync runs periodic sync in the background.
func (s *PromptSyncService) backgroundSync() {
	ticker := time.NewTicker(time.Duration(s.cfg.SyncIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.Sync(); err != nil {
				log.Printf("Background prompt sync failed: %v", err)
			}
		case <-s.stopChan:
			return
		}
	}
}

// Sync fetches prompts from prompt-manager and updates the local cache.
// On connection failure, re-resolves the prompt-manager URL via discovery
// to handle port drift after prompt-manager restarts.
func (s *PromptSyncService) Sync() error {
	if s.cfg.PromptManagerURL == "" {
		s.reResolveURL()
		if s.cfg.PromptManagerURL == "" {
			return fmt.Errorf("prompt-manager URL not available")
		}
	}

	resp, err := s.doHTTPWithRetry("GET", "/api/v1/skills/sync?tag=skill", nil)
	if err != nil {
		return fmt.Errorf("failed to fetch prompts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("prompt-manager returned status %d: %s", resp.StatusCode, string(body))
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return fmt.Errorf("failed to decode sync response: %w", err)
	}

	// Check if anything changed
	if syncResp.Hash == s.lastSyncHash {
		return nil // No changes
	}

	// Convert prompts to skills
	newSkills := make(map[string]*SkillResponse)
	for _, p := range syncResp.Skills {
		skill := s.promptToSkill(p)
		newSkills[skill.ID] = skill
	}

	// Update cache atomically
	s.mu.Lock()
	s.skills = newSkills
	s.lastSyncHash = syncResp.Hash
	s.mu.Unlock()

	log.Printf("Synced %d skills from prompt-manager", len(newSkills))
	return nil
}

// promptToSkill converts a PromptResponse to a SkillResponse.
func (s *PromptSyncService) promptToSkill(p PromptResponse) *SkillResponse {
	// Default values
	icon := p.Icon
	var targetToolID string

	// Apply overrides if present
	if override, exists := s.overrides[p.ID]; exists {
		if override.Icon != "" {
			icon = override.Icon
		}
		if override.TargetToolID != nil {
			targetToolID = *override.TargetToolID
		}
	}

	// Handle nil targetToolId from prompt-manager
	if p.TargetToolID != nil && targetToolID == "" {
		targetToolID = *p.TargetToolID
	}

	return &SkillResponse{
		Skill: Skill{
			ID:           p.ID,
			Name:         p.Name,
			Description:  p.Description,
			Icon:         icon,
			Modes:        p.Modes,
			Content:      p.Content,
			Tags:         p.Tags,
			TargetToolID: targetToolID,
			Draft:        p.Draft,
		},
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// loadOverrides loads skill overrides from the config file.
func (s *PromptSyncService) loadOverrides() {
	data, err := os.ReadFile(s.cfg.SkillOverridesPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Warning: could not read skill overrides file: %v", err)
		}
		return
	}

	var cfg SkillsConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("Warning: could not parse skill overrides file: %v", err)
		return
	}

	for _, override := range cfg.SkillOverrides {
		s.overrides[override.PromptID] = override
	}

	log.Printf("Loaded %d skill overrides", len(s.overrides))
}

// reResolveURL attempts to re-discover the prompt-manager URL via api-core discovery.
// Returns true if a new URL was found, false otherwise.
func (s *PromptSyncService) reResolveURL() bool {
	// Check env var override first
	if url := os.Getenv("PROMPT_MANAGER_URL"); url != "" {
		s.cfg.PromptManagerURL = url
		return true
	}

	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url, err := resolver.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		log.Printf("Prompt sync: re-resolution failed: %v", err)
		return false
	}

	if url != "" && url != s.cfg.PromptManagerURL {
		log.Printf("Prompt sync: re-resolved prompt-manager URL to %s", url)
		s.cfg.PromptManagerURL = url
		return true
	}
	return false
}

// doHTTPWithRetry performs an HTTP request with retry, circuit breaker, and URL re-resolution.
// On retry attempts > 1, it re-resolves the prompt-manager URL.
// 4xx responses are marked as permanent (non-retryable) errors.
func (s *PromptSyncService) doHTTPWithRetry(method, path string, body []byte) (*http.Response, error) {
	var resp *http.Response
	ctx := context.Background()

	err := resilience.Retry(ctx, s.retryCfg, func(ctx context.Context, attempt int) error {
		if attempt > 1 {
			s.reResolveURL()
		}

		if s.cfg.PromptManagerURL == "" {
			return fmt.Errorf("prompt-manager URL not available")
		}

		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}

		req, err := http.NewRequest(method, s.cfg.PromptManagerURL+path, reqBody)
		if err != nil {
			return resilience.Permanent(err)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		doReq := func(ctx context.Context) error {
			var doErr error
			resp, doErr = s.client.Do(req)
			return doErr
		}

		if s.cb != nil {
			err = s.cb.Execute(ctx, doReq)
		} else {
			err = doReq(ctx)
		}
		if err != nil {
			return err
		}

		// Mark 4xx as permanent
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return resilience.Permanent(fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)))
		}

		return nil
	})

	return resp, err
}

// ListSkills returns all skills from prompt-manager.
func (s *PromptSyncService) ListSkills() (*SkillListResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SkillResponse, 0, len(s.skills))

	// Add synced skills from prompt-manager
	for _, skill := range s.skills {
		result = append(result, *skill)
	}

	return &SkillListResponse{
		Skills: result,
		Count:  len(result),
	}, nil
}

// GetSkill returns a single skill by ID.
func (s *PromptSyncService) GetSkill(id string) (*SkillResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check synced skills first
	if skill, exists := s.skills[id]; exists {
		return skill, nil
	}

	// Check local skills
	if skill, exists := s.localSkills[id]; exists {
		return skill, nil
	}

	return nil, fmt.Errorf("skill not found: %s", id)
}

// CreateSkill creates a new skill via prompt-manager.
// Falls back to local storage only if prompt-manager is unavailable.
func (s *PromptSyncService) CreateSkill(sk *Skill) (*SkillResponse, error) {
	// Try to create in prompt-manager first
	req := &CreateSkillRequest{
		Name:         sk.Name,
		Description:  sk.Description,
		Content:      sk.Content,
		Modes:        sk.Modes,
		Tags:         sk.Tags,
		Icon:         sk.Icon,
		Draft:        sk.Draft,
		Folder:       "local",
		TargetToolID: sk.TargetToolID,
	}

	result, err := s.CreateSkillInPromptManager(req)
	if err == nil {
		return result, nil
	}

	// Fall back to local storage if prompt-manager unavailable
	log.Printf("Prompt-manager unavailable, creating skill locally: %v", err)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not provided
	if sk.ID == "" {
		sk.ID = fmt.Sprintf("user-%d-%s", time.Now().UnixMilli(), uuid.New().String()[:8])
	}

	// Check if ID already exists
	if _, exists := s.skills[sk.ID]; exists {
		return nil, fmt.Errorf("skill with ID %s already exists", sk.ID)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	resp := &SkillResponse{
		Skill:     *sk,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Note: localSkills is kept for fallback but skills now primarily come from prompt-manager
	s.localSkills[sk.ID] = resp

	return resp, nil
}

// UpdateSkill updates an existing skill via prompt-manager.
func (s *PromptSyncService) UpdateSkill(id string, updates *Skill) (*SkillResponse, error) {
	// Try to update via prompt-manager
	result, err := s.UpdateSkillInPromptManager(id, updates)
	if err == nil {
		return result, nil
	}

	// Log the error but continue - may be a local-only skill
	log.Printf("Could not update skill %s in prompt-manager: %v", id, err)

	// Check if this is a local skill that can be updated directly
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.localSkills[id]
	if !exists {
		// If not found locally and prompt-manager failed, return the error
		return nil, fmt.Errorf("skill not found: %s", id)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Apply updates
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.Icon != "" {
		existing.Icon = updates.Icon
	}
	if updates.Content != "" {
		existing.Content = updates.Content
	}
	if updates.Modes != nil {
		existing.Modes = updates.Modes
	}
	if updates.Tags != nil {
		existing.Tags = updates.Tags
	}
	if updates.TargetToolID != "" {
		existing.TargetToolID = updates.TargetToolID
	}
	existing.Draft = updates.Draft
	existing.UpdatedAt = now

	return existing, nil
}

// UpdateSkillInPromptManager sends skill updates to prompt-manager.
func (s *PromptSyncService) UpdateSkillInPromptManager(id string, updates *Skill) (*SkillResponse, error) {
	if !s.cfg.Enabled {
		return nil, fmt.Errorf("prompt-manager sync is not enabled")
	}
	if s.cfg.PromptManagerURL == "" {
		s.reResolveURL()
		if s.cfg.PromptManagerURL == "" {
			return nil, fmt.Errorf("prompt-manager URL not available")
		}
	}

	// Build the update payload
	pmReq := map[string]interface{}{}
	if updates.Name != "" {
		pmReq["name"] = updates.Name
	}
	if updates.Description != "" {
		pmReq["description"] = updates.Description
	}
	if updates.Content != "" {
		pmReq["content"] = updates.Content
	}
	if updates.Icon != "" {
		pmReq["icon"] = updates.Icon
	}
	if updates.Modes != nil {
		pmReq["modes"] = updates.Modes
	}
	if updates.Tags != nil {
		pmReq["tags"] = updates.Tags
	}
	pmReq["draft"] = updates.Draft

	body, err := json.Marshal(pmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	path := fmt.Sprintf("/api/v1/skills/%s", id)
	resp, err := s.doHTTPWithRetry(http.MethodPut, path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to update skill in prompt-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prompt-manager returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse the response
	var pmResp PromptResponse
	if err := json.NewDecoder(resp.Body).Decode(&pmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// If targetToolId is specified, save it as a local override
	if updates.TargetToolID != "" {
		if err := s.SaveOverride(id, updates.Icon, updates.TargetToolID); err != nil {
			log.Printf("Warning: failed to save override for skill %s: %v", id, err)
		}
	}

	// Trigger a sync to update our cache
	if err := s.Sync(); err != nil {
		log.Printf("Warning: sync after update failed: %v", err)
	}

	// Return the skill from our cache (with override applied)
	return s.GetSkill(id)
}

// DeleteSkill deletes a skill via prompt-manager.
func (s *PromptSyncService) DeleteSkill(id string) error {
	// Try to delete via prompt-manager
	err := s.DeleteSkillInPromptManager(id)
	if err == nil {
		return nil
	}

	// Log the error but continue - may be a local-only skill
	log.Printf("Could not delete skill %s from prompt-manager: %v", id, err)

	// Check if this is a local skill that can be deleted directly
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.localSkills[id]; !exists {
		// If not found locally and prompt-manager failed, return the error
		return fmt.Errorf("skill not found: %s", id)
	}

	delete(s.localSkills, id)
	return nil
}

// DeleteSkillInPromptManager deletes a skill from prompt-manager.
func (s *PromptSyncService) DeleteSkillInPromptManager(id string) error {
	if !s.cfg.Enabled {
		return fmt.Errorf("prompt-manager sync is not enabled")
	}
	if s.cfg.PromptManagerURL == "" {
		s.reResolveURL()
		if s.cfg.PromptManagerURL == "" {
			return fmt.Errorf("prompt-manager URL not available")
		}
	}

	path := fmt.Sprintf("/api/v1/skills/%s", id)
	resp, err := s.doHTTPWithRetry(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete skill from prompt-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("prompt-manager returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Trigger a sync to update our cache
	if err := s.Sync(); err != nil {
		log.Printf("Warning: sync after delete failed: %v", err)
	}

	return nil
}

// ImportSkills imports multiple local skills.
func (s *PromptSyncService) ImportSkills(skills []Skill) (int, error) {
	imported := 0
	for _, sk := range skills {
		skCopy := sk
		_, err := s.CreateSkill(&skCopy)
		if err != nil {
			// Skip duplicates, continue with others
			continue
		}
		imported++
	}
	return imported, nil
}

// ExportSkills exports all local skills.
func (s *PromptSyncService) ExportSkills() ([]Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Skill, 0, len(s.localSkills))
	for _, resp := range s.localSkills {
		result = append(result, resp.Skill)
	}
	return result, nil
}

// EnsureDirectories is a no-op since we use in-memory storage for local skills.
func (s *PromptSyncService) EnsureDirectories() error {
	return nil
}

// RecordUsage records usage of a skill by calling prompt-manager.
func (s *PromptSyncService) RecordUsage(id string) error {
	if !s.cfg.Enabled || s.cfg.PromptManagerURL == "" {
		return nil
	}

	path := fmt.Sprintf("/api/v1/skills/%s/use", id)
	resp, err := s.doHTTPWithRetry("POST", path, nil)
	if err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to record usage: status %d", resp.StatusCode)
	}

	return nil
}

// IsSyncEnabled returns whether prompt sync is enabled.
func (s *PromptSyncService) IsSyncEnabled() bool {
	return s.cfg.Enabled
}

// GetSyncStatus returns the current sync status.
func (s *PromptSyncService) GetSyncStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"enabled":      s.cfg.Enabled,
		"skillCount":   len(s.skills),
		"localCount":   len(s.localSkills),
		"lastSyncHash": s.lastSyncHash,
		"sourceUrl":    s.cfg.PromptManagerURL,
	}
}

// SyncStatus contains the result of a sync operation.
type SyncStatus struct {
	Success    bool   `json:"success"`
	SkillCount int    `json:"skillCount"`
	LocalCount int    `json:"localCount"`
	Hash       string `json:"hash"`
	Error      string `json:"error,omitempty"`
}

// TriggerSync forces an immediate sync and returns the status.
func (s *PromptSyncService) TriggerSync() (*SyncStatus, error) {
	if !s.cfg.Enabled {
		return &SyncStatus{
			Success:    false,
			SkillCount: 0,
			LocalCount: len(s.localSkills),
			Error:      "sync is disabled",
		}, nil
	}

	err := s.Sync()

	s.mu.RLock()
	defer s.mu.RUnlock()

	status := &SyncStatus{
		Success:    err == nil,
		SkillCount: len(s.skills),
		LocalCount: len(s.localSkills),
		Hash:       s.lastSyncHash,
	}

	if err != nil {
		status.Error = err.Error()
		return status, err
	}

	return status, nil
}

// CreateSkillRequest is the request to create a skill in prompt-manager.
type CreateSkillRequest struct {
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Content      string   `json:"content"`
	Modes        []string `json:"modes,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Icon         string   `json:"icon,omitempty"`
	Draft        bool     `json:"draft,omitempty"`
	Folder       string   `json:"folder"`
	TargetToolID string   `json:"-"` // Not sent to prompt-manager, stored locally
}

// CreateSkillInPromptManager creates a skill in prompt-manager and optionally stores a local override.
func (s *PromptSyncService) CreateSkillInPromptManager(req *CreateSkillRequest) (*SkillResponse, error) {
	if !s.cfg.Enabled {
		return nil, fmt.Errorf("prompt-manager sync is not enabled")
	}
	if s.cfg.PromptManagerURL == "" {
		s.reResolveURL()
		if s.cfg.PromptManagerURL == "" {
			return nil, fmt.Errorf("prompt-manager URL not available")
		}
	}

	// Prepare the request body for prompt-manager
	pmReq := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"content":     req.Content,
		"folder":      req.Folder,
	}
	if req.ID != "" {
		pmReq["id"] = req.ID
	}
	if req.Modes != nil {
		pmReq["modes"] = req.Modes
	}
	if req.Tags != nil {
		pmReq["tags"] = req.Tags
	}
	if req.Icon != "" {
		pmReq["icon"] = req.Icon
	}
	if req.Draft {
		pmReq["draft"] = req.Draft
	}

	body, err := json.Marshal(pmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.doHTTPWithRetry("POST", "/api/v1/skills", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create skill in prompt-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prompt-manager returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse the response
	var pmResp PromptResponse
	if err := json.NewDecoder(resp.Body).Decode(&pmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// If targetToolId is specified, save it as a local override
	if req.TargetToolID != "" {
		if err := s.SaveOverride(pmResp.ID, req.Icon, req.TargetToolID); err != nil {
			log.Printf("Warning: failed to save override for skill %s: %v", pmResp.ID, err)
		}
	}

	// Trigger a sync to get the new skill into our cache
	if err := s.Sync(); err != nil {
		log.Printf("Warning: sync after create failed: %v", err)
	}

	// Return the skill from our cache (with override applied)
	return s.GetSkill(pmResp.ID)
}

// SaveOverride saves or updates a skill override in the config file.
func (s *PromptSyncService) SaveOverride(skillID, icon, targetToolID string) error {
	// Read current config
	data, err := os.ReadFile(s.cfg.SkillOverridesPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg SkillsConfigFile
	if len(data) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Find or create override
	found := false
	for i := range cfg.SkillOverrides {
		if cfg.SkillOverrides[i].PromptID == skillID {
			if icon != "" {
				cfg.SkillOverrides[i].Icon = icon
			}
			if targetToolID != "" {
				cfg.SkillOverrides[i].TargetToolID = &targetToolID
			}
			found = true
			break
		}
	}

	if !found {
		override := SkillOverride{PromptID: skillID}
		if icon != "" {
			override.Icon = icon
		}
		if targetToolID != "" {
			override.TargetToolID = &targetToolID
		}
		cfg.SkillOverrides = append(cfg.SkillOverrides, override)
	}

	// Write back to file
	updatedData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(s.cfg.SkillOverridesPath, updatedData, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Reload overrides into memory
	s.loadOverrides()

	return nil
}
