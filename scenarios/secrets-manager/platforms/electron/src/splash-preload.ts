/**
 * Splash Window Preload Script
 *
 * DOC: docs/internal/SEAMS.md#splash-window-ipc
 *
 * This preload script establishes the IPC bridge between the main process
 * and the splash window. It exposes a minimal, secure API for:
 * - Receiving status updates from the main process
 * - Sending escape key events back to main process
 *
 * Security: Uses contextBridge to prevent direct access to Node.js APIs.
 */

import { contextBridge, ipcRenderer } from "electron";

/**
 * Status update received from the main process.
 * Mirrors the SplashStatus type from types.ts.
 */
interface SplashStatus {
    phase: string;
    message: string;
    progress?: number;
    error?: {
        title: string;
        message: string;
        recoverable: boolean;
        logs?: string[];
        stderr?: string;
        exitCode?: number | null;
        suggestion?: string;
    };
}

/**
 * Log entry received from the main process.
 */
interface SplashLogEntry {
    timestamp?: Date;
    level: "info" | "warn" | "error";
    message: string;
    source?: string;
}

/**
 * Result of copy logs operation.
 */
interface CopyLogsResult {
    success: boolean;
    logs?: string;
    error?: string;
}

/**
 * Callback type for status update listeners.
 */
type StatusUpdateCallback = (status: SplashStatus) => void;
type LogAppendCallback = (entry: SplashLogEntry) => void;
type LogClearCallback = () => void;
type CopyLogsResultCallback = (result: CopyLogsResult) => void;

/**
 * API exposed to the splash window's renderer context.
 * This is the only way the splash HTML can communicate with main.
 */
interface SplashAPI {
    /**
     * Register a callback for status updates.
     * @param callback - Function called when status changes
     */
    onStatusUpdate: (callback: StatusUpdateCallback) => void;

    /**
     * Remove a status update listener.
     * @param callback - The callback to remove
     */
    offStatusUpdate: (callback: StatusUpdateCallback) => void;

    /**
     * Notify main process that escape key was pressed.
     * Used as emergency exit for stuck splash screens.
     */
    notifyEscape: () => void;

    /**
     * Notify main process that splash is ready (fully loaded).
     */
    notifyReady: () => void;

    /**
     * Register a callback for log entries.
     * @param callback - Function called when a log is appended
     */
    onLogAppend: (callback: LogAppendCallback) => void;

    /**
     * Register a callback for log clear events.
     * @param callback - Function called when logs are cleared
     */
    onLogClear: (callback: LogClearCallback) => void;

    /**
     * Request to copy logs to clipboard.
     * Main process will handle clipboard access.
     */
    copyLogs: () => void;

    /**
     * Register a callback for copy logs result.
     * @param callback - Function called with the result
     */
    onCopyLogsResult: (callback: CopyLogsResultCallback) => void;

    /**
     * Request retry after an error.
     */
    retry: () => void;
}

// Store callbacks for cleanup
const statusCallbacks = new Set<StatusUpdateCallback>();
const logAppendCallbacks = new Set<LogAppendCallback>();
const logClearCallbacks = new Set<LogClearCallback>();
const copyLogsResultCallbacks = new Set<CopyLogsResultCallback>();

// IPC channel names (must match SPLASH_IPC_CHANNELS in types.ts)
const CHANNELS = {
    STATUS_UPDATE: "splash:status-update",
    ESCAPE_PRESSED: "splash:escape-pressed",
    READY: "splash:ready",
    LOG_APPEND: "splash:log-append",
    LOG_CLEAR: "splash:log-clear",
    COPY_LOGS: "splash:copy-logs",
    COPY_LOGS_RESULT: "splash:copy-logs-result",
    RETRY: "splash:retry",
};

// Create the splash API
const splashAPI: SplashAPI = {
    onStatusUpdate: (callback: StatusUpdateCallback) => {
        statusCallbacks.add(callback);

        // Create the IPC handler if this is the first callback
        if (statusCallbacks.size === 1) {
            ipcRenderer.on(CHANNELS.STATUS_UPDATE, (_event, status: SplashStatus) => {
                statusCallbacks.forEach((cb) => {
                    try {
                        cb(status);
                    } catch (error) {
                        console.error("[Splash] Error in status callback:", error);
                    }
                });
            });
        }
    },

    offStatusUpdate: (callback: StatusUpdateCallback) => {
        statusCallbacks.delete(callback);

        // Remove the IPC handler if no callbacks remain
        if (statusCallbacks.size === 0) {
            ipcRenderer.removeAllListeners(CHANNELS.STATUS_UPDATE);
        }
    },

    notifyEscape: () => {
        console.log("[Splash] Escape key pressed, notifying main process");
        ipcRenderer.send(CHANNELS.ESCAPE_PRESSED);
    },

    notifyReady: () => {
        console.log("[Splash] Splash window ready");
        ipcRenderer.send(CHANNELS.READY);
    },

    onLogAppend: (callback: LogAppendCallback) => {
        logAppendCallbacks.add(callback);

        if (logAppendCallbacks.size === 1) {
            ipcRenderer.on(CHANNELS.LOG_APPEND, (_event, entry: SplashLogEntry) => {
                logAppendCallbacks.forEach((cb) => {
                    try {
                        cb(entry);
                    } catch (error) {
                        console.error("[Splash] Error in log append callback:", error);
                    }
                });
            });
        }
    },

    onLogClear: (callback: LogClearCallback) => {
        logClearCallbacks.add(callback);

        if (logClearCallbacks.size === 1) {
            ipcRenderer.on(CHANNELS.LOG_CLEAR, () => {
                logClearCallbacks.forEach((cb) => {
                    try {
                        cb();
                    } catch (error) {
                        console.error("[Splash] Error in log clear callback:", error);
                    }
                });
            });
        }
    },

    copyLogs: () => {
        console.log("[Splash] Copy logs requested");
        ipcRenderer.send(CHANNELS.COPY_LOGS);
    },

    onCopyLogsResult: (callback: CopyLogsResultCallback) => {
        copyLogsResultCallbacks.add(callback);

        if (copyLogsResultCallbacks.size === 1) {
            ipcRenderer.on(CHANNELS.COPY_LOGS_RESULT, (_event, result: CopyLogsResult) => {
                copyLogsResultCallbacks.forEach((cb) => {
                    try {
                        cb(result);
                    } catch (error) {
                        console.error("[Splash] Error in copy logs result callback:", error);
                    }
                });
            });
        }
    },

    retry: () => {
        console.log("[Splash] Retry requested");
        ipcRenderer.send(CHANNELS.RETRY);
    },
};

// Expose the API to the renderer process
contextBridge.exposeInMainWorld("splashAPI", splashAPI);

// Add TypeScript declarations for the window object
declare global {
    interface Window {
        splashAPI: SplashAPI;
    }
}

// Log initialization
console.log("[Splash Preload] Initialized successfully");
