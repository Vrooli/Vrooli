/**
 * Constants for the ReplayPlayer component
 *
 * Contains all configuration constants and option arrays:
 * - Duration limits
 * - Cursor scale limits
 * - Default profiles
 * - Dropdown options
 * - Fallback values
 */

import type {
  CursorSpeedProfile,
  CursorPathStyle,
  Dimensions,
  SpeedProfileOption,
  CursorPathStyleOption,
} from './types';

// =============================================================================
// Duration Constants
// =============================================================================

export const DEFAULT_DURATION = 1600;
export const MIN_DURATION = 800;
export const MAX_DURATION = 6000;

// =============================================================================
// Cursor Scale Constants
// =============================================================================

export const MIN_CURSOR_SCALE = 0.6;
export const MAX_CURSOR_SCALE = 1.8;

// =============================================================================
// Default Profiles
// =============================================================================

export const DEFAULT_SPEED_PROFILE: CursorSpeedProfile = 'linear';
export const DEFAULT_PATH_STYLE: CursorPathStyle = 'linear';

// =============================================================================
// Dropdown Options
// =============================================================================

export const SPEED_PROFILE_OPTIONS: SpeedProfileOption[] = [
  { id: 'linear', label: 'Linear', description: 'Consistent motion between frames' },
  { id: 'easeIn', label: 'Ease in', description: 'Begin slowly, accelerate toward the target' },
  { id: 'easeOut', label: 'Ease out', description: 'Move quickly at first, settle into the target' },
  { id: 'easeInOut', label: 'Ease in/out', description: 'Smooth acceleration and deceleration' },
  { id: 'instant', label: 'Instant', description: 'Jump directly at the end of the step' },
];

export const CURSOR_PATH_STYLE_OPTIONS: CursorPathStyleOption[] = [
  { id: 'linear', label: 'Linear', description: 'Direct line between previous and current positions' },
  { id: 'parabolicUp', label: 'Parabolic (up)', description: 'Arcs upward before easing into the target' },
  { id: 'parabolicDown', label: 'Parabolic (down)', description: 'Arcs downward before easing into the target' },
  { id: 'cubic', label: 'Cubic', description: 'Smooth S-curve with a gentle overshoot' },
  { id: 'pseudorandom', label: 'Pseudorandom', description: 'Organic path generated from deterministic random waypoints' },
];

// =============================================================================
// Fallback Values
// =============================================================================

export const FALLBACK_DIMENSIONS: Dimensions = { width: 1920, height: 1080 };
export const FALLBACK_CLICK_COLOR: [string, string, string] = ['59', '130', '246'];

// =============================================================================
// SVG Paths
// =============================================================================

export const REPLAY_ARROW_CURSOR_PATH = 'M6 3L6 22L10.4 18.1L13.1 26.4L15.9 25.2L13.1 17.5L22 17.5L6 3Z';
