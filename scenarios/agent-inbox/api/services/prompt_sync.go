// Package services provides business logic orchestration.
// This file implements the core prompt sync service: construction, lifecycle
// management, sync logic, and read-only accessors.
package services

import (
	"agent-inbox/config"
	"agent-inbox/resilience"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

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
	if err := s.ensurePromptManagerURL(); err != nil {
		return err
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

// ListSkills returns all skills from prompt-manager.
func (s *PromptSyncService) ListSkills() (*SkillListResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SkillResponse, 0, len(s.skills))
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

	if skill, exists := s.skills[id]; exists {
		return skill, nil
	}
	if skill, exists := s.localSkills[id]; exists {
		return skill, nil
	}

	return nil, fmt.Errorf("skill not found: %s", id)
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
