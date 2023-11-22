import { parseSearchParams, stringifySearchParams } from "./searchParams";

describe("parseSearchParams", () => {
    // Save the original window.location
    const originalLocation = window.location;

    // Helper function to set window.location.search
    const setWindowSearch = (search) => {
        Object.defineProperty(window, "location", {
            value: {
                search,
            },
            writable: true,
        });
    };

    afterEach(() => {
        // Restore the original window.location after each test
        Object.defineProperty(window, "location", {
            value: originalLocation,
        });
    });

    it("returns an empty object for empty search params", () => {
        setWindowSearch("");
        expect(parseSearchParams()).toEqual({});
    });

    it("parses a single key-value pair correctly", () => {
        setWindowSearch("?key=%22value%22");
        expect(parseSearchParams()).toEqual({ key: "value" });
    });

    it("parses multiple key-value pairs correctly", () => {
        setWindowSearch("?key1=%22value1%22&key2=%22value2%22");
        expect(parseSearchParams()).toEqual({ key1: "value1", key2: "value2" });
    });

    it("decodes special characters correctly", () => {
        setWindowSearch("?key%20one=%22value%2Fone%22");
        expect(parseSearchParams()).toEqual({ "key one": "value/one" });
    });

    it("parses nested objects correctly, including nested number, boolean, and null", () => {
        setWindowSearch("?key=%7B%22nestedKey%22%3A%22nestedValue%22%2C%22nestedNumber%22%3A-123%2C%22nestedBoolean%22%3Atrue%2C%22nestedNull%22%3Anull%7D");
        expect(parseSearchParams()).toEqual({
            key: {
                nestedKey: "nestedValue",
                nestedNumber: -123,
                nestedBoolean: true,
                nestedNull: null,
            },
        });
    });

    it("parses mixed types correctly", () => {
        setWindowSearch("?string=%22value%22&number=123&boolean=true&null=null");
        expect(parseSearchParams()).toEqual({ string: "value", number: 123, boolean: true, null: null });
    });

    it("returns an empty object for invalid format", () => {
        // Mock console.error
        const originalConsoleError = console.error;
        console.error = jest.fn();

        setWindowSearch("?invalid");
        expect(parseSearchParams()).toEqual({});

        // Restore console.error
        console.error = originalConsoleError;
    });

    it("parses a top-level array of numbers correctly", () => {
        setWindowSearch("?numbers=%5B123%2C456%2C789%5D");
        expect(parseSearchParams()).toEqual({
            numbers: [123, 456, 789],
        });
    });

    it("parses an array of objects nested within an object correctly", () => {
        setWindowSearch("?data=%7B%22items%22%3A%5B%7B%22id%22%3A1%2C%22name%22%3A%22Item1%22%7D%2C%7B%22id%22%3A2%2C%22name%22%3A%22Item2%22%7D%5D%7D"); // URL-encoded for {"items":[{"id":1,"name":"Item1"},{"id":2,"name":"Item2"}]}
        expect(parseSearchParams()).toEqual({
            data: {
                items: [
                    { id: 1, name: "Item1" },
                    { id: 2, name: "Item2" },
                ],
            },
        });
    });
});

describe("stringifySearchParams", () => {
    it("returns an empty string for an empty object", () => {
        expect(stringifySearchParams({})).toBe("");
    });

    it("returns an empty string for an object with only null and undefined values", () => {
        expect(stringifySearchParams({ key1: null, key2: undefined })).toBe("");
    });

    it("formats a single key-value pair correctly", () => {
        expect(stringifySearchParams({ key: "value" })).toBe("?key=%22value%22");
    });

    it("formats multiple key-value pairs correctly", () => {
        expect(stringifySearchParams({ key1: "value1", key2: "value2" })).toBe("?key1=%22value1%22&key2=%22value2%22");
    });

    it("ignores keys with null or undefined values", () => {
        expect(stringifySearchParams({ key1: null, key2: undefined, key3: "value" })).toBe("?key3=%22value%22");
    });

    it("encodes special characters correctly", () => {
        expect(stringifySearchParams({ "key one": "value/one" })).toBe("?key%20one=%22value%2Fone%22");
    });

    it("stringifies and encodes nested objects correctly", () => {
        expect(stringifySearchParams({ key: { nestedKey: "nestedValue" } })).toBe("?key=%7B%22nestedKey%22%3A%22nestedValue%22%7D");
    });
});
