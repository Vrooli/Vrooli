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
 * Priority levels for event processing
 */
export const PRIORITY_LEVELS = {
    LOW: "low" as const,
    MEDIUM: "medium" as const,
    HIGH: "high" as const,
    CRITICAL: "critical" as const,
} as const;

/**
 * Bot priority calculation constants
 */
export const PRIORITY_WEIGHTS = {
    CONFIG_PRIORITY: 1000,
    PATTERN_SPECIFICITY: 100,
    ROLE_WEIGHT: 10,
    EXPERTISE_MATCH: 5,
} as const;

export const ROLE_WEIGHTS = {
    ARBITRATOR: 10,
    LEADER: 7,
    COORDINATOR: 5,
    SPECIALIST: 4,
    MEMBER: 2,
    DEFAULT: 2,
} as const;

export const PATTERN_SCORING = {
    EXACT_MATCH_SCORE: 10,
    SEGMENT_SCORE: 2,
    WILDCARD_PENALTY: 5,
    EXPERTISE_MATCH_SCORE: 5,
} as const;

/**
 * Event Bus Monitor constants
 */
export const MONITOR_SCORING = {
    DEDUCTION_HIGH_LATENCY: 20,
    DEDUCTION_HIGH_ERRORS: 30,
    DEDUCTION_LOW_THROUGHPUT: 15,
    DEDUCTION_SLOW_PATTERNS: 10,
    THRESHOLD_HEALTHY: 80,
    THRESHOLD_DEGRADED: 50,
} as const;

export const MONITOR_THRESHOLDS = {
    HIGH_FREQUENCY: 1000,
    LOW_PROCESSING_TIME: 10,
    LOW_CACHE_HIT_RATE: 0.5,
    MIN_PATTERN_MATCHES_FOR_OPTIMIZATION: 100,
    HIGH_EVENT_COUNT: 10000,
    SLOW_PATTERN_MS: 10,
    COMPLEX_PATTERN_WILDCARD_COUNT: 3,
    TOP_EVENT_TYPES_COUNT: 10,
    SLOWEST_PATTERNS_COUNT: 5,
} as const;

export const MONITOR_PERFORMANCE_LIMITS = {
    MAX_LATENCY_MS: 1000,
    MAX_QUEUE_SIZE: 10000,
    MIN_THROUGHPUT: 10, // events/second
    MAX_ERROR_RATE: 0.05, // 5%
    MAX_MEMORY_MB: 500,
} as const;
