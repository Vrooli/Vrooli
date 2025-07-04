/**
 * Comprehensive tests for resource aggregation and tracking utilities
 * 
 * These tests ensure proper resource tracking, hierarchical allocation,
 * and BigInt arithmetic for credit calculations.
 */

import { beforeEach, describe, expect, it } from "vitest";
import type {
    CoreResourceAllocation,
    ExecutionResourceUsage,
} from "./core.js";
import {
    ResourceAggregator,
    ResourceTracker,
} from "./resourceAggregation.js";

describe("ResourceAggregator", () => {
    function createUsage(overrides: Partial<ExecutionResourceUsage> = {}): ExecutionResourceUsage {
        return {
            creditsUsed: "100",
            durationMs: 1000,
            memoryUsedMB: 256,
            stepsExecuted: 5,
            toolCalls: 2,
            ...overrides,
        };
    }

    function createAllocation(overrides: Partial<CoreResourceAllocation> = {}): CoreResourceAllocation {
        return {
            maxCredits: "10000",
            maxDurationMs: 60000,
            maxMemoryMB: 2048,
            maxConcurrentSteps: 10,
            ...overrides,
        };
    }

    describe("aggregateUsage", () => {
        it("should aggregate parallel execution correctly", () => {
            const childUsages = [
                createUsage({ creditsUsed: "100", durationMs: 1000, memoryUsedMB: 256 }),
                createUsage({ creditsUsed: "200", durationMs: 1500, memoryUsedMB: 512 }),
                createUsage({ creditsUsed: "150", durationMs: 800, memoryUsedMB: 128 }),
            ];

            const result = ResourceAggregator.aggregateUsage(childUsages, "parallel");

            expect(result.creditsUsed).toBe("450"); // Sum of all credits
            expect(result.durationMs).toBe(1500); // Max duration for parallel
            expect(result.memoryUsedMB).toBe(512); // Peak memory usage
            expect(result.stepsExecuted).toBe(15); // Sum of all steps
            expect(result.toolCalls).toBe(6); // Sum of all API calls
        });

        it("should aggregate sequential execution correctly", () => {
            const childUsages = [
                createUsage({ creditsUsed: "100", durationMs: 1000, memoryUsedMB: 256 }),
                createUsage({ creditsUsed: "200", durationMs: 1500, memoryUsedMB: 512 }),
                createUsage({ creditsUsed: "150", durationMs: 800, memoryUsedMB: 128 }),
            ];

            const result = ResourceAggregator.aggregateUsage(childUsages, "sequential");

            expect(result.creditsUsed).toBe("450"); // Sum of all credits
            expect(result.durationMs).toBe(3300); // Sum of durations for sequential
            expect(result.memoryUsedMB).toBe(512); // Peak memory usage
            expect(result.stepsExecuted).toBe(15); // Sum of all steps
        });

        it("should handle empty child usages", () => {
            const result = ResourceAggregator.aggregateUsage([]);

            expect(result.creditsUsed).toBe("0");
            expect(result.durationMs).toBe(0);
            expect(result.memoryUsedMB).toBe(0);
            expect(result.stepsExecuted).toBe(0);
            expect(result.toolCalls).toBe(0);
        });

        it("should handle single child usage", () => {
            const singleUsage = createUsage();
            const result = ResourceAggregator.aggregateUsage([singleUsage]);

            expect(result).toEqual(singleUsage);
        });

        it("should handle BigInt credit calculations", () => {
            const largeCredits = [
                createUsage({ creditsUsed: "999999999999999999" }),
                createUsage({ creditsUsed: "888888888888888888" }),
            ];

            const result = ResourceAggregator.aggregateUsage(largeCredits);
            expect(result.creditsUsed).toBe("1888888888888888887");
        });
    });

    describe("createDetailedUsage", () => {
        it("should create detailed usage breakdown", () => {
            const ownUsage = createUsage({ creditsUsed: "50", durationMs: 500 });
            const childUsages = [
                { executionId: "child-1", usage: createUsage({ creditsUsed: "100", durationMs: 1000 }) },
                { executionId: "child-2", usage: createUsage({ creditsUsed: "200", durationMs: 1500 }) },
            ];

            const detailed = ResourceAggregator.createDetailedUsage(ownUsage, childUsages);

            expect(detailed.creditsUsed).toBe("350"); // Own + aggregated child
            expect(detailed.childUsages).toEqual(childUsages);
            expect(detailed.phaseBreakdown).toBeDefined();
            expect(detailed.phaseBreakdown?.execution.creditsUsed).toBe("350");
        });

        it("should handle no child usages", () => {
            const ownUsage = createUsage();
            const detailed = ResourceAggregator.createDetailedUsage(ownUsage, []);

            expect(detailed.creditsUsed).toBe(ownUsage.creditsUsed);
            expect(detailed.childUsages).toEqual([]);
        });
    });

    describe("wouldExceedAllocation", () => {
        const allocation = createAllocation();

        it("should detect credit limit exceeded", () => {
            const currentUsage = createUsage({ creditsUsed: "9000" });
            const additionalUsage = createUsage({ creditsUsed: "2000" });

            const result = ResourceAggregator.wouldExceedAllocation(
                currentUsage,
                additionalUsage,
                allocation,
            );

            expect(result.wouldExceed).toBe(true);
            expect(result.exceededResources).toContain("credits");
            expect(result.projectedUsage.creditsUsed).toBe("11000");
        });

        it("should detect duration limit exceeded", () => {
            const currentUsage = createUsage({ durationMs: 50000 });
            const additionalUsage = createUsage({ durationMs: 20000 });

            const result = ResourceAggregator.wouldExceedAllocation(
                currentUsage,
                additionalUsage,
                allocation,
            );

            expect(result.wouldExceed).toBe(true);
            expect(result.exceededResources).toContain("duration");
        });

        it("should detect memory limit exceeded", () => {
            const currentUsage = createUsage({ memoryUsedMB: 1800 });
            const additionalUsage = createUsage({ memoryUsedMB: 2200 }); // Peak memory exceeds limit

            const result = ResourceAggregator.wouldExceedAllocation(
                currentUsage,
                additionalUsage,
                allocation,
            );

            expect(result.wouldExceed).toBe(true);
            expect(result.exceededResources).toContain("memory");
        });

        it("should detect multiple limits exceeded", () => {
            const currentUsage = createUsage({ creditsUsed: "9500", durationMs: 55000 });
            const additionalUsage = createUsage({ creditsUsed: "1000", durationMs: 10000 });

            const result = ResourceAggregator.wouldExceedAllocation(
                currentUsage,
                additionalUsage,
                allocation,
            );

            expect(result.wouldExceed).toBe(true);
            expect(result.exceededResources).toContain("credits");
            expect(result.exceededResources).toContain("duration");
        });

        it("should pass when within limits", () => {
            const currentUsage = createUsage({ creditsUsed: "1000", durationMs: 10000 });
            const additionalUsage = createUsage({ creditsUsed: "2000", durationMs: 20000 });

            const result = ResourceAggregator.wouldExceedAllocation(
                currentUsage,
                additionalUsage,
                allocation,
            );

            expect(result.wouldExceed).toBe(false);
            expect(result.exceededResources).toEqual([]);
        });
    });

    describe("calculateRemaining", () => {
        it("should calculate remaining resources correctly", () => {
            const allocation = createAllocation({
                maxCredits: "5000",
                maxDurationMs: 30000,
                maxMemoryMB: 1024,
            });
            const usage = createUsage({
                creditsUsed: "2000",
                durationMs: 10000,
                memoryUsedMB: 512,
            });

            const remaining = ResourceAggregator.calculateRemaining(allocation, usage);

            expect(remaining.maxCredits).toBe("3000");
            expect(remaining.maxDurationMs).toBe(20000);
            expect(remaining.maxMemoryMB).toBe(512);
            expect(remaining.maxConcurrentSteps).toBe(allocation.maxConcurrentSteps);
        });

        it("should not go below zero", () => {
            const allocation = createAllocation({ maxCredits: "1000", maxDurationMs: 5000 });
            const usage = createUsage({ creditsUsed: "2000", durationMs: 10000 });

            const remaining = ResourceAggregator.calculateRemaining(allocation, usage);

            expect(remaining.maxCredits).toBe("0");
            expect(remaining.maxDurationMs).toBe(0);
        });
    });

    describe("validateAllocationHierarchy", () => {
        const parentAllocation = createAllocation({
            maxCredits: "10000",
            maxDurationMs: 60000,
            maxMemoryMB: 2048,
            maxConcurrentSteps: 10,
        });

        it("should validate correct child allocation", () => {
            const childAllocation = createAllocation({
                maxCredits: "5000",
                maxDurationMs: 30000,
                maxMemoryMB: 1024,
                maxConcurrentSteps: 5,
            });

            const result = ResourceAggregator.validateAllocationHierarchy(
                childAllocation,
                parentAllocation,
            );

            expect(result.isValid).toBe(true);
            expect(result.violations).toEqual([]);
        });

        it("should detect credit violations", () => {
            const childAllocation = createAllocation({ maxCredits: "15000" });

            const result = ResourceAggregator.validateAllocationHierarchy(
                childAllocation,
                parentAllocation,
            );

            expect(result.isValid).toBe(false);
            expect(result.violations).toHaveLength(1);
            expect(result.violations[0]).toContain("credits");
        });

        it("should detect multiple violations", () => {
            const childAllocation = createAllocation({
                maxCredits: "15000",
                maxDurationMs: 90000,
                maxMemoryMB: 4096,
            });

            const result = ResourceAggregator.validateAllocationHierarchy(
                childAllocation,
                parentAllocation,
            );

            expect(result.isValid).toBe(false);
            expect(result.violations.length).toBeGreaterThan(1);
        });
    });

    describe("createHierarchicalAllocation", () => {
        it("should create hierarchical allocation with parent tracking", () => {
            const allocation = createAllocation();
            const parentAllocation = createAllocation({ maxCredits: "20000" });

            const hierarchical = ResourceAggregator.createHierarchicalAllocation(
                allocation,
                "parent-exec-123",
                parentAllocation,
                "elastic",
            );

            expect(hierarchical.strategy).toBe("elastic");
            expect(hierarchical.allocatedAt).toBeInstanceOf(Date);
            expect(hierarchical.parentAllocation).toBeDefined();
            expect(hierarchical.parentAllocation?.executionId).toBe("parent-exec-123");
            expect(hierarchical.parentAllocation?.allocatedToChild).toEqual(allocation);
        });

        it("should create allocation without parent", () => {
            const allocation = createAllocation();

            const hierarchical = ResourceAggregator.createHierarchicalAllocation(allocation);

            expect(hierarchical.strategy).toBe("strict");
            expect(hierarchical.parentAllocation).toBeUndefined();
        });
    });
});

describe("ResourceTracker", () => {
    let tracker: ResourceTracker;
    const allocation = {
        maxCredits: "10000",
        maxDurationMs: 60000,
        maxMemoryMB: 2048,
        maxConcurrentSteps: 10,
    };

    beforeEach(() => {
        tracker = new ResourceTracker(allocation);
    });

    describe("constructor", () => {
        it("should initialize with empty usage", () => {
            const usage = tracker.getCurrentUsage();

            expect(usage.creditsUsed).toBe("0");
            expect(usage.memoryUsedMB).toBe(0);
            expect(usage.stepsExecuted).toBe(0);
            expect(usage.toolCalls).toBe(0);
        });

        it("should track real-time duration", () => {
            const usage1 = tracker.getCurrentUsage();

            // Wait a bit
            return new Promise(resolve => {
                setTimeout(() => {
                    const usage2 = tracker.getCurrentUsage();
                    expect(usage2.durationMs).toBeGreaterThan(usage1.durationMs);
                    resolve(undefined);
                }, 10);
            });
        });
    });

    describe("addUsage", () => {
        it("should add usage successfully within limits", () => {
            const result = tracker.addUsage({
                creditsUsed: "1000",
                durationMs: 5000,
                memoryUsedMB: 256,
                stepsExecuted: 3,
                toolCalls: 5,
            });

            expect(result.success).toBe(true);
            expect(result.error).toBeUndefined();

            const usage = tracker.getCurrentUsage();
            expect(usage.creditsUsed).toBe("1000");
            expect(usage.memoryUsedMB).toBe(256);
            expect(usage.stepsExecuted).toBe(3);
        });

        it("should accumulate multiple usage additions", () => {
            tracker.addUsage({ creditsUsed: "1000", stepsExecuted: 2 });
            tracker.addUsage({ creditsUsed: "2000", stepsExecuted: 3 });

            const usage = tracker.getCurrentUsage();
            expect(usage.creditsUsed).toBe("3000");
            expect(usage.stepsExecuted).toBe(5);
        });

        it("should track peak memory usage", () => {
            tracker.addUsage({ memoryUsedMB: 256 });
            tracker.addUsage({ memoryUsedMB: 128 }); // Lower usage

            const usage = tracker.getCurrentUsage();
            expect(usage.memoryUsedMB).toBe(256); // Should keep peak
        });

        it("should reject usage that exceeds credit limits", () => {
            const result = tracker.addUsage({ creditsUsed: "15000" }); // Exceeds 10000 limit

            expect(result.success).toBe(false);
            expect(result.error).toContain("Resource limit exceeded");
            expect(result.error).toContain("credits");
        });

        it("should reject usage that exceeds memory limits", () => {
            const result = tracker.addUsage({ memoryUsedMB: 3000 }); // Exceeds 2048 limit

            expect(result.success).toBe(false);
            expect(result.error).toContain("memory");
        });

        it("should handle partial usage objects", () => {
            const result = tracker.addUsage({ creditsUsed: "500" }); // Only credits

            expect(result.success).toBe(true);

            const usage = tracker.getCurrentUsage();
            expect(usage.creditsUsed).toBe("500");
            expect(usage.stepsExecuted).toBe(0); // Should remain unchanged
        });
    });

    describe("getRemainingAllocation", () => {
        it("should calculate remaining allocation correctly", () => {
            tracker.addUsage({
                creditsUsed: "3000",
                memoryUsedMB: 512,
                stepsExecuted: 5,
            });

            const remaining = tracker.getRemainingAllocation();

            expect(remaining.maxCredits).toBe("7000");
            expect(remaining.maxMemoryMB).toBe(1536); // 2048 - 512
            expect(remaining.maxConcurrentSteps).toBe(10); // Unchanged
        });

        it("should not go below zero for remaining resources", () => {
            // First add usage up to the limit
            tracker.addUsage({ creditsUsed: "9000" });

            // Then add more to go over
            tracker.addUsage({ creditsUsed: "2000" }); // This would exceed, so it's rejected

            // The remaining should be based on actual usage (9000), not attempted usage
            const remaining = tracker.getRemainingAllocation();
            expect(remaining.maxCredits).toBe("1000"); // 10000 - 9000
        });
    });

    describe("canContinue", () => {
        it("should return true when no additional usage provided", () => {
            expect(tracker.canContinue()).toBe(true);
        });

        it("should return true when additional usage is within limits", () => {
            tracker.addUsage({ creditsUsed: "2000" });

            const canContinue = tracker.canContinue({
                creditsUsed: "5000", // Total would be 7000, within 10000 limit
            });

            expect(canContinue).toBe(true);
        });

        it("should return false when additional usage would exceed limits", () => {
            tracker.addUsage({ creditsUsed: "8000" });

            const canContinue = tracker.canContinue({
                creditsUsed: "5000", // Total would be 13000, exceeding 10000 limit
            });

            expect(canContinue).toBe(false);
        });

        it("should consider all resource types", () => {
            const canContinue = tracker.canContinue({
                creditsUsed: "15000", // Exceeds credit limit
                memoryUsedMB: 1000,   // Within memory limit
            });

            expect(canContinue).toBe(false);
        });
    });

    describe("BigInt edge cases", () => {
        it("should handle very large credit values", () => {
            const largeTracker = new ResourceTracker({
                maxCredits: "999999999999999999999",
                maxDurationMs: 60000,
                maxMemoryMB: 2048,
                maxConcurrentSteps: 10,
            });

            const result = largeTracker.addUsage({
                creditsUsed: "888888888888888888888",
            });

            expect(result.success).toBe(true);

            const usage = largeTracker.getCurrentUsage();
            expect(usage.creditsUsed).toBe("888888888888888888888");
        });

        it("should handle unlimited credits", () => {
            const unlimitedTracker = new ResourceTracker({
                maxCredits: "unlimited",
                maxDurationMs: 60000,
                maxMemoryMB: 2048,
                maxConcurrentSteps: 10,
            });

            // This should not work with current implementation as BigInt("unlimited") would throw
            // This test documents the limitation
            expect(() => {
                unlimitedTracker.addUsage({ creditsUsed: "999999999999999999" });
            }).toThrow();
        });
    });

    describe("performance", () => {
        it("should handle frequent usage updates efficiently", () => {
            const startTime = Date.now();

            for (let i = 0; i < 1000; i++) {
                tracker.addUsage({
                    creditsUsed: "1",
                    memoryUsedMB: 1,
                    stepsExecuted: 1,
                });
            }

            const duration = Date.now() - startTime;
            expect(duration).toBeLessThan(100); // Should complete within 100ms

            const finalUsage = tracker.getCurrentUsage();
            expect(finalUsage.creditsUsed).toBe("1000");
            expect(finalUsage.stepsExecuted).toBe(1000);
        });
    });
});

