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
    on(channel: string, listener: (event: IpcMainEvent, ...args: any[]) => void): void;
    removeAllListeners(channel: string): void;
}

/**
 * Dependencies required by SplashWindowManager.
 * Injecting these allows testing without Electron.
 */
export interface SplashManagerDeps {
    windowFactory: IWindowFactory;
    pathResolver: IPathResolver;
    ipcMain: IIpcMain;
    /** Optional logger for debugging */
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
    private readonly config: SplashWindowConfig;
    private readonly deps: SplashManagerDeps;
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

            // Set up IPC listeners
            this.setupIpcListeners();

            // Load the splash HTML
            const htmlPath = this.deps.pathResolver.join(
                this.deps.pathResolver.getAppPath(),
                this.config.htmlPath
            );
            await this.window.loadFile(htmlPath);

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
     * Clean up resources.
     */
    private cleanup(): void {
        this.window = null;
        this.escapeCallbacks = [];

        // Remove IPC listeners
        try {
            this.deps.ipcMain.removeAllListeners(SPLASH_IPC_CHANNELS.ESCAPE_PRESSED);
            this.deps.ipcMain.removeAllListeners(SPLASH_IPC_CHANNELS.READY);
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
        log: (...args) => console.log("[Desktop App]", ...args),
    };

    return new SplashWindowManager(deps, config);
}
