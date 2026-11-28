// Package persistence provides data types and persistence interfaces for session profiles.
package persistence

import (
	"encoding/json"
	"time"
)

// ProfileID is a typed string for session profile identifiers.
type ProfileID string

// SessionProfile captures a persisted browser session state along with metadata.
// This is the aggregate root for session profile management.
type SessionProfile struct {
	ID              ProfileID        `json:"id"`
	Name            string           `json:"name"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	LastUsedAt      time.Time        `json:"last_used_at"`
	StorageState    json.RawMessage  `json:"storage_state,omitempty"`
	BrowserProfile  *BrowserProfile  `json:"browser_profile,omitempty"`
	History         []HistoryEntry   `json:"history,omitempty"`          // Navigation history entries (newest first)
	HistorySettings *HistorySettings `json:"history_settings,omitempty"` // History capture configuration
	OpenTabs        []TabState       `json:"open_tabs,omitempty"`        // Tabs to restore on session resume
}

// BrowserProfile contains optional browser identity and behavior settings.
// Effects depend on the selected backend and do not guarantee that a site will
// accept or classify the session as human-operated.
type BrowserProfile struct {
	Preset           string                 `json:"preset,omitempty"`            // legacy preset names: stealth, balanced, fast, none
	Fingerprint      *FingerprintSettings   `json:"fingerprint,omitempty"`       // Browser identity settings
	Behavior         *BehaviorSettings      `json:"behavior,omitempty"`          // Configurable interaction timing and movement settings
	AntiDetection    *AntiDetectionSettings `json:"anti_detection,omitempty"`    // Legacy API name for optional signal adjustments; no bypass guarantee
	Proxy            *ProxySettings         `json:"proxy,omitempty"`             // Proxy configuration
	ExtraHeaders     map[string]string      `json:"extra_headers,omitempty"`     // Custom HTTP headers sent with every request
	MotionPreference string                 `json:"motion_preference,omitempty"` // no-preference, reduce
	InteractionState string                 `json:"interaction_state,omitempty"` // rest, hover, focus-visible, pressed, disabled
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

// BehaviorSettings controls configurable interaction timing and movement patterns.
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

// AntiDetectionSettings is a legacy field group for requested browser-signal
// adjustments. Support and effect vary by backend; these settings do not
// establish a detection pass or bypass guarantee.
type AntiDetectionSettings struct {
	// Browser launch args
	DisableAutomationControlled bool `json:"disable_automation_controlled,omitempty"` // --disable-blink-features=AutomationControlled
	DisableWebRTC               bool `json:"disable_webrtc,omitempty"`                // Request WebRTC disablement; network behavior needs separate validation

	// Navigator property patches
	PatchNavigatorWebdriver bool `json:"patch_navigator_webdriver,omitempty"` // Request a navigator.webdriver adjustment
	PatchNavigatorPlugins   bool `json:"patch_navigator_plugins,omitempty"`   // Request a plugins-array adjustment
	PatchNavigatorLanguages bool `json:"patch_navigator_languages,omitempty"` // Request a languages adjustment
	PatchWebGL              bool `json:"patch_webgl,omitempty"`               // Request a WebGL renderer/vendor adjustment
	PatchCanvas             bool `json:"patch_canvas,omitempty"`              // Request a canvas-output adjustment
	PatchAudioContext       bool `json:"patch_audio_context,omitempty"`       // Request an AudioContext-output adjustment
	HeadlessDetectionBypass bool `json:"headless_detection_bypass,omitempty"` // Legacy flag name; no detection-bypass guarantee

	// Additional fingerprinting protection
	PatchFonts            bool `json:"patch_fonts,omitempty"`             // Request a font-enumeration adjustment
	PatchScreenProperties bool `json:"patch_screen_properties,omitempty"` // Request screen dimensions matching the viewport
	PatchBatteryAPI       bool `json:"patch_battery_api,omitempty"`       // Request a battery-status adjustment
	PatchConnectionAPI    bool `json:"patch_connection_api,omitempty"`    // Request a connection-type adjustment

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

// MaxRestoredTabs limits how many tabs can be restored to prevent resource exhaustion.
const MaxRestoredTabs = 20

// LimitRestoredTabs applies the single persistence-owned cap used when saving
// and restoring browser tabs. It preserves order and all supplied values.
func LimitRestoredTabs(tabs []TabState) []TabState {
	if len(tabs) > MaxRestoredTabs {
		return tabs[:MaxRestoredTabs]
	}
	return tabs
}

// TabState captures a single tab's state for restoration when resuming a session.
type TabState struct {
	URL         string `json:"url"`                    // URL to navigate to
	Title       string `json:"title,omitempty"`        // Tab title (for display before load)
	IsActive    bool   `json:"is_active,omitempty"`    // Whether this was the active tab
	Order       int    `json:"order"`                  // Creation order for consistent tab ordering
	OpenerOrder *int   `json:"opener_order,omitempty"` // Saved order of the tab that opened this popup
}

// SessionEndState bundles all data to persist when a session ends.
// This eliminates scattered construction code in handlers.
type SessionEndState struct {
	StorageState json.RawMessage // Browser cookies and localStorage
	OpenTabs     []TabState      // Tabs to restore on next session
}
