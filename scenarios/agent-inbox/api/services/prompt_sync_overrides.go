// Package services provides business logic orchestration.
// This file handles skill override loading/saving and local skill management
// (import, export, and fallback CRUD for when prompt-manager is unavailable).
package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
)

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
			applyOverrideFields(&cfg.SkillOverrides[i], icon, targetToolID)
			found = true
			break
		}
	}

	if !found {
		override := SkillOverride{PromptID: skillID}
		applyOverrideFields(&override, icon, targetToolID)
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

// applyOverrideFields sets icon and targetToolID on a SkillOverride if non-empty.
func applyOverrideFields(override *SkillOverride, icon, targetToolID string) {
	if icon != "" {
		override.Icon = icon
	}
	if targetToolID != "" {
		override.TargetToolID = &targetToolID
	}
}

// promptToSkill converts a PromptResponse to a SkillResponse,
// applying any local overrides.
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

// createLocalSkill creates a skill in local in-memory storage (fallback).
func (s *PromptSyncService) createLocalSkill(sk *Skill) (*SkillResponse, error) {
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

	s.localSkills[sk.ID] = resp
	return resp, nil
}

// updateLocalSkill applies partial updates to a local skill.
func (s *PromptSyncService) updateLocalSkill(id string, updates *Skill) (*SkillResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.localSkills[id]
	if !exists {
		return nil, fmt.Errorf("skill not found: %s", id)
	}

	now := time.Now().UTC().Format(time.RFC3339)

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
