import { describe, it, expect } from "vitest";
import {
  ambientBackground,
  ansi256ToHex,
  cellBackgroundHex,
  dominantBackground,
  dominantWeightedBackground,
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

describe("dominantWeightedBackground", () => {
  it("picks the color with the largest total weight above the threshold", () => {
    expect(
      dominantWeightedBackground(
        [
          { hex: "#111111", weight: 3 },
          { hex: "#222222", weight: 1 },
        ],
        0.6,
      ),
    ).toBe("#111111");
  });

  it("lets a heavier color overcome a more numerous lighter one", () => {
    // Two edge cells (weight 2) of B vs one corner cell (weight 3) of A.
    // A holds 3/7 ≈ 0.43, B holds 4/7 ≈ 0.57 → with a 0.5 threshold B wins.
    const samples = [
      { hex: "#aaaaaa", weight: 3 },
      { hex: "#bbbbbb", weight: 2 },
      { hex: "#bbbbbb", weight: 2 },
    ];
    expect(dominantWeightedBackground(samples, 0.5)).toBe("#bbbbbb");
    // …but the same split is ambiguous at the stricter perimeter threshold.
    expect(dominantWeightedBackground(samples, 0.6)).toBeNull();
  });

  it("ignores null hexes and non-positive weights", () => {
    expect(
      dominantWeightedBackground(
        [
          { hex: null, weight: 10 },
          { hex: "#cccccc", weight: 0 },
          { hex: "#dddddd", weight: -5 },
          { hex: "#eeeeee", weight: 2 },
        ],
        0.6,
      ),
    ).toBe("#eeeeee");
  });

  it("returns null for an empty / all-null sample", () => {
    expect(dominantWeightedBackground([], 0.6)).toBeNull();
    expect(dominantWeightedBackground([{ hex: null, weight: 3 }], 0.6)).toBeNull();
  });
});

describe("ambientBackground", () => {
  /** Build a `rows × cols` grid filled with a single color. */
  function fillGrid(rows: number, cols: number, color: string | null): (string | null)[][] {
    return Array.from({ length: rows }, () => Array.from({ length: cols }, () => color));
  }

  /** Set a cell without a non-null assertion (lint forbids `!`). */
  function setCell(grid: (string | null)[][], r: number, c: number, color: string | null): void {
    const row = grid[r];
    if (row) row[c] = color;
  }

  const BASE = "#1e1e1e";
  const CONTENT = "#553377";

  it("ignores a large center-only content block when the perimeter is base", () => {
    // 20×40 terminal. A coding-agent user-message block fills the strict
    // interior (rows 3–15, cols 4–35) — 52% of all cells — but leaves every
    // perimeter (corner/edge) cell at the base color.
    const grid = fillGrid(20, 40, BASE);
    let contentCells = 0;
    for (let r = 3; r <= 15; r++) {
      for (let c = 4; c <= 35; c++) {
        setCell(grid, r, c, CONTENT);
        contentCells++;
      }
    }
    // Sanity: the content block really is a flat-histogram majority…
    expect(contentCells / (20 * 40)).toBeGreaterThan(0.5);
    expect(dominantBackground(grid.flat(), 0.5)).toBe(CONTENT);
    // …yet ambient sampling follows the perimeter base color.
    expect(ambientBackground(grid)).toBe(BASE);
  });

  it("follows a true full-screen TUI background that reaches the edges", () => {
    expect(ambientBackground(fillGrid(20, 40, "#0d1117"))).toBe("#0d1117");
  });

  it("excludes the bottom status row so a status band cannot hijack the color", () => {
    const grid = fillGrid(20, 40, BASE);
    for (let c = 0; c < 40; c++) setCell(grid, 19, c, "#0dbc79"); // green status line
    expect(ambientBackground(grid)).toBe(BASE);
  });

  it("returns null when the perimeter has no dominant color (quadrants)", () => {
    // Four distinct quadrants: no single color reaches either threshold.
    const grid = fillGrid(20, 40, "#111111");
    for (let r = 0; r < 20; r++) {
      for (let c = 0; c < 40; c++) {
        const top = r < 10;
        const left = c < 20;
        setCell(grid, r, c, top ? (left ? "#111111" : "#222222") : left ? "#333333" : "#444444");
      }
    }
    expect(ambientBackground(grid)).toBeNull();
  });

  it("does not flip on an exact 50/50 perimeter tie (threshold above 0.5)", () => {
    // Left half one color, right half another. The status-row exclusion takes
    // a full row from both halves equally, so the perimeter stays an exact
    // 50/50 tie. The >0.5 threshold makes the result a deterministic null.
    const grid = fillGrid(20, 40, "#101010");
    for (let r = 0; r < 20; r++) {
      for (let c = 20; c < 40; c++) setCell(grid, r, c, "#202020");
    }
    expect(ambientBackground(grid)).toBeNull();
  });

  it("retints when a content block grows to fill the whole usable screen", () => {
    // Interior + edges all the new color, only the excluded status row differs.
    const grid = fillGrid(20, 40, "#2b2b40");
    for (let c = 0; c < 40; c++) setCell(grid, 19, c, "#0dbc79");
    expect(ambientBackground(grid)).toBe("#2b2b40");
  });

  it("handles tiny terminals without throwing", () => {
    // 1 row + statusRows=1 would leave nothing → falls back to the whole grid.
    expect(ambientBackground([["#abcdef", "#abcdef", "#abcdef"]])).toBe("#abcdef");
    // 2 narrow rows, uniform.
    expect(ambientBackground([["#123456", "#123456"], ["#123456", "#123456"]])).toBe("#123456");
    // All-null sample → null.
    expect(ambientBackground([[null, null], [null, null]])).toBeNull();
    // Empty grid → null.
    expect(ambientBackground([])).toBeNull();
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
