/**
 * Recording Domain Constants
 *
 * Single source of truth for default values used across the recording system.
 * This prevents scattered magic numbers and ensures consistency.
 *
 * IMPORTANT: When changing these values, consider the impact on:
 * - Stream quality/bandwidth tradeoffs
 * - User experience (preview smoothness)
 * - Server/client resource usage
 */

// =============================================================================
// Frame Streaming Defaults
// =============================================================================

/**
 * Default target FPS for frame streaming.
 *
 * Note: For CDP screencast (Chromium), Chrome's compositor controls actual
 * frame delivery rate. This value serves as a "hint" and is primarily used
 * by the polling fallback strategy.
 *
 * Trade-off:
 * - Higher FPS = smoother preview, more bandwidth, higher CPU
 * - Lower FPS = choppier preview, less bandwidth, lower CPU
 *
 * 30 FPS provides smooth preview without excessive bandwidth.
 */
export const DEFAULT_STREAM_FPS = 30;

/**
 * Default JPEG quality for frame streaming (1-100).
 *
 * Trade-off:
 * - Higher quality = sharper image, larger file size, more bandwidth
 * - Lower quality = more artifacts, smaller file size, less bandwidth
 *
 * 55 provides good balance for preview purposes.
 */
export const DEFAULT_STREAM_QUALITY = 55;

/**
 * Default scale mode for screenshots.
 *
 * 'css' = 1x scale (standard resolution)
 * 'device' = devicePixelRatio scale (HiDPI/Retina)
 *
 * 'css' is default for bandwidth efficiency.
 */
export const DEFAULT_STREAM_SCALE: 'css' | 'device' = 'css';

// =============================================================================
// FPS Bounds
// =============================================================================

/**
 * Minimum allowed FPS for stream settings.
 */
export const MIN_STREAM_FPS = 1;

/**
 * Maximum allowed FPS for stream settings.
 */
export const MAX_STREAM_FPS = 60;

// =============================================================================
// Quality Bounds
// =============================================================================

/**
 * Minimum allowed quality for stream settings.
 */
export const MIN_STREAM_QUALITY = 1;

/**
 * Maximum allowed quality for stream settings.
 */
export const MAX_STREAM_QUALITY = 100;

// =============================================================================
// Stream Settings Presets
// =============================================================================

/**
 * Stream settings preset definitions.
 * Used by StreamSettings component for quick configuration.
 */
export const STREAM_PRESETS = {
  fast: {
    quality: 35,
    fps: DEFAULT_STREAM_FPS,
    scale: 'css' as const,
    label: 'Fast',
    description: 'Lower quality, reduced bandwidth',
  },
  balanced: {
    quality: DEFAULT_STREAM_QUALITY,
    fps: DEFAULT_STREAM_FPS,
    scale: 'css' as const,
    label: 'Balanced',
    description: 'Good balance of quality and performance',
  },
  sharp: {
    quality: 75,
    fps: DEFAULT_STREAM_FPS,
    scale: 'css' as const,
    label: 'Sharp',
    description: 'Higher quality, more bandwidth',
  },
  hidpi: {
    quality: DEFAULT_STREAM_QUALITY,
    fps: DEFAULT_STREAM_FPS,
    scale: 'device' as const,
    label: 'HiDPI',
    description: 'Device pixel ratio for Retina displays',
  },
} as const;

/**
 * Default preset name.
 */
export const DEFAULT_STREAM_PRESET = 'balanced';
