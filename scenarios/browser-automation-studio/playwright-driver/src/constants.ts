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

export const VERSION = '2.0.0';
