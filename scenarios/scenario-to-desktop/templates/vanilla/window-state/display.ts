/**
 * Display Provider
 *
 * DOC: docs/internal/SEAMS.md#window-state-display-provider
 *
 * Abstracts Electron's screen API for display/monitor information.
 * This module provides:
 * - Display enumeration for multi-monitor setups
 * - Point-to-display mapping
 * - Primary display identification
 *
 * Testing Seams:
 * - IScreen interface allows mocking Electron's screen module
 * - Can simulate multi-monitor scenarios in tests
 */

import type { DisplayBounds, IDisplayProvider } from "./types";

/**
 * Interface for Electron's screen module.
 * Seam for testing without Electron.
 */
export interface IScreen {
    getAllDisplays(): Array<{
        id: number;
        bounds: { x: number; y: number; width: number; height: number };
    }>;
    getPrimaryDisplay(): {
        id: number;
        bounds: { x: number; y: number; width: number; height: number };
    };
    getDisplayNearestPoint(point: { x: number; y: number }): {
        id: number;
        bounds: { x: number; y: number; width: number; height: number };
    };
}

/**
 * Dependencies for ElectronDisplayProvider.
 */
export interface DisplayProviderDeps {
    screen: IScreen;
    /** Optional logger for debugging */
    log?: (message: string, ...args: unknown[]) => void;
}

/**
 * Display provider implementation using Electron's screen API.
 *
 * Responsibilities:
 * - Enumerate connected displays
 * - Map coordinates to displays
 * - Identify primary display
 *
 * NOT responsible for:
 * - Window state validation
 * - State persistence
 * - Window management
 */
export class ElectronDisplayProvider implements IDisplayProvider {
    private readonly deps: DisplayProviderDeps;
    private readonly log: (message: string, ...args: unknown[]) => void;

    constructor(deps: DisplayProviderDeps) {
        this.deps = deps;
        this.log = deps.log ?? (() => { /* silent by default */ });
    }

    /**
     * Get all connected displays.
     */
    getAllDisplays(): DisplayBounds[] {
        const displays = this.deps.screen.getAllDisplays();
        return displays.map((d) => this.toDisplayBounds(d));
    }

    /**
     * Get the primary display.
     */
    getPrimaryDisplay(): DisplayBounds {
        const primary = this.deps.screen.getPrimaryDisplay();
        return this.toDisplayBounds(primary);
    }

    /**
     * Get the display containing the given point.
     * Uses Electron's getDisplayNearestPoint which always returns a display.
     * We validate that the point is actually within the returned display's bounds.
     */
    getDisplayAtPoint(x: number, y: number): DisplayBounds | null {
        const display = this.deps.screen.getDisplayNearestPoint({ x, y });
        const bounds = this.toDisplayBounds(display);

        // Verify the point is actually within this display
        if (this.isPointInBounds(x, y, bounds)) {
            return bounds;
        }

        // Point is not within any display (edge case)
        this.log(`Point (${x}, ${y}) not within any display bounds`);
        return null;
    }

    /**
     * Convert Electron display to our DisplayBounds type.
     */
    private toDisplayBounds(display: {
        id: number;
        bounds: { x: number; y: number; width: number; height: number };
    }): DisplayBounds {
        return {
            id: display.id,
            x: display.bounds.x,
            y: display.bounds.y,
            width: display.bounds.width,
            height: display.bounds.height,
        };
    }

    /**
     * Check if a point is within display bounds.
     */
    private isPointInBounds(x: number, y: number, bounds: DisplayBounds): boolean {
        return (
            x >= bounds.x &&
            x < bounds.x + bounds.width &&
            y >= bounds.y &&
            y < bounds.y + bounds.height
        );
    }
}

/**
 * Factory function to create ElectronDisplayProvider with Electron dependencies.
 *
 * This is the production factory - tests should create ElectronDisplayProvider
 * directly with mock dependencies.
 */
export function createDisplayProvider(
    screen: Electron.Screen
): ElectronDisplayProvider {
    return new ElectronDisplayProvider({
        screen: {
            getAllDisplays: () => screen.getAllDisplays(),
            getPrimaryDisplay: () => screen.getPrimaryDisplay(),
            getDisplayNearestPoint: (point) => screen.getDisplayNearestPoint(point),
        },
    });
}
