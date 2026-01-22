// Package services provides business logic orchestration.
// This file implements the prompt sync service that fetches skills from prompt-manager.
package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"agent-inbox/config"

	"github.com/google/uuid"
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

// SkillSource indicates where a skill came from.
type SkillSource string

const (
	SkillSourceDefault  SkillSource = "default"
	SkillSourceUser     SkillSource = "user"
	SkillSourceModified SkillSource = "modified" // User modified a default
)

// SkillResponse is a skill with additional metadata for API responses.
type SkillResponse struct {
	Skill
	Source     SkillSource `json:"source"`
	HasDefault bool        `json:"hasDefault"`
	CreatedAt  string      `json:"createdAt,omitempty"`
	UpdatedAt  string      `json:"updatedAt,omitempty"`
}

// SkillListResponse is the response for listing skills.
type SkillListResponse struct {
	Skills                []SkillResponse `json:"skills"`
	DefaultsCount         int             `json:"defaults_count"`
	UserCount             int             `json:"user_count"`
	ModifiedDefaultsCount int             `json:"modified_defaults_count"`
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
	Prompts     []PromptResponse `json:"prompts"`
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
}

// NewPromptSyncService creates a new prompt sync service.
func NewPromptSyncService(cfg *config.PromptSyncConfig, skillsCfg *config.SkillsConfig) *PromptSyncService {
	svc := &PromptSyncService{
		cfg:         cfg,
		skillsCfg:   skillsCfg,
		client: &http.Client{
			Timeout: cfg.SyncTimeout,
		},
		skills:      make(map[string]*SkillResponse),
		localSkills: make(map[string]*SkillResponse),
		overrides:   make(map[string]SkillOverride),
		stopChan:    make(chan struct{}),
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
func (s *PromptSyncService) Sync() error {
	if s.cfg.PromptManagerURL == "" {
		return fmt.Errorf("prompt-manager URL not available")
	}

	url := fmt.Sprintf("%s/api/v1/prompts/sync?tag=skill", s.cfg.PromptManagerURL)

	resp, err := s.client.Get(url)
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
	for _, p := range syncResp.Prompts {
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
		Source:     SkillSourceDefault, // All synced skills are treated as defaults
		HasDefault: true,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
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

// ListSkills returns all skills (synced + local).
func (s *PromptSyncService) ListSkills() (*SkillListResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SkillResponse, 0, len(s.skills)+len(s.localSkills))

	// Add synced skills
	for _, skill := range s.skills {
		result = append(result, *skill)
	}

	// Add local skills
	for _, skill := range s.localSkills {
		result = append(result, *skill)
	}

	return &SkillListResponse{
		Skills:                result,
		DefaultsCount:         len(s.skills),
		UserCount:             len(s.localSkills),
		ModifiedDefaultsCount: 0,
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

// CreateSkill creates a new local skill.
func (s *PromptSyncService) CreateSkill(sk *Skill) (*SkillResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not provided
	if sk.ID == "" {
		sk.ID = fmt.Sprintf("user-%d-%s", time.Now().UnixMilli(), uuid.New().String()[:8])
	}

	// Check if ID already exists
	if _, exists := s.skills[sk.ID]; exists {
		return nil, fmt.Errorf("skill with ID %s already exists (synced)", sk.ID)
	}
	if _, exists := s.localSkills[sk.ID]; exists {
		return nil, fmt.Errorf("skill with ID %s already exists (local)", sk.ID)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	resp := &SkillResponse{
		Skill:      *sk,
		Source:     SkillSourceUser,
		HasDefault: false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	s.localSkills[sk.ID] = resp

	return resp, nil
}

// UpdateSkill updates an existing local skill.
func (s *PromptSyncService) UpdateSkill(id string, updates *Skill) (*SkillResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if this is a synced skill
	if _, isSynced := s.skills[id]; isSynced {
		return nil, fmt.Errorf("cannot update synced skill %s - edit in prompt-manager instead", id)
	}

	// Check if local skill exists
	existing, exists := s.localSkills[id]
	if !exists {
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

// DeleteSkill deletes a local skill.
func (s *PromptSyncService) DeleteSkill(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if this is a synced skill
	if _, isSynced := s.skills[id]; isSynced {
		return fmt.Errorf("cannot delete synced skill %s - delete in prompt-manager instead", id)
	}

	// Check if local skill exists
	if _, exists := s.localSkills[id]; !exists {
		return fmt.Errorf("skill not found: %s", id)
	}

	delete(s.localSkills, id)
	return nil
}

// UpdateDefaultSkill is not supported for synced skills.
func (s *PromptSyncService) UpdateDefaultSkill(id string, updates *Skill) (*SkillResponse, error) {
	return nil, fmt.Errorf("cannot update default skill %s - edit in prompt-manager instead", id)
}

// ResetSkill refreshes a synced skill from prompt-manager.
func (s *PromptSyncService) ResetSkill(id string) (*SkillResponse, error) {
	s.mu.RLock()
	_, isSynced := s.skills[id]
	s.mu.RUnlock()

	if isSynced {
		// Force a sync to get latest
		if err := s.Sync(); err != nil {
			return nil, fmt.Errorf("failed to refresh from prompt-manager: %w", err)
		}
		return s.GetSkill(id)
	}

	return nil, fmt.Errorf("skill not found or not a synced skill: %s", id)
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
	if !s.cfg.Enabled {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/prompts/%s/use", s.cfg.PromptManagerURL, id)
	resp, err := s.client.Post(url, "application/json", nil)
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

