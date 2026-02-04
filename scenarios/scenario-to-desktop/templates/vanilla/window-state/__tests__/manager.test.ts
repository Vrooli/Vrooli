/**
 * Window State Manager Tests
 *
 * DOC: docs/internal/SEAMS.md#window-state-manager-tests
 *
 * Tests for WindowStateManager orchestration logic.
 * Uses mock factories injected via the testing seams.
 */

import { WindowStateManager } from "../manager";
import type {
    IStateStorage,
    IDisplayProvider,
    IManagedWindow,
    WindowState,
    DisplayBounds,
} from "../types";

// ===== Simple Test Runner =====
// This allows tests to run without a specific test framework
// Must be defined before use due to ESM hoisting rules

type TestFn = () => void | Promise<void>;
const tests: Array<{ name: string; fn: TestFn; suite: string; parent: string }> = [];
let currentSuite = "";
let currentParent = "";

function describe(name: string, fn: () => void): void {
    const prevParent = currentParent;
    const prevSuite = currentSuite;
    if (currentParent) {
        currentParent = `${currentParent} > ${name}`;
    } else {
        currentParent = name;
    }
    currentSuite = name;
    fn();
    currentSuite = prevSuite;
    currentParent = prevParent;
}

function test(name: string, fn: TestFn): void {
    tests.push({ name, fn, suite: currentSuite, parent: currentParent });
}

// ===== Test Runner Agnostic Assertions =====

function assertEquals<T>(actual: T, expected: T, message?: string): void {
    if (actual !== expected) {
        throw new Error(message ?? `Expected ${expected} but got ${actual}`);
    }
}

function assertTrue(actual: boolean, message?: string): void {
    if (!actual) {
        throw new Error(message ?? `Expected true but got ${actual}`);
    }
}

function assertFalse(actual: boolean, message?: string): void {
    if (actual) {
        throw new Error(message ?? `Expected false but got ${actual}`);
    }
}

function assertNotNull<T>(actual: T | null | undefined, message?: string): asserts actual is T {
    if (actual === null || actual === undefined) {
        throw new Error(message ?? `Expected non-null value but got ${actual}`);
    }
}

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
    const primary = primaryDisplay ?? displays[0];
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

            assertEquals(state.x, 200);
            assertEquals(state.y, 150);
            assertEquals(state.width, 1000);
            assertEquals(state.height, 700);
        });

        test("returns defaults when no saved state", async () => {
            const storage = createMockStorage(null);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager(
                { storage, displayProvider },
                { defaultWidth: 1200, defaultHeight: 800 }
            );

            const state = await manager.getInitialState();

            assertEquals(state.width, 1200);
            assertEquals(state.height, 800);
            // Should be centered
            assertEquals(state.x, (1920 - 1200) / 2);
            assertEquals(state.y, (1080 - 800) / 2);
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
            assertEquals(state.x, (1920 - 800) / 2);
            assertEquals(state.y, (1080 - 600) / 2);
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

            assertEquals(state.width, 1000);
            assertEquals(state.height, 700);
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

            assertTrue(window.eventHandlers["close"]?.length > 0, "Should have close handler");
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

            assertEquals(storage.saveCalls, 1);
            assertNotNull(storage.lastSavedState);
            assertEquals(storage.lastSavedState.x, 300);
            assertEquals(storage.lastSavedState.y, 200);
            assertEquals(storage.lastSavedState.width, 900);
            assertEquals(storage.lastSavedState.height, 650);
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

            assertNotNull(storage.lastSavedState);
            assertTrue(storage.lastSavedState.isMaximized);
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
            assertTrue(
                (window1.eventHandlers["close"]?.length ?? 0) < window1HandlerCount,
                "Window1 should have handler removed"
            );
            assertTrue(window2.eventHandlers["close"]?.length > 0, "Window2 should have handler");
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

            assertEquals(storage.saveCalls, 0);
        });
    });

    describe("wasMaximized", () => {
        test("returns true when saved state was maximized", async () => {
            const storage = createMockStorage(MAXIMIZED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });

            await manager.getInitialState();

            assertTrue(manager.wasMaximized());
        });

        test("returns false when saved state was not maximized", async () => {
            const storage = createMockStorage(SAVED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });

            await manager.getInitialState();

            assertFalse(manager.wasMaximized());
        });

        test("returns false when restoreMaximized is disabled", async () => {
            const storage = createMockStorage(MAXIMIZED_STATE);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager(
                { storage, displayProvider },
                { defaultWidth: 800, defaultHeight: 600, restoreMaximized: false }
            );

            await manager.getInitialState();

            assertFalse(manager.wasMaximized());
        });

        test("returns false when no saved state", async () => {
            const storage = createMockStorage(null);
            const displayProvider = createMockDisplayProvider();
            const manager = new WindowStateManager({ storage, displayProvider });

            await manager.getInitialState();

            assertFalse(manager.wasMaximized());
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

            assertTrue(manager.wasFullScreen());
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

            assertFalse(manager.wasFullScreen());
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

            assertEquals(storage.saveCalls, 1);
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
            let threw = false;
            try {
                await manager.saveState();
            } catch {
                threw = true;
            }
            assertFalse(threw, "Should not throw on save error");
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

            assertEquals(state.x, 2100);
            assertEquals(state.y, 200);
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
            assertEquals(state.x, (1920 - 800) / 2);
            assertEquals(state.y, (1080 - 600) / 2);
        });
    });
});

// Run tests - always run when imported via tsx
(async () => {
    let passed = 0;
    let failed = 0;
    let currentParentName = "";

    for (const t of tests) {
        if (t.parent !== currentParentName) {
            currentParentName = t.parent;
            console.log(`\n${t.parent}`);
        }

        try {
            await t.fn();
            console.log(`  ✓ ${t.name}`);
            passed++;
        } catch (error) {
            console.log(`  ✗ ${t.name}`);
            console.log(`    ${error}`);
            failed++;
        }
    }

    console.log(`\n${passed} passed, ${failed} failed`);
    process.exit(failed > 0 ? 1 : 0);
})();
