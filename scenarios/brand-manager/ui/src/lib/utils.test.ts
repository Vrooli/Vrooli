import { describe, it, expect } from "vitest";
import { cn, formatDate, formatDateTime, formatContrastRatio } from "./utils";

// [REQ:BM-REQ-UI-DASHBOARD] [REQ:BM-REQ-WCAG-CALC]

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("a", "b")).toBe("a b");
  });

  it("handles conditional classes", () => {
    const conditional = false;
    expect(cn("a", conditional && "b", "c")).toBe("a c");
  });

  it("merges tailwind conflicts", () => {
    expect(cn("p-2", "p-4")).toBe("p-4");
  });
});

describe("formatDate", () => {
  it("formats ISO date string to locale date", () => {
    const result = formatDate("2026-03-15T10:30:00Z");
    // Locale-dependent, but should contain the date parts
    expect(result).toBeTruthy();
    expect(result).toContain("2026");
  });

  it("handles date-only ISO strings", () => {
    const result = formatDate("2026-01-01T00:00:00Z");
    expect(result).toBeTruthy();
  });
});

describe("formatDateTime", () => {
  it("formats ISO string to locale date-time", () => {
    const result = formatDateTime("2026-03-15T10:30:00Z");
    expect(result).toBeTruthy();
    expect(result).toContain("2026");
  });

  it("includes time component", () => {
    const result = formatDateTime("2026-06-15T14:30:00Z");
    // Should be longer than formatDate since it includes time
    const dateOnly = formatDate("2026-06-15T14:30:00Z");
    expect(result.length).toBeGreaterThan(dateOnly.length);
  });
});

describe("formatContrastRatio", () => {
  it("formats ratio with one decimal and :1 suffix", () => {
    expect(formatContrastRatio(4.523)).toBe("4.5:1");
  });

  it("formats exact integers", () => {
    expect(formatContrastRatio(7)).toBe("7.0:1");
  });

  it("rounds correctly", () => {
    expect(formatContrastRatio(3.06)).toBe("3.1:1");
    expect(formatContrastRatio(3.04)).toBe("3.0:1");
  });

  it("handles ratio of 1", () => {
    expect(formatContrastRatio(1)).toBe("1.0:1");
  });

  it("handles 21:1 max ratio", () => {
    expect(formatContrastRatio(21)).toBe("21.0:1");
  });
});
