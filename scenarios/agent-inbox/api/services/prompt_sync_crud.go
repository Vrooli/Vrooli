// Package services provides business logic orchestration.
// This file implements CRUD operations for skills, delegating to prompt-manager
// with fallback to local storage when prompt-manager is unavailable.
package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// CreateSkill creates a new skill via prompt-manager.
// Falls back to local storage only if prompt-manager is unavailable.
func (s *PromptSyncService) CreateSkill(sk *Skill) (*SkillResponse, error) {
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
	return s.createLocalSkill(sk)
}

// UpdateSkill updates an existing skill via prompt-manager.
func (s *PromptSyncService) UpdateSkill(id string, updates *Skill) (*SkillResponse, error) {
	result, err := s.UpdateSkillInPromptManager(id, updates)
	if err == nil {
		return result, nil
	}

	log.Printf("Could not update skill %s in prompt-manager: %v", id, err)
	return s.updateLocalSkill(id, updates)
}

// DeleteSkill deletes a skill via prompt-manager.
func (s *PromptSyncService) DeleteSkill(id string) error {
	err := s.DeleteSkillInPromptManager(id)
	if err == nil {
		return nil
	}

	log.Printf("Could not delete skill %s from prompt-manager: %v", id, err)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.localSkills[id]; !exists {
		return fmt.Errorf("skill not found: %s", id)
	}

	delete(s.localSkills, id)
	return nil
}

// CreateSkillInPromptManager creates a skill in prompt-manager and optionally stores a local override.
func (s *PromptSyncService) CreateSkillInPromptManager(req *CreateSkillRequest) (*SkillResponse, error) {
	if err := s.ensureEnabledAndReachable(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(buildCreatePayload(req))
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

	return s.GetSkill(pmResp.ID)
}

// buildCreatePayload constructs the JSON payload map for creating a skill.
func buildCreatePayload(req *CreateSkillRequest) map[string]interface{} {
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
	return pmReq
}

// UpdateSkillInPromptManager sends skill updates to prompt-manager.
func (s *PromptSyncService) UpdateSkillInPromptManager(id string, updates *Skill) (*SkillResponse, error) {
	if err := s.ensureEnabledAndReachable(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(buildUpdatePayload(updates))
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

	if err := s.Sync(); err != nil {
		log.Printf("Warning: sync after update failed: %v", err)
	}

	return s.GetSkill(id)
}

// buildUpdatePayload constructs the JSON payload map for updating a skill.
func buildUpdatePayload(updates *Skill) map[string]interface{} {
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
	return pmReq
}

// DeleteSkillInPromptManager deletes a skill from prompt-manager.
func (s *PromptSyncService) DeleteSkillInPromptManager(id string) error {
	if err := s.ensureEnabledAndReachable(); err != nil {
		return err
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

	if err := s.Sync(); err != nil {
		log.Printf("Warning: sync after delete failed: %v", err)
	}

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
