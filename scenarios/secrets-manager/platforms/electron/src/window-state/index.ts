/**
 * Window State Module
 *
 * DOC: docs/internal/SEAMS.md#window-state-module
 *
 * Exports all window state components for use in main.ts.
 * This module provides:
 * - Type definitions for type safety
 * - WindowStateManager for state lifecycle
 * - Storage and display abstractions for testability
 * - Pure validation functions for state verification
 *
 * Usage in main.ts:
 * ```typescript
 * import {
 *     WindowStateManager,
 *     createWindowStateStorage,
 *     createDisplayProvider,
 * } from "./window-state";
 *
 * // In createMainWindow():
 * const storage = createWindowStateStorage(app, fs, path);
 * const displayProvider = createDisplayProvider(screen);
 * const stateManager = new WindowStateManager(
 *     { storage, displayProvider },
 *     { defaultWidth: 1200, defaultHeight: 800 }
 * );
 *
 * const state = await stateManager.getInitialState();
 * mainWindow = new BrowserWindow({
 *     x: state.x,
 *     y: state.y,
 *     width: state.width,
 *     height: state.height,
 *     ...
 * });
 *
 * stateManager.manage(mainWindow);
 * if (stateManager.wasMaximized()) mainWindow.maximize();
 * if (stateManager.wasFullScreen()) mainWindow.setFullScreen(true);
 * ```
 */

// ===== Type Exports =====

export type {
    WindowState,
    WindowStateConfig,
    DisplayBounds,
    ValidationResult,
    IWindowStateManager,
    IManagedWindow,
    IStateStorage,
    IDisplayProvider,
} from "./types";

export {
    DEFAULT_WINDOW_STATE_CONFIG,
    createDefaultState,
} from "./types";

// ===== Storage Exports =====

export type {
    IFileSystem,
    IPathProvider,
    StorageDeps,
} from "./storage";

export {
    WindowStateStorage,
    createWindowStateStorage,
} from "./storage";

// ===== Display Provider Exports =====

export type {
    IScreen,
    DisplayProviderDeps,
} from "./display";

export {
    ElectronDisplayProvider,
    createDisplayProvider,
} from "./display";

// ===== Validator Exports =====

export {
    validateWindowState,
    checkWindowVisibility,
    calculateVisibleArea,
    centerOnDisplay,
    applyMinimumSize,
    clampToDisplay,
    findWindowDisplay,
} from "./validator";

// ===== Manager Exports =====

export type {
    WindowStateManagerDeps,
} from "./manager";

export {
    WindowStateManager,
    createWindowStateManager,
} from "./manager";
