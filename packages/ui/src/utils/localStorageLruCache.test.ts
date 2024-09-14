import { noop } from "@local/shared";
import { LocalStorageLruCache } from "./localStorageLruCache";

describe("LocalStorageLruCache", () => {
    beforeAll(() => {
        jest.spyOn(console, "warn").mockImplementation(noop);
    });
    beforeEach(() => {
        jest.clearAllMocks();
        global.localStorage.clear();
    });
    afterAll(() => {
        global.localStorage.clear();
        jest.restoreAllMocks();
    });

    test("should store and retrieve an item", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        expect(cache.get("key1")).toBe("value1");
    });

    test("should overwrite the least recently used item when limit is reached", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        cache.set("key2", "value2");
        cache.set("key3", "value3"); // This should evict key1

        expect(cache.get("key1")).toBeUndefined();
        expect(cache.get("key2")).toBe("value2");
        expect(cache.get("key3")).toBe("value3");
    });

    test("should update item's position when accessed", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        cache.set("key2", "value2");
        cache.get("key1"); // Access key1 to move it to the front
        cache.set("key3", "value3"); // This should evict key2

        expect(cache.get("key1")).toBe("value1");
        expect(cache.get("key2")).toBeUndefined();
        expect(cache.get("key3")).toBe("value3");
    });

    test("should not store items that exceed maxSize", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2, 10); // maxSize of 10 bytes
        cache.set("key1", "value1"); // This should be stored
        cache.set("key2", "this is a very long string"); // This should not be stored

        expect(cache.get("key1")).toBe("value1");
        expect(cache.get("key2")).toBeUndefined();
    });

    test("should correctly report the size", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 3);
        expect(cache.size()).toBe(0);

        cache.set("key1", "value1");
        cache.set("key2", "value2");
        expect(cache.size()).toBe(2);

        cache.set("key3", "value3");
        cache.set("key4", "value4"); // This should evict key1
        expect(cache.size()).toBe(3);
    });

    test("should remove an item", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");
        cache.set("key2", "value2");

        expect(cache.get("key1")).toBe("value1"); // Ensure the item is there before removal
        cache.remove("key1");
        expect(cache.get("key1")).toBeUndefined(); // The item should be gone after removal
        expect(cache.size()).toBe(1); // Size should be decremented
    });

    test("removing a non-existent item doesn't affect the cache", () => {
        const cache = new LocalStorageLruCache<string>("cache1", 2);
        cache.set("key1", "value1");

        expect(cache.size()).toBe(1); // Initial size check
        cache.remove("nonExistentKey"); // Attempt to remove a key that doesn't exist
        expect(cache.size()).toBe(1); // Size should remain unchanged
        expect(cache.get("key1")).toBe("value1"); // Other items should remain unaffected
    });

    test("should store and retrieve non-string values", () => {
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

    test("namespaces should not overwrite each other", () => {
        const cache1 = new LocalStorageLruCache<string>("namespace1", 2);
        const cache2 = new LocalStorageLruCache<string>("namespace2", 2);

        cache1.set("sharedKey", "valueFromCache1");
        cache2.set("sharedKey", "valueFromCache2");

        expect(cache1.get("sharedKey")).toBe("valueFromCache1");
        expect(cache2.get("sharedKey")).toBe("valueFromCache2");
    });

    test("namespaces should not affect each other's sizes", () => {
        const cache1 = new LocalStorageLruCache<string>("namespace1", 2);
        const cache2 = new LocalStorageLruCache<string>("namespace2", 3);

        cache1.set("key1", "value1");
        cache1.set("key2", "value2");

        cache2.set("keyA", "valueA");
        cache2.set("keyB", "valueB");
        cache2.set("keyC", "valueC");

        expect(cache1.size()).toBe(2);
        expect(cache2.size()).toBe(3);
    });

    test("evicting items in one namespace should not affect the other", () => {
        const cache1 = new LocalStorageLruCache<string>("namespace1", 2);
        const cache2 = new LocalStorageLruCache<string>("namespace2", 2);

        cache1.set("key1", "value1");
        cache1.set("key2", "value2");

        cache2.set("keyA", "valueA");
        cache2.set("keyB", "valueB");
        cache2.set("keyC", "valueC"); // This should evict keyA in namespace2

        expect(cache1.get("key1")).toBe("value1"); // Still present in namespace1
        expect(cache2.get("keyA")).toBeUndefined(); // Evicted from namespace2
        expect(cache2.get("keyB")).toBe("valueB");
        expect(cache2.get("keyC")).toBe("valueC");
    });

    test("should remove keys with specific value using predicate", () => {
        const cache = new LocalStorageLruCache<{ chatId: string }>("cache1", 5);

        cache.set("key1", { chatId: "chat1" });
        cache.set("key2", { chatId: "chat2" });
        cache.set("key3", { chatId: "chat1" });
        cache.set("key4", { chatId: "chat3" });

        expect(cache.size()).toBe(4);

        // Remove keys with chatId "chat1"
        cache.removeKeysWithValue((key, value) => value.chatId === "chat1");

        expect(cache.size()).toBe(2);
        expect(cache.get("key1")).toBeUndefined();
        expect(cache.get("key3")).toBeUndefined();
        expect(cache.get("key2")).toEqual({ chatId: "chat2" });
        expect(cache.get("key4")).toEqual({ chatId: "chat3" });
    });

    test("removing keys with non-matching predicate doesn't affect the cache", () => {
        const cache = new LocalStorageLruCache<{ chatId: string }>("cache1", 5);

        cache.set("key1", { chatId: "chat1" });
        cache.set("key2", { chatId: "chat2" });

        expect(cache.size()).toBe(2);

        // Attempt to remove keys with chatId "chat3" which doesn't exist
        cache.removeKeysWithValue((key, value) => value.chatId === "chat3");

        expect(cache.size()).toBe(2);
        expect(cache.get("key1")).toEqual({ chatId: "chat1" });
        expect(cache.get("key2")).toEqual({ chatId: "chat2" });
    });

});
