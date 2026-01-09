/**
 * Browser Profile Configuration
 *
 * CONTROL SURFACE: Default values for anti-detection and fingerprint spoofing.
 *
 * These defaults are designed to make automated browsers appear as
 * legitimate user browsers, reducing detection by anti-bot systems.
 *
 * ## Configuration Philosophy:
 * - Defaults should represent a "typical" real user's browser
 * - Values should be internally consistent (e.g., viewport matches screen)
 * - Trade-offs are between detection avoidance and accuracy
 */

// =============================================================================
// HARDWARE FINGERPRINT DEFAULTS
// =============================================================================

/**
 * Default hardware specifications for fingerprint spoofing.
 * These should match common consumer hardware to avoid outlier detection.
 */
export const HARDWARE_DEFAULTS = {
  /**
   * CPU core count.
   * Most consumer devices have 4-8 cores. 4 is a conservative default.
   */
  hardwareConcurrency: 4,

  /**
   * Device memory in GB.
   * 8GB is common for mid-range laptops/desktops.
   */
  deviceMemory: 8,

  /**
   * Default viewport/screen dimensions (desktop).
   * 1920x1080 is the most common resolution globally.
   */
  viewportWidth: 1920,
  viewportHeight: 1080,

  /**
   * Taskbar/chrome height reduction for availHeight.
   * 40px is typical for Windows taskbar.
   */
  taskbarHeight: 40,

  /**
   * Color/pixel depth (bits).
   * 24-bit color is standard for modern displays.
   */
  colorDepth: 24,
  pixelDepth: 24,
} as const;

// =============================================================================
// WINDOW DIMENSIONS FOR HEADLESS DETECTION BYPASS
// =============================================================================

/**
 * Window chrome dimensions to simulate non-headless browser.
 * Real browsers have window decorations that increase outer dimensions.
 */
export const WINDOW_CHROME = {
  /**
   * Width difference between inner and outer window (scrollbar + frame).
   * 16px is typical for a vertical scrollbar.
   */
  widthDelta: 16,

  /**
   * Height difference between inner and outer window (title bar + toolbar).
   * 88px accounts for title bar (~30px) and browser toolbar (~58px).
   */
  heightDelta: 88,
} as const;

// =============================================================================
// NETWORK CONNECTION SPOOFING
// =============================================================================

/**
 * Network connection properties for Connection API spoofing.
 * These represent typical residential WiFi connections.
 */
export const CONNECTION_DEFAULTS = {
  /**
   * Effective connection type.
   * '4g' indicates a fast connection (>10Mbps effective).
   */
  effectiveType: '4g',

  /**
   * Connection type.
   * 'wifi' is most common for desktop browsers.
   */
  type: 'wifi',

  /**
   * Downlink bandwidth in Mbps.
   * 10 Mbps is conservative for residential broadband.
   */
  downlink: 10,

  /**
   * Round-trip time in ms.
   * 50ms is typical for residential connections.
   */
  rtt: 50,

  /**
   * Data saver mode.
   * false is the default for most users.
   */
  saveData: false,
} as const;

// =============================================================================
// COMMON FONTS LIST
// =============================================================================

/**
 * List of fonts commonly available on most systems.
 * Used to mask the actual font list which can be used for fingerprinting.
 */
export const COMMON_FONTS = new Set([
  'Arial',
  'Arial Black',
  'Comic Sans MS',
  'Courier New',
  'Georgia',
  'Helvetica',
  'Impact',
  'Lucida Console',
  'Lucida Sans Unicode',
  'Palatino Linotype',
  'Tahoma',
  'Times New Roman',
  'Trebuchet MS',
  'Verdana',
]);

// =============================================================================
// WEBGL VENDOR/RENDERER SPOOFING
// =============================================================================

/**
 * WebGL renderer info to spoof.
 * Intel integrated graphics is common and less suspicious than discrete GPUs.
 */
export const WEBGL_DEFAULTS = {
  /**
   * GPU vendor string.
   */
  vendor: 'Intel Inc.',

  /**
   * GPU renderer string.
   * Intel Iris is common in modern laptops.
   */
  renderer: 'Intel Iris OpenGL Engine',
} as const;

// =============================================================================
// CANVAS FINGERPRINT NOISE
// =============================================================================

/**
 * Canvas fingerprint noise settings.
 */
export const CANVAS_NOISE = {
  /**
   * Number of pixels to modify per canvas read operation.
   * Trade-off: Higher = more noise/harder to fingerprint, but may affect visuals.
   */
  pixelsToModify: 10,
} as const;

// =============================================================================
// AUDIO CONTEXT FINGERPRINT NOISE
// =============================================================================

/**
 * Audio context fingerprint noise settings.
 */
export const AUDIO_NOISE = {
  /**
   * Probability of modifying each audio sample (0-1).
   * Trade-off: Higher = more noise, but may affect audio quality.
   */
  sampleModifyProbability: 0.05,

  /**
   * Maximum noise amplitude for float frequency data.
   */
  floatNoiseAmplitude: 0.1,

  /**
   * Maximum noise amplitude for audio buffer data.
   * Very small to avoid audible artifacts.
   */
  bufferNoiseAmplitude: 0.0001,

  /**
   * Probability of modifying each sample in audio buffers (0-1).
   * Very low to avoid audible changes.
   */
  bufferModifyProbability: 0.001,
} as const;

// =============================================================================
// DEFAULT PLUGINS AND MIME TYPES
// =============================================================================

/**
 * Default browser plugins to report.
 * These are standard Chrome plugins present on all installations.
 */
export const DEFAULT_PLUGINS = [
  { name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
  { name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
  { name: 'Native Client', filename: 'internal-nacl-plugin', description: '' },
] as const;

/**
 * Default MIME types to report.
 */
export const DEFAULT_MIME_TYPES = [
  { type: 'application/pdf', description: 'Portable Document Format', suffixes: 'pdf' },
  { type: 'text/pdf', description: 'Portable Document Format', suffixes: 'pdf' },
] as const;

// =============================================================================
// BATTERY API DEFAULTS
// =============================================================================

/**
 * Default battery status to report.
 * Plugged in and fully charged is the most common state for desktop use.
 */
export const BATTERY_DEFAULTS = {
  charging: true,
  chargingTime: 0,
  dischargingTime: Infinity,
  level: 1.0,
} as const;
