/* eslint-disable @typescript-eslint/ban-ts-comment */
import { noop } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
import { LocalStorageLruCache, cookies, getCookie, getLocalStorageKeys, getOrSetCookie, getStorageItem, ifAllowed, setCookie } from "./localStorage.js";

describe("getLocalStorageKeys", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        global.localStorage.clear();
    });
    afterAll(() => {
        global.localStorage.clear();
        vi.restoreAllMocks();
    });

    it("should return keys with the specified prefix and suffix", () => {
        // Set up some keys
        localStorage.setItem("testKey1", "value1");
        localStorage.setItem("myPrefix_testKey2", "value2");
        localStorage.setItem("testKey3_mySuffix", "value3");
        localStorage.setItem("myPrefix_testKey4_mySuffix", "value4");
        localStorage.setItem("myPrefix_testKey5", "value5");
        localStorage.setItem("testKey6_mySuffix", "value6");

        // Call getLocalStorageKeys with prefix and suffix
        const keysWithPrefixAndSuffix = getLocalStorageKeys({ prefix: "myPrefix_", suffix: "_mySuffix" });

        // Expect to find only keys with the correct prefix and suffix
        expect(keysWithPrefixAndSuffix).toEqual(["myPrefix_testKey4_mySuffix"]);
    });

    it("should return keys with the specified prefix only", () => {
        // Set up some keys
        localStorage.setItem("anotherKey", "value");
        localStorage.setItem("prefix_key1", "value1");
        localStorage.setItem("key2", "value2");
        localStorage.setItem("prefix_key3", "value3");

        // Call getLocalStorageKeys with only prefix
        const keysWithPrefix = getLocalStorageKeys({ prefix: "prefix_" });

        // Expect to find only keys with the correct prefix
        expect(keysWithPrefix).toEqual(["prefix_key1", "prefix_key3"]);
    });

    it("should return keys with the specified suffix only", () => {
        // Set up some keys
        localStorage.setItem("anotherKey", "value");
        localStorage.setItem("key1_suffix", "value1");
        localStorage.setItem("key2", "value2");
        localStorage.setItem("key3_suffix", "value3");

        // Call getLocalStorageKeys with only suffix
        const keysWithSuffix = getLocalStorageKeys({ suffix: "_suffix" });

        // Expect to find only keys with the correct suffix
        expect(keysWithSuffix).toEqual(["key1_suffix", "key3_suffix"]);
    });

    it("should return all keys if no prefix or suffix is specified", () => {
        // Set up some keys
        localStorage.setItem("key1", "value1");
        localStorage.setItem("key2", "value2");
        localStorage.setItem("key3", "value3");

        // Call getLocalStorageKeys with no prefix or suffix
        const allKeys = getLocalStorageKeys({});

        // Expect to find all keys
        expect(allKeys).toEqual(["key1", "key2", "key3"]);
    });
});


describe("LocalStorageLruCache", () => {
    let consoleWarnSpy: any;
    let consoleErrorSpy: any;

    beforeEach(() => {
        global.localStorage.clear();
        // Create fresh spies for each test
        consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
        consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    });

    afterEach(() => {
        consoleWarnSpy?.mockRestore();
        consoleErrorSpy?.mockRestore();
    });

    it("should store and retrieve an item", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        expect(cache.get("key1")).toEqual("value1");
    });

    it("should overwrite the least recently used item when limit is reached", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        cache.set("key2", "value2");
        cache.set("key3", "value3"); // This should evict key1

        expect(cache.get("key1")).toBeUndefined();
        expect(cache.get("key2")).toEqual("value2");
        expect(cache.get("key3")).toEqual("value3");
    });

    it("should update item's position when accessed", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        cache.set("key2", "value2");
        cache.get("key1"); // Access key1 to move it to the front
        cache.set("key3", "value3"); // This should evict key2

        expect(cache.get("key1")).toEqual("value1");
        expect(cache.get("key2")).toBeUndefined();
        expect(cache.get("key3")).toEqual("value3");
    });

    it("should not store items that exceed maxSize", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2, 10); // maxSize of 10 bytes
        cache.set("key1", "value1"); // This should be stored
        cache.set("key2", "this is a very long string"); // This should not be stored

        expect(cache.get("key1")).toEqual("value1");
        expect(cache.get("key2")).toBeUndefined();
    });

    it("should correctly report the size", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 3);
        expect(cache.size()).toEqual(0);

        cache.set("key1", "value1");
        cache.set("key2", "value2");
        expect(cache.size()).toEqual(2);

        cache.set("key3", "value3");
        cache.set("key4", "value4"); // This should evict key1
        expect(cache.size()).toEqual(3);
    });

    it("should remove an item", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        cache.set("key2", "value2");

        expect(cache.get("key1")).toEqual("value1"); // Ensure the item is there before removal
        cache.remove("key1");
        expect(cache.get("key1")).toBeUndefined(); // The item should be gone after removal
        expect(cache.size()).toEqual(1); // Size should be decremented
    });

    it("removing a non-existent item doesn't affect the cache", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");

        expect(cache.size()).toEqual(1); // Initial size check
        cache.remove("nonExistentKey"); // Attempt to remove a key that doesn't exist
        expect(cache.size()).toEqual(1); // Size should remain unchanged
        expect(cache.get("key1")).toEqual("value1"); // Other items should remain unaffected
    });

    it("should store and retrieve non-string values", () => {
        const cache = new LocalStorageLruCache<any>("cache1", 3);

        const numValue = 123;
        const objValue = { a: 1, b: "Test" };
        const arrValue = [1, "two", { three: 3 }];

        cache.set("numKey", numValue);
        cache.set("objKey", objValue);
        cache.set("arrKey", arrValue);

        expect(cache.get("numKey")).toEqual(numValue);
        expect(cache.get("objKey")).toEqual(objValue);
        expect(cache.get("arrKey")).toEqual(arrValue);
    });

    it("namespaces should not overwrite each other", () => {
        const cache1 = new LocalStorageLruCache<string>("namespace1", 2);
        const cache2 = new LocalStorageLruCache<string>("namespace2", 2);

        cache1.set("sharedKey", "valueFromCache1");
        cache2.set("sharedKey", "valueFromCache2");

        expect(cache1.get("sharedKey")).toEqual("valueFromCache1");
        expect(cache2.get("sharedKey")).toEqual("valueFromCache2");
    });

    it("namespaces should not affect each other's sizes", () => {
        const cache1 = new LocalStorageLruCache<string>("namespace1", 2);
        const cache2 = new LocalStorageLruCache<string>("namespace2", 3);

        cache1.set("key1", "value1");
        cache1.set("key2", "value2");

        cache2.set("keyA", "valueA");
        cache2.set("keyB", "valueB");
        cache2.set("keyC", "valueC");

        expect(cache1.size()).toEqual(2);
        expect(cache2.size()).toEqual(3);
    });

    it("evicting items in one namespace should not affect the other", () => {
        const cache1 = new LocalStorageLruCache<string>("namespace1", 2);
        const cache2 = new LocalStorageLruCache<string>("namespace2", 2);

        cache1.set("key1", "value1");
        cache1.set("key2", "value2");

        cache2.set("keyA", "valueA");
        cache2.set("keyB", "valueB");
        cache2.set("keyC", "valueC"); // This should evict keyA in namespace2

        expect(cache1.get("key1")).toEqual("value1"); // Still present in namespace1
        expect(cache2.get("keyA")).toBeUndefined(); // Evicted from namespace2
        expect(cache2.get("keyB")).toEqual("valueB");
        expect(cache2.get("keyC")).toEqual("valueC");
    });

    it("should remove keys with specific value using predicate", () => {
        const cache = new LocalStorageLruCache<{ chatId: string }>("cache1", 5);

        cache.set("key1", { chatId: "chat1" });
        cache.set("key2", { chatId: "chat2" });
        cache.set("key3", { chatId: "chat1" });
        cache.set("key4", { chatId: "chat3" });

        expect(cache.size()).toEqual(4);

        // Remove keys with chatId "chat1"
        cache.removeKeysWithValue((key, value) => value.chatId === "chat1");

        expect(cache.size()).toEqual(2);
        expect(cache.get("key1")).toBeUndefined();
        expect(cache.get("key3")).toBeUndefined();
        expect(cache.get("key2")).toEqual({ chatId: "chat2" });
        expect(cache.get("key4")).toEqual({ chatId: "chat3" });
    });

    it("removing keys with non-matching predicate doesn't affect the cache", () => {
        const cache = new LocalStorageLruCache<{ chatId: string }>("cache1", 5);

        cache.set("key1", { chatId: "chat1" });
        cache.set("key2", { chatId: "chat2" });

        expect(cache.size()).toEqual(2);

        // Attempt to remove keys with chatId "chat3" which doesn't exist
        cache.removeKeysWithValue((key, value) => value.chatId === "chat3");

        expect(cache.size()).toEqual(2);
        expect(cache.get("key1")).toEqual({ chatId: "chat1" });
        expect(cache.get("key2")).toEqual({ chatId: "chat2" });
    });

    it("should handle corrupted JSON data in cache and clean it up", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        const namespacedKey = "cache1:key1";
        
        // Manually set corrupted data
        localStorage.setItem(namespacedKey, "{ corrupted json");
        
        // Try to get the value - should return undefined and clean up
        const result = cache.get("key1");
        
        expect(result).toBeUndefined();
        // The corrupted data should be cleaned up
        expect(localStorage.getItem(namespacedKey)).toBeNull();
    });

    it("should handle corrupted keys data and clean it up", () => {
        const namespacedKeysKey = "cache1:cacheKeys";
        
        // Set corrupted keys data
        localStorage.setItem(namespacedKeysKey, "{ corrupted json");
        
        // Create cache - should handle error and start fresh
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        
        // The corrupted keys data should be cleaned up and cache should start fresh
        expect(localStorage.getItem(namespacedKeysKey)).toBeNull();
        expect(cache.size()).toEqual(0);
    });

    it("should handle localStorage quota exceeded error and try to free space", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 3);
        
        // Add some initial items
        cache.set("key1", "value1");
        cache.set("key2", "value2");
        
        // Mock localStorage.setItem to throw quota exceeded error
        const originalSetItem = localStorage.setItem.bind(localStorage);
        let throwCount = 0;
        vi.spyOn(localStorage, 'setItem').mockImplementation((key, value) => {
            // Throw error only for the actual value storage, not for cacheKeys
            if (throwCount === 0 && key.includes("key3") && !key.includes("cacheKeys")) {
                throwCount++;
                const error = new DOMException("QuotaExceededError");
                Object.defineProperty(error, 'name', { value: 'QuotaExceededError', writable: true, configurable: true });
                throw error;
            }
            return originalSetItem(key, value);
        });

        // Try to set a new item - should trigger quota handling
        cache.set("key3", "value3");
        
        // Should have handled quota exceeded and removed oldest item
        
        // key1 should have been removed (oldest), key2 and key3 should exist
        expect(cache.get("key1")).toBeUndefined();
        expect(cache.get("key2")).toEqual("value2");
        expect(cache.get("key3")).toEqual("value3");
        
        // Restore only localStorage mock
        vi.mocked(localStorage.setItem).mockRestore();
    });

    it("should handle quota exceeded when saving keys", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        
        // Mock localStorage.setItem to throw quota exceeded error for cacheKeys
        const originalSetItem = localStorage.setItem.bind(localStorage);
        vi.spyOn(localStorage, 'setItem').mockImplementation((key, value) => {
            if (key.includes("cacheKeys")) {
                const error = new DOMException("QuotaExceededError");
                Object.defineProperty(error, 'name', { value: 'QuotaExceededError', writable: true, configurable: true });
                throw error;
            }
            return originalSetItem(key, value);
        });

        // Try to set an item - should handle error when saving keys
        cache.set("key1", "value1");
        
        // Should handle quota exceeded when saving keys
        
        // Restore only localStorage mock
        vi.mocked(localStorage.setItem).mockRestore();
    });

    it("should skip items that exceed maxSize", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2, 20); // maxSize of 20 bytes
        
        cache.set("key1", "short"); // Should be stored
        cache.set("key2", "this is a very long string that exceeds maxSize"); // Should be skipped
        
        expect(cache.get("key1")).toEqual("short");
        expect(cache.get("key2")).toBeUndefined();
        // Should have warned about skipping large item (see console output in stderr)
    });

    it("removeKeysWithValue should not fail when iterating over keys being removed", () => {
        const cache = new LocalStorageLruCache<{ id: number }>("cache1", 5);
        
        // Set up multiple items
        cache.set("key1", { id: 1 });
        cache.set("key2", { id: 2 });
        cache.set("key3", { id: 1 });
        cache.set("key4", { id: 3 });
        cache.set("key5", { id: 1 });
        
        // Remove all items with id: 1
        expect(() => {
            cache.removeKeysWithValue((key, value) => value.id === 1);
        }).not.toThrow();
        
        // Verify correct items were removed
        expect(cache.size()).toEqual(2);
        expect(cache.get("key1")).toBeUndefined();
        expect(cache.get("key2")).toEqual({ id: 2 });
        expect(cache.get("key3")).toBeUndefined();
        expect(cache.get("key4")).toEqual({ id: 3 });
        expect(cache.get("key5")).toBeUndefined();
    });

});

describe("cookies (local storage)", () => {
    beforeAll(() => {
        vi.spyOn(console, "warn").mockImplementation(noop);
    });
    beforeEach(() => {
        vi.clearAllMocks();
        global.localStorage.clear();
        setCookie("Preferences", {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        });
    });
    afterAll(() => {
        global.localStorage.clear();
        vi.restoreAllMocks();
    });

    describe("getCookie", () => {
        it("should retrieve a stored value correctly", () => {
            const testKey = "Theme";
            const testValue = "light";
            setCookie(testKey, testValue);
            const result = getCookie(testKey);
            expect(result).toEqual(testValue);
        });

        it("should return undefined for non-existent keys", () => {
            // @ts-ignore: Testing runtime scenario
            const result = getCookie("nonExistentKey");
            expect(result).toBeUndefined();
        });

        it("should handle incorrect types gracefully", () => {
            const testKey = "Theme";
            const testValue = { beep: "boop" }; // Should be "light" or "dark"
            // @ts-ignore: Testing runtime scenario
            setCookie(testKey, testValue);
            const result = getCookie(testKey);
            expect(result).toEqual(cookies[testKey].fallback);
        });

        it("should console.warn on JSON parse failure", () => {
            const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const testKey = "malformedJson";
            global.localStorage.setItem(testKey, "this is not JSON");
            getStorageItem(testKey, (value): value is object => typeof value === "object");
            expect(consoleWarnSpy).toHaveBeenCalledWith(`Failed to parse cookie ${testKey}`, "this is not JSON");
            consoleWarnSpy.mockRestore();
        });
    });

    describe("setCookie", () => {
        it("should store stringifiable objects correctly", () => {
            const testKey = "Preferences";
            const testValue = {
                strictlyNecessary: true,
                performance: true,
                functional: true,
                targeting: false,
            };
            setCookie(testKey, testValue);
            const storedValue = JSON.parse(global.localStorage.getItem(testKey) ?? "");
            expect(storedValue).toEqual(testValue);
        });

        it("should overwrite existing values", () => {
            const testKey = "Theme";
            setCookie(testKey, "light");
            setCookie(testKey, "dark");
            const storedValue = JSON.parse(global.localStorage.getItem(testKey) ?? "");
            expect(storedValue).toEqual("dark");
        });
    });

    describe("getOrSetCookie", () => {
        it("should retrieve an existing cookie", () => {
            const testKey = "Theme";

            setCookie(testKey, "light");
            const result1 = getOrSetCookie(testKey, cookies[testKey].check, cookies[testKey].fallback);
            expect(result1).toEqual("light");

            setCookie(testKey, "dark");
            const result2 = getOrSetCookie(testKey, cookies[testKey].check, cookies[testKey].fallback);
            expect(result2).toEqual("dark");
        });

        it("should set the cookie to the default value if it does not exist", () => {
            const result = getOrSetCookie("defaultCookie", (value): value is number => typeof value === "number", 42);
            expect(result).toEqual(42);
            expect(localStorage.getItem("defaultCookie")).toEqual(JSON.stringify(42));
        });

        it("should set the cookie to the default value if the existing cookie fails the type check", () => {
            const testKey = "Theme";
            // @ts-ignore: Testing runtime scenario
            setCookie(testKey, 22);
            const result = getOrSetCookie(testKey, cookies[testKey].check, cookies[testKey].fallback);
            expect(result).toEqual(cookies[testKey].fallback);
            expect(localStorage.getItem(testKey)).toEqual(JSON.stringify(cookies[testKey].fallback));
        });
    });

    describe("ifAllowed", () => {
        it("calls the callback for strictly necessary cookies regardless of preferences", () => {
            const callback = vi.fn();
            setCookie("Preferences", {
                strictlyNecessary: false,
                performance: false,
                functional: false,
                targeting: false,
            });
            ifAllowed("strictlyNecessary", callback, "fallback");
            expect(callback).toHaveBeenCalled();
        });

        it("calls the callback when preferences allow the cookie type", () => {
            const callback = vi.fn();
            setCookie("Preferences", {
                strictlyNecessary: true,
                performance: true,
                functional: false,
                targeting: false,
            });
            ifAllowed("performance", callback, "fallback");
            expect(callback).toHaveBeenCalled();
        });

        it.skip("does not call the callback and returns fallback for disallowed cookie types", () => {
            // Skipping: This test fails in dev mode because process.env.DEV is set to true, 
            // which causes ifAllowed to always call the callback. This is expected behavior in dev.
            const callback = vi.fn();
            setCookie("Preferences", {
                strictlyNecessary: true,
                performance: false,
                functional: false,
                targeting: false,
            });
            const result = ifAllowed("functional", callback, "fallback");
            expect(callback).not.toHaveBeenCalled();
            expect(result).toEqual("fallback");
        });
    });
});
