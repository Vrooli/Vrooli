import type {
  ProfilePreset,
  BehaviorSettings,
  AntiDetectionSettings,
  UserAgentPreset,
} from '@/domains/recording/types/types';

/** Preset configurations for behavior and anti-detection settings */
export const PRESET_CONFIGS: Record<
  ProfilePreset,
  { behavior: Partial<BehaviorSettings>; antiDetection: Partial<AntiDetectionSettings> }
> = {
  stealth: {
    behavior: {
      typing_delay_min: 50,
      typing_delay_max: 150,
      typing_start_delay_min: 100,
      typing_start_delay_max: 300,
      typing_paste_threshold: 200,
      typing_variance_enabled: true,
      mouse_movement_style: 'natural',
      mouse_jitter_amount: 2,
      click_delay_min: 100,
      click_delay_max: 300,
      scroll_style: 'stepped',
      micro_pause_enabled: true,
      micro_pause_min_ms: 500,
      micro_pause_max_ms: 2000,
      micro_pause_frequency: 0.15,
    },
    antiDetection: {
      disable_automation_controlled: true,
      patch_navigator_webdriver: true,
      patch_navigator_plugins: true,
      patch_navigator_languages: true,
      patch_webgl: true,
      patch_canvas: true,
      headless_detection_bypass: true,
      disable_webrtc: true,
      ad_blocking_mode: 'ads_and_tracking',
    },
  },
  balanced: {
    behavior: {
      typing_delay_min: 30,
      typing_delay_max: 80,
      typing_start_delay_min: 50,
      typing_start_delay_max: 150,
      typing_paste_threshold: 150,
      typing_variance_enabled: true,
      mouse_movement_style: 'bezier',
      mouse_jitter_amount: 1,
      click_delay_min: 50,
      click_delay_max: 150,
      scroll_style: 'smooth',
      micro_pause_enabled: true,
      micro_pause_min_ms: 200,
      micro_pause_max_ms: 800,
      micro_pause_frequency: 0.08,
    },
    antiDetection: {
      disable_automation_controlled: true,
      patch_navigator_webdriver: true,
      patch_navigator_plugins: true,
      patch_navigator_languages: false,
      patch_webgl: false,
      patch_canvas: false,
      headless_detection_bypass: true,
      disable_webrtc: false,
      ad_blocking_mode: 'ads_and_tracking',
    },
  },
  fast: {
    behavior: {
      typing_delay_min: 10,
      typing_delay_max: 30,
      typing_start_delay_min: 10,
      typing_start_delay_max: 30,
      typing_paste_threshold: 100,
      typing_variance_enabled: true,
      mouse_movement_style: 'linear',
      mouse_jitter_amount: 0,
      click_delay_min: 20,
      click_delay_max: 50,
      scroll_style: 'smooth',
      micro_pause_enabled: false,
      micro_pause_min_ms: 0,
      micro_pause_max_ms: 0,
      micro_pause_frequency: 0,
    },
    antiDetection: {
      disable_automation_controlled: true,
      patch_navigator_webdriver: true,
      patch_navigator_plugins: false,
      patch_navigator_languages: false,
      patch_webgl: false,
      patch_canvas: false,
      headless_detection_bypass: false,
      disable_webrtc: false,
      ad_blocking_mode: 'ads_and_tracking',
    },
  },
  none: {
    behavior: {
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
      micro_pause_enabled: false,
      micro_pause_min_ms: 0,
      micro_pause_max_ms: 0,
      micro_pause_frequency: 0,
    },
    antiDetection: {
      disable_automation_controlled: false,
      patch_navigator_webdriver: false,
      patch_navigator_plugins: false,
      patch_navigator_languages: false,
      patch_webgl: false,
      patch_canvas: false,
      headless_detection_bypass: false,
      disable_webrtc: false,
      ad_blocking_mode: 'none',
    },
  },
};

/** User agent preset labels for display */
export const USER_AGENT_LABELS: Record<UserAgentPreset, string> = {
  chrome_windows: 'Chrome on Windows',
  chrome_mac: 'Chrome on macOS',
  chrome_linux: 'Chrome on Linux',
  firefox_windows: 'Firefox on Windows',
  firefox_mac: 'Firefox on macOS',
  safari_mac: 'Safari on macOS',
  edge_windows: 'Edge on Windows',
  custom: 'Custom User Agent',
};

/** Preset metadata for UI display */
export const PRESET_METADATA: {
  id: ProfilePreset;
  name: string;
  description: string;
  recommended?: boolean;
}[] = [
  {
    id: 'stealth',
    name: 'Stealth',
    description: 'Maximum anti-detection with natural human-like behavior. Best for sites with strict bot detection.',
    recommended: true,
  },
  {
    id: 'balanced',
    name: 'Balanced',
    description: 'Good anti-detection with reasonable speed. Suitable for most websites.',
  },
  {
    id: 'fast',
    name: 'Fast',
    description: 'Minimal delays with basic anti-detection. Good for trusted internal tools.',
  },
  {
    id: 'none',
    name: 'None',
    description: 'No modifications. Raw automation speed for debugging or testing.',
  },
];
