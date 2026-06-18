import { describe, expect, it } from "vitest";

import {
  applyAspect,
  clampRect,
  contentRect,
  displayPointToImage,
  fullImageRect,
  imageRectToDisplay,
  roundRect,
} from "./cropMath";

describe("contentRect", () => {
  it("letterboxes a wide image inside a square element (vertical bars)", () => {
    // 200×100 image inside 100×100 element → scale 0.5, content 100×50 centered.
    const rect = contentRect({ width: 200, height: 100 }, { width: 100, height: 100 });
    expect(rect.scale).toBe(0.5);
    expect(rect.width).toBe(100);
    expect(rect.height).toBe(50);
    expect(rect.x).toBe(0);
    expect(rect.y).toBe(25);
  });

  it("letterboxes a tall image inside a square element (horizontal bars)", () => {
    const rect = contentRect({ width: 100, height: 200 }, { width: 100, height: 100 });
    expect(rect.scale).toBe(0.5);
    expect(rect.width).toBe(50);
    expect(rect.height).toBe(100);
    expect(rect.x).toBe(25);
    expect(rect.y).toBe(0);
  });

  it("returns a zero-scale rect when a dimension is non-positive", () => {
    expect(contentRect({ width: 0, height: 100 }, { width: 100, height: 100 }).scale).toBe(0);
  });
});

describe("coordinate conversions round-trip", () => {
  const natural = { width: 200, height: 100 };
  const client = { width: 100, height: 100 };
  const content = contentRect(natural, client);

  it("maps an image rect to display and back", () => {
    const imageRect = { x: 50, y: 25, width: 100, height: 50 };
    const display = imageRectToDisplay(imageRect, content);
    // image (50,25) → display (0 + 50*0.5, 25 + 25*0.5) = (25, 37.5)
    expect(display).toEqual({ x: 25, y: 37.5, width: 50, height: 25 });
    const back = displayPointToImage({ x: display.x, y: display.y }, content);
    expect(back).toEqual({ x: 50, y: 25 });
  });

  it("returns origin when scale is zero", () => {
    const dead = contentRect({ width: 0, height: 0 }, client);
    expect(displayPointToImage({ x: 10, y: 10 }, dead)).toEqual({ x: 0, y: 0 });
  });
});

describe("clampRect", () => {
  const natural = { width: 200, height: 100 };

  it("keeps a rect inside the image bounds", () => {
    expect(clampRect({ x: 180, y: 90, width: 60, height: 40 }, natural)).toEqual({
      x: 140,
      y: 60,
      width: 60,
      height: 40,
    });
  });

  it("caps oversized rects and enforces a 1px minimum", () => {
    expect(clampRect({ x: -10, y: -10, width: 999, height: 999 }, natural)).toEqual({
      x: 0,
      y: 0,
      width: 200,
      height: 100,
    });
    expect(clampRect({ x: 0, y: 0, width: 0, height: 0 }, natural)).toEqual({
      x: 0,
      y: 0,
      width: 1,
      height: 1,
    });
  });
});

describe("applyAspect", () => {
  const natural = { width: 400, height: 400 };

  it("leaves the rect untouched for a free ratio", () => {
    const rect = { x: 10, y: 20, width: 100, height: 73 };
    expect(applyAspect(rect, 0, natural)).toEqual(clampRect(rect, natural));
  });

  it("derives height from width for a 1:1 ratio", () => {
    const rect = applyAspect({ x: 0, y: 0, width: 120, height: 999 }, 1, natural);
    expect(rect.width).toBe(rect.height);
    expect(rect.width).toBe(120);
  });

  it("shrinks to fit when the derived box would overflow", () => {
    // 16:9 box at width 400 → height 225, fits; near the bottom it must shrink.
    const rect = applyAspect({ x: 0, y: 350, width: 400, height: 999 }, 16 / 9, natural);
    expect(rect.y + rect.height).toBeLessThanOrEqual(400);
    expect(Math.abs(rect.width / rect.height - 16 / 9)).toBeLessThan(0.01);
  });
});

describe("roundRect / fullImageRect", () => {
  it("rounds to integers", () => {
    expect(roundRect({ x: 1.4, y: 2.6, width: 3.5, height: 4.1 })).toEqual({
      x: 1,
      y: 3,
      width: 4,
      height: 4,
    });
  });

  it("spans the whole image", () => {
    expect(fullImageRect({ width: 640, height: 480 })).toEqual({
      x: 0,
      y: 0,
      width: 640,
      height: 480,
    });
  });
});
