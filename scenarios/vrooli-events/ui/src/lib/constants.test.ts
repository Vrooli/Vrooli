// @vitest-environment node
import { describe, expect, it } from "vitest";
import {
  HEALTH_POLL_INTERVAL_MS,
  METRICS_POLL_INTERVAL_MS,
  STREAM_MAX_EVENTS,
  QUERY_LIMIT_OPTIONS,
  INPUT_CLASS,
  STATUS_COLORS,
} from "./constants";

// [REQ:REQ-ES-004] Verify tunable UI constants have valid values

describe("UI constants", () => {
  it("polling intervals are positive and reasonable", () => {
    expect(HEALTH_POLL_INTERVAL_MS).toBeGreaterThanOrEqual(1000);
    expect(HEALTH_POLL_INTERVAL_MS).toBeLessThanOrEqual(60000);
    expect(METRICS_POLL_INTERVAL_MS).toBeGreaterThanOrEqual(1000);
    expect(METRICS_POLL_INTERVAL_MS).toBeLessThanOrEqual(60000);
  });

  it("stream max events is positive", () => {
    expect(STREAM_MAX_EVENTS).toBeGreaterThan(0);
  });

  it("query limit options are sorted ascending", () => {
    const opts = [...QUERY_LIMIT_OPTIONS];
    for (let i = 1; i < opts.length; i++) {
      const prev = opts[i - 1];
      const curr = opts[i];
      if (prev === undefined || curr === undefined) {
        throw new Error("unexpected undefined in QUERY_LIMIT_OPTIONS");
      }
      expect(curr).toBeGreaterThan(prev);
    }
  });

  it("INPUT_CLASS is a non-empty string", () => {
    expect(INPUT_CLASS.length).toBeGreaterThan(0);
  });

  it("STATUS_COLORS covers all expected statuses", () => {
    for (const key of ["healthy", "degraded", "unhealthy", "unknown"]) {
      expect(STATUS_COLORS[key]).toBeDefined();
    }
  });
});
