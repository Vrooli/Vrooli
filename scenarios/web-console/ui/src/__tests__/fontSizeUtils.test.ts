import { describe, it, expect } from "vitest";
import { clampFontSize, FONT_SIZE_MIN, FONT_SIZE_MAX } from "../lib/fontSizeUtils";

describe("clampFontSize", () => {
  it("passes through values within range", () => {
    expect(clampFontSize(14)).toBe(14);
    expect(clampFontSize(8)).toBe(8);
    expect(clampFontSize(24)).toBe(24);
  });

  it("clamps below min to FONT_SIZE_MIN", () => {
    expect(clampFontSize(4)).toBe(FONT_SIZE_MIN);
    expect(clampFontSize(-1)).toBe(FONT_SIZE_MIN);
    expect(clampFontSize(0)).toBe(FONT_SIZE_MIN);
  });

  it("clamps above max to FONT_SIZE_MAX", () => {
    expect(clampFontSize(30)).toBe(FONT_SIZE_MAX);
    expect(clampFontSize(100)).toBe(FONT_SIZE_MAX);
  });

  it("returns 14 for NaN", () => {
    expect(clampFontSize(NaN)).toBe(14);
  });

  it("returns 14 for Infinity", () => {
    expect(clampFontSize(Infinity)).toBe(14);
    expect(clampFontSize(-Infinity)).toBe(14);
  });

  it("rounds fractional values", () => {
    expect(clampFontSize(14.4)).toBe(14);
    expect(clampFontSize(14.6)).toBe(15);
  });
});
