import { describe, it, expect } from "vitest";
import { TOOLBAR_KEYS, applyModifiers } from "../consts/toolbar-keys";
// [REQ:P0-007b] Terminal Key/Chord Mapping
describe("Toolbar key/chord escape sequences", () => {
    it("Ctrl+C via modifier produces correct escape sequence (0x03)", () => {
        const result = applyModifiers("c", { ctrl: true, alt: false, shift: false });
        expect(result.data).toBe("\x03");
        expect(result.consumed).toBe(true);
    });
    it("Ctrl+D via modifier produces correct escape sequence (0x04)", () => {
        const result = applyModifiers("d", { ctrl: true, alt: false, shift: false });
        expect(result.data).toBe("\x04");
        expect(result.consumed).toBe(true);
    });
    it("Ctrl+Z via modifier produces correct escape sequence (0x1a)", () => {
        const result = applyModifiers("z", { ctrl: true, alt: false, shift: false });
        expect(result.data).toBe("\x1a");
        expect(result.consumed).toBe(true);
    });
    it("arrow keys send correct ANSI escape sequences", () => {
        const up = TOOLBAR_KEYS.find((k) => k.label === "\u2191");
        const down = TOOLBAR_KEYS.find((k) => k.label === "\u2193");
        const left = TOOLBAR_KEYS.find((k) => k.label === "\u2190");
        const right = TOOLBAR_KEYS.find((k) => k.label === "\u2192");
        expect(up?.input).toBe("\x1b[A");
        expect(down?.input).toBe("\x1b[B");
        expect(left?.input).toBe("\x1b[D");
        expect(right?.input).toBe("\x1b[C");
    });
    it("Esc sends correct escape character (0x1b)", () => {
        const esc = TOOLBAR_KEYS.find((k) => k.label === "Esc");
        expect(esc).toBeDefined();
        expect(esc?.input).toBe("\x1b");
    });
    it("Tab sends correct tab character", () => {
        const tab = TOOLBAR_KEYS.find((k) => k.label === "Tab");
        expect(tab).toBeDefined();
        expect(tab?.input).toBe("\t");
    });
});
