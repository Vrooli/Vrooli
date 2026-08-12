import { describe, it, expect } from "vitest";
import { TOOLBAR_KEYS } from "../consts/toolbar-keys";
// [REQ:P0-007a] Floating Toolbar Component
describe("MobileToolbar component", () => {
    it("exports TOOLBAR_KEYS with essential terminal keys", () => {
        const labels = TOOLBAR_KEYS.map((k) => k.label);
        expect(labels).toContain("Esc");
        expect(labels).toContain("Tab");
        expect(labels).toContain("\u2191"); // Up arrow
        expect(labels).toContain("\u2193"); // Down arrow
        expect(labels).toContain("\u2190"); // Left arrow
        expect(labels).toContain("\u2192"); // Right arrow
    });
    it("component module exports default component", async () => {
        const mod = await import("../components/MobileToolbar");
        // forwardRef wraps the component as an object with $$typeof and render
        expect(mod.default).toBeTruthy();
        expect(typeof mod.default === "function" || typeof mod.default === "object").toBe(true);
    });
    it("all keys have valid input sequences", () => {
        for (const key of TOOLBAR_KEYS) {
            expect(key.input.length).toBeGreaterThan(0);
            expect(key.label.length).toBeGreaterThan(0);
        }
    });
});
