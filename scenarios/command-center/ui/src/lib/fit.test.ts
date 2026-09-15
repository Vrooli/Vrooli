import { describe, expect, it } from "vitest";
import { pageRanges, pageRows, rowsOf, scrollStops, visibleRange } from "./fit";

const rows = (count: number, height = 50, gap = 0) => Array.from({ length: count }, (_, index) => ({ top: index * (height + gap), height }));

describe("strip paging", () => { // [REQ:CC-P1-017]
  it("groups tiles into rows by their top edge and keeps the tallest tile's height", () => {
    expect(rowsOf([{ top: 0, height: 40 }, { top: 1, height: 60 }, { top: 80, height: 30 }])).toEqual([{ top: 0, height: 60 }, { top: 80, height: 30 }]);
  });
  it("fits every row on one page when the strip has room", () => {
    expect(pageRows(rows(2, 100), 220, 10)).toEqual([2]);
  });
  it("pages rows that overrun the strip's budget, counting the gap between rows", () => {
    expect(pageRows(rows(3, 100), 210, 10)).toEqual([2, 1]);
    expect(pageRows(rows(3, 100), 209, 10)).toEqual([1, 1, 1]);
  });
  it("gives a row taller than the budget a page of its own instead of dropping it", () => {
    expect(pageRows([{ top: 0, height: 300 }, { top: 310, height: 50 }], 200, 10)).toEqual([1, 1]);
  });
  it("maps rows per page to tile ranges at a fixed column count, ending on the last tile", () => {
    expect(pageRanges([1, 1], 6, 13)).toEqual([[0, 6], [6, 13]]);
    expect(pageRanges([2], 6, 9)).toEqual([[0, 9]]);
    expect(pageRanges([], 6, 4)).toEqual([[0, 4]]);
  });
});

describe("auto-scroll stops", () => { // [REQ:CC-P1-017]
  it("does not move a list that fits", () => {
    expect(scrollStops(rows(3), 150, 200)).toEqual([0]);
  });
  it("steps one row at a time and ends with the last row in view", () => {
    expect(scrollStops(rows(10), 500, 200)).toEqual([0, 50, 100, 150, 200, 250, 300]);
  });
  it("ends exactly at the bottom when rows do not land on it", () => {
    expect(scrollStops(rows(5, 50, 10), 290, 200)).toEqual([0, 60, 90]);
  });
  it("steps a viewport at a time under reduced motion, never skipping a row", () => {
    const stops = scrollStops(rows(10), 500, 200, true);
    expect(stops).toEqual([0, 200, 300]);
    for (const row of rows(10)) expect(stops.some((stop) => row.top >= stop && row.top + row.height <= stop + 200)).toBe(true);
  });
  it("names the rows wholly in view", () => {
    expect(visibleRange(rows(10), 0, 200)).toEqual([1, 4]);
    expect(visibleRange(rows(10), 300, 200)).toEqual([7, 10]);
    expect(visibleRange(rows(2, 300), 0, 200)).toBeNull();
  });
});
