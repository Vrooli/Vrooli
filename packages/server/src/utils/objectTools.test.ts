import { describe, it, beforeEach, expect } from "vitest";
import { parseJsonOrDefault, sortObjectKeys, sortify } from "./objectTools.js";

describe("sortObjectKeys", () => {
    it("should return the same value for non-objects", () => {
        expect(sortObjectKeys(123)).toEqual(123);
        expect(sortObjectKeys("string")).toEqual("string");
        expect(sortObjectKeys(null)).toEqual(null);
        expect(sortObjectKeys(undefined)).toEqual(undefined);
    });

    it("should not modify arrays", () => {
        const array = [3, 1, 2];
        expect(sortObjectKeys(array)).toEqual([3, 1, 2]);
    });

    it("should handle date objects properly", () => {
        const date = new Date();
        expect(sortObjectKeys(date)).toEqual(date);
    });

    it("should return an empty object for an empty object", () => {
        expect(sortObjectKeys({})).toEqual({});
    });

    it("should sort the keys of a simple object alphabetically", () => {
        const obj = { b: 2, a: 1, c: 3 };
        const expected = { a: 1, b: 2, c: 3 };
        expect(sortObjectKeys(obj)).toEqual(expected);
    });

    it("should recursively sort the keys of nested objects", () => {
        const nestedObj = {
            charlie: { delta: 4, bravo: 2 },
            alpha: { echo: 5, foxtrot: 6 },
        };
        const expected = {
            alpha: { echo: 5, foxtrot: 6 },
            charlie: { bravo: 2, delta: 4 },
        };
        expect(sortObjectKeys(nestedObj)).toEqual(expected);
    });

    it("should not alter the internal data structure of objects", () => {
        const complexObj = {
            id: 1,
            details: {
                name: "Alice",
                age: 30,
                attributes: {
                    height: "5ft",
                    weight: "130lbs",
                },
            },
        };
        const expected = {
            details: {
                age: 30,
                attributes: {
                    height: "5ft",
                    weight: "130lbs",
                },
                name: "Alice",
            },
            id: 1,
        };
        expect(sortObjectKeys(complexObj)).toEqual(expected);
    });
});

describe("sortify", () => {
    it("should parse, sort keys alphabetically, and restringify JSON", () => {
        const json = JSON.stringify({ b: 2, a: { d: 4, c: 3 } });
        const expected = JSON.stringify({ a: { c: 3, d: 4 }, b: 2 });
        expect(sortify(json)).toEqual(expected);
    });

    it("should throw a CustomError with proper language and message on invalid JSON", () => {
        const invalidJson = "{\"a\":1,";
        expect(() => sortify(invalidJson)).toThrow();
    });

    it("should handle and sort nested objects correctly", () => {
        const nestedJson = JSON.stringify({ z: 1, y: { b: 2, a: 1 } });
        const expected = JSON.stringify({ y: { a: 1, b: 2 }, z: 1 });
        expect(sortify(nestedJson)).toEqual(expected);
    });

    it("should correctly handle arrays without altering their order", () => {
        const arrayJson = JSON.stringify({ fruits: ["apple", "orange"], veggies: ["carrot", "beet"] });
        const expected = JSON.stringify({ fruits: ["apple", "orange"], veggies: ["carrot", "beet"] });
        expect(sortify(arrayJson)).toEqual(expected);
    });

    it("should process empty objects and return an empty object string", () => {
        const emptyJson = JSON.stringify({});
        expect(sortify(emptyJson)).toEqual("{}");
    });
});

describe("parseJsonOrDefault", () => {

    it("should correctly parse valid JSON strings", () => {
        const json = "{\"name\":\"John\", \"age\":30}";
        expect(parseJsonOrDefault(json, {})).toEqual({ name: "John", age: 30 });
    });

    it("should return the default value if parsing fails", () => {
        const invalidJson = "{\"name\": \"John\"";
        const defaultValue = { name: "Default", age: 25 };
        expect(parseJsonOrDefault(invalidJson, defaultValue)).toEqual(defaultValue);
    });

    it("should return the default value if null is passed as JSON string", () => {
        const defaultValue = { name: "Default", age: 25 };
        expect(parseJsonOrDefault(null, defaultValue)).toEqual(defaultValue);
    });

    it("should return the default value if an empty string is passed", () => {
        const defaultValue = { name: "Default", age: 25 };
        expect(parseJsonOrDefault("", defaultValue)).toEqual(defaultValue);
    });

    it("should handle different types of default values", () => {
        expect(parseJsonOrDefault(null, "default string")).toEqual("default string");
        expect(parseJsonOrDefault(null, 0)).toEqual(0);
        expect(parseJsonOrDefault(null, false)).toEqual(false);
        expect(parseJsonOrDefault(null, [])).toEqual([]);
    });

    it("should handle complex and nested JSON objects", () => {
        const complexJson = "{\"data\": {\"list\": [1, 2, 3], \"valid\": true}}";
        const expected = { data: { list: [1, 2, 3], valid: true } };
        expect(parseJsonOrDefault(complexJson, {})).toEqual(expected);
    });
});

