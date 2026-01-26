package aisearch

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
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
	if err := s.vectorStore.Upsert(ctx, qdrantPointID(skill.ID), vector, payload); err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}

	log.Printf("[aisearch] Indexed skill: %s (%s)", skill.Name, skill.ID)
	return nil
}

// DeleteFromIndex removes a skill from the vector index.
func (s *Service) DeleteFromIndex(ctx context.Context, skillID string) error {
	if err := s.vectorStore.Delete(ctx, qdrantPointID(skillID)); err != nil {
		return fmt.Errorf("delete from index failed: %w", err)
	}
	log.Printf("[aisearch] Deleted skill from index: %s", skillID)
	return nil
}

// ReindexAll rebuilds the entire vector index from all skills.
func (s *Service) ReindexAll(ctx context.Context) (*ReindexResponse, error) {
	return s.reindexAllWithProgress(ctx, nil, nil)
}

func (s *Service) reindexAllWithProgress(
	ctx context.Context,
	progress func(indexed, skipped, errors int),
	setTotal func(total int),
) (*ReindexResponse, error) {
	// Ensure collection exists
	if err := s.vectorStore.EnsureCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	// Get all skills
	allSkills, err := s.skillStore.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}
	if setTotal != nil {
		setTotal(len(allSkills))
	}

	var indexed, skipped, errors int

	for _, skill := range allSkills {
		if err := ctx.Err(); err != nil {
			return &ReindexResponse{
				Indexed: indexed,
				Skipped: skipped,
				Errors:  errors,
				Message: fmt.Sprintf("Indexed %d skills, skipped %d, errors %d", indexed, skipped, errors),
			}, err
		}

		// Extract folder from file path
		folder, filename := extractFolderAndFile(skill.File)
		if folder == "" {
			skipped++
			if progress != nil {
				progress(indexed, skipped, errors)
			}
			continue
		}

		// Load content
		content, err := s.skillStore.GetContent(folder, filename)
		if err != nil {
			log.Printf("[aisearch] Failed to load content for %s: %v", skill.ID, err)
			errors++
			if progress != nil {
				progress(indexed, skipped, errors)
			}
			continue
		}

		// Compose text for embedding
		embeddingText := composeEmbeddingText(&skill, content)

		// Generate embedding
		vector, err := s.embedder.Embed(ctx, embeddingText)
		if err != nil {
			log.Printf("[aisearch] Failed to embed %s: %v", skill.ID, err)
			errors++
			if progress != nil {
				progress(indexed, skipped, errors)
			}
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
		if err := s.vectorStore.Upsert(ctx, qdrantPointID(skill.ID), vector, payload); err != nil {
			log.Printf("[aisearch] Failed to upsert %s: %v", skill.ID, err)
			errors++
			if progress != nil {
				progress(indexed, skipped, errors)
			}
			continue
		}

		indexed++
		if progress != nil {
			progress(indexed, skipped, errors)
		}
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

var qdrantNamespace = [16]byte{
	0x6b, 0xa7, 0xb8, 0x10,
	0x9d, 0xad, 0x11, 0xd1,
	0x80, 0xb4, 0x00, 0xc0,
	0x4f, 0xd4, 0x30, 0xc8,
}

func qdrantPointID(skillID string) string {
	name := strings.TrimSpace(skillID)
	if name == "" {
		name = "unknown"
	}
	return uuidV5(qdrantNamespace, "prompt-manager:"+name)
}

func uuidV5(namespace [16]byte, name string) string {
	hash := sha1.New()
	_, _ = hash.Write(namespace[:])
	_, _ = hash.Write([]byte(name))
	sum := hash.Sum(nil)

	var uuid [16]byte
	copy(uuid[:], sum[:16])

	// Set version (5) and RFC4122 variant bits.
	uuid[6] = (uuid[6] & 0x0f) | 0x50
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	hexStr := hex.EncodeToString(uuid[:])
	return hexStr[0:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16] + "-" + hexStr[16:20] + "-" + hexStr[20:32]
}
