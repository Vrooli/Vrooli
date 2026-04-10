/**
 * Tunable UI constants for Brand Manager.
 *
 * These values shape polling, retry, and display behavior. They are
 * intentionally co-located so operators can tune the UI without hunting
 * through component files.
 *
 * DOC: docs/reference/configuration.md#ui-constants
 */

// --- Health Polling ---

/** How many times to retry a failed health check before showing "offline". */
export const HEALTH_CHECK_RETRY = 2;

/** Interval (ms) between automatic health-check refetches. */
export const HEALTH_CHECK_INTERVAL_MS = 30_000;

