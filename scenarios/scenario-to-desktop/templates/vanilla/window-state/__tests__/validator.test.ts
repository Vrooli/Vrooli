/**
 * Window State Validator Tests
 *
 * DOC: docs/internal/SEAMS.md#window-state-validator-tests
 *
 * Tests for pure validation functions.
 * These tests require no mocks since the functions are pure.
 */

import {
    validateWindowState,
    checkWindowVisibility,
    calculateVisibleArea,
    centerOnDisplay,
    applyMinimumSize,
    clampToDisplay,
    findWindowDisplay,
} from "../validator";
import type { DisplayBounds, WindowState, WindowStateConfig } from "../types";

// ===== Simple Test Runner =====
// This allows tests to run without a specific test framework
// Must be defined before use due to ESM hoisting rules

type TestFn = () => void | Promise<void>;
const tests: Array<{ name: string; fn: TestFn; suite: string }> = [];
let currentSuite = "";

function describe(name: string, fn: () => void): void {
    currentSuite = name;
    fn();
    currentSuite = "";
}

function test(name: string, fn: TestFn): void {
    tests.push({ name, fn, suite: currentSuite });
}

// ===== Test Fixtures =====

const PRIMARY_DISPLAY: DisplayBounds = {
    id: 1,
    x: 0,
    y: 0,
    width: 1920,
    height: 1080,
};

const SECONDARY_DISPLAY: DisplayBounds = {
    id: 2,
    x: 1920,
    y: 0,
    width: 1920,
    height: 1080,
};

const DISPLAYS = [PRIMARY_DISPLAY, SECONDARY_DISPLAY];

const DEFAULT_CONFIG: WindowStateConfig = {
    defaultWidth: 1200,
    defaultHeight: 800,
    minWidth: 400,
    minHeight: 300,
};

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

function assertContains(actual: string | undefined, expected: string, message?: string): void {
    if (!actual || !actual.includes(expected)) {
        throw new Error(message ?? `Expected "${actual}" to contain "${expected}"`);
    }
}

// ===== calculateVisibleArea Tests =====

describe("calculateVisibleArea", () => {
    test("returns full area when window is entirely within display", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const area = calculateVisibleArea(state, PRIMARY_DISPLAY);
        assertEquals(area, 800 * 600);
    });

    test("returns partial area when window overlaps display edge", () => {
        const state: WindowState = {
            x: 1600, // 320 pixels past right edge of primary
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        // Only 320 pixels visible on primary (1920 - 1600)
        const area = calculateVisibleArea(state, PRIMARY_DISPLAY);
        assertEquals(area, 320 * 600);
    });

    test("returns 0 when window is entirely outside display", () => {
        const state: WindowState = {
            x: 2000, // Past primary display
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const area = calculateVisibleArea(state, PRIMARY_DISPLAY);
        assertEquals(area, 0);
    });

    test("returns 0 when state has no position", () => {
        const state: WindowState = {
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const area = calculateVisibleArea(state, PRIMARY_DISPLAY);
        assertEquals(area, 0);
    });

    test("handles negative display coordinates (left of primary)", () => {
        const leftDisplay: DisplayBounds = {
            id: 3,
            x: -1920,
            y: 0,
            width: 1920,
            height: 1080,
        };

        const state: WindowState = {
            x: -1000,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const area = calculateVisibleArea(state, leftDisplay);
        assertEquals(area, 800 * 600);
    });
});

// ===== checkWindowVisibility Tests =====

describe("checkWindowVisibility", () => {
    test("returns visible when window is on a display", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = checkWindowVisibility(state, DISPLAYS);
        assertTrue(result.isVisible);
    });

    test("returns visible when window spans multiple displays", () => {
        const state: WindowState = {
            x: 1600, // Spans primary and secondary
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = checkWindowVisibility(state, DISPLAYS);
        assertTrue(result.isVisible);
    });

    test("returns not visible when window is off all displays", () => {
        const state: WindowState = {
            x: 5000,
            y: 5000,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = checkWindowVisibility(state, DISPLAYS);
        assertFalse(result.isVisible);
        assertContains(result.reason, "not visible");
    });

    test("returns not visible when state has no position", () => {
        const state: WindowState = {
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = checkWindowVisibility(state, DISPLAYS);
        assertFalse(result.isVisible);
        assertEquals(result.reason, "No position defined");
    });

    test("returns not visible when no displays available", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = checkWindowVisibility(state, []);
        assertFalse(result.isVisible);
        assertEquals(result.reason, "No displays available");
    });
});

// ===== centerOnDisplay Tests =====

describe("centerOnDisplay", () => {
    test("centers window on display", () => {
        const state: WindowState = {
            x: 0,
            y: 0,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const centered = centerOnDisplay(state, PRIMARY_DISPLAY);

        assertEquals(centered.x, 560); // (1920 - 800) / 2
        assertEquals(centered.y, 240); // (1080 - 600) / 2
        assertEquals(centered.displayId, 1);
    });

    test("centers on secondary display", () => {
        const state: WindowState = {
            x: 0,
            y: 0,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const centered = centerOnDisplay(state, SECONDARY_DISPLAY);

        assertEquals(centered.x, 1920 + 560); // Secondary x + (1920 - 800) / 2
        assertEquals(centered.y, 240);
        assertEquals(centered.displayId, 2);
    });

    test("preserves other state properties", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: true,
            isFullScreen: false,
        };

        const centered = centerOnDisplay(state, PRIMARY_DISPLAY);

        assertEquals(centered.width, 800);
        assertEquals(centered.height, 600);
        assertTrue(centered.isMaximized);
        assertFalse(centered.isFullScreen);
    });
});

// ===== applyMinimumSize Tests =====

describe("applyMinimumSize", () => {
    test("returns same state when above minimum", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = applyMinimumSize(state, DEFAULT_CONFIG);

        assertTrue(result === state, "Should return same reference");
    });

    test("enforces minimum width", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 200, // Below minimum
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = applyMinimumSize(state, DEFAULT_CONFIG);

        assertEquals(result.width, 400);
        assertEquals(result.height, 600);
    });

    test("enforces minimum height", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 100, // Below minimum
            isMaximized: false,
            isFullScreen: false,
        };

        const result = applyMinimumSize(state, DEFAULT_CONFIG);

        assertEquals(result.width, 800);
        assertEquals(result.height, 300);
    });

    test("enforces both minimums", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 100,
            height: 100,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = applyMinimumSize(state, DEFAULT_CONFIG);

        assertEquals(result.width, 400);
        assertEquals(result.height, 300);
    });
});

// ===== clampToDisplay Tests =====

describe("clampToDisplay", () => {
    test("returns same position when window fits", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const clamped = clampToDisplay(state, PRIMARY_DISPLAY);

        assertEquals(clamped.x, 100);
        assertEquals(clamped.y, 100);
    });

    test("clamps window that extends past right edge", () => {
        const state: WindowState = {
            x: 1500, // Would extend to 2300
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const clamped = clampToDisplay(state, PRIMARY_DISPLAY);

        assertEquals(clamped.x, 1920 - 800); // Pushed left to fit
        assertEquals(clamped.y, 100);
    });

    test("clamps window that extends past bottom edge", () => {
        const state: WindowState = {
            x: 100,
            y: 800, // Would extend to 1400
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const clamped = clampToDisplay(state, PRIMARY_DISPLAY);

        assertEquals(clamped.x, 100);
        assertEquals(clamped.y, 1080 - 600); // Pushed up to fit
    });

    test("reduces size for window larger than display", () => {
        const state: WindowState = {
            x: 0,
            y: 0,
            width: 2500,
            height: 1500,
            isMaximized: false,
            isFullScreen: false,
        };

        const clamped = clampToDisplay(state, PRIMARY_DISPLAY);

        assertEquals(clamped.width, 1920 - 20); // Max with padding
        assertEquals(clamped.height, 1080 - 20);
    });

    test("sets displayId on result", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const clamped = clampToDisplay(state, SECONDARY_DISPLAY);

        assertEquals(clamped.displayId, 2);
    });
});

// ===== findWindowDisplay Tests =====

describe("findWindowDisplay", () => {
    test("finds primary display for window on primary", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const display = findWindowDisplay(state, DISPLAYS, PRIMARY_DISPLAY);

        assertEquals(display.id, 1);
    });

    test("finds secondary display for window on secondary", () => {
        const state: WindowState = {
            x: 2000,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const display = findWindowDisplay(state, DISPLAYS, PRIMARY_DISPLAY);

        assertEquals(display.id, 2);
    });

    test("uses center point to determine display", () => {
        // Window spans both displays, but center is on secondary
        const state: WindowState = {
            x: 1800, // Starts on primary
            y: 100,
            width: 800, // Ends at 2600, well into secondary
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        // Center is at 1800 + 400 = 2200, which is on secondary
        const display = findWindowDisplay(state, DISPLAYS, PRIMARY_DISPLAY);

        assertEquals(display.id, 2);
    });

    test("returns primary for window with no position", () => {
        const state: WindowState = {
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const display = findWindowDisplay(state, DISPLAYS, PRIMARY_DISPLAY);

        assertEquals(display.id, 1);
    });

    test("returns primary for window off all displays", () => {
        const state: WindowState = {
            x: 5000,
            y: 5000,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const display = findWindowDisplay(state, DISPLAYS, PRIMARY_DISPLAY);

        assertEquals(display.id, 1);
    });
});

// ===== validateWindowState Tests =====

describe("validateWindowState", () => {
    test("returns valid state when window is visible", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = validateWindowState(state, DISPLAYS, PRIMARY_DISPLAY, DEFAULT_CONFIG);

        assertTrue(result.isValid);
        assertEquals(result.state.x, 100);
        assertEquals(result.state.y, 100);
    });

    test("centers on primary when no position saved", () => {
        const state: WindowState = {
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = validateWindowState(state, DISPLAYS, PRIMARY_DISPLAY, DEFAULT_CONFIG);

        assertTrue(result.isValid);
        assertEquals(result.state.x, 560);
        assertEquals(result.state.y, 240);
        assertContains(result.adjustmentReason, "no saved position");
    });

    test("repositions window when off all displays", () => {
        const state: WindowState = {
            x: 5000,
            y: 5000,
            width: 800,
            height: 600,
            isMaximized: false,
            isFullScreen: false,
        };

        const result = validateWindowState(state, DISPLAYS, PRIMARY_DISPLAY, DEFAULT_CONFIG);

        assertFalse(result.isValid);
        assertEquals(result.state.x, 560); // Centered on primary
        assertEquals(result.state.y, 240);
        assertContains(result.adjustmentReason, "not visible");
    });

    test("applies minimum size constraints", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 200, // Below minimum
            height: 200, // Below minimum
            isMaximized: false,
            isFullScreen: false,
        };

        const result = validateWindowState(state, DISPLAYS, PRIMARY_DISPLAY, DEFAULT_CONFIG);

        assertTrue(result.isValid);
        assertEquals(result.state.width, 400);
        assertEquals(result.state.height, 300);
        assertContains(result.adjustmentReason, "minimum size");
    });

    test("preserves maximized/fullscreen state", () => {
        const state: WindowState = {
            x: 100,
            y: 100,
            width: 800,
            height: 600,
            isMaximized: true,
            isFullScreen: false,
        };

        const result = validateWindowState(state, DISPLAYS, PRIMARY_DISPLAY, DEFAULT_CONFIG);

        assertTrue(result.state.isMaximized);
        assertFalse(result.state.isFullScreen);
    });
});

// Run tests - always run when imported via tsx
(async () => {
        let passed = 0;
        let failed = 0;
        let currentSuiteName = "";

        for (const t of tests) {
            if (t.suite !== currentSuiteName) {
                currentSuiteName = t.suite;
                console.log(`\n${t.suite}`);
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
