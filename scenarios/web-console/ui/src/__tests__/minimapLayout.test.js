import { describe, it, expect } from "vitest";
import { buildMinimapRowMarkers, viewportFromScrollMetrics, scrollTopFromMinimapPointer, } from "../lib/minimapLayout";
describe("buildMinimapRowMarkers", () => {
    it("returns empty for 0 rows", () => {
        expect(buildMinimapRowMarkers(0)).toEqual([]);
    });
    it("returns single marker for 1 row", () => {
        const markers = buildMinimapRowMarkers(1);
        expect(markers).toEqual([{ rowIndex: 0, topPercent: 0, heightPercent: 100 }]);
    });
    it("returns correct percentages for N rows", () => {
        const markers = buildMinimapRowMarkers(4);
        expect(markers).toHaveLength(4);
        expect(markers[0]).toEqual({ rowIndex: 0, topPercent: 0, heightPercent: 25 });
        expect(markers[1]).toEqual({ rowIndex: 1, topPercent: 25, heightPercent: 25 });
        expect(markers[3]).toEqual({ rowIndex: 3, topPercent: 75, heightPercent: 25 });
    });
    it("returns empty for non-finite input", () => {
        expect(buildMinimapRowMarkers(NaN)).toEqual([]);
        expect(buildMinimapRowMarkers(Infinity)).toEqual([]);
        expect(buildMinimapRowMarkers(-1)).toEqual([]);
    });
});
describe("viewportFromScrollMetrics", () => {
    it("returns full viewport when no overflow", () => {
        const result = viewportFromScrollMetrics({
            scrollTop: 0,
            scrollHeight: 500,
            clientHeight: 500,
        });
        expect(result.topPercent).toBe(0);
        expect(result.heightPercent).toBe(100);
        expect(result.maxScrollable).toBe(0);
    });
    it("calculates correct position when half-scrolled", () => {
        const result = viewportFromScrollMetrics({
            scrollTop: 500,
            scrollHeight: 2000,
            clientHeight: 1000,
        });
        expect(result.maxScrollable).toBe(1000);
        expect(result.heightPercent).toBe(50);
        // topPercent: (500/1000) * (100-50) = 25
        expect(result.topPercent).toBe(25);
    });
    it("applies minViewportPercent floor", () => {
        const result = viewportFromScrollMetrics({
            scrollTop: 0,
            scrollHeight: 100000,
            clientHeight: 100,
            minViewportPercent: 10,
        });
        expect(result.heightPercent).toBe(10);
    });
    it("handles NaN inputs gracefully", () => {
        const result = viewportFromScrollMetrics({
            scrollTop: NaN,
            scrollHeight: NaN,
            clientHeight: NaN,
        });
        expect(Number.isFinite(result.topPercent)).toBe(true);
        expect(Number.isFinite(result.heightPercent)).toBe(true);
        expect(Number.isFinite(result.maxScrollable)).toBe(true);
    });
});
describe("scrollTopFromMinimapPointer", () => {
    it("returns 0 at top of rail", () => {
        expect(scrollTopFromMinimapPointer(0, 200, 2000, 1000)).toBe(0);
    });
    it("returns maxScrollable at bottom of rail", () => {
        expect(scrollTopFromMinimapPointer(200, 200, 2000, 1000)).toBe(1000);
    });
    it("returns proportional value in middle", () => {
        expect(scrollTopFromMinimapPointer(100, 200, 2000, 1000)).toBe(500);
    });
    it("returns 0 for zero railHeight", () => {
        expect(scrollTopFromMinimapPointer(50, 0, 2000, 1000)).toBe(0);
    });
    it("clamps pointer outside rail bounds", () => {
        expect(scrollTopFromMinimapPointer(-10, 200, 2000, 1000)).toBe(0);
        expect(scrollTopFromMinimapPointer(300, 200, 2000, 1000)).toBe(1000);
    });
});
