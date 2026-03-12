// [REQ:REQ-P0-001] Reference Scenario Registry - Utility tests
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cn, formatDate, formatRelativeDate } from "./utils";

describe("cn - className utility", () => {
  it("merges class names correctly", () => {
    // ARRANGE & ACT
    const result = cn("text-sm", "font-bold");

    // ASSERT
    expect(result).toBe("text-sm font-bold");
  });

  it("handles conditional classes", () => {
    // ARRANGE
    const isActive = true;

    // ACT
    const result = cn("base", isActive && "active");

    // ASSERT
    expect(result).toBe("base active");
  });

  it("removes falsy values", () => {
    // ACT
    const result = cn("base", false, null, undefined, "final");

    // ASSERT
    expect(result).toBe("base final");
  });

  it("merges Tailwind classes correctly", () => {
    // ACT - twMerge should handle Tailwind specificity
    const result = cn("p-2 p-4");

    // ASSERT - twMerge removes duplicate utilities
    expect(result).toBe("p-4");
  });

  it("handles array of classes", () => {
    // ACT
    const result = cn(["text-sm", "text-blue-500"]);

    // ASSERT
    expect(result).toBe("text-sm text-blue-500");
  });

  it("handles object syntax", () => {
    // ACT
    const result = cn({
      "text-sm": true,
      "font-bold": false,
      "text-red-500": true
    });

    // ASSERT
    expect(result).toBe("text-sm text-red-500");
  });
});

describe("formatDate", () => {
  it("formats date without time by default", () => {
    // ARRANGE
    const dateStr = "2024-06-15T10:30:00Z";

    // ACT
    const result = formatDate(dateStr);

    // ASSERT - Format: "Jun 15, 2024"
    expect(result).toMatch(/Jun\s+15,\s+2024/);
  });

  it("formats date with time when includeTime is true", () => {
    // ARRANGE
    const dateStr = "2024-06-15T10:30:00Z";

    // ACT
    const result = formatDate(dateStr, { includeTime: true });

    // ASSERT - Should include time portion
    expect(result).toContain("2024");
    expect(result).toMatch(/\d{1,2}:\d{2}/); // Time format
  });

  it("handles different date strings", () => {
    // ARRANGE
    const dates = [
      "2020-01-01T00:00:00Z",
      "2023-12-31T23:59:59Z",
      "2024-02-29T12:00:00Z" // Leap year
    ];

    // ACT & ASSERT - Should not throw for valid dates
    for (const dateStr of dates) {
      expect(() => formatDate(dateStr)).not.toThrow();
    }
  });
});

describe("formatRelativeDate", () => {
  let mockNow: Date;

  beforeEach(() => {
    // Mock the current time to ensure consistent tests
    mockNow = new Date("2024-06-15T12:00:00Z");
    vi.useFakeTimers();
    vi.setSystemTime(mockNow);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns 'today' for same day", () => {
    // ARRANGE
    const dateStr = "2024-06-15T08:00:00Z";

    // ACT
    const result = formatRelativeDate(dateStr);

    // ASSERT
    expect(result).toBe("today");
  });

  it("returns 'yesterday' for one day ago", () => {
    // ARRANGE
    const dateStr = "2024-06-14T08:00:00Z";

    // ACT
    const result = formatRelativeDate(dateStr);

    // ASSERT
    expect(result).toBe("yesterday");
  });

  it("returns 'X days ago' for dates within 30 days", () => {
    // ARRANGE
    const dateStr = "2024-06-10T08:00:00Z"; // 5 days ago

    // ACT
    const result = formatRelativeDate(dateStr);

    // ASSERT
    expect(result).toBe("5 days ago");
  });

  it("returns absolute date for dates > 30 days ago", () => {
    // ARRANGE
    const dateStr = "2024-05-01T08:00:00Z"; // 45 days ago

    // ACT
    const result = formatRelativeDate(dateStr);

    // ASSERT - Should fall back to formatDate output
    expect(result).toMatch(/May\s+1,\s+2024/);
  });

  it("handles boundary at exactly 30 days", () => {
    // ARRANGE - Exactly 29 days ago (still relative)
    const dateStr29 = "2024-05-17T12:00:00Z";

    // ACT
    const result29 = formatRelativeDate(dateStr29);

    // ASSERT
    expect(result29).toBe("29 days ago");
  });
});
