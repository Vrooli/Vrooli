/**
 * Browser Profile Presets
 *
 * Predefined configurations for different automation use cases.
 */

import type {
  BrowserProfile,
  ProfilePreset,
  FingerprintSettings,
  BehaviorSettings,
  AntiDetectionSettings,
  AdBlockingMode,
} from '../types/browser-profile';

/**
 * Default fingerprint settings used when not specified.
 */
export const DEFAULT_FINGERPRINT: Required<FingerprintSettings> = {
  viewport_width: 1280,
  viewport_height: 720,
  device_scale_factor: 2,
  hardware_concurrency: 4,
  device_memory: 8,
  user_agent: '',
  user_agent_preset: 'chrome-win',
  locale: 'en-US',
  timezone_id: 'America/New_York',
  geolocation_enabled: false,
  latitude: 0,
  longitude: 0,
  accuracy: 100,
  color_scheme: 'light',
};

/**
 * Default behavior settings (no delays - instant execution).
 */
export const DEFAULT_BEHAVIOR: Required<BehaviorSettings> = {
  typing_delay_min: 0,
  typing_delay_max: 0,
  typing_start_delay_min: 0,
  typing_start_delay_max: 0,
  typing_paste_threshold: 0,
  typing_variance_enabled: false,
  mouse_movement_style: 'linear',
  mouse_jitter_amount: 0,
  click_delay_min: 0,
  click_delay_max: 0,
  scroll_style: 'smooth',
  scroll_speed_min: 100,
  scroll_speed_max: 300,
  micro_pause_enabled: false,
  micro_pause_min_ms: 0,
  micro_pause_max_ms: 0,
  micro_pause_frequency: 0,
};

/**
 * Default anti-detection settings (ad blocking enabled by default).
 */
export const DEFAULT_ANTI_DETECTION: Required<AntiDetectionSettings> = {
  disable_automation_controlled: false,
  disable_webrtc: false,
  patch_navigator_webdriver: false,
  patch_navigator_plugins: false,
  patch_navigator_languages: false,
  patch_webgl: false,
  patch_canvas: false,
  headless_detection_bypass: false,
  ad_blocking_mode: 'ads_and_tracking',
};

/**
 * Stealth preset - Maximum anti-detection with realistic human behavior.
 * Use for sites with aggressive bot detection.
 */
export const PRESET_STEALTH: BrowserProfile = {
  preset: 'stealth',
  fingerprint: {
    device_scale_factor: 1, // Match common displays
    hardware_concurrency: 4,
    device_memory: 8,
  },
  behavior: {
    typing_delay_min: 50,
    typing_delay_max: 150,
    typing_start_delay_min: 100,
    typing_start_delay_max: 300,
    typing_paste_threshold: 200, // Paste text longer than 200 chars
    typing_variance_enabled: true,
    mouse_movement_style: 'natural',
    mouse_jitter_amount: 2,
    click_delay_min: 100,
    click_delay_max: 300,
    scroll_style: 'smooth',
    micro_pause_enabled: true,
    micro_pause_min_ms: 200,
    micro_pause_max_ms: 800,
    micro_pause_frequency: 0.15,
  },
  anti_detection: {
    disable_automation_controlled: true,
    disable_webrtc: true,
    patch_navigator_webdriver: true,
    patch_navigator_plugins: true,
    patch_navigator_languages: true,
    patch_webgl: true,
    patch_canvas: true,
    headless_detection_bypass: true,
  },
};

/**
 * Balanced preset - Good protection with reasonable speed.
 * Recommended for most use cases.
 */
export const PRESET_BALANCED: BrowserProfile = {
  preset: 'balanced',
  behavior: {
    typing_delay_min: 30,
    typing_delay_max: 80,
    typing_start_delay_min: 50,
    typing_start_delay_max: 150,
    typing_paste_threshold: 150, // Paste text longer than 150 chars
    typing_variance_enabled: true,
    mouse_movement_style: 'bezier',
    click_delay_min: 50,
    click_delay_max: 150,
    micro_pause_enabled: true,
    micro_pause_min_ms: 100,
    micro_pause_max_ms: 400,
    micro_pause_frequency: 0.08,
  },
  anti_detection: {
    disable_automation_controlled: true,
    patch_navigator_webdriver: true,
  },
};

/**
 * Fast preset - Minimal delays with basic fingerprint masking.
 * Use for trusted sites or internal tools.
 */
export const PRESET_FAST: BrowserProfile = {
  preset: 'fast',
  behavior: {
    typing_delay_min: 10,
    typing_delay_max: 30,
    typing_start_delay_min: 10,
    typing_start_delay_max: 30,
    typing_paste_threshold: 100, // Paste text longer than 100 chars
    typing_variance_enabled: true,
    mouse_movement_style: 'linear',
    click_delay_min: 20,
    click_delay_max: 50,
    micro_pause_enabled: false,
  },
  anti_detection: {
    disable_automation_controlled: true,
    patch_navigator_webdriver: true,
  },
};

/**
 * None preset - Raw automation with no modifications.
 * Use for internal tools only.
 */
export const PRESET_NONE: BrowserProfile = {
  preset: 'none',
  anti_detection: {
    ad_blocking_mode: 'none',
  },
};

/**
 * Map of preset names to configurations.
 */
export const PRESETS: Record<ProfilePreset, BrowserProfile> = {
  stealth: PRESET_STEALTH,
  balanced: PRESET_BALANCED,
  fast: PRESET_FAST,
  none: PRESET_NONE,
};

/**
 * Get the preset configuration for a given preset name.
 */
export function getPreset(preset: ProfilePreset): BrowserProfile {
  return PRESETS[preset] ?? PRESET_NONE;
}

/**
 * Merge a browser profile with preset defaults.
 * Custom values override preset values.
 */
export function mergeWithPreset(profile?: BrowserProfile): {
  fingerprint: Required<FingerprintSettings>;
  behavior: Required<BehaviorSettings>;
  antiDetection: Required<AntiDetectionSettings>;
} {
  // Start with defaults
  const fingerprint = { ...DEFAULT_FINGERPRINT };
  const behavior = { ...DEFAULT_BEHAVIOR };
  const antiDetection = { ...DEFAULT_ANTI_DETECTION };

  if (!profile) {
    return { fingerprint, behavior, antiDetection };
  }

  // Apply preset if specified
  const preset = profile.preset ? getPreset(profile.preset) : null;
  if (preset) {
    if (preset.fingerprint) {
      Object.assign(fingerprint, preset.fingerprint);
    }
    if (preset.behavior) {
      Object.assign(behavior, preset.behavior);
    }
    if (preset.anti_detection) {
      Object.assign(antiDetection, preset.anti_detection);
    }
  }

  // Apply custom overrides
  if (profile.fingerprint) {
    Object.assign(fingerprint, profile.fingerprint);
  }
  if (profile.behavior) {
    Object.assign(behavior, profile.behavior);
  }
  if (profile.anti_detection) {
    Object.assign(antiDetection, profile.anti_detection);
  }

  return { fingerprint, behavior, antiDetection };
}

/**
 * User agent strings for common browser/OS combinations.
 */
export const USER_AGENTS: Record<string, string> = {
  'chrome-win':
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  'chrome-mac':
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  'chrome-linux':
    'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  'firefox-win':
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0',
  'firefox-mac':
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0',
  'firefox-linux': 'Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0',
  'safari-mac':
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15',
  'edge-win':
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0',
};

/**
 * Get the user agent string for a preset or custom value.
 */
export function resolveUserAgent(fingerprint: FingerprintSettings): string {
  // Use custom user agent if provided
  if (fingerprint.user_agent) {
    return fingerprint.user_agent;
  }

  // Use preset user agent if specified
  if (fingerprint.user_agent_preset && USER_AGENTS[fingerprint.user_agent_preset]) {
    return USER_AGENTS[fingerprint.user_agent_preset];
  }

  // Default to Chrome on Windows
  return USER_AGENTS['chrome-win'];
}
