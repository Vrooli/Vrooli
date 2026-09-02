import { describe, expect, it, vi } from "vitest";
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

describe("probeTier on a constrained device", () => {
  it("drops to the reduced tier when the machine reports few cores", () => {
    const original = Object.getOwnPropertyDescriptor(Navigator.prototype, "hardwareConcurrency");
    Object.defineProperty(navigator, "hardwareConcurrency", { configurable: true, value: 2 });
    try {
      expect(probeTier(null)).toBe("reduced");
    } finally {
      if (original) Object.defineProperty(Navigator.prototype, "hardwareConcurrency", original);
      else delete (navigator as unknown as Record<string, unknown>).hardwareConcurrency;
    }
  });
});

describe("probeTier when the probe itself fails", () => {
  it("never blocks first paint: a throwing canvas probe resolves to the reduced tier", () => {
    const spy = vi.spyOn(document, "createElement").mockImplementation(() => { throw new Error("no canvas here"); });
    try {
      expect(probeTier(null)).toBe("reduced");
    } finally {
      spy.mockRestore();
    }
  });
});
