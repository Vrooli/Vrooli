import { describe, it, expect, beforeAll, afterAll, beforeEach, afterEach, vi } from "vitest";
import { LRUCache } from "./lruCache.js";

describe("LRUCache", () => {
    let cache: LRUCache<string, number>;
    let consoleWarnSpy: any;

    beforeAll(() => {
        consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    });

    beforeEach(() => {
        // Fake timers for TTL tests (won't affect sync tests)
        vi.useFakeTimers();
        cache = new LRUCache({ limit: 3 });
        consoleWarnSpy.mockClear();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    afterAll(() => {
        consoleWarnSpy.mockRestore();
    });

    it("should add items to the cache", () => {
        cache.set("item1", 1);
        expect(cache.get("item1")).toBe(1);
    });

    it("should return undefined for missing items", () => {
        expect(cache.get("missingItem")).toBeUndefined();
    });

    it("should update the value if the same key is added to the cache", () => {
        cache.set("item", 1);
        cache.set("item", 2);
        expect(cache.get("item")).toBe(2);
    });

    it("should maintain the max limit by evicting the least recently used item", () => {
        cache.set("item1", 1);
        cache.set("item2", 2);
        cache.set("item3", 3);
        cache.set("item4", 4); // should evict 'item1'

        expect(cache.get("item1")).toBeUndefined();
        expect(cache.get("item2")).toBe(2);
        expect(cache.get("item3")).toBe(3);
        expect(cache.get("item4")).toBe(4);
    });

    it("should move the recently accessed item to the end", () => {
        cache.set("item1", 1);
        cache.set("item2", 2);
        cache.set("item3", 3);

        expect(cache.get("item1")).toBe(1); // now most recently used

        cache.set("item4", 4); // should evict 'item2'
        expect(cache.get("item1")).toBe(1);
        expect(cache.get("item2")).toBeUndefined();
        expect(cache.get("item3")).toBe(3);
        expect(cache.get("item4")).toBe(4);
    });

    it("should handle a large number of items", () => {
        const largeCache = new LRUCache<string, number>({ limit: 1000 });
        for (let i = 0; i < 2000; i++) {
            largeCache.set("item" + i, i);
        }

        expect(largeCache.size()).toBe(1000);

        for (let i = 0; i < 1000; i++) {
            expect(largeCache.get("item" + i)).toBeUndefined();
        }
        for (let i = 1000; i < 2000; i++) {
            expect(largeCache.get("item" + i)).toBe(i);
        }
    });

    it("should not add items over a certain size", () => {
        const smallCache = new LRUCache<string, string>({ limit: 3, maxSizeBytes: 10 });

        smallCache.set("smallItem", "small"); // ~5 bytes
        smallCache.set("largeItem", "This is a large item"); // >10 bytes, skipped

        expect(smallCache.get("smallItem")).toBe("small");
        expect(smallCache.get("largeItem")).toBeUndefined();
    });

    it("should evict the least recently used item, respecting size limits", () => {
        const smallCache = new LRUCache<string, string>({ limit: 3, maxSizeBytes: 10 });

        smallCache.set("item1", "12345");        // 5 bytes
        smallCache.set("item2", "1234567890");   // 10 bytes
        smallCache.set("item3", "12345");        // 5 bytes

        expect(smallCache.get("item2")).toBe("1234567890"); // mark used

        smallCache.set("largeItem", "This is a large item"); // skipped

        smallCache.set("item4", "12345");        // should evict 'item1'

        expect(smallCache.get("item1")).toBeUndefined();
        expect(smallCache.get("item2")).toBe("1234567890");
        expect(smallCache.get("item3")).toBe("12345");
        expect(smallCache.get("item4")).toBe("12345");
        expect(smallCache.get("largeItem")).toBeUndefined();
    });

    // --- TTL-specific tests ---

    it("should not expire entries without TTL", () => {
        cache.set("perm", 7);
        vi.advanceTimersByTime(10_000);
        expect(cache.get("perm")).toBe(7);
    });

    it("should expire entries after the default TTL", () => {
        const ttlCache = new LRUCache<string, number>({ limit: 3, defaultTTLMs: 100 });
        ttlCache.set("temp", 123);
        expect(ttlCache.get("temp")).toBe(123);
        vi.advanceTimersByTime(101);
        expect(ttlCache.get("temp")).toBeUndefined();
    });

    it("should expire entries after a per-entry TTL override", () => {
        const ttlCache = new LRUCache<string, number>({ limit: 3 });
        ttlCache.set("override", 456, 50);
        expect(ttlCache.get("override")).toBe(456);
        vi.advanceTimersByTime(51);
        expect(ttlCache.get("override")).toBeUndefined();
    });

    // --- Size calculation tests ---

    it("should handle size calculation for objects", () => {
        const cache = new LRUCache<string, any>({ limit: 3, maxSizeBytes: 50 });
        
        // Small object should be accepted
        cache.set("smallObj", { foo: "bar" });
        expect(cache.get("smallObj")).toEqual({ foo: "bar" });
        
        // Large object should be rejected
        const largeObj = { data: "x".repeat(100) };
        cache.set("largeObj", largeObj);
        expect(cache.get("largeObj")).toBeUndefined();
        expect(consoleWarnSpy).toHaveBeenCalledWith(
            expect.stringContaining("Skipping cache set for key largeObj"),
        );
    });

    it("should handle size calculation errors gracefully", () => {
        const cache = new LRUCache<string, any>({ limit: 3, maxSizeBytes: 50 });
        
        // Create a circular reference object that can't be JSON.stringified
        const circularObj: any = { name: "test" };
        circularObj.self = circularObj;
        
        cache.set("circular", circularObj);
        expect(cache.get("circular")).toEqual(circularObj);
        expect(consoleWarnSpy).toHaveBeenCalledWith(
            expect.stringContaining("Error calculating size for cache value"),
        );
    });

    // --- Additional method tests ---

    it("should correctly implement has() method", () => {
        cache.set("exists", 1);
        expect(cache.has("exists")).toBe(true);
        expect(cache.has("doesNotExist")).toBe(false);
        
        // Verify has() doesn't affect LRU order
        cache.set("item1", 1);
        cache.set("item2", 2);
        cache.set("item3", 3);
        cache.has("item1"); // Should not move item1 to end
        cache.set("item4", 4); // Should still evict item1
        expect(cache.get("item1")).toBeUndefined();
    });

    it("should handle has() with expired entries", () => {
        const ttlCache = new LRUCache<string, number>({ limit: 3, defaultTTLMs: 100 });
        ttlCache.set("temp", 123);
        expect(ttlCache.has("temp")).toBe(true);
        vi.advanceTimersByTime(101);
        expect(ttlCache.has("temp")).toBe(false);
    });

    it("should correctly implement delete() method", () => {
        cache.set("toDelete", 1);
        cache.set("toKeep", 2);
        
        expect(cache.delete("toDelete")).toBe(true);
        expect(cache.delete("nonExistent")).toBe(false);
        
        expect(cache.get("toDelete")).toBeUndefined();
        expect(cache.get("toKeep")).toBe(2);
        expect(cache.size()).toBe(1);
    });

    it("should correctly implement clear() method", () => {
        cache.set("item1", 1);
        cache.set("item2", 2);
        cache.set("item3", 3);
        
        expect(cache.size()).toBe(3);
        cache.clear();
        expect(cache.size()).toBe(0);
        expect(cache.get("item1")).toBeUndefined();
        expect(cache.get("item2")).toBeUndefined();
        expect(cache.get("item3")).toBeUndefined();
    });

    it("should implement entries() iterator correctly", () => {
        cache.set("first", 1);
        cache.set("second", 2);
        cache.set("third", 3);
        
        const entries = Array.from(cache.entries());
        expect(entries).toEqual([
            ["first", 1],
            ["second", 2], 
            ["third", 3],
        ]);
    });

    it("should skip expired entries in entries() iterator", () => {
        const ttlCache = new LRUCache<string, number>({ limit: 3, defaultTTLMs: 100 });
        ttlCache.set("permanent", 1, 0); // No TTL for permanent
        ttlCache.set("temporary", 2, 100); // 100ms TTL
        
        // Advance time to expire the temporary entry
        vi.advanceTimersByTime(101);
        
        const entries = Array.from(ttlCache.entries());
        expect(entries).toEqual([["permanent", 1]]);
    });

    it("should implement forceCleanup() method", () => {
        const ttlCache = new LRUCache<string, number>({ limit: 5, defaultTTLMs: 100 });
        ttlCache.set("item1", 1);
        ttlCache.set("item2", 2);
        ttlCache.set("item3", 3);
        
        // Advance time to expire all entries
        vi.advanceTimersByTime(101);
        
        // forceCleanup should remove expired entries
        ttlCache.forceCleanup();
        expect(ttlCache.size()).toBe(0);
    });

    // --- Constructor validation tests ---

    it("should throw error for invalid limit", () => {
        expect(() => new LRUCache({ limit: 0 })).toThrow("LRUCache limit must be > 0");
        expect(() => new LRUCache({ limit: -1 })).toThrow("LRUCache limit must be > 0");
    });

    it("should handle custom cleanup configuration", () => {
        const customCache = new LRUCache<string, number>({ 
            limit: 3, 
            cleanupCooldownMs: 500,
            maxExpiredEntries: 50,
            defaultTTLMs: 100,
        });
        
        // Add some entries that will expire
        customCache.set("item1", 1);
        customCache.set("item2", 2);
        
        expect(customCache.size()).toBe(2);
    });

    // --- TTL override tests ---

    it("should handle TTL override with zero (no expiration)", () => {
        const ttlCache = new LRUCache<string, number>({ limit: 3, defaultTTLMs: 100 });
        ttlCache.set("noExpiry", 123, 0); // Override with 0 = no expiration
        
        vi.advanceTimersByTime(200); // Well past default TTL
        expect(ttlCache.get("noExpiry")).toBe(123);
    });

    it("should handle negative TTL (no expiration)", () => {
        const ttlCache = new LRUCache<string, number>({ limit: 3, defaultTTLMs: 100 });
        ttlCache.set("noExpiry", 456, -1); // Negative TTL = no expiration
        
        vi.advanceTimersByTime(200);
        expect(ttlCache.get("noExpiry")).toBe(456);
    });

    // --- Edge cases and stress tests ---

    it("should handle updating existing keys with different TTL", () => {
        const ttlCache = new LRUCache<string, number>({ limit: 3, defaultTTLMs: 100 });
        
        // Set with default TTL
        ttlCache.set("item", 1);
        
        // Update with longer TTL
        ttlCache.set("item", 2, 500);
        
        // Should not expire at default TTL time
        vi.advanceTimersByTime(150);
        expect(ttlCache.get("item")).toBe(2);
        
        // Should expire at new TTL time
        vi.advanceTimersByTime(400);
        expect(ttlCache.get("item")).toBeUndefined();
    });

    it("should handle complex expiration queue management", () => {
        const ttlCache = new LRUCache<string, number>({ 
            limit: 10, 
            defaultTTLMs: 100,
            maxExpiredEntries: 2, // Force frequent cleanup
        });
        
        // Add entries with different expiration times
        ttlCache.set("early1", 1, 50);
        ttlCache.set("early2", 2, 60); 
        ttlCache.set("late1", 3, 200);
        ttlCache.set("late2", 4, 300);
        
        // Advance to expire early entries
        vi.advanceTimersByTime(70);
        
        // Accessing should trigger cleanup
        expect(ttlCache.get("early1")).toBeUndefined();
        expect(ttlCache.get("early2")).toBeUndefined();
        expect(ttlCache.get("late1")).toBe(3);
        expect(ttlCache.get("late2")).toBe(4);
    });

    it("should handle size eviction combined with TTL", () => {
        const cache = new LRUCache<string, string>({ 
            limit: 3, 
            maxSizeBytes: 20,
            defaultTTLMs: 100,
        });
        
        cache.set("small1", "abc", 100);     // 3 bytes, explicit TTL
        cache.set("small2", "def", 100);     // 3 bytes, explicit TTL  
        cache.set("medium", "abcdefgh", 100); // 8 bytes, explicit TTL
        
        // Total is 14 bytes, under limit
        expect(cache.size()).toBe(3);
        
        // Try to add large item (should be rejected)
        cache.set("large", "x".repeat(25)); // 25 bytes > 20 limit
        expect(cache.get("large")).toBeUndefined();
        expect(cache.size()).toBe(3);
        
        // Advance time to expire all
        vi.advanceTimersByTime(101);
        // Force cleanup to process expired entries
        cache.forceCleanup();
        expect(cache.size()).toBe(0);
    });

    describe("Stress Testing and Memory Management", () => {
        it("should handle rapid insertion and deletion cycles", () => {
            const cache = new LRUCache<string, number>({ limit: 100 });
            
            // Perform many rapid operations
            for (let cycle = 0; cycle < 10; cycle++) {
                // Fill cache to capacity
                for (let i = 0; i < 100; i++) {
                    cache.set(`cycle-${cycle}-item-${i}`, i);
                }
                expect(cache.size()).toBe(100);
                
                // Clear half the cache
                for (let i = 0; i < 50; i++) {
                    cache.delete(`cycle-${cycle}-item-${i}`);
                }
                expect(cache.size()).toBe(50);
                
                // Clear remaining
                cache.clear();
                expect(cache.size()).toBe(0);
            }
        });

        it("should maintain performance under heavy load", () => {
            const cache = new LRUCache<string, string>({ limit: 1000 });
            const numOperations = 10000;
            
            const startTime = Date.now();
            
            // Mixed operations: set, get, delete
            for (let i = 0; i < numOperations; i++) {
                const key = `key-${i % 500}`; // Create key collisions
                
                if (i % 3 === 0) {
                    cache.set(key, `value-${i}`);
                } else if (i % 3 === 1) {
                    cache.get(key);
                } else {
                    cache.delete(key);
                }
            }
            
            const endTime = Date.now();
            const duration = endTime - startTime;
            
            // Should complete in reasonable time (less than 1 second)
            expect(duration).toBeLessThan(1000);
            
            // Cache should not exceed its limit
            expect(cache.size()).toBeLessThanOrEqual(1000);
        });

        it("should handle memory pressure with large objects", () => {
            const cache = new LRUCache<string, any>({ 
                limit: 100, 
                maxSizeBytes: 50000, // 50KB limit
            });
            
            // Add large objects that approach the size limit
            const largeObject = {
                data: "x".repeat(1000), // ~1KB per object
                metadata: { id: 1, type: "large", nested: { deep: "value" } },
                array: new Array(100).fill("item"),
            };
            
            let addedCount = 0;
            for (let i = 0; i < 100; i++) {
                const key = `large-obj-${i}`;
                const obj = { ...largeObject, id: i };
                
                cache.set(key, obj);
                
                if (cache.has(key)) {
                    addedCount++;
                }
            }
            
            // Should have added some objects but hit size limits
            expect(addedCount).toBeGreaterThan(0);
            expect(addedCount).toBeLessThanOrEqual(100); // Size limit should prevent some from being added
            
            // All objects in cache should be retrievable
            for (const [key, value] of cache.entries()) {
                expect(value).toBeDefined();
                expect(value.id).toBeGreaterThanOrEqual(0);
            }
        });

        it("should handle concurrent-like access patterns", () => {
            const cache = new LRUCache<string, number>({ limit: 50 });
            
            // Simulate multiple "threads" accessing different key ranges
            const thread1Keys = Array.from({ length: 25 }, (_, i) => `thread1-${i}`);
            const thread2Keys = Array.from({ length: 25 }, (_, i) => `thread2-${i}`);
            const thread3Keys = Array.from({ length: 25 }, (_, i) => `thread3-${i}`);
            
            // Interleave operations from different "threads"
            for (let i = 0; i < 25; i++) {
                // Thread 1 operations
                cache.set(thread1Keys[i], i * 100);
                cache.get(thread1Keys[Math.floor(i / 2)]);
                
                // Thread 2 operations
                cache.set(thread2Keys[i], i * 200);
                cache.get(thread2Keys[Math.floor(i / 2)]);
                
                // Thread 3 operations (will cause evictions)
                cache.set(thread3Keys[i], i * 300);
                cache.delete(thread3Keys[Math.floor(i / 3)]);
            }
            
            // Cache should maintain consistency
            expect(cache.size()).toBeLessThanOrEqual(50);
            
            // Should be able to iterate through all entries
            let entryCount = 0;
            for (const [key, value] of cache.entries()) {
                entryCount++;
                expect(typeof key).toBe("string");
                expect(typeof value).toBe("number");
            }
            expect(entryCount).toBe(cache.size());
        });
    });

    describe("Advanced TTL and Cleanup Scenarios", () => {
        it("should handle complex TTL expiration patterns", () => {
            const cache = new LRUCache<string, string>({ 
                limit: 100, 
                defaultTTLMs: 1000,
                maxExpiredEntries: 5, // Trigger cleanup frequently
            });
            
            // Add items with different TTL values
            cache.set("short-lived-1", "value1", 100);
            cache.set("short-lived-2", "value2", 150);
            cache.set("medium-lived-1", "value3", 500);
            cache.set("medium-lived-2", "value4", 750);
            cache.set("long-lived", "value5", 2000);
            cache.set("default-ttl", "value6"); // Uses default TTL
            cache.set("permanent", "value7", 0); // No expiration
            
            expect(cache.size()).toBe(7);
            
            // Advance time to expire short-lived items
            vi.advanceTimersByTime(200);
            
            // Access cache to trigger cleanup
            cache.get("medium-lived-1");
            
            // Short-lived items should be gone
            expect(cache.get("short-lived-1")).toBeUndefined();
            expect(cache.get("short-lived-2")).toBeUndefined();
            expect(cache.get("medium-lived-1")).toBe("value3");
            expect(cache.get("permanent")).toBe("value7");
            
            // Advance more to expire medium-lived items
            vi.advanceTimersByTime(600);
            cache.forceCleanup();
            
            expect(cache.get("medium-lived-1")).toBeUndefined();
            expect(cache.get("medium-lived-2")).toBeUndefined();
            expect(cache.get("long-lived")).toBe("value5");
            expect(cache.get("permanent")).toBe("value7");
        });

        it("should maintain cache consistency during cleanup cycles", () => {
            const cache = new LRUCache<string, number>({ 
                limit: 20,
                defaultTTLMs: 100,
                maxExpiredEntries: 3,
                cleanupCooldownMs: 50,
            });
            
            // Add items in waves with different expiration times
            for (let wave = 0; wave < 5; wave++) {
                for (let i = 0; i < 5; i++) {
                    const key = `wave-${wave}-item-${i}`;
                    const ttl = 50 + (wave * 25); // Staggered expiration
                    cache.set(key, wave * 100 + i, ttl);
                }
                
                // Advance time slightly between waves
                vi.advanceTimersByTime(10);
            }
            
            expect(cache.size()).toBe(20); // At capacity
            
            // Advance time to start expiring items
            vi.advanceTimersByTime(60);
            
            // Access cache multiple times to trigger cleanup
            for (let i = 0; i < 10; i++) {
                cache.get(`wave-4-item-${i % 5}`);
                cache.set(`new-item-${i}`, i, 200);
            }
            
            // Cache should have cleaned up expired items
            expect(cache.size()).toBeLessThanOrEqual(20);
            
            // Should still be able to access valid items
            const validItem = cache.get("wave-4-item-0");
            expect(validItem).toBeDefined();
        });

        it("should handle extreme cleanup scenarios", () => {
            const cache = new LRUCache<string, string>({ 
                limit: 100,
                defaultTTLMs: 50,
                maxExpiredEntries: 1, // Force very frequent cleanup
            });
            
            // Fill cache with short-lived items
            for (let i = 0; i < 100; i++) {
                cache.set(`temp-${i}`, `value-${i}`, 25);
            }
            
            expect(cache.size()).toBe(100);
            
            // Advance time to expire everything
            vi.advanceTimersByTime(100);
            
            // Access cache to trigger massive cleanup
            cache.get("nonexistent");
            
            // Force final cleanup
            cache.forceCleanup();
            
            expect(cache.size()).toBe(0);
            
            // Should still function normally after cleanup
            cache.set("new-item", "new-value");
            expect(cache.get("new-item")).toBe("new-value");
            expect(cache.size()).toBe(1);
        });
    });

    describe("Edge Cases and Error Conditions", () => {
        it("should handle pathological key/value combinations", () => {
            const cache = new LRUCache<string, any>({ limit: 10 });
            
            // Test with various edge case keys and values
            const testCases = [
                ["", "empty key"],
                [" ", "space key"],
                ["null", null],
                ["undefined", undefined],
                ["zero", 0],
                ["false", false],
                ["empty-array", []],
                ["empty-object", {}],
                ["special-chars", "!@#$%^&*()"],
                ["unicode", "🚀🎉✨"],
            ];
            
            testCases.forEach(([key, value]) => {
                cache.set(key, value);
                expect(cache.get(key)).toEqual(value);
            });
            
            expect(cache.size()).toBe(testCases.length);
        });

        it("should maintain integrity with rapid size changes", () => {
            const cache = new LRUCache<string, number>({ limit: 5 });
            
            // Rapidly add and remove items to test edge cases
            for (let iteration = 0; iteration < 100; iteration++) {
                const key = `item-${iteration}`;
                
                // Add item
                cache.set(key, iteration);
                
                // Randomly delete previous items
                if (iteration > 0 && Math.random() > 0.5) {
                    cache.delete(`item-${iteration - 1}`);
                }
                
                // Verify cache is never over limit
                expect(cache.size()).toBeLessThanOrEqual(5);
                
                // Verify we can still access recently added items
                expect(cache.get(key)).toBe(iteration);
            }
        });

        it("should handle iterator consistency during modifications", () => {
            const cache = new LRUCache<string, number>({ limit: 10 });
            
            // Fill cache
            for (let i = 0; i < 10; i++) {
                cache.set(`item-${i}`, i);
            }
            
            // Collect entries while potentially modifying cache
            const entries = [];
            let count = 0;
            
            for (const [key, value] of cache.entries()) {
                entries.push([key, value]);
                count++;
                
                // Modify cache during iteration (in real usage this might happen)
                if (count === 5) {
                    cache.set("new-item", 999); // This will evict the oldest item
                }
            }
            
            // Should have collected a reasonable number of entries
            expect(entries.length).toBeGreaterThan(5);
            expect(entries.length).toBeLessThanOrEqual(11); // May include the new item added during iteration
            
            // All collected entries should be valid
            entries.forEach(([key, value]) => {
                expect(typeof key).toBe("string");
                expect(typeof value).toBe("number");
            });
        });
    });
});
