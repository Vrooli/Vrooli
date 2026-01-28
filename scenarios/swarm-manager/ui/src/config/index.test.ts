import { describe, it, expect } from "vitest";
import {
  config,
  dataFetchingConfig,
  displayLimitsConfig,
  recommendationConfig,
  insightsConfig,
  uiBehaviorConfig,
  apiConfig,
} from "./index";

/**
 * Configuration module tests.
 *
 * These tests verify that the configuration values are valid and
 * within expected bounds. They serve as a safety net to catch
 * accidental changes that could break the application.
 */
describe("Configuration", () => {
  describe("dataFetchingConfig", () => {
    it("has valid retry count", () => {
      expect(dataFetchingConfig.retryCount).toBeGreaterThanOrEqual(0);
      expect(dataFetchingConfig.retryCount).toBeLessThanOrEqual(5);
    });

    it("has valid retry delay", () => {
      expect(dataFetchingConfig.retryDelayMs).toBeGreaterThanOrEqual(500);
      expect(dataFetchingConfig.retryDelayMs).toBeLessThanOrEqual(5000);
    });

    it("has valid stale time", () => {
      expect(dataFetchingConfig.staleTimeMs).toBeGreaterThanOrEqual(5000);
      expect(dataFetchingConfig.staleTimeMs).toBeLessThanOrEqual(300000);
    });

    it("has valid cache time", () => {
      expect(dataFetchingConfig.cacheTimeMs).toBeGreaterThanOrEqual(60000);
      expect(dataFetchingConfig.cacheTimeMs).toBeLessThanOrEqual(600000);
    });

    it("has boolean refetchOnWindowFocus", () => {
      expect(typeof dataFetchingConfig.refetchOnWindowFocus).toBe("boolean");
    });
  });

  describe("displayLimitsConfig", () => {
    it("has valid idea card max tags", () => {
      expect(displayLimitsConfig.ideaCardMaxTags).toBeGreaterThanOrEqual(1);
      expect(displayLimitsConfig.ideaCardMaxTags).toBeLessThanOrEqual(10);
    });

    it("has valid scenario card max tags", () => {
      expect(displayLimitsConfig.scenarioCardMaxTags).toBeGreaterThanOrEqual(1);
      expect(displayLimitsConfig.scenarioCardMaxTags).toBeLessThanOrEqual(10);
    });

    it("has valid description line clamp", () => {
      expect(displayLimitsConfig.descriptionLineClamp).toBeGreaterThanOrEqual(1);
      expect(displayLimitsConfig.descriptionLineClamp).toBeLessThanOrEqual(5);
    });

    it("has valid default page size", () => {
      expect(displayLimitsConfig.defaultPageSize).toBeGreaterThanOrEqual(10);
      expect(displayLimitsConfig.defaultPageSize).toBeLessThanOrEqual(100);
    });
  });

  describe("recommendationConfig", () => {
    it("has valid minimum completeness threshold", () => {
      expect(recommendationConfig.minimumCompletenessThreshold).toBeGreaterThanOrEqual(0);
      expect(recommendationConfig.minimumCompletenessThreshold).toBeLessThanOrEqual(100);
    });

    it("has valid max active recommendations", () => {
      expect(recommendationConfig.maxActiveRecommendationsPerScenario).toBeGreaterThanOrEqual(1);
      expect(recommendationConfig.maxActiveRecommendationsPerScenario).toBeLessThanOrEqual(20);
    });

    it("has valid YOLO mode delay", () => {
      expect(recommendationConfig.yoloModeDelayMs).toBeGreaterThanOrEqual(1000);
      expect(recommendationConfig.yoloModeDelayMs).toBeLessThanOrEqual(60000);
    });

    it("has valid YOLO mode allowed priorities", () => {
      expect(Array.isArray(recommendationConfig.yoloModeAllowedPriorities)).toBe(true);
      expect(recommendationConfig.yoloModeAllowedPriorities.length).toBeGreaterThan(0);
      recommendationConfig.yoloModeAllowedPriorities.forEach((priority) => {
        expect(priority).toBeGreaterThanOrEqual(1);
        expect(priority).toBeLessThanOrEqual(5);
      });
    });
  });

  describe("insightsConfig", () => {
    it("has valid minimum completed scenarios", () => {
      expect(insightsConfig.minimumCompletedScenarios).toBeGreaterThanOrEqual(1);
      expect(insightsConfig.minimumCompletedScenarios).toBeLessThanOrEqual(10);
    });

    it("has valid pattern window size", () => {
      expect(insightsConfig.patternWindowSize).toBeGreaterThanOrEqual(10);
      expect(insightsConfig.patternWindowSize).toBeLessThanOrEqual(200);
    });

    it("has valid confidence threshold", () => {
      expect(insightsConfig.confidenceThreshold).toBeGreaterThanOrEqual(0.5);
      expect(insightsConfig.confidenceThreshold).toBeLessThanOrEqual(0.95);
    });
  });

  describe("uiBehaviorConfig", () => {
    it("has valid search debounce delay", () => {
      expect(uiBehaviorConfig.searchDebounceMs).toBeGreaterThanOrEqual(100);
      expect(uiBehaviorConfig.searchDebounceMs).toBeLessThanOrEqual(1000);
    });

    it("has valid toast duration", () => {
      expect(uiBehaviorConfig.toastDurationMs).toBeGreaterThanOrEqual(2000);
      expect(uiBehaviorConfig.toastDurationMs).toBeLessThanOrEqual(10000);
    });

    it("has boolean flags", () => {
      expect(typeof uiBehaviorConfig.useSkeletonLoading).toBe("boolean");
      expect(typeof uiBehaviorConfig.enableKeyboardShortcuts).toBe("boolean");
      expect(typeof uiBehaviorConfig.confirmDestructiveActions).toBe("boolean");
    });
  });

  describe("apiConfig", () => {
    it("has valid request timeout", () => {
      expect(apiConfig.requestTimeoutMs).toBeGreaterThanOrEqual(5000);
      expect(apiConfig.requestTimeoutMs).toBeLessThanOrEqual(120000);
    });

    it("has valid API version", () => {
      expect(apiConfig.apiVersion).toBe("v1");
    });
  });

  describe("combined config object", () => {
    it("includes all configuration sections", () => {
      expect(config.dataFetching).toBe(dataFetchingConfig);
      expect(config.displayLimits).toBe(displayLimitsConfig);
      expect(config.recommendation).toBe(recommendationConfig);
      expect(config.insights).toBe(insightsConfig);
      expect(config.uiBehavior).toBe(uiBehaviorConfig);
      expect(config.api).toBe(apiConfig);
    });
  });
});
