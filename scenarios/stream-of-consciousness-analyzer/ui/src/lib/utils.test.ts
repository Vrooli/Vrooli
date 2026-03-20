import { describe, it, expect } from "vitest";
import { cn, randomCanvasPosition } from "./utils";

// [REQ:P0-002] [REQ:P0-003] Utility functions

describe("cn", () => {
  it("merges simple class names", () => {
    expect(cn("px-2", "py-1")).toBe("px-2 py-1");
  });

  it("resolves conflicting Tailwind classes (last wins)", () => {
    expect(cn("px-2", "px-4")).toBe("px-4");
  });

  it("handles conditional classes via clsx", () => {
    expect(cn("base", false && "hidden", "text-sm")).toBe("base text-sm");
  });

  it("handles undefined and null inputs", () => {
    expect(cn("a", undefined, null, "b")).toBe("a b");
  });

  it("returns empty string for no inputs", () => {
    expect(cn()).toBe("");
  });
});

describe("randomCanvasPosition", () => {
  it("returns x within [0, width)", () => {
    for (let i = 0; i < 50; i++) {
      const { x } = randomCanvasPosition(600, 400);
      expect(x).toBeGreaterThanOrEqual(0);
      expect(x).toBeLessThan(600);
    }
  });

  it("returns y within [0, height)", () => {
    for (let i = 0; i < 50; i++) {
      const { y } = randomCanvasPosition(600, 400);
      expect(y).toBeGreaterThanOrEqual(0);
      expect(y).toBeLessThan(400);
    }
  });

  it("respects custom dimensions", () => {
    for (let i = 0; i < 50; i++) {
      const { x, y } = randomCanvasPosition(100, 50);
      expect(x).toBeLessThan(100);
      expect(y).toBeLessThan(50);
    }
  });

  it("returns an object with x and y properties", () => {
    const pos = randomCanvasPosition(500, 300);
    expect(pos).toHaveProperty("x");
    expect(pos).toHaveProperty("y");
    expect(typeof pos.x).toBe("number");
    expect(typeof pos.y).toBe("number");
  });
});
