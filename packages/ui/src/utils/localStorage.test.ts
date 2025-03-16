/* eslint-disable @typescript-eslint/ban-ts-comment */
import { noop } from "@local/shared";
import { expect } from "chai";
import sinon from "sinon";
import { LocalStorageLruCache, cookies, getCookie, getLocalStorageKeys, getOrSetCookie, getStorageItem, ifAllowed, setCookie } from "./localStorage.js";

describe("getLocalStorageKeys", () => {
    beforeEach(() => {
        jest.clearAllMocks();
        global.localStorage.clear();
    });
    after(() => {
        global.localStorage.clear();
        jest.restoreAllMocks();
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
        expect(keysWithPrefixAndSuffix).to.deep.equal(["myPrefix_testKey4_mySuffix"]);
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
        expect(keysWithPrefix).to.deep.equal(["prefix_key1", "prefix_key3"]);
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
        expect(keysWithSuffix).to.deep.equal(["key1_suffix", "key3_suffix"]);
    });

    it("should return all keys if no prefix or suffix is specified", () => {
        // Set up some keys
        localStorage.setItem("key1", "value1");
        localStorage.setItem("key2", "value2");
        localStorage.setItem("key3", "value3");

        // Call getLocalStorageKeys with no prefix or suffix
        const allKeys = getLocalStorageKeys({});

        // Expect to find all keys
        expect(allKeys).to.deep.equal(["key1", "key2", "key3"]);
    });
});


describe("LocalStorageLruCache", () => {
    let consoleWarnStub: sinon.SinonStub;

    before(() => {
        consoleWarnStub = sinon.stub(console, "warn");
    });

    beforeEach(() => {
        consoleWarnStub.resetHistory();
        global.localStorage.clear();
    });

    after(() => {
        consoleWarnStub.restore();
        global.localStorage.clear();
    });

    it("should store and retrieve an item", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        expect(cache.get("key1")).to.equal("value1");
    });

    it("should overwrite the least recently used item when limit is reached", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        cache.set("key2", "value2");
        cache.set("key3", "value3"); // This should evict key1

        expect(cache.get("key1")).to.be.undefined;
        expect(cache.get("key2")).to.equal("value2");
        expect(cache.get("key3")).to.equal("value3");
    });

    it("should update item's position when accessed", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        cache.set("key2", "value2");
        cache.get("key1"); // Access key1 to move it to the front
        cache.set("key3", "value3"); // This should evict key2

        expect(cache.get("key1")).to.equal("value1");
        expect(cache.get("key2")).to.be.undefined;
        expect(cache.get("key3")).to.equal("value3");
    });

    it("should not store items that exceed maxSize", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2, 10); // maxSize of 10 bytes
        cache.set("key1", "value1"); // This should be stored
        cache.set("key2", "this is a very long string"); // This should not be stored

        expect(cache.get("key1")).to.equal("value1");
        expect(cache.get("key2")).to.be.undefined;
    });

    it("should correctly report the size", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 3);
        expect(cache.size()).to.equal(0);

        cache.set("key1", "value1");
        cache.set("key2", "value2");
        expect(cache.size()).to.equal(2);

        cache.set("key3", "value3");
        cache.set("key4", "value4"); // This should evict key1
        expect(cache.size()).to.equal(3);
    });

    it("should remove an item", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        cache.set("key2", "value2");

        expect(cache.get("key1")).to.equal("value1"); // Ensure the item is there before removal
        cache.remove("key1");
        expect(cache.get("key1")).to.be.undefined; // The item should be gone after removal
        expect(cache.size()).to.equal(1); // Size should be decremented
    });

    it("removing a non-existent item doesn't affect the cache", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");

        expect(cache.size()).to.equal(1); // Initial size check
        cache.remove("nonExistentKey"); // Attempt to remove a key that doesn't exist
        expect(cache.size()).to.equal(1); // Size should remain unchanged
        expect(cache.get("key1")).to.equal("value1"); // Other items should remain unaffected
    });

    it("should store and retrieve non-string values", () => {
        const cache = new LocalStorageLruCache<any>("cache1", 3);

        const numValue = 123;
        const objValue = { a: 1, b: "Test" };
        const arrValue = [1, "two", { three: 3 }];

        cache.set("numKey", numValue);
        cache.set("objKey", objValue);
        cache.set("arrKey", arrValue);

        expect(cache.get("numKey")).to.deep.equal(numValue);
        expect(cache.get("objKey")).to.deep.equal(objValue);
        expect(cache.get("arrKey")).to.deep.equal(arrValue);
    });

    it("namespaces should not overwrite each other", () => {
        const cache1 = new LocalStorageLruCache<string>("namespace1", 2);
        const cache2 = new LocalStorageLruCache<string>("namespace2", 2);

        cache1.set("sharedKey", "valueFromCache1");
        cache2.set("sharedKey", "valueFromCache2");

        expect(cache1.get("sharedKey")).to.equal("valueFromCache1");
        expect(cache2.get("sharedKey")).to.equal("valueFromCache2");
    });

    it("namespaces should not affect each other's sizes", () => {
        const cache1 = new LocalStorageLruCache<string>("namespace1", 2);
        const cache2 = new LocalStorageLruCache<string>("namespace2", 3);

        cache1.set("key1", "value1");
        cache1.set("key2", "value2");

        cache2.set("keyA", "valueA");
        cache2.set("keyB", "valueB");
        cache2.set("keyC", "valueC");

        expect(cache1.size()).to.equal(2);
        expect(cache2.size()).to.equal(3);
    });

    it("evicting items in one namespace should not affect the other", () => {
        const cache1 = new LocalStorageLruCache<string>("namespace1", 2);
        const cache2 = new LocalStorageLruCache<string>("namespace2", 2);

        cache1.set("key1", "value1");
        cache1.set("key2", "value2");

        cache2.set("keyA", "valueA");
        cache2.set("keyB", "valueB");
        cache2.set("keyC", "valueC"); // This should evict keyA in namespace2

        expect(cache1.get("key1")).to.equal("value1"); // Still present in namespace1
        expect(cache2.get("keyA")).to.be.undefined; // Evicted from namespace2
        expect(cache2.get("keyB")).to.equal("valueB");
        expect(cache2.get("keyC")).to.equal("valueC");
    });

    it("should remove keys with specific value using predicate", () => {
        const cache = new LocalStorageLruCache<{ chatId: string }>("cache1", 5);

        cache.set("key1", { chatId: "chat1" });
        cache.set("key2", { chatId: "chat2" });
        cache.set("key3", { chatId: "chat1" });
        cache.set("key4", { chatId: "chat3" });

        expect(cache.size()).to.equal(4);

        // Remove keys with chatId "chat1"
        cache.removeKeysWithValue((key, value) => value.chatId === "chat1");

        expect(cache.size()).to.equal(2);
        expect(cache.get("key1")).to.be.undefined;
        expect(cache.get("key3")).to.be.undefined;
        expect(cache.get("key2")).to.deep.equal({ chatId: "chat2" });
        expect(cache.get("key4")).to.deep.equal({ chatId: "chat3" });
    });

    it("removing keys with non-matching predicate doesn't affect the cache", () => {
        const cache = new LocalStorageLruCache<{ chatId: string }>("cache1", 5);

        cache.set("key1", { chatId: "chat1" });
        cache.set("key2", { chatId: "chat2" });

        expect(cache.size()).to.equal(2);

        // Attempt to remove keys with chatId "chat3" which doesn't exist
        cache.removeKeysWithValue((key, value) => value.chatId === "chat3");

        expect(cache.size()).to.equal(2);
        expect(cache.get("key1")).to.deep.equal({ chatId: "chat1" });
        expect(cache.get("key2")).to.deep.equal({ chatId: "chat2" });
    });

});

describe("cookies (local storage)", () => {
    before(() => {
        jest.spyOn(console, "warn").mockImplementation(noop);
    });
    beforeEach(() => {
        jest.clearAllMocks();
        global.localStorage.clear();
        setCookie("Preferences", {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        });
    });
    after(() => {
        global.localStorage.clear();
        jest.restoreAllMocks();
    });

    describe("getCookie", () => {
        it("should retrieve a stored value correctly", () => {
            const testKey = "Theme";
            const testValue = "light";
            setCookie(testKey, testValue);
            const result = getCookie(testKey);
            expect(result).to.deep.equal(testValue);
        });

        it("should return undefined for non-existent keys", () => {
            // @ts-ignore: Testing runtime scenario
            const result = getCookie("nonExistentKey");
            expect(result).to.be.undefined;
        });

        it("should handle incorrect types gracefully", () => {
            const testKey = "Theme";
            const testValue = { beep: "boop" }; // Should be "light" or "dark"
            // @ts-ignore: Testing runtime scenario
            setCookie(testKey, testValue);
            const result = getCookie(testKey);
            expect(result).to.equal(cookies[testKey].fallback);
        });

        it("should console.warn on JSON parse failure", () => {
            const testKey = "malformedJson";
            global.localStorage.setItem(testKey, "this is not JSON");
            getStorageItem(testKey, (value): value is object => typeof value === "object");
            expect(console.warn).toHaveBeenCalledWith(`Failed to parse cookie ${testKey}`, "this is not JSON");
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
            expect(storedValue).to.deep.equal(testValue);
        });

        it("should overwrite existing values", () => {
            const testKey = "Theme";
            setCookie(testKey, "light");
            setCookie(testKey, "dark");
            const storedValue = JSON.parse(global.localStorage.getItem(testKey) ?? "");
            expect(storedValue).to.deep.equal("dark");
        });
    });

    describe("getOrSetCookie", () => {
        it("should retrieve an existing cookie", () => {
            const testKey = "Theme";

            setCookie(testKey, "light");
            const result1 = getOrSetCookie(testKey, cookies[testKey].check, cookies[testKey].fallback);
            expect(result1).to.deep.equal("light");

            setCookie(testKey, "dark");
            const result2 = getOrSetCookie(testKey, cookies[testKey].check, cookies[testKey].fallback);
            expect(result2).to.deep.equal("dark");
        });

        it("should set the cookie to the default value if it does not exist", () => {
            const result = getOrSetCookie("defaultCookie", (value): value is number => typeof value === "number", 42);
            expect(result).to.deep.equal(42);
            expect(localStorage.getItem("defaultCookie")).to.deep.equal(JSON.stringify(42));
        });

        it("should set the cookie to the default value if the existing cookie fails the type check", () => {
            const testKey = "Theme";
            // @ts-ignore: Testing runtime scenario
            setCookie(testKey, 22);
            const result = getOrSetCookie(testKey, cookies[testKey].check, cookies[testKey].fallback);
            expect(result).to.deep.equal(cookies[testKey].fallback);
            expect(localStorage.getItem(testKey)).to.deep.equal(JSON.stringify(cookies[testKey].fallback));
        });
    });

    describe("ifAllowed", () => {
        it("calls the callback for strictly necessary cookies regardless of preferences", () => {
            const callback = jest.fn();
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
            const callback = jest.fn();
            setCookie("Preferences", {
                strictlyNecessary: true,
                performance: true,
                functional: false,
                targeting: false,
            });
            ifAllowed("performance", callback, "fallback");
            expect(callback).toHaveBeenCalled();
        });

        it("does not call the callback and returns fallback for disallowed cookie types", () => {
            const callback = jest.fn();
            setCookie("Preferences", {
                strictlyNecessary: true,
                performance: false,
                functional: false,
                targeting: false,
            });
            const result = ifAllowed("functional", callback, "fallback");
            expect(callback).not.toHaveBeenCalled();
            expect(result).to.equal("fallback");
        });
    });
});
