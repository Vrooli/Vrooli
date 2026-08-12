import { describe, it, expect } from "vitest";
import { clampScale, distance, fitTransform, formatScalePercent, midpoint, panBy, parseViewBox, resetTransform, zoomAroundPoint, MAX_FIT_SCALE, MAX_SCALE, MIN_SCALE, } from "../zoomTransform";
describe("clampScale", () => {
    it("clamps below the minimum", () => {
        expect(clampScale(0.0001)).toBe(MIN_SCALE);
    });
    it("clamps above the maximum", () => {
        expect(clampScale(1000)).toBe(MAX_SCALE);
    });
    it("returns the minimum for non-finite input", () => {
        expect(clampScale(Number.NaN)).toBe(MIN_SCALE);
    });
    it("respects custom bounds", () => {
        expect(clampScale(5, 1, 2)).toBe(2);
        expect(clampScale(0.5, 1, 2)).toBe(1);
    });
});
describe("zoomAroundPoint", () => {
    it("keeps the focal point fixed in surface space", () => {
        const start = { scale: 1, x: 0, y: 0 };
        const px = 200;
        const py = 150;
        const next = zoomAroundPoint(start, 2, px, py);
        // The content point under (px, py) before and after must map to the same
        // surface coordinate.
        const worldBeforeX = (px - start.x) / start.scale;
        const worldBeforeY = (py - start.y) / start.scale;
        const screenAfterX = next.x + worldBeforeX * next.scale;
        const screenAfterY = next.y + worldBeforeY * next.scale;
        expect(screenAfterX).toBeCloseTo(px, 5);
        expect(screenAfterY).toBeCloseTo(py, 5);
        expect(next.scale).toBe(2);
    });
    it("does not exceed scale bounds while preserving the focal point", () => {
        const start = { scale: MAX_SCALE, x: 10, y: 10 };
        const next = zoomAroundPoint(start, 4, 100, 100);
        expect(next.scale).toBe(MAX_SCALE);
    });
});
describe("panBy", () => {
    it("translates without changing scale", () => {
        const next = panBy({ scale: 1.5, x: 10, y: 20 }, 5, -7);
        expect(next).toEqual({ scale: 1.5, x: 15, y: 13 });
    });
});
describe("resetTransform", () => {
    it("returns the identity transform", () => {
        expect(resetTransform()).toEqual({ scale: 1, x: 0, y: 0 });
    });
});
describe("fitTransform", () => {
    it("centers and scales content to fit with padding", () => {
        const t = fitTransform({ width: 1000, height: 1000 }, { width: 500, height: 500 }, { padding: 0 });
        expect(t.scale).toBe(MAX_FIT_SCALE);
        // 500 * 1.5 = 750, centered in 1000 -> (1000 - 750) / 2 = 125
        expect(t.x).toBeCloseTo(125, 5);
        expect(t.y).toBeCloseTo(125, 5);
    });
    it("scales a large diagram down to fit", () => {
        const t = fitTransform({ width: 400, height: 400 }, { width: 2000, height: 1000 }, { padding: 0 });
        // limited by width: 400 / 2000 = 0.2
        expect(t.scale).toBeCloseTo(0.2, 5);
        expect(t.x).toBeCloseTo(0, 5);
        expect(t.y).toBeCloseTo(100, 5); // (400 - 1000*0.2)/2
    });
    it("falls back to identity for zero content dimensions", () => {
        expect(fitTransform({ width: 800, height: 600 }, { width: 0, height: 0 })).toEqual(resetTransform());
    });
    it("falls back to identity for non-finite viewport", () => {
        expect(fitTransform({ width: Number.NaN, height: 600 }, { width: 100, height: 100 })).toEqual(resetTransform());
    });
});
describe("parseViewBox", () => {
    it("parses a standard viewBox", () => {
        expect(parseViewBox("0 0 320 240")).toEqual({ width: 320, height: 240 });
    });
    it("parses comma-separated values", () => {
        expect(parseViewBox("0,0,10,20")).toEqual({ width: 10, height: 20 });
    });
    it("returns null for malformed or empty input", () => {
        expect(parseViewBox("")).toBeNull();
        expect(parseViewBox(null)).toBeNull();
        expect(parseViewBox("0 0 100")).toBeNull();
        expect(parseViewBox("0 0 0 0")).toBeNull();
    });
});
describe("geometry helpers", () => {
    it("computes distance", () => {
        expect(distance(0, 0, 3, 4)).toBe(5);
    });
    it("computes midpoint", () => {
        expect(midpoint(0, 0, 10, 20)).toEqual({ x: 5, y: 10 });
    });
    it("formats scale as a percentage", () => {
        expect(formatScalePercent(1)).toBe("100%");
        expect(formatScalePercent(0.755)).toBe("76%");
    });
});
