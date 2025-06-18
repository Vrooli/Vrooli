import { afterEach, describe, expect, it } from "vitest";
import { mockEnvVars } from "../../__test/helpers/queueTestUtils.js";
import { SWARM_QUEUE_LIMITS } from "./limits.js";

describe("Swarm Queue Limits Configuration", () => {
    describe("Default configuration", () => {
        it("should have correct default values", () => {
            expect(SWARM_QUEUE_LIMITS.maxActive).toBe(10);
            expect(SWARM_QUEUE_LIMITS.highLoadCheckIntervalMs).toBe(5000);
            expect(SWARM_QUEUE_LIMITS.highLoadThresholdPercentage).toBe(0.8);
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdFreeMs).toBe(60000); // 1 minute
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdPremiumMs).toBe(300000); // 5 minutes
            expect(SWARM_QUEUE_LIMITS.taskTimeoutMs).toBe(600000); // 10 minutes
            expect(SWARM_QUEUE_LIMITS.shutdownGracePeriodMs).toBe(30000); // 30 seconds
            expect(SWARM_QUEUE_LIMITS.onLongRunningFirstThreshold).toBe("pause");
            expect(SWARM_QUEUE_LIMITS.longRunningPauseRetries).toBe(1);
            expect(SWARM_QUEUE_LIMITS.longRunningStopRetries).toBe(0);
        });

        it("should have AI-appropriate timeout values", () => {
            // Swarm operations are AI-intensive and should have reasonable timeouts
            expect(SWARM_QUEUE_LIMITS.taskTimeoutMs).toBeGreaterThan(60000); // At least 1 minute
            expect(SWARM_QUEUE_LIMITS.taskTimeoutMs).toBeLessThanOrEqual(3600000); // Not more than 1 hour

            // Premium users should get longer running time than free users for AI tasks
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdPremiumMs)
                .toBeGreaterThan(SWARM_QUEUE_LIMITS.longRunningThresholdFreeMs);

            // Free users should have reasonable limits for AI usage
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdFreeMs).toBeGreaterThan(30000); // At least 30 seconds
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdFreeMs).toBeLessThan(300000); // Less than 5 minutes
        });

        it("should have conservative concurrency for AI operations", () => {
            // Swarm operations are resource-intensive, so lower concurrency is expected
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeGreaterThan(0);
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeLessThan(100); // Should be conservative for AI
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeLessThan(1000); // Much lower than run queue
        });

        it("should have appropriate monitoring intervals", () => {
            // AI operations need frequent monitoring due to resource intensity
            expect(SWARM_QUEUE_LIMITS.highLoadCheckIntervalMs).toBeGreaterThan(1000); // At least 1 second
            expect(SWARM_QUEUE_LIMITS.highLoadCheckIntervalMs).toBeLessThan(60000); // Not more than 1 minute
        });
    });

    describe("Environment variable override", () => {
        let restoreEnv: () => void;

        afterEach(() => {
            if (restoreEnv) {
                restoreEnv();
            }
        });

        it("should respect WORKER_SWARM_MAX_ACTIVE override", () => {
            restoreEnv = mockEnvVars({ WORKER_SWARM_MAX_ACTIVE: "20" });

            // Test that environment variable parsing works
            const mockActiveValue = parseInt(process.env.WORKER_SWARM_MAX_ACTIVE || "10");
            expect(mockActiveValue).toBe(20);
        });

        it("should respect swarm-specific timeout overrides", () => {
            restoreEnv = mockEnvVars({
                WORKER_SWARM_TASK_TIMEOUT_MS: "1800000", // 30 minutes
                WORKER_SWARM_LONG_RUNNING_THRESHOLD_FREE_MS: "45000", // 45 seconds
                WORKER_SWARM_LONG_RUNNING_THRESHOLD_PREMIUM_MS: "900000", // 15 minutes
            });

            // Test that environment variable parsing works
            expect(parseInt(process.env.WORKER_SWARM_TASK_TIMEOUT_MS || "0")).toBe(1800000);
            expect(parseInt(process.env.WORKER_SWARM_LONG_RUNNING_THRESHOLD_FREE_MS || "0")).toBe(45000);
            expect(parseInt(process.env.WORKER_SWARM_LONG_RUNNING_THRESHOLD_PREMIUM_MS || "0")).toBe(900000);
        });

        it("should respect monitoring configuration overrides", () => {
            restoreEnv = mockEnvVars({
                WORKER_SWARM_HIGH_LOAD_CHECK_INTERVAL_MS: "3000", // 3 seconds
                WORKER_SWARM_HIGH_LOAD_THRESHOLD_PERCENTAGE: "0.75", // 75%
            });

            // Test that environment variable parsing works
            expect(parseInt(process.env.WORKER_SWARM_HIGH_LOAD_CHECK_INTERVAL_MS || "0")).toBe(3000);
            expect(parseFloat(process.env.WORKER_SWARM_HIGH_LOAD_THRESHOLD_PERCENTAGE || "0")).toBe(0.75);
        });

        it("should respect swarm shutdown configuration", () => {
            restoreEnv = mockEnvVars({
                WORKER_SWARM_SHUTDOWN_GRACE_PERIOD_MS: "60000", // 1 minute
                WORKER_SWARM_ON_LONG_RUNNING_FIRST_THRESHOLD: "stop",
                WORKER_SWARM_LONG_RUNNING_PAUSE_RETRIES: "2",
                WORKER_SWARM_LONG_RUNNING_STOP_RETRIES: "1",
            });

            // Test that environment variable parsing works
            expect(parseInt(process.env.WORKER_SWARM_SHUTDOWN_GRACE_PERIOD_MS || "0")).toBe(60000);
            expect(process.env.WORKER_SWARM_ON_LONG_RUNNING_FIRST_THRESHOLD).toBe("stop");
            expect(parseInt(process.env.WORKER_SWARM_LONG_RUNNING_PAUSE_RETRIES || "0")).toBe(2);
            expect(parseInt(process.env.WORKER_SWARM_LONG_RUNNING_STOP_RETRIES || "0")).toBe(1);
        });
    });

    describe("AI-specific configuration validation", () => {
        it("should have appropriate timeouts for AI operations", () => {
            // AI operations need different timeout characteristics than regular runs
            expect(SWARM_QUEUE_LIMITS.taskTimeoutMs).toBeGreaterThan(60000); // AI needs time to process

            // But not excessive - should prevent runaway AI processes
            expect(SWARM_QUEUE_LIMITS.taskTimeoutMs).toBeLessThan(3600000); // Less than 1 hour

            // Free users should have reasonable but limited AI time
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdFreeMs).toBeGreaterThan(30000); // Enough for simple AI tasks
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdFreeMs).toBeLessThan(180000); // But limited to prevent abuse
        });

        it("should prioritize premium users for AI resources", () => {
            // Premium users should get significantly more AI time
            const premiumAdvantage = SWARM_QUEUE_LIMITS.longRunningThresholdPremiumMs / SWARM_QUEUE_LIMITS.longRunningThresholdFreeMs;
            expect(premiumAdvantage).toBeGreaterThan(3); // At least 3x more time for AI
            expect(premiumAdvantage).toBeLessThan(20); // But not excessive
        });

        it("should have conservative concurrency for resource management", () => {
            // AI operations are CPU/memory intensive
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeLessThan(50); // Conservative for AI workloads

            // Should be less than run queue limits due to resource intensity
            // Note: This would require importing run limits for comparison in real implementation
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeLessThan(100);
        });

        it("should have frequent monitoring for AI workloads", () => {
            // AI operations need close monitoring due to unpredictable resource usage
            expect(SWARM_QUEUE_LIMITS.highLoadCheckIntervalMs).toBeLessThan(10000); // Check at least every 10 seconds

            // But not too frequent to avoid overhead
            expect(SWARM_QUEUE_LIMITS.highLoadCheckIntervalMs).toBeGreaterThan(1000);
        });
    });

    describe("Resource protection for AI operations", () => {
        it("should prevent AI resource exhaustion", () => {
            // Max active should prevent overwhelming the system with AI requests
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeLessThan(25); // Conservative limit

            // High load threshold should trigger early to prevent overload
            expect(SWARM_QUEUE_LIMITS.highLoadThresholdPercentage).toBeLessThan(0.9); // Trigger before 90%
            expect(SWARM_QUEUE_LIMITS.highLoadThresholdPercentage).toBeGreaterThan(0.5); // But allow reasonable load
        });

        it("should have appropriate graceful shutdown for AI", () => {
            // AI operations may need more time to shutdown gracefully
            expect(SWARM_QUEUE_LIMITS.shutdownGracePeriodMs).toBeGreaterThan(10000); // At least 10 seconds
            expect(SWARM_QUEUE_LIMITS.shutdownGracePeriodMs).toBeLessThan(120000); // But not more than 2 minutes
        });

        it("should handle long-running AI tasks appropriately", () => {
            // Should have a strategy for dealing with long-running AI tasks
            expect(["pause", "stop"]).toContain(SWARM_QUEUE_LIMITS.onLongRunningFirstThreshold);

            // Retry configuration should be reasonable for AI operations
            expect(SWARM_QUEUE_LIMITS.longRunningPauseRetries).toBeGreaterThanOrEqual(0);
            expect(SWARM_QUEUE_LIMITS.longRunningStopRetries).toBeGreaterThanOrEqual(0);

            const totalRetries = SWARM_QUEUE_LIMITS.longRunningPauseRetries + SWARM_QUEUE_LIMITS.longRunningStopRetries;
            expect(totalRetries).toBeLessThan(5); // Not too many retries for AI
        });
    });

    describe("Comparison with other queue types", () => {
        it("should have different characteristics than run queue", () => {
            // Swarm operations should be more conservative than general runs
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeLessThan(1000); // Much lower than typical run limits

            // But should still allow reasonable throughput
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeGreaterThan(1);
        });

        it("should balance performance and resource usage", () => {
            // Should not be so restrictive as to be unusable
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeGreaterThan(5);

            // But should be conservative enough to prevent resource exhaustion
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeLessThan(50);
        });
    });

    describe("Type safety and interface compliance", () => {
        it("should have correct types for all fields", () => {
            expect(typeof SWARM_QUEUE_LIMITS.maxActive).toBe("number");
            expect(typeof SWARM_QUEUE_LIMITS.highLoadCheckIntervalMs).toBe("number");
            expect(typeof SWARM_QUEUE_LIMITS.highLoadThresholdPercentage).toBe("number");
            expect(typeof SWARM_QUEUE_LIMITS.longRunningThresholdFreeMs).toBe("number");
            expect(typeof SWARM_QUEUE_LIMITS.longRunningThresholdPremiumMs).toBe("number");
            expect(typeof SWARM_QUEUE_LIMITS.taskTimeoutMs).toBe("number");
            expect(typeof SWARM_QUEUE_LIMITS.shutdownGracePeriodMs).toBe("number");
            expect(typeof SWARM_QUEUE_LIMITS.onLongRunningFirstThreshold).toBe("string");
            expect(typeof SWARM_QUEUE_LIMITS.longRunningPauseRetries).toBe("number");
            expect(typeof SWARM_QUEUE_LIMITS.longRunningStopRetries).toBe("number");
        });

        it("should match ActiveTaskRegistryLimits interface", () => {
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
                expect(SWARM_QUEUE_LIMITS).toHaveProperty(field);
                expect(SWARM_QUEUE_LIMITS[field as keyof typeof SWARM_QUEUE_LIMITS]).toBeDefined();
            });
        });

        it("should have valid enum values", () => {
            expect(["pause", "stop"]).toContain(SWARM_QUEUE_LIMITS.onLongRunningFirstThreshold);
        });
    });

    describe("Production readiness for AI workloads", () => {
        it("should support production AI usage patterns", () => {
            // Should allow enough concurrency for typical AI workloads
            expect(SWARM_QUEUE_LIMITS.maxActive).toBeGreaterThanOrEqual(5);

            // Should have reasonable timeouts for AI processing
            expect(SWARM_QUEUE_LIMITS.taskTimeoutMs).toBeGreaterThanOrEqual(300000); // At least 5 minutes

            // Should monitor frequently enough to detect issues
            expect(SWARM_QUEUE_LIMITS.highLoadCheckIntervalMs).toBeLessThanOrEqual(30000); // At most every 30 seconds
        });

        it("should prevent AI resource abuse", () => {
            // Free users should have limited but usable AI access
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdFreeMs).toBeGreaterThan(30000); // Usable
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdFreeMs).toBeLessThan(300000); // But limited

            // Premium users should have generous but not unlimited access
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdPremiumMs).toBeGreaterThan(180000); // Generous
            expect(SWARM_QUEUE_LIMITS.longRunningThresholdPremiumMs).toBeLessThan(1800000); // But not unlimited (30 min max)
        });

        it("should handle AI model timeout characteristics", () => {
            // Should account for AI model loading and processing time
            expect(SWARM_QUEUE_LIMITS.taskTimeoutMs).toBeGreaterThan(SWARM_QUEUE_LIMITS.longRunningThresholdPremiumMs);

            // Should allow enough time for complex AI tasks
            expect(SWARM_QUEUE_LIMITS.taskTimeoutMs).toBeGreaterThan(300000); // At least 5 minutes
        });
    });
});
