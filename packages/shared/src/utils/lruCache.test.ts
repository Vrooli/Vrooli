import { expect } from "chai";
import { LRUCache } from "./lruCache.js";
import sinon from "sinon";

describe("LRUCache", () => {
    let cache: LRUCache<string, number>;
    let consoleWarnStub: sinon.SinonStub;

    before(() => {
        consoleWarnStub = sinon.stub(console, "warn");
    });

    beforeEach(() => {
        // Initialize a new cache before each test with a limit of 3 items
        cache = new LRUCache<string, number>(3);
        consoleWarnStub.resetHistory();
    });

    after(() => {
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
        cache.set("item4", 4); // This should evict 'item1'

        expect(cache.get("item1")).to.be.undefined; // 'item1' should have been evicted
        expect(cache.get("item2")).to.equal(2);
        expect(cache.get("item3")).to.equal(3);
        expect(cache.get("item4")).to.equal(4);
    });

    it("should move the recently accessed item to the end", () => {
        cache.set("item1", 1);
        cache.set("item2", 2);
        cache.set("item3", 3);

        // Access 'item1' so it becomes the most recently used
        expect(cache.get("item1")).to.equal(1);

        // Add another item, which should evict 'item2' instead of 'item1'
        cache.set("item4", 4);

        expect(cache.get("item1")).to.equal(1);
        expect(cache.get("item2")).to.be.undefined;
        expect(cache.get("item3")).to.equal(3);
        expect(cache.get("item4")).to.equal(4);
    });

    it("should handle a large number of items", () => {
        const largeCache = new LRUCache<string, number>(1000);
        for (let i = 0; i < 2000; i++) {
            largeCache.set("item" + i, i);
        }

        // The cache should not grow beyond 1000 items
        expect(largeCache.size()).to.equal(1000);

        // The first 1000 items should be evicted
        for (let i = 0; i < 1000; i++) {
            expect(largeCache.get("item" + i)).to.be.undefined;
        }

        // The last 1000 items should be present
        for (let i = 1000; i < 2000; i++) {
            expect(largeCache.get("item" + i)).to.equal(i);
        }
    });

    it("should not add items over a certain size", () => {
        // Initialize a cache with a max size of 10 bytes
        const smallCache = new LRUCache<string, string>(3, 10);

        smallCache.set("smallItem", "small"); // This should be added
        smallCache.set("largeItem", "This is a large item"); // This should be skipped

        expect(smallCache.get("smallItem")).to.equal("small");
        expect(smallCache.get("largeItem")).to.be.undefined;
    });

    it("should evict the least recently used item, respecting size limits", () => {
        // Initialize a cache with a max size of 10 bytes
        const smallCache = new LRUCache<string, string>(3, 10);

        smallCache.set("item1", "12345"); // 5 bytes
        smallCache.set("item2", "1234567890"); // 10 bytes, at the limit
        smallCache.set("item3", "12345"); // 5 bytes, 'item1' should be evicted next if needed

        // Access 'item2' to make it recently used
        expect(smallCache.get("item2")).to.equal("1234567890");

        // Attempt to add an item that would exceed the size limit
        smallCache.set("largeItem", "This is a large item"); // This should be skipped

        // 'item1' should still be evicted as 'largeItem' wasn't added
        smallCache.set("item4", "12345"); // 'item1' should be evicted

        expect(smallCache.get("item1")).to.be.undefined;
        expect(smallCache.get("item2")).to.equal("1234567890");
        expect(smallCache.get("item3")).to.equal("12345");
        expect(smallCache.get("item4")).to.equal("12345");
        expect(smallCache.get("largeItem")).to.be.undefined; // 'largeItem' was never added
    });
});
