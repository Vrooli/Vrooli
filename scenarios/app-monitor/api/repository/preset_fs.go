package repository

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
)

var validPresetID = regexp.MustCompile(`^[0-9a-f-]+$`)

// FilePresetRepository implements WorkspacePresetRepository using the filesystem.
type FilePresetRepository struct {
	resolver *storage.Resolver
	opts     storage.Options
}

// NewFilePresetRepository creates a FilePresetRepository with default options.
func NewFilePresetRepository(resolver *storage.Resolver) *FilePresetRepository {
	return &FilePresetRepository{
		resolver: resolver,
		opts:     storage.Options{ScenarioID: "app-monitor"},
	}
}

// NewFilePresetRepositoryWithOpts creates a FilePresetRepository with custom options (for tests).
func NewFilePresetRepositoryWithOpts(resolver *storage.Resolver, opts storage.Options) *FilePresetRepository {
	return &FilePresetRepository{
		resolver: resolver,
		opts:     opts,
	}
}

// presetsDir ensures and returns the presets directory path.
func (r *FilePresetRepository) presetsDir() (string, error) {
	dataDir, err := storage.EnsureClassDir(r.resolver, r.opts, storage.ClassData, 0)
	if err != nil {
		return "", fmt.Errorf("ensure data dir: %w", err)
	}
	presetsDir := filepath.Join(dataDir, "presets")
	if err := os.MkdirAll(presetsDir, 0o755); err != nil {
		return "", fmt.Errorf("create presets dir: %w", err)
	}
	return presetsDir, nil
}

// validateID checks that the preset ID is non-empty and matches the expected format.
func validateID(id string) error {
	if id == "" {
		return fmt.Errorf("preset id is empty")
	}
	if !validPresetID.MatchString(id) {
		return fmt.Errorf("preset id contains invalid characters: %s", id)
	}
	return nil
}

// generateUUID generates a v4 UUID using crypto/rand.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ListPresets returns all presets sorted by UpdatedAt descending.
func (r *FilePresetRepository) ListPresets(_ context.Context) ([]WorkspacePreset, error) {
	dir, err := r.presetsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read presets dir: %w", err)
	}

	var presets []WorkspacePreset
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue // skip unreadable files
		}
		var preset WorkspacePreset
		if err := json.Unmarshal(data, &preset); err != nil {
			continue // skip malformed files
		}
		presets = append(presets, preset)
	}

	sort.Slice(presets, func(i, j int) bool {
		return presets[i].UpdatedAt.After(presets[j].UpdatedAt)
	})

	return presets, nil
}

// GetPreset returns a single preset by ID.
func (r *FilePresetRepository) GetPreset(_ context.Context, id string) (*WorkspacePreset, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}

	path, err := r.resolver.Path(r.opts, storage.ClassData, "presets/"+id+".json")
	if err != nil {
		return nil, fmt.Errorf("resolve preset path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("preset not found: %w", os.ErrNotExist)
		}
		return nil, fmt.Errorf("read preset file: %w", err)
	}

	var preset WorkspacePreset
	if err := json.Unmarshal(data, &preset); err != nil {
		return nil, fmt.Errorf("unmarshal preset: %w", err)
	}

	return &preset, nil
}

// CreatePreset persists a new preset. The ID and timestamps are generated automatically.
func (r *FilePresetRepository) CreatePreset(_ context.Context, preset *WorkspacePreset) error {
	preset.ID = generateUUID()
	now := time.Now().UTC()
	preset.CreatedAt = now
	preset.UpdatedAt = now

	data, err := json.MarshalIndent(preset, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal preset: %w", err)
	}

	path, err := r.resolver.Path(r.opts, storage.ClassData, "presets/"+preset.ID+".json")
	if err != nil {
		return fmt.Errorf("resolve preset path: %w", err)
	}

	if err := storage.WriteFileAtomic(path, data, 0); err != nil {
		return fmt.Errorf("write preset file: %w", err)
	}

	return nil
}

// UpdatePreset updates an existing preset. The existing file must already exist.
func (r *FilePresetRepository) UpdatePreset(_ context.Context, preset *WorkspacePreset) error {
	if err := validateID(preset.ID); err != nil {
		return err
	}

	path, err := r.resolver.Path(r.opts, storage.ClassData, "presets/"+preset.ID+".json")
	if err != nil {
		return fmt.Errorf("resolve preset path: %w", err)
	}

	// Verify the preset exists
	existingData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("preset not found: %w", os.ErrNotExist)
		}
		return fmt.Errorf("read existing preset: %w", err)
	}

	// Preserve the original creation timestamp
	var existing WorkspacePreset
	if err := json.Unmarshal(existingData, &existing); err != nil {
		return fmt.Errorf("unmarshal existing preset: %w", err)
	}
	preset.CreatedAt = existing.CreatedAt
	preset.UpdatedAt = time.Now().UTC()

	data, err := json.MarshalIndent(preset, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal preset: %w", err)
	}

	if err := storage.WriteFileAtomic(path, data, 0); err != nil {
		return fmt.Errorf("write preset file: %w", err)
	}

	return nil
}

// DeletePreset removes a preset by ID.
func (r *FilePresetRepository) DeletePreset(_ context.Context, id string) error {
	if err := validateID(id); err != nil {
		return err
	}

	path, err := r.resolver.Path(r.opts, storage.ClassData, "presets/"+id+".json")
	if err != nil {
		return fmt.Errorf("resolve preset path: %w", err)
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("preset not found: %w", os.ErrNotExist)
		}
		return fmt.Errorf("stat preset file: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove preset file: %w", err)
	}

	return nil
}
