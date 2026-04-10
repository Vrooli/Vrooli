// Package sessionprofile provides session profile management for browser automation.
// It extracts session profile functionality from the archive-ingestion package into
// a cleanly-architected dedicated service with repository pattern persistence.
package sessionprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/clock"
	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// Service manages session profiles with business logic and active session tracking.
type Service struct {
	repo     persistence.Repository
	sessions *ActiveSessionRegistry
	log      *logrus.Logger
	clock    clock.Clock
}

// ServiceConfig configures the session profile service.
type ServiceConfig struct {
	// Clock provides time operations. If nil, uses the real system clock.
	Clock clock.Clock
}

// NewService creates a new session profile service.
func NewService(repo persistence.Repository, log *logrus.Logger) *Service {
	return NewServiceWithConfig(repo, log, ServiceConfig{})
}

// NewServiceWithConfig creates a new session profile service with the given configuration.
func NewServiceWithConfig(repo persistence.Repository, log *logrus.Logger, config ServiceConfig) *Service {
	clk := config.Clock
	if clk == nil {
		clk = clock.New()
	}
	return &Service{
		repo:     repo,
		sessions: NewActiveSessionRegistry(),
		log:      log,
		clock:    clk,
	}
}

// GetOrCreateProfile returns an existing profile or creates a new one.
// If requestedID is empty, returns the most recently used profile or creates a new one.
func (s *Service) GetOrCreateProfile(requestedID persistence.ProfileID) (*persistence.SessionProfile, error) {
	id := persistence.ProfileID(strings.TrimSpace(string(requestedID)))

	// If a specific profile is requested, get it
	if id != "" {
		profile, err := s.repo.Get(id)
		if err != nil {
			return nil, err
		}
		if profile == nil {
			return nil, fmt.Errorf("profile not found: %s", id)
		}
		return profile, nil
	}

	// Get most recent profile
	profiles, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	if len(profiles) > 0 {
		return &profiles[0], nil
	}

	// Create a new profile with auto-generated name
	return s.CreateProfile("")
}

// CreateProfile creates a new session profile with an optional name.
// If name is empty, generates a name like "Session N".
func (s *Service) CreateProfile(name string) (*persistence.SessionProfile, error) {
	now := s.clock.Now().UTC()

	// Generate name if not provided
	if strings.TrimSpace(name) == "" {
		name = s.generateProfileName()
	}

	profile := &persistence.SessionProfile{
		ID:         persistence.ProfileID(uuid.NewString()),
		Name:       name,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}

	if err := s.repo.Create(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// GetProfile retrieves a profile by ID.
func (s *Service) GetProfile(id persistence.ProfileID) (*persistence.SessionProfile, error) {
	if id == "" {
		return nil, fmt.Errorf("profile id is required")
	}
	profile, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, fmt.Errorf("profile not found")
	}
	return profile, nil
}

// ListProfiles returns all profiles sorted by last_used_at (desc).
func (s *Service) ListProfiles() ([]persistence.SessionProfile, error) {
	return s.repo.List()
}

// DeleteProfile removes a profile and clears any active session associations.
func (s *Service) DeleteProfile(id persistence.ProfileID) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	s.sessions.ClearForProfile(string(id))
	return nil
}

// RenameProfile updates the profile display name.
func (s *Service) RenameProfile(id persistence.ProfileID, name string) (*persistence.SessionProfile, error) {
	profile, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}

	profile.Name = strings.TrimSpace(name)
	if profile.Name == "" {
		profile.Name = s.generateProfileName()
	}
	profile.UpdatedAt = s.clock.Now().UTC()

	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// UpdateBrowserProfile updates the browser profile settings.
func (s *Service) UpdateBrowserProfile(id persistence.ProfileID, browserProfile *persistence.BrowserProfile) (*persistence.SessionProfile, error) {
	profile, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}

	// Validate browser profile if provided
	if browserProfile != nil {
		if err := ValidateBrowserProfile(browserProfile); err != nil {
			return nil, fmt.Errorf("invalid browser profile: %w", err)
		}
	}

	profile.BrowserProfile = browserProfile
	profile.UpdatedAt = s.clock.Now().UTC()

	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// StartSession associates a browser session with a profile.
// This updates the profile's last_used_at timestamp.
func (s *Service) StartSession(sessionID string, profileID persistence.ProfileID) error {
	if sessionID == "" || profileID == "" {
		return nil
	}

	s.sessions.Set(sessionID, string(profileID))

	// Touch the profile to update last_used_at
	profile, err := s.repo.Get(profileID)
	if err != nil {
		return err
	}
	if profile == nil {
		return nil // Profile doesn't exist, nothing to update
	}

	now := s.clock.Now().UTC()
	profile.LastUsedAt = now
	profile.UpdatedAt = now

	return s.repo.Save(profile)
}

// EndSession persists session state to the profile and clears the association.
// This is the single atomic save point that replaces scattered SaveX() calls.
func (s *Service) EndSession(ctx context.Context, sessionID string, state *persistence.SessionEndState) error {
	profileID := s.sessions.Get(sessionID)
	if profileID == "" {
		return nil // No profile associated
	}

	profile, err := s.repo.Get(persistence.ProfileID(profileID))
	if err != nil {
		return err
	}
	if profile == nil {
		s.sessions.Clear(sessionID)
		return nil // Profile doesn't exist
	}

	// Update profile with session end state
	if state != nil {
		if len(state.StorageState) > 0 {
			profile.StorageState = state.StorageState
		}
		// Limit tabs to prevent resource exhaustion
		if len(state.OpenTabs) > persistence.MaxRestoredTabs {
			state.OpenTabs = state.OpenTabs[:persistence.MaxRestoredTabs]
		}
		profile.OpenTabs = state.OpenTabs
	}

	now := s.clock.Now().UTC()
	profile.UpdatedAt = now

	// Single atomic save
	if err := s.repo.Save(profile); err != nil {
		return err
	}

	s.sessions.Clear(sessionID)
	return nil
}

// PersistSessionState saves storage state and tabs without ending the session.
// Used for mid-session saves (e.g., beforeunload, manual persist).
func (s *Service) PersistSessionState(profileID persistence.ProfileID, state *persistence.SessionEndState) error {
	profile, err := s.GetProfile(profileID)
	if err != nil {
		return err
	}

	if state != nil {
		if len(state.StorageState) > 0 {
			profile.StorageState = state.StorageState
		}
		if len(state.OpenTabs) > persistence.MaxRestoredTabs {
			state.OpenTabs = state.OpenTabs[:persistence.MaxRestoredTabs]
		}
		profile.OpenTabs = state.OpenTabs
	}

	now := s.clock.Now().UTC()
	profile.UpdatedAt = now
	profile.LastUsedAt = now

	return s.repo.Save(profile)
}

// SaveStorageState persists the storage state and bumps last_used_at.
func (s *Service) SaveStorageState(id persistence.ProfileID, storageState []byte) (*persistence.SessionProfile, error) {
	profile, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}

	if len(storageState) > 0 {
		profile.StorageState = storageState
	}
	now := s.clock.Now().UTC()
	profile.LastUsedAt = now
	profile.UpdatedAt = now

	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// SaveOpenTabs persists the tab state for session restoration.
func (s *Service) SaveOpenTabs(id persistence.ProfileID, tabs []persistence.TabState) (*persistence.SessionProfile, error) {
	profile, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}

	// Limit tabs to prevent resource exhaustion
	if len(tabs) > persistence.MaxRestoredTabs {
		tabs = tabs[:persistence.MaxRestoredTabs]
	}

	profile.OpenTabs = tabs
	profile.UpdatedAt = s.clock.Now().UTC()

	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// Touch updates the last_used_at timestamp.
func (s *Service) Touch(id persistence.ProfileID) (*persistence.SessionProfile, error) {
	profile, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now().UTC()
	profile.LastUsedAt = now
	profile.UpdatedAt = now

	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// AddHistoryEntry appends a history entry to the profile.
// Entries are stored newest-first. Pruning is applied automatically.
func (s *Service) AddHistoryEntry(id persistence.ProfileID, entry persistence.HistoryEntry) (*persistence.SessionProfile, error) {
	profile, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}

	// Get settings (use defaults if not set)
	settings := profile.HistorySettings
	if settings == nil {
		settings = persistence.DefaultHistorySettings()
	}

	// Prepend new entry (newest first)
	profile.History = append([]persistence.HistoryEntry{entry}, profile.History...)

	// Prune to maxEntries
	if settings.MaxEntries > 0 && len(profile.History) > settings.MaxEntries {
		profile.History = profile.History[:settings.MaxEntries]
	}

	// Prune by TTL
	profile.History = s.pruneHistoryByTTL(profile.History, settings)

	profile.UpdatedAt = s.clock.Now().UTC()

	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// ClearHistory removes all history entries from a profile.
func (s *Service) ClearHistory(id persistence.ProfileID) (*persistence.SessionProfile, error) {
	profile, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}

	profile.History = nil
	profile.UpdatedAt = s.clock.Now().UTC()

	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// DeleteHistoryEntry removes a single entry by ID.
func (s *Service) DeleteHistoryEntry(id persistence.ProfileID, entryID string) (*persistence.SessionProfile, error) {
	profile, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}

	// Find and remove the entry
	found := false
	newHistory := make([]persistence.HistoryEntry, 0, len(profile.History))
	for _, entry := range profile.History {
		if entry.ID == entryID {
			found = true
			continue
		}
		newHistory = append(newHistory, entry)
	}

	if !found {
		return nil, fmt.Errorf("history entry not found: %s", entryID)
	}

	profile.History = newHistory
	profile.UpdatedAt = s.clock.Now().UTC()

	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// UpdateHistorySettings updates the history configuration.
func (s *Service) UpdateHistorySettings(id persistence.ProfileID, settings *persistence.HistorySettings) (*persistence.SessionProfile, error) {
	profile, err := s.GetProfile(id)
	if err != nil {
		return nil, err
	}

	// Validate settings
	if err := ValidateHistorySettings(settings); err != nil {
		return nil, err
	}

	profile.HistorySettings = settings

	// Apply pruning if we have history and settings
	if len(profile.History) > 0 && settings != nil {
		// Prune to maxEntries
		if settings.MaxEntries > 0 && len(profile.History) > settings.MaxEntries {
			profile.History = profile.History[:settings.MaxEntries]
		}
		// Prune by TTL
		profile.History = s.pruneHistoryByTTL(profile.History, settings)
	}

	profile.UpdatedAt = s.clock.Now().UTC()

	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// GetHistoryWithPruning returns the history with TTL pruning applied (but doesn't save).
func (s *Service) GetHistoryWithPruning(id persistence.ProfileID) ([]persistence.HistoryEntry, *persistence.HistorySettings, error) {
	profile, err := s.GetProfile(id)
	if err != nil {
		return nil, nil, err
	}

	settings := profile.HistorySettings
	if settings == nil {
		settings = persistence.DefaultHistorySettings()
	}

	// Apply TTL pruning for display (doesn't persist)
	entries := s.pruneHistoryByTTL(profile.History, settings)

	return entries, settings, nil
}

// GetActiveSession returns the profile ID associated with a browser session.
func (s *Service) GetActiveSession(browserSessionID string) string {
	return s.sessions.Get(browserSessionID)
}

// SetActiveSession associates a browser session with a profile.
func (s *Service) SetActiveSession(browserSessionID, profileID string) {
	if browserSessionID != "" && profileID != "" {
		s.sessions.Set(browserSessionID, profileID)
	}
}

// ClearActiveSession removes the association between a browser session and its profile.
func (s *Service) ClearActiveSession(browserSessionID string) string {
	profileID := s.sessions.Get(browserSessionID)
	s.sessions.Clear(browserSessionID)
	return profileID
}

// GetSessionForProfile returns the browser session ID associated with a profile.
func (s *Service) GetSessionForProfile(profileID string) string {
	return s.sessions.GetByProfile(profileID)
}

// ClearSessionsForProfile removes all browser session associations for a given profile.
func (s *Service) ClearSessionsForProfile(profileID string) {
	s.sessions.ClearForProfile(profileID)
}

// generateProfileName creates a name like "Session N" based on existing profiles.
func (s *Service) generateProfileName() string {
	profiles, err := s.repo.List()
	if err != nil {
		return "Session"
	}
	return fmt.Sprintf("Session %d", len(profiles)+1)
}

// pruneHistoryByTTL removes entries older than the retention period.
func (s *Service) pruneHistoryByTTL(entries []persistence.HistoryEntry, settings *persistence.HistorySettings) []persistence.HistoryEntry {
	if settings == nil || settings.RetentionDays <= 0 || len(entries) == 0 {
		return entries
	}

	cutoff := s.clock.Now().AddDate(0, 0, -settings.RetentionDays)
	result := make([]persistence.HistoryEntry, 0, len(entries))
	for _, entry := range entries {
		t, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			// Keep entries with unparseable timestamps
			result = append(result, entry)
			continue
		}
		if t.After(cutoff) {
			result = append(result, entry)
		}
	}
	return result
}

// ActiveSessionRegistry tracks browser session to profile ID mappings.
type ActiveSessionRegistry struct {
	mu       sync.Mutex
	sessions map[string]string // browserSessionID -> profileID
}

// NewActiveSessionRegistry creates a new registry.
func NewActiveSessionRegistry() *ActiveSessionRegistry {
	return &ActiveSessionRegistry{
		sessions: make(map[string]string),
	}
}

// Set associates a browser session with a profile.
func (r *ActiveSessionRegistry) Set(browserSessionID, profileID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[browserSessionID] = profileID
}

// Get returns the profile ID for a browser session.
func (r *ActiveSessionRegistry) Get(browserSessionID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessions[browserSessionID]
}

// GetByProfile returns the browser session ID for a profile (reverse lookup).
func (r *ActiveSessionRegistry) GetByProfile(profileID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for sessionID, pid := range r.sessions {
		if pid == profileID {
			return sessionID
		}
	}
	return ""
}

// Clear removes the association for a browser session.
func (r *ActiveSessionRegistry) Clear(browserSessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, browserSessionID)
}

// ClearForProfile removes all associations for a profile.
func (r *ActiveSessionRegistry) ClearForProfile(profileID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for sessionID, pid := range r.sessions {
		if pid == profileID {
			delete(r.sessions, sessionID)
		}
	}
}

// =============================================================================
// Storage State Masking
// =============================================================================

// MaskedStorageState represents the storage state with httpOnly cookie values hidden.
// This is used for the API response to prevent exposing sensitive session cookies.
type MaskedStorageState struct {
	Cookies []MaskedCookie        `json:"cookies"`
	Origins []MaskedOrigin        `json:"origins"`
	Stats   MaskedStorageStats    `json:"stats"`
}

// MaskedCookie represents a cookie with optional value masking.
type MaskedCookie struct {
	Name        string  `json:"name"`
	Value       string  `json:"value"`
	ValueMasked bool    `json:"valueMasked"`
	Domain      string  `json:"domain"`
	Path        string  `json:"path"`
	Expires     float64 `json:"expires"`
	HttpOnly    bool    `json:"httpOnly"`
	Secure      bool    `json:"secure"`
	SameSite    string  `json:"sameSite"`
}

// MaskedOrigin represents localStorage for a specific origin.
type MaskedOrigin struct {
	Origin       string                   `json:"origin"`
	LocalStorage []MaskedLocalStorageItem `json:"localStorage"`
}

// MaskedLocalStorageItem represents a localStorage key-value pair.
type MaskedLocalStorageItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MaskedStorageStats provides summary statistics.
type MaskedStorageStats struct {
	CookieCount       int `json:"cookieCount"`
	LocalStorageCount int `json:"localStorageCount"`
	OriginCount       int `json:"originCount"`
}

// playwrightStorageStateInternal matches the Playwright storage state format for parsing.
type playwrightStorageStateInternal struct {
	Cookies []struct {
		Name     string  `json:"name"`
		Value    string  `json:"value"`
		Domain   string  `json:"domain"`
		Path     string  `json:"path"`
		Expires  float64 `json:"expires"`
		HttpOnly bool    `json:"httpOnly"`
		Secure   bool    `json:"secure"`
		SameSite string  `json:"sameSite"`
	} `json:"cookies"`
	Origins []struct {
		Origin       string `json:"origin"`
		LocalStorage []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"localStorage"`
	} `json:"origins"`
}

// MaskStorageState parses the raw storage state and masks httpOnly cookie values.
// This consolidates the masking logic from the handler into the service layer,
// ensuring consistent security behavior and testability.
func (s *Service) MaskStorageState(rawStorageState []byte) (*MaskedStorageState, error) {
	// Return empty result for nil or empty input
	if len(rawStorageState) == 0 {
		return &MaskedStorageState{
			Cookies: []MaskedCookie{},
			Origins: []MaskedOrigin{},
			Stats:   MaskedStorageStats{},
		}, nil
	}

	// Parse the Playwright storage state
	var pwState playwrightStorageStateInternal
	if err := json.Unmarshal(rawStorageState, &pwState); err != nil {
		return nil, fmt.Errorf("failed to parse storage state: %w", err)
	}

	// Build response with masked httpOnly cookie values
	cookies := make([]MaskedCookie, 0, len(pwState.Cookies))
	for _, c := range pwState.Cookies {
		cookie := MaskedCookie{
			Name:     c.Name,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires,
			HttpOnly: c.HttpOnly,
			Secure:   c.Secure,
			SameSite: c.SameSite,
		}
		if c.HttpOnly {
			cookie.Value = "[HIDDEN]"
			cookie.ValueMasked = true
		} else {
			cookie.Value = c.Value
			cookie.ValueMasked = false
		}
		cookies = append(cookies, cookie)
	}

	// Build origins with localStorage
	origins := make([]MaskedOrigin, 0, len(pwState.Origins))
	localStorageCount := 0
	for _, o := range pwState.Origins {
		items := make([]MaskedLocalStorageItem, 0, len(o.LocalStorage))
		for _, item := range o.LocalStorage {
			items = append(items, MaskedLocalStorageItem{
				Name:  item.Name,
				Value: item.Value,
			})
		}
		localStorageCount += len(items)
		origins = append(origins, MaskedOrigin{
			Origin:       o.Origin,
			LocalStorage: items,
		})
	}

	return &MaskedStorageState{
		Cookies: cookies,
		Origins: origins,
		Stats: MaskedStorageStats{
			CookieCount:       len(cookies),
			LocalStorageCount: localStorageCount,
			OriginCount:       len(origins),
		},
	}, nil
}

// NewServiceWithPath creates a service with a file-based repository at the given path.
// This is a convenience constructor for production use.
func NewServiceWithPath(root string, log *logrus.Logger) *Service {
	repo := persistence.NewFileRepository(root, log)
	return NewService(repo, log)
}

// NewServiceWithDefaultPath creates a service with the default data directory.
func NewServiceWithDefaultPath(log *logrus.Logger) *Service {
	root := os.Getenv("SESSION_PROFILES_ROOT")
	if root == "" {
		root = "data/session-profiles"
	}
	return NewServiceWithPath(root, log)
}
