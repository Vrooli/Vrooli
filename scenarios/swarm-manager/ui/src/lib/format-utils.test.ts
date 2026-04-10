/**
 * Tests for Formatting Utilities
 *
 * [REQ:REQ-P0-003] Shared utility tests for format functions
 */

import { describe, it, expect } from "vitest";
import { formatFileSize, capitalize, formatDisplayText, getFileExtension, formatRelativeTime } from "./format-utils";

describe("formatFileSize", () => {
  it("returns '0 B' for zero bytes", () => {
    expect(formatFileSize(0)).toBe("0 B");
  });

  it("formats bytes without decimal", () => {
    expect(formatFileSize(500)).toBe("500 B");
  });

  it("formats kilobytes with one decimal place", () => {
    expect(formatFileSize(1024)).toBe("1.0 KB");
    expect(formatFileSize(1536)).toBe("1.5 KB");
    expect(formatFileSize(2048)).toBe("2.0 KB");
  });

  it("formats megabytes with one decimal place", () => {
    expect(formatFileSize(1048576)).toBe("1.0 MB");
    expect(formatFileSize(1572864)).toBe("1.5 MB");
  });

  it("formats gigabytes with one decimal place", () => {
    expect(formatFileSize(1073741824)).toBe("1.0 GB");
    expect(formatFileSize(1610612736)).toBe("1.5 GB");
  });

  it("handles large values", () => {
    expect(formatFileSize(5368709120)).toBe("5.0 GB");
  });
});

describe("capitalize", () => {
  it("capitalizes lowercase string", () => {
    expect(capitalize("running")).toBe("Running");
  });

  it("preserves already capitalized string", () => {
    expect(capitalize("Running")).toBe("Running");
  });

  it("returns empty string for empty input", () => {
    expect(capitalize("")).toBe("");
  });

  it("handles single character", () => {
    expect(capitalize("a")).toBe("A");
  });

  it("only capitalizes first letter", () => {
    expect(capitalize("hello world")).toBe("Hello world");
  });
});

describe("formatDisplayText", () => {
  it("formats snake_case text", () => {
    expect(formatDisplayText("in_progress")).toBe("In progress");
  });

  it("formats kebab-case text", () => {
    expect(formatDisplayText("high-priority")).toBe("High priority");
  });

  it("capitalizes simple text", () => {
    expect(formatDisplayText("ready")).toBe("Ready");
  });

  it("handles multiple separators", () => {
    expect(formatDisplayText("some_long_status")).toBe("Some long status");
  });

  it("returns empty string for empty input", () => {
    expect(formatDisplayText("")).toBe("");
  });

  it("handles mixed separators", () => {
    expect(formatDisplayText("some_mixed-case")).toBe("Some mixed case");
  });
});

describe("getFileExtension", () => {
  it("extracts simple file extension", () => {
    expect(getFileExtension("document.pdf")).toBe("pdf");
  });

  it("extracts extension in lowercase", () => {
    expect(getFileExtension("IMAGE.PNG")).toBe("png");
    expect(getFileExtension("readme.MD")).toBe("md");
  });

  it("handles multiple dots in filename", () => {
    expect(getFileExtension("script.test.ts")).toBe("ts");
    expect(getFileExtension("file.backup.tar.gz")).toBe("gz");
  });

  it("returns empty string for no extension", () => {
    expect(getFileExtension("README")).toBe("");
    expect(getFileExtension("Makefile")).toBe("");
  });

  it("handles dotfiles (files starting with dot)", () => {
    expect(getFileExtension(".gitignore")).toBe("gitignore");
    expect(getFileExtension(".env")).toBe("env");
  });

  it("returns empty string for empty input", () => {
    expect(getFileExtension("")).toBe("");
  });

  it("handles common code file extensions", () => {
    expect(getFileExtension("app.tsx")).toBe("tsx");
    expect(getFileExtension("main.go")).toBe("go");
    expect(getFileExtension("styles.css")).toBe("css");
    expect(getFileExtension("config.json")).toBe("json");
  });
});

describe("formatRelativeTime", () => {
  // Create a fixed "now" for testing
  const now = new Date("2026-01-28T12:00:00.000Z");

  describe("recent times", () => {
    it("returns 'just now' for times within 60 seconds", () => {
      expect(formatRelativeTime(now, now)).toBe("just now");
      expect(formatRelativeTime(new Date(now.getTime() - 30000), now)).toBe("just now");
      expect(formatRelativeTime(new Date(now.getTime() - 59000), now)).toBe("just now");
    });

    it("returns minutes ago for times 1-59 minutes", () => {
      expect(formatRelativeTime(new Date(now.getTime() - 60000), now)).toBe("1 minute ago");
      expect(formatRelativeTime(new Date(now.getTime() - 120000), now)).toBe("2 minutes ago");
      expect(formatRelativeTime(new Date(now.getTime() - 3540000), now)).toBe("59 minutes ago");
    });

    it("returns hours ago for times 1-23 hours", () => {
      expect(formatRelativeTime(new Date(now.getTime() - 3600000), now)).toBe("1 hour ago");
      expect(formatRelativeTime(new Date(now.getTime() - 7200000), now)).toBe("2 hours ago");
      expect(formatRelativeTime(new Date(now.getTime() - 82800000), now)).toBe("23 hours ago");
    });
  });

  describe("days ago", () => {
    it("returns days ago for times 1-30 days", () => {
      expect(formatRelativeTime(new Date(now.getTime() - 86400000), now)).toBe("1 day ago");
      expect(formatRelativeTime(new Date(now.getTime() - 172800000), now)).toBe("2 days ago");
      expect(formatRelativeTime(new Date(now.getTime() - 2592000000), now)).toBe("30 days ago");
    });
  });

  describe("older dates", () => {
    it("returns short date for times older than 30 days", () => {
      // 31 days ago should show date format
      const oldDate = new Date(now.getTime() - 2678400000);
      const result = formatRelativeTime(oldDate, now);
      // Result should be a date format like "Dec 28, 2025" (depends on locale)
      expect(result).toMatch(/\w{3}\s+\d{1,2},\s+\d{4}/);
    });
  });

  describe("edge cases", () => {
    it("handles string dates", () => {
      const dateStr = new Date(now.getTime() - 3600000).toISOString();
      expect(formatRelativeTime(dateStr, now)).toBe("1 hour ago");
    });

    it("returns 'Unknown' for invalid dates", () => {
      expect(formatRelativeTime("invalid-date")).toBe("Unknown");
    });

    it("handles future dates gracefully", () => {
      const futureDate = new Date(now.getTime() + 86400000);
      const result = formatRelativeTime(futureDate, now);
      // Should return formatted date, not "X ago"
      expect(result).toMatch(/\w{3}\s+\d{1,2},\s+\d{4}/);
    });
  });
});
