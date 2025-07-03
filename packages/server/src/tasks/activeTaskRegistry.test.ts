import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { checkLongRunningTasksInRegistry } from "./activeTaskRegistry.js";
import { logger } from "../events/logger.js";

// Mock registry for testing
interface MockActiveTaskRecord {
    hasPremium: boolean;
    startTime: number;
    id: string;
    userId?: string;
}

interface MockRegistry {
    count(): number;
    getOrderedActiveRecords(): MockActiveTaskRecord[];
    remove(id: string): void;
}

// Queue limits for testing
const mockLimits = {
    maxActive: 5,
    maxActiveFree: 2,
    maxActivePremium: 10,
    taskTimeoutMs: 300000, // 5 minutes
    highLoadCheckIntervalMs: 30000,
};

describe("activeTaskRegistry", () => {
    let mockRegistry: MockRegistry;
    let loggerWarnSpy: vi.Mock;
    let loggerInfoSpy: vi.Mock;

    beforeEach(() => {
        // Setup mock registry
        const records: MockActiveTaskRecord[] = [];
        mockRegistry = {
            count: () => records.length,
            getOrderedActiveRecords: () => [...records].sort((a, b) => a.startTime - b.startTime),
            remove: (id: string) => {
                const index = records.findIndex(r => r.id === id);
                if (index !== -1) {
                    records.splice(index, 1);
                }
            },
        };

        // Add test records
        const addRecord = (record: MockActiveTaskRecord) => {
            records.push(record);
        };

        // Store for test access
        (mockRegistry as any).addRecord = addRecord;
        (mockRegistry as any).records = records;

        // Setup logger spies
        loggerWarnSpy = vi.spyOn(logger, "warn").mockImplementation(() => {});
        loggerInfoSpy = vi.spyOn(logger, "info").mockImplementation(() => {});
    });

    afterEach(() => {
        loggerWarnSpy.mockRestore();
        loggerInfoSpy.mockRestore();
    });

    describe("checkLongRunningTasksInRegistry", () => {
        it("should not take action when under limits", async () => {
            // Add tasks within limits
            (mockRegistry as any).addRecord({
                id: "task1",
                userId: "user1",
                hasPremium: false,
                startTime: Date.now() - 60000, // 1 minute ago
            });

            (mockRegistry as any).addRecord({
                id: "task2", 
                userId: "user2",
                hasPremium: true,
                startTime: Date.now() - 120000, // 2 minutes ago
            });

            await checkLongRunningTasksInRegistry(
                mockRegistry as any,
                mockLimits,
                "Test",
            );

            // Should not log warnings or remove tasks
            expect(loggerWarnSpy).not.toHaveBeenCalled();
            expect(mockRegistry.count()).toBe(2);
        });

        it("should timeout old tasks", async () => {
            const oldTaskTime = Date.now() - mockLimits.taskTimeoutMs - 10000; // Beyond timeout

            (mockRegistry as any).addRecord({
                id: "old-task",
                userId: "user1",
                hasPremium: false,
                startTime: oldTaskTime,
            });

            (mockRegistry as any).addRecord({
                id: "new-task",
                userId: "user2", 
                hasPremium: false,
                startTime: Date.now() - 60000, // Recent
            });

            await checkLongRunningTasksInRegistry(
                mockRegistry as any,
                mockLimits,
                "Test",
            );

            // Should log timeout warning and remove old task
            expect(loggerWarnSpy).toHaveBeenCalledWith(
                expect.stringContaining("Test task old-task has exceeded timeout"),
                expect.any(Object),
            );
            expect(mockRegistry.count()).toBe(1);
            expect(mockRegistry.getOrderedActiveRecords()[0].id).toBe("new-task");
        });

        it("should handle high load by removing oldest free tier tasks", async () => {
            const currentTime = Date.now();

            // Add tasks to exceed limit
            for (let i = 0; i < 7; i++) {
                (mockRegistry as any).addRecord({
                    id: `task${i}`,
                    userId: `user${i}`,
                    hasPremium: false,
                    startTime: currentTime - (i * 10000), // Stagger times
                });
            }

            expect(mockRegistry.count()).toBe(7); // Exceeds maxActive (5)

            await checkLongRunningTasksInRegistry(
                mockRegistry as any,
                mockLimits,
                "Test",
            );

            // Should log high load warning and remove oldest tasks
            expect(loggerWarnSpy).toHaveBeenCalledWith(
                expect.stringContaining("High load detected"),
                expect.objectContaining({
                    currentCount: 7,
                    maxActive: 5,
                }),
            );

            // Should remove 2 oldest tasks (7 - 5 = 2)
            expect(mockRegistry.count()).toBe(5);
            
            // Verify oldest tasks were removed
            const remaining = mockRegistry.getOrderedActiveRecords();
            expect(remaining.map(r => r.id)).not.toContain("task5");
            expect(remaining.map(r => r.id)).not.toContain("task6");
        });

        it("should prioritize premium users during high load", async () => {
            const currentTime = Date.now();

            // Add mix of premium and free tasks
            (mockRegistry as any).addRecord({
                id: "premium1",
                userId: "premium-user1",
                hasPremium: true,
                startTime: currentTime - 60000, // Older
            });

            (mockRegistry as any).addRecord({
                id: "free1",
                userId: "free-user1", 
                hasPremium: false,
                startTime: currentTime - 30000, // Newer
            });

            // Add more to exceed limits
            for (let i = 0; i < 5; i++) {
                (mockRegistry as any).addRecord({
                    id: `free${i + 2}`,
                    userId: `free-user${i + 2}`,
                    hasPremium: false,
                    startTime: currentTime - (i * 5000),
                });
            }

            expect(mockRegistry.count()).toBe(7); // Exceeds limit

            await checkLongRunningTasksInRegistry(
                mockRegistry as any,
                mockLimits,
                "Test",
            );

            // Should preserve premium task and remove free tasks
            const remaining = mockRegistry.getOrderedActiveRecords();
            const premiumRemaining = remaining.filter(r => r.hasPremium);
            expect(premiumRemaining).toHaveLength(1);
            expect(premiumRemaining[0].id).toBe("premium1");
        });

        it("should handle empty registry gracefully", async () => {
            expect(mockRegistry.count()).toBe(0);

            await checkLongRunningTasksInRegistry(
                mockRegistry as any,
                mockLimits,
                "Test",
            );

            // Should not log anything or crash
            expect(loggerWarnSpy).not.toHaveBeenCalled();
            expect(loggerInfoSpy).not.toHaveBeenCalled();
        });

        it("should log summary when tasks are removed", async () => {
            const currentTime = Date.now();

            // Add tasks to trigger removal
            for (let i = 0; i < 3; i++) {
                (mockRegistry as any).addRecord({
                    id: `timeout-task${i}`,
                    userId: `user${i}`,
                    hasPremium: false,
                    startTime: currentTime - mockLimits.taskTimeoutMs - 10000, // All old
                });
            }

            await checkLongRunningTasksInRegistry(
                mockRegistry as any,
                mockLimits,
                "Test",
            );

            // Should log summary
            expect(loggerInfoSpy).toHaveBeenCalledWith(
                "Test task cleanup completed",
                expect.objectContaining({
                    removedCount: 3,
                    remainingCount: 0,
                }),
            );
        });

        it("should handle mixed timeout and high load scenarios", async () => {
            const currentTime = Date.now();

            // Add old tasks that should timeout
            (mockRegistry as any).addRecord({
                id: "old1",
                userId: "user1",
                hasPremium: false,
                startTime: currentTime - mockLimits.taskTimeoutMs - 5000,
            });

            (mockRegistry as any).addRecord({
                id: "old2",
                userId: "user2",
                hasPremium: true,
                startTime: currentTime - mockLimits.taskTimeoutMs - 3000,
            });

            // Add new tasks that cause high load
            for (let i = 0; i < 6; i++) {
                (mockRegistry as any).addRecord({
                    id: `new${i}`,
                    userId: `user${i + 3}`,
                    hasPremium: false,
                    startTime: currentTime - (i * 5000),
                });
            }

            expect(mockRegistry.count()).toBe(8); // 2 old + 6 new

            await checkLongRunningTasksInRegistry(
                mockRegistry as any,
                mockLimits,
                "Test",
            );

            // Should handle both timeout and high load
            expect(loggerWarnSpy).toHaveBeenCalledWith(
                expect.stringContaining("has exceeded timeout"),
                expect.any(Object),
            );
            expect(loggerWarnSpy).toHaveBeenCalledWith(
                expect.stringContaining("High load detected"),
                expect.any(Object),
            );

            // Final count should be within limits
            expect(mockRegistry.count()).toBeLessThanOrEqual(mockLimits.maxActive);
        });
    });
});
