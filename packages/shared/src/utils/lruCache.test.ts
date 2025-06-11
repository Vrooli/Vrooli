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
            expect.stringContaining("Skipping cache set for key largeObj")
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
            expect.stringContaining("Error calculating size for cache value")
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
            ["third", 3]
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
            defaultTTLMs: 100
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
            maxExpiredEntries: 2 // Force frequent cleanup
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
            defaultTTLMs: 100
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
});
