/**
 * Tests for pipeline utility functions.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  mapPipelineStatus,
  parseLogEntry,
  parseLogs,
  filterLogsByLevel,
  getLogLevelStyle,
  formatLogTimestamp,
  getLatestSignificantLog,
  generateIdempotencyKey,
  getSessionId,
  resetSessionId,
  generateUniqueIdempotencyKey,
  type MappedBuildStatus,
  type LogLevel,
  type ParsedLogEntry,
} from "./pipeline-utils";

// ============================================================================
// Pipeline Status Mapping
// ============================================================================

describe("mapPipelineStatus", () => {
  it("maps pending to building", () => {
    expect(mapPipelineStatus("pending")).toBe("building");
  });

  it("maps running to building", () => {
    expect(mapPipelineStatus("running")).toBe("building");
  });

  it("maps completed to ready", () => {
    expect(mapPipelineStatus("completed")).toBe("ready");
  });

  it("maps failed to failed", () => {
    expect(mapPipelineStatus("failed")).toBe("failed");
  });

  it("maps cancelled to failed", () => {
    expect(mapPipelineStatus("cancelled")).toBe("failed");
  });

  it("maps unknown status to building", () => {
    expect(mapPipelineStatus("unknown")).toBe("building");
    expect(mapPipelineStatus("")).toBe("building");
  });
});

// ============================================================================
// Log Parsing
// ============================================================================

describe("parseLogEntry", () => {
  it("parses structured log entry correctly", () => {
    const raw = "[2024-01-15T10:30:00Z] [INFO] Stage bundle starting";
    const result = parseLogEntry(raw);
    expect(result.timestamp).toBe("2024-01-15T10:30:00Z");
    expect(result.level).toBe("INFO");
    expect(result.message).toBe("Stage bundle starting");
    expect(result.raw).toBe(raw);
  });

  it("parses log with milliseconds in timestamp", () => {
    const raw = "[2024-01-15T10:30:00.123Z] [WARN] Warning message";
    const result = parseLogEntry(raw);
    expect(result.timestamp).toBe("2024-01-15T10:30:00.123Z");
    expect(result.level).toBe("WARN");
    expect(result.message).toBe("Warning message");
  });

  it("parses ERROR level logs", () => {
    const raw = "[2024-01-15T10:30:00Z] [ERROR] Something went wrong";
    const result = parseLogEntry(raw);
    expect(result.level).toBe("ERROR");
    expect(result.message).toBe("Something went wrong");
  });

  it("parses DEBUG level logs", () => {
    const raw = "[2024-01-15T10:30:00Z] [DEBUG] Debug info";
    const result = parseLogEntry(raw);
    expect(result.level).toBe("DEBUG");
    expect(result.message).toBe("Debug info");
  });

  it("returns fallback for unstructured logs", () => {
    const raw = "Plain text log message";
    const result = parseLogEntry(raw);
    expect(result.level).toBe("INFO");
    expect(result.message).toBe("Plain text log message");
    expect(result.raw).toBe(raw);
    // timestamp should be current time (roughly)
    const parsed = new Date(result.timestamp);
    expect(parsed.getTime()).toBeCloseTo(Date.now(), -3); // within 1 second
  });

  it("handles log with extra whitespace", () => {
    // The \s* in the regex consumes whitespace after [INFO]
    const raw = "[2024-01-15T10:30:00Z]  [INFO]  Extra spaces";
    const result = parseLogEntry(raw);
    expect(result.level).toBe("INFO");
    expect(result.message).toBe("Extra spaces");
  });
});

describe("parseLogs", () => {
  it("parses array of logs", () => {
    const logs = [
      "[2024-01-15T10:30:00Z] [INFO] First",
      "[2024-01-15T10:30:01Z] [ERROR] Second",
      "Unstructured third",
    ];
    const result = parseLogs(logs);
    expect(result).toHaveLength(3);
    expect(result[0].level).toBe("INFO");
    expect(result[1].level).toBe("ERROR");
    expect(result[2].level).toBe("INFO"); // fallback
  });

  it("handles empty array", () => {
    expect(parseLogs([])).toEqual([]);
  });
});

describe("filterLogsByLevel", () => {
  const logs: ParsedLogEntry[] = [
    { timestamp: "t1", level: "DEBUG", message: "debug", raw: "r1" },
    { timestamp: "t2", level: "INFO", message: "info", raw: "r2" },
    { timestamp: "t3", level: "WARN", message: "warn", raw: "r3" },
    { timestamp: "t4", level: "ERROR", message: "error", raw: "r4" },
  ];

  it("filters to DEBUG and above", () => {
    const result = filterLogsByLevel(logs, "DEBUG");
    expect(result).toHaveLength(4);
  });

  it("filters to INFO and above", () => {
    const result = filterLogsByLevel(logs, "INFO");
    expect(result).toHaveLength(3);
    expect(result.find(l => l.level === "DEBUG")).toBeUndefined();
  });

  it("filters to WARN and above", () => {
    const result = filterLogsByLevel(logs, "WARN");
    expect(result).toHaveLength(2);
    expect(result.map(l => l.level)).toEqual(["WARN", "ERROR"]);
  });

  it("filters to ERROR only", () => {
    const result = filterLogsByLevel(logs, "ERROR");
    expect(result).toHaveLength(1);
    expect(result[0].level).toBe("ERROR");
  });
});

// ============================================================================
// Log Level Styling
// ============================================================================

describe("getLogLevelStyle", () => {
  it("returns red colors for ERROR", () => {
    const style = getLogLevelStyle("ERROR");
    expect(style.color).toContain("red");
    expect(style.bg).toContain("red");
  });

  it("returns yellow colors for WARN", () => {
    const style = getLogLevelStyle("WARN");
    expect(style.color).toContain("yellow");
    expect(style.bg).toContain("yellow");
  });

  it("returns slate colors for INFO", () => {
    const style = getLogLevelStyle("INFO");
    expect(style.color).toContain("slate");
    expect(style.bg).toBe("");
  });

  it("returns slate colors for DEBUG", () => {
    const style = getLogLevelStyle("DEBUG");
    expect(style.color).toContain("slate");
    expect(style.bg).toBe("");
  });
});

// ============================================================================
// Log Timestamp Formatting
// ============================================================================

describe("formatLogTimestamp", () => {
  it("formats recent timestamps as relative time", () => {
    const now = new Date();
    const recent = new Date(now.getTime() - 30000); // 30 seconds ago
    const result = formatLogTimestamp(recent.toISOString());
    expect(result).toMatch(/30s ago/);
  });

  it("formats timestamps from minutes ago", () => {
    const now = new Date();
    const minutesAgo = new Date(now.getTime() - 5 * 60 * 1000); // 5 minutes ago
    const result = formatLogTimestamp(minutesAgo.toISOString());
    expect(result).toMatch(/5m ago/);
  });

  it("formats older timestamps as absolute time", () => {
    const now = new Date();
    const hoursAgo = new Date(now.getTime() - 2 * 60 * 60 * 1000); // 2 hours ago
    const result = formatLogTimestamp(hoursAgo.toISOString());
    // Should contain time format like HH:MM:SS
    expect(result).toMatch(/\d{1,2}:\d{2}:\d{2}/);
  });
});

// ============================================================================
// Latest Significant Log
// ============================================================================

describe("getLatestSignificantLog", () => {
  it("returns null for empty array", () => {
    expect(getLatestSignificantLog([])).toBeNull();
  });

  it("returns latest ERROR if present", () => {
    const logs: ParsedLogEntry[] = [
      { timestamp: "t1", level: "INFO", message: "info", raw: "r1" },
      { timestamp: "t2", level: "ERROR", message: "first error", raw: "r2" },
      { timestamp: "t3", level: "INFO", message: "more info", raw: "r3" },
      { timestamp: "t4", level: "ERROR", message: "latest error", raw: "r4" },
    ];
    const result = getLatestSignificantLog(logs);
    expect(result?.message).toBe("latest error");
  });

  it("returns latest WARN if no ERROR", () => {
    const logs: ParsedLogEntry[] = [
      { timestamp: "t1", level: "INFO", message: "info", raw: "r1" },
      { timestamp: "t2", level: "WARN", message: "first warn", raw: "r2" },
      { timestamp: "t3", level: "WARN", message: "latest warn", raw: "r3" },
    ];
    const result = getLatestSignificantLog(logs);
    expect(result?.message).toBe("latest warn");
  });

  it("returns latest log if no ERROR or WARN", () => {
    const logs: ParsedLogEntry[] = [
      { timestamp: "t1", level: "INFO", message: "first", raw: "r1" },
      { timestamp: "t2", level: "DEBUG", message: "debug", raw: "r2" },
      { timestamp: "t3", level: "INFO", message: "latest", raw: "r3" },
    ];
    const result = getLatestSignificantLog(logs);
    expect(result?.message).toBe("latest");
  });
});

// ============================================================================
// Idempotency Key Generation
// ============================================================================

describe("getSessionId", () => {
  beforeEach(() => {
    resetSessionId();
  });

  it("returns a session ID", () => {
    const id = getSessionId();
    expect(id).toBeTypeOf("string");
    expect(id.length).toBeGreaterThan(0);
  });

  it("returns the same ID on subsequent calls", () => {
    const id1 = getSessionId();
    const id2 = getSessionId();
    expect(id1).toBe(id2);
  });
});

describe("resetSessionId", () => {
  it("causes getSessionId to return a new ID", () => {
    const id1 = getSessionId();
    resetSessionId();
    const id2 = getSessionId();
    expect(id1).not.toBe(id2);
  });
});

describe("generateIdempotencyKey", () => {
  beforeEach(() => {
    resetSessionId();
  });

  it("generates key from scenario name", () => {
    const key = generateIdempotencyKey("my-scenario");
    expect(key).toContain("my-scenario");
  });

  it("includes stage when provided", () => {
    const key = generateIdempotencyKey("my-scenario", "bundle");
    expect(key).toContain("my-scenario");
    expect(key).toContain("bundle");
  });

  it("generates same key for same inputs within session", () => {
    const key1 = generateIdempotencyKey("my-scenario", "build");
    const key2 = generateIdempotencyKey("my-scenario", "build");
    expect(key1).toBe(key2);
  });

  it("generates different key for different scenarios", () => {
    const key1 = generateIdempotencyKey("scenario-a");
    const key2 = generateIdempotencyKey("scenario-b");
    expect(key1).not.toBe(key2);
  });

  it("generates different key for different stages", () => {
    const key1 = generateIdempotencyKey("my-scenario", "bundle");
    const key2 = generateIdempotencyKey("my-scenario", "build");
    expect(key1).not.toBe(key2);
  });

  it("uses custom session ID when provided", () => {
    const key1 = generateIdempotencyKey("my-scenario", undefined, "session-1");
    const key2 = generateIdempotencyKey("my-scenario", undefined, "session-2");
    expect(key1).not.toBe(key2);
    expect(key1).toContain("session-1");
    expect(key2).toContain("session-2");
  });
});

describe("generateUniqueIdempotencyKey", () => {
  it("generates unique keys on each call", () => {
    const key1 = generateUniqueIdempotencyKey("my-scenario");
    const key2 = generateUniqueIdempotencyKey("my-scenario");
    expect(key1).not.toBe(key2);
  });

  it("includes scenario name", () => {
    const key = generateUniqueIdempotencyKey("test-scenario");
    expect(key).toContain("test-scenario");
  });

  it("includes stage when provided", () => {
    const key = generateUniqueIdempotencyKey("my-scenario", "bundle");
    expect(key).toContain("bundle");
  });
});
