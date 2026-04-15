/**
 * Tests for preflight utility functions.
 * These are pure functions with no side effects.
 */

import { describe, it, expect } from "vitest";
import {
  formatDuration,
  parseTimestamp,
  formatTimestamp,
  formatBytes,
  countLines,
  getListenURL,
  getServiceURL,
  getManifestHealthConfig,
  normalizeHealthPath,
  formatPortSummary,
  getBundleRootFromManifestPath,
  detectLikelyRootMismatch,
} from "./preflight-utils";

// ============================================================================
// Duration & Time Formatting
// ============================================================================

describe("formatDuration", () => {
  it("formats seconds correctly", () => {
    expect(formatDuration(5000)).toBe("5s");
    expect(formatDuration(1000)).toBe("1s");
    expect(formatDuration(0)).toBe("0s");
  });

  it("formats minutes and seconds", () => {
    expect(formatDuration(65000)).toBe("1m 5s");
    expect(formatDuration(120000)).toBe("2m 0s");
    expect(formatDuration(90000)).toBe("1m 30s");
  });

  it("formats hours and minutes", () => {
    expect(formatDuration(3665000)).toBe("1h 1m");
    expect(formatDuration(7200000)).toBe("2h 0m");
    expect(formatDuration(3600000)).toBe("1h 0m");
  });

  it("returns n/a for negative values", () => {
    expect(formatDuration(-1)).toBe("n/a");
    expect(formatDuration(-1000)).toBe("n/a");
  });

  it("returns n/a for non-finite values", () => {
    expect(formatDuration(Number.NaN)).toBe("n/a");
    expect(formatDuration(Number.POSITIVE_INFINITY)).toBe("n/a");
    expect(formatDuration(Number.NEGATIVE_INFINITY)).toBe("n/a");
  });
});

describe("parseTimestamp", () => {
  it("parses valid ISO timestamps", () => {
    const ts = parseTimestamp("2024-01-15T10:30:00Z");
    expect(ts).toBeTypeOf("number");
    expect(ts).toBeGreaterThan(0);
  });

  it("returns null for undefined", () => {
    expect(parseTimestamp(undefined)).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(parseTimestamp("")).toBeNull();
  });

  it("returns null for invalid timestamps", () => {
    expect(parseTimestamp("not-a-date")).toBeNull();
  });

  it("returns null for very old dates (before 2000)", () => {
    expect(parseTimestamp("1999-12-31T23:59:59Z")).toBeNull();
  });

  it("accepts dates from 2000 onwards", () => {
    // Use a date clearly within valid range (year 2001)
    const ts = parseTimestamp("2001-01-01T00:00:00Z");
    expect(ts).toBeTypeOf("number");
    expect(ts).toBeGreaterThan(0);
  });
});

describe("formatTimestamp", () => {
  it("formats valid timestamps", () => {
    const result = formatTimestamp("2024-01-15T10:30:00Z");
    // Result should be a localized time string
    expect(result).toMatch(/\d/);
  });

  it("returns empty string for undefined", () => {
    expect(formatTimestamp(undefined)).toBe("");
  });

  it("returns empty string for invalid timestamps", () => {
    expect(formatTimestamp("not-a-date")).toBe("");
  });
});

// ============================================================================
// Size & Count Formatting
// ============================================================================

describe("formatBytes", () => {
  it("returns empty string for zero or undefined", () => {
    expect(formatBytes(0)).toBe("");
    expect(formatBytes(undefined)).toBe("");
  });

  it("returns empty string for negative values", () => {
    expect(formatBytes(-100)).toBe("");
  });

  it("formats bytes correctly", () => {
    expect(formatBytes(100)).toBe("100 B");
    expect(formatBytes(1023)).toBe("1023 B");
  });

  it("formats kilobytes correctly", () => {
    expect(formatBytes(1024)).toBe("1.0 KB");
    expect(formatBytes(1536)).toBe("1.5 KB");
    expect(formatBytes(1048575)).toBe("1024.0 KB");
  });

  it("formats megabytes correctly", () => {
    expect(formatBytes(1048576)).toBe("1.0 MB");
    expect(formatBytes(1572864)).toBe("1.5 MB");
  });
});

describe("countLines", () => {
  it("returns 0 for undefined", () => {
    expect(countLines(undefined)).toBe(0);
  });

  it("returns 0 for empty string", () => {
    expect(countLines("")).toBe(0);
  });

  it("counts single line", () => {
    expect(countLines("hello")).toBe(1);
  });

  it("counts multiple lines", () => {
    expect(countLines("line1\nline2\nline3")).toBe(3);
  });

  it("handles trailing newline correctly", () => {
    expect(countLines("line1\nline2\n")).toBe(2);
  });

  it("handles Windows line endings", () => {
    expect(countLines("line1\r\nline2\r\n")).toBe(2);
  });
});

// ============================================================================
// URL & Port Extraction
// ============================================================================

describe("getListenURL", () => {
  it("returns null for undefined", () => {
    expect(getListenURL(undefined)).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(getListenURL("")).toBeNull();
  });

  it("extracts port from listening message", () => {
    expect(getListenURL("listening on 3000")).toBe("http://localhost:3000");
    expect(getListenURL("Listening on 8080")).toBe("http://localhost:8080");
  });

  it("handles messages with surrounding text", () => {
    expect(getListenURL("Server is listening on 5000 now")).toBe("http://localhost:5000");
  });

  it("returns null for messages without port", () => {
    expect(getListenURL("Server started")).toBeNull();
    expect(getListenURL("listening")).toBeNull();
  });

  it("returns null for invalid port numbers", () => {
    expect(getListenURL("listening on 0")).toBeNull();
    expect(getListenURL("listening on -1")).toBeNull();
  });
});

describe("getServiceURL", () => {
  const ports = {
    "web-app": { ui: 3000, api: 4000 },
    "api-server": { http: 8080 },
    "worker": { debug: 9229 },
    "invalid": { broken: -1 },
  };

  it("returns null for undefined ports", () => {
    expect(getServiceURL("web-app", undefined)).toBeNull();
  });

  it("returns null for unknown service", () => {
    expect(getServiceURL("unknown", ports)).toBeNull();
  });

  it("returns URL with preferred port name", () => {
    const result = getServiceURL("web-app", ports, "api");
    expect(result).toEqual({
      url: "http://localhost:4000",
      port: 4000,
      portName: "api",
    });
  });

  it("returns null when preferred port not found", () => {
    expect(getServiceURL("web-app", ports, "nonexistent")).toBeNull();
  });

  it("prioritizes ui > api > http for auto-selection", () => {
    const result1 = getServiceURL("web-app", ports);
    expect(result1?.portName).toBe("ui");

    const result2 = getServiceURL("api-server", ports);
    expect(result2?.portName).toBe("http");
  });

  it("falls back to first available port", () => {
    const result = getServiceURL("worker", ports);
    expect(result?.portName).toBe("debug");
  });

  it("returns null for invalid port values", () => {
    expect(getServiceURL("invalid", ports)).toBeNull();
  });
});

// ============================================================================
// Manifest Parsing
// ============================================================================

describe("getManifestHealthConfig", () => {
  it("returns null for null manifest", () => {
    expect(getManifestHealthConfig(null, "service")).toBeNull();
  });

  it("returns null for non-object manifest", () => {
    expect(getManifestHealthConfig("string", "service")).toBeNull();
    expect(getManifestHealthConfig(123, "service")).toBeNull();
  });

  it("returns null when services is not an array", () => {
    expect(getManifestHealthConfig({ services: {} }, "service")).toBeNull();
  });

  it("returns null when service not found", () => {
    const manifest = {
      services: [{ id: "other-service", health: { type: "http" } }],
    };
    expect(getManifestHealthConfig(manifest, "my-service")).toBeNull();
  });

  it("returns null when service has no health config", () => {
    const manifest = {
      services: [{ id: "my-service" }],
    };
    expect(getManifestHealthConfig(manifest, "my-service")).toBeNull();
  });

  it("extracts health config correctly", () => {
    const manifest = {
      services: [
        {
          id: "my-service",
          health: {
            type: "http",
            path: "/healthz",
            port_name: "api",
          },
        },
      ],
    };
    const result = getManifestHealthConfig(manifest, "my-service");
    expect(result).toEqual({
      type: "http",
      path: "/healthz",
      portName: "api",
    });
  });

  it("handles missing optional fields", () => {
    const manifest = {
      services: [
        {
          id: "my-service",
          health: { type: "tcp" },
        },
      ],
    };
    const result = getManifestHealthConfig(manifest, "my-service");
    expect(result).toEqual({
      type: "tcp",
      path: undefined,
      portName: undefined,
    });
  });
});

describe("normalizeHealthPath", () => {
  it("returns null for undefined", () => {
    expect(normalizeHealthPath(undefined)).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(normalizeHealthPath("")).toBeNull();
  });

  it("returns null for whitespace only", () => {
    expect(normalizeHealthPath("   ")).toBeNull();
  });

  it("preserves paths starting with /", () => {
    expect(normalizeHealthPath("/health")).toBe("/health");
    expect(normalizeHealthPath("/api/status")).toBe("/api/status");
  });

  it("adds leading / to paths without it", () => {
    expect(normalizeHealthPath("health")).toBe("/health");
    expect(normalizeHealthPath("api/status")).toBe("/api/status");
  });

  it("trims whitespace", () => {
    expect(normalizeHealthPath("  /health  ")).toBe("/health");
    expect(normalizeHealthPath("  health  ")).toBe("/health");
  });
});

// ============================================================================
// Port Summary
// ============================================================================

describe("formatPortSummary", () => {
  it("returns empty string for undefined", () => {
    expect(formatPortSummary(undefined)).toBe("");
  });

  it("returns empty string for empty object", () => {
    expect(formatPortSummary({})).toBe("");
  });

  it("formats single service", () => {
    const ports = { "web-app": { ui: 3000 } };
    expect(formatPortSummary(ports)).toBe("web-app(ui:3000)");
  });

  it("formats multiple ports per service", () => {
    const ports = { "web-app": { ui: 3000, api: 4000 } };
    expect(formatPortSummary(ports)).toBe("web-app(ui:3000, api:4000)");
  });

  it("formats multiple services", () => {
    const ports = {
      "web-app": { ui: 3000 },
      "api-server": { http: 8080 },
    };
    const result = formatPortSummary(ports);
    expect(result).toContain("web-app(ui:3000)");
    expect(result).toContain("api-server(http:8080)");
    expect(result).toContain(" · ");
  });
});

// ============================================================================
// Bundle Path Utilities
// ============================================================================

describe("getBundleRootFromManifestPath", () => {
  it("returns empty string for empty input", () => {
    expect(getBundleRootFromManifestPath("")).toBe("");
    expect(getBundleRootFromManifestPath("   ")).toBe("");
  });

  it("removes filename from Unix paths", () => {
    expect(getBundleRootFromManifestPath("/home/user/bundle/manifest.yaml")).toBe("/home/user/bundle");
  });

  it("removes filename from Windows paths", () => {
    expect(getBundleRootFromManifestPath("C:\\Users\\user\\bundle\\manifest.yaml")).toBe("C:\\Users\\user\\bundle");
  });

  it("handles paths without directory", () => {
    // When there's no directory separator, the regex doesn't match and returns the original
    expect(getBundleRootFromManifestPath("manifest.yaml")).toBe("manifest.yaml");
  });
});

describe("detectLikelyRootMismatch", () => {
  it("returns false when validation is not failed", () => {
    expect(detectLikelyRootMismatch(true, 1, 0, "/some/path")).toBe(false);
    expect(detectLikelyRootMismatch(undefined, 1, 0, "/some/path")).toBe(false);
  });

  it("returns false when no missing artifacts", () => {
    expect(detectLikelyRootMismatch(false, 0, 0, "/some/path")).toBe(false);
  });

  it("returns false when path includes staging directory", () => {
    expect(detectLikelyRootMismatch(false, 1, 0, "/path/vrooli/scenario-to-desktop/staging/manifest.yaml")).toBe(false);
  });

  it("returns true when artifacts missing and path is not staging", () => {
    expect(detectLikelyRootMismatch(false, 1, 0, "/home/user/my-bundle/manifest.yaml")).toBe(true);
    expect(detectLikelyRootMismatch(false, 0, 1, "/home/user/my-bundle/manifest.yaml")).toBe(true);
  });

  it("returns false for empty path", () => {
    expect(detectLikelyRootMismatch(false, 1, 0, "")).toBe(false);
    expect(detectLikelyRootMismatch(false, 1, 0, "   ")).toBe(false);
  });
});
