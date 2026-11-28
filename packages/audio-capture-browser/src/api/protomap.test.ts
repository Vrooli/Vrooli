import { describe, expect, it } from "vitest";
import { timestampToISO } from "./protomap";

describe("timestampToISO", () => {
  it("returns an empty string for an absent timestamp", () => {
    expect(timestampToISO(undefined)).toBe("");
  });

  it("converts bigint seconds and floors nanos to milliseconds", () => {
    expect(timestampToISO({ seconds: 1_700_000_000n, nanos: 500_999_999 }))
      .toBe("2023-11-14T22:13:20.500Z");
  });

  it("converts numeric seconds and defaults missing timestamp fields to zero", () => {
    expect(timestampToISO({ seconds: 1, nanos: 1_999_999 })).toBe("1970-01-01T00:00:01.001Z");
    expect(timestampToISO({})).toBe("1970-01-01T00:00:00.000Z");
  });

  it("passes through legacy strings", () => {
    expect(timestampToISO("already formatted")).toBe("already formatted");
  });
});
