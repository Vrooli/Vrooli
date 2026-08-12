import { describe, it, expect } from "vitest";
import { KEY_COMBOS, CATEGORY_ORDER, filterCombos } from "../consts/key-combos";
describe("Key combo definitions", () => {
    it("all combos have unique IDs", () => {
        const ids = KEY_COMBOS.map((c) => c.id);
        expect(new Set(ids).size).toBe(ids.length);
    });
    it("all combos have non-empty label, keys, and sequence", () => {
        for (const combo of KEY_COMBOS) {
            expect(combo.label.length).toBeGreaterThan(0);
            expect(combo.keys.length).toBeGreaterThan(0);
            expect(combo.sequence.length).toBeGreaterThan(0);
        }
    });
    it("every combo's first step has no delay", () => {
        for (const combo of KEY_COMBOS) {
            const first = combo.sequence[0];
            expect(first?.delayMs ?? 0).toBe(0);
        }
    });
    it("all combos use valid categories", () => {
        const validCategories = new Set(CATEGORY_ORDER);
        for (const combo of KEY_COMBOS) {
            expect(validCategories.has(combo.category)).toBe(true);
        }
    });
});
describe("filterCombos", () => {
    it("returns full list for empty query", () => {
        expect(filterCombos(KEY_COMBOS, "")).toEqual(KEY_COMBOS);
        expect(filterCombos(KEY_COMBOS, "  ")).toEqual(KEY_COMBOS);
    });
    it("matches by label (case-insensitive)", () => {
        const result = filterCombos(KEY_COMBOS, "interrupt");
        expect(result.some((c) => c.id === "ctrl-c")).toBe(true);
    });
    it("matches by keys string", () => {
        const result = filterCombos(KEY_COMBOS, "Ctrl+D");
        expect(result.some((c) => c.id === "ctrl-d")).toBe(true);
    });
    it("matches by category", () => {
        const result = filterCombos(KEY_COMBOS, "history");
        expect(result.length).toBe(KEY_COMBOS.filter((c) => c.category === "History").length);
    });
    it("matches by searchTerms", () => {
        const result = filterCombos(KEY_COMBOS, "sigint");
        expect(result.some((c) => c.id === "ctrl-c")).toBe(true);
    });
    it("returns empty for no-match query", () => {
        expect(filterCombos(KEY_COMBOS, "zzzznonexistent")).toEqual([]);
    });
});
