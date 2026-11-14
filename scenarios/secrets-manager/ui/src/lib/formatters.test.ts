import { describe, expect, it } from "vitest";
import { formatTimestamp, percentage } from "./formatters";

describe("formatters", () => {
  it("preserves missing and invalid timestamps while formatting valid values", () => {
    const value = "2026-07-23T12:34:56.000Z";
    expect(formatTimestamp()).toBe("—");
    expect(formatTimestamp("not-a-date")).toBe("not-a-date");
    expect(formatTimestamp(value)).toBe(new Date(value).toLocaleString());
  });

  it("returns rounded percentages and handles empty totals", () => {
    expect(percentage(3, 0)).toBe(0);
    expect(percentage(1, 3)).toBe(33);
    expect(percentage(2, 3)).toBe(67);
  });
});
