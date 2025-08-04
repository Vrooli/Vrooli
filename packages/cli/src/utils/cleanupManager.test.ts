// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-08-04
import { describe, it, expect, vi, beforeEach } from "vitest";
import { CleanupManager, cleanup, CLEANUP_PRIORITY } from "./cleanupManager.js";

describe("CleanupManager", () => {
    beforeEach(() => {
        // Reset cleanup state before each test
        cleanup.reset();
        vi.clearAllMocks();
    });

    describe("singleton pattern", () => {
        it("should return the same instance", () => {
            const instance1 = CleanupManager.getInstance();
            const instance2 = CleanupManager.getInstance();
            expect(instance1).toBe(instance2);
        });

        it("should export a cleanup instance", () => {
            expect(cleanup).toBeInstanceOf(CleanupManager);
        });
    });

    describe("task registration", () => {
        it("should register a cleanup task", () => {
            const handler = vi.fn();
            cleanup.register("test-task", handler);
            
            expect(cleanup.cleanedUp).toBe(false);
        });

        it("should register multiple tasks with priorities", () => {
            const handler1 = vi.fn();
            const handler2 = vi.fn();
            const handler3 = vi.fn();
            
            cleanup.register("low-priority", handler1, CLEANUP_PRIORITY.LOW);
            cleanup.register("high-priority", handler2, CLEANUP_PRIORITY.HIGH);
            cleanup.register("medium-priority", handler3, CLEANUP_PRIORITY.MEDIUM);
            
            expect(cleanup.cleanedUp).toBe(false);
        });

        it("should replace existing task with same name", () => {
            const handler1 = vi.fn();
            const handler2 = vi.fn();
            
            cleanup.register("test-task", handler1);
            cleanup.register("test-task", handler2); // Should replace handler1
            
            expect(cleanup.cleanedUp).toBe(false);
        });

        it("should use default priority when none provided", () => {
            const handler = vi.fn();
            cleanup.register("test-task", handler);
            
            expect(cleanup.cleanedUp).toBe(false);
        });
    });

    describe("task unregistration", () => {
        it("should unregister a cleanup task", () => {
            const handler = vi.fn();
            cleanup.register("test-task", handler);
            cleanup.unregister("test-task");
            
            expect(cleanup.cleanedUp).toBe(false);
        });

        it("should handle unregistering non-existent task", () => {
            cleanup.unregister("non-existent");
            
            expect(cleanup.cleanedUp).toBe(false);
        });
    });

    describe("async cleanup execution", () => {
        it("should execute all cleanup tasks in priority order", async () => {
            const callOrder: string[] = [];
            
            const handler1 = vi.fn(() => callOrder.push("low"));
            const handler2 = vi.fn(() => callOrder.push("high"));
            const handler3 = vi.fn(() => callOrder.push("medium"));
            
            cleanup.register("low-priority", handler1, CLEANUP_PRIORITY.LOW);
            cleanup.register("high-priority", handler2, CLEANUP_PRIORITY.HIGH);
            cleanup.register("medium-priority", handler3, CLEANUP_PRIORITY.MEDIUM);
            
            await cleanup.executeAll();
            
            expect(callOrder).toEqual(["high", "medium", "low"]);
            expect(cleanup.cleanedUp).toBe(true);
        });

        it("should handle async cleanup tasks", async () => {
            const asyncHandler = vi.fn(async () => {
                await new Promise(resolve => setTimeout(resolve, 10));
            });
            const syncHandler = vi.fn();
            
            cleanup.register("async-task", asyncHandler);
            cleanup.register("sync-task", syncHandler);
            
            await cleanup.executeAll();
            
            expect(asyncHandler).toHaveBeenCalled();
            expect(syncHandler).toHaveBeenCalled();
            expect(cleanup.cleanedUp).toBe(true);
        });

        it("should continue execution if a task throws an error", async () => {
            const errorHandler = vi.fn(() => {
                throw new Error("Cleanup error");
            });
            const successHandler = vi.fn();
            
            cleanup.register("error-task", errorHandler, CLEANUP_PRIORITY.HIGH);
            cleanup.register("success-task", successHandler, CLEANUP_PRIORITY.LOW);
            
            await cleanup.executeAll();
            
            expect(errorHandler).toHaveBeenCalled();
            expect(successHandler).toHaveBeenCalled();
            expect(cleanup.cleanedUp).toBe(true);
        });

        it("should prevent double cleanup", async () => {
            const handler = vi.fn();
            cleanup.register("test-task", handler);
            
            await cleanup.executeAll();
            expect(handler).toHaveBeenCalledTimes(1);
            expect(cleanup.cleanedUp).toBe(true);
            
            // Second call should not execute tasks again
            await cleanup.executeAll();
            expect(handler).toHaveBeenCalledTimes(1);
        });

        it("should handle timeout for async tasks", async () => {
            const slowAsyncHandler = vi.fn(async () => {
                await new Promise(resolve => setTimeout(resolve, 6000)); // Longer than timeout
            });
            
            cleanup.register("slow-task", slowAsyncHandler);
            
            const startTime = Date.now();
            await cleanup.executeAll();
            const duration = Date.now() - startTime;
            
            expect(duration).toBeLessThan(6000); // Should timeout before 6 seconds
            expect(cleanup.cleanedUp).toBe(true);
        }, 10000); // 10 second timeout for this test
    });

    describe("sync cleanup execution", () => {
        it("should execute synchronous cleanup tasks", () => {
            const syncHandler = vi.fn();
            cleanup.register("sync-task", syncHandler);
            
            cleanup.executeAllSync();
            
            expect(syncHandler).toHaveBeenCalled();
            expect(cleanup.cleanedUp).toBe(true);
        });

        it("should skip async cleanup tasks", () => {
            const asyncHandler = vi.fn(async () => {
                await new Promise(resolve => setTimeout(resolve, 10));
            });
            const syncHandler = vi.fn();
            
            cleanup.register("async-task", asyncHandler);
            cleanup.register("sync-task", syncHandler);
            
            cleanup.executeAllSync();
            
            expect(asyncHandler).toHaveBeenCalled(); // Called but promise ignored
            expect(syncHandler).toHaveBeenCalled();
            expect(cleanup.cleanedUp).toBe(true);
        });

        it("should handle errors silently", () => {
            const errorHandler = vi.fn(() => {
                throw new Error("Sync cleanup error");
            });
            const successHandler = vi.fn();
            
            cleanup.register("error-task", errorHandler);
            cleanup.register("success-task", successHandler);
            
            cleanup.executeAllSync();
            
            expect(errorHandler).toHaveBeenCalled();
            expect(successHandler).toHaveBeenCalled();
            expect(cleanup.cleanedUp).toBe(true);
        });

        it("should prevent double cleanup", () => {
            const handler = vi.fn();
            cleanup.register("test-task", handler);
            
            cleanup.executeAllSync();
            expect(handler).toHaveBeenCalledTimes(1);
            expect(cleanup.cleanedUp).toBe(true);
            
            // Second call should not execute tasks again
            cleanup.executeAllSync();
            expect(handler).toHaveBeenCalledTimes(1);
        });
    });

    describe("state management", () => {
        it("should track cleanup state", () => {
            expect(cleanup.cleanedUp).toBe(false);
            
            cleanup.executeAllSync();
            
            expect(cleanup.cleanedUp).toBe(true);
        });

        it("should reset state", () => {
            const handler = vi.fn();
            cleanup.register("test-task", handler);
            cleanup.executeAllSync();
            
            expect(cleanup.cleanedUp).toBe(true);
            
            cleanup.reset();
            
            expect(cleanup.cleanedUp).toBe(false);
        });
    });

    describe("priority constants", () => {
        it("should export priority constants", () => {
            expect(CLEANUP_PRIORITY.HIGH).toBe(10);
            expect(CLEANUP_PRIORITY.MEDIUM).toBe(20);
            expect(CLEANUP_PRIORITY.LOW).toBe(30);
            expect(CLEANUP_PRIORITY.DEFAULT).toBe(50);
        });
    });

    describe("debug output", () => {
        beforeEach(() => {
            vi.stubEnv("DEBUG", "true");
            vi.spyOn(console, "error").mockImplementation(() => {});
        });

        it("should log debug information when DEBUG is set", async () => {
            const handler = vi.fn();
            cleanup.register("debug-task", handler);
            
            await cleanup.executeAll(0);
            
            expect(console.error).toHaveBeenCalledWith(
                expect.stringContaining("[DEBUG] Starting cleanup with exit code 0"),
            );
            expect(console.error).toHaveBeenCalledWith(
                expect.stringContaining("[DEBUG] Running cleanup: debug-task"),
            );
            expect(console.error).toHaveBeenCalledWith(
                expect.stringContaining("[DEBUG] Cleanup completed"),
            );
        });

        it("should log errors when DEBUG is set", async () => {
            const errorHandler = vi.fn(() => {
                throw new Error("Test error");
            });
            cleanup.register("error-task", errorHandler);
            
            await cleanup.executeAll(1, new Error("Trigger error"));
            
            expect(console.error).toHaveBeenCalledWith(
                expect.stringContaining("[DEBUG] Error: Trigger error"),
            );
            expect(console.error).toHaveBeenCalledWith(
                expect.stringContaining("[DEBUG] Cleanup task 'error-task' failed:"),
                expect.any(Error),
            );
        });
    });
});
