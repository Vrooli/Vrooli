import { describe, it, expect } from "vitest";
import {
  ansi256ToHex,
  cellBackgroundHex,
  dominantBackground,
  parseOscColor,
  type BgCell,
} from "../lib/terminalBackground";

/** Build a synthetic xterm cell for a given color mode. */
function cell(mode: "default" | "palette" | "rgb", value = 0): BgCell {
  return {
    isBgDefault: () => mode === "default",
    isBgPalette: () => mode === "palette",
    isBgRGB: () => mode === "rgb",
    getBgColor: () => value,
  };
}

describe("ansi256ToHex", () => {
  it("resolves the 16 base ANSI colors", () => {
    expect(ansi256ToHex(0)).toBe("#000000");
    expect(ansi256ToHex(1)).toBe("#cd3131");
    expect(ansi256ToHex(15)).toBe("#e5e5e5");
  });

  it("resolves the 6x6x6 color cube", () => {
    expect(ansi256ToHex(16)).toBe("#000000"); // cube origin
    expect(ansi256ToHex(231)).toBe("#ffffff"); // cube max
    expect(ansi256ToHex(21)).toBe("#0000ff"); // pure blue corner
  });

  it("resolves the grayscale ramp", () => {
    expect(ansi256ToHex(232)).toBe("#080808");
    expect(ansi256ToHex(255)).toBe("#eeeeee");
  });

  it("returns null for out-of-range indices", () => {
    expect(ansi256ToHex(-1)).toBeNull();
    expect(ansi256ToHex(256)).toBeNull();
  });
});

describe("cellBackgroundHex", () => {
  it("DEFAULT-mode cells resolve to the provided default background", () => {
    expect(cellBackgroundHex(cell("default"), "#0f172a")).toBe("#0f172a");
    expect(cellBackgroundHex(cell("default"), null)).toBeNull();
  });

  it("PALETTE-mode cells resolve against the ANSI palette", () => {
    expect(cellBackgroundHex(cell("palette", 0), "#000000")).toBe("#000000");
    expect(cellBackgroundHex(cell("palette", 1), "#000000")).toBe("#cd3131");
    expect(cellBackgroundHex(cell("palette", 231), "#000000")).toBe("#ffffff");
  });

  it("RGB-mode cells use the packed truecolor value", () => {
    expect(cellBackgroundHex(cell("rgb", 0x1e1e1e), null)).toBe("#1e1e1e");
    expect(cellBackgroundHex(cell("rgb", 0xff8800), null)).toBe("#ff8800");
  });
});

describe("dominantBackground", () => {
  it("picks the dominant color above the threshold", () => {
    const sample = ["#1e1e1e", "#1e1e1e", "#1e1e1e", "#ffffff"];
    expect(dominantBackground(sample, 0.5)).toBe("#1e1e1e");
  });

  it("is case-insensitive when counting", () => {
    expect(dominantBackground(["#AABBCC", "#aabbcc", "#000000"], 0.5)).toBe("#aabbcc");
  });

  it("ignores null cells (DEFAULT with no fallback)", () => {
    expect(dominantBackground([null, null, "#222222", "#222222"], 0.5)).toBe("#222222");
  });

  it("returns null when no color is confidently dominant", () => {
    // Four distinct colors, each 25% — below the 0.5 threshold.
    expect(dominantBackground(["#111111", "#222222", "#333333", "#444444"], 0.5)).toBeNull();
  });

  it("returns null for an empty / all-null sample", () => {
    expect(dominantBackground([], 0.5)).toBeNull();
    expect(dominantBackground([null, null], 0.5)).toBeNull();
  });
});

describe("parseOscColor", () => {
  it("parses rgb: form with 2 hex digits per channel", () => {
    expect(parseOscColor("rgb:1e/1e/1e")).toBe("#1e1e1e");
  });

  it("parses rgb: form with 4 hex digits per channel (scales down)", () => {
    expect(parseOscColor("rgb:ffff/0000/0000")).toBe("#ff0000");
  });

  it("parses #rrggbb and #rgb forms", () => {
    expect(parseOscColor("#1e1e1e")).toBe("#1e1e1e");
    expect(parseOscColor("#abc")).toBe("#aabbcc");
  });

  it("parses #rrrrggggbbbb (16-bit per channel)", () => {
    expect(parseOscColor("#1e1e1e1e1e1e")).toBe("#1e1e1e");
  });

  it("returns null for queries and garbage", () => {
    expect(parseOscColor("?")).toBeNull();
    expect(parseOscColor("")).toBeNull();
    expect(parseOscColor("notacolor")).toBeNull();
  });
});
