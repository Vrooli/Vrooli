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

/**
 * Dependencies for WindowStateManager.
 * Injecting these allows testing without Electron.
 */
export interface WindowStateManagerDeps {
    storage: IStateStorage;
    displayProvider: IDisplayProvider;
    /** Optional logger for debugging */
    log?: (message: string, ...args: unknown[]) => void;
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

    constructor(
        deps: WindowStateManagerDeps,
        config: Partial<WindowStateConfig> = {}
    ) {
        this.deps = deps;
        this.config = { ...DEFAULT_WINDOW_STATE_CONFIG, ...config };
        this.log = deps.log ?? ((...args) => console.log("[WindowStateManager]", ...args));
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
     *
     * Note: We save on 'close' rather than 'resize'/'move' to avoid
     * excessive disk writes and to capture the final state accurately.
     */
    manage(window: IManagedWindow): void {
        if (this.managedWindow) {
            this.log("Warning: Already managing a window, detaching from previous");
            this.detach();
        }

        this.managedWindow = window;

        // Create save handler
        this.saveHandler = () => {
            if (this.managedWindow && !this.managedWindow.isDestroyed()) {
                this.captureCurrentState();
                void this.saveState();
            }
        };

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
                isFullScreen: this.managedWindow.isFullScreen(),
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
        if (this.managedWindow && this.saveHandler) {
            try {
                this.managedWindow.removeListener("close", this.saveHandler);
            } catch {
                // Window may already be destroyed
            }
        }
        this.managedWindow = null;
        this.saveHandler = null;
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
