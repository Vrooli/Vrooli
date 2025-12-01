/**
 * Constants and Configuration for Test Genie
 * Centralized constants for use across the application
 */

// API Configuration
export const API_BASE_URL = '/api/v1';

// Responsive Breakpoints
export const MOBILE_BREAKPOINT = 768; // pixels

// Storage Keys
export const STORAGE_KEYS = {
    DEFAULT_SETTINGS: 'testGenie.defaultSettings',
    SELECTED_MODEL: 'testGenie.selectedModel',
};

// Default Model Configuration
export const DEFAULT_MODEL = 'openrouter/x-ai/grok-code-fast-1';

// Vault Phase Definitions
export const DEFAULT_VAULT_PHASES = ['setup', 'develop', 'test'];

export const VAULT_PHASE_DEFINITIONS = {
    setup: {
        label: 'Setup',
        description: 'Provision dependencies, seed data, confirm configuration drift.',
        defaultTimeout: 600
    },
    develop: {
        label: 'Develop',
        description: 'Run developer-focused checks: lint, unit, integration, feature toggles.',
        defaultTimeout: 900
    },
    test: {
        label: 'Test',
        description: 'Execute regression and smoke suites that guard critical flows.',
        defaultTimeout: 1200
    },
    deploy: {
        label: 'Deploy',
        description: 'Validate rollout scripts, migrations, and packaging steps.',
        defaultTimeout: 900
    },
    monitor: {
        label: 'Monitor',
        description: 'Observe health metrics, load, and canary checks post-deploy.',
        defaultTimeout: 900
    }
};

// Test Generation Phases
export const DEFAULT_GENERATION_PHASES = [
    'dependencies',
    'structure',
    'unit',
    'integration',
    'business',
    'performance'
];

// Status Descriptors
export const STATUS_DESCRIPTORS = {
    // Execution statuses
    pending: { key: 'pending', label: 'Pending', icon: 'clock' },
    queued: { key: 'queued', label: 'Queued', icon: 'hourglass' },
    running: { key: 'running', label: 'Running', icon: 'zap' },
    passed: { key: 'passed', label: 'Passed', icon: 'check-circle' },
    failed: { key: 'failed', label: 'Failed', icon: 'x-circle' },
    error: { key: 'error', label: 'Error', icon: 'alert-triangle' },
    completed: { key: 'completed', label: 'Completed', icon: 'check' },
    skipped: { key: 'skipped', label: 'Skipped', icon: 'skip-forward' },
    cancelled: { key: 'cancelled', label: 'Cancelled', icon: 'x' },
    timeout: { key: 'timeout', label: 'Timeout', icon: 'clock' },

    // Health statuses
    healthy: { key: 'healthy', label: 'Healthy', icon: 'check-circle' },
    degraded: { key: 'degraded', label: 'Degraded', icon: 'alert-triangle' },

    // Default fallback
    unknown: { key: 'unknown', label: 'Unknown', icon: 'help-circle' }
};

// Default Settings
export const DEFAULT_SETTINGS = {
    coverageTarget: 80,
    phases: new Set(DEFAULT_GENERATION_PHASES)
};

// Timing Constants
export const TIMING = {
    HEALTH_CHECK_INTERVAL: 30000,      // 30 seconds
    DASHBOARD_UPDATE_INTERVAL: 60000,  // 60 seconds
    WEBSOCKET_RECONNECT_DELAY: 5000,   // 5 seconds
    NOTIFICATION_DURATION: 5000,       // 5 seconds
    NOTIFICATION_SLIDE_DELAY: 100,     // 100ms
    NOTIFICATION_FADE_DURATION: 300    // 300ms
};

// Reports Configuration
export const REPORTS = {
    DEFAULT_WINDOW_DAYS: 30,
    COVERAGE_TARGET: 95,
    CHART_HEIGHT: 220,
    CHART_MARGINS: { top: 20, right: 24, bottom: 32, left: 40 }
};

// Coverage Configuration
export const COVERAGE = {
    MIN_TARGET: 50,
    MAX_TARGET: 100,
    DEFAULT_TARGET: 80
};

// Execution Configuration
export const EXECUTION = {
    DEFAULT_TIMEOUT: 300,  // 5 minutes in seconds
};

// UI Configuration
export const UI = {
    MAX_RECENT_EXECUTIONS: 10,
    TABLE_PAGE_SIZE: 50,
    DRAG_SCROLL_SKIP_SELECTOR: 'button, a, input, select, textarea, [role="button"]'
};

// Regular Expression Patterns
export const PATTERNS = {
    UUID: /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,
    ISO_DATE: /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/
};

// Chart Configuration
export const CHART = {
    GRID_PERCENTAGES: [0, 25, 50, 75, 100],
    LINE_COLOR: 'rgba(0, 255, 65, 0.8)',
    POINT_COLOR: 'rgba(0, 255, 65, 1)',
    BAR_COLOR: 'rgba(255, 107, 53, 0.6)',
    GRID_COLOR: 'rgba(255, 255, 255, 0.08)',
    TEXT_COLOR: 'rgba(255, 255, 255, 0.35)',
    FONT: '10px Inter, sans-serif',
    BAR_HEIGHT_RATIO: 0.35
};

// Error Messages
export const ERROR_MESSAGES = {
    API_CALL_FAILED: 'API call failed',
    NO_DATA_AVAILABLE: 'No data available',
    LOAD_FAILED: 'Failed to load data',
    HEALTH_CHECK_FAILED: 'Health check failed',
    REQUIRED_FIELDS: 'Please fill in all required fields',
    INVALID_INPUT: 'Invalid input provided',
    OPERATION_FAILED: 'Operation failed'
};

// Success Messages
export const SUCCESS_MESSAGES = {
    DATA_LOADED: 'Data loaded successfully',
    DATA_REFRESHED: 'Data refreshed',
    PAGE_REFRESHED: 'Page refreshed!',
    OPERATION_COMPLETE: 'Operation completed successfully',
    SETTINGS_SAVED: 'Settings saved'
};

// Info Messages
export const INFO_MESSAGES = {
    NO_CHANGES: 'No changes detected',
    LOADING: 'Loading...',
    PROCESSING: 'Processing...'
};
