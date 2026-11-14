import { describe, expect, it } from "vitest";
import {
  CHROME_PALETTE_TOKEN_NAMES,
  deriveChromePalette,
  oklchToRgb,
  paletteTripleToHex,
  rgbToOklch,
  type Rgb,
} from "../lib/chromePalette";
import { contrastRatioRgb, hexToRgb } from "../lib/paneColor";

function triple(value: string): Rgb {
  const [r = 0, g = 0, b = 0] = value.split(/\s+/).map((part) => Number.parseInt(part, 10));
  return { r, g, b };
}

function token(palette: Record<string, string>, name: string): string {
  const value = palette[name];
  expect(value).toBeDefined();
  return value ?? "";
}

function expectContrast(fg: Rgb, bg: Rgb, target: number): void {
  expect(contrastRatioRgb(fg, bg)).toBeGreaterThanOrEqual(target);
}

describe("OKLCH conversion", () => {
  it("round-trips sRGB within channel tolerance", () => {
    for (const rgb of [
      { r: 15, g: 23, b: 42 },
      { r: 248, g: 250, b: 252 },
      { r: 0, g: 43, b: 54 },
      { r: 245, g: 245, b: 245 },
      { r: 80, g: 30, b: 120 },
    ]) {
      const roundTrip = oklchToRgb(rgbToOklch(rgb));
      expect(Math.abs(roundTrip.r - rgb.r)).toBeLessThanOrEqual(1);
      expect(Math.abs(roundTrip.g - rgb.g)).toBeLessThanOrEqual(1);
      expect(Math.abs(roundTrip.b - rgb.b)).toBeLessThanOrEqual(1);
    }
  });
});

describe("deriveChromePalette", () => {
  it("returns a stable full token shape in RGB triple format", () => {
    const palette = deriveChromePalette("#002b36");
    expect(Object.keys(palette).sort()).toEqual([...CHROME_PALETTE_TOKEN_NAMES].sort());
    for (const name of [
      "--wc-surface-base",
      "--wc-surface-raised",
      "--wc-surface-input",
      "--wc-text-primary",
      "--wc-text-secondary",
      "--wc-text-muted",
      "--wc-text-faint",
      "--wc-accent",
      "--wc-accent-fg",
    ]) {
      expect(palette[name]).toMatch(/^\d{1,3} \d{1,3} \d{1,3}$/);
    }
  });

  it("guarantees text and accent contrast for dark, light, mid, and saturated seeds", () => {
    for (const seed of ["#002b36", "#f5f5f5", "#6b7280", "#4c1d95"]) {
      const palette = deriveChromePalette(seed);
      const base = triple(token(palette, "--wc-surface-base"));
      const raised = triple(token(palette, "--wc-surface-raised"));
      const input = triple(token(palette, "--wc-surface-input"));
      expectContrast(triple(token(palette, "--wc-text-primary")), base, 4.5);
      expectContrast(triple(token(palette, "--wc-text-secondary")), base, 4.5);
      expectContrast(triple(token(palette, "--wc-text-muted")), raised, 3);
      expectContrast(triple(token(palette, "--wc-text-faint")), input, 3);
      expectContrast(triple(token(palette, "--wc-accent")), raised, 3);
      expectContrast(triple(token(palette, "--wc-accent-fg")), triple(token(palette, "--wc-accent")), 4.5);
    }
  });

  it("uses opposite surface polarity for light and dark seeds", () => {
    const dark = deriveChromePalette("#002b36");
    const light = deriveChromePalette("#f5f5f5");
    expect(contrastRatioRgb(triple(token(dark, "--wc-text-primary")), triple(token(dark, "--wc-surface-base")))).toBeGreaterThan(4.5);
    expect(contrastRatioRgb(triple(token(light, "--wc-text-primary")), triple(token(light, "--wc-surface-base")))).toBeGreaterThan(4.5);
    const darkPrimary = hexToRgb(paletteTripleToHex(token(dark, "--wc-text-primary")));
    const lightPrimary = hexToRgb(paletteTripleToHex(token(light, "--wc-text-primary")));
    expect(darkPrimary?.r).toBeGreaterThan(180);
    expect(lightPrimary?.r).toBeLessThan(100);
  });
});
