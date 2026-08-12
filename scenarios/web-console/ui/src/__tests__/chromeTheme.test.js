import { describe, it, expect, beforeEach } from "vitest";
import { CHROME_PALETTE_TOKEN_NAMES, deriveChromePalette } from "../lib/chromePalette";
import { chromeTheme } from "../lib/chromeTheme";
function chromeColorVar() {
    return document.documentElement.style.getPropertyValue("--wc-chrome-color");
}
function chromeFgVar() {
    return document.documentElement.style.getPropertyValue("--wc-chrome-fg");
}
function tokenVar(name) {
    return document.documentElement.style.getPropertyValue(name);
}
function metaContent() {
    return document.querySelector('meta[name="theme-color"]')?.getAttribute("content") ?? null;
}
describe("chromeTheme controller", () => {
    beforeEach(() => {
        chromeTheme._reset();
        document.documentElement.removeAttribute("style");
        document.head.innerHTML = '<meta name="theme-color" content="#0f172a" />';
    });
    it("does nothing visible until enabled (default chrome)", () => {
        chromeTheme.setConfig({ enabled: false, ownerSessionId: null, fallbackColor: null });
        expect(chromeColorVar()).toBe("");
        expect(metaContent()).toBe("#0f172a");
    });
    it("applies the owner's fallback color when no detection has arrived", () => {
        chromeTheme.setConfig({ enabled: true, ownerSessionId: "s1", fallbackColor: "#002b36" });
        const palette = deriveChromePalette("#002b36");
        expect(chromeColorVar()).toBe("#002b36");
        expect(tokenVar("--wc-surface-raised")).toBe(palette["--wc-surface-raised"]);
        expect(tokenVar("--wc-accent")).toBe(palette["--wc-accent"]);
        expect(document.documentElement.dataset.wcAdaptiveChrome).toBe("true");
        expect(metaContent()).toBe("#002b36");
    });
    it("prefers the detected color over the fallback for the owner pane", () => {
        chromeTheme.setConfig({ enabled: true, ownerSessionId: "s1", fallbackColor: "#002b36" });
        chromeTheme.setDetected("s1", "#1e1e1e");
        expect(chromeColorVar()).toBe("#1e1e1e");
        expect(metaContent()).toBe("#1e1e1e");
    });
    it("ignores detection from a non-owner pane", () => {
        chromeTheme.setConfig({ enabled: true, ownerSessionId: "s1", fallbackColor: "#002b36" });
        chromeTheme.setDetected("s2", "#ff0000");
        expect(chromeColorVar()).toBe("#002b36");
    });
    it("falls back to the theme color when detection clears", () => {
        chromeTheme.setConfig({ enabled: true, ownerSessionId: "s1", fallbackColor: "#002b36" });
        chromeTheme.setDetected("s1", "#1e1e1e");
        chromeTheme.setDetected("s1", null);
        expect(chromeColorVar()).toBe("#002b36");
    });
    it("removes the vars and resets the meta to default when disabled", () => {
        chromeTheme.setConfig({ enabled: true, ownerSessionId: "s1", fallbackColor: "#002b36" });
        chromeTheme.setConfig({ enabled: false, ownerSessionId: null, fallbackColor: null });
        expect(chromeColorVar()).toBe("");
        expect(chromeFgVar()).toBe("");
        for (const name of CHROME_PALETTE_TOKEN_NAMES) {
            expect(tokenVar(name)).toBe("");
        }
        expect(document.documentElement.dataset.wcAdaptiveChrome).toBeUndefined();
        expect(metaContent()).toBe("#0f172a");
    });
    it("uses the derived secondary text as chrome foreground for light and dark tints", () => {
        chromeTheme.setConfig({ enabled: true, ownerSessionId: "s1", fallbackColor: "#0f172a" });
        expect(chromeFgVar()).toBe(`rgb(${deriveChromePalette("#0f172a")["--wc-text-secondary"]})`);
        chromeTheme.setDetected("s1", "#f5f5f5");
        expect(chromeFgVar()).toBe(`rgb(${deriveChromePalette("#f5f5f5")["--wc-text-secondary"]})`);
    });
    it("change-guard: re-applying an identical detected color is a no-op", () => {
        chromeTheme.setConfig({ enabled: true, ownerSessionId: "s1", fallbackColor: "#002b36" });
        chromeTheme.setDetected("s1", "#1e1e1e");
        // Mutate the DOM var out-of-band; an identical re-apply must NOT rewrite it.
        document.documentElement.style.setProperty("--wc-chrome-color", "#sentinel");
        chromeTheme.setDetected("s1", "#1e1e1e");
        expect(chromeColorVar()).toBe("#sentinel");
        expect(chromeTheme.getAppliedColor()).toBe("#1e1e1e");
    });
});
