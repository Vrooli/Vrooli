/**
 * Window State Validator
 *
 * DOC: docs/internal/SEAMS.md#window-state-validator
 *
 * Pure functions for validating and adjusting window state.
 * This module provides:
 * - Display visibility validation
 * - Multi-monitor boundary handling
 * - State adjustment for display changes
 *
 * All functions are pure with no side effects, making them
 * trivially testable without mocks.
 */

import type {
    DisplayBounds,
    ValidationResult,
    WindowState,
    WindowStateConfig,
} from "./types";
import { DEFAULT_WINDOW_STATE_CONFIG } from "./types";

/**
 * Minimum visible area required for a window to be considered "on screen".
 * If less than this many pixels are visible, the window will be repositioned.
 */
const MIN_VISIBLE_PIXELS = 100;

/**
 * Validate window state against available displays.
 * Ensures the window will be visible when restored.
 *
 * @param state - The window state to validate
 * @param displays - Available displays
 * @param primaryDisplay - The primary display (fallback)
 * @param config - Window state configuration
 * @returns Validation result with potentially adjusted state
 */
export function validateWindowState(
    state: WindowState,
    displays: DisplayBounds[],
    primaryDisplay: DisplayBounds,
    config: WindowStateConfig = DEFAULT_WINDOW_STATE_CONFIG
): ValidationResult {
    // If no position saved, center on primary display
    if (state.x === undefined || state.y === undefined) {
        return {
            isValid: true,
            state: centerOnDisplay(state, primaryDisplay),
            adjustmentReason: "Centered on primary display (no saved position)",
        };
    }

    // Check if window is visible on any display
    const visibilityResult = checkWindowVisibility(state, displays);

    if (visibilityResult.isVisible) {
        // Apply minimum size constraints
        const constrainedState = applyMinimumSize(state, config);
        if (constrainedState.width !== state.width || constrainedState.height !== state.height) {
            return {
                isValid: true,
                state: constrainedState,
                adjustmentReason: "Applied minimum size constraints",
            };
        }
        return { isValid: true, state };
    }

    // Window is not visible - reposition to primary display
    const repositioned = centerOnDisplay(state, primaryDisplay);
    return {
        isValid: false,
        state: repositioned,
        adjustmentReason: visibilityResult.reason || "Window not visible on any display",
    };
}

/**
 * Check if a window is sufficiently visible on any display.
 *
 * @param state - The window state to check
 * @param displays - Available displays
 * @returns Whether the window is visible and why not if not
 */
export function checkWindowVisibility(
    state: WindowState,
    displays: DisplayBounds[]
): { isVisible: boolean; reason?: string } {
    if (state.x === undefined || state.y === undefined) {
        return { isVisible: false, reason: "No position defined" };
    }

    if (displays.length === 0) {
        return { isVisible: false, reason: "No displays available" };
    }

    // Calculate visible area on each display
    for (const display of displays) {
        const visibleArea = calculateVisibleArea(state, display);
        if (visibleArea >= MIN_VISIBLE_PIXELS) {
            return { isVisible: true };
        }
    }

    return {
        isVisible: false,
        reason: `Window at (${state.x}, ${state.y}) not visible on any of ${displays.length} display(s)`,
    };
}

/**
 * Calculate how many pixels of a window are visible on a display.
 *
 * @param state - The window state
 * @param display - The display bounds
 * @returns Number of visible pixels (area)
 */
export function calculateVisibleArea(
    state: WindowState,
    display: DisplayBounds
): number {
    if (state.x === undefined || state.y === undefined) {
        return 0;
    }

    // Calculate overlap rectangle
    const left = Math.max(state.x, display.x);
    const top = Math.max(state.y, display.y);
    const right = Math.min(state.x + state.width, display.x + display.width);
    const bottom = Math.min(state.y + state.height, display.y + display.height);

    // If no overlap, return 0
    if (left >= right || top >= bottom) {
        return 0;
    }

    return (right - left) * (bottom - top);
}

/**
 * Center a window state on a display.
 * Preserves the window's current size.
 *
 * @param state - The window state to center
 * @param display - The display to center on
 * @returns New state centered on the display
 */
export function centerOnDisplay(
    state: WindowState,
    display: DisplayBounds
): WindowState {
    return {
        ...state,
        x: Math.round(display.x + (display.width - state.width) / 2),
        y: Math.round(display.y + (display.height - state.height) / 2),
        displayId: display.id,
    };
}

/**
 * Apply minimum size constraints to a window state.
 *
 * @param state - The window state
 * @param config - Configuration with minimum sizes
 * @returns State with minimum sizes enforced
 */
export function applyMinimumSize(
    state: WindowState,
    config: WindowStateConfig
): WindowState {
    const minWidth = config.minWidth ?? DEFAULT_WINDOW_STATE_CONFIG.minWidth;
    const minHeight = config.minHeight ?? DEFAULT_WINDOW_STATE_CONFIG.minHeight;

    const newWidth = Math.max(state.width, minWidth);
    const newHeight = Math.max(state.height, minHeight);

    if (newWidth === state.width && newHeight === state.height) {
        return state;
    }

    return {
        ...state,
        width: newWidth,
        height: newHeight,
    };
}

/**
 * Clamp a window state to fit within a display.
 * Used when a window is larger than the display or positioned off-screen.
 *
 * @param state - The window state
 * @param display - The display to clamp to
 * @returns State adjusted to fit within display
 */
export function clampToDisplay(
    state: WindowState,
    display: DisplayBounds
): WindowState {
    let { x, y, width, height } = state;
    x = x ?? display.x;
    y = y ?? display.y;

    // Clamp size to display dimensions (with some padding)
    const maxWidth = display.width - 20;
    const maxHeight = display.height - 20;
    width = Math.min(width, maxWidth);
    height = Math.min(height, maxHeight);

    // Clamp position to keep window on screen
    x = Math.max(display.x, Math.min(x, display.x + display.width - width));
    y = Math.max(display.y, Math.min(y, display.y + display.height - height));

    return {
        ...state,
        x,
        y,
        width,
        height,
        displayId: display.id,
    };
}

/**
 * Find the display that a window should be associated with.
 * Uses the window's center point to determine the display.
 *
 * @param state - The window state
 * @param displays - Available displays
 * @param primaryDisplay - Fallback display
 * @returns The display the window belongs to
 */
export function findWindowDisplay(
    state: WindowState,
    displays: DisplayBounds[],
    primaryDisplay: DisplayBounds
): DisplayBounds {
    if (state.x === undefined || state.y === undefined) {
        return primaryDisplay;
    }

    // Use the window's center point
    const centerX = state.x + state.width / 2;
    const centerY = state.y + state.height / 2;

    // Find display containing the center point
    for (const display of displays) {
        if (
            centerX >= display.x &&
            centerX < display.x + display.width &&
            centerY >= display.y &&
            centerY < display.y + display.height
        ) {
            return display;
        }
    }

    // Fallback to primary
    return primaryDisplay;
}
