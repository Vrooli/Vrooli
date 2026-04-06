// @vitest-environment node
import { describe, expect, it } from "vitest";
import { formatBytes, formatTimestamp, safeStringify, truncate } from "./utils";

// [REQ:REQ-UI-001] Shared utility functions used across UI components

describe("formatBytes", () => {
  it("returns '0 B' for zero bytes", () => {
    expect(formatBytes(0)).toBe("0 B");
  });

  it("formats bytes under 1 KB", () => {
    expect(formatBytes(512)).toBe("512 B");
  });

  it("formats kilobytes", () => {
    expect(formatBytes(1024)).toBe("1.0 KB");
    expect(formatBytes(1536)).toBe("1.5 KB");
  });

  it("formats megabytes", () => {
    expect(formatBytes(1048576)).toBe("1.0 MB");
  });

  it("formats gigabytes", () => {
    expect(formatBytes(2147483648)).toBe("2.0 GB");
  });
});

describe("truncate", () => {
  it("returns the string unchanged if within max", () => {
    expect(truncate("hello", 10)).toBe("hello");
  });

  it("truncates and adds ellipsis if over max", () => {
    const result = truncate("hello world", 5);
    expect(result).toBe("hello\u2026");
    expect(result.length).toBe(6);
  });

  it("returns the exact string when length equals max", () => {
    expect(truncate("12345", 5)).toBe("12345");
  });
});

describe("formatTimestamp", () => {
  it("formats a valid ISO timestamp", () => {
    const result = formatTimestamp("2024-01-15T10:30:00Z");
    expect(result).not.toBe("Invalid date");
    expect(result.length).toBeGreaterThan(0);
  });

  it("returns 'Invalid date' for invalid input", () => {
    expect(formatTimestamp("not-a-date")).toBe("Invalid date");
  });

  it("returns 'Invalid date' for empty string", () => {
    expect(formatTimestamp("")).toBe("Invalid date");
  });
});

describe("safeStringify", () => {
  it("stringifies a plain object", () => {
    expect(safeStringify({ a: 1 })).toBe('{\n  "a": 1\n}');
  });

  it("stringifies an array", () => {
    expect(safeStringify([1, 2])).toBe('[\n  1,\n  2\n]');
  });

  it("stringifies primitives", () => {
    expect(safeStringify("hello")).toBe('"hello"');
    expect(safeStringify(42)).toBe("42");
    expect(safeStringify(null)).toBe("null");
  });

  it("returns fallback for circular references", () => {
    const obj: Record<string, unknown> = {};
    obj.self = obj;
    expect(safeStringify(obj)).toBe("[Unable to serialize value]");
  });
});
