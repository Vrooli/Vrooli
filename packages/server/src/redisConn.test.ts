import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
import { logger } from "./events/logger.js";
import { CacheService } from "./redisConn.js";

describe("CacheService Integration Tests", () => {
    let cacheService: CacheService;

    beforeAll(() => {
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    beforeEach(() => {
        // Get a fresh instance of CacheService
        cacheService = CacheService.get();
    });

    afterAll(async () => {
        // Clean up
        await cacheService.flushAll();
        await cacheService.close();
        vi.restoreAllMocks();
    });

    describe("CacheService basic operations", () => {
        it("should set and get a value", async () => {
            const key = "test-key";
            const value = { data: "test value", count: 42 };
            
            await cacheService.set(key, value);
            const retrieved = await cacheService.get<typeof value>(key);
            
            expect(retrieved).toEqual(value);
        });

        it("should return null for non-existent key", async () => {
            const result = await cacheService.get("non-existent-key");
            expect(result).toBeNull();
        });

        it("should delete a key", async () => {
            const key = "delete-test";
            const value = "test value";
            
            await cacheService.set(key, value);
            await cacheService.del(key);
            
            const result = await cacheService.get(key);
            expect(result).toBeNull();
        });

        it("should respect TTL", async () => {
            const key = "ttl-test";
            const value = "expires soon";
            
            // Set with 1 second TTL
            await cacheService.set(key, value, 1);
            
            // Should exist immediately
            const immediate = await cacheService.get(key);
            expect(immediate).toBe(value);
            
            // Wait for expiration
            await new Promise(resolve => setTimeout(resolve, 1100));
            
            // Should be gone
            const expired = await cacheService.get(key);
            expect(expired).toBeNull();
        });
    });

    describe("CacheService memo function", () => {
        it("should cache function results", async () => {
            const key = "memo-test";
            let callCount = 0;
            
            const expensiveFn = async () => {
                callCount++;
                return { result: "expensive computation", count: callCount };
            };
            
            // First call should execute function
            const result1 = await cacheService.memo(key, 60, expensiveFn);
            expect(result1.count).toBe(1);
            
            // Second call should return cached result
            const result2 = await cacheService.memo(key, 60, expensiveFn);
            expect(result2.count).toBe(1); // Same count, function not called again
            
            expect(callCount).toBe(1);
        });

        it("should handle concurrent memo calls", async () => {
            const key = "concurrent-memo";
            let callCount = 0;
            
            const slowFn = async () => {
                callCount++;
                await new Promise(resolve => setTimeout(resolve, 100));
                return callCount;
            };
            
            // Start multiple concurrent requests
            const promises = [
                cacheService.memo(key, 60, slowFn),
                cacheService.memo(key, 60, slowFn),
                cacheService.memo(key, 60, slowFn),
            ];
            
            const results = await Promise.all(promises);
            
            // All should return the same result
            expect(results[0]).toBe(1);
            expect(results[1]).toBe(1);
            expect(results[2]).toBe(1);
            
            // Function should only be called once
            expect(callCount).toBe(1);
        });
    });

    describe("CacheService singleton behavior", () => {
        it("should return the same instance", () => {
            const instance1 = CacheService.get();
            const instance2 = CacheService.get();
            
            expect(instance1).toBe(instance2);
        });
    });

    describe("CacheService namespace support", () => {
        it("should use namespace from environment", async () => {
            const key = "namespace-test";
            const value = "namespaced value";
            
            await cacheService.set(key, value);
            
            // Access raw Redis to check actual key
            const rawClient = await cacheService.raw();
            const namespace = process.env.CACHE_NAMESPACE || "vrooli";
            const actualKey = `${namespace}:${key}`;
            
            const rawValue = await rawClient.get(actualKey);
            expect(rawValue).toBe(JSON.stringify(value));
        });
    });
});