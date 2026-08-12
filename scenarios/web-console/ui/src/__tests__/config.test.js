import { describe, it, expect } from "vitest";
import { TERMINAL_THEMES, DEFAULT_THEME_ID, TERMINAL_FONT_SIZE, TERMINAL_FONT_FAMILY, DEFAULT_COLS, DEFAULT_ROWS, PANE_MIN_WIDTH_PX, PANE_MIN_HEIGHT_PX, HEALTH_RETRY_COUNT, HEALTH_RETRY_DELAY_MS, } from "../consts/config";
// [REQ:P1-002a] Shortcut Profile Management - config exports
describe("UI configuration constants", () => {
    it("exports terminal appearance defaults", () => {
        const defaultTheme = TERMINAL_THEMES[DEFAULT_THEME_ID];
        expect(defaultTheme).toBeDefined();
        expect(defaultTheme?.colors.background).toBe("#0f172a");
        expect(defaultTheme?.colors.foreground).toBe("#e2e8f0");
        expect(defaultTheme?.colors.cursor).toBe("#38bdf8");
        expect(TERMINAL_FONT_SIZE).toBe(14);
        expect(TERMINAL_FONT_FAMILY).toContain("monospace");
    });
    it("exports terminal dimension defaults", () => {
        expect(DEFAULT_COLS).toBe(80);
        expect(DEFAULT_ROWS).toBe(24);
    });
    it("exports pane layout thresholds", () => {
        expect(PANE_MIN_WIDTH_PX).toBeGreaterThan(0);
        expect(PANE_MIN_HEIGHT_PX).toBeGreaterThan(0);
    });
    it("exports health retry settings", () => {
        expect(HEALTH_RETRY_COUNT).toBeGreaterThan(0);
        expect(HEALTH_RETRY_DELAY_MS).toBeGreaterThan(0);
    });
});
