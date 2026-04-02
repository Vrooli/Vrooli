import { describe, it, expect } from "vitest";
import { buildQueryString, defaultQueryOptions } from "./query-utils";
import { dataFetchingConfig } from "../config";

/**
 * Query Utilities Tests
 *
 * These tests verify:
 * - Default query options are correctly sourced from config
 * - Retry delay function follows exponential backoff
 *
 * [REQ:PHASE12] Test query utility consolidation
 */

describe("defaultQueryOptions", () => {
  it("has retry count from config", () => {
    expect(defaultQueryOptions.retry).toBe(dataFetchingConfig.retryCount);
  });

  it("has stale time from config", () => {
    expect(defaultQueryOptions.staleTime).toBe(dataFetchingConfig.staleTimeMs);
  });

  it("has gc time from config", () => {
    expect(defaultQueryOptions.gcTime).toBe(dataFetchingConfig.cacheTimeMs);
  });

  it("has refetch on window focus from config", () => {
    expect(defaultQueryOptions.refetchOnWindowFocus).toBe(
      dataFetchingConfig.refetchOnWindowFocus
    );
  });

  describe("retryDelay", () => {
    it("returns base delay for first attempt", () => {
      const delay = defaultQueryOptions.retryDelay(0);
      expect(delay).toBe(dataFetchingConfig.retryDelayMs);
    });

    it("doubles delay for each subsequent attempt (exponential backoff)", () => {
      const base = dataFetchingConfig.retryDelayMs;

      expect(defaultQueryOptions.retryDelay(0)).toBe(base * 1);   // 2^0 = 1
      expect(defaultQueryOptions.retryDelay(1)).toBe(base * 2);   // 2^1 = 2
      expect(defaultQueryOptions.retryDelay(2)).toBe(base * 4);   // 2^2 = 4
      expect(defaultQueryOptions.retryDelay(3)).toBe(base * 8);   // 2^3 = 8
    });

    it("follows formula: retryDelayMs * 2^attemptIndex", () => {
      const base = dataFetchingConfig.retryDelayMs;

      for (let i = 0; i < 5; i++) {
        const expected = base * Math.pow(2, i);
        expect(defaultQueryOptions.retryDelay(i)).toBe(expected);
      }
    });
  });
});

describe("buildQueryString", () => {
  it("returns empty string for empty params", () => {
    expect(buildQueryString({})).toBe("");
  });

  it("returns query string for a single param", () => {
    expect(buildQueryString({ key: "value" })).toBe("?key=value");
  });

  it("returns query string for multiple params", () => {
    const result = buildQueryString({ key1: "value1", key2: "value2" });
    expect(result).toBe("?key1=value1&key2=value2");
  });

  it("joins array values with commas", () => {
    const result = buildQueryString({ kinds: ["idea", "fix"] });
    expect(result).toBe("?kinds=idea%2Cfix");
  });

  it("skips undefined values", () => {
    const result = buildQueryString({ status: "active", mode: undefined });
    expect(result).toBe("?status=active");
  });

  it("skips null values", () => {
    const result = buildQueryString({ status: "active", mode: null });
    expect(result).toBe("?status=active");
  });

  it("skips empty string values", () => {
    const result = buildQueryString({ status: "active", mode: "" });
    expect(result).toBe("?status=active");
  });

  it("skips empty arrays", () => {
    const result = buildQueryString({ status: "active", tags: [] });
    expect(result).toBe("?status=active");
  });

  it("stringifies boolean values", () => {
    const result = buildQueryString({ active: true, archived: false });
    expect(result).toBe("?active=true&archived=false");
  });

  it("stringifies number values", () => {
    const result = buildQueryString({ limit: 10, offset: 0 });
    expect(result).toBe("?limit=10&offset=0");
  });

  it("returns empty string when all values are skipped", () => {
    expect(buildQueryString({ a: undefined, b: null, c: "" })).toBe("");
  });
});
