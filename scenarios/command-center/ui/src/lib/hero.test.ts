import { describe, expect, it } from "vitest";
import type { Reading } from "../lib/api";
import { pickHero } from "./hero";
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
