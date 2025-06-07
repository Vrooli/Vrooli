import { describe, it, expect, beforeAll, afterAll, beforeEach, afterEach } from "vitest";
import sinon from "sinon";
import { LRUCache } from "./lruCache.js";

describe("LRUCache", () => {
    let cache: LRUCache<string, number>;
    let consoleWarnStub: sinon.SinonStub;
    let clock: sinon.SinonFakeTimers;

    beforeAll(() => {
        consoleWarnStub = sinon.stub(console, "warn");
    });

    beforeEach(() => {
        // Fake timers for TTL tests (won't affect sync tests)
        clock = sinon.useFakeTimers();
        cache = new LRUCache({ limit: 3 });
        consoleWarnStub.resetHistory();
    });

    afterEach(() => {
        clock.restore();
    });

    afterAll(() => {
        consoleWarnStub.restore();
    });

    it("should add items to the cache", () => {
        cache.set("item1", 1);
        expect(cache.get("item1")).to.equal(1);
    });

    it("should return undefined for missing items", () => {
        expect(cache.get("missingItem")).to.be.undefined;
    });

    it("should update the value if the same key is added to the cache", () => {
        cache.set("item", 1);
        cache.set("item", 2);
        expect(cache.get("item")).to.equal(2);
    });

    it("should maintain the max limit by evicting the least recently used item", () => {
        cache.set("item1", 1);
        cache.set("item2", 2);
        cache.set("item3", 3);
        cache.set("item4", 4); // should evict 'item1'

        expect(cache.get("item1")).to.be.undefined;
        expect(cache.get("item2")).to.equal(2);
        expect(cache.get("item3")).to.equal(3);
        expect(cache.get("item4")).to.equal(4);
    });

    it("should move the recently accessed item to the end", () => {
        cache.set("item1", 1);
        cache.set("item2", 2);
        cache.set("item3", 3);

        expect(cache.get("item1")).to.equal(1); // now most recently used

        cache.set("item4", 4); // should evict 'item2'
        expect(cache.get("item1")).to.equal(1);
        expect(cache.get("item2")).to.be.undefined;
        expect(cache.get("item3")).to.equal(3);
        expect(cache.get("item4")).to.equal(4);
    });

    it("should handle a large number of items", () => {
        const largeCache = new LRUCache<string, number>({ limit: 1000 });
        for (let i = 0; i < 2000; i++) {
            largeCache.set("item" + i, i);
        }

        expect(largeCache.size()).to.equal(1000);

        for (let i = 0; i < 1000; i++) {
            expect(largeCache.get("item" + i)).to.be.undefined;
        }
        for (let i = 1000; i < 2000; i++) {
            expect(largeCache.get("item" + i)).to.equal(i);
        }
    });

    it("should not add items over a certain size", () => {
        const smallCache = new LRUCache<string, string>({ limit: 3, maxSizeBytes: 10 });

        smallCache.set("smallItem", "small"); // ~5 bytes
        smallCache.set("largeItem", "This is a large item"); // >10 bytes, skipped

        expect(smallCache.get("smallItem")).to.equal("small");
        expect(smallCache.get("largeItem")).to.be.undefined;
    });

    it("should evict the least recently used item, respecting size limits", () => {
        const smallCache = new LRUCache<string, string>({ limit: 3, maxSizeBytes: 10 });

        smallCache.set("item1", "12345");        // 5 bytes
        smallCache.set("item2", "1234567890");   // 10 bytes
        smallCache.set("item3", "12345");        // 5 bytes

        expect(smallCache.get("item2")).to.equal("1234567890"); // mark used

        smallCache.set("largeItem", "This is a large item"); // skipped

        smallCache.set("item4", "12345");        // should evict 'item1'

        expect(smallCache.get("item1")).to.be.undefined;
        expect(smallCache.get("item2")).to.equal("1234567890");
        expect(smallCache.get("item3")).to.equal("12345");
        expect(smallCache.get("item4")).to.equal("12345");
        expect(smallCache.get("largeItem")).to.be.undefined;
    });

    // --- TTL-specific tests ---

    it("should not expire entries without TTL", () => {
        cache.set("perm", 7);
        clock.tick(10_000);
        expect(cache.get("perm")).to.equal(7);
    });

    it("should expire entries after the default TTL", () => {
        const ttlCache = new LRUCache<string, number>({ limit: 3, defaultTTLMs: 100 });
        ttlCache.set("temp", 123);
        expect(ttlCache.get("temp")).to.equal(123);
        clock.tick(101);
        expect(ttlCache.get("temp")).to.be.undefined;
    });

    it("should expire entries after a per-entry TTL override", () => {
        const ttlCache = new LRUCache<string, number>({ limit: 3 });
        ttlCache.set("override", 456, 50);
        expect(ttlCache.get("override")).to.equal(456);
        clock.tick(51);
        expect(ttlCache.get("override")).to.be.undefined;
    });
});
