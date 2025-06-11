import { describe, it, expect } from "vitest";
import { isEqual } from "./isEqual.js";

describe("isEqual", () => {
    it("should return true for two identical primitive values", () => {
        expect(isEqual(5, 5)).toBe(true);
        expect(isEqual("test", "test")).toBe(true);
        expect(isEqual(true, true)).toBe(true);
        expect(isEqual(undefined, undefined)).toBe(true);
    });

    it("should return false for two different primitive values", () => {
        expect(isEqual(5, 6)).toBe(false);
        expect(isEqual("test", "test1")).toBe(false);
        expect(isEqual(true, false)).toBe(false);
    });

    it("should return true for two identical arrays", () => {
        expect(isEqual([1, 2, 3], [1, 2, 3])).toBe(true);
        expect(isEqual(["a", "b", "c"], ["a", "b", "c"])).toBe(true);
        expect(isEqual([true, false], [true, false])).toBe(true);
    });

    it("should return false for two different arrays", () => {
        expect(isEqual([1, 2, 3], [1, 2])).toBe(false);
        expect(isEqual(["a", "b"], ["a", "b", "c"])).toBe(false);
        expect(isEqual([true, false], [false, true])).toBe(false);
    });

    it("should return true for two identical objects", () => {
        expect(isEqual({ a: 1, b: 2 }, { a: 1, b: 2 })).toBe(true);
        expect(isEqual({ name: "John", age: 25 }, { name: "John", age: 25 })).toBe(true);
    });

    it("should return false for two different objects", () => {
        expect(isEqual({ a: 1, b: 2 }, { a: 1 })).toBe(false);
        expect(isEqual({ name: "John", age: 25 }, { name: "Doe", age: 25 })).toBe(false);
    });

    it("should return true for nested structures that are the same", () => {
        const obj1 = { a: [1, 2, { x: 10, y: 20 }], b: "test" };
        const obj2 = { a: [1, 2, { x: 10, y: 20 }], b: "test" };
        expect(isEqual(obj1, obj2)).toBe(true);
    });

    it("should return false for nested structures that are different", () => {
        const obj1 = { a: [1, 2, { x: 10, y: 20 }], b: "test" };
        const obj2 = { a: [1, 2, { x: 10, y: 30 }], b: "test" };
        expect(isEqual(obj1, obj2)).toBe(false);
    });

    it("should handle null and undefined values correctly", () => {
        expect(isEqual(null, null)).toBe(true);
        expect(isEqual(undefined, undefined)).toBe(true);
        expect(isEqual(null, undefined)).toBe(false);
        expect(isEqual(undefined, null)).toBe(false);
        expect(isEqual(null, 0)).toBe(false);
        expect(isEqual(0, null)).toBe(false);
        expect(isEqual(undefined, "")).toBe(false);
        expect(isEqual("", undefined)).toBe(false);
    });

    it("should handle different types correctly", () => {
        expect(isEqual(5, "5")).toBe(false);
        expect(isEqual(true, 1)).toBe(false);
        expect(isEqual(false, 0)).toBe(false);
        expect(isEqual([], {})).toBe(false);
        expect(isEqual({}, [])).toBe(false);
        expect(isEqual(new Date("2023-01-01"), "2023-01-01")).toBe(false);
    });

    it("should handle empty objects and arrays", () => {
        expect(isEqual({}, {})).toBe(true);
        expect(isEqual([], [])).toBe(true);
        expect(isEqual({}, [])).toBe(false);
        expect(isEqual([], {})).toBe(false);
    });

    it("should handle objects with undefined values", () => {
        expect(isEqual({ a: undefined }, { a: undefined })).toBe(true);
        expect(isEqual({ a: undefined }, {})).toBe(false);
        expect(isEqual({}, { a: undefined })).toBe(false);
        expect(isEqual({ a: 1, b: undefined }, { a: 1, b: undefined })).toBe(true);
        expect(isEqual({ a: 1, b: undefined }, { a: 1 })).toBe(false);
    });

    it("should handle self-references correctly", () => {
        const obj1: any = { a: 1 };
        obj1.self = obj1;
        
        // Same object reference should be equal to itself
        expect(isEqual(obj1, obj1)).toBe(true);
        
        // Different objects with same structure but no circular reference
        const obj2 = { a: 1, self: null };
        expect(isEqual(obj1, obj2)).toBe(false);
    });


    it("should handle Date objects correctly", () => {
        const date1 = new Date("2023-01-01");
        const date2 = new Date("2023-01-01");
        const date3 = new Date("2023-01-02");
        
        // Same Date instance should be equal
        expect(isEqual(date1, date1)).toBe(true);
        
        // Different Date instances with same time should be equal
        expect(isEqual(date1, date2)).toBe(true);
        // Different Date instances with different times should not be equal
        expect(isEqual(date1, date3)).toBe(false);
    });


    it("should handle arrays with different order", () => {
        expect(isEqual([1, 2, 3], [3, 2, 1])).toBe(false);
        expect(isEqual(["a", "b"], ["b", "a"])).toBe(false);
    });

    it("should handle NaN values", () => {
        // NaN should equal NaN in deep equality
        expect(isEqual(NaN, NaN)).toBe(true);
        expect(isEqual({ a: NaN }, { a: NaN })).toBe(true);
        expect(isEqual([NaN], [NaN])).toBe(true);
        
        // NaN should not equal other values
        expect(isEqual(NaN, undefined)).toBe(false);
        expect(isEqual(NaN, null)).toBe(false);
        expect(isEqual(NaN, 0)).toBe(false);
    });


    it("should handle Symbol values correctly", () => {
        const sym1 = Symbol("test");
        const sym2 = Symbol("test");
        const sym3 = Symbol.for("global");
        const sym4 = Symbol.for("global");
        
        // Same symbol reference should be equal
        expect(isEqual(sym1, sym1)).toBe(true);
        // Different symbols with same description should not be equal
        expect(isEqual(sym1, sym2)).toBe(false);
        // Global symbols with same key should be equal
        expect(isEqual(sym3, sym4)).toBe(true);
        // Symbol vs other types should not be equal
        expect(isEqual(sym1, "test")).toBe(false);
    });

    it("should handle BigInt values", () => {
        expect(isEqual(BigInt(123), BigInt(123))).toBe(true);
        expect(isEqual(BigInt(123), BigInt(456))).toBe(false);
        // BigInt vs Number
        expect(isEqual(BigInt(123), 123)).toBe(false);
        expect(isEqual(123, BigInt(123))).toBe(false);
    });

    it("should handle functions", () => {
        const func1 = () => "test";
        const func2 = () => "test";
        
        // Same function reference should be equal
        expect(isEqual(func1, func1)).toBe(true);
        // Different function instances should not be equal
        expect(isEqual(func1, func2)).toBe(false);
    });


    it("should handle RegExp objects", () => {
        const regex1 = /test/g;
        const regex2 = /test/g;
        const regex3 = /different/g;
        const regex4 = /test/i; // Different flags
        
        // Same RegExp instance
        expect(isEqual(regex1, regex1)).toBe(true);
        
        // Different RegExp instances with same pattern and flags should be equal
        expect(isEqual(regex1, regex2)).toBe(true);
        // Different patterns should not be equal
        expect(isEqual(regex1, regex3)).toBe(false);
        // Same pattern but different flags should not be equal
        expect(isEqual(regex1, regex4)).toBe(false);
    });


    it("should handle Set objects", () => {
        const set1 = new Set([1, 2, 3]);
        const set2 = new Set([1, 2, 3]);
        const set3 = new Set([1, 2]);
        const set4 = new Set([3, 2, 1]); // Different order
        
        // Same Set instance
        expect(isEqual(set1, set1)).toBe(true);
        
        // Different Set instances with same values should be equal
        expect(isEqual(set1, set2)).toBe(true);
        // Different values should not be equal
        expect(isEqual(set1, set3)).toBe(false);
        // Order doesn't matter in Sets
        expect(isEqual(set1, set4)).toBe(true);
    });


    it("should handle Map objects with proper order semantics", () => {
        const map1 = new Map([["a", 1], ["b", 2]]);
        const map2 = new Map([["a", 1], ["b", 2]]);
        const map3 = new Map([["a", 1]]);
        const map4 = new Map([["b", 2], ["a", 1]]); // Different insertion order
        const map5 = new Map([["a", 1], ["b", 3]]); // Different value
        
        expect(isEqual(map1, map1)).toBe(true);
        expect(isEqual(map1, map2)).toBe(true);
        expect(isEqual(map1, map3)).toBe(false);
        // Maps with same entries but different insertion order should be different
        expect(isEqual(map1, map4)).toBe(false);
        expect(isEqual(map1, map5)).toBe(false);
    });


    it("should compare objects by constructor and content", () => {
        class Person {
            constructor(public name: string, public age: number) {}
        }
        
        class Employee {
            constructor(public name: string, public age: number) {}
        }
        
        const person1 = new Person("John", 30);
        const person2 = new Person("John", 30);
        const employee = new Employee("John", 30);
        
        // Same constructor and content should be equal
        expect(isEqual(person1, person2)).toBe(true);
        // Different constructors should not be equal even with same content
        expect(isEqual(person1, employee)).toBe(false);
    });


    it("should only compare enumerable properties", () => {
        const obj1 = { visible: "same" };
        const obj2 = { visible: "same" };
        
        Object.defineProperty(obj1, "hidden", {
            value: "secret",
            enumerable: false
        });
        
        Object.defineProperty(obj2, "hidden", {
            value: "different",
            enumerable: false
        });
        
        // Only enumerable properties should affect equality
        expect(isEqual(obj1, obj2)).toBe(true);
    });


    it("should handle Boolean objects", () => {
        expect(isEqual(new Boolean(true), new Boolean(true))).toBe(true);
        expect(isEqual(new Boolean(false), new Boolean(false))).toBe(true);
        expect(isEqual(new Boolean(true), new Boolean(false))).toBe(false);
        
        // Boolean object vs primitive
        expect(isEqual(new Boolean(true), true)).toBe(false);
        expect(isEqual(true, new Boolean(true))).toBe(false);
    });

    it("should handle Number objects", () => {
        expect(isEqual(new Number(123), new Number(123))).toBe(true);
        expect(isEqual(new Number(123), new Number(456))).toBe(false);
        expect(isEqual(new Number(0), new Number(0))).toBe(true);
        expect(isEqual(new Number(-0), new Number(0))).toBe(true); // -0 === 0
        
        // Number object vs primitive
        expect(isEqual(new Number(123), 123)).toBe(false);
        expect(isEqual(123, new Number(123))).toBe(false);
    });

    it("should handle String objects", () => {
        expect(isEqual(new String("hello"), new String("hello"))).toBe(true);
        expect(isEqual(new String("hello"), new String("world"))).toBe(false);
        expect(isEqual(new String(""), new String(""))).toBe(true);
        
        // String object vs primitive
        expect(isEqual(new String("hello"), "hello")).toBe(false);
        expect(isEqual("hello", new String("hello"))).toBe(false);
    });

    it("should handle arrays with undefined elements correctly", () => {
        expect(isEqual([1, undefined, 3], [1, undefined, 3])).toBe(true);
        // Sparse arrays should be treated consistently with explicit undefined
        expect(isEqual([1, , 3], [1, undefined, 3])).toBe(true);
        expect(isEqual([undefined], [])).toBe(false);
        expect(isEqual([1, undefined], [1])).toBe(false);
    });

    it("should handle objects with numeric string keys", () => {
        expect(isEqual({ "0": "a", "1": "b" }, { "0": "a", "1": "b" })).toBe(true);
        expect(isEqual({ "0": "a", "1": "b" }, { "1": "b", "0": "a" })).toBe(true);
    });
});
