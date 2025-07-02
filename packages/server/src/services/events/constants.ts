/**
 * Event System Constants
 * 
 * Centralized constants for the unified event system to avoid magic numbers
 * and strings throughout the codebase.
 */

/**
 * Event bus configuration constants
 */
export const EVENT_BUS_CONSTANTS = {
    /** Maximum number of events to keep in history */
    MAX_EVENT_HISTORY: 1000,
    /** Maximum number of listeners for a single event pattern */
    MAX_EVENT_LISTENERS: 1000,
    /** Maximum concurrent handlers per event to prevent backpressure */
    MAX_CONCURRENT_HANDLERS_PER_EVENT: 100,
    /** Length of pattern suffix for wildcard matching */
    PATTERN_SUFFIX_LENGTH: 2,
    /** Base delay for retry attempts in milliseconds */
    RETRY_BASE_DELAY_MS: 100,
    /** Exponential backoff factor for retries */
    RETRY_EXPONENTIAL_FACTOR: 2,
    /** Default timeout for barrier sync in milliseconds */
    DEFAULT_BARRIER_TIMEOUT_MS: 30000,
} as const;



/**
 * Delivery guarantee levels
 */
export const DELIVERY_GUARANTEES = {
    FIRE_AND_FORGET: "fire-and-forget" as const,
    RELIABLE: "reliable" as const,
    BARRIER_SYNC: "barrier-sync" as const,
} as const;

/**
 * Priority levels for event processing
 */
export const PRIORITY_LEVELS = {
    LOW: "low" as const,
    MEDIUM: "medium" as const,
    HIGH: "high" as const,
    CRITICAL: "critical" as const,
} as const;
