/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, it, expect } from "vitest";
import { convertToDot, hasObjectChanged, valueFromDot } from "./objectTools.js";

describe("valueFromDot", () => {
    it("should handle empty path", () => {
        const obj = { test: "value" };
        expect(valueFromDot(obj, "")).toBeNull();
    });

    it("should handle single level path", () => {
        const obj = { test: "value" };
        expect(valueFromDot(obj, "test")).toBe("value");
    });

    it("should handle multi level path", () => {
        const obj = { level1: { level2: { level3: "value" } } };
        expect(valueFromDot(obj, "level1.level2.level3")).toBe("value");
    });

    it("should handle undefined values", () => {
        const obj = { test: undefined };
        expect(valueFromDot(obj, "test")).toBeNull();
    });

    it("should handle non-existent paths", () => {
        const obj = { test: "value" };
        expect(valueFromDot(obj, "nonexistent")).toBeNull();
    });
});

describe("convertToDot", () => {
    it("should handle empty object", () => {
        expect(convertToDot({})).toEqual({});
    });

    it("should handle single level object", () => {
        const obj = { test: "value" };
        expect(convertToDot(obj)).toEqual({ "test": "value" });
    });

    it("should handle multi level object", () => {
        const obj = { level1: { level2: { level3: "value" } } };
        expect(convertToDot(obj)).toEqual({ "level1.level2.level3": "value" });
    });

    it("should handle multiple paths", () => {
        const obj = { path1: "value1", path2: { nested: "value2" } };
        expect(convertToDot(obj)).toEqual({
            "path1": "value1",
            "path2.nested": "value2",
        });
    });
});

describe("hasObjectChanged", () => {
    it("should detect changes in simple values", () => {
        const original = { a: 1, b: 2 };
        const updated = { a: 1, b: 3 };
        expect(hasObjectChanged(original, updated)).toBe(true);
    });

    it("should detect changes in nested objects", () => {
        const original = { a: { x: 1 }, b: 2 };
        const updated = { a: { x: 2 }, b: 2 };
        expect(hasObjectChanged(original, updated)).toBe(true);
    });

    it("should detect changes in arrays", () => {
        const original = { arr: [1, 2, 3] };
        const updated = { arr: [1, 2, 4] };
        expect(hasObjectChanged(original, updated)).toBe(true);
    });

    it("should detect changes in specific fields", () => {
        const original = { a: 1, b: 2, c: 3 };
        const updated = { a: 1, b: 3, c: 3 };
        expect(hasObjectChanged(original, updated, ["b"])).toBe(true);
        expect(hasObjectChanged(original, updated, ["a"])).toBe(false);
        expect(hasObjectChanged(original, updated, ["a", "c"])).toBe(false);
        expect(hasObjectChanged(original, updated, ["a", "b", "c"])).toBe(true);
    });

    it("should handle missing fields in updated object", () => {
        const original = { a: 1, b: 2, c: 3 };
        const updated = { a: 1, b: 2 };
        // When checking all fields, missing field should be detected as change
        expect(hasObjectChanged(original, updated)).toBe(true);
        // When checking specific fields that exist in both, no change
        expect(hasObjectChanged(original, updated, ["a", "b"])).toBe(false);
    });

    it("should handle new fields in updated object", () => {
        const original = { a: 1, b: 2 };
        const updated = { a: 1, b: 2, c: 3 };
        // New field should be detected as change
        expect(hasObjectChanged(original, updated)).toBe(true);
        // When checking only existing fields, no change
        expect(hasObjectChanged(original, updated, ["a", "b"])).toBe(false);
    });

    it("should handle null and undefined correctly", () => {
        // When original is null/undefined but updated has value, it's a change
        expect(hasObjectChanged(null, { a: 1 })).toBe(true);
        expect(hasObjectChanged(undefined, { a: 1 })).toBe(true);
        
        // When updated is null/undefined but original has value, it's a change
        expect(hasObjectChanged({ a: 1 }, null)).toBe(true);
        expect(hasObjectChanged({ a: 1 }, undefined)).toBe(true);
        
        // Both null/undefined should be no change
        expect(hasObjectChanged(null, null)).toBe(false);
        expect(hasObjectChanged(undefined, undefined)).toBe(false);
        expect(hasObjectChanged(null, undefined)).toBe(false); // Both represent "no value"
        expect(hasObjectChanged(undefined, null)).toBe(false); // Both represent "no value"
    });

    it("should handle non-object values", () => {
        // Primitive comparisons
        expect(hasObjectChanged(5, 5)).toBe(false);
        expect(hasObjectChanged(5, 6)).toBe(true);
        expect(hasObjectChanged("hello", "hello")).toBe(false);
        expect(hasObjectChanged("hello", "world")).toBe(true);
        expect(hasObjectChanged(true, true)).toBe(false);
        expect(hasObjectChanged(true, false)).toBe(true);
    });

    it("should handle dot notation in field names", () => {
        const original = { 
            user: { 
                profile: { 
                    name: "John",
                    age: 25
                },
                settings: {
                    theme: "dark"
                }
            } 
        };
        const updated = { 
            user: { 
                profile: { 
                    name: "Jane",
                    age: 25
                },
                settings: {
                    theme: "dark"
                }
            } 
        };
        
        // Check specific nested field
        expect(hasObjectChanged(original, updated, ["user.profile.name"])).toBe(true);
        expect(hasObjectChanged(original, updated, ["user.profile.age"])).toBe(false);
        expect(hasObjectChanged(original, updated, ["user.settings.theme"])).toBe(false);
        
        // Check multiple fields with dot notation
        expect(hasObjectChanged(original, updated, ["user.profile.age", "user.settings.theme"])).toBe(false);
        expect(hasObjectChanged(original, updated, ["user.profile.name", "user.settings.theme"])).toBe(true);
    });

    it("should handle arrays with different lengths", () => {
        const original = { arr: [1, 2, 3] };
        const updated1 = { arr: [1, 2] };
        const updated2 = { arr: [1, 2, 3, 4] };
        
        expect(hasObjectChanged(original, updated1)).toBe(true);
        expect(hasObjectChanged(original, updated2)).toBe(true);
    });

    it("should handle arrays of objects", () => {
        const original = { 
            items: [
                { id: 1, name: "Item 1" },
                { id: 2, name: "Item 2" }
            ] 
        };
        const updated = { 
            items: [
                { id: 1, name: "Item 1" },
                { id: 2, name: "Item 2 Updated" }
            ] 
        };
        
        expect(hasObjectChanged(original, updated)).toBe(true);
        expect(hasObjectChanged(original, updated, ["items"])).toBe(true);
    });

    it("should handle deeply nested changes", () => {
        const original = {
            level1: {
                level2: {
                    level3: {
                        level4: {
                            value: "original"
                        }
                    }
                }
            }
        };
        const updated = {
            level1: {
                level2: {
                    level3: {
                        level4: {
                            value: "updated"
                        }
                    }
                }
            }
        };
        
        expect(hasObjectChanged(original, updated)).toBe(true);
        expect(hasObjectChanged(original, updated, ["level1.level2.level3.level4.value"])).toBe(true);
    });

    it("should handle mixed types in comparisons", () => {
        const original = { value: "5" };
        const updated = { value: 5 };
        
        // String vs number should be detected as change
        expect(hasObjectChanged(original, updated)).toBe(true);
    });

    it("should handle empty arrays and objects", () => {
        expect(hasObjectChanged({ arr: [] }, { arr: [] })).toBe(false);
        expect(hasObjectChanged({ obj: {} }, { obj: {} })).toBe(false);
        expect(hasObjectChanged({ arr: [] }, { arr: [1] })).toBe(true);
        expect(hasObjectChanged({ obj: {} }, { obj: { a: 1 } })).toBe(true);
    });

    it("should handle fields that don't exist in either object", () => {
        const original = { a: 1 };
        const updated = { a: 1 };
        
        // Check for non-existent field
        expect(hasObjectChanged(original, updated, ["nonExistent"])).toBe(false);
    });

    it("should handle checking nested fields when parent is not an object", () => {
        const original = { user: "string value" };
        const updated = { user: "string value" };
        
        // Trying to access nested field on non-object should return false
        expect(hasObjectChanged(original, updated, ["user.profile.name"])).toBe(false);
    });

    it("should handle array comparisons with specific fields", () => {
        const original = { data: [1, 2, 3], other: "value" };
        const updated = { data: [1, 2, 3], other: "changed" };
        
        expect(hasObjectChanged(original, updated, ["data"])).toBe(false);
        expect(hasObjectChanged(original, updated, ["other"])).toBe(true);
    });
});

describe("valueFromDot edge cases", () => {
    it("should handle null object", () => {
        expect(valueFromDot(null as any, "test")).toBeNull();
    });

    it("should handle nested null values", () => {
        const obj = { parent: null };
        expect(valueFromDot(obj, "parent.child")).toBeNull();
    });

    it("should handle arrays in path", () => {
        const obj = { items: [{ name: "first" }, { name: "second" }] };
        expect(valueFromDot(obj, "items.0.name")).toBe("first");
        expect(valueFromDot(obj, "items.1.name")).toBe("second");
    });

    it("should handle false values correctly", () => {
        const obj = { flag: false, zero: 0 };
        expect(valueFromDot(obj, "flag")).toBe(false); // exists() returns true for false
        expect(valueFromDot(obj, "zero")).toBe(0); // exists() returns true for 0
    });

    it("should handle empty string values", () => {
        const obj = { empty: "" };
        expect(valueFromDot(obj, "empty")).toBe(""); // exists() returns true for empty string
    });

    it("should handle complex nested structures", () => {
        const obj = {
            users: {
                admin: {
                    permissions: {
                        read: true,
                        write: true
                    }
                }
            }
        };
        expect(valueFromDot(obj, "users.admin.permissions.read")).toBe(true);
        expect(valueFromDot(obj, "users.admin.permissions.delete")).toBeNull();
    });
});

describe("convertToDot edge cases", () => {
    it("should handle null values", () => {
        const obj = { a: null, b: { c: null } };
        expect(convertToDot(obj)).toEqual({
            "a": null,
            "b.c": null
        });
    });

    it("should handle arrays", () => {
        const obj = { arr: [1, 2, 3] };
        // Arrays are converted to dot notation with numeric indices
        // This is the intended behavior for deep object comparison
        expect(convertToDot(obj)).toEqual({
            "arr.0": 1,
            "arr.1": 2,
            "arr.2": 3
        });
    });

    it("should handle deeply nested structures", () => {
        const obj = {
            level1: {
                level2: {
                    level3: {
                        level4: {
                            value: "deep"
                        }
                    }
                }
            }
        };
        expect(convertToDot(obj)).toEqual({
            "level1.level2.level3.level4.value": "deep"
        });
    });

    it("should handle mixed value types", () => {
        const obj = {
            string: "text",
            number: 42,
            boolean: true,
            null: null,
            undefined: undefined,
            nested: {
                array: [1, 2, 3],
                object: { key: "value" }
            }
        };
        expect(convertToDot(obj)).toEqual({
            "string": "text",
            "number": 42,
            "boolean": true,
            "null": null,
            "undefined": undefined,
            "nested.array.0": 1,
            "nested.array.1": 2,
            "nested.array.2": 3,
            "nested.object.key": "value"
        });
    });

    it("should handle objects with numeric keys", () => {
        const obj = { "0": "first", "1": "second", "2": { "3": "nested" } };
        expect(convertToDot(obj)).toEqual({
            "0": "first",
            "1": "second",
            "2.3": "nested"
        });
    });
});
