import { describe, it, expect } from "vitest";
import { hexToRgb, relativeLuminance, isLightColor, isHexColor, parsePaneColor, serializePaneColor, paneColorStyle, nextGroupColor, paneAccentStyle, } from "../lib/paneColor";
import { HEADER_COLORS } from "../consts/config";
describe("parsePaneColor", () => {
    it("treats transparent / empty / nullish as transparent", () => {
        for (const value of ["transparent", "", null, undefined]) {
            expect(parsePaneColor(value)).toEqual({ colors: [], isTransparent: true });
        }
    });
    it("parses a legacy single hex color", () => {
        expect(parsePaneColor("#ff6b6b")).toEqual({ colors: ["#ff6b6b"], isTransparent: false });
    });
    it("parses a two-color stripe", () => {
        expect(parsePaneColor("#ff6b6b|#4dabf7")).toEqual({
            colors: ["#ff6b6b", "#4dabf7"],
            isTransparent: false,
        });
    });
    it("caps at two colors", () => {
        expect(parsePaneColor("#111111|#222222|#333333").colors).toEqual([
            "#111111",
            "#222222",
        ]);
    });
    it("drops malformed parts and keeps valid ones", () => {
        expect(parsePaneColor("notacolor|#4dabf7")).toEqual({
            colors: ["#4dabf7"],
            isTransparent: false,
        });
    });
    it("degrades fully-malformed input to transparent", () => {
        expect(parsePaneColor("garbage")).toEqual({ colors: [], isTransparent: true });
    });
});
describe("serializePaneColor", () => {
    it("round-trips a single color", () => {
        expect(serializePaneColor(["#ff6b6b"])).toBe("#ff6b6b");
    });
    it("round-trips two colors with the | delimiter", () => {
        expect(serializePaneColor(["#ff6b6b", "#4dabf7"])).toBe("#ff6b6b|#4dabf7");
    });
    it("returns transparent for an empty/invalid set", () => {
        expect(serializePaneColor([])).toBe("transparent");
        expect(serializePaneColor(["nope"])).toBe("transparent");
    });
    it("caps at two colors", () => {
        expect(serializePaneColor(["#111111", "#222222", "#333333"])).toBe("#111111|#222222");
    });
    it("is the inverse of parse for legacy single hex and transparent", () => {
        for (const value of ["#ff6b6b", "transparent"]) {
            const parsed = parsePaneColor(value);
            expect(serializePaneColor(parsed.colors)).toBe(value);
        }
    });
});
describe("paneColorStyle", () => {
    it("returns undefined for transparent/malformed", () => {
        expect(paneColorStyle("transparent", "bar")).toBeUndefined();
        expect(paneColorStyle("garbage", "header")).toBeUndefined();
    });
    it("returns a solid backgroundColor for a single color", () => {
        expect(paneColorStyle("#ff6b6b", "bar")).toEqual({ backgroundColor: "#ff6b6b" });
        expect(paneColorStyle("#ff6b6b", "header")).toEqual({ backgroundColor: "#ff6b6b" });
    });
    it("returns stacked bands for the bar variant with two colors", () => {
        const style = paneColorStyle("#ff6b6b|#4dabf7", "bar");
        expect(style?.backgroundImage).toContain("linear-gradient(180deg");
        expect(style?.backgroundImage).toContain("#ff6b6b 0 50%");
        expect(style?.backgroundImage).toContain("#4dabf7 50% 100%");
    });
    it("returns a candy-cane for the header variant with two colors", () => {
        const style = paneColorStyle("#ff6b6b|#4dabf7", "header");
        expect(style?.backgroundImage).toContain("repeating-linear-gradient(45deg");
        expect(style?.backgroundImage).toContain("#ff6b6b 0 10px");
        expect(style?.backgroundImage).toContain("#4dabf7 10px 20px");
    });
});
describe("nextGroupColor", () => {
    it("returns the first palette color when none are used", () => {
        expect(nextGroupColor([])).toBe(HEADER_COLORS[0]);
    });
    it("avoids colors already in use", () => {
        const used = [HEADER_COLORS[0], HEADER_COLORS[1]];
        expect(nextGroupColor(used)).toBe(HEADER_COLORS[2]);
    });
    it("wraps by count when every palette color is used", () => {
        const allUsed = [...HEADER_COLORS];
        // count === length → index 0
        expect(nextGroupColor(allUsed)).toBe(HEADER_COLORS[0]);
        // one extra beyond the full palette → index 1
        expect(nextGroupColor([...allUsed, "#000000"])).toBe(HEADER_COLORS[1]);
    });
});
describe("isHexColor", () => {
    it("accepts 3- and 6-digit hex", () => {
        expect(isHexColor("#fff")).toBe(true);
        expect(isHexColor("#ff6b6b")).toBe(true);
    });
    it("rejects non-hex", () => {
        expect(isHexColor("transparent")).toBe(false);
        expect(isHexColor("ff6b6b")).toBe(false);
        expect(isHexColor("#ff6b6b|#4dabf7")).toBe(false);
    });
});
describe("hexToRgb", () => {
    it("parses 6-digit hex", () => {
        expect(hexToRgb("#ff8800")).toEqual({ r: 255, g: 136, b: 0 });
    });
    it("expands 3-digit hex", () => {
        expect(hexToRgb("#abc")).toEqual({ r: 170, g: 187, b: 204 });
    });
    it("returns null for invalid input", () => {
        expect(hexToRgb("transparent")).toBeNull();
        expect(hexToRgb(null)).toBeNull();
    });
});
describe("relativeLuminance / isLightColor", () => {
    it("black is dark, white is light", () => {
        expect(relativeLuminance("#000000")).toBeCloseTo(0, 5);
        expect(relativeLuminance("#ffffff")).toBeCloseTo(1, 5);
        expect(isLightColor("#000000")).toBe(false);
        expect(isLightColor("#ffffff")).toBe(true);
    });
    it("classifies dark terminal backgrounds as dark", () => {
        // All built-in theme backgrounds want a light foreground.
        for (const bg of ["#0f172a", "#282a36", "#002b36", "#272822", "#2e3440", "#0d1117"]) {
            expect(isLightColor(bg)).toBe(false);
        }
    });
    it("classifies a near-white TUI background as light", () => {
        expect(isLightColor("#f5f5f5")).toBe(true);
    });
    it("treats invalid input as dark (0 luminance)", () => {
        expect(relativeLuminance("nope")).toBe(0);
    });
});
describe("paneAccentStyle", () => {
    it("prefers the pane's own color over the group's", () => {
        expect(paneAccentStyle("#ff6b6b", "#4dabf7", "bar")).toEqual({ backgroundColor: "#ff6b6b" });
    });
    it("falls back to the group color when the pane is transparent", () => {
        expect(paneAccentStyle("transparent", "#4dabf7", "bar")).toEqual({ backgroundColor: "#4dabf7" });
    });
    it("renders nothing when neither the pane nor a group supplies a color", () => {
        expect(paneAccentStyle("transparent", null, "bar")).toBeUndefined();
        expect(paneAccentStyle("transparent", undefined, "header")).toBeUndefined();
    });
    it("honours the variant for the group fallback too", () => {
        // The sidebar bar and the grid pane header use different geometry; the
        // fallback must not silently render the bar treatment in a header.
        expect(paneAccentStyle("transparent", "#4dabf7|#ff6b6b", "header")).toEqual({
            backgroundImage: "repeating-linear-gradient(45deg, #4dabf7 0 10px, #ff6b6b 10px 20px)",
        });
    });
});
