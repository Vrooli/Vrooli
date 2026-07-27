import { describe, expect, it } from "vitest";
import { chromeTier, fitDeviceGrid, fitGrid, MIN_LEGIBLE_FONT_PX, screenAperture, surplusRatio } from "./followerViewport";

describe("fitGrid", () => {
  it.each([
    [45, 30, 1600, 900, "full", 1],
    [200, 50, 390, 700, "hairline", 0],
    [80, 24, 1440, 900, "full", 1],
    [35, 50, 1000, 1000, "hairline", 1],
    [120, 40, 768, 1024, "full", 1],
    [24, 80, 900, 600, "hairline", 0],
    [100, 30, 1280, 720, "full", 1],
    [60, 40, 320, 568, "full", 1],
  ])("fits %ix%i inside %ix%i", (cols, rows, width, height, tier, expectedScale) => {
    const rect = fitGrid(cols, rows, width, height, 0.5);
    expect(rect.x).toBeGreaterThanOrEqual(0); expect(rect.y).toBeGreaterThanOrEqual(0);
    expect(rect.x + rect.width).toBeLessThanOrEqual(width); expect(rect.y + rect.height).toBeLessThanOrEqual(height);
    expect(rect.width / rect.height).toBeCloseTo((cols * 0.5) / rows, 4);
    expect(chromeTier(surplusRatio(rect, width, height), rect.scale)).toBe(tier);
    if (expectedScale === 1) expect(rect.scale).toBe(1); else expect(rect.scale).toBeLessThan(1);
    expect(rect.fontSize).toBeGreaterThanOrEqual(MIN_LEGIBLE_FONT_PX);
  });
});

describe("fitDeviceGrid", () => {
  it("keeps the terminal inside a monitor's bezel-safe screen aperture", () => {
    const aperture = screenAperture("monitor", "hairline");
    const { frame, screen } = fitDeviceGrid(236, 75, 390, 700, 0.5, aperture);
    expect(screen.x).toBeGreaterThan(frame.x);
    expect(screen.y).toBeGreaterThan(frame.y);
    expect(screen.x + screen.width).toBeLessThan(frame.x + frame.width);
    expect(screen.y + screen.height).toBeLessThan(frame.y + frame.height);
    expect(screen.width / screen.height).toBeCloseTo((236 * 0.5) / 75, 4);
  });
});
