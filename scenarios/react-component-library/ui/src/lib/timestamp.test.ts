import { describe, expect, it } from "vitest";

import { timestampToDate } from "./timestamp";

describe("timestampToDate", () => {
  const date = new Date("2025-04-03T12:30:00.000Z");

  it.each([
    ["ISO string", date.toISOString(), date],
    ["epoch milliseconds", date.getTime(), date],
    ["Date object", date, date],
    ["protobuf style seconds", { seconds: BigInt(date.getTime() / 1000) }, date],
    ["Timestamp instance", { toDate: () => date }, date],
  ])("reads %s values", (_label, input, expected) => {
    expect(timestampToDate(input)).toEqual(expected);
  });

  it.each([undefined, null, 0, "", "not a date", Number.NaN, {}, { seconds: "42" }])(
    "returns undefined for unsupported or invalid values (%j)",
    (input) => expect(timestampToDate(input)).toBeUndefined(),
  );

  it("rejects invalid Date values returned by supported timestamp objects", () => {
    expect(timestampToDate(new Date(Number.NaN))).toBeUndefined();
    expect(timestampToDate({ toDate: () => new Date(Number.NaN) })).toBeUndefined();
    expect(timestampToDate({ seconds: Number.POSITIVE_INFINITY })).toBeUndefined();
  });
});
