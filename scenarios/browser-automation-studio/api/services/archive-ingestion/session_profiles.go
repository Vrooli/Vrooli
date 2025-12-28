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
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	LastUsedAt     time.Time       `json:"last_used_at"`
	StorageState   json.RawMessage `json:"storage_state,omitempty"`
	BrowserProfile *BrowserProfile `json:"browser_profile,omitempty"`
}

// BrowserProfile contains anti-detection and fingerprint settings for browser automation.
type BrowserProfile struct {
	Preset        string                 `json:"preset,omitempty"`        // stealth, balanced, fast, none
	Fingerprint   *FingerprintSettings   `json:"fingerprint,omitempty"`   // Browser identity settings
	Behavior      *BehaviorSettings      `json:"behavior,omitempty"`      // Human-like behavior settings
	AntiDetection *AntiDetectionSettings `json:"anti_detection,omitempty"` // Bot detection bypass settings
}

// FingerprintSettings controls browser identity and device characteristics.
type FingerprintSettings struct {
	// Viewport dimensions
	ViewportWidth  int `json:"viewport_width,omitempty"`
	ViewportHeight int `json:"viewport_height,omitempty"`

	// Device characteristics
	DeviceScaleFactor   float64 `json:"device_scale_factor,omitempty"`
	HardwareConcurrency int     `json:"hardware_concurrency,omitempty"` // CPU cores to report
	DeviceMemory        int     `json:"device_memory,omitempty"`        // GB of RAM to report

	// Browser identity
	UserAgent       string `json:"user_agent,omitempty"`        // Custom user agent string
	UserAgentPreset string `json:"user_agent_preset,omitempty"` // chrome-win, chrome-mac, firefox-linux, etc.

	// Locale and timezone
	Locale     string `json:"locale,omitempty"`      // e.g., "en-US"
	TimezoneID string `json:"timezone_id,omitempty"` // e.g., "America/New_York"

	// Geolocation
	GeolocationEnabled bool    `json:"geolocation_enabled,omitempty"`
	Latitude           float64 `json:"latitude,omitempty"`
	Longitude          float64 `json:"longitude,omitempty"`
	Accuracy           float64 `json:"accuracy,omitempty"` // meters

	// Display
	ColorScheme string `json:"color_scheme,omitempty"` // light, dark, no-preference
}

// BehaviorSettings controls human-like interaction patterns.
type BehaviorSettings struct {
	// Typing behavior
	TypingDelayMin int `json:"typing_delay_min,omitempty"` // Min ms between keystrokes
	TypingDelayMax int `json:"typing_delay_max,omitempty"` // Max ms between keystrokes

	// Mouse movement
	MouseMovementStyle string  `json:"mouse_movement_style,omitempty"` // linear, bezier, natural
	MouseJitterAmount  float64 `json:"mouse_jitter_amount,omitempty"`  // Pixels of random movement

	// Click behavior
	ClickDelayMin int `json:"click_delay_min,omitempty"` // Min ms before clicking
	ClickDelayMax int `json:"click_delay_max,omitempty"` // Max ms before clicking

	// Scroll behavior
	ScrollStyle    string `json:"scroll_style,omitempty"`     // smooth, stepped
	ScrollSpeedMin int    `json:"scroll_speed_min,omitempty"` // Min pixels per step
	ScrollSpeedMax int    `json:"scroll_speed_max,omitempty"` // Max pixels per step

	// Random micro-pauses between actions
	MicroPauseEnabled   bool    `json:"micro_pause_enabled,omitempty"`
	MicroPauseMinMs     int     `json:"micro_pause_min_ms,omitempty"`
	MicroPauseMaxMs     int     `json:"micro_pause_max_ms,omitempty"`
	MicroPauseFrequency float64 `json:"micro_pause_frequency,omitempty"` // 0.0-1.0
}

// AntiDetectionSettings controls bot detection bypass techniques.
type AntiDetectionSettings struct {
	// Browser launch args
	DisableAutomationControlled bool `json:"disable_automation_controlled,omitempty"` // --disable-blink-features=AutomationControlled
	DisableWebRTC               bool `json:"disable_webrtc,omitempty"`                // Prevent IP leak via WebRTC

	// Navigator property patches
	PatchNavigatorWebdriver  bool `json:"patch_navigator_webdriver,omitempty"`  // Remove navigator.webdriver
	PatchNavigatorPlugins    bool `json:"patch_navigator_plugins,omitempty"`    // Spoof plugins array
	PatchNavigatorLanguages  bool `json:"patch_navigator_languages,omitempty"`  // Ensure consistent languages
	PatchWebGL               bool `json:"patch_webgl,omitempty"`                // Spoof WebGL renderer/vendor
	PatchCanvas              bool `json:"patch_canvas,omitempty"`               // Add noise to canvas fingerprint
	HeadlessDetectionBypass  bool `json:"headless_detection_bypass,omitempty"`  // Bypass headless detection
}

// Preset names for browser profiles
const (
	PresetStealth  = "stealth"
	PresetBalanced = "balanced"
	PresetFast     = "fast"
	PresetNone     = "none"
)

// GetPresetBrowserProfile returns the default settings for a given preset name.
// Returns nil for "none" preset or unknown preset names.
func GetPresetBrowserProfile(preset string) *BrowserProfile {
	switch preset {
	case PresetStealth:
		return &BrowserProfile{
			Preset: PresetStealth,
			Fingerprint: &FingerprintSettings{
				DeviceScaleFactor:   1,
				HardwareConcurrency: 4,
				DeviceMemory:        8,
			},
			Behavior: &BehaviorSettings{
				TypingDelayMin:      50,
				TypingDelayMax:      150,
				MouseMovementStyle:  "natural",
				MouseJitterAmount:   2,
				ClickDelayMin:       100,
				ClickDelayMax:       300,
				ScrollStyle:         "smooth",
				MicroPauseEnabled:   true,
				MicroPauseMinMs:     200,
				MicroPauseMaxMs:     800,
				MicroPauseFrequency: 0.15,
			},
			AntiDetection: &AntiDetectionSettings{
				DisableAutomationControlled: true,
				DisableWebRTC:               true,
				PatchNavigatorWebdriver:     true,
				PatchNavigatorPlugins:       true,
				PatchNavigatorLanguages:     true,
				PatchWebGL:                  true,
				PatchCanvas:                 true,
				HeadlessDetectionBypass:     true,
			},
		}
	case PresetBalanced:
		return &BrowserProfile{
			Preset: PresetBalanced,
			Behavior: &BehaviorSettings{
				TypingDelayMin:      30,
				TypingDelayMax:      80,
				MouseMovementStyle:  "bezier",
				ClickDelayMin:       50,
				ClickDelayMax:       150,
				MicroPauseEnabled:   true,
				MicroPauseMinMs:     100,
				MicroPauseMaxMs:     400,
				MicroPauseFrequency: 0.08,
			},
			AntiDetection: &AntiDetectionSettings{
				DisableAutomationControlled: true,
				PatchNavigatorWebdriver:     true,
			},
		}
	case PresetFast:
		return &BrowserProfile{
			Preset: PresetFast,
			Behavior: &BehaviorSettings{
				TypingDelayMin:     10,
				TypingDelayMax:     30,
				MouseMovementStyle: "linear",
				ClickDelayMin:      20,
				ClickDelayMax:      50,
				MicroPauseEnabled:  false,
			},
			AntiDetection: &AntiDetectionSettings{
				DisableAutomationControlled: true,
				PatchNavigatorWebdriver:     true,
			},
		}
	case PresetNone:
		return &BrowserProfile{
			Preset: PresetNone,
		}
	default:
		return nil
	}
}

// SessionProfileStore manages persisted session profiles on disk.
// It also tracks active browser sessions to their associated profiles in-memory.
type SessionProfileStore struct {
	root           string
	log            *logrus.Logger
	mu             sync.Mutex
	activeSessions map[string]string // browserSessionID -> profileID
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
		root:           root,
		log:            log,
		activeSessions: make(map[string]string),
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

// UpdateBrowserProfile updates the browser profile settings for anti-detection and fingerprinting.
func (s *SessionProfileStore) UpdateBrowserProfile(id string, browserProfile *BrowserProfile) (*SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := s.loadProfileLocked(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	// Validate browser profile if provided
	if browserProfile != nil {
		if err := validateBrowserProfile(browserProfile); err != nil {
			return nil, fmt.Errorf("invalid browser profile: %w", err)
		}
	}

	profile.BrowserProfile = browserProfile
	profile.UpdatedAt = time.Now().UTC()

	if err := s.saveProfileLocked(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// validateBrowserProfile checks that browser profile settings are within valid ranges.
func validateBrowserProfile(bp *BrowserProfile) error {
	if bp == nil {
		return nil
	}

	// Validate preset
	validPresets := map[string]bool{"": true, "stealth": true, "balanced": true, "fast": true, "none": true}
	if !validPresets[bp.Preset] {
		return fmt.Errorf("invalid preset: %s (must be stealth, balanced, fast, or none)", bp.Preset)
	}

	// Validate fingerprint settings
	if fp := bp.Fingerprint; fp != nil {
		if fp.ViewportWidth < 0 || fp.ViewportWidth > 7680 {
			return fmt.Errorf("viewport_width must be between 0 and 7680")
		}
		if fp.ViewportHeight < 0 || fp.ViewportHeight > 4320 {
			return fmt.Errorf("viewport_height must be between 0 and 4320")
		}
		if fp.DeviceScaleFactor < 0 || fp.DeviceScaleFactor > 5 {
			return fmt.Errorf("device_scale_factor must be between 0 and 5")
		}
		if fp.HardwareConcurrency < 0 || fp.HardwareConcurrency > 128 {
			return fmt.Errorf("hardware_concurrency must be between 0 and 128")
		}
		if fp.DeviceMemory < 0 || fp.DeviceMemory > 512 {
			return fmt.Errorf("device_memory must be between 0 and 512")
		}
		if fp.Latitude < -90 || fp.Latitude > 90 {
			return fmt.Errorf("latitude must be between -90 and 90")
		}
		if fp.Longitude < -180 || fp.Longitude > 180 {
			return fmt.Errorf("longitude must be between -180 and 180")
		}
		validColorSchemes := map[string]bool{"": true, "light": true, "dark": true, "no-preference": true}
		if !validColorSchemes[fp.ColorScheme] {
			return fmt.Errorf("color_scheme must be light, dark, or no-preference")
		}
	}

	// Validate behavior settings
	if bh := bp.Behavior; bh != nil {
		if bh.TypingDelayMin < 0 || bh.TypingDelayMin > 5000 {
			return fmt.Errorf("typing_delay_min must be between 0 and 5000")
		}
		if bh.TypingDelayMax < 0 || bh.TypingDelayMax > 5000 {
			return fmt.Errorf("typing_delay_max must be between 0 and 5000")
		}
		if bh.TypingDelayMin > bh.TypingDelayMax && bh.TypingDelayMax > 0 {
			return fmt.Errorf("typing_delay_min cannot exceed typing_delay_max")
		}
		validMouseStyles := map[string]bool{"": true, "linear": true, "bezier": true, "natural": true}
		if !validMouseStyles[bh.MouseMovementStyle] {
			return fmt.Errorf("mouse_movement_style must be linear, bezier, or natural")
		}
		if bh.MouseJitterAmount < 0 || bh.MouseJitterAmount > 100 {
			return fmt.Errorf("mouse_jitter_amount must be between 0 and 100")
		}
		if bh.ClickDelayMin < 0 || bh.ClickDelayMin > 5000 {
			return fmt.Errorf("click_delay_min must be between 0 and 5000")
		}
		if bh.ClickDelayMax < 0 || bh.ClickDelayMax > 5000 {
			return fmt.Errorf("click_delay_max must be between 0 and 5000")
		}
		if bh.ClickDelayMin > bh.ClickDelayMax && bh.ClickDelayMax > 0 {
			return fmt.Errorf("click_delay_min cannot exceed click_delay_max")
		}
		validScrollStyles := map[string]bool{"": true, "smooth": true, "stepped": true}
		if !validScrollStyles[bh.ScrollStyle] {
			return fmt.Errorf("scroll_style must be smooth or stepped")
		}
		if bh.MicroPauseFrequency < 0 || bh.MicroPauseFrequency > 1 {
			return fmt.Errorf("micro_pause_frequency must be between 0 and 1")
		}
		if bh.MicroPauseMinMs < 0 || bh.MicroPauseMinMs > 10000 {
			return fmt.Errorf("micro_pause_min_ms must be between 0 and 10000")
		}
		if bh.MicroPauseMaxMs < 0 || bh.MicroPauseMaxMs > 10000 {
			return fmt.Errorf("micro_pause_max_ms must be between 0 and 10000")
		}
	}

	return nil
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

// ========================================================================
// Active Session Tracking (in-memory)
// ========================================================================

// SetActiveSession associates a browser session with a profile.
// This is used to track which profile should receive storage state updates
// when a recording session is closed.
func (s *SessionProfileStore) SetActiveSession(browserSessionID, profileID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if browserSessionID != "" && profileID != "" {
		s.activeSessions[browserSessionID] = profileID
	}
}

// GetActiveSession returns the profile ID associated with a browser session.
// Returns empty string if no association exists.
func (s *SessionProfileStore) GetActiveSession(browserSessionID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeSessions[browserSessionID]
}

// ClearActiveSession removes the association between a browser session and its profile.
// Returns the profile ID that was cleared (for logging/persistence).
func (s *SessionProfileStore) ClearActiveSession(browserSessionID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	profileID := s.activeSessions[browserSessionID]
	delete(s.activeSessions, browserSessionID)
	return profileID
}

// ClearSessionsForProfile removes all browser session associations for a given profile.
// Used when deleting a profile to clean up any stale references.
func (s *SessionProfileStore) ClearSessionsForProfile(profileID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sessionID, pid := range s.activeSessions {
		if pid == profileID {
			delete(s.activeSessions, sessionID)
		}
	}
}
