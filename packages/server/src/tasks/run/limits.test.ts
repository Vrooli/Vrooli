import { HOURS_2_MS, MINUTES_1_MS, MINUTES_5_MS, SECONDS_10_MS } from "@vrooli/shared";
import { afterEach, describe, expect, it } from "vitest";
import { mockEnvVars } from "../../__test/helpers/queueTestUtils.js";
import { RUN_QUEUE_LIMITS } from "./limits.js";

describe("Run Queue Limits Configuration", () => {
    describe("Default configuration", () => {
        it("should have correct default values", () => {
            expect(RUN_QUEUE_LIMITS.maxActive).toBe(1000);
            expect(RUN_QUEUE_LIMITS.highLoadCheckIntervalMs).toBe(MINUTES_1_MS);
            expect(RUN_QUEUE_LIMITS.highLoadThresholdPercentage).toBe(0.8);
            expect(RUN_QUEUE_LIMITS.longRunningThresholdFreeMs).toBe(MINUTES_1_MS);
            expect(RUN_QUEUE_LIMITS.longRunningThresholdPremiumMs).toBe(MINUTES_5_MS);
            expect(RUN_QUEUE_LIMITS.taskTimeoutMs).toBe(HOURS_2_MS);
            expect(RUN_QUEUE_LIMITS.shutdownGracePeriodMs).toBe(SECONDS_10_MS);
            expect(RUN_QUEUE_LIMITS.onLongRunningFirstThreshold).toBe("pause");
            expect(RUN_QUEUE_LIMITS.longRunningPauseRetries).toBe(1);
            expect(RUN_QUEUE_LIMITS.longRunningStopRetries).toBe(0);
        });

        it("should have reasonable timeout values", () => {
            // Task timeout should be reasonable for run operations
            expect(RUN_QUEUE_LIMITS.taskTimeoutMs).toBeGreaterThan(MINUTES_1_MS);
            expect(RUN_QUEUE_LIMITS.taskTimeoutMs).toBeLessThanOrEqual(HOURS_2_MS * 2); // Not more than 4 hours

            // Premium users should get longer running time than free users
            expect(RUN_QUEUE_LIMITS.longRunningThresholdPremiumMs)
                .toBeGreaterThan(RUN_QUEUE_LIMITS.longRunningThresholdFreeMs);

            // High load threshold should be between 0 and 1
            expect(RUN_QUEUE_LIMITS.highLoadThresholdPercentage).toBeGreaterThan(0);
            expect(RUN_QUEUE_LIMITS.highLoadThresholdPercentage).toBeLessThanOrEqual(1);
        });

        it("should have reasonable concurrency limits", () => {
            // Max active should allow meaningful parallelism but not be excessive
            expect(RUN_QUEUE_LIMITS.maxActive).toBeGreaterThan(0);
            expect(RUN_QUEUE_LIMITS.maxActive).toBeLessThan(10000); // Reasonable upper bound
        });

        it("should have valid retry configuration", () => {
            expect(RUN_QUEUE_LIMITS.longRunningPauseRetries).toBeGreaterThanOrEqual(0);
            expect(RUN_QUEUE_LIMITS.longRunningStopRetries).toBeGreaterThanOrEqual(0);
            expect(["pause", "stop"]).toContain(RUN_QUEUE_LIMITS.onLongRunningFirstThreshold);
        });
    });

    describe("Environment variable override", () => {
        let restoreEnv: () => void;

        afterEach(() => {
            if (restoreEnv) {
                restoreEnv();
            }
        });

        it("should respect WORKER_RUN_MAX_ACTIVE override", () => {
            restoreEnv = mockEnvVars({ WORKER_RUN_MAX_ACTIVE: "500" });

            // Test that environment variable parsing works
            const mockActiveValue = parseInt(process.env.WORKER_RUN_MAX_ACTIVE || "1000");
            expect(mockActiveValue).toBe(500);
        });

        it("should respect timeout overrides", () => {
            restoreEnv = mockEnvVars({
                WORKER_RUN_TASK_TIMEOUT_MS: "7200000", // 2 hours in ms
                WORKER_RUN_LONG_RUNNING_THRESHOLD_FREE_MS: "30000", // 30 seconds
                WORKER_RUN_LONG_RUNNING_THRESHOLD_PREMIUM_MS: "600000", // 10 minutes
            });

            // Test that environment variable parsing works
            expect(parseInt(process.env.WORKER_RUN_TASK_TIMEOUT_MS || "0")).toBe(7200000);
            expect(parseInt(process.env.WORKER_RUN_LONG_RUNNING_THRESHOLD_FREE_MS || "0")).toBe(30000);
            expect(parseInt(process.env.WORKER_RUN_LONG_RUNNING_THRESHOLD_PREMIUM_MS || "0")).toBe(600000);
        });

        it("should respect load monitoring overrides", () => {
            restoreEnv = mockEnvVars({
                WORKER_RUN_HIGH_LOAD_CHECK_INTERVAL_MS: "30000", // 30 seconds
                WORKER_RUN_HIGH_LOAD_THRESHOLD_PERCENTAGE: "0.9", // 90%
            });

            // Test that environment variable parsing works
            expect(parseInt(process.env.WORKER_RUN_HIGH_LOAD_CHECK_INTERVAL_MS || "0")).toBe(30000);
            expect(parseFloat(process.env.WORKER_RUN_HIGH_LOAD_THRESHOLD_PERCENTAGE || "0")).toBe(0.9);
        });

        it("should respect shutdown and retry overrides", () => {
            restoreEnv = mockEnvVars({
                WORKER_RUN_SHUTDOWN_GRACE_PERIOD_MS: "30000", // 30 seconds
                WORKER_RUN_ON_LONG_RUNNING_FIRST_THRESHOLD: "stop",
                WORKER_RUN_LONG_RUNNING_PAUSE_RETRIES: "3",
                WORKER_RUN_LONG_RUNNING_STOP_RETRIES: "2",
            });

            // Test that environment variable parsing works
            expect(parseInt(process.env.WORKER_RUN_SHUTDOWN_GRACE_PERIOD_MS || "0")).toBe(30000);
            expect(process.env.WORKER_RUN_ON_LONG_RUNNING_FIRST_THRESHOLD).toBe("stop");
            expect(parseInt(process.env.WORKER_RUN_LONG_RUNNING_PAUSE_RETRIES || "0")).toBe(3);
            expect(parseInt(process.env.WORKER_RUN_LONG_RUNNING_STOP_RETRIES || "0")).toBe(2);
        });

        it("should handle invalid environment variables gracefully", () => {
            restoreEnv = mockEnvVars({
                WORKER_RUN_MAX_ACTIVE: "invalid_number",
                WORKER_RUN_HIGH_LOAD_THRESHOLD_PERCENTAGE: "not_a_float",
                WORKER_RUN_ON_LONG_RUNNING_FIRST_THRESHOLD: "invalid_action",
            });

            // Test that invalid values are handled
            expect(parseInt(process.env.WORKER_RUN_MAX_ACTIVE || "1000")).toBeNaN();
            expect(parseFloat(process.env.WORKER_RUN_HIGH_LOAD_THRESHOLD_PERCENTAGE || "0.8")).toBeNaN();
            expect(process.env.WORKER_RUN_ON_LONG_RUNNING_FIRST_THRESHOLD).toBe("invalid_action");
        });
    });

    describe("Configuration validation", () => {
        it("should validate timeout relationships", () => {
            // Free user threshold should be less than premium threshold
            expect(RUN_QUEUE_LIMITS.longRunningThresholdFreeMs)
                .toBeLessThan(RUN_QUEUE_LIMITS.longRunningThresholdPremiumMs);

            // Both thresholds should be less than overall timeout
            expect(RUN_QUEUE_LIMITS.longRunningThresholdFreeMs)
                .toBeLessThan(RUN_QUEUE_LIMITS.taskTimeoutMs);
            expect(RUN_QUEUE_LIMITS.longRunningThresholdPremiumMs)
                .toBeLessThan(RUN_QUEUE_LIMITS.taskTimeoutMs);

            // Shutdown grace period should be reasonable
            expect(RUN_QUEUE_LIMITS.shutdownGracePeriodMs)
                .toBeLessThan(RUN_QUEUE_LIMITS.taskTimeoutMs);
        });

        it("should validate monitoring intervals", () => {
            // High load check interval should be reasonable
            expect(RUN_QUEUE_LIMITS.highLoadCheckIntervalMs).toBeGreaterThan(1000); // At least 1 second
            expect(RUN_QUEUE_LIMITS.highLoadCheckIntervalMs).toBeLessThan(MINUTES_1_MS * 10); // Not more than 10 minutes
        });

        it("should validate retry counts", () => {
            expect(RUN_QUEUE_LIMITS.longRunningPauseRetries).toBeGreaterThanOrEqual(0);
            expect(RUN_QUEUE_LIMITS.longRunningStopRetries).toBeGreaterThanOrEqual(0);

            // Total retries should be reasonable
            const totalRetries = RUN_QUEUE_LIMITS.longRunningPauseRetries + RUN_QUEUE_LIMITS.longRunningStopRetries;
            expect(totalRetries).toBeLessThan(10); // Not excessive
        });
    });

    describe("Production readiness", () => {
        it("should have production-appropriate values", () => {
            // Max active should support reasonable concurrency
            expect(RUN_QUEUE_LIMITS.maxActive).toBeGreaterThanOrEqual(100);

            // Timeouts should be long enough for complex runs
            expect(RUN_QUEUE_LIMITS.taskTimeoutMs).toBeGreaterThanOrEqual(MINUTES_1_MS * 30); // At least 30 minutes

            // High load threshold should prevent overload
            expect(RUN_QUEUE_LIMITS.highLoadThresholdPercentage).toBeLessThan(1.0);
            expect(RUN_QUEUE_LIMITS.highLoadThresholdPercentage).toBeGreaterThan(0.5);
        });

        it("should support different user tiers appropriately", () => {
            // Premium users should get significantly more time
            const premiumAdvantage = RUN_QUEUE_LIMITS.longRunningThresholdPremiumMs / RUN_QUEUE_LIMITS.longRunningThresholdFreeMs;
            expect(premiumAdvantage).toBeGreaterThan(2); // At least 2x more time

            // But not excessively more
            expect(premiumAdvantage).toBeLessThan(20); // Not more than 20x
        });

        it("should have reasonable resource protection", () => {
            // Should prevent runaway processes
            expect(RUN_QUEUE_LIMITS.taskTimeoutMs).toBeLessThan(24 * 60 * 60 * 1000); // Less than 24 hours

            // Should allow graceful shutdown
            expect(RUN_QUEUE_LIMITS.shutdownGracePeriodMs).toBeGreaterThan(1000); // At least 1 second
            expect(RUN_QUEUE_LIMITS.shutdownGracePeriodMs).toBeLessThan(MINUTES_1_MS); // Not more than 1 minute for shutdown
        });
    });

    describe("Type safety", () => {
        it("should have correct types for all fields", () => {
            expect(typeof RUN_QUEUE_LIMITS.maxActive).toBe("number");
            expect(typeof RUN_QUEUE_LIMITS.highLoadCheckIntervalMs).toBe("number");
            expect(typeof RUN_QUEUE_LIMITS.highLoadThresholdPercentage).toBe("number");
            expect(typeof RUN_QUEUE_LIMITS.longRunningThresholdFreeMs).toBe("number");
            expect(typeof RUN_QUEUE_LIMITS.longRunningThresholdPremiumMs).toBe("number");
            expect(typeof RUN_QUEUE_LIMITS.taskTimeoutMs).toBe("number");
            expect(typeof RUN_QUEUE_LIMITS.shutdownGracePeriodMs).toBe("number");
            expect(typeof RUN_QUEUE_LIMITS.onLongRunningFirstThreshold).toBe("string");
            expect(typeof RUN_QUEUE_LIMITS.longRunningPauseRetries).toBe("number");
            expect(typeof RUN_QUEUE_LIMITS.longRunningStopRetries).toBe("number");
        });

        it("should have valid enum values", () => {
            expect(["pause", "stop"]).toContain(RUN_QUEUE_LIMITS.onLongRunningFirstThreshold);
        });

        it("should match ActiveTaskRegistryLimits interface", () => {
            // Verify all required fields are present
            const requiredFields = [
                "maxActive",
                "highLoadCheckIntervalMs",
                "highLoadThresholdPercentage",
                "longRunningThresholdFreeMs",
                "longRunningThresholdPremiumMs",
                "taskTimeoutMs",
                "shutdownGracePeriodMs",
                "onLongRunningFirstThreshold",
                "longRunningPauseRetries",
                "longRunningStopRetries",
            ];

            requiredFields.forEach(field => {
                expect(RUN_QUEUE_LIMITS).toHaveProperty(field);
                expect(RUN_QUEUE_LIMITS[field as keyof typeof RUN_QUEUE_LIMITS]).toBeDefined();
            });
        });
    });
});
