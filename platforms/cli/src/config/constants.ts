/**
 * Centralized configuration constants for the CLI
 * These values can be overridden via environment variables
 */

export const CLI_CONFIG = {
    /** API request timeout in milliseconds */
    API_TIMEOUT: parseInt(process.env.CLI_API_TIMEOUT || "30000", 10),
    
    /** Default page size for list operations */
    DEFAULT_PAGE_SIZE: parseInt(process.env.CLI_PAGE_SIZE || "20", 10),
    
    /** Maximum page size allowed */
    MAX_PAGE_SIZE: 100,
    
    /** WebSocket reconnection settings */
    WS_RECONNECT: {
        MAX_ATTEMPTS: parseInt(process.env.CLI_WS_MAX_RETRIES || "3", 10),
        BASE_DELAY: parseInt(process.env.CLI_WS_RETRY_DELAY || "1000", 10),
        MAX_DELAY: parseInt(process.env.CLI_WS_MAX_DELAY || "10000", 10),
    },
    
    /** Command history settings */
    HISTORY: {
        MAX_ENTRIES: parseInt(process.env.CLI_HISTORY_MAX || "10000", 10),
        DEFAULT_LIMIT: 20,
    },
    
    /** File import/export settings */
    FILE_IO: {
        MAX_BATCH_SIZE: parseInt(process.env.CLI_MAX_BATCH || "50", 10),
        IMPORT_CHUNK_SIZE: 10,
    },
    
    /** Display settings */
    DISPLAY: {
        MAX_TABLE_WIDTH: 120,
        TRUNCATE_LENGTH: 50,
    },
    
    /** Default URLs for different environments */
    DEFAULT_URLS: {
        local: "http://localhost:5329",
        development: "https://dev.vrooli.com",
        staging: "https://staging.vrooli.com",
        production: "https://api.vrooli.com",
    },
    
    /** Test environment settings */
    TEST: {
        DEFAULT_PORT: 5329,
        DEFAULT_URL: "http://localhost:5329",
    },
} as const;

/** File paths */
export const PATHS = {
    CONFIG_DIR: process.env.VROOLI_CONFIG_DIR || `${process.env.HOME}/.vrooli`,
    CONFIG_FILE: "config.json",
    HISTORY_DB: "history.db",
    HISTORY_JSON: "history.json",
    COMPLETIONS_CACHE: "completions.cache",
} as const;

/** Command names for consistency */
export const COMMANDS = {
    AUTH: "auth",
    ROUTINE: "routine",
    CHAT: "chat",
    AGENT: "agent",
    TEAM: "team",
    HISTORY: "history",
    PROFILE: "profile",
    COMPLETION: "completion",
} as const;

/** Exit codes */
export const EXIT_CODES = {
    SUCCESS: 0,
    GENERAL_ERROR: 1,
    AUTHENTICATION_ERROR: 2,
    NETWORK_ERROR: 3,
    VALIDATION_ERROR: 4,
    NOT_FOUND: 5,
} as const;
