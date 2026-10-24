import { describe, expect, it } from "vitest";
import type { Constellation, Reading } from "./api";
import { skyState, tallySky } from "./sky";
import { authoredSample, makeReading } from "../test-utils/readings";

const reading = (id: string, overrides: Partial<Reading> = {}): Reading => makeReading({ id, label: id, ...overrides });
const room = (id: string, readings: Reading[]): Constellation => ({ room: { id, title: id }, readings });

describe("skyState", () => {
  it("counts a current NOW reading as measured", () => {
    expect(skyState(reading("a", { value: 3 }))).toBe("measured");
    expect(skyState(reading("a", { value: 3, trust: "CACHED" }))).toBe("measured");
  });
  it("treats a NOW sensor with no number or an untrusted one as failing", () => {
    expect(skyState(reading("a", { trust: "UNAVAILABLE" }))).toBe("failing");
    expect(skyState(reading("a", { value: 900, trust: "UNTRUSTED" }))).toBe("failing");
  });
  it("keeps an in-reach reading in reach even when a number came back, as the resolver draws it", () => {
    expect(skyState(reading("a", { coverage: "IN-REACH", value: 168, sample: authoredSample(2840) }))).toBe("in-reach");
  });
  it("treats missing and unregistered coverage as missing", () => {
    expect(skyState(reading("a", { coverage: "MISSING", trust: "UNAVAILABLE" }))).toBe("missing");
    expect(skyState(reading("a", { coverage: "UNREGISTERED", trust: "UNAVAILABLE" }))).toBe("missing");
  });
});

describe("tallySky", () => {
  it("counts a signal shown in two rooms once and splits the rest by reason", () => {
    const shared = reading("shared", { value: 1 });
    const tally = tallySky([
      room("hive", [shared, reading("cached", { value: 2, trust: "CACHED" }), reading("reach", { coverage: "IN-REACH" })]),
      room("forge", [shared, reading("down", { trust: "UNAVAILABLE" }), reading("gap", { coverage: "MISSING", trust: "UNAVAILABLE" })]),
    ]);
    expect(tally).toEqual({ rooms: 2, total: 5, measured: 2, cached: 1, failing: 1, inReach: 1, missing: 1 });
  });
  it("is empty for an empty board", () => {
    expect(tallySky([])).toMatchObject({ rooms: 0, total: 0, measured: 0 });
  });
});
