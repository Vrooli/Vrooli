import { describe, expect, it } from "vitest";
import type { Reading } from "../lib/api";
import { nextMeasuredBeat, pickHero } from "./hero";
import { authoredSample, makeReading } from "../test-utils/readings";

const reading = (id: string, overrides: Partial<Reading>): Reading => makeReading({ id, label: id, ...overrides });

describe("pickHero", () => {
  it("prefers the first measured reading in registry order", () => {
    const hero = pickHero([reading("a", { coverage: "MISSING", trust: "UNAVAILABLE", sample: authoredSample(1) }), reading("b", { value: 7 }), reading("c", { value: 9 })]);
    expect(hero?.id).toBe("b");
  });
  it("falls back to the first illustrative reading so a room without measurements still composes", () => {
    const hero = pickHero([reading("a", { trust: "UNAVAILABLE" }), reading("b", { coverage: "IN-REACH", trust: "UNAVAILABLE", sample: authoredSample(3) })]);
    expect(hero?.id).toBe("b");
  });
  it("returns null for an empty room", () => {
    expect(pickHero([])).toBeNull();
  });
});

describe("nextMeasuredBeat", () => { // [REQ:CC-P1-008]
  const beats = [{ hero: "a" }, { hero: "b" }, { hero: "c" }];

  it("skips forward to the next beat whose hero is measured", () => {
    expect(nextMeasuredBeat(beats, new Set(["c"]), 0)).toBe(2);
  });

  it("wraps to an earlier measured hero", () => {
    expect(nextMeasuredBeat(beats, new Set(["a"]), 2)).toBe(0);
  });

  it("stops instead of advancing forever when no hero is measured", () => {
    expect(nextMeasuredBeat(beats, new Set(["supporting"]), 1)).toBe(-1);
    expect(nextMeasuredBeat(beats, new Set(), 1)).toBe(-1);
  });
});
