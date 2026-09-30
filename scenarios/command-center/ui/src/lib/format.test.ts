import { describe, expect, it } from "vitest";
import { displayFormat, formatClock } from "./format";

describe("formatClock", () => {
  it("renders a 24-hour clock with seconds for the eyebrow", () => {
    expect(formatClock(new Date(2026, 8, 1, 17, 5, 9))).toBe("17:05:09");
  });
});

describe("displayFormat", () => {
  it("uses compact notation for large count and currency figures", () => {
    expect(displayFormat(2_134, "integer")).toBe("compact");
    expect(displayFormat(1_000_000, "currency")).toBe("currency.compact");
    expect(displayFormat(213, "integer")).toBe("integer");
  });
});
