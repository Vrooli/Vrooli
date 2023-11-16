import { getLocalStorageKeys } from "./localStorage";

let mockLocalStorage = {};

export const setupLocalStorageMock = () => {
    mockLocalStorage = {};
    global.localStorage = {
        getItem(key) {
            return key in mockLocalStorage ? mockLocalStorage[key] : null;
        },
        setItem(key, value) {
            mockLocalStorage[key] = String(value);
        },
        removeItem(key) {
            delete mockLocalStorage[key];
        },
        clear() {
            mockLocalStorage = {};
        },
        key(i) {
            const keys = Object.keys(mockLocalStorage);
            return keys[i] || null;
        },
        get length() {
            return Object.keys(mockLocalStorage).length;
        },
    };
};

export const teardownLocalStorageMock = () => {
    mockLocalStorage = {};
    global.localStorage.clear();
};

describe("getLocalStorageKeys", () => {
    beforeEach(() => {
        setupLocalStorageMock();
    });

    afterEach(() => {
        teardownLocalStorageMock();
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
