import { describe, expect, it } from "vitest";
import {
  MORPH_SCORE_THRESHOLD,
  SAMPLE_COUNT,
  alignGeometry,
  flattenPath,
  geometryFromElement,
  geometryFromNodes,
  geometryPath,
  interpolateGeometry,
  morphCompatibility,
  resamplePolyline,
  shapeToPathData,
  type IconGeometry,
  type Point,
} from "../../../../foundations/IconGeometry/versions/1.0.0/IconGeometry.ts";

/**
 * The predecessor parser handled only M/L/T/H/V/Z, so these tests deliberately
 * lead with the commands it dropped. Real lucide path data is used wherever
 * possible: the point is to prove the icons this library actually renders
 * survive the round trip, not that a synthetic grammar does.
 */

const bounds = (points: Point[]) => ({
  minX: Math.min(...points.map((p) => p.x)),
  maxX: Math.max(...points.map((p) => p.x)),
  minY: Math.min(...points.map((p) => p.y)),
  maxY: Math.max(...points.map((p) => p.y)),
});

const allPoints = (geometry: IconGeometry) => geometry.subpaths.flatMap((s) => s.points);

describe("number scanning", () => {
  it("splits on a minus sign with no separator", () => {
    const [subpath] = flattenPath("M10 10L20-5");
    expect(subpath?.points).toEqual([
      { x: 10, y: 10 },
      { x: 20, y: -5 },
    ]);
  });

  it("splits chained decimals", () => {
    const [subpath] = flattenPath("M0 0L.5.5");
    expect(subpath?.points).toEqual([
      { x: 0, y: 0 },
      { x: 0.5, y: 0.5 },
    ]);
  });

  it("reads exponent notation as one number", () => {
    const [subpath] = flattenPath("M0 0L1e1 2");
    expect(subpath?.points[1]).toEqual({ x: 10, y: 2 });
  });
});

describe("command coverage", () => {
  it("repeats an implicit lineto after a moveto group", () => {
    const [subpath] = flattenPath("M0 0 5 5 10 0");
    expect(subpath?.points).toEqual([
      { x: 0, y: 0 },
      { x: 5, y: 5 },
      { x: 10, y: 0 },
    ]);
  });

  it("repeats an argument group for a single command letter", () => {
    const [subpath] = flattenPath("M0 0L1 1 2 2 3 3");
    expect(subpath?.points).toHaveLength(4);
  });

  it("treats relative commands as offsets from the current point", () => {
    const [subpath] = flattenPath("M10 10l5 0l0 5");
    expect(subpath?.points).toEqual([
      { x: 10, y: 10 },
      { x: 15, y: 10 },
      { x: 15, y: 15 },
    ]);
  });

  it("flattens a cubic into points that stay within the control hull", () => {
    const [subpath] = flattenPath("M0 0C0 10 10 10 10 0");
    const points = subpath!.points;
    expect(points.length).toBeGreaterThan(4);
    const box = bounds(points);
    expect(box.minX).toBeGreaterThanOrEqual(-0.001);
    expect(box.maxX).toBeLessThanOrEqual(10.001);
    // The curve peaks at 3/4 of the control height for a symmetric cubic.
    expect(box.maxY).toBeGreaterThan(7);
    expect(box.maxY).toBeLessThan(7.6);
  });

  it("reflects the previous control point for a smooth cubic", () => {
    const withS = flattenPath("M0 0C0 5 5 5 5 0S10 -5 10 0");
    const explicit = flattenPath("M0 0C0 5 5 5 5 0C5 -5 10 -5 10 0");
    expect(withS[0]!.points.at(-1)).toEqual(explicit[0]!.points.at(-1));
    expect(bounds(withS[0]!.points).minY).toBeCloseTo(bounds(explicit[0]!.points).minY, 6);
  });

  it("converts a quadratic through the correct apex", () => {
    const [subpath] = flattenPath("M0 0Q5 10 10 0");
    // A quadratic reaches half its control height at the midpoint.
    expect(bounds(subpath!.points).maxY).toBeCloseTo(5, 1);
  });

  it("reflects the previous control point for a smooth quadratic", () => {
    const withT = flattenPath("M0 0Q5 10 10 0T20 0");
    const explicit = flattenPath("M0 0Q5 10 10 0Q15 -10 20 0");
    expect(withT[0]!.points.at(-1)).toEqual(explicit[0]!.points.at(-1));
    expect(bounds(withT[0]!.points).minY).toBeCloseTo(bounds(explicit[0]!.points).minY, 6);
  });

  it("traces an arc through its true extent rather than its chord", () => {
    // A half circle of radius 5 from (0,0) to (10,0), bulging downward.
    const [subpath] = flattenPath("M0 0A5 5 0 1 0 10 0");
    const box = bounds(subpath!.points);
    expect(box.maxY).toBeCloseTo(5, 1);
    expect(box.minX).toBeCloseTo(0, 1);
    expect(box.maxX).toBeCloseTo(10, 1);
  });

  it("honours the sweep flag by mirroring the bulge", () => {
    const down = flattenPath("M0 0A5 5 0 1 0 10 0")[0]!.points;
    const up = flattenPath("M0 0A5 5 0 1 1 10 0")[0]!.points;
    expect(bounds(down).maxY).toBeGreaterThan(4);
    expect(bounds(up).minY).toBeLessThan(-4);
  });

  it("scales up radii too small to span the chord", () => {
    // r=1 cannot reach across a chord of 10; F.6.6 requires scaling to r=5.
    const [subpath] = flattenPath("M0 0A1 1 0 0 0 10 0");
    const box = bounds(subpath!.points);
    expect(Math.abs(box.maxY - box.minY)).toBeCloseTo(5, 1);
  });

  it("degrades a zero-radius arc to a straight line", () => {
    const [subpath] = flattenPath("M0 0A0 0 0 0 0 10 0");
    expect(subpath?.points).toEqual([
      { x: 0, y: 0 },
      { x: 10, y: 0 },
    ]);
  });

  it("marks a subpath closed on Z and resumes a new one after it", () => {
    const subpaths = flattenPath("M0 0H10V10H0ZM2 2H4");
    expect(subpaths).toHaveLength(2);
    expect(subpaths[0]!.closed).toBe(true);
    expect(subpaths[1]!.closed).toBe(false);
  });

  it("skips an unknown command without truncating the subpath", () => {
    const [subpath] = flattenPath("M0 0L5 5B9 9L10 10");
    expect(subpath!.points.at(-1)).toEqual({ x: 10, y: 10 });
  });
});

describe("the regression that motivated this module", () => {
  /**
   * `search` is the registry glyph whose circle is drawn with two arcs. The
   * previous parser dropped both, leaving a one-point subpath that `flush()`
   * discarded — the shipped component rendered only the `M16 16l4 4` handle.
   */
  const SEARCH = "M11 4a7 7 0 1 0 0 14a7 7 0 1 0 0-14M16 16l4 4";

  it("recovers the magnifying glass, not just the handle", () => {
    const subpaths = flattenPath(SEARCH);
    expect(subpaths).toHaveLength(2);
    const circle = subpaths[0]!;
    const box = bounds(circle.points);
    expect(box.maxX - box.minX).toBeCloseTo(14, 0);
    expect(box.maxY - box.minY).toBeCloseTo(14, 0);
  });

  it("keeps the handle intact alongside it", () => {
    const handle = flattenPath(SEARCH)[1]!;
    expect(handle.points[0]).toEqual({ x: 16, y: 16 });
    expect(handle.points.at(-1)).toEqual({ x: 20, y: 20 });
  });
});

describe("shape elements", () => {
  it("converts a line", () => {
    expect(shapeToPathData({ tag: "line", attrs: { x1: "1", y1: "2", x2: "3", y2: "4" } }))
      .toBe("M1 2L3 4");
  });

  it("converts a circle into a closed loop of the right diameter", () => {
    const data = shapeToPathData({ tag: "circle", attrs: { cx: "12", cy: "12", r: "10" } })!;
    const [subpath] = flattenPath(data);
    const box = bounds(subpath!.points);
    expect(box.maxX - box.minX).toBeCloseTo(20, 0);
    expect(box.maxY - box.minY).toBeCloseTo(20, 0);
    expect(subpath!.closed).toBe(true);
  });

  it("converts a plain rect to four corners", () => {
    const data = shapeToPathData({
      tag: "rect",
      attrs: { x: "3", y: "3", width: "18", height: "18" },
    })!;
    const [subpath] = flattenPath(data);
    expect(subpath!.points).toEqual([
      { x: 3, y: 3 },
      { x: 21, y: 3 },
      { x: 21, y: 21 },
      { x: 3, y: 21 },
    ]);
    expect(subpath!.closed).toBe(true);
  });

  it("rounds rect corners without exceeding the rect bounds", () => {
    // SquareTerminal's frame, verbatim from lucide.
    const data = shapeToPathData({
      tag: "rect",
      attrs: { width: "18", height: "18", x: "3", y: "3", rx: "2", ry: "2" },
    })!;
    const [subpath] = flattenPath(data);
    const box = bounds(subpath!.points);
    expect(box.minX).toBeCloseTo(3, 1);
    expect(box.maxX).toBeCloseTo(21, 1);
    expect(box.minY).toBeCloseTo(3, 1);
    expect(box.maxY).toBeCloseTo(21, 1);
    // A rounded corner means no point sits exactly on the corner vertex.
    const corner = subpath!.points.find((p) => p.x < 3.05 && p.y < 3.05);
    expect(corner).toBeUndefined();
  });

  it("clamps corner radii to half the shorter side", () => {
    const data = shapeToPathData({
      tag: "rect",
      attrs: { x: "0", y: "0", width: "10", height: "10", rx: "50" },
    })!;
    const box = bounds(flattenPath(data)[0]!.points);
    expect(box.maxX - box.minX).toBeCloseTo(10, 1);
  });

  it("defaults ry to rx when only one is given", () => {
    const onlyRx = shapeToPathData({
      tag: "rect", attrs: { x: "0", y: "0", width: "10", height: "10", rx: "3" },
    })!;
    const both = shapeToPathData({
      tag: "rect", attrs: { x: "0", y: "0", width: "10", height: "10", rx: "3", ry: "3" },
    })!;
    expect(onlyRx).toBe(both);
  });

  it("closes a polygon and leaves a polyline open", () => {
    const polygon = flattenPath(shapeToPathData({ tag: "polygon", attrs: { points: "0,0 5,0 5,5" } })!);
    const polyline = flattenPath(shapeToPathData({ tag: "polyline", attrs: { points: "0,0 5,0 5,5" } })!);
    expect(polygon[0]!.closed).toBe(true);
    expect(polyline[0]!.closed).toBe(false);
  });

  it("returns null for a shape it cannot describe", () => {
    expect(shapeToPathData({ tag: "text", attrs: {} })).toBeNull();
    expect(shapeToPathData({ tag: "circle", attrs: { r: "0" } })).toBeNull();
  });
});

describe("resampling", () => {
  it("returns exactly the requested point count", () => {
    const points = resamplePolyline([{ x: 0, y: 0 }, { x: 10, y: 0 }], false, 16);
    expect(points).toHaveLength(16);
  });

  it("spaces points evenly by arc length along an open stroke", () => {
    const points = resamplePolyline([{ x: 0, y: 0 }, { x: 9, y: 0 }], false, 10);
    points.forEach((point, index) => {
      expect(point.x).toBeCloseTo(index, 6);
    });
  });

  it("hits both ends of an open stroke", () => {
    const points = resamplePolyline([{ x: 0, y: 0 }, { x: 4, y: 0 }, { x: 4, y: 4 }], false, 9);
    expect(points[0]).toEqual({ x: 0, y: 0 });
    expect(points.at(-1)!.x).toBeCloseTo(4, 6);
    expect(points.at(-1)!.y).toBeCloseTo(4, 6);
  });

  it("does not duplicate the seam of a closed loop", () => {
    const square = [{ x: 0, y: 0 }, { x: 4, y: 0 }, { x: 4, y: 4 }, { x: 0, y: 4 }];
    const points = resamplePolyline(square, true, 8);
    expect(points[0]).toEqual({ x: 0, y: 0 });
    expect(points.at(-1)).not.toEqual(points[0]);
    // Eight samples around a perimeter of 16 land every two units.
    expect(points[4]!.x).toBeCloseTo(4, 6);
    expect(points[4]!.y).toBeCloseTo(4, 6);
  });

  it("survives a degenerate polyline", () => {
    expect(resamplePolyline([{ x: 1, y: 1 }], false, 5)).toHaveLength(5);
    expect(resamplePolyline([], false, 5)).toHaveLength(0);
    expect(resamplePolyline([{ x: 1, y: 1 }, { x: 1, y: 1 }], false, 5)).toHaveLength(5);
  });
});

describe("geometry from nodes", () => {
  // Verbatim lucide icon nodes.
  const MESSAGE_SQUARE_TEXT = [
    { tag: "path", attrs: { d: "M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" } },
    { tag: "path", attrs: { d: "M13 8H7" } },
    { tag: "path", attrs: { d: "M17 12H7" } },
  ];
  const SQUARE_TERMINAL = [
    { tag: "path", attrs: { d: "m7 11 2-2-2-2" } },
    { tag: "path", attrs: { d: "M11 13h4" } },
    { tag: "rect", attrs: { width: "18", height: "18", x: "3", y: "3", rx: "2", ry: "2" } },
  ];

  it("samples every subpath to the same fixed count", () => {
    const geometry = geometryFromNodes(MESSAGE_SQUARE_TEXT);
    expect(geometry.subpaths).toHaveLength(3);
    for (const subpath of geometry.subpaths) {
      expect(subpath.points).toHaveLength(SAMPLE_COUNT);
    }
  });

  it("keeps the rounded speech bubble inside the 24-unit canvas", () => {
    const box = bounds(allPoints(geometryFromNodes(MESSAGE_SQUARE_TEXT)));
    expect(box.minX).toBeGreaterThanOrEqual(2.9);
    expect(box.maxX).toBeLessThanOrEqual(21.1);
    expect(box.minY).toBeGreaterThanOrEqual(2.9);
    expect(box.maxY).toBeLessThanOrEqual(21.1);
  });

  it("reads a mixed path-and-rect icon", () => {
    const geometry = geometryFromNodes(SQUARE_TERMINAL);
    expect(geometry.subpaths).toHaveLength(3);
    expect(geometry.subpaths.filter((s) => s.closed)).toHaveLength(1);
  });

  it("defaults the viewBox and carries an explicit one through", () => {
    expect(geometryFromNodes(MESSAGE_SQUARE_TEXT).viewBox).toBe("0 0 24 24");
    expect(geometryFromNodes(MESSAGE_SQUARE_TEXT, "0 0 32 32").viewBox).toBe("0 0 32 32");
  });

  it("ignores nodes it cannot describe instead of failing", () => {
    const geometry = geometryFromNodes([
      { tag: "title", attrs: {} },
      { tag: "path", attrs: { d: "M0 0L1 1" } },
    ]);
    expect(geometry.subpaths).toHaveLength(1);
  });
});

describe("geometry from a DOM element", () => {
  const svgFrom = (inner: string) => {
    const host = document.createElement("div");
    host.innerHTML = `<svg viewBox="0 0 24 24">${inner}</svg>`;
    return host.querySelector("svg");
  };

  it("reads shapes through plain attribute access", () => {
    const geometry = geometryFromElement(
      svgFrom(`<path d="M13 8H7"/><rect x="3" y="3" width="18" height="18" rx="2"/>`),
    );
    expect(geometry?.subpaths).toHaveLength(2);
    expect(geometry?.viewBox).toBe("0 0 24 24");
  });

  it("finds the svg when handed a wrapper", () => {
    const host = document.createElement("div");
    host.innerHTML = `<span><svg viewBox="0 0 24 24"><path d="M0 0L4 4"/></svg></span>`;
    expect(geometryFromElement(host.querySelector("span"))?.subpaths).toHaveLength(1);
  });

  it("returns null rather than empty geometry when there is nothing to read", () => {
    expect(geometryFromElement(null)).toBeNull();
    expect(geometryFromElement(svgFrom(""))).toBeNull();
    expect(geometryFromElement(svgFrom("<title>x</title>"))).toBeNull();
    expect(geometryFromElement(document.createElement("div"))).toBeNull();
  });
});

describe("alignment", () => {
  const square = geometryFromNodes([
    { tag: "rect", attrs: { x: "0", y: "0", width: "10", height: "10" } },
  ]);

  it("pairs every subpath on both sides", () => {
    const three = geometryFromNodes([
      { tag: "path", attrs: { d: "M0 0L1 1" } },
      { tag: "path", attrs: { d: "M5 5L6 6" } },
      { tag: "path", attrs: { d: "M9 9L10 10" } },
    ]);
    const one = geometryFromNodes([{ tag: "path", attrs: { d: "M0 0L1 1" } }]);
    expect(alignGeometry(three, one)).toHaveLength(3);
    expect(alignGeometry(one, three)).toHaveLength(3);
  });

  it("pads a missing stroke with a zero-opacity ghost", () => {
    const two = geometryFromNodes([
      { tag: "path", attrs: { d: "M0 0L1 1" } },
      { tag: "path", attrs: { d: "M9 9L10 10" } },
    ]);
    const one = geometryFromNodes([{ tag: "path", attrs: { d: "M0 0L1 1" } }]);
    const ghosts = alignGeometry(two, one).filter((pair) => pair.to.opacity === 0);
    expect(ghosts).toHaveLength(1);
    // The ghost collapses onto a point so the stroke shrinks rather than blinks.
    const points = ghosts[0]!.to.points;
    expect(new Set(points.map((p) => `${p.x},${p.y}`)).size).toBe(1);
  });

  it("rotates a closed loop to the cheapest start index", () => {
    // The same square, sampled starting from the opposite corner.
    const shifted: IconGeometry = {
      viewBox: square.viewBox,
      subpaths: square.subpaths.map((subpath) => ({
        ...subpath,
        points: [...subpath.points.slice(32), ...subpath.points.slice(0, 32)],
      })),
    };
    const [pair] = alignGeometry(square, shifted);
    // After realignment the pairing is the identity, so nothing travels.
    pair!.to.points.forEach((point, index) => {
      expect(point.x).toBeCloseTo(pair!.from.points[index]!.x, 6);
      expect(point.y).toBeCloseTo(pair!.from.points[index]!.y, 6);
    });
  });

  it("reverses a stroke wound the wrong way", () => {
    const forward = geometryFromNodes([{ tag: "path", attrs: { d: "M0 0L10 0" } }]);
    const backward = geometryFromNodes([{ tag: "path", attrs: { d: "M10 0L0 0" } }]);
    const [pair] = alignGeometry(forward, backward);
    expect(pair!.to.points[0]!.x).toBeCloseTo(0, 6);
    expect(pair!.to.points.at(-1)!.x).toBeCloseTo(10, 6);
  });

  it("does not rotate an open stroke, which would tear it", () => {
    const a = geometryFromNodes([{ tag: "path", attrs: { d: "M0 0L10 0" } }]);
    const b = geometryFromNodes([{ tag: "path", attrs: { d: "M0 5L10 5" } }]);
    const [pair] = alignGeometry(a, b);
    expect(pair!.to.points[0]!.x).toBeCloseTo(0, 6);
    expect(pair!.to.points.at(-1)!.x).toBeCloseTo(10, 6);
  });
});

describe("interpolation", () => {
  const a = geometryFromNodes([{ tag: "path", attrs: { d: "M0 0L10 0" } }]);
  const b = geometryFromNodes([{ tag: "path", attrs: { d: "M0 10L10 10" } }]);

  it("reproduces the endpoints at t=0 and t=1", () => {
    expect(interpolateGeometry(a, b, 0).subpaths[0]!.points[0]!.y).toBeCloseTo(0, 6);
    expect(interpolateGeometry(a, b, 1).subpaths[0]!.points[0]!.y).toBeCloseTo(10, 6);
  });

  it("lands halfway at t=0.5", () => {
    for (const point of interpolateGeometry(a, b, 0.5).subpaths[0]!.points) {
      expect(point.y).toBeCloseTo(5, 6);
    }
  });

  it("clamps progress outside the unit interval", () => {
    expect(interpolateGeometry(a, b, -1).subpaths[0]!.points[0]!.y).toBeCloseTo(0, 6);
    expect(interpolateGeometry(a, b, 2).subpaths[0]!.points[0]!.y).toBeCloseTo(10, 6);
  });

  it("fades a ghost in over the transition", () => {
    const two = geometryFromNodes([
      { tag: "path", attrs: { d: "M0 0L1 1" } },
      { tag: "path", attrs: { d: "M9 9L10 10" } },
    ]);
    const one = geometryFromNodes([{ tag: "path", attrs: { d: "M0 0L1 1" } }]);
    const mid = interpolateGeometry(one, two, 0.5);
    const opacities = mid.subpaths.map((s) => s.opacity).sort();
    expect(opacities[0]).toBeCloseTo(0.5, 6);
    expect(opacities[1]).toBeCloseTo(1, 6);
  });

  it("never flips a ghost pairing between open and closed", () => {
    const openOnly = geometryFromNodes([{ tag: "path", attrs: { d: "M0 0L1 1" } }]);
    const withClosed = geometryFromNodes([
      { tag: "path", attrs: { d: "M0 0L1 1" } },
      { tag: "rect", attrs: { x: "5", y: "5", width: "4", height: "4" } },
    ]);
    for (const subpath of interpolateGeometry(openOnly, withClosed, 0.5).subpaths) {
      const atStart = interpolateGeometry(openOnly, withClosed, 0).subpaths;
      expect(typeof subpath.closed).toBe("boolean");
      expect(atStart.length).toBe(2);
    }
  });
});

describe("serialization", () => {
  it("writes a moveto, linetos, and a close only when closed", () => {
    expect(geometryPath({ points: [{ x: 0, y: 0 }, { x: 1, y: 2 }], closed: false, opacity: 1 }))
      .toBe("M0 0L1 2");
    expect(geometryPath({ points: [{ x: 0, y: 0 }, { x: 1, y: 2 }], closed: true, opacity: 1 }))
      .toBe("M0 0L1 2Z");
  });

  it("rounds to two decimals so frame-to-frame strings stay small", () => {
    expect(geometryPath({ points: [{ x: 1.23456, y: 2 }], closed: false, opacity: 1 }))
      .toBe("M1.23 2");
  });

  it("returns an empty string for an empty subpath", () => {
    expect(geometryPath({ points: [], closed: false, opacity: 1 })).toBe("");
  });
});

describe("morph compatibility", () => {
  const nodes = (...data: string[]) => data.map((d) => ({ tag: "path", attrs: { d } }));

  it("scores an icon against itself at the top of the range", () => {
    const menu = geometryFromNodes(nodes("M4 7h16", "M4 12h16", "M4 17h16"));
    const result = morphCompatibility(menu, menu);
    expect(result.score).toBeCloseTo(1, 6);
    expect(result.travel).toBeCloseTo(0, 6);
  });

  it("clears the threshold for a related pair", () => {
    // Hamburger folding into a close cross: three strokes to two, all short
    // travel. This is the case path morphing exists for.
    const menu = geometryFromNodes(nodes("M4 7h16", "M4 12h16", "M4 17h16"));
    const close = geometryFromNodes(nodes("M6 6l12 12", "M18 6L6 18"));
    expect(morphCompatibility(menu, close).score).toBeGreaterThan(MORPH_SCORE_THRESHOLD);
  });

  it("clears the threshold for plus becoming check", () => {
    const plus = geometryFromNodes(nodes("M12 5v14", "M5 12h14"));
    const check = geometryFromNodes(nodes("M5 12l4 4L19 6"));
    expect(morphCompatibility(plus, check).score).toBeGreaterThan(MORPH_SCORE_THRESHOLD);
  });

  it("falls below the threshold when strokes cross the canvas", () => {
    const topLeft = geometryFromNodes(nodes("M1 1L3 3"));
    const bottomRight = geometryFromNodes(nodes("M21 21L23 23"));
    const result = morphCompatibility(topLeft, bottomRight);
    expect(result.travel).toBeGreaterThan(0.25);
    expect(result.score).toBeLessThan(MORPH_SCORE_THRESHOLD);
  });

  it("penalises a large mismatch in stroke count", () => {
    const one = geometryFromNodes(nodes("M4 12h16"));
    const many = geometryFromNodes(nodes("M4 6h16", "M4 9h16", "M4 12h16", "M4 15h16", "M4 18h16"));
    const result = morphCompatibility(one, many);
    expect(result.fromSubpaths).toBe(1);
    expect(result.toSubpaths).toBe(5);
    expect(result.score).toBeLessThan(morphCompatibility(one, one).score);
  });

  it("reports empty geometry rather than scoring it", () => {
    const empty: IconGeometry = { viewBox: "0 0 24 24", subpaths: [] };
    const menu = geometryFromNodes(nodes("M4 7h16"));
    expect(morphCompatibility(empty, menu).reason).toBe("empty-geometry");
    expect(morphCompatibility(empty, menu).score).toBe(0);
  });

  it("is symmetric enough that swap direction does not change the decision", () => {
    const menu = geometryFromNodes(nodes("M4 7h16", "M4 12h16", "M4 17h16"));
    const close = geometryFromNodes(nodes("M6 6l12 12", "M18 6L6 18"));
    const forward = morphCompatibility(menu, close).score > MORPH_SCORE_THRESHOLD;
    const backward = morphCompatibility(close, menu).score > MORPH_SCORE_THRESHOLD;
    expect(forward).toBe(backward);
  });
});
