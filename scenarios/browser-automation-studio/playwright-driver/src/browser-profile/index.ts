/**
 * Browser Profile Module
 *
 * Anti-detection and human-like behavior configuration for browser automation.
 */

// Re-export types
export type {
  BrowserProfile,
  FingerprintSettings,
  BehaviorSettings,
  AntiDetectionSettings,
  AdBlockingMode,
  ProxySettings,
  ProfilePreset,
  UserAgentPreset,
  ResolvedBrowserProfile,
} from '../types/browser-profile';

// Export presets
export {
  PRESETS,
  PRESET_STEALTH,
  PRESET_BALANCED,
  PRESET_FAST,
  PRESET_NONE,
  DEFAULT_FINGERPRINT,
  DEFAULT_BEHAVIOR,
  DEFAULT_ANTI_DETECTION,
  DEFAULT_PROXY,
  getPreset,
  mergeWithPreset,
  USER_AGENTS,
  resolveUserAgent,
} from './presets';

// Export anti-detection utilities
export {
  buildAntiDetectionArgs,
  generateAntiDetectionScript,
  applyAntiDetection,
  getEnabledPatches,
} from './anti-detection';

// Export patch registry for advanced use (testing, debugging, selective application)
export {
  PATCH_REGISTRY,
  composePatches,
  type PatchRegistryEntry,
  type PatchGenerator,
} from './patches';

// Export human-like behavior utilities
export {
  HumanBehavior,
  sleep,
  typeWithDelay,
  moveMouseAlongPath,
  type Point,
} from './human-behavior';

// Export ad blocking utilities
export { applyAdBlocking, isAdBlockingEnabled } from './ad-blocker';

// Export Client Hints utilities
export {
  generateClientHints,
  mergeClientHintsWithHeaders,
  parseUserAgent,
  type ClientHintsHeaders,
  type ParsedUserAgent,
} from './client-hints';

// Export context integration (behavior settings key)
export { BEHAVIOR_SETTINGS_KEY } from './context-integration';

// Export configuration defaults (anti-detection tunable levers)
export {
  HARDWARE_DEFAULTS,
  WINDOW_CHROME,
  CONNECTION_DEFAULTS,
  COMMON_FONTS,
  WEBGL_DEFAULTS,
  CANVAS_NOISE,
  AUDIO_NOISE,
  DEFAULT_PLUGINS,
  DEFAULT_MIME_TYPES,
  BATTERY_DEFAULTS,
} from './config';
