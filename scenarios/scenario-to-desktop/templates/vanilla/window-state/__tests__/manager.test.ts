/**
 * Window State Manager Tests
 *
 * DOC: docs/internal/SEAMS.md#window-state-manager-tests
 *
 * Tests for WindowStateManager orchestration logic.
 * Uses mock factories injected via the testing seams.
 */

import { describe, test, expect, vi } from "vitest";
import { WindowStateManager } from "../manager";
import type {
    IStateStorage,
    IDisplayProvider,
    IManagedWindow,
    WindowState,
    DisplayBounds,
} from "../types";

// ===== Mock Factories =====

interface MockStorage extends IStateStorage {
    loadCalls: number;
    saveCalls: number;
    lastSavedState: WindowState | null;
}

function createMockStorage(savedState: WindowState | null = null): MockStorage {
    return {
        loadCalls: 0,
        saveCalls: 0,
        lastSavedState: null,
        load: async function () {
            this.loadCalls++;
            return savedState;
        },
        save: async function (state: WindowState) {
            this.saveCalls++;
            this.lastSavedState = state;
        },
    };
}

function createMockDisplayProvider(
    displays: DisplayBounds[] = [{ id: 1, x: 0, y: 0, width: 1920, height: 1080 }],
    primaryDisplay?: DisplayBounds
): IDisplayProvider {
    const defaultDisplay: DisplayBounds = { id: 1, x: 0, y: 0, width: 1920, height: 1080 };
    const primary = primaryDisplay ?? displays[0] ?? defaultDisplay;
    return {
        getAllDisplays: () => displays,
        getPrimaryDisplay: () => primary,
        getDisplayAtPoint: (x: number, y: number) => {
            for (const d of displays) {
                if (x >= d.x && x < d.x + d.width && y >= d.y && y < d.y + d.height) {
                    return d;
                }
            }
            return null;
        },
    };
}

interface MockWindow extends IManagedWindow {
    bounds: { x: number; y: number; width: number; height: number };
    maximized: boolean;
    fullScreen: boolean;
    destroyed: boolean;
    eventHandlers: Record<string, Array<() => void>>;
    triggerEvent: (event: string) => void;
}

function createMockWindow(
    bounds = { x: 100, y: 100, width: 800, height: 600 },
    isMaximized = false,
    isFullScreen = false
): MockWindow {
    const eventHandlers: Record<string, Array<() => void>> = {};

    return {
        bounds,
        maximized: isMaximized,
        fullScreen: isFullScreen,
        destroyed: false,
        eventHandlers,
        getNormalBounds: function () {
            return this.bounds;
        },
        getBounds: function () {
            return this.bounds;
        },
        isMaximized: function () {
            return this.maximized;
        },
        isFullScreen: function () {
            return this.fullScreen;
        },
        isDestroyed: function () {
            return this.destroyed;
        },
        on: function (event: string, handler: () => void) {
            if (!this.eventHandlers[event]) {
                this.eventHandlers[event] = [];
            }
            this.eventHandlers[event].push(handler);
        },
        removeListener: function (event: string, handler: () => void) {
            if (this.eventHandlers[event]) {
                const index = this.eventHandlers[event].indexOf(handler);
                if (index >= 0) {
                    this.eventHandlers[event].splice(index, 1);
                }
            }
        },
        triggerEvent: function (event: string) {
            this.eventHandlers[event]?.forEach((h) => h());
        },
    };
}

// ===== Test Fixtures =====

const PRIMARY_DISPLAY: DisplayBounds = {
    id: 1,
    x: 0,
    y: 0,
    width: 1920,
    height: 1080,
};

const SAVED_STATE: WindowState = {
    x: 200,
    y: 150,
    width: 1000,
    height: 700,
    isMaximized: false,
    isFullScreen: false,
};

const MAXIMIZED_STATE: WindowState = {
    x: 200,
    y: 150,
    width: 1000,
    height: 700,
    isMaximized: true,
    isFullScreen: false,
};

// ===== Tests =====

describe("WindowStateManager", () => {
    describe("getInitialState", () => {
        test("returns saved state when valid", async () => {
            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });

            const state = await manager.getInitialState();

            expect(state.x).toBe(200);
            expect(state.y).toBe(150);
            expect(state.width).toBe(1000);
            expect(state.height).toBe(700);
        });

        test("returns defaults when no saved state", async () => {
            const storage = createMockStorage(null);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager(
                { storage, displayProvider },
                { defaultWidth: 1200, defaultHeight: 800 }
            );

            const state = await manager.getInitialState();

            expect(state.width).toBe(1200);
            expect(state.height).toBe(800);
            // Should be centered
            expect(state.x).toBe((1920 - 1200) / 2);
            expect(state.y).toBe((1080 - 800) / 2);
        });

        test("adjusts state when window is off-screen", async () => {
            const offScreenState: WindowState = {
                x: 5000,
                y: 5000,
                width: 800,
                height: 600,
                isMaximized: false,
                isFullScreen: false,
            };
            const storage = createMockStorage(offScreenState);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });

            const state = await manager.getInitialState();

            // Should be repositioned to primary display
            expect(state.x).toBe((1920 - 800) / 2);
            expect(state.y).toBe((1080 - 600) / 2);
        });

        test("handles storage load error gracefully", async () => {
            const storage = createMockStorage(null);
            storage.load = async () => {
                throw new Error("Disk error");
            };

            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager(
                { storage, displayProvider },
                { defaultWidth: 1000, defaultHeight: 700 }
            );

            const state = await manager.getInitialState();

            expect(state.width).toBe(1000);
            expect(state.height).toBe(700);
        });
    });

    describe("manage", () => {
        test("registers close handler on window", async () => {
            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });
            const window = createMockWindow();

            await manager.getInitialState();
            manager.manage(window);

            expect(window.eventHandlers["close"]?.length).toBeGreaterThan(0);
        });

        test("saves state on window close", async () => {
            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });
            const window = createMockWindow({ x: 300, y: 200, width: 900, height: 650 });

            await manager.getInitialState();
            manager.manage(window);

            // Trigger close event
            window.triggerEvent("close");

            // Wait for async save
            await new Promise((resolve) => setTimeout(resolve, 10));

            expect(storage.saveCalls).toBe(1);
            expect(storage.lastSavedState).not.toBeNull();
            expect(storage.lastSavedState!.x).toBe(300);
            expect(storage.lastSavedState!.y).toBe(200);
            expect(storage.lastSavedState!.width).toBe(900);
            expect(storage.lastSavedState!.height).toBe(650);
        });

        test("captures maximized state correctly", async () => {
            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });
            const window = createMockWindow(
                { x: 100, y: 100, width: 800, height: 600 },
                true, // isMaximized
                false
            );

            await manager.getInitialState();
            manager.manage(window);

            window.triggerEvent("close");

            await new Promise((resolve) => setTimeout(resolve, 10));

            expect(storage.lastSavedState).not.toBeNull();
            expect(storage.lastSavedState!.isMaximized).toBe(true);
        });

        test("captures fullscreen state via events (not isFullScreen at close time)", async () => {
            // This test verifies the fix for the macOS timing issue where
            // the fullscreen exit animation completes before the close event fires,
            // causing isFullScreen() to return false at close time.
            //
            // The fix tracks fullscreen state via enter/leave-full-screen events.

            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });
            const window = createMockWindow(
                { x: 100, y: 100, width: 800, height: 600 },
                false,
                false // Initially not fullscreen
            );

            await manager.getInitialState();
            manager.manage(window);

            // User enters fullscreen
            window.fullScreen = true;
            window.triggerEvent("enter-full-screen");

            // Simulate macOS behavior: fullscreen exits before close event fires
            // The isFullScreen() method returns false, but we should still save as fullscreen
            window.fullScreen = false;

            // Trigger close event
            window.triggerEvent("close");

            await new Promise((resolve) => setTimeout(resolve, 10));

            expect(storage.lastSavedState).not.toBeNull();
            // Should be true because we track via events, not isFullScreen() at close time
            expect(storage.lastSavedState!.isFullScreen).toBe(true);
        });

        test("tracks fullscreen state changes correctly through multiple toggles", async () => {
            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });
            const window = createMockWindow();

            await manager.getInitialState();
            manager.manage(window);

            // Enter fullscreen
            window.triggerEvent("enter-full-screen");

            // Leave fullscreen
            window.triggerEvent("leave-full-screen");

            // Enter fullscreen again
            window.triggerEvent("enter-full-screen");

            // Close while tracked state is fullscreen
            window.triggerEvent("close");

            await new Promise((resolve) => setTimeout(resolve, 10));

            expect(storage.lastSavedState).not.toBeNull();
            expect(storage.lastSavedState!.isFullScreen).toBe(true);
        });

        test("preserves fullscreen state when leave-full-screen fires after close (macOS/Linux behavior)", async () => {
            // On macOS and some Linux environments, when closing a fullscreen window:
            // 1. close event fires
            // 2. fullscreen exit animation plays
            // 3. leave-full-screen fires
            //
            // The fullscreen state should be preserved for saving even though
            // leave-full-screen fires after close.

            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });
            const window = createMockWindow();

            await manager.getInitialState();
            manager.manage(window);

            // User enters fullscreen
            window.triggerEvent("enter-full-screen");

            // User closes window while in fullscreen
            window.triggerEvent("close");

            // Fullscreen exit happens AFTER close on macOS/Linux
            window.triggerEvent("leave-full-screen");

            await new Promise((resolve) => setTimeout(resolve, 10));

            expect(storage.lastSavedState).not.toBeNull();
            // Should be true because we preserve state when closing
            expect(storage.lastSavedState!.isFullScreen).toBe(true);
        });

        test("detaches from previous window when managing new one", async () => {
            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });
            const window1 = createMockWindow();
            const window2 = createMockWindow();

            await manager.getInitialState();
            manager.manage(window1);
            const window1HandlerCount = window1.eventHandlers["close"]?.length ?? 0;

            manager.manage(window2);

            // Window1 should have its handler removed
            expect(window1.eventHandlers["close"]?.length ?? 0).toBeLessThan(window1HandlerCount);
            expect(window2.eventHandlers["close"]?.length).toBeGreaterThan(0);
        });

        test("does not save when window is destroyed", async () => {
            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });
            const window = createMockWindow();
            window.destroyed = true;

            await manager.getInitialState();
            manager.manage(window);

            window.triggerEvent("close");

            // Give it time to potentially save
            await new Promise((resolve) => setTimeout(resolve, 10));

            expect(storage.saveCalls).toBe(0);
        });
    });

    describe("wasMaximized", () => {
        test("returns true when saved state was maximized", async () => {
            const storage = createMockStorage(MAXIMIZED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });

            await manager.getInitialState();

            expect(manager.wasMaximized()).toBe(true);
        });

        test("returns false when saved state was not maximized", async () => {
            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });

            await manager.getInitialState();

            expect(manager.wasMaximized()).toBe(false);
        });

        test("returns false when restoreMaximized is disabled", async () => {
            const storage = createMockStorage(MAXIMIZED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager(
                { storage, displayProvider },
                { defaultWidth: 800, defaultHeight: 600, restoreMaximized: false }
            );

            await manager.getInitialState();

            expect(manager.wasMaximized()).toBe(false);
        });

        test("returns false when no saved state", async () => {
            const storage = createMockStorage(null);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });

            await manager.getInitialState();

            expect(manager.wasMaximized()).toBe(false);
        });
    });

    describe("wasFullScreen", () => {
        test("returns true when saved state was full-screen", async () => {
            const fullScreenState: WindowState = {
                ...SAVED_STATE,
                isFullScreen: true,
            };
            const storage = createMockStorage(fullScreenState);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });

            await manager.getInitialState();

            expect(manager.wasFullScreen()).toBe(true);
        });

        test("returns false when restoreFullScreen is disabled", async () => {
            const fullScreenState: WindowState = {
                ...SAVED_STATE,
                isFullScreen: true,
            };
            const storage = createMockStorage(fullScreenState);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager(
                { storage, displayProvider },
                { defaultWidth: 800, defaultHeight: 600, restoreFullScreen: false }
            );

            await manager.getInitialState();

            expect(manager.wasFullScreen()).toBe(false);
        });
    });

    describe("saveState", () => {
        test("saves current state immediately", async () => {
            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });
            const window = createMockWindow({ x: 400, y: 300, width: 1100, height: 750 });

            await manager.getInitialState();
            manager.manage(window);

            // Force save without close
            await manager.saveState();

            expect(storage.saveCalls).toBe(1);
        });

        test("handles save error gracefully", async () => {
            const storage = createMockStorage(SAVED_STATE);
            storage.save = async () => {
                throw new Error("Disk full");
            };

            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });
            const window = createMockWindow();

            await manager.getInitialState();
            manager.manage(window);

            // Should not throw
            await expect(manager.saveState()).resolves.toBeUndefined();
        });
    });

    describe("multi-monitor scenarios", () => {
        const SECONDARY_DISPLAY: DisplayBounds = {
            id: 2,
            x: 1920,
            y: 0,
            width: 1920,
            height: 1080,
        };

        test("restores window on secondary display", async () => {
            const secondaryState: WindowState = {
                x: 2100,
                y: 200,
                width: 800,
                height: 600,
                isMaximized: false,
                isFullScreen: false,
            };
            const storage = createMockStorage(secondaryState);
            const displayProvider = createMockDisplayProvider([PRIMARY_DISPLAY, SECONDARY_DISPLAY]);
            const manager = new WindowStateManager({ storage, displayProvider });

            const state = await manager.getInitialState();

            expect(state.x).toBe(2100);
            expect(state.y).toBe(200);
        });

        test("moves window to primary when secondary disconnected", async () => {
            const secondaryState: WindowState = {
                x: 2100,
                y: 200,
                width: 800,
                height: 600,
                isMaximized: false,
                isFullScreen: false,
            };
            const storage = createMockStorage(secondaryState);
            // Only primary display available now
            const displayProvider = createMockDisplayProvider([PRIMARY_DISPLAY]);
            const manager = new WindowStateManager({ storage, displayProvider });

            const state = await manager.getInitialState();

            // Should be moved to primary and centered
            expect(state.x).toBe((1920 - 800) / 2);
            expect(state.y).toBe((1080 - 600) / 2);
        });
    });
});
