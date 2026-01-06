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
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	LastUsedAt      time.Time        `json:"last_used_at"`
	StorageState    json.RawMessage  `json:"storage_state,omitempty"`
	BrowserProfile  *BrowserProfile  `json:"browser_profile,omitempty"`
	History         []HistoryEntry   `json:"history,omitempty"`          // Navigation history entries (newest first)
	HistorySettings *HistorySettings `json:"history_settings,omitempty"` // History capture configuration
}

// BrowserProfile contains anti-detection and fingerprint settings for browser automation.
type BrowserProfile struct {
	Preset        string                 `json:"preset,omitempty"`         // stealth, balanced, fast, none
	Fingerprint   *FingerprintSettings   `json:"fingerprint,omitempty"`    // Browser identity settings
	Behavior      *BehaviorSettings      `json:"behavior,omitempty"`       // Human-like behavior settings
	AntiDetection *AntiDetectionSettings `json:"anti_detection,omitempty"` // Bot detection bypass settings
	Proxy         *ProxySettings         `json:"proxy,omitempty"`          // Proxy configuration
	ExtraHeaders  map[string]string      `json:"extra_headers,omitempty"`  // Custom HTTP headers sent with every request
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
	// Typing behavior - inter-keystroke delays
	TypingDelayMin int `json:"typing_delay_min,omitempty"` // Min ms between keystrokes
	TypingDelayMax int `json:"typing_delay_max,omitempty"` // Max ms between keystrokes

	// Typing behavior - pre-typing delay (pause before starting to type)
	TypingStartDelayMin int `json:"typing_start_delay_min,omitempty"` // Min ms to wait before starting to type
	TypingStartDelayMax int `json:"typing_start_delay_max,omitempty"` // Max ms to wait before starting to type

	// Typing behavior - paste threshold (paste long text instead of typing)
	TypingPasteThreshold int `json:"typing_paste_threshold,omitempty"` // Paste if text > this length (0 = always type, -1 = always paste)

	// Typing behavior - enhanced variance (simulate human typing patterns)
	TypingVarianceEnabled bool `json:"typing_variance_enabled,omitempty"` // Enable digraph/shift/symbol variance

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
	PatchNavigatorWebdriver bool `json:"patch_navigator_webdriver,omitempty"` // Remove navigator.webdriver
	PatchNavigatorPlugins   bool `json:"patch_navigator_plugins,omitempty"`   // Spoof plugins array
	PatchNavigatorLanguages bool `json:"patch_navigator_languages,omitempty"` // Ensure consistent languages
	PatchWebGL              bool `json:"patch_webgl,omitempty"`               // Spoof WebGL renderer/vendor
	PatchCanvas             bool `json:"patch_canvas,omitempty"`              // Add noise to canvas fingerprint
	PatchAudioContext       bool `json:"patch_audio_context,omitempty"`       // Add noise to AudioContext to prevent audio fingerprinting
	HeadlessDetectionBypass bool `json:"headless_detection_bypass,omitempty"` // Bypass headless detection

	// Additional fingerprinting protection
	PatchFonts            bool `json:"patch_fonts,omitempty"`             // Spoof font enumeration to return common fonts only
	PatchScreenProperties bool `json:"patch_screen_properties,omitempty"` // Spoof screen dimensions to match viewport
	PatchBatteryAPI       bool `json:"patch_battery_api,omitempty"`       // Return consistent battery status
	PatchConnectionAPI    bool `json:"patch_connection_api,omitempty"`    // Spoof network connection type

	// Ad blocking mode: "none", "ads_only", or "ads_and_tracking"
	AdBlockingMode string `json:"ad_blocking_mode,omitempty"`
}

// ProxySettings controls routing browser traffic through a proxy server.
type ProxySettings struct {
	Enabled  bool   `json:"enabled,omitempty"`  // Whether proxy is enabled
	Server   string `json:"server,omitempty"`   // Proxy URL (e.g., "http://proxy:8080" or "socks5://proxy:1080")
	Bypass   string `json:"bypass,omitempty"`   // Comma-separated domains to bypass proxy
	Username string `json:"username,omitempty"` // Proxy authentication username
	Password string `json:"password,omitempty"` // Proxy authentication password
}

// HistoryEntry represents a single navigation event in the browser history.
type HistoryEntry struct {
	ID        string `json:"id"`                  // Unique identifier for this entry
	URL       string `json:"url"`                 // Page URL
	Title     string `json:"title"`               // Page title at time of navigation
	Timestamp string `json:"timestamp"`           // ISO 8601 timestamp when navigation occurred
	Thumbnail string `json:"thumbnail,omitempty"` // Base64-encoded JPEG thumbnail (~150x100)
}

// HistorySettings configures history capture behavior.
type HistorySettings struct {
	MaxEntries        int  `json:"maxEntries"`        // Maximum number of entries to retain (default: 100)
	RetentionDays     int  `json:"retentionDays"`     // TTL in days - entries older than this are pruned (default: 30, 0 = no TTL)
	CaptureThumbnails bool `json:"captureThumbnails"` // Whether to capture thumbnails (default: true)
}

// DefaultHistorySettings returns the default history configuration.
func DefaultHistorySettings() *HistorySettings {
	return &HistorySettings{
		MaxEntries:        100,
		RetentionDays:     30,
		CaptureThumbnails: true,
	}
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
				TypingDelayMin:        50,
				TypingDelayMax:        150,
				TypingStartDelayMin:   100,
				TypingStartDelayMax:   300,
				TypingPasteThreshold:  200, // Paste text longer than 200 chars
				TypingVarianceEnabled: true,
				MouseMovementStyle:    "natural",
				MouseJitterAmount:     2,
				ClickDelayMin:         100,
				ClickDelayMax:         300,
				ScrollStyle:           "smooth",
				MicroPauseEnabled:     true,
				MicroPauseMinMs:       200,
				MicroPauseMaxMs:       800,
				MicroPauseFrequency:   0.15,
			},
			AntiDetection: &AntiDetectionSettings{
				DisableAutomationControlled: true,
				DisableWebRTC:               true,
				PatchNavigatorWebdriver:     true,
				PatchNavigatorPlugins:       true,
				PatchNavigatorLanguages:     true,
				PatchWebGL:                  true,
				PatchCanvas:                 true,
				PatchAudioContext:           true,
				HeadlessDetectionBypass:     true,
				PatchFonts:                  true,
				PatchScreenProperties:       true,
				PatchBatteryAPI:             true,
				PatchConnectionAPI:          true,
				AdBlockingMode:              "ads_and_tracking",
			},
		}
	case PresetBalanced:
		return &BrowserProfile{
			Preset: PresetBalanced,
			Behavior: &BehaviorSettings{
				TypingDelayMin:        30,
				TypingDelayMax:        80,
				TypingStartDelayMin:   50,
				TypingStartDelayMax:   150,
				TypingPasteThreshold:  150, // Paste text longer than 150 chars
				TypingVarianceEnabled: true,
				MouseMovementStyle:    "bezier",
				ClickDelayMin:         50,
				ClickDelayMax:         150,
				MicroPauseEnabled:     true,
				MicroPauseMinMs:       100,
				MicroPauseMaxMs:       400,
				MicroPauseFrequency:   0.08,
			},
			AntiDetection: &AntiDetectionSettings{
				DisableAutomationControlled: true,
				PatchNavigatorWebdriver:     true,
				PatchAudioContext:           true,
				PatchFonts:                  true,
				PatchScreenProperties:       true,
				AdBlockingMode:              "ads_and_tracking",
			},
		}
	case PresetFast:
		return &BrowserProfile{
			Preset: PresetFast,
			Behavior: &BehaviorSettings{
				TypingDelayMin:        10,
				TypingDelayMax:        30,
				TypingStartDelayMin:   10,
				TypingStartDelayMax:   30,
				TypingPasteThreshold:  100, // Paste text longer than 100 chars
				TypingVarianceEnabled: true,
				MouseMovementStyle:    "linear",
				ClickDelayMin:         20,
				ClickDelayMax:         50,
				MicroPauseEnabled:     false,
			},
			AntiDetection: &AntiDetectionSettings{
				DisableAutomationControlled: true,
				PatchNavigatorWebdriver:     true,
				AdBlockingMode:              "ads_and_tracking",
			},
		}
	case PresetNone:
		return &BrowserProfile{
			Preset: PresetNone,
			AntiDetection: &AntiDetectionSettings{
				AdBlockingMode: "none",
			},
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
		if err := ValidateBrowserProfile(browserProfile); err != nil {
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

// ValidateBrowserProfile checks that browser profile settings are within valid ranges.
// Exported for use by the execution service during workflow execution.
func ValidateBrowserProfile(bp *BrowserProfile) error {
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
		// Typing delay validation
		if bh.TypingDelayMin < 0 || bh.TypingDelayMin > 5000 {
			return fmt.Errorf("typing_delay_min must be between 0 and 5000")
		}
		if bh.TypingDelayMax < 0 || bh.TypingDelayMax > 5000 {
			return fmt.Errorf("typing_delay_max must be between 0 and 5000")
		}
		if bh.TypingDelayMin > bh.TypingDelayMax && bh.TypingDelayMax > 0 {
			return fmt.Errorf("typing_delay_min cannot exceed typing_delay_max")
		}

		// Typing start delay validation
		if bh.TypingStartDelayMin < 0 || bh.TypingStartDelayMin > 5000 {
			return fmt.Errorf("typing_start_delay_min must be between 0 and 5000")
		}
		if bh.TypingStartDelayMax < 0 || bh.TypingStartDelayMax > 5000 {
			return fmt.Errorf("typing_start_delay_max must be between 0 and 5000")
		}
		if bh.TypingStartDelayMin > bh.TypingStartDelayMax && bh.TypingStartDelayMax > 0 {
			return fmt.Errorf("typing_start_delay_min cannot exceed typing_start_delay_max")
		}

		// Typing paste threshold validation (-1 = always paste, 0 = always type, >0 = paste if text > threshold)
		if bh.TypingPasteThreshold < -1 || bh.TypingPasteThreshold > 10000 {
			return fmt.Errorf("typing_paste_threshold must be between -1 and 10000")
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

	// Validate anti-detection settings
	if ad := bp.AntiDetection; ad != nil {
		validAdBlockingModes := map[string]bool{"": true, "none": true, "ads_only": true, "ads_and_tracking": true}
		if !validAdBlockingModes[ad.AdBlockingMode] {
			return fmt.Errorf("ad_blocking_mode must be none, ads_only, or ads_and_tracking")
		}
	}

	// Validate proxy settings
	if proxy := bp.Proxy; proxy != nil {
		if proxy.Enabled && proxy.Server == "" {
			return fmt.Errorf("proxy server is required when proxy is enabled")
		}
		if proxy.Server != "" {
			if !strings.HasPrefix(proxy.Server, "http://") &&
				!strings.HasPrefix(proxy.Server, "https://") &&
				!strings.HasPrefix(proxy.Server, "socks5://") {
				return fmt.Errorf("proxy server must start with http://, https://, or socks5://")
			}
		}
		// If password is set, username should also be set
		if proxy.Password != "" && proxy.Username == "" {
			return fmt.Errorf("proxy username is required when password is set")
		}
	}

	// Validate extra headers
	if err := validateExtraHeaders(bp.ExtraHeaders); err != nil {
		return err
	}

	return nil
}

// blockedHeaders contains HTTP headers that cannot be set via extra_headers
// because they may break routing or conflict with other browser features.
var blockedHeaders = map[string]bool{
	"host":           true, // Can break routing
	"content-length": true, // Managed by browser
	"cookie":         true, // Use storage_state instead
}

// validateExtraHeaders checks that no blocked headers are included.
func validateExtraHeaders(headers map[string]string) error {
	for k := range headers {
		if blockedHeaders[strings.ToLower(k)] {
			return fmt.Errorf("header %q cannot be set via extra_headers", k)
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

// GetSessionForProfile returns the browser session ID associated with a profile.
// Returns empty string if no active session exists for this profile.
// This is the reverse lookup of GetActiveSession.
func (s *SessionProfileStore) GetSessionForProfile(profileID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sessionID, pid := range s.activeSessions {
		if pid == profileID {
			return sessionID
		}
	}
	return ""
}

// ========================================================================
// Browser History Management
// ========================================================================

// AddHistoryEntry appends a history entry to the profile, pruning if necessary.
// Entries are stored newest-first. Pruning by max entries and TTL is applied automatically.
func (s *SessionProfileStore) AddHistoryEntry(id string, entry HistoryEntry) (*SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := s.loadProfileLocked(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	// Get settings (use defaults if not set)
	settings := profile.HistorySettings
	if settings == nil {
		settings = DefaultHistorySettings()
	}

	// Prepend new entry (newest first)
	profile.History = append([]HistoryEntry{entry}, profile.History...)

	// Prune to maxEntries
	if settings.MaxEntries > 0 && len(profile.History) > settings.MaxEntries {
		profile.History = profile.History[:settings.MaxEntries]
	}

	// Prune by TTL
	profile.History = pruneHistoryByTTL(profile.History, settings)

	profile.UpdatedAt = time.Now().UTC()

	if err := s.saveProfileLocked(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// ClearHistory removes all history entries from a profile.
func (s *SessionProfileStore) ClearHistory(id string) (*SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := s.loadProfileLocked(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	profile.History = nil
	profile.UpdatedAt = time.Now().UTC()

	if err := s.saveProfileLocked(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// DeleteHistoryEntry removes a single entry by ID.
func (s *SessionProfileStore) DeleteHistoryEntry(id, entryID string) (*SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := s.loadProfileLocked(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	// Find and remove the entry
	found := false
	newHistory := make([]HistoryEntry, 0, len(profile.History))
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
	profile.UpdatedAt = time.Now().UTC()

	if err := s.saveProfileLocked(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// UpdateHistorySettings updates the history configuration.
// Also triggers pruning if maxEntries is lowered.
func (s *SessionProfileStore) UpdateHistorySettings(id string, settings *HistorySettings) (*SessionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := s.loadProfileLocked(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	// Validate settings
	if settings != nil {
		if settings.MaxEntries < 0 || settings.MaxEntries > 10000 {
			return nil, fmt.Errorf("max_entries must be between 0 and 10000")
		}
		if settings.RetentionDays < 0 || settings.RetentionDays > 3650 {
			return nil, fmt.Errorf("retention_days must be between 0 and 3650")
		}
	}

	profile.HistorySettings = settings

	// Apply pruning if we have history and settings
	if len(profile.History) > 0 && settings != nil {
		// Prune to maxEntries
		if settings.MaxEntries > 0 && len(profile.History) > settings.MaxEntries {
			profile.History = profile.History[:settings.MaxEntries]
		}
		// Prune by TTL
		profile.History = pruneHistoryByTTL(profile.History, settings)
	}

	profile.UpdatedAt = time.Now().UTC()

	if err := s.saveProfileLocked(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// GetHistoryWithPruning returns the history with TTL pruning applied (but doesn't save).
// Use this for read-only access where you want to filter expired entries.
func (s *SessionProfileStore) GetHistoryWithPruning(id string) ([]HistoryEntry, *HistorySettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := s.loadProfileLocked(strings.TrimSpace(id))
	if err != nil {
		return nil, nil, err
	}

	settings := profile.HistorySettings
	if settings == nil {
		settings = DefaultHistorySettings()
	}

	// Apply TTL pruning for display (doesn't persist)
	entries := pruneHistoryByTTL(profile.History, settings)

	return entries, settings, nil
}

// pruneHistoryByTTL removes entries older than the retention period.
func pruneHistoryByTTL(entries []HistoryEntry, settings *HistorySettings) []HistoryEntry {
	if settings == nil || settings.RetentionDays <= 0 || len(entries) == 0 {
		return entries
	}

	cutoff := time.Now().AddDate(0, 0, -settings.RetentionDays)
	result := make([]HistoryEntry, 0, len(entries))
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
