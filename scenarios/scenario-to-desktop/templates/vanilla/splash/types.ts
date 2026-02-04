/**
 * Splash Window Type Definitions
 *
 * DOC: docs/internal/SEAMS.md#splash-window-architecture
 *
 * Defines the contract between splash window components.
 * These types establish clear boundaries for testing and implementation.
 */

/**
 * Status phases during application startup.
 * Each phase represents a distinct step in the startup process.
 */
export type StartupPhase =
    | "initializing"
    | "validating-bundle"
    | "allocating-ports"
    | "starting-runtime"
    | "waiting-for-token"
    | "checking-health"
    | "checking-ready"
    | "discovering-ports"
    | "loading-ui"
    | "ready"
    | "error";

/**
 * Status update sent to the splash window via IPC.
 */
export interface SplashStatus {
    /** Current startup phase */
    phase: StartupPhase;
    /** Human-readable message to display */
    message: string;
    /** Progress percentage (0-100), if determinable */
    progress?: number;
    /** Error details if phase is "error" */
    error?: {
        title: string;
        message: string;
        recoverable: boolean;
        /** Detailed logs for debugging (shown in expandable panel) */
        logs?: string[];
        /** Stderr output from runtime process */
        stderr?: string;
        /** Exit code if process crashed */
        exitCode?: number | null;
        /** Suggested action for user */
        suggestion?: string;
    };
}

/**
 * Log entry for streaming to splash window.
 * DOC: docs/internal/SEAMS.md#splash-log-streaming
 */
export interface SplashLogEntry {
    /** Timestamp of the log entry */
    timestamp: Date;
    /** Log level */
    level: "info" | "warn" | "error";
    /** Log message */
    message: string;
    /** Optional source/component */
    source?: string;
}

/**
 * Configuration for the splash window.
 * Separating config from implementation enables testing with different configurations.
 */
export interface SplashWindowConfig {
    /** Window width in pixels */
    width: number;
    /** Window height in pixels */
    height: number;
    /** Whether the window should have a frame */
    frame: boolean;
    /**
     * Whether the window should stay on top of other windows.
     * WARNING: Setting to true can cause focus trapping issues.
     * @default false
     */
    alwaysOnTop: boolean;
    /** Whether the window background should be transparent */
    transparent: boolean;
    /** Path to the splash HTML file relative to app root */
    htmlPath: string;
    /** Path to the splash preload script */
    preloadPath: string;
    /**
     * Whether to allow escape key to force-close the splash.
     * Useful as an emergency exit when startup hangs.
     * @default true
     */
    allowEscapeClose: boolean;
}

/**
 * Result of attempting to close the splash window.
 */
export interface SplashCloseResult {
    /** Whether the close operation succeeded */
    success: boolean;
    /** Error message if close failed */
    error?: string;
    /** Whether the window was already closed */
    alreadyClosed?: boolean;
}

/**
 * Interface for the splash window manager.
 * This is the main seam for testing splash window behavior.
 * DOC: docs/internal/SEAMS.md#splash-window-manager
 */
export interface ISplashWindowManager {
    /**
     * Create and show the splash window.
     * @returns true if window was created successfully
     */
    create(): Promise<boolean>;

    /**
     * Send a status update to the splash window.
     * @param status - The status to display
     */
    updateStatus(status: SplashStatus): void;

    /**
     * Close the splash window gracefully.
     * Uses destroy() to ensure immediate closure.
     * @returns Result of the close operation
     */
    close(): Promise<SplashCloseResult>;

    /**
     * Check if the splash window exists and is visible.
     */
    isVisible(): boolean;

    /**
     * Register a callback for when escape key is pressed.
     * @param callback - Function to call on escape
     */
    onEscapePressed(callback: () => void): void;

    /**
     * Append a log entry to the splash window's log panel.
     * @param entry - The log entry to append
     */
    appendLog(entry: SplashLogEntry): void;

    /**
     * Clear all logs from the splash window's log panel.
     */
    clearLogs(): void;

    /**
     * Register a callback for when user requests to copy logs.
     * @param callback - Function that returns the logs to copy
     */
    onCopyLogs(callback: () => string): void;

    /**
     * Register a callback for when user requests retry.
     * @param callback - Function to call on retry
     */
    onRetry(callback: () => void): void;
}

/**
 * Interface for creating BrowserWindow instances.
 * This seam allows mocking Electron in tests.
 */
export interface IWindowFactory {
    createWindow(options: Electron.BrowserWindowConstructorOptions): Electron.BrowserWindow;
}

/**
 * IPC channel names for splash window communication.
 * Centralized to prevent typos and enable compile-time checking.
 */
export const SPLASH_IPC_CHANNELS = {
    /** Main -> Splash: Status update */
    STATUS_UPDATE: "splash:status-update",
    /** Splash -> Main: Escape key pressed */
    ESCAPE_PRESSED: "splash:escape-pressed",
    /** Splash -> Main: Splash ready (loaded) */
    READY: "splash:ready",
    /** Main -> Splash: Append log entry */
    LOG_APPEND: "splash:log-append",
    /** Main -> Splash: Clear all logs */
    LOG_CLEAR: "splash:log-clear",
    /** Splash -> Main: Request to copy logs to clipboard */
    COPY_LOGS: "splash:copy-logs",
    /** Main -> Splash: Confirm logs copied */
    COPY_LOGS_RESULT: "splash:copy-logs-result",
    /** Splash -> Main: Request retry */
    RETRY: "splash:retry",
} as const;

/**
 * Default splash window configuration.
 * These defaults prioritize user experience over visual aesthetics.
 */
export const DEFAULT_SPLASH_CONFIG: SplashWindowConfig = {
    width: 400,
    height: 300,
    frame: false,
    alwaysOnTop: false, // Changed from true to prevent focus trapping
    transparent: true,
    htmlPath: "src/splash.html",
    preloadPath: "splash-preload.js",
    allowEscapeClose: true,
};

/**
 * Human-readable messages for each startup phase.
 * These messages are shown to the user during startup.
 */
export const PHASE_MESSAGES: Record<StartupPhase, string> = {
    initializing: "Initializing...",
    "validating-bundle": "Validating application bundle...",
    "allocating-ports": "Allocating network ports...",
    "starting-runtime": "Starting runtime services...",
    "waiting-for-token": "Waiting for authentication...",
    "checking-health": "Checking service health...",
    "checking-ready": "Verifying services are ready...",
    "discovering-ports": "Discovering service endpoints...",
    "loading-ui": "Loading user interface...",
    ready: "Ready!",
    error: "An error occurred",
};
