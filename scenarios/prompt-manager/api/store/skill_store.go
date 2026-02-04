package store

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// skillIDRegex validates skill IDs: lowercase letters, numbers, hyphens.
// Must start with a letter, not end with a hyphen.
var skillIDRegex = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// isValidSkillID validates that a skill ID follows naming conventions.
func isValidSkillID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	return skillIDRegex.MatchString(id)
}

// FileSkillStore implements SkillStore using the file system
type FileSkillStore struct {
	storeDir string
}

// NewFileSkillStore creates a new file-based skill store
func NewFileSkillStore(storeDir string) *FileSkillStore {
	return &FileSkillStore{storeDir: storeDir}
}

// skillsDir returns the path to the skills directory
func (s *FileSkillStore) skillsDir() string {
	return filepath.Join(s.storeDir, "skills")
}

// packsDir returns the path to the packs directory
func (s *FileSkillStore) packsDir() string {
	return filepath.Join(s.skillsDir(), "packs")
}

// getPackOrder loads the pack order configuration
func (s *FileSkillStore) getPackOrder() (*PackOrder, error) {
	path := filepath.Join(s.skillsDir(), "_pack-order.json")
	return LoadJSON[PackOrder](path)
}

// getActivePacks returns the list of active packs in precedence order
func (s *FileSkillStore) getActivePacks() ([]string, error) {
	order, err := s.getPackOrder()
	if err != nil {
		// Default to checking all packs if config missing
		return ListDirectories(s.packsDir())
	}
	return order.ActivePacks, nil
}

// List returns all skills from active packs
func (s *FileSkillStore) List(ctx context.Context) ([]Skill, error) {
	packs, err := s.getActivePacks()
	if err != nil {
		return nil, fmt.Errorf("getting active packs: %w", err)
	}

	seen := make(map[string]bool)
	var skills []Skill

	// Process packs in precedence order - first pack wins for same ID
	for _, pack := range packs {
		packPath := filepath.Join(s.packsDir(), pack)
		skillDirs, err := ListDirectories(packPath)
		if err != nil {
			continue // Skip inaccessible packs
		}

		for _, skillID := range skillDirs {
			if seen[skillID] {
				continue // Already have this skill from higher precedence pack
			}

			skill, err := s.loadSkill(pack, skillID)
			if err != nil {
				continue // Skip malformed skills
			}

			skill.Pack = pack
			skills = append(skills, *skill)
			seen[skillID] = true
		}
	}

	return skills, nil
}

// Get retrieves a skill by ID, searching through active packs
func (s *FileSkillStore) Get(ctx context.Context, id string) (*Skill, error) {
	packs, err := s.getActivePacks()
	if err != nil {
		return nil, fmt.Errorf("getting active packs: %w", err)
	}

	for _, pack := range packs {
		skill, err := s.loadSkill(pack, id)
		if err == nil {
			skill.Pack = pack
			return skill, nil
		}
	}

	return nil, fmt.Errorf("skill not found: %s", id)
}

// GetWithContent retrieves a skill with its content
func (s *FileSkillStore) GetWithContent(ctx context.Context, id string) (*Skill, string, error) {
	skill, err := s.Get(ctx, id)
	if err != nil {
		return nil, "", err
	}

	contentPath := filepath.Join(s.packsDir(), skill.Pack, id, skill.Entry)
	content, err := ReadContent(contentPath)
	if err != nil {
		return skill, "", fmt.Errorf("reading content: %w", err)
	}

	return skill, content, nil
}

// Create creates a new skill in the specified pack
func (s *FileSkillStore) Create(ctx context.Context, pack string, skill *Skill, content string) error {
	// Validate pack is active
	packs, err := s.getActivePacks()
	if err != nil {
		return fmt.Errorf("getting active packs: %w", err)
	}

	packValid := false
	for _, p := range packs {
		if p == pack {
			packValid = true
			break
		}
	}
	if !packValid {
		return fmt.Errorf("invalid pack: %s", pack)
	}

	// Check skill doesn't already exist
	if _, err := s.Get(ctx, skill.ID); err == nil {
		return fmt.Errorf("skill already exists: %s", skill.ID)
	}

	// Set up skill entity
	skill.Kind = KindSkill
	skill.SchemaVersion = CurrentSchemaVersion
	skill.Entry = "SKILL.md"
	skill.Timestamps = NewTimestamps()
	skill.Pack = pack

	skillDir := filepath.Join(s.packsDir(), pack, skill.ID)

	// Create skill directory and files
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}

	// Write skill.json
	if err := SaveJSON(filepath.Join(skillDir, "skill.json"), skill); err != nil {
		return fmt.Errorf("writing skill.json: %w", err)
	}

	// Write content
	if err := WriteContent(filepath.Join(skillDir, "SKILL.md"), content); err != nil {
		return fmt.Errorf("writing content: %w", err)
	}

	// Create initial history entry
	entry := HistoryEntry{
		Version:   1,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    "created",
		Summary:   "Initial version",
	}
	if err := AppendJSONL(filepath.Join(skillDir, "history.jsonl"), entry); err != nil {
		return fmt.Errorf("writing history: %w", err)
	}

	return nil
}

// Update updates an existing skill
func (s *FileSkillStore) Update(ctx context.Context, id string, updates *Skill, content *string) error {
	skill, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	skillDir := filepath.Join(s.packsDir(), skill.Pack, id)

	// Apply updates
	if updates.Name != "" {
		skill.Name = updates.Name
	}
	if updates.Description != "" {
		skill.Description = updates.Description
	}
	if len(updates.Modes) > 0 {
		skill.Modes = updates.Modes
	}
	if len(updates.Tags) > 0 {
		skill.Tags = updates.Tags
	}
	if updates.Icon != "" {
		skill.Icon = updates.Icon
	}
	if updates.Status != "" {
		skill.Status = updates.Status
	}
	if updates.TargetToolID != nil {
		skill.TargetToolID = updates.TargetToolID
	}
	if updates.Requires != nil {
		skill.Requires = updates.Requires
	}
	// DefaultScope is always applied (empty string clears it)
	skill.DefaultScope = updates.DefaultScope

	skill.UpdateTimestamp()

	// Write updated skill.json
	if err := SaveJSON(filepath.Join(skillDir, "skill.json"), skill); err != nil {
		return fmt.Errorf("writing skill.json: %w", err)
	}

	// Write content if provided
	if content != nil {
		if err := WriteContent(filepath.Join(skillDir, "SKILL.md"), *content); err != nil {
			return fmt.Errorf("writing content: %w", err)
		}
	}

	// Add history entry
	entry := HistoryEntry{
		Version:   skill.Revision,
		Timestamp: skill.UpdatedAt,
		Action:    "updated",
		Summary:   "Updated skill",
	}
	if err := AppendJSONL(filepath.Join(skillDir, "history.jsonl"), entry); err != nil {
		return fmt.Errorf("writing history: %w", err)
	}

	return nil
}

// Delete removes a skill
func (s *FileSkillStore) Delete(ctx context.Context, id string) error {
	skill, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	skillDir := filepath.Join(s.packsDir(), skill.Pack, id)
	return DeleteDirectory(skillDir)
}

// GetVersionHistory returns version history for a skill
func (s *FileSkillStore) GetVersionHistory(ctx context.Context, id string) ([]HistoryEntry, error) {
	skill, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	historyPath := filepath.Join(s.packsDir(), skill.Pack, id, "history.jsonl")

	file, err := os.Open(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, fmt.Errorf("opening history: %w", err)
	}
	defer file.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed entries
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading history: %w", err)
	}

	return entries, nil
}

// loadSkill loads a skill from a pack directory
func (s *FileSkillStore) loadSkill(pack, skillID string) (*Skill, error) {
	skillPath := filepath.Join(s.packsDir(), pack, skillID, "skill.json")
	return LoadJSON[Skill](skillPath)
}

// ContentPath returns the path where a skill's content would be stored.
// This is useful for pre-writing content during move operations.
func (s *FileSkillStore) ContentPath(pack, skillID string) string {
	return filepath.Join(s.packsDir(), pack, skillID, "SKILL.md")
}

// Rename renames a skill by changing its directory name and updating skill.json.
// This is an atomic operation: if any step fails, the original state is preserved.
// Implements store.SkillStore.Rename()
func (s *FileSkillStore) Rename(ctx context.Context, oldID, newID string) (*Skill, error) {
	// Validate new ID format
	if !isValidSkillID(newID) {
		return nil, fmt.Errorf("invalid skill ID format: %s", newID)
	}

	// Get the existing skill to find its pack
	skill, err := s.Get(ctx, oldID)
	if err != nil {
		return nil, err
	}

	// Check new ID doesn't already exist
	if _, err := s.Get(ctx, newID); err == nil {
		return nil, fmt.Errorf("skill already exists: %s", newID)
	}

	oldDir := filepath.Join(s.packsDir(), skill.Pack, oldID)
	newDir := filepath.Join(s.packsDir(), skill.Pack, newID)

	// Rename the directory
	if err := os.Rename(oldDir, newDir); err != nil {
		return nil, fmt.Errorf("renaming skill directory: %w", err)
	}

	// Update skill.json with new ID
	skill.ID = newID
	skill.UpdateTimestamp()

	skillJSONPath := filepath.Join(newDir, "skill.json")
	if err := SaveJSON(skillJSONPath, skill); err != nil {
		// Rollback: rename directory back
		_ = os.Rename(newDir, oldDir)
		return nil, fmt.Errorf("updating skill.json: %w", err)
	}

	// Add history entry
	entry := HistoryEntry{
		Version:   skill.Revision,
		Timestamp: skill.UpdatedAt,
		Action:    "renamed",
		Summary:   fmt.Sprintf("Renamed from %s to %s", oldID, newID),
	}
	if err := AppendJSONL(filepath.Join(newDir, "history.jsonl"), entry); err != nil {
		// Non-fatal: history is supplementary
		fmt.Printf("[skill_store] Warning: failed to write history for rename: %v\n", err)
	}

	return skill, nil
}
