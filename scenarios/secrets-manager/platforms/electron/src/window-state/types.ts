/**
 * Window State Type Definitions
 *
 * DOC: docs/internal/SEAMS.md#window-state-architecture
 *
 * Defines the contract for window state persistence across app restarts.
 * These types establish clear boundaries for testing and implementation.
 */

/**
 * Persisted window state data.
 * Stored to disk and restored on app launch.
 */
export interface WindowState {
    /** Window x position in screen coordinates */
    x?: number;
    /** Window y position in screen coordinates */
    y?: number;
    /** Window width in pixels */
    width: number;
    /** Window height in pixels */
    height: number;
    /** Whether the window was maximized */
    isMaximized: boolean;
    /** Whether the window was in full-screen mode */
    isFullScreen: boolean;
    /** Display ID where the window was last positioned (for multi-monitor) */
    displayId?: number;
}

/**
 * Configuration for the window state manager.
 * Separating config from implementation enables testing with different configurations.
 */
export interface WindowStateConfig {
    /** Default window width if no saved state exists */
    defaultWidth: number;
    /** Default window height if no saved state exists */
    defaultHeight: number;
    /** Minimum window width to enforce */
    minWidth?: number;
    /** Minimum window height to enforce */
    minHeight?: number;
    /**
     * Storage file name (without path).
     * Stored in the app's userData directory.
     * @default "window-state.json"
     */
    stateFileName?: string;
    /**
     * Whether to restore maximized state.
     * @default true
     */
    restoreMaximized?: boolean;
    /**
     * Whether to restore full-screen state.
     * @default true
     */
    restoreFullScreen?: boolean;
}

/**
 * Display/monitor bounds information.
 * Abstracted from Electron's Display type for testability.
 */
export interface DisplayBounds {
    /** Display identifier */
    id: number;
    /** X coordinate of display origin */
    x: number;
    /** Y coordinate of display origin */
    y: number;
    /** Display width in pixels */
    width: number;
    /** Display height in pixels */
    height: number;
}

/**
 * Result of validating window state against available displays.
 */
export interface ValidationResult {
    /** Whether the state is valid (window would be visible) */
    isValid: boolean;
    /** The validated/adjusted state */
    state: WindowState;
    /** Reason for adjustment if state was modified */
    adjustmentReason?: string;
}

/**
 * Interface for the window state manager.
 * This is the main seam for testing window state behavior.
 * DOC: docs/internal/SEAMS.md#window-state-manager
 */
export interface IWindowStateManager {
    /**
     * Get the initial window options for BrowserWindow creation.
     * Loads saved state if available, otherwise returns defaults.
     */
    getInitialState(): Promise<WindowState>;

    /**
     * Attach to a BrowserWindow to track and save state changes.
     * @param window - The BrowserWindow to manage
     */
    manage(window: IManagedWindow): void;

    /**
     * Force save the current window state immediately.
     * Normally state is saved automatically on close.
     */
    saveState(): Promise<void>;

    /**
     * Check if the window was previously maximized.
     * Call after manage() to determine if maximize() should be called.
     */
    wasMaximized(): boolean;

    /**
     * Check if the window was previously in full-screen.
     * Call after manage() to determine if setFullScreen(true) should be called.
     */
    wasFullScreen(): boolean;
}

/**
 * Window events that can be listened to.
 * Includes close, resize, move, and fullscreen state changes.
 */
export type WindowEvent = "close" | "resize" | "move" | "enter-full-screen" | "leave-full-screen";

/**
 * Interface for a managed window.
 * Abstracted from Electron's BrowserWindow for testability.
 * DOC: docs/internal/SEAMS.md#window-state-managed-window
 */
export interface IManagedWindow {
    /** Get the window bounds (position and size) when not maximized */
    getNormalBounds(): { x: number; y: number; width: number; height: number };
    /** Get current window bounds */
    getBounds(): { x: number; y: number; width: number; height: number };
    /** Check if window is maximized */
    isMaximized(): boolean;
    /** Check if window is in full-screen mode */
    isFullScreen(): boolean;
    /** Check if window has been destroyed */
    isDestroyed(): boolean;
    /** Register event listener */
    on(event: WindowEvent, callback: () => void): void;
    /** Remove event listener */
    removeListener(event: WindowEvent, callback: () => void): void;
}

/**
 * Interface for state storage operations.
 * Seam for testing without filesystem access.
 * DOC: docs/internal/SEAMS.md#window-state-storage
 */
export interface IStateStorage {
    /**
     * Load saved state from storage.
     * @returns The saved state, or null if no saved state exists
     */
    load(): Promise<WindowState | null>;

    /**
     * Save state to storage.
     * @param state - The state to save
     */
    save(state: WindowState): Promise<void>;
}

/**
 * Interface for display/screen information.
 * Seam for testing multi-monitor scenarios.
 * DOC: docs/internal/SEAMS.md#window-state-display-provider
 */
export interface IDisplayProvider {
    /**
     * Get all connected displays.
     */
    getAllDisplays(): DisplayBounds[];

    /**
     * Get the primary display.
     */
    getPrimaryDisplay(): DisplayBounds;

    /**
     * Get the display containing the given point.
     * @param x - X coordinate
     * @param y - Y coordinate
     * @returns The display containing the point, or null if none
     */
    getDisplayAtPoint(x: number, y: number): DisplayBounds | null;
}

/**
 * Default window state configuration.
 */
export const DEFAULT_WINDOW_STATE_CONFIG: Required<WindowStateConfig> = {
    defaultWidth: 1200,
    defaultHeight: 800,
    minWidth: 400,
    minHeight: 300,
    stateFileName: "window-state.json",
    restoreMaximized: true,
    restoreFullScreen: true,
};

/**
 * Create a default window state with the given dimensions.
 */
export function createDefaultState(config: WindowStateConfig): WindowState {
    return {
        width: config.defaultWidth,
        height: config.defaultHeight,
        isMaximized: false,
        isFullScreen: false,
    };
}
