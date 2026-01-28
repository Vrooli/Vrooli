import { describe, it, expect } from "vitest";
import { defaultQueryOptions } from "./query-utils";
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
