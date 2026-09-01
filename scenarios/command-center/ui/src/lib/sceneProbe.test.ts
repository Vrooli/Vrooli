import { describe, expect, it } from "vitest";
import { frameIsBlank, probeTier } from "./sceneProbe";

const context = (alpha: number, size = 64): CanvasRenderingContext2D => ({
  getImageData: () => ({ data: new Uint8ClampedArray(size * size * 4).fill(alpha) }),
} as unknown as CanvasRenderingContext2D);

describe("frameIsBlank — a mounted scene that draws nothing is a failure", () => {
  it("flags a frame with no painted pixels", () => {
    expect(frameIsBlank(context(0), 64, 64)).toBe(true);
  });
  it("accepts a frame that painted", () => {
    expect(frameIsBlank(context(200), 64, 64)).toBe(false);
  });
  it("treats a degenerate canvas as blank", () => {
    expect(frameIsBlank(context(200), 1, 1)).toBe(true);
  });
});

describe("probeTier", () => {
  it("honours a forced tier from the URL", () => {
    expect(probeTier("still")).toBe("still");
    expect(probeTier("reduced")).toBe("reduced");
    expect(probeTier("full")).toBe("full");
  });
  it("never selects the full tier without WebGL", () => {
    expect(["still", "reduced"]).toContain(probeTier(null));
  });
});
