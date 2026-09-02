import { describe, expect, it } from "vitest";
import { formatClock } from "./format";

describe("formatClock", () => {
  it("renders a 24-hour clock with seconds for the eyebrow", () => {
    expect(formatClock(new Date(2026, 8, 1, 17, 5, 9))).toBe("17:05:09");
  });
});
