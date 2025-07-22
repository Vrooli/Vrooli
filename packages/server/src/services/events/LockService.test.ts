/**
 * LockService tests
 * 
 * Tests for distributed locking to prevent race conditions.
 * Critical infrastructure for data integrity across the system.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { RedisLockService, InMemoryLockService } from "./LockService.js";
import type { Redis } from "ioredis";
import type { LockOptions } from "./types.js";

// Mock logger to suppress output during tests
vi.mock("../../events/logger.js", () => ({
    logger: {
        debug: vi.fn(),
        error: vi.fn(),
        info: vi.fn(),
        warning: vi.fn(),
        warn: vi.fn(),
    },
}));

describe("LockService", () => {
    describe("RedisLockService", () => {
        let mockRedis: jest.Mocked<Redis>;
        let lockService: RedisLockService;

        beforeEach(() => {
            // Create comprehensive Redis mock
            mockRedis = {
                set: vi.fn(),
                eval: vi.fn(),
            } as any;
            
            lockService = new RedisLockService(mockRedis);
        });

        afterEach(() => {
            vi.clearAllMocks();
        });

        describe("acquire", () => {
            it("should acquire lock successfully on first attempt", async () => {
                mockRedis.set.mockResolvedValue("OK");
                
                const options: LockOptions = { ttl: 5000 };
                const lock = await lockService.acquire("test-key", options);

                expect(mockRedis.set).toHaveBeenCalledWith(
                    "lock:test-key",
                    expect.any(String),
                    "PX",
                    5000,
                    "NX",
                );
                expect(lock).toBeDefined();
            });

            it("should retry when lock is already held", async () => {
                // First attempt fails, second succeeds
                mockRedis.set
                    .mockResolvedValueOnce(null) // Lock already exists
                    .mockResolvedValueOnce("OK"); // Lock acquired

                const options: LockOptions = { ttl: 5000, retries: 1 };
                
                // Mock setTimeout to resolve immediately for testing
                vi.spyOn(global, "setTimeout").mockImplementation((fn: any) => {
                    fn();
                    return {} as any;
                });

                const lock = await lockService.acquire("test-key", options);

                expect(mockRedis.set).toHaveBeenCalledTimes(2);
                expect(lock).toBeDefined();

                vi.restoreAllMocks();
            });

            it("should fail after max retries exceeded", async () => {
                mockRedis.set.mockResolvedValue(null); // Always fails

                const options: LockOptions = { ttl: 5000, retries: 2 };
                
                // Mock setTimeout to resolve immediately for testing
                vi.spyOn(global, "setTimeout").mockImplementation((fn: any) => {
                    fn();
                    return {} as any;
                });

                await expect(lockService.acquire("test-key", options))
                    .rejects.toThrow("Failed to acquire lock after 3 attempts: test-key");

                expect(mockRedis.set).toHaveBeenCalledTimes(3);

                vi.restoreAllMocks();
            });

            it("should handle Redis errors and retry", async () => {
                // First attempt throws error, second succeeds
                mockRedis.set
                    .mockRejectedValueOnce(new Error("Redis connection error"))
                    .mockResolvedValueOnce("OK");

                const options: LockOptions = { ttl: 5000, retries: 1 };
                
                // Mock setTimeout to resolve immediately for testing
                vi.spyOn(global, "setTimeout").mockImplementation((fn: any) => {
                    fn();
                    return {} as any;
                });

                const lock = await lockService.acquire("test-key", options);

                expect(mockRedis.set).toHaveBeenCalledTimes(2);
                expect(lock).toBeDefined();

                vi.restoreAllMocks();
            });

            it("should fail when Redis errors persist", async () => {
                const redisError = new Error("Persistent Redis error");
                mockRedis.set.mockRejectedValue(redisError);

                const options: LockOptions = { ttl: 5000, retries: 1 };
                
                // Mock setTimeout to resolve immediately for testing
                vi.spyOn(global, "setTimeout").mockImplementation((fn: any) => {
                    fn();
                    return {} as any;
                });

                await expect(lockService.acquire("test-key", options))
                    .rejects.toThrow("Persistent Redis error");

                expect(mockRedis.set).toHaveBeenCalledTimes(2);

                vi.restoreAllMocks();
            });

            it("should use default retries when not specified", async () => {
                mockRedis.set.mockResolvedValue("OK");

                const options: LockOptions = { ttl: 5000 }; // No retries specified
                const lock = await lockService.acquire("test-key", options);

                expect(lock).toBeDefined();
                expect(mockRedis.set).toHaveBeenCalledTimes(1);
            });

            it("should generate unique lock values", async () => {
                mockRedis.set.mockResolvedValue("OK");

                const options: LockOptions = { ttl: 5000 };
                
                await lockService.acquire("test-key-1", options);
                await lockService.acquire("test-key-2", options);

                expect(mockRedis.set).toHaveBeenCalledTimes(2);
                
                const firstCall = mockRedis.set.mock.calls[0];
                const secondCall = mockRedis.set.mock.calls[1];
                
                // Lock values should be different
                expect(firstCall[1]).not.toBe(secondCall[1]);
            });
        });

        describe("RedisLock", () => {
            it("should release lock successfully", async () => {
                mockRedis.set.mockResolvedValue("OK");
                mockRedis.eval.mockResolvedValue(1); // Successful deletion

                const options: LockOptions = { ttl: 5000 };
                const lock = await lockService.acquire("test-key", options);

                await lock.release();

                expect(mockRedis.eval).toHaveBeenCalledWith(
                    expect.stringContaining("redis.call(\"get\", KEYS[1])"),
                    1,
                    "lock:test-key",
                    expect.any(String),
                );
            });

            it("should handle lock already released", async () => {
                mockRedis.set.mockResolvedValue("OK");
                mockRedis.eval.mockResolvedValue(0); // Lock not found

                const options: LockOptions = { ttl: 5000 };
                const lock = await lockService.acquire("test-key", options);

                await lock.release();

                expect(mockRedis.eval).toHaveBeenCalled();
                // Should not throw error even if lock was already released
            });

            it("should handle multiple release calls gracefully", async () => {
                mockRedis.set.mockResolvedValue("OK");
                mockRedis.eval.mockResolvedValue(1);

                const options: LockOptions = { ttl: 5000 };
                const lock = await lockService.acquire("test-key", options);

                await lock.release();
                await lock.release(); // Second release should be no-op

                expect(mockRedis.eval).toHaveBeenCalledTimes(1);
            });

            it("should handle release errors", async () => {
                mockRedis.set.mockResolvedValue("OK");
                mockRedis.eval.mockRejectedValue(new Error("Redis eval error"));

                const options: LockOptions = { ttl: 5000 };
                const lock = await lockService.acquire("test-key", options);

                await expect(lock.release()).rejects.toThrow("Redis eval error");
            });

            it("should start renewal for long-lived locks", async () => {
                mockRedis.set.mockResolvedValue("OK");
                mockRedis.eval.mockResolvedValue(1);

                // Mock setInterval to capture the renewal function
                let renewalFn: () => void;
                vi.spyOn(global, "setInterval").mockImplementation((fn: any, interval: number) => {
                    renewalFn = fn;
                    expect(interval).toBe(Math.floor(15000 / 3)); // Should be 1/3 of TTL
                    return {} as any;
                });

                const options: LockOptions = { ttl: 15000 }; // Long-lived lock
                const lock = await lockService.acquire("test-key", options);

                expect(setInterval).toHaveBeenCalled();

                // Test renewal logic
                renewalFn!();
                await new Promise(resolve => setTimeout(resolve, 0)); // Let async operations complete

                expect(mockRedis.eval).toHaveBeenCalledWith(
                    expect.stringContaining("redis.call(\"pexpire\", KEYS[1], ARGV[2])"),
                    1,
                    "lock:test-key",
                    expect.any(String),
                    "15000",
                );

                vi.restoreAllMocks();
            });

            it("should not start renewal for short-lived locks", async () => {
                mockRedis.set.mockResolvedValue("OK");

                vi.spyOn(global, "setInterval");

                const options: LockOptions = { ttl: 5000 }; // Short-lived lock
                await lockService.acquire("test-key", options);

                expect(setInterval).not.toHaveBeenCalled();

                vi.restoreAllMocks();
            });

            it("should stop renewal when lock is released", async () => {
                mockRedis.set.mockResolvedValue("OK");
                mockRedis.eval.mockResolvedValue(1);

                let intervalId: any;
                vi.spyOn(global, "setInterval").mockImplementation((fn: any, interval: number) => {
                    intervalId = { id: "test-interval" };
                    return intervalId;
                });
                vi.spyOn(global, "clearInterval");

                const options: LockOptions = { ttl: 15000 };
                const lock = await lockService.acquire("test-key", options);

                await lock.release();

                expect(clearInterval).toHaveBeenCalledWith(intervalId);

                vi.restoreAllMocks();
            });
        });
    });

    describe("InMemoryLockService", () => {
        let lockService: InMemoryLockService;

        beforeEach(() => {
            lockService = new InMemoryLockService();
        });

        describe("acquire", () => {
            it("should acquire lock successfully", async () => {
                const options: LockOptions = { ttl: 5000 };
                const lock = await lockService.acquire("test-key", options);

                expect(lock).toBeDefined();
            });

            it("should prevent concurrent locks on same key", async () => {
                const options: LockOptions = { ttl: 5000, retries: 0 };
                
                const lock1 = await lockService.acquire("test-key", options);
                
                await expect(lockService.acquire("test-key", options))
                    .rejects.toThrow("Failed to acquire lock after 1 attempts: test-key");

                // Release first lock
                await lock1.release();

                // Should be able to acquire again
                const lock2 = await lockService.acquire("test-key", options);
                expect(lock2).toBeDefined();
            });

            it("should allow concurrent locks on different keys", async () => {
                const options: LockOptions = { ttl: 5000 };
                
                const lock1 = await lockService.acquire("test-key-1", options);
                const lock2 = await lockService.acquire("test-key-2", options);

                expect(lock1).toBeDefined();
                expect(lock2).toBeDefined();
            });

            it("should retry when lock becomes available", async () => {
                const options1: LockOptions = { ttl: 100 }; // Short TTL
                const options2: LockOptions = { ttl: 5000, retries: 2 };

                // Mock setTimeout to advance time and resolve quickly
                const timeoutCallbacks: (() => void)[] = [];
                vi.spyOn(global, "setTimeout").mockImplementation((fn: any, delay: number) => {
                    if (delay === 100) {
                        // This is the retry delay, execute immediately
                        fn();
                    } else {
                        timeoutCallbacks.push(fn);
                    }
                    return {} as any;
                });

                // Acquire first lock
                const lock1 = await lockService.acquire("test-key", options1);

                // Try to acquire second lock (should retry)
                const lock2Promise = lockService.acquire("test-key", options2);

                // Manually expire the first lock by advancing time
                // In real implementation, this would be handled by TTL
                await lock1.release();

                const lock2 = await lock2Promise;
                expect(lock2).toBeDefined();

                vi.restoreAllMocks();
            });

            it("should clean up expired locks", async () => {
                const options: LockOptions = { ttl: 1 }; // Very short TTL
                
                await lockService.acquire("test-key", options);

                // Wait for lock to expire
                await new Promise(resolve => setTimeout(resolve, 10));

                // Should be able to acquire again (expired lock cleaned up)
                const lock2 = await lockService.acquire("test-key", options);
                expect(lock2).toBeDefined();
            });

            it("should use default retries when not specified", async () => {
                const options: LockOptions = { ttl: 5000 };
                const lock = await lockService.acquire("test-key", options);

                expect(lock).toBeDefined();
            });
        });

        describe("InMemoryLock", () => {
            it("should release lock successfully", async () => {
                const options: LockOptions = { ttl: 5000 };
                const lock = await lockService.acquire("test-key", options);

                await lock.release();

                // Should be able to acquire same key again
                const lock2 = await lockService.acquire("test-key", options);
                expect(lock2).toBeDefined();
            });

            it("should handle multiple release calls gracefully", async () => {
                const options: LockOptions = { ttl: 5000 };
                const lock = await lockService.acquire("test-key", options);

                await lock.release();
                await lock.release(); // Second release should be no-op

                // Should still be able to acquire same key
                const lock2 = await lockService.acquire("test-key", options);
                expect(lock2).toBeDefined();
            });

            it("should only release own lock", async () => {
                const options: LockOptions = { ttl: 5000 };
                const lock1 = await lockService.acquire("test-key-1", options);
                const lock2 = await lockService.acquire("test-key-2", options);

                await lock1.release();

                // lock2 should still be active
                await expect(lockService.acquire("test-key-2", { ttl: 5000, retries: 0 }))
                    .rejects.toThrow("Failed to acquire lock");

                await lock2.release();

                // Now should be able to acquire test-key-2
                const lock3 = await lockService.acquire("test-key-2", options);
                expect(lock3).toBeDefined();
            });
        });

        describe("concurrency scenarios", () => {
            it("should handle rapid sequential acquisitions", async () => {
                const options: LockOptions = { ttl: 5000 };
                const keys = Array.from({ length: 10 }, (_, i) => `test-key-${i}`);

                const locks = await Promise.all(
                    keys.map(key => lockService.acquire(key, options)),
                );

                expect(locks).toHaveLength(10);
                
                // All locks should be independent
                for (let i = 0; i < locks.length; i++) {
                    await locks[i].release();
                }
            });

            it("should handle lock contention properly", async () => {
                const options: LockOptions = { ttl: 100, retries: 5 };
                
                // Start multiple acquisition attempts
                const acquisitions = Array.from({ length: 5 }, () => 
                    lockService.acquire("contended-key", options),
                );

                // Only one should succeed, others should fail or retry
                const results = await Promise.allSettled(acquisitions);
                
                const successful = results.filter(r => r.status === "fulfilled");
                const failed = results.filter(r => r.status === "rejected");

                expect(successful.length).toBe(1);
                expect(failed.length).toBe(4);

                // Release the successful lock
                if (successful.length > 0) {
                    await (successful[0] as any).value.release();
                }
            });
        });
    });

    describe("interface compliance", () => {
        it("should implement ILockService interface correctly", () => {
            const redisService = new RedisLockService({} as Redis);
            const memoryService = new InMemoryLockService();

            expect(typeof redisService.acquire).toBe("function");
            expect(typeof memoryService.acquire).toBe("function");
        });

        it("should return locks that implement Lock interface", async () => {
            const memoryService = new InMemoryLockService();
            const options: LockOptions = { ttl: 5000 };
            
            const lock = await memoryService.acquire("test-key", options);
            
            expect(typeof lock.release).toBe("function");
        });
    });
});
