package archiveingestion

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SessionProfile captures a persisted Playwright storage state along with metadata.
type SessionProfile struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	LastUsedAt   time.Time       `json:"last_used_at"`
	StorageState json.RawMessage `json:"storage_state,omitempty"`
}

// SessionProfileStore manages persisted session profiles on disk.
type SessionProfileStore struct {
	root string
	log  *logrus.Logger
	mu   sync.Mutex
}

// NewSessionProfileStore creates a store rooted at the given path, ensuring the directory exists.
func NewSessionProfileStore(root string, log *logrus.Logger) *SessionProfileStore {
	if root == "" {
		root = filepath.Join("scenarios", "browser-automation-studio", "data", "session-profiles")
	}
	if resolved, err := filepath.Abs(root); err == nil {
		root = resolved
	}
	if err := os.MkdirAll(root, 0o755); err != nil && log != nil {
		log.WithError(err).Warn("Failed to ensure session profiles directory exists")
	}
	return &SessionProfileStore{
		root: root,
		log:  log,
	}
}

// List returns all profiles sorted by last_used_at (desc) then created_at.
func (s *SessionProfileStore) List() ([]SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("read session profiles: %w", err)
	}

	profiles := make([]SessionProfile, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		profile, err := s.loadProfileLocked(strings.TrimSuffix(file.Name(), ".json"))
		if err != nil {
			if s.log != nil {
				s.log.WithError(err).WithField("file", file.Name()).Warn("Skipping unreadable session profile")
			}
			continue
		}
		profiles = append(profiles, *profile)
	}

	sort.Slice(profiles, func(i, j int) bool {
		if !profiles[i].LastUsedAt.Equal(profiles[j].LastUsedAt) {
			return profiles[i].LastUsedAt.After(profiles[j].LastUsedAt)
		}
		return profiles[i].CreatedAt.After(profiles[j].CreatedAt)
	})

	return profiles, nil
}

// Get retrieves a profile by ID.
func (s *SessionProfileStore) Get(id string) (*SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadProfileLocked(strings.TrimSpace(id))
}

// Create generates a new profile with an optional name and no storage state.
func (s *SessionProfileStore) Create(name string) (*SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	profile := &SessionProfile{
		ID:         uuid.NewString(),
		Name:       s.normalizeNameLocked(name),
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}

	if err := s.saveProfileLocked(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// Rename updates the profile display name.
func (s *SessionProfileStore) Rename(id, name string) (*SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := s.loadProfileLocked(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	profile.Name = s.normalizeNameLocked(name)
	profile.UpdatedAt = time.Now().UTC()

	if err := s.saveProfileLocked(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// Delete removes the profile from disk.
func (s *SessionProfileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(id) == "" {
		return errors.New("profile id is required")
	}
	path := s.profilePath(id)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("profile not found")
		}
		return fmt.Errorf("delete profile: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	return nil
}

// SaveStorageState persists the storage state and bumps last_used_at.
func (s *SessionProfileStore) SaveStorageState(id string, state json.RawMessage) (*SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := s.loadProfileLocked(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if len(state) > 0 {
		profile.StorageState = state
	}
	now := time.Now().UTC()
	profile.LastUsedAt = now
	profile.UpdatedAt = now

	if err := s.saveProfileLocked(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// Touch updates the last_used_at timestamp.
func (s *SessionProfileStore) Touch(id string) (*SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := s.loadProfileLocked(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	profile.LastUsedAt = time.Now().UTC()
	profile.UpdatedAt = profile.LastUsedAt

	if err := s.saveProfileLocked(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// MostRecent returns the most recently used profile if one exists.
func (s *SessionProfileStore) MostRecent() (*SessionProfile, error) {
	profiles, err := s.List()
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}
	return &profiles[0], nil
}

func (s *SessionProfileStore) loadProfileLocked(id string) (*SessionProfile, error) {
	if id == "" {
		return nil, errors.New("profile id is required")
	}
	data, err := os.ReadFile(s.profilePath(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("profile not found")
		}
		return nil, fmt.Errorf("read profile: %w", err)
	}
	var profile SessionProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}
	return &profile, nil
}

func (s *SessionProfileStore) saveProfileLocked(profile *SessionProfile) error {
	if profile == nil {
		return errors.New("profile is nil")
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now().UTC()
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
	if err := os.WriteFile(s.profilePath(profile.ID), data, 0o600); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}
	return nil
}

func (s *SessionProfileStore) profilePath(id string) string {
	return filepath.Join(s.root, fmt.Sprintf("%s.json", id))
}

func (s *SessionProfileStore) normalizeNameLocked(name string) string {
	clean := strings.TrimSpace(name)
	if clean != "" {
		return clean
	}

	// Generate a simple incremental name: Session N
	files, err := os.ReadDir(s.root)
	if err != nil {
		return "Session"
	}
	// Count existing session files to derive a friendly ordinal
	count := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		count++
	}
	return fmt.Sprintf("Session %d", count+1)
}
