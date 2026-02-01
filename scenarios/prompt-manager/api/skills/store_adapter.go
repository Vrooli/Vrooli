// Package skills provides the core domain types and operations for skill management.
//
// This file contains the adapter that bridges the new pack-based storage (store.FileSkillStore)
// to the legacy folder-based interface (skills.SkillStore) used by handlers.
//
// DOC: docs/concepts/STORE-MIGRATION.md
package skills

import (
	"context"
	"fmt"
	"path/filepath"

	"prompt-manager/store"
)

// StoreAdapter adapts store.SkillStore (pack-based) to skills.SkillStore (folder-based).
// This allows existing handlers to work unchanged while reading from the new storage.
type StoreAdapter struct {
	fileStore *store.FileSkillStore
}

// NewStoreAdapter creates a new store adapter.
func NewStoreAdapter(fileStore *store.FileSkillStore) *StoreAdapter {
	return &StoreAdapter{fileStore: fileStore}
}

// GetAll returns all skills from active packs, converting to the old Metadata format.
// Implements skills.SkillStore.GetAll()
func (a *StoreAdapter) GetAll() ([]Metadata, error) {
	ctx := context.Background()
	skills, err := a.fileStore.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Metadata, 0, len(skills))
	for _, s := range skills {
		result = append(result, a.toMetadata(s))
	}
	return result, nil
}

// FindByID searches all packs for a skill with the given ID.
// Returns the skill metadata and the pack (folder) it was found in.
// Implements skills.SkillStore.FindByID()
func (a *StoreAdapter) FindByID(id string) (*Metadata, string, error) {
	ctx := context.Background()
	skill, err := a.fileStore.Get(ctx, id)
	if err != nil {
		return nil, "", err
	}

	meta := a.toMetadata(*skill)
	return &meta, skill.Pack, nil
}

// LoadMetadata loads skills from a pack's skill directories.
// Implements skills.SkillStore.LoadMetadata()
func (a *StoreAdapter) LoadMetadata(folder string) ([]Metadata, error) {
	ctx := context.Background()
	skills, err := a.fileStore.List(ctx)
	if err != nil {
		return nil, err
	}

	var result []Metadata
	for _, s := range skills {
		if s.Pack == folder {
			result = append(result, a.toMetadata(s))
		}
	}
	return result, nil
}

// SaveMetadata saves skill metadata - delegates to individual skill updates.
// In the new storage model, each skill has its own skill.json.
// Implements skills.SkillStore.SaveMetadata()
func (a *StoreAdapter) SaveMetadata(folder string, skills []Metadata) error {
	ctx := context.Background()

	// Get existing skills in this pack for deletion detection
	existing, err := a.LoadMetadata(folder)
	if err != nil {
		return err
	}

	existingIDs := make(map[string]bool)
	for _, s := range existing {
		existingIDs[s.ID] = true
	}

	incomingIDs := make(map[string]bool)
	for _, s := range skills {
		incomingIDs[s.ID] = true
	}

	// Handle deletions - skills in existing but not in incoming
	for id := range existingIDs {
		if !incomingIDs[id] {
			if err := a.fileStore.Delete(ctx, id); err != nil {
				return fmt.Errorf("deleting skill %s: %w", id, err)
			}
		}
	}

	// Handle creates and updates
	for _, meta := range skills {
		skill := a.fromMetadata(meta, folder)

		if existingIDs[meta.ID] {
			// Update existing skill (metadata only, content handled separately)
			if err := a.fileStore.Update(ctx, meta.ID, &skill, nil); err != nil {
				return fmt.Errorf("updating skill %s: %w", meta.ID, err)
			}
		} else {
			// Create new skill - need content
			content, err := a.GetContent(folder, meta.File)
			if err != nil {
				content = "" // May not exist yet
			}
			if err := a.fileStore.Create(ctx, folder, &skill, content); err != nil {
				return fmt.Errorf("creating skill %s: %w", meta.ID, err)
			}
		}
	}

	return nil
}

// GetContent reads a skill's markdown content.
// In the new storage, content is at packs/{pack}/{id}/SKILL.md
// Implements skills.SkillStore.GetContent()
func (a *StoreAdapter) GetContent(folder, filename string) (string, error) {
	ctx := context.Background()

	// Extract skill ID from filename (remove .md extension)
	id := filename
	if ext := filepath.Ext(filename); ext == ".md" {
		id = filename[:len(filename)-len(ext)]
	}

	_, content, err := a.fileStore.GetWithContent(ctx, id)
	if err != nil {
		return "", err
	}
	return content, nil
}

// SaveContent writes a skill's markdown content.
// Implements skills.SkillStore.SaveContent()
func (a *StoreAdapter) SaveContent(folder, filename, content string) error {
	ctx := context.Background()

	// Extract skill ID from filename
	id := filename
	if ext := filepath.Ext(filename); ext == ".md" {
		id = filename[:len(filename)-len(ext)]
	}

	// Check if skill exists - if so, update; otherwise this is part of a create
	skill, err := a.fileStore.Get(ctx, id)
	if err != nil {
		// Skill doesn't exist yet - content will be saved during Create
		// Return nil to allow the create flow to continue
		return nil
	}

	// Update with new content
	return a.fileStore.Update(ctx, id, skill, &content)
}

// DeleteContent removes a skill's markdown file.
// In the new storage, this is a no-op as content is deleted with the skill directory.
// Implements skills.SkillStore.DeleteContent()
func (a *StoreAdapter) DeleteContent(folder, filename string) error {
	// Content is stored alongside skill.json - deleting the skill deletes the content
	// This is called after metadata is updated, so the skill may already be gone
	return nil
}

// GetVersions returns version history for a skill.
// Implements skills.SkillStore.GetVersions()
func (a *StoreAdapter) GetVersions(skillID string) ([]SkillVersion, error) {
	ctx := context.Background()
	entries, err := a.fileStore.GetVersionHistory(ctx, skillID)
	if err != nil {
		return nil, err
	}

	versions := make([]SkillVersion, 0, len(entries))
	for _, e := range entries {
		versions = append(versions, SkillVersion{
			Version:   e.Version,
			UpdatedAt: e.Timestamp,
			// Note: Content not stored in history.jsonl format
		})
	}
	return versions, nil
}

// SaveVersion saves the current state as a new version.
// In the new storage, history is appended to history.jsonl automatically on update.
// Implements skills.SkillStore.SaveVersion()
func (a *StoreAdapter) SaveVersion(skillID, folder string, skill *Metadata, content string) error {
	// History is managed automatically by the file store on updates
	// This is a compatibility shim
	return nil
}

// GetVersionContent returns the content of a specific version.
// Note: The new storage format doesn't store full content in history.
// Implements skills.SkillStore.GetVersionContent()
func (a *StoreAdapter) GetVersionContent(skillID string, version int) (*SkillVersion, error) {
	versions, err := a.GetVersions(skillID)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		if v.Version == version {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("version %d not found for skill: %s", version, skillID)
}

// LoadVersions loads all version files for a folder.
// Implements skills.SkillStore.LoadVersions()
func (a *StoreAdapter) LoadVersions(folder string) (map[string]*VersionFile, error) {
	// Get all skills in folder and load their versions
	skills, err := a.LoadMetadata(folder)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*VersionFile)
	for _, s := range skills {
		versions, err := a.GetVersions(s.ID)
		if err != nil {
			continue
		}
		if len(versions) > 0 {
			result[s.ID] = &VersionFile{
				SkillID:  s.ID,
				Versions: versions,
			}
		}
	}
	return result, nil
}

// SaveVersions saves version files for a folder.
// In the new storage, versions are per-skill in history.jsonl.
// Implements skills.SkillStore.SaveVersions()
func (a *StoreAdapter) SaveVersions(folder string, versions map[string]*VersionFile) error {
	// History is managed per-skill in the new format
	return nil
}

// toMetadata converts a store.Skill to skills.Metadata
func (a *StoreAdapter) toMetadata(s store.Skill) Metadata {
	// Map status to draft flag
	draft := s.Status == store.StatusDraft

	// Use entry as the file path, prefixed with pack
	file := s.ID + ".md"
	if s.Pack != "" {
		file = s.Pack + "/" + s.ID + ".md"
	}

	return Metadata{
		ID:           s.ID,
		File:         file,
		Name:         s.Name,
		Description:  s.Description,
		Modes:        s.Modes,
		Tags:         s.Tags,
		Icon:         s.Icon,
		TargetToolID: s.TargetToolID,
		Draft:        draft,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// fromMetadata converts skills.Metadata to store.Skill
func (a *StoreAdapter) fromMetadata(m Metadata, pack string) store.Skill {
	status := store.StatusActive
	if m.Draft {
		status = store.StatusDraft
	}

	return store.Skill{
		BaseEntity: store.BaseEntity{
			Kind:          store.KindSkill,
			SchemaVersion: store.CurrentSchemaVersion,
		},
		ID:           m.ID,
		Name:         m.Name,
		Description:  m.Description,
		Modes:        m.Modes,
		Tags:         m.Tags,
		Icon:         m.Icon,
		Status:       status,
		Entry:        "SKILL.md",
		TargetToolID: m.TargetToolID,
		Timestamps: store.Timestamps{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		Pack: pack,
	}
}
