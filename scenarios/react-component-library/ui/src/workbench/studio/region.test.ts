import { describe, expect, it } from "vitest";

import {
  MIN_REGION,
  applyAspect,
  clampRegion,
  moveRegion,
  regionFromDrag,
  regionToPixels,
  resizeRegion,
  type Region,
} from "./region";

const BOUNDS = { width: 100, height: 80 };

function region(x: number, y: number, w: number, h: number): Region {
  return { x, y, w, h };
}

describe("clampRegion", () => {
  it("clamps at the left and top edges", () => {
    expect(clampRegion(region(-20, -5, 40, 30), BOUNDS)).toEqual(region(0, 0, 40, 30));
  });

  it("clamps at the right and bottom edges", () => {
    expect(clampRegion(region(90, 70, 40, 30), BOUNDS)).toEqual(region(60, 50, 40, 30));
  });

  it("clamps a region that is outside on both axes", () => {
    expect(clampRegion(region(200, 200, 40, 30), BOUNDS)).toEqual(region(60, 50, 40, 30));
  });

  it("enforces the minimum size and nudges the region back inside", () => {
    expect(clampRegion(region(95, 75, 2, 2), BOUNDS)).toEqual(region(84, 64, 16, 16));
    expect(clampRegion(region(10, 10, 2, 3), BOUNDS)).toEqual(region(10, 10, 16, 16));
  });

  it("accepts an image smaller than the minimum", () => {
    expect(clampRegion(region(0, 0, 4, 4), { width: 10, height: 8 })).toEqual(
      region(0, 0, 10, 8),
    );
  });

  it("rounds fractional input to whole pixels", () => {
    expect(clampRegion(region(10.6, 20.2, 30.5, 40.4), BOUNDS)).toEqual(region(11, 20, 31, 40));
  });
});

describe("regionFromDrag", () => {
  it("normalises an inverted drag", () => {
    expect(regionFromDrag({ x: 60, y: 50 }, { x: 20, y: 10 }, BOUNDS)).toEqual(
      region(20, 10, 40, 40),
    );
  });

  it("clamps a drag that leaves the image", () => {
    expect(regionFromDrag({ x: 90, y: 70 }, { x: -10, y: -10 }, BOUNDS)).toEqual(
      region(0, 0, 100, 80),
    );
  });

  it("grows a tiny drag to the minimum size", () => {
    expect(regionFromDrag({ x: 50, y: 40 }, { x: 51, y: 41 }, BOUNDS)).toEqual(
      region(50, 40, MIN_REGION, MIN_REGION),
    );
  });
});

describe("moveRegion", () => {
  it("clamps movement at every edge", () => {
    const base = region(10, 10, 20, 20);
    expect(moveRegion(base, -100, -100, BOUNDS)).toEqual(region(0, 0, 20, 20));
    expect(moveRegion(base, 100, 100, BOUNDS)).toEqual(region(80, 60, 20, 20));
  });
});

describe("resizeRegion", () => {
  it("grows from the south-east corner without crossing the image edge", () => {
    const base = region(10, 10, 20, 20);
    expect(resizeRegion(base, "se", 10, 5, BOUNDS)).toEqual(region(10, 10, 30, 25));
    expect(resizeRegion(base, "se", 500, 500, BOUNDS)).toEqual(region(10, 10, 90, 70));
  });

  it("shrinks from the south-east corner down to the minimum", () => {
    expect(resizeRegion(region(10, 10, 40, 40), "se", -100, -100, BOUNDS)).toEqual(
      region(10, 10, 16, 16),
    );
  });

  it("moves the north-west corner and keeps the opposite edge fixed", () => {
    const base = region(10, 10, 20, 20);
    expect(resizeRegion(base, "nw", -5, -5, BOUNDS)).toEqual(region(5, 5, 25, 25));
    expect(resizeRegion(base, "nw", 100, 100, BOUNDS)).toEqual(region(14, 14, 16, 16));
  });

  it("moves the north-east and south-west corners independently", () => {
    const base = region(10, 10, 20, 20);
    expect(resizeRegion(base, "ne", 10, -10, BOUNDS)).toEqual(region(10, 0, 30, 30));
    expect(resizeRegion(base, "sw", -10, 10, BOUNDS)).toEqual(region(0, 10, 30, 30));
  });
});

describe("applyAspect", () => {
  it("returns a clamped region when the preset is Free", () => {
    expect(applyAspect(region(-10, 0, 40, 30), null, BOUNDS)).toEqual(region(0, 0, 40, 30));
  });

  it("makes a square centred on the current region", () => {
    expect(applyAspect(region(200, 150, 400, 300), 1, { width: 800, height: 600 })).toEqual(
      region(200, 100, 400, 400),
    );
  });

  it("fits a wide ratio without exceeding the bounds", () => {
    expect(applyAspect(region(200, 150, 400, 300), 16 / 9, { width: 800, height: 600 })).toEqual(
      region(200, 188, 400, 225),
    );
  });

  it("anchors at the top-left when asked", () => {
    expect(
      applyAspect(region(200, 150, 400, 300), 1, { width: 800, height: 600 }, "nw"),
    ).toEqual(region(200, 150, 400, 400));
  });

  it("shrinks the width until ratio-fitted height fits the image", () => {
    expect(applyAspect(region(0, 0, 800, 600), 1, { width: 800, height: 600 })).toEqual(
      region(100, 0, 600, 600),
    );
    expect(applyAspect(region(0, 0, 800, 600), 4, { width: 800, height: 600 })).toEqual(
      region(0, 200, 800, 200),
    );
  });
});

describe("regionToPixels", () => {
  it("is the identity when the preview is not scaled", () => {
    expect(
      regionToPixels(region(10, 20, 30, 40), { width: 100, height: 100 }, { width: 100, height: 100 }),
    ).toEqual(region(10, 20, 30, 40));
  });

  it("scales a half-size preview back to natural pixels", () => {
    expect(
      regionToPixels(region(10, 20, 30, 40), { width: 800, height: 600 }, { width: 400, height: 300 }),
    ).toEqual(region(20, 40, 60, 80));
  });

  it("handles non-uniform fractional scaling", () => {
    expect(
      regionToPixels(region(10, 10, 50, 25), { width: 1000, height: 500 }, { width: 250, height: 125 }),
    ).toEqual(region(40, 40, 200, 100));
  });

  it("keeps a converted edge region inside the natural image", () => {
    expect(
      regionToPixels(region(280, 110, 30, 20), { width: 1000, height: 500 }, { width: 300, height: 120 }),
    ).toEqual(region(900, 417, 100, 83));
  });

  it("degrades safely when the preview has no size yet", () => {
    expect(
      regionToPixels(region(10, 10, 20, 20), { width: 100, height: 100 }, { width: 0, height: 0 }),
    ).toEqual(region(0, 0, 1, 1));
  });
});
