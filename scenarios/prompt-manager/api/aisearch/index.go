package aisearch

import (
	"context"
	"fmt"
	"log"
	"strings"

	"prompt-manager/skills"
)

// IndexSkill indexes a single skill into the vector store.
// This is designed to be called asynchronously after skill CRUD operations.
func (s *Service) IndexSkill(ctx context.Context, skillID string) error {
	skill, folder, err := s.skillStore.FindByID(skillID)
	if err != nil {
		return fmt.Errorf("skill not found: %w", err)
	}

	// Load content
	_, filename := extractFolderAndFile(skill.File)
	content, err := s.skillStore.GetContent(folder, filename)
	if err != nil {
		return fmt.Errorf("failed to load content: %w", err)
	}

	// Compose text for embedding
	embeddingText := composeEmbeddingText(skill, content)

	// Generate embedding
	vector, err := s.embedder.Embed(ctx, embeddingText)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	// Build payload
	payload := map[string]interface{}{
		"skill_id":    skill.ID,
		"name":        skill.Name,
		"description": skill.Description,
		"folder":      folder,
		"tags":        skill.Tags,
		"modes":       skill.Modes,
	}

	// Upsert into vector store
	if err := s.vectorStore.Upsert(ctx, skill.ID, vector, payload); err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}

	log.Printf("[aisearch] Indexed skill: %s (%s)", skill.Name, skill.ID)
	return nil
}

// DeleteFromIndex removes a skill from the vector index.
func (s *Service) DeleteFromIndex(ctx context.Context, skillID string) error {
	if err := s.vectorStore.Delete(ctx, skillID); err != nil {
		return fmt.Errorf("delete from index failed: %w", err)
	}
	log.Printf("[aisearch] Deleted skill from index: %s", skillID)
	return nil
}

// ReindexAll rebuilds the entire vector index from all skills.
func (s *Service) ReindexAll(ctx context.Context) (*ReindexResponse, error) {
	// Ensure collection exists
	if err := s.vectorStore.EnsureCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	// Get all skills
	allSkills, err := s.skillStore.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}

	var indexed, skipped, errors int

	for _, skill := range allSkills {
		// Extract folder from file path
		folder, filename := extractFolderAndFile(skill.File)
		if folder == "" {
			skipped++
			continue
		}

		// Load content
		content, err := s.skillStore.GetContent(folder, filename)
		if err != nil {
			log.Printf("[aisearch] Failed to load content for %s: %v", skill.ID, err)
			errors++
			continue
		}

		// Compose text for embedding
		embeddingText := composeEmbeddingText(&skill, content)

		// Generate embedding
		vector, err := s.embedder.Embed(ctx, embeddingText)
		if err != nil {
			log.Printf("[aisearch] Failed to embed %s: %v", skill.ID, err)
			errors++
			continue
		}

		// Build payload
		payload := map[string]interface{}{
			"skill_id":    skill.ID,
			"name":        skill.Name,
			"description": skill.Description,
			"folder":      folder,
			"tags":        skill.Tags,
			"modes":       skill.Modes,
		}

		// Upsert into vector store
		if err := s.vectorStore.Upsert(ctx, skill.ID, vector, payload); err != nil {
			log.Printf("[aisearch] Failed to upsert %s: %v", skill.ID, err)
			errors++
			continue
		}

		indexed++
	}

	return &ReindexResponse{
		Indexed: indexed,
		Skipped: skipped,
		Errors:  errors,
		Message: fmt.Sprintf("Indexed %d skills, skipped %d, errors %d", indexed, skipped, errors),
	}, nil
}

// composeEmbeddingText creates a rich text representation for embedding.
func composeEmbeddingText(skill *skills.Metadata, content string) string {
	var parts []string

	// Name first for high relevance
	parts = append(parts, skill.Name)

	// Description
	if skill.Description != "" {
		parts = append(parts, skill.Description)
	}

	// Tags
	if len(skill.Tags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(skill.Tags, ", "))
	}

	// Modes (categories)
	if len(skill.Modes) > 0 {
		parts = append(parts, "Categories: "+strings.Join(skill.Modes, " / "))
	}

	// Content (truncated to avoid token limits)
	if content != "" {
		truncated := content
		if len(truncated) > 2000 {
			truncated = truncated[:2000] + "..."
		}
		parts = append(parts, truncated)
	}

	return strings.Join(parts, "\n\n")
}

// extractFolderAndFile splits a file path like "local/skill.md" into folder and filename.
func extractFolderAndFile(file string) (folder, filename string) {
	parts := strings.SplitN(file, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", file
}
