package store

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
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
	configDir        string
	roots            *filerouting.RoutedRoots
	scenarioRoots    []string
	scenarioCacheDir string
}

// NewFileSkillStore creates a new file-based skill store
func NewFileSkillStore(configDir string) *FileSkillStore {
	return &FileSkillStore{configDir: configDir}
}

// NewFileSkillStoreWithScenarioRoots indexes scenario-owned skills in addition
// to prompt-manager's governed packs. The scenario directories remain the
// source of truth; the registry only synthesizes read metadata.
func NewFileSkillStoreWithScenarioRoots(configDir string, scenarioRoots ...string) *FileSkillStore {
	return &FileSkillStore{configDir: configDir, scenarioRoots: append([]string(nil), scenarioRoots...)}
}

// NewFileSkillStoreWithScenarioRootsAndCache adds a rebuildable derived index
// for scenario-owned skills. The cache is never read as authority: deleting it
// cannot hide a source skill, and rebuilding always walks the declared roots.
func NewFileSkillStoreWithScenarioRootsAndCache(configDir, cacheDir string, scenarioRoots ...string) *FileSkillStore {
	return &FileSkillStore{configDir: configDir, scenarioRoots: append([]string(nil), scenarioRoots...), scenarioCacheDir: cacheDir}
}

// NewRoutedFileSkillStore creates a skill store whose Config-class root is
// selected for every request. The primary root stays in use outside test mode;
// a dev-routing lease redirects only test-mode requests to its disposable copy.
func NewRoutedFileSkillStore(roots *filerouting.RoutedRoots) *FileSkillStore {
	return &FileSkillStore{roots: roots}
}

func (s *FileSkillStore) forContext(ctx context.Context) (*FileSkillStore, error) {
	if s.roots == nil {
		return s, nil
	}
	configDir, err := s.roots.Pick(ctx, storage.ClassConfig)
	if err != nil {
		return nil, fmt.Errorf("resolve routed config root: %w", err)
	}
	copy := *s
	copy.configDir = configDir
	copy.roots = nil
	return &copy, nil
}

func (s *FileSkillStore) recordWrite(ctx context.Context) {
	if s.roots != nil {
		s.roots.RecordWrite(ctx)
	}
}

// skillsDir returns the path to the skills directory
func (s *FileSkillStore) skillsDir() string {
	return filepath.Join(s.configDir, "skills")
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
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return nil, err
	}
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
			entry := skill.Entry
			if entry == "" {
				entry = "SKILL.md"
			}
			if _, err := os.Stat(filepath.Join(packPath, skillID, entry)); err != nil {
				continue // Compatibility sidecar without authored bytes; root owns it.
			}

			skill.Pack = pack
			skills = append(skills, *skill)
			seen[skillID] = true
		}
	}
	for _, scenarioRoot := range s.scenarioRoots {
		scenarioSkills, err := s.listScenarioRoot(scenarioRoot)
		if err != nil {
			return nil, err
		}
		for _, skill := range scenarioSkills {
			if seen[skill.ID] {
				return nil, fmt.Errorf("duplicate skill identifier %q across registry roots", skill.ID)
			}
			seen[skill.ID] = true
			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// Get retrieves a skill by ID, searching through active packs
func (s *FileSkillStore) Get(ctx context.Context, id string) (*Skill, error) {
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return nil, err
	}
	packs, err := s.getActivePacks()
	if err != nil {
		return nil, fmt.Errorf("getting active packs: %w", err)
	}

	for _, pack := range packs {
		skill, err := s.loadSkill(pack, id)
		if err == nil {
			entry := skill.Entry
			if entry == "" {
				entry = "SKILL.md"
			}
			if _, statErr := os.Stat(filepath.Join(s.packsDir(), pack, id, entry)); statErr != nil {
				continue // Generated registry index; source is discovered below.
			}
			if skill.Origin != nil && skill.Origin.Kind == OriginImported && skill.Origin.Review.Verdict != ReviewVerdictPassed {
				return nil, fmt.Errorf("skill %s is quarantined: review verdict is %s", id, skill.Origin.Review.Verdict)
			}
			skill.Pack = pack
			return skill, nil
		}
	}
	for _, scenarioRoot := range s.scenarioRoots {
		scenarioSkills, err := s.listScenarioRoot(scenarioRoot)
		if err != nil {
			return nil, err
		}
		for _, skill := range scenarioSkills {
			if skill.ID == id {
				skill.Pack = "scenario"
				return &skill, nil
			}
		}
	}
	// Keep quarantine visible to callers as a named refusal rather than making
	// a pending import look like a typo or a missing skill.
	if pending, pendingErr := s.loadSkill("vendor", id); pendingErr == nil && pending.Origin != nil && pending.Origin.Kind == OriginImported && pending.Origin.Review.Verdict != ReviewVerdictPassed {
		return nil, fmt.Errorf("skill %s is quarantined: review verdict is %s", id, pending.Origin.Review.Verdict)
	}

	return nil, fmt.Errorf("skill not found: %s", id)
}

// GetWithContent retrieves a skill with its content
func (s *FileSkillStore) GetWithContent(ctx context.Context, id string) (*Skill, string, error) {
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return nil, "", err
	}
	skill, err := s.Get(ctx, id)
	if err != nil {
		return nil, "", err
	}

	contentPath := filepath.Join(s.packsDir(), skill.Pack, id, skill.Entry)
	if skill.SourceDir != "" {
		contentPath = filepath.Join(skill.SourceDir, skill.Entry)
	}
	content, err := ReadContent(contentPath)
	if err != nil {
		return skill, "", fmt.Errorf("reading content: %w", err)
	}

	return skill, content, nil
}

// Create creates a new skill in the specified pack
func (s *FileSkillStore) Create(ctx context.Context, pack string, skill *Skill, content string) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
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
	if err := WriteContent(filepath.Join(skillDir, "SKILL.md"), EnsureSkillFrontmatter(skill, content)); err != nil {
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

	original.recordWrite(ctx)
	return nil
}

// Update updates an existing skill
func (s *FileSkillStore) Update(ctx context.Context, id string, updates *Skill, content *string) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
	skill, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if skill.SourceDir != "" {
		return fmt.Errorf("cannot edit scenario-owned skill %s in place; edit its source at %s", id, skill.SourceDir)
	}
	if skill.Origin != nil && skill.Origin.Kind == OriginImported {
		return fmt.Errorf("cannot edit vendored skill %s in place; write an overlay under %s", id, s.ImportedSkillOverlayPath(id))
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
	// TargetDimensions and ProgrammaticHome are always applied (nil/empty clears
	// them). The store_adapter round-trips the full current Metadata on every
	// update, so partial edits still carry these forward; applying them
	// unconditionally is what lets an explicit clear actually persist. A
	// nil-guard here would silently drop both set-to-nil and the EM dimension
	// edits, which is the bug this replaces.
	skill.TargetDimensions = updates.TargetDimensions
	skill.ProgrammaticHome = updates.ProgrammaticHome

	skill.UpdateTimestamp()

	// Write updated skill.json
	if err := SaveJSON(filepath.Join(skillDir, "skill.json"), skill); err != nil {
		return fmt.Errorf("writing skill.json: %w", err)
	}

	// Write content if provided
	if content != nil {
		if err := WriteContent(filepath.Join(skillDir, "SKILL.md"), EnsureSkillFrontmatter(skill, *content)); err != nil {
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

	original.recordWrite(ctx)
	return nil
}

// Delete removes a skill
func (s *FileSkillStore) Delete(ctx context.Context, id string) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
	skill, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	skillDir := filepath.Join(s.packsDir(), skill.Pack, id)
	if err := DeleteDirectory(skillDir); err != nil {
		return err
	}
	original.recordWrite(ctx)
	return nil
}

// GetVersionHistory returns version history for a skill
func (s *FileSkillStore) GetVersionHistory(ctx context.Context, id string) ([]HistoryEntry, error) {
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return nil, err
	}
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

func (s *FileSkillStore) listScenarioRoot(root string) ([]Skill, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	result := make([]Skill, 0)
	for _, scenario := range entries {
		if !scenario.IsDir() {
			continue
		}
		skillsDir := filepath.Join(root, scenario.Name(), "skills")
		for _, entry := range mustReadDir(skillsDir) {
			if !entry.IsDir() {
				continue
			}
			sourceDir := filepath.Join(skillsDir, entry.Name())
			content, readErr := os.ReadFile(filepath.Join(sourceDir, "SKILL.md"))
			if readErr != nil {
				continue
			}
			name, description, valid := frontmatterSummary(string(content))
			if !valid || name != entry.Name() {
				return nil, fmt.Errorf("scenario skill %s has invalid or mismatched frontmatter", sourceDir)
			}
			result = append(result, Skill{ID: entry.Name(), Name: name, Description: description, Status: StatusActive, Entry: "SKILL.md", Pack: "scenario", SourceDir: sourceDir, Timestamps: Timestamps{Revision: 1}})
		}
	}
	return result, nil
}

type scenarioSkillCacheEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SourceDir   string `json:"sourceDir"`
}

// ScenarioSkillCachePath returns the derived cache location. An empty result
// means this store was created without a cache seam (as in small unit tests).
func (s *FileSkillStore) ScenarioSkillCachePath() string {
	if s == nil || s.scenarioCacheDir == "" {
		return ""
	}
	return filepath.Join(s.scenarioCacheDir, "skills", "scenario-index.json")
}

// RebuildScenarioSkillCache walks the source roots and writes a deterministic
// index. It intentionally does not make List depend on this file; the cache is
// an acceleration/evidence artifact, never a second source of truth.
func (s *FileSkillStore) RebuildScenarioSkillCache(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("skill store is nil")
	}
	resolved, err := s.forContext(ctx)
	if err != nil {
		return err
	}
	path := resolved.ScenarioSkillCachePath()
	if path == "" {
		return fmt.Errorf("scenario skill cache is not configured")
	}
	var entries []scenarioSkillCacheEntry
	for _, root := range resolved.scenarioRoots {
		items, err := resolved.listScenarioRoot(root)
		if err != nil {
			return err
		}
		for _, item := range items {
			entries = append(entries, scenarioSkillCacheEntry{ID: item.ID, Name: item.Name, Description: item.Description, SourceDir: item.SourceDir})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ID == entries[j].ID {
			return entries[i].SourceDir < entries[j].SourceDir
		}
		return entries[i].ID < entries[j].ID
	})
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func mustReadDir(path string) []os.DirEntry {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	return entries
}

func frontmatterSummary(content string) (name, description string, valid bool) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", false
	}
	closed := false
	for _, raw := range lines[1:] {
		line := strings.TrimSpace(raw)
		if line == "---" {
			closed = true
			break
		}
		if strings.HasPrefix(line, "name:") {
			name = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "name:")), "\"'")
		}
		if strings.HasPrefix(line, "description:") {
			description = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "description:")), "\"'")
		}
	}
	return name, description, closed && name != "" && description != ""
}

// ContentPath returns the path where a skill's content would be stored.
// This is useful for pre-writing content during move operations.
func (s *FileSkillStore) ContentPath(pack, skillID string) string {
	return filepath.Join(s.packsDir(), pack, skillID, "SKILL.md")
}

// ContentPathContext resolves the direct-content compatibility path through
// the same request-scoped Config root as normal skill operations.
func (s *FileSkillStore) ContentPathContext(ctx context.Context, pack, skillID string) (string, error) {
	resolved, err := s.forContext(ctx)
	if err != nil {
		return "", err
	}
	return resolved.ContentPath(pack, skillID), nil
}

// RecordWrite is used by compatibility adapters that perform a direct content
// write before skill metadata exists.
func (s *FileSkillStore) RecordWrite(ctx context.Context) { s.recordWrite(ctx) }

// Rename renames a skill by changing its directory name and updating skill.json.
// This is an atomic operation: if any step fails, the original state is preserved.
// Implements store.SkillStore.Rename()
func (s *FileSkillStore) Rename(ctx context.Context, oldID, newID string) (*Skill, error) {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return nil, err
	}
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

	original.recordWrite(ctx)
	return skill, nil
}
