/**
 * Constants used throughout the driver
 *
 * NOTE: Timeout values are now configurable via config.ts (execution group).
 * These constants serve as:
 * 1. Default values when config is not available (e.g., early initialization)
 * 2. Documentation of the default behavior
 * 3. Import targets for code that doesn't have config access
 *
 * For runtime configuration, prefer using config.execution.* values.
 * See config.ts for environment variable names and tuning guidance.
 */

// Execution timeouts (defaults - see config.execution for configurable versions)
export const DEFAULT_TIMEOUT_MS = 30000;
export const DEFAULT_NAVIGATION_TIMEOUT_MS = 45000;
export const DEFAULT_WAIT_TIMEOUT_MS = 30000;
export const DEFAULT_ASSERTION_TIMEOUT_MS = 15000;

// Telemetry limits (see config.telemetry for configurable versions)
export const MAX_CONSOLE_ENTRIES = 100;
export const MAX_NETWORK_EVENTS = 200;
export const MAX_SCREENSHOT_SIZE_BYTES = 512 * 1024;
export const MAX_DOM_SIZE_BYTES = 512 * 1024;

// Session limits
/**
 * Maximum executed instructions tracked per session for idempotency.
 * When exceeded, oldest entries are evicted (FIFO).
 * Trade-off: Higher = more replay protection, more memory per session.
 */
export const MAX_EXECUTED_INSTRUCTIONS_PER_SESSION = 1000;

// Recording injection retry configuration
/**
 * Maximum attempts to re-inject the recording script after navigation.
 * Scripts must be re-injected because pages lose injected JS on navigation.
 */
export const INJECTION_RETRY_MAX_ATTEMPTS = 3;

/**
 * Base delay (ms) for exponential backoff during injection retry.
 * Actual delays: 100ms, 200ms, 400ms (attempts 0, 1, 2).
 */
export const INJECTION_RETRY_BASE_DELAY_MS = 100;

// Gesture defaults
/**
 * Default number of animation steps for drag operations.
 * Higher = smoother animation but slower execution.
 */
export const DEFAULT_DRAG_ANIMATION_STEPS = 10;

export const VERSION = '2.0.0';

// =============================================================================
// HUMAN BEHAVIOR SIMULATION
// =============================================================================

/**
 * Minimum scroll distance (pixels) to consider element "close enough" to target.
 * Below this threshold, instant scroll is used instead of stepped scroll.
 * Trade-off: Lower = smoother micro-scrolls, higher = faster small movements.
 */
export const SCROLL_CLOSE_ENOUGH_THRESHOLD_PX = 10;

/**
 * Default minimum delay between scroll steps (ms).
 * Trade-off: Lower = faster scroll, higher = more natural appearance.
 */
export const SCROLL_STEP_MIN_DELAY_MS = 10;

/**
 * Default maximum delay between scroll steps (ms).
 * Actual delay is randomized between min and max for natural variation.
 */
export const SCROLL_STEP_MAX_DELAY_MS = 30;

/**
 * Base duration for smooth scroll animation (ms).
 * Final duration = base + (distance / SCROLL_DURATION_DISTANCE_FACTOR).
 */
export const SMOOTH_SCROLL_BASE_DURATION_MS = 300;

/**
 * Distance factor for smooth scroll duration calculation.
 * Duration increases by 1ms per this many pixels of distance.
 * Trade-off: Lower = faster long scrolls, higher = more time for content to load.
 */
export const SMOOTH_SCROLL_DISTANCE_FACTOR = 10;

/**
 * Maximum smooth scroll duration cap (ms).
 * Prevents extremely long scroll animations on very tall pages.
 */
export const SMOOTH_SCROLL_MAX_DURATION_MS = 1000;

/**
 * Default number of points in a natural mouse movement path.
 * Trade-off: Higher = smoother curves, more realistic; lower = faster.
 */
export const MOUSE_PATH_DEFAULT_STEPS = 15;

/**
 * Default duration for mouse movement along a path (ms).
 * Trade-off: Higher = slower, more natural; lower = faster execution.
 */
export const MOUSE_MOVEMENT_DEFAULT_DURATION_MS = 150;

// =============================================================================
// LOOP DETECTION
// =============================================================================

// Loop detection configuration
/**
 * Time window (ms) for detecting redirect loops.
 * Navigations within this window are counted together.
 */
export const LOOP_DETECTION_WINDOW_MS = 5000;

/**
 * Maximum navigations to same domain within window before triggering loop detection.
 * 4+ navigations to the same domain in 5 seconds = redirect loop.
 */
export const LOOP_DETECTION_MAX_NAVIGATIONS = 4;

/**
 * Number of recent navigations to track for loop detection.
 * Older entries are dropped to prevent unbounded memory growth.
 */
export const LOOP_DETECTION_HISTORY_SIZE = 10;

// =============================================================================
// RECORDING MODE
// =============================================================================

/**
 * Frame cache time-to-live (ms) for recording mode.
 * Cached frames are reused within this window to reduce redundant captures.
 * Trade-off: Lower = fresher frames, higher = less CPU usage.
 */
export const RECORDING_FRAME_CACHE_TTL_MS = 150;

/**
 * Timeout for page event handlers in recording lifecycle (ms).
 * Used for waitForFunction and similar page operations.
 */
export const RECORDING_PAGE_EVENT_TIMEOUT_MS = 5000;

/**
 * Timeout for callback HTTP requests in recording mode (ms).
 */
export const RECORDING_CALLBACK_TIMEOUT_MS = 5000;

// =============================================================================
// FRAME STREAMING
// =============================================================================

/**
 * ACK timeout for screencast frames (ms).
 * How long to wait for frame acknowledgment before considering it failed.
 * Trade-off: Lower = faster detection of dropped frames, higher = more tolerant of latency.
 */
export const SCREENCAST_ACK_TIMEOUT_MS = 1000;

/**
 * WebSocket reconnection delay (ms).
 * Delay between reconnection attempts after disconnect.
 */
export const FRAME_STREAMING_RECONNECT_DELAY_MS = 1000;

/**
 * FPS logging interval (frames).
 * Log FPS statistics every N frames for debugging.
 */
export const FRAME_STREAMING_FPS_LOG_INTERVAL = 30;

/**
 * Screenshot capture timeout (ms) for polling strategy.
 * Maximum time to wait for a single screenshot operation.
 */
export const FRAME_STREAMING_SCREENSHOT_TIMEOUT_MS = 200;

// =============================================================================
// SERVER LIFECYCLE
// =============================================================================

/**
 * Maximum drain time during graceful shutdown (ms).
 * After this time, remaining connections are forcibly closed.
 */
export const SERVER_DRAIN_TIMEOUT_MS = 30_000;

/**
 * Drain check interval during graceful shutdown (ms).
 */
export const SERVER_DRAIN_INTERVAL_MS = 100;
