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
} from './anti-detection';

// Export human-like behavior utilities
export {
  HumanBehavior,
  sleep,
  typeWithDelay,
  moveMouseAlongPath,
  type Point,
} from './human-behavior';
