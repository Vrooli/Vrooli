/**
 * CLI-specific constants for timeouts, dimensions, and other configuration values
 */

// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-07-14

import { 
    SECONDS_1_MS, 
    MINUTES_1_S, 
    DAYS_90_MS as _DAYS_90_MS,
    DAYS_30_MS as _DAYS_30_MS, 
} from "@vrooli/shared";

// Time constants (imported from shared)
export { 
    SECONDS_1_MS, 
    MINUTES_1_S, 
    DAYS_90_MS,
    DAYS_30_MS, 
} from "@vrooli/shared";

// Terminal and UI dimensions
export const TERMINAL_DIMENSIONS = {
    DEFAULT_WIDTH: 80,
    DEFAULT_HEIGHT: 24,
    FALLBACK_WIDTH: 80,
    MAX_MESSAGE_LENGTH: 2000,
} as const;

// Timeout values (in milliseconds)
export const TIMEOUTS = {
    CHAT_ENGINE_DEFAULT: 2000,
    CHAT_ENGINE_MINUTES_TO_MS: 60000,
    TOOL_APPROVAL_DEFAULT: 1000,
    TOOL_APPROVAL_SHORT: 50,
    TOOL_APPROVAL_MEDIUM: 500,
    CONFIG_SAVE_DEBOUNCE: 1000,
    MAX_WAIT_SECONDS: 300, // 5 minutes
    COMMAND_TIMEOUT_MS: 30000, // 30 seconds
} as const;

// Chat and routine configuration
export const CHAT_CONFIG = {
    DEFAULT_LIMIT: 10,
    PRIORITY_OFFSET: -10,
    PAGINATION_LIMIT: 10,
} as const;

// Display and formatting
export const DISPLAY_CONFIG = {
    ROUTINE_WIDTH_PADDING: 8,
    MAX_DISPLAY_WIDTH: 98,
    TABLE_COLUMN_WIDTHS: {
        SMALL: 8,
        MEDIUM: 20,
        LARGE: 30,
        EXTRA_LARGE: 20,
    },
} as const;

// Context management
export const CONTEXT_CONFIG = {
    MAX_CONTEXT_ENTRIES: 10,
    CONTEXT_SIZE_LIMIT: 1048576, // 1MB (1024 * 1024)
    CONTEXT_BUFFER_SIZE: 1024,
    MAX_RECENT_CONVERSATIONS: 20,
} as const;

// Export timing
export const EXPORT_CONFIG = {
    TIMESTAMP_MINUTES: 60,
} as const;

// Slash commands
export const SLASH_COMMANDS_CONFIG = {
    HELP_LIMIT: 10,
} as const;

// HTTP and API constants
export const HTTP_STATUS = {
    OK: 200,
    CREATED: 201,
    BAD_REQUEST: 400,
    UNAUTHORIZED: 401,
    FORBIDDEN: 403,
    NOT_FOUND: 404,
    INTERNAL_SERVER_ERROR: 500,
} as const;

// General limits and constraints
export const LIMITS = {
    DEFAULT_PAGE_SIZE: 20,
    MAX_CONTEXT_SIZE_BYTES: 10485760, // 10MB (10 * 1024 * 1024)
    MAX_CONTEXT_MESSAGES: 100,
    MAX_MESSAGE_DISPLAY: 10,
    MAX_RESULT_DISPLAY_LENGTH: 500,
} as const;

// UI constants
export const UI = {
    SEPARATOR_LENGTH: 50,
    POLLING_INTERVAL_MS: 50,
    CONTENT_WIDTH: 98,
    DESCRIPTION_WIDTH: 30,
    TERMINAL_WIDTH: 80, // Default terminal width
    PADDING: {
        TOP: 8,
        RIGHT: 20,
        BOTTOM: 30,
        LEFT: 20,
    },
    WIDTH_ADJUSTMENT: 8,
} as const;

// Tool approval constants
export const TOOL_APPROVAL = {
    POLLING_INTERVAL_MS: 50,
    SEPARATOR_LENGTH: 50,
} as const;

// CLI-specific constants for linting fixes
export const CLI_LIMITS = {
    DEFAULT_HISTORY_LIMIT: 10,
    DEFAULT_SUGGESTIONS_LIMIT: 5,
    STATS_TOP_COMMANDS_LIMIT: 10,
    EXPORT_PREVIEW_LENGTH: 1000,
    CACHE_MAX_SIZE: 1000,
    SUGGESTION_MULTIPLIER: 2,
    DISPLAY_RECENT_DAYS: 7,
    INBOX_MULTIPLIER: 8,
    ID_DISPLAY_LENGTH: 8,
} as const;

const CACHE_DEFAULT_TTL_MULTIPLIER = 5;

export const CACHE_CONSTANTS = {
    DEFAULT_TTL_SECONDS: CACHE_DEFAULT_TTL_MULTIPLIER * MINUTES_1_S, // 5 minutes
    FLUSH_INTERVAL_MS: MINUTES_1_S * SECONDS_1_MS, // 1 minute
    MAX_SIZE: 1000,
} as const;

export const HISTORY_CONSTANTS = {
    DEFAULT_CLEANUP_DAYS: 90,
    DEFAULT_KEEP_COUNT: 1000,
    RECENT_ACTIVITY_DAYS: 30,
    RECENT_ACTIVITY_LIMIT: 30,
    PAGINATION_DEFAULT_SIZE: 10,
} as const;

export const EXIT_CODES = {
    SUCCESS: 0,
    ERROR: 1,
    SIGINT: 130,
    SIGTERM: 143,
} as const;

// WebSocket constants
export const WEBSOCKET_CONFIG = {
    MAX_RECONNECT_ATTEMPTS: 5,
    BASE_RECONNECT_DELAY_MS: 1000,
    MAX_RECONNECT_DELAY_MS: 16000,
    CONNECTION_TIMEOUT_MS: 10000,
} as const;

// Authentication constants
export const AUTH_CONFIG = {
    MIN_PASSWORD_LENGTH: 8,
} as const;
