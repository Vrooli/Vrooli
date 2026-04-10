package autosteer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	ProfileKindProfile  = "profile"
	ProfileKindTemplate = "template"
)

// ProfileMetadataIndex is the authoritative registry for file-backed profiles.
type ProfileMetadataIndex struct {
	Profiles []ProfileMetadata `json:"profiles"`
}

// ProfileMetadata describes a single profile entry in the registry.
type ProfileMetadata struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	Kind        string   `json:"kind"`
	File        string   `json:"file"`
}

// FileProfileRepository implements profile storage using the filesystem registry.
type FileProfileRepository struct {
	rootDir   string
	indexPath string

	mu       sync.RWMutex
	profiles map[string]*AutoSteerProfile
	metadata map[string]ProfileMetadata
}

var (
	_ ProfileRepository = (*FileProfileRepository)(nil)
	_ ProfileServiceAPI = (*FileProfileRepository)(nil)
)

// NewFileProfileRepository creates a repository backed by the profile registry folder.
func NewFileProfileRepository(rootDir string) (*FileProfileRepository, error) {
	repo := &FileProfileRepository{
		rootDir:   rootDir,
		indexPath: filepath.Join(rootDir, "metadata.json"),
		profiles:  make(map[string]*AutoSteerProfile),
		metadata:  make(map[string]ProfileMetadata),
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

// Reload refreshes the in-memory cache from disk.
func (r *FileProfileRepository) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loadLocked()
}

// GetProfile retrieves a profile by ID.
func (r *FileProfileRepository) GetProfile(id string) (*AutoSteerProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.profiles[id]
	if !ok {
		return nil, fmt.Errorf("profile not found: %s", id)
	}

	copy := *profile
	return &copy, nil
}

// ListProfiles retrieves all non-template profiles with optional tag filtering.
func (r *FileProfileRepository) ListProfiles(tags []string) ([]*AutoSteerProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.listLocked(ProfileKindProfile, tags), nil
}

// GetTemplates returns all template profiles.
func (r *FileProfileRepository) GetTemplates() []*AutoSteerProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.listLocked(ProfileKindTemplate, nil)
}

// CreateProfile inserts a new profile and writes it to disk.
func (r *FileProfileRepository) CreateProfile(profile *AutoSteerProfile) error {
	if profile == nil {
		return fmt.Errorf("profile is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}
	if _, exists := r.metadata[profile.ID]; exists {
		return fmt.Errorf("profile already exists: %s", profile.ID)
	}
	if nameExistsLocked(r.metadata, profile.Name, profile.ID) {
		return fmt.Errorf("profile name already exists: %s", profile.Name)
	}

	now := time.Now().UTC()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	if err := ValidateProfile(profile); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	entry := ProfileMetadata{
		ID:          profile.ID,
		Name:        profile.Name,
		Description: profile.Description,
		Tags:        append([]string(nil), profile.Tags...),
		Kind:        ProfileKindProfile,
		File:        filepath.ToSlash(filepath.Join(profile.ID, "profile.json")),
	}

	if err := r.writeProfileFile(entry, profile); err != nil {
		return err
	}

	r.metadata[profile.ID] = entry
	r.profiles[profile.ID] = cloneProfile(profile)

	if err := r.flushMetadataLocked(); err != nil {
		delete(r.metadata, profile.ID)
		delete(r.profiles, profile.ID)
		_ = r.removeProfileFile(entry)
		return err
	}

	return nil
}

// UpdateProfile updates an existing profile and writes it to disk.
func (r *FileProfileRepository) UpdateProfile(id string, updates *AutoSteerProfile) error {
	if updates == nil {
		return fmt.Errorf("profile updates are required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.profiles[id]
	if !ok {
		return fmt.Errorf("profile not found: %s", id)
	}
	previousProfile := cloneProfile(existing)
	previousEntry := r.metadata[id]

	if nameExistsLocked(r.metadata, updates.Name, id) {
		return fmt.Errorf("profile name already exists: %s", updates.Name)
	}

	updates.ID = id
	updates.CreatedAt = existing.CreatedAt
	updates.UpdatedAt = time.Now().UTC()

	if err := ValidateProfile(updates); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	entry := r.metadata[id]
	entry.Name = updates.Name
	entry.Description = updates.Description
	entry.Tags = append([]string(nil), updates.Tags...)

	if err := r.writeProfileFile(entry, updates); err != nil {
		return err
	}

	r.metadata[id] = entry
	r.profiles[id] = cloneProfile(updates)

	if err := r.flushMetadataLocked(); err != nil {
		r.metadata[id] = previousEntry
		r.profiles[id] = previousProfile
		_ = r.writeProfileFile(previousEntry, previousProfile)
		return err
	}

	return nil
}

// DeleteProfile removes a profile from disk and the registry.
func (r *FileProfileRepository) DeleteProfile(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.metadata[id]
	if !ok {
		return fmt.Errorf("profile not found: %s", id)
	}
	previousProfile := cloneProfile(r.profiles[id])
	previousEntry := entry

	if err := r.removeProfileFile(entry); err != nil {
		return err
	}

	delete(r.metadata, id)
	delete(r.profiles, id)

	if err := r.flushMetadataLocked(); err != nil {
		r.metadata[id] = previousEntry
		r.profiles[id] = previousProfile
		_ = r.writeProfileFile(previousEntry, previousProfile)
		return err
	}

	return nil
}

func (r *FileProfileRepository) listLocked(kind string, tags []string) []*AutoSteerProfile {
	profiles := make([]*AutoSteerProfile, 0, len(r.profiles))
	for id, profile := range r.profiles {
		entry, ok := r.metadata[id]
		if !ok {
			continue
		}
		if kind != "" && entry.Kind != kind {
			continue
		}
		if len(tags) > 0 && !hasAnyTag(profile.Tags, tags) {
			continue
		}
		copy := *profile
		profiles = append(profiles, &copy)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return strings.ToLower(profiles[i].Name) < strings.ToLower(profiles[j].Name)
	})

	return profiles
}

func (r *FileProfileRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loadLocked()
}

func (r *FileProfileRepository) loadLocked() error {
	data, err := os.ReadFile(r.indexPath)
	if err != nil {
		return fmt.Errorf("failed to read profile metadata: %w", err)
	}

	var index ProfileMetadataIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("failed to parse profile metadata: %w", err)
	}

	profiles := make(map[string]*AutoSteerProfile, len(index.Profiles))
	metadata := make(map[string]ProfileMetadata, len(index.Profiles))
	names := make(map[string]string, len(index.Profiles))

	for _, entry := range index.Profiles {
		if err := validateMetadataEntry(entry); err != nil {
			return err
		}

		normalizedKind := normalizeProfileKind(entry.Kind)
		entry.Kind = normalizedKind

		if _, exists := metadata[entry.ID]; exists {
			return fmt.Errorf("duplicate profile id in metadata: %s", entry.ID)
		}

		nameKey := strings.ToLower(entry.Name)
		if existingID, exists := names[nameKey]; exists {
			return fmt.Errorf("duplicate profile name '%s' in metadata (%s vs %s)", entry.Name, existingID, entry.ID)
		}
		names[nameKey] = entry.ID

		profilePath, err := r.resolveProfilePath(entry.File)
		if err != nil {
			return err
		}

		raw, err := os.ReadFile(profilePath)
		if err != nil {
			return fmt.Errorf("failed to read profile file %s: %w", profilePath, err)
		}

		var profile AutoSteerProfile
		if err := json.Unmarshal(raw, &profile); err != nil {
			return fmt.Errorf("failed to parse profile file %s: %w", profilePath, err)
		}

		if profile.ID == "" {
			profile.ID = entry.ID
		}
		if profile.ID != entry.ID {
			return fmt.Errorf("profile file %s has mismatched id %s (expected %s)", profilePath, profile.ID, entry.ID)
		}

		if profile.Name != "" && profile.Name != entry.Name {
			return fmt.Errorf("profile file %s has mismatched name '%s' (expected '%s')", profilePath, profile.Name, entry.Name)
		}
		if profile.Description != "" && profile.Description != entry.Description {
			return fmt.Errorf("profile file %s has mismatched description", profilePath)
		}
		if len(profile.Tags) > 0 && !tagsMatch(profile.Tags, entry.Tags) {
			return fmt.Errorf("profile file %s has mismatched tags", profilePath)
		}

		profile.Name = entry.Name
		profile.Description = entry.Description
		profile.Tags = append([]string(nil), entry.Tags...)

		if err := ValidateProfile(&profile); err != nil {
			return fmt.Errorf("profile %s invalid: %w", entry.ID, err)
		}

		profiles[entry.ID] = &profile
		metadata[entry.ID] = entry
	}

	r.profiles = profiles
	r.metadata = metadata

	return nil
}

func (r *FileProfileRepository) writeProfileFile(entry ProfileMetadata, profile *AutoSteerProfile) error {
	if entry.File == "" {
		return fmt.Errorf("profile file path is required")
	}

	profilePath, err := r.resolveProfilePath(entry.File)
	if err != nil {
		return err
	}

	copy := cloneProfile(profile)
	copy.Name = entry.Name
	copy.Description = entry.Description
	copy.Tags = append([]string(nil), entry.Tags...)

	if err := writeJSONAtomic(profilePath, copy); err != nil {
		return fmt.Errorf("failed to write profile file: %w", err)
	}

	return nil
}

func (r *FileProfileRepository) removeProfileFile(entry ProfileMetadata) error {
	profilePath, err := r.resolveProfilePath(entry.File)
	if err != nil {
		return err
	}

	if err := os.Remove(profilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove profile file: %w", err)
	}

	dir := filepath.Dir(profilePath)
	if dir != r.rootDir {
		_ = os.Remove(dir)
	}

	return nil
}

func (r *FileProfileRepository) flushMetadataLocked() error {
	index := r.buildIndexLocked()
	if err := writeJSONAtomic(r.indexPath, index); err != nil {
		return fmt.Errorf("failed to write profile metadata: %w", err)
	}
	return nil
}

func (r *FileProfileRepository) buildIndexLocked() ProfileMetadataIndex {
	entries := make([]ProfileMetadata, 0, len(r.metadata))
	for _, entry := range r.metadata {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if strings.EqualFold(entries[i].Name, entries[j].Name) {
			return entries[i].ID < entries[j].ID
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return ProfileMetadataIndex{Profiles: entries}
}

func (r *FileProfileRepository) resolveProfilePath(rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("profile file path is required")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("profile file path must be relative: %s", rel)
	}

	cleaned := filepath.Clean(rel)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("profile file path escapes registry root: %s", rel)
	}

	return filepath.Join(r.rootDir, cleaned), nil
}

func validateMetadataEntry(entry ProfileMetadata) error {
	if entry.ID == "" {
		return fmt.Errorf("profile metadata entry missing id")
	}
	if entry.Name == "" {
		return fmt.Errorf("profile metadata entry %s missing name", entry.ID)
	}
	if entry.File == "" {
		return fmt.Errorf("profile metadata entry %s missing file", entry.ID)
	}
	kind := normalizeProfileKind(entry.Kind)
	if kind == "" {
		return fmt.Errorf("profile metadata entry %s has invalid kind '%s'", entry.ID, entry.Kind)
	}
	return nil
}

func normalizeProfileKind(kind string) string {
	if kind == "" {
		return ProfileKindProfile
	}
	normalized := strings.ToLower(strings.TrimSpace(kind))
	switch normalized {
	case ProfileKindProfile, ProfileKindTemplate:
		return normalized
	default:
		return ""
	}
}

func nameExistsLocked(metadata map[string]ProfileMetadata, name string, ignoreID string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	nameKey := strings.ToLower(strings.TrimSpace(name))
	for id, entry := range metadata {
		if id == ignoreID {
			continue
		}
		if strings.ToLower(strings.TrimSpace(entry.Name)) == nameKey {
			return true
		}
	}
	return false
}

func tagsMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	set := make(map[string]int, len(a))
	for _, tag := range a {
		set[tag]++
	}
	for _, tag := range b {
		if set[tag] == 0 {
			return false
		}
		set[tag]--
	}
	return true
}

func cloneProfile(profile *AutoSteerProfile) *AutoSteerProfile {
	copy := *profile
	copy.Tags = append([]string(nil), profile.Tags...)
	copy.Phases = clonePhases(profile.Phases)
	copy.QualityGates = cloneQualityGates(profile.QualityGates)
	return &copy
}

func clonePhases(phases []SteerPhase) []SteerPhase {
	if len(phases) == 0 {
		return nil
	}
	cloned := make([]SteerPhase, len(phases))
	for i, phase := range phases {
		cloned[i] = phase
		cloned[i].SkillIDs = append([]string(nil), phase.SkillIDs...)
		cloned[i].StopConditions = cloneStopConditions(phase.StopConditions)
	}
	return cloned
}

func cloneQualityGates(gates []QualityGate) []QualityGate {
	if len(gates) == 0 {
		return nil
	}
	cloned := make([]QualityGate, len(gates))
	for i, gate := range gates {
		cloned[i] = gate
		cloned[i].Condition = cloneStopCondition(gate.Condition)
	}
	return cloned
}

func cloneStopConditions(conditions []StopCondition) []StopCondition {
	if len(conditions) == 0 {
		return nil
	}
	cloned := make([]StopCondition, len(conditions))
	for i, condition := range conditions {
		cloned[i] = cloneStopCondition(condition)
	}
	return cloned
}

func cloneStopCondition(condition StopCondition) StopCondition {
	clone := condition
	if len(condition.Conditions) > 0 {
		clone.Conditions = cloneStopConditions(condition.Conditions)
	}
	return clone
}

func writeJSONAtomic(path string, value any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, "tmp-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to encode json: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("failed to move temp file into place: %w", err)
	}

	if err := os.Chmod(path, 0o644); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
