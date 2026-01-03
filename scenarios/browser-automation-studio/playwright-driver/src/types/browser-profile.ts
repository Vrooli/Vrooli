/**
 * Browser Profile Types
 *
 * Type definitions for anti-detection browser profiles.
 * These types mirror the Go types in api/services/archive-ingestion/session_profiles.go
 */

/**
 * Fingerprint settings control browser identity and device characteristics.
 */
export interface FingerprintSettings {
  // Viewport dimensions
  viewport_width?: number;
  viewport_height?: number;

  // Device characteristics
  device_scale_factor?: number;
  hardware_concurrency?: number; // CPU cores to report
  device_memory?: number; // GB of RAM to report

  // Browser identity
  user_agent?: string; // Custom user agent string
  user_agent_preset?: UserAgentPreset; // Predefined user agent

  // Locale and timezone
  locale?: string; // e.g., "en-US"
  timezone_id?: string; // e.g., "America/New_York"

  // Geolocation
  geolocation_enabled?: boolean;
  latitude?: number;
  longitude?: number;
  accuracy?: number; // meters

  // Display
  color_scheme?: 'light' | 'dark' | 'no-preference';
}

/**
 * Predefined user agent presets.
 */
export type UserAgentPreset =
  | 'chrome-win'
  | 'chrome-mac'
  | 'chrome-linux'
  | 'firefox-win'
  | 'firefox-mac'
  | 'firefox-linux'
  | 'safari-mac'
  | 'edge-win';

/**
 * Behavior settings control human-like interaction patterns.
 */
export interface BehaviorSettings {
  // Typing behavior - inter-keystroke delays
  typing_delay_min?: number; // Min ms between keystrokes
  typing_delay_max?: number; // Max ms between keystrokes

  // Typing behavior - pre-typing delay (pause before starting to type)
  typing_start_delay_min?: number; // Min ms to wait before starting to type
  typing_start_delay_max?: number; // Max ms to wait before starting to type

  // Typing behavior - paste threshold (paste long text instead of typing)
  typing_paste_threshold?: number; // If text length > this, paste instead of type (0 = always type, -1 = always paste)

  // Typing behavior - enhanced variance (simulate human typing patterns)
  typing_variance_enabled?: boolean; // Enable digraph/shift/symbol variance

  // Mouse movement
  mouse_movement_style?: 'linear' | 'bezier' | 'natural';
  mouse_jitter_amount?: number; // Pixels of random movement

  // Click behavior
  click_delay_min?: number; // Min ms before clicking
  click_delay_max?: number; // Max ms before clicking

  // Scroll behavior
  scroll_style?: 'smooth' | 'stepped';
  scroll_speed_min?: number; // Min pixels per step
  scroll_speed_max?: number; // Max pixels per step

  // Random micro-pauses between actions
  micro_pause_enabled?: boolean;
  micro_pause_min_ms?: number;
  micro_pause_max_ms?: number;
  micro_pause_frequency?: number; // 0.0-1.0
}

/**
 * Ad blocking mode options.
 * - 'none': No ad blocking
 * - 'ads_only': Block advertisements only (EasyList + uBlock filters)
 * - 'ads_and_tracking': Block ads + tracking scripts (includes privacy lists)
 */
export type AdBlockingMode = 'none' | 'ads_only' | 'ads_and_tracking';

/**
 * Proxy settings for routing browser traffic through a proxy server.
 * Supports HTTP, HTTPS, and SOCKS5 proxies.
 */
export interface ProxySettings {
  enabled?: boolean; // Whether proxy is enabled
  server?: string; // Proxy URL (e.g., "http://proxy:8080" or "socks5://proxy:1080")
  bypass?: string; // Comma-separated domains to bypass proxy (e.g., "localhost,.internal.com")
  username?: string; // Proxy authentication username
  password?: string; // Proxy authentication password
}

/**
 * Anti-detection settings control bot detection bypass techniques.
 */
export interface AntiDetectionSettings {
  // Browser launch args
  disable_automation_controlled?: boolean; // --disable-blink-features=AutomationControlled
  disable_webrtc?: boolean; // Prevent IP leak via WebRTC

  // Navigator property patches
  patch_navigator_webdriver?: boolean; // Remove navigator.webdriver
  patch_navigator_plugins?: boolean; // Spoof plugins array
  patch_navigator_languages?: boolean; // Ensure consistent languages
  patch_webgl?: boolean; // Spoof WebGL renderer/vendor
  patch_canvas?: boolean; // Add noise to canvas fingerprint
  patch_audio_context?: boolean; // Add noise to AudioContext to prevent audio fingerprinting
  headless_detection_bypass?: boolean; // Bypass headless detection

  // Ad blocking
  ad_blocking_mode?: AdBlockingMode; // Block ads and/or tracking
}

/**
 * Quality presets for browser profiles.
 */
export type ProfilePreset = 'stealth' | 'balanced' | 'fast' | 'none';

/**
 * Complete browser profile configuration.
 */
export interface BrowserProfile {
  preset?: ProfilePreset;
  fingerprint?: FingerprintSettings;
  behavior?: BehaviorSettings;
  anti_detection?: AntiDetectionSettings;
  proxy?: ProxySettings;
}

/**
 * Resolved browser profile with all settings merged from preset and custom values.
 */
export interface ResolvedBrowserProfile {
  fingerprint: Required<FingerprintSettings>;
  behavior: Required<BehaviorSettings>;
  antiDetection: Required<AntiDetectionSettings>;
  proxy: Required<ProxySettings>;
}
