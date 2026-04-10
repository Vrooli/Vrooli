// Package persistence provides data access for session profile management.
package persistence

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// FileRepository implements Repository using JSON files with atomic writes.
type FileRepository struct {
	root string
	log  *logrus.Logger
	fs   FileSystem
}

// FileRepositoryConfig configures the FileRepository.
type FileRepositoryConfig struct {
	// FileSystem provides file operations. If nil, uses the real OS filesystem.
	FileSystem FileSystem
}

// NewFileRepository creates a file-based repository at the given path.
// The directory is created if it does not exist.
func NewFileRepository(root string, log *logrus.Logger) *FileRepository {
	return NewFileRepositoryWithConfig(root, log, FileRepositoryConfig{})
}

// NewFileRepositoryWithConfig creates a file-based repository with the given configuration.
func NewFileRepositoryWithConfig(root string, log *logrus.Logger, config FileRepositoryConfig) *FileRepository {
	if root == "" {
		root = filepath.Join("scenarios", "browser-automation-studio", "data", "session-profiles")
	}
	if resolved, err := filepath.Abs(root); err == nil {
		root = resolved
	}
	fsys := config.FileSystem
	if fsys == nil {
		fsys = NewOSFileSystem()
	}
	if err := fsys.MkdirAll(root, 0o755); err != nil && log != nil {
		log.WithError(err).Warn("Failed to ensure session profiles directory exists")
	}
	return &FileRepository{
		root: root,
		log:  log,
		fs:   fsys,
	}
}

// Get retrieves a profile by ID.
func (r *FileRepository) Get(id ProfileID) (*SessionProfile, error) {
	if id == "" {
		return nil, errors.New("profile id is required")
	}
	data, err := r.fs.ReadFile(r.profilePath(id))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil // Profile not found, return nil without error
		}
		return nil, fmt.Errorf("read profile: %w", err)
	}
	var profile SessionProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}
	return &profile, nil
}

// List returns all profiles sorted by last_used_at (desc) then created_at.
func (r *FileRepository) List() ([]SessionProfile, error) {
	files, err := r.fs.ReadDir(r.root)
	if err != nil {
		return nil, fmt.Errorf("read session profiles: %w", err)
	}

	profiles := make([]SessionProfile, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		id := ProfileID(strings.TrimSuffix(file.Name(), ".json"))
		profile, err := r.Get(id)
		if err != nil {
			if r.log != nil {
				r.log.WithError(err).WithField("file", file.Name()).Warn("Skipping unreadable session profile")
			}
			continue
		}
		if profile != nil {
			profiles = append(profiles, *profile)
		}
	}

	sort.Slice(profiles, func(i, j int) bool {
		if !profiles[i].LastUsedAt.Equal(profiles[j].LastUsedAt) {
			return profiles[i].LastUsedAt.After(profiles[j].LastUsedAt)
		}
		return profiles[i].CreatedAt.After(profiles[j].CreatedAt)
	})

	return profiles, nil
}

// Create persists a new profile.
func (r *FileRepository) Create(profile *SessionProfile) error {
	if profile == nil {
		return errors.New("profile is nil")
	}
	if profile.ID == "" {
		return errors.New("profile id is required")
	}

	// Check if profile already exists
	existing, err := r.Get(profile.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("profile already exists: %s", profile.ID)
	}

	return r.Save(profile)
}

// Save atomically persists the entire profile.
// Uses write-to-temp-file-then-rename pattern for crash safety.
func (r *FileRepository) Save(profile *SessionProfile) error {
	if profile == nil {
		return errors.New("profile is nil")
	}
	if profile.ID == "" {
		return errors.New("profile id is required")
	}

	// Set timestamps if not already set
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = profile.UpdatedAt
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = profile.CreatedAt
	}
	if profile.LastUsedAt.IsZero() {
		profile.LastUsedAt = profile.CreatedAt
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profile: %w", err)
	}

	// Write to temp file first for atomic operation
	targetPath := r.profilePath(profile.ID)
	tempPath := targetPath + ".tmp"

	if err := r.fs.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp profile: %w", err)
	}

	// Atomic rename (on POSIX systems, this is atomic for same-filesystem renames)
	if err := r.fs.Rename(tempPath, targetPath); err != nil {
		// Clean up temp file on failure
		_ = r.fs.Remove(tempPath)
		return fmt.Errorf("rename profile: %w", err)
	}

	return nil
}

// Delete removes a profile by ID.
func (r *FileRepository) Delete(id ProfileID) error {
	if id == "" {
		return errors.New("profile id is required")
	}
	path := r.profilePath(id)
	if _, err := r.fs.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("profile not found")
		}
		return fmt.Errorf("delete profile: %w", err)
	}
	if err := r.fs.Remove(path); err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	return nil
}

func (r *FileRepository) profilePath(id ProfileID) string {
	return filepath.Join(r.root, fmt.Sprintf("%s.json", id))
}

// Ensure FileRepository implements Repository at compile time.
var _ Repository = (*FileRepository)(nil)
