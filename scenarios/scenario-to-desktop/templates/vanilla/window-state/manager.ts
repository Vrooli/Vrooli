/**
 * Window State Manager
 *
 * DOC: docs/internal/SEAMS.md#window-state-manager
 *
 * Manages window state persistence across app restarts.
 * This module provides:
 * - Automatic state saving on window close
 * - State restoration with display validation
 * - Multi-monitor support
 *
 * Testing Seams:
 * - IStateStorage: Mock state persistence
 * - IDisplayProvider: Mock display enumeration
 * - IManagedWindow: Mock BrowserWindow
 *
 * Responsibilities:
 * - Orchestrate state loading and saving
 * - Validate state against current displays
 * - Attach to window events for automatic saving
 *
 * NOT responsible for:
 * - File I/O (delegated to IStateStorage)
 * - Display enumeration (delegated to IDisplayProvider)
 * - State validation logic (delegated to validator functions)
 */

import * as fs from "fs";
import type {
    IDisplayProvider,
    IManagedWindow,
    IStateStorage,
    IWindowStateManager,
    WindowState,
    WindowStateConfig,
} from "./types";
import { createDefaultState, DEFAULT_WINDOW_STATE_CONFIG } from "./types";
import { validateWindowState } from "./validator";

// Debug file logger for diagnosing fullscreen persistence issues
function createDebugLogger(logPath: string) {
    return (message: string, ...args: unknown[]): void => {
        const timestamp = new Date().toISOString();
        const argsStr = args.map(a => typeof a === "object" ? JSON.stringify(a) : String(a)).join(" ");
        const line = `[${timestamp}] ${message} ${argsStr}\n`;
        try {
            fs.appendFileSync(logPath, line);
        } catch {
            // Ignore write errors
        }
        console.log("[WindowStateManager]", message, ...args);
    };
}

/**
 * Dependencies for WindowStateManager.
 * Injecting these allows testing without Electron.
 */
export interface WindowStateManagerDeps {
    storage: IStateStorage;
    displayProvider: IDisplayProvider;
    /** Optional logger for debugging */
    log?: (message: string, ...args: unknown[]) => void;
    /** Optional path for debug log file (enables file-based logging for diagnostics) */
    debugLogPath?: string;
}

/**
 * Default implementation of WindowStateManager.
 *
 * Usage pattern:
 * 1. Call getInitialState() before creating BrowserWindow
 * 2. Create BrowserWindow with returned state
 * 3. Call manage(window) to enable auto-save
 * 4. Call wasMaximized()/wasFullScreen() to restore display state
 */
export class WindowStateManager implements IWindowStateManager {
    private readonly deps: WindowStateManagerDeps;
    private readonly config: Required<WindowStateConfig>;
    private readonly log: (message: string, ...args: unknown[]) => void;

    private currentState: WindowState | null = null;
    private managedWindow: IManagedWindow | null = null;
    private saveHandler: (() => void) | null = null;
    private loadedState: WindowState | null = null;

    /**
     * Tracked fullscreen state, updated via enter/leave-full-screen events.
     *
     * On macOS, when closing a fullscreen window:
     * 1. leave-full-screen fires FIRST (as animation completes)
     * 2. close event fires SECOND
     *
     * To handle this, we use a debounce approach: when leave-full-screen fires,
     * we wait a short time before setting trackedFullScreenState to false.
     * If the close event fires during that time, we cancel the debounce timer
     * and preserve the fullscreen state.
     *
     * @see https://github.com/electron/electron/issues/20263
     * @see https://github.com/mawie81/electron-window-state
     */
    private trackedFullScreenState: boolean = false;
    private leaveFullScreenDebounceTimer: ReturnType<typeof setTimeout> | null = null;
    private enterFullScreenHandler: (() => void) | null = null;
    private leaveFullScreenHandler: (() => void) | null = null;
    private resizeHandler: (() => void) | null = null;

    constructor(
        deps: WindowStateManagerDeps,
        config: Partial<WindowStateConfig> = {}
    ) {
        this.deps = deps;
        this.config = { ...DEFAULT_WINDOW_STATE_CONFIG, ...config };
        // Use file-based debug logger if debugLogPath is provided
        if (deps.debugLogPath) {
            this.log = createDebugLogger(deps.debugLogPath);
        } else {
            this.log = deps.log ?? ((...args) => console.log("[WindowStateManager]", ...args));
        }
    }

    /**
     * Get the initial window state for BrowserWindow creation.
     *
     * Process:
     * 1. Load saved state from storage
     * 2. Validate against current displays
     * 3. Adjust if necessary (e.g., display disconnected)
     * 4. Return defaults if no valid saved state
     */
    async getInitialState(): Promise<WindowState> {
        try {
            // Load saved state
            const savedState = await this.deps.storage.load();

            if (!savedState) {
                this.log("No saved state, using defaults");
                const defaultState = createDefaultState(this.config);
                this.currentState = this.centerOnPrimaryDisplay(defaultState);
                return this.currentState;
            }

            // Validate against current displays
            const displays = this.deps.displayProvider.getAllDisplays();
            const primaryDisplay = this.deps.displayProvider.getPrimaryDisplay();

            const validationResult = validateWindowState(
                savedState,
                displays,
                primaryDisplay,
                this.config
            );

            if (validationResult.adjustmentReason) {
                this.log(`State adjusted: ${validationResult.adjustmentReason}`);
            }

            // Store the loaded state for wasMaximized/wasFullScreen checks
            this.loadedState = savedState;
            this.currentState = validationResult.state;
            return this.currentState;
        } catch (error) {
            this.log("Error loading state, using defaults:", error);
            const defaultState = createDefaultState(this.config);
            this.currentState = this.centerOnPrimaryDisplay(defaultState);
            return this.currentState;
        }
    }

    /**
     * Attach to a BrowserWindow to track and save state changes.
     *
     * Registers event listeners for:
     * - close: Save final state before window closes
     * - enter-full-screen: Track when window enters fullscreen
     * - leave-full-screen: Track when window leaves fullscreen
     *
     * Note: We save on 'close' rather than 'resize'/'move' to avoid
     * excessive disk writes and to capture the final state accurately.
     *
     * Fullscreen state is tracked via events rather than reading isFullScreen()
     * at close time, because on macOS the fullscreen exit animation completes
     * before the close event fires.
     */
    manage(window: IManagedWindow): void {
        if (this.managedWindow) {
            this.log("Warning: Already managing a window, detaching from previous");
            this.detach();
        }

        this.managedWindow = window;

        // Initialize tracked fullscreen state from current window state
        this.trackedFullScreenState = window.isFullScreen();
        this.leaveFullScreenDebounceTimer = null;
        this.log("Initial fullscreen state:", this.trackedFullScreenState);

        // Create fullscreen state handlers
        this.enterFullScreenHandler = () => {
            // Cancel any pending debounce timer
            if (this.leaveFullScreenDebounceTimer) {
                clearTimeout(this.leaveFullScreenDebounceTimer);
                this.leaveFullScreenDebounceTimer = null;
            }
            this.trackedFullScreenState = true;
            this.log("Window entered fullscreen, trackedFullScreenState =", this.trackedFullScreenState);
            // Save state immediately on fullscreen change for reliability
            this.captureCurrentState();
            void this.saveState();
        };

        this.leaveFullScreenHandler = () => {
            // Use debounce to handle macOS timing issue where leave-full-screen
            // fires BEFORE close when closing a fullscreen window.
            // If close fires within the debounce window, we preserve fullscreen state.
            this.log("leave-full-screen event fired, starting debounce timer");
            if (this.leaveFullScreenDebounceTimer) {
                clearTimeout(this.leaveFullScreenDebounceTimer);
            }
            this.leaveFullScreenDebounceTimer = setTimeout(() => {
                this.leaveFullScreenDebounceTimer = null;
                this.trackedFullScreenState = false;
                this.log("Window left fullscreen (debounced), trackedFullScreenState =", this.trackedFullScreenState);
                // Save state after leaving fullscreen (user pressed Escape, not closing)
                this.captureCurrentState();
                void this.saveState();
            }, 200); // 200ms debounce - long enough to catch close event
        };

        // Create save handler
        this.saveHandler = () => {
            // Check isFullScreen() as fallback in case events didn't fire (Linux WM issues)
            const isCurrentlyFullScreen = this.managedWindow?.isFullScreen() ?? false;
            this.log("Close event fired, isFullScreen() =", isCurrentlyFullScreen, ", trackedFullScreenState =", this.trackedFullScreenState);

            // Cancel debounce timer to preserve fullscreen state if leave-full-screen
            // fired just before close (macOS behavior)
            if (this.leaveFullScreenDebounceTimer) {
                clearTimeout(this.leaveFullScreenDebounceTimer);
                this.leaveFullScreenDebounceTimer = null;
                this.log("Cancelled leave-fullscreen debounce, preserving fullscreen state");
            }

            // Use tracked state OR current state (fallback for Linux where events may not fire)
            if (isCurrentlyFullScreen && !this.trackedFullScreenState) {
                this.log("Fullscreen detected via isFullScreen() but not tracked - using isFullScreen() value");
                this.trackedFullScreenState = true;
            }

            if (this.managedWindow && !this.managedWindow.isDestroyed()) {
                this.captureCurrentState();
                void this.saveState();
            }
        };

        // Register fullscreen state change handlers
        window.on("enter-full-screen", this.enterFullScreenHandler);
        window.on("leave-full-screen", this.leaveFullScreenHandler);

        // Register resize handler for Linux fullscreen detection
        // On Linux with some WMs, enter-full-screen events don't fire, so we
        // detect fullscreen heuristically by checking if window covers the display
        this.resizeHandler = () => {
            const isFullScreenNow = window.isFullScreen();
            const isMaximizedNow = window.isMaximized();
            const bounds = window.getBounds();

            // Heuristic: Check if window covers the primary display (Linux fullscreen detection)
            const primaryDisplay = this.deps.displayProvider.getPrimaryDisplay();
            const coversDisplay = bounds.x <= primaryDisplay.x &&
                bounds.y <= primaryDisplay.y &&
                bounds.width >= primaryDisplay.width &&
                bounds.height >= primaryDisplay.height;

            this.log("Resize event: isFullScreen()=", isFullScreenNow,
                "isMaximized()=", isMaximizedNow,
                "bounds=", bounds,
                "display=", primaryDisplay,
                "coversDisplay=", coversDisplay,
                "trackedFullScreenState=", this.trackedFullScreenState);

            // If window covers display and events didn't fire, assume fullscreen
            if (coversDisplay && !this.trackedFullScreenState && !isMaximizedNow) {
                this.log("Heuristic fullscreen detection: window covers display, setting trackedFullScreenState=true");
                this.trackedFullScreenState = true;
                this.captureCurrentState();
                void this.saveState();
            }
        };
        window.on("resize", this.resizeHandler);

        // Register close handler - this is when we save
        window.on("close", this.saveHandler);

        this.log("Now managing window");
    }

    /**
     * Force save the current window state immediately.
     */
    async saveState(): Promise<void> {
        if (!this.currentState) {
            this.log("No state to save");
            return;
        }

        try {
            await this.deps.storage.save(this.currentState);
        } catch (error) {
            this.log("Failed to save state:", error);
        }
    }

    /**
     * Check if the window was previously maximized.
     * Should be called after manage() to determine if maximize() should be called.
     */
    wasMaximized(): boolean {
        if (!this.config.restoreMaximized) {
            return false;
        }
        return this.loadedState?.isMaximized ?? false;
    }

    /**
     * Check if the window was previously in full-screen.
     * Should be called after manage() to determine if setFullScreen(true) should be called.
     */
    wasFullScreen(): boolean {
        if (!this.config.restoreFullScreen) {
            return false;
        }
        return this.loadedState?.isFullScreen ?? false;
    }

    /**
     * Capture the current window state from the managed window.
     * Uses getNormalBounds() to get the non-maximized size.
     *
     * Note: Uses trackedFullScreenState instead of isFullScreen() because
     * on macOS, the fullscreen exit animation completes before the close
     * event fires, making isFullScreen() unreliable at close time.
     */
    private captureCurrentState(): void {
        if (!this.managedWindow || this.managedWindow.isDestroyed()) {
            return;
        }

        try {
            // getNormalBounds returns the bounds when not maximized/fullscreen
            const bounds = this.managedWindow.getNormalBounds();

            this.currentState = {
                x: bounds.x,
                y: bounds.y,
                width: bounds.width,
                height: bounds.height,
                isMaximized: this.managedWindow.isMaximized(),
                // Use tracked state instead of isFullScreen() due to macOS timing issue
                isFullScreen: this.trackedFullScreenState,
            };

            this.log("Captured state:", this.currentState);
        } catch (error) {
            this.log("Error capturing state:", error);
        }
    }

    /**
     * Detach from the currently managed window.
     */
    private detach(): void {
        if (this.managedWindow) {
            try {
                if (this.saveHandler) {
                    this.managedWindow.removeListener("close", this.saveHandler);
                }
                if (this.enterFullScreenHandler) {
                    this.managedWindow.removeListener("enter-full-screen", this.enterFullScreenHandler);
                }
                if (this.leaveFullScreenHandler) {
                    this.managedWindow.removeListener("leave-full-screen", this.leaveFullScreenHandler);
                }
                if (this.resizeHandler) {
                    this.managedWindow.removeListener("resize", this.resizeHandler);
                }
            } catch {
                // Window may already be destroyed
            }
        }
        this.managedWindow = null;
        this.saveHandler = null;
        this.enterFullScreenHandler = null;
        this.leaveFullScreenHandler = null;
        this.resizeHandler = null;
        this.trackedFullScreenState = false;
        if (this.leaveFullScreenDebounceTimer) {
            clearTimeout(this.leaveFullScreenDebounceTimer);
            this.leaveFullScreenDebounceTimer = null;
        }
    }

    /**
     * Center a state on the primary display if no position is set.
     */
    private centerOnPrimaryDisplay(state: WindowState): WindowState {
        if (state.x !== undefined && state.y !== undefined) {
            return state;
        }

        const primary = this.deps.displayProvider.getPrimaryDisplay();
        return {
            ...state,
            x: Math.round(primary.x + (primary.width - state.width) / 2),
            y: Math.round(primary.y + (primary.height - state.height) / 2),
        };
    }
}

/**
 * Factory function to create WindowStateManager with all dependencies.
 *
 * This is the production factory - tests should create WindowStateManager
 * directly with mock dependencies.
 */
export function createWindowStateManager(
    storage: IStateStorage,
    displayProvider: IDisplayProvider,
    config?: Partial<WindowStateConfig>
): WindowStateManager {
    return new WindowStateManager(
        { storage, displayProvider },
        config
    );
}
