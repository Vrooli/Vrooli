/**
 * Splash Window Manager
 *
 * DOC: docs/internal/SEAMS.md#splash-window-manager
 *
 * Manages the splash window lifecycle with proper separation of concerns:
 * - Window creation and destruction
 * - Status communication via IPC
 * - Escape key handling for emergency exit
 *
 * Testing Seams:
 * - IWindowFactory: Mock Electron BrowserWindow creation
 * - IPathResolver: Mock path resolution for different environments
 * - Event callbacks: Directly test escape handling
 */

import type { BrowserWindow, IpcMainEvent } from "electron";
import type {
    ISplashWindowManager,
    IWindowFactory,
    SplashCloseResult,
    SplashLogEntry,
    SplashStatus,
    SplashWindowConfig,
} from "./types";
import { DEFAULT_SPLASH_CONFIG, SPLASH_IPC_CHANNELS } from "./types";

/**
 * Interface for resolving file paths.
 * Seam for testing path resolution without filesystem access.
 */
export interface IPathResolver {
    /** Get the app's root path */
    getAppPath(): string;
    /** Join path segments */
    join(...segments: string[]): string;
}

/**
 * Interface for IPC main process operations.
 * Seam for testing IPC without Electron.
 */
export interface IIpcMain {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Matches Electron's IPC API signature
    on(channel: string, listener: (event: IpcMainEvent, ...args: any[]) => void): void;
    removeAllListeners(channel: string): void;
}

/**
 * Interface for clipboard operations.
 * Seam for testing clipboard without Electron.
 */
export interface IClipboard {
    writeText(text: string): void;
}

/**
 * Dependencies required by SplashWindowManager.
 * Injecting these allows testing without Electron.
 */
export interface SplashManagerDeps {
    windowFactory: IWindowFactory;
    pathResolver: IPathResolver;
    ipcMain: IIpcMain;
    /** Optional clipboard for copy operations */
    clipboard?: IClipboard;
    /** Optional logger for debugging */
    // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Logger args are intentionally flexible
    log?: (message: string, ...args: any[]) => void;
}

/**
 * Default implementation of SplashWindowManager.
 *
 * Responsibilities:
 * - Create splash window with proper configuration
 * - Send status updates via IPC
 * - Handle escape key for emergency exit
 * - Clean up resources on close
 *
 * NOT responsible for:
 * - Application startup logic
 * - Error dialog display
 * - Main window management
 */
export class SplashWindowManager implements ISplashWindowManager {
    private window: BrowserWindow | null = null;
    private escapeCallbacks: Array<() => void> = [];
    private copyLogsCallback: (() => string) | null = null;
    private retryCallback: (() => void) | null = null;
    private readonly config: SplashWindowConfig;
    private readonly deps: SplashManagerDeps;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Logger args are intentionally flexible
    private readonly log: (message: string, ...args: any[]) => void;

    constructor(
        deps: SplashManagerDeps,
        config: Partial<SplashWindowConfig> = {}
    ) {
        this.deps = deps;
        this.config = { ...DEFAULT_SPLASH_CONFIG, ...config };
        this.log = deps.log ?? ((...args) => console.log("[SplashManager]", ...args));
    }

    /**
     * Create and show the splash window.
     *
     * @returns true if window was created successfully
     */
    async create(): Promise<boolean> {
        if (this.window) {
            this.log("Splash window already exists, skipping creation");
            return true;
        }

        try {
            this.log("Creating splash window...");

            const preloadPath = this.deps.pathResolver.join(
                this.deps.pathResolver.getAppPath(),
                this.config.preloadPath
            );

            this.window = this.deps.windowFactory.createWindow({
                width: this.config.width,
                height: this.config.height,
                frame: this.config.frame,
                alwaysOnTop: this.config.alwaysOnTop,
                transparent: this.config.transparent,
                // Ensure window is focusable but doesn't trap focus
                skipTaskbar: false,
                resizable: false,
                movable: true,
                minimizable: true, // Allow minimizing if user needs to interact with other windows
                webPreferences: {
                    nodeIntegration: false,
                    contextIsolation: true,
                    preload: preloadPath,
                },
            });
            await this.config.onPhase?.("created");
            this.window.on("ready-to-show", () => {
                void Promise.resolve(this.config.onPhase?.("ready_to_show"));
                if (this.window && !this.window.isDestroyed()) void Promise.resolve(this.config.onPhase?.("shown"));
            });

            // Set up IPC listeners
            this.setupIpcListeners();

            // Load the splash HTML
            const htmlPath = this.deps.pathResolver.join(
                this.deps.pathResolver.getAppPath(),
                this.config.htmlPath
            );
            await this.window.loadFile(htmlPath);
            await this.config.onPhase?.("load_completed");
            if (this.window.isVisible()) await this.config.onPhase?.("shown");

            // Set up escape key handling
            if (this.config.allowEscapeClose) {
                this.setupEscapeHandler();
            }

            // Handle window close event
            this.window.on("closed", () => {
                this.log("Splash window closed");
                this.cleanup();
            });

            this.log("Splash window created successfully");
            return true;
        } catch (error) {
            this.log("Failed to create splash window:", error);
            this.cleanup();
            return false;
        }
    }

    /**
     * Set up IPC listeners for splash window communication.
     */
    private setupIpcListeners(): void {
        // Listen for escape key from splash window
        this.deps.ipcMain.on(SPLASH_IPC_CHANNELS.ESCAPE_PRESSED, () => {
            this.log("Escape key received from splash window");
            this.escapeCallbacks.forEach((cb) => {
                try {
                    cb();
                } catch (error) {
                    this.log("Error in escape callback:", error);
                }
            });
        });

        // Listen for splash ready notification
        this.deps.ipcMain.on(SPLASH_IPC_CHANNELS.READY, () => {
            this.log("Splash window reports ready");
            void Promise.resolve(this.config.onPhase?.("first_paint"));
        });

        // Listen for copy logs request
        this.deps.ipcMain.on(SPLASH_IPC_CHANNELS.COPY_LOGS, () => {
            this.log("Copy logs requested from splash window");
            if (this.copyLogsCallback && this.window && !this.window.isDestroyed()) {
                try {
                    const logs = this.copyLogsCallback();
                    // Copy to clipboard if clipboard is available
                    if (this.deps.clipboard) {
                        this.deps.clipboard.writeText(logs);
                        this.log("Logs copied to clipboard");
                    }
                    this.window.webContents.send(SPLASH_IPC_CHANNELS.COPY_LOGS_RESULT, { success: true, logs });
                } catch (error) {
                    this.log("Error getting logs for copy:", error);
                    this.window.webContents.send(SPLASH_IPC_CHANNELS.COPY_LOGS_RESULT, { success: false, error: String(error) });
                }
            }
        });

        // Listen for retry request
        this.deps.ipcMain.on(SPLASH_IPC_CHANNELS.RETRY, () => {
            this.log("Retry requested from splash window");
            if (this.retryCallback) {
                try {
                    this.retryCallback();
                } catch (error) {
                    this.log("Error in retry callback:", error);
                }
            }
        });
    }

    /**
     * Set up keyboard handler for escape key.
     * This provides a fallback in case the splash window's preload doesn't load.
     */
    private setupEscapeHandler(): void {
        if (!this.window) return;

        this.window.webContents.on("before-input-event", (_event, input) => {
            if (input.key === "Escape" && input.type === "keyDown") {
                this.log("Escape key detected via webContents");
                this.escapeCallbacks.forEach((cb) => {
                    try {
                        cb();
                    } catch (error) {
                        this.log("Error in escape callback:", error);
                    }
                });
            }
        });
    }

    /**
     * Send a status update to the splash window.
     *
     * @param status - The status to display
     */
    updateStatus(status: SplashStatus): void {
        if (!this.window || this.window.isDestroyed()) {
            this.log("Cannot update status: window not available");
            return;
        }

        try {
            this.window.webContents.send(SPLASH_IPC_CHANNELS.STATUS_UPDATE, status);
            this.log(`Status updated: ${status.phase} - ${status.message}`);
        } catch (error) {
            this.log("Failed to send status update:", error);
        }
    }

    /**
     * Close the splash window gracefully.
     *
     * Uses destroy() instead of close() to ensure immediate closure
     * and prevent race conditions with error dialogs.
     *
     * @returns Result of the close operation
     */
    async close(): Promise<SplashCloseResult> {
        if (!this.window) {
            return { success: true, alreadyClosed: true };
        }

        if (this.window.isDestroyed()) {
            this.cleanup();
            return { success: true, alreadyClosed: true };
        }

        try {
            this.log("Closing splash window...");

            // Use destroy() for immediate, guaranteed closure
            // This prevents race conditions where close() might be deferred
            this.window.destroy();

            // Small delay to ensure the window is fully destroyed
            // This prevents issues with error dialogs appearing behind a "ghost" window
            await new Promise((resolve) => setTimeout(resolve, 50));

            this.cleanup();
            this.log("Splash window closed successfully");
            return { success: true };
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            this.log("Error closing splash window:", errorMessage);

            // Force cleanup even if destroy failed
            this.cleanup();

            return { success: false, error: errorMessage };
        }
    }

    /**
     * Check if the splash window exists and is visible.
     */
    isVisible(): boolean {
        return this.window !== null &&
            !this.window.isDestroyed() &&
            this.window.isVisible();
    }

    /**
     * Register a callback for when escape key is pressed.
     *
     * @param callback - Function to call on escape
     */
    onEscapePressed(callback: () => void): void {
        this.escapeCallbacks.push(callback);
    }

    /**
     * Append a log entry to the splash window's log panel.
     *
     * @param entry - The log entry to append
     */
    appendLog(entry: SplashLogEntry): void {
        if (!this.window || this.window.isDestroyed()) {
            return;
        }

        try {
            this.window.webContents.send(SPLASH_IPC_CHANNELS.LOG_APPEND, entry);
        } catch (error) {
            this.log("Failed to append log:", error);
        }
    }

    /**
     * Clear all logs from the splash window's log panel.
     */
    clearLogs(): void {
        if (!this.window || this.window.isDestroyed()) {
            return;
        }

        try {
            this.window.webContents.send(SPLASH_IPC_CHANNELS.LOG_CLEAR);
        } catch (error) {
            this.log("Failed to clear logs:", error);
        }
    }

    /**
     * Register a callback for when user requests to copy logs.
     *
     * @param callback - Function that returns the logs to copy
     */
    onCopyLogs(callback: () => string): void {
        this.copyLogsCallback = callback;
    }

    /**
     * Register a callback for when user requests retry.
     *
     * @param callback - Function to call on retry
     */
    onRetry(callback: () => void): void {
        this.retryCallback = callback;
    }

    /**
     * Clean up resources.
     */
    private cleanup(): void {
        this.window = null;
        this.escapeCallbacks = [];
        this.copyLogsCallback = null;
        this.retryCallback = null;

        // Remove IPC listeners
        try {
            this.deps.ipcMain.removeAllListeners(SPLASH_IPC_CHANNELS.ESCAPE_PRESSED);
            this.deps.ipcMain.removeAllListeners(SPLASH_IPC_CHANNELS.READY);
            this.deps.ipcMain.removeAllListeners(SPLASH_IPC_CHANNELS.COPY_LOGS);
            this.deps.ipcMain.removeAllListeners(SPLASH_IPC_CHANNELS.RETRY);
        } catch (error) {
            this.log("Error removing IPC listeners:", error);
        }
    }
}

/**
 * Factory function to create a SplashWindowManager with Electron dependencies.
 *
 * This is the production factory - tests should create SplashWindowManager
 * directly with mock dependencies.
 */
export function createSplashManager(
    BrowserWindow: typeof Electron.BrowserWindow,
    app: Electron.App,
    ipcMain: Electron.IpcMain,
    path: { join: (...args: string[]) => string },
    clipboard: Electron.Clipboard,
    config?: Partial<SplashWindowConfig>
): SplashWindowManager {
    const deps: SplashManagerDeps = {
        windowFactory: {
            createWindow: (options) => new BrowserWindow(options),
        },
        pathResolver: {
            getAppPath: () => app.getAppPath(),
            join: (...segments) => path.join(...segments),
        },
        ipcMain: {
            on: (channel, listener) => ipcMain.on(channel, listener),
            removeAllListeners: (channel) => ipcMain.removeAllListeners(channel),
        },
        clipboard: {
            writeText: (text) => clipboard.writeText(text),
        },
        log: (...args) => console.log("[Desktop App]", ...args),
    };

    return new SplashWindowManager(deps, config);
}
