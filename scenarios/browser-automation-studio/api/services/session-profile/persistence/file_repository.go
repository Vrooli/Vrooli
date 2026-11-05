// Package persistence provides data access for session profile management.
package persistence

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// FileRepository implements Repository using JSON files with atomic writes.
type FileRepository struct {
	root      string
	fs        FileSystem
	authority func() (*credentialauthority.Authority, error)
}

// FileRepositoryConfig configures the FileRepository.
type FileRepositoryConfig struct {
	// FileSystem provides file operations. If nil, uses the real OS filesystem.
	FileSystem FileSystem
	// Authority resolves credentials in process. Nil uses the platform authority.
	Authority func() (*credentialauthority.Authority, error)
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
	authority := config.Authority
	if authority == nil {
		authority = credentialauthority.Default
	}
	return &FileRepository{
		root:      root,
		fs:        fsys,
		authority: authority,
	}
}

// Get retrieves a profile by ID.
func (r *FileRepository) Get(id ProfileID) (*SessionProfile, error) {
	path, err := r.profilePath(id)
	if err != nil {
		return nil, err
	}
	data, err := r.fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil // Profile not found, return nil without error
		}
		return nil, fmt.Errorf("read profile: %w", err)
	}
	return r.decodeProfile(id, data)
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
			return nil, fmt.Errorf("recover session profile %q: %w", id, err)
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
	return r.withWriteLock(profile.ID, func(path string) error {
		existing, err := r.Get(profile.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			return fmt.Errorf("profile already exists: %s", profile.ID)
		}
		return r.save(profile, path)
	})
}

// Update reads and changes one current snapshot while holding write ownership.
func (r *FileRepository) Update(id ProfileID, modify func(*SessionProfile) error) (*SessionProfile, error) {
	var profile *SessionProfile
	err := r.withWriteLock(id, func(path string) error {
		var err error
		profile, err = r.Get(id)
		if err != nil {
			return err
		}
		if profile == nil {
			return ErrProfileNotFound
		}
		if err := modify(profile); err != nil {
			return err
		}
		if profile.ID != id {
			return errors.New("profile update cannot change identity")
		}
		return r.save(profile, path)
	})
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (r *FileRepository) withWriteLock(id ProfileID, write func(path string) error) error {
	path, err := r.profilePath(id)
	if err != nil {
		return err
	}
	release, err := r.fs.Lock(filepath.Join(r.root, ".profiles.lock"))
	if err != nil {
		return fmt.Errorf("acquire profile write ownership: %w", err)
	}
	defer release()
	return write(path)
}

func (r *FileRepository) save(profile *SessionProfile, path string) error {
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

	data, err := r.encodeProfile(profile)
	if err != nil {
		return err
	}
	if err := r.fs.WriteFileAtomic(path, data, 0o600); err != nil {
		return fmt.Errorf("commit profile: %w", err)
	}

	return nil
}

// Delete removes a profile by ID.
func (r *FileRepository) Delete(id ProfileID) error {
	return r.withWriteLock(id, func(path string) error {
		if err := r.fs.Remove(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return ErrProfileNotFound
			}
			return fmt.Errorf("delete profile: %w", err)
		}
		return nil
	})
}

func (r *FileRepository) profilePath(id ProfileID) (string, error) {
	if id == "" {
		return "", errors.New("profile id is required")
	}
	// Reject separators on every host, including Windows drive/stream syntax.
	if id == "." || id == ".." || strings.ContainsAny(string(id), "/\\:\x00") {
		return "", ErrInvalidProfileID
	}
	return filepath.Join(r.root, string(id)+".json"), nil
}

// Ensure FileRepository implements Repository at compile time.
var _ Repository = (*FileRepository)(nil)
