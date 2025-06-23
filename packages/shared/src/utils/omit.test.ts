// AI_CHECK: TEST_COVERAGE=7 | LAST: 2025-06-18
import { describe, it, expect } from "vitest";
import { omit } from "./omit.js";

describe("omit", () => {
    it("should omit top-level keys from an object", () => {
        const obj = { a: 1, b: 2, c: 3 };
        expect(omit(obj, ["a"])).toEqual({ b: 2, c: 3 });
        expect(omit(obj, ["a", "b"])).toEqual({ c: 3 });
    });

    it("should omit nested keys from an object using dot notation", () => {
        const obj = { a: { b: 2, c: 3 }, d: 4 };
        expect(omit(obj, ["a.b"])).toEqual({ a: { c: 3 }, d: 4 });
        expect(omit(obj, ["a.b", "a.c"])).toEqual({ d: 4 });
    });

    it("should remove parent objects if they are empty after omitting keys", () => {
        const obj = { a: { b: { c: 3, d: 4 } }, e: 5 };
        expect(omit(obj, ["a.b.c", "a.b.d"])).toEqual({ e: 5 });
    });

    it("should handle cases where keys are not present in the object", () => {
        const obj = { a: 1, b: 2, c: 3 };
        expect(omit(obj, ["d"])).toEqual({ a: 1, b: 2, c: 3 });
        expect(omit(obj, ["a.d"])).toEqual({ a: 1, b: 2, c: 3 });
    });

    it("should return a new object and not modify the original object", () => {
        const obj = { a: 1, b: 2, c: 3 };
        const result = omit(obj, ["a"]);
        expect(result).not.toBe(obj);
        expect(obj).toEqual({ a: 1, b: 2, c: 3 });
    });

    it("should handle empty arrays of keys to omit", () => {
        const obj = { a: 1, b: 2, c: 3 };
        expect(omit(obj, [])).toEqual(obj);
    });

    it("should handle deeply nested objects", () => {
        const obj = { a: { b: { c: { d: 4, e: 5 } } } };
        expect(omit(obj, ["a.b.c.d"])).toEqual({ a: { b: { c: { e: 5 } } } });
        expect(omit(obj, ["a.b.c.d", "a.b.c.e"])).toEqual({});
    });

    it("should handle security-sensitive keys safely", () => {
        // Test that dangerous prototype pollution keys are ignored
        const obj = { a: 1, b: 2, c: 3, regular: "value" };
        
        // These should not cause prototype pollution or errors
        expect(omit(obj, ["__proto__"])).toEqual({ a: 1, b: 2, c: 3, regular: "value" });
        expect(omit(obj, ["constructor"])).toEqual({ a: 1, b: 2, c: 3, regular: "value" });
        expect(omit(obj, ["prototype"])).toEqual({ a: 1, b: 2, c: 3, regular: "value" });
        
        // Should still omit regular keys
        expect(omit(obj, ["a", "__proto__", "b"])).toEqual({ c: 3, regular: "value" });
    });

    it("should handle unsafe keys in nested paths", () => {
        const obj = { 
            nested: { 
                __proto__: "dangerous", 
                constructor: "also dangerous",
                prototype: "still dangerous",
                safe: "this is ok", 
            },
            other: "value",
        };
        
        // Attempting to omit nested unsafe keys should be ignored safely
        expect(omit(obj, ["nested.__proto__"])).toEqual(obj);
        expect(omit(obj, ["nested.constructor"])).toEqual(obj);
        expect(omit(obj, ["nested.prototype"])).toEqual(obj);
        
        // Should still omit safe nested keys
        expect(omit(obj, ["nested.safe"])).toEqual({ 
            nested: { 
                __proto__: "dangerous", 
                constructor: "also dangerous",
                prototype: "still dangerous",
            },
            other: "value",
        });
    });

    it("should handle edge cases with null and undefined values", () => {
        const obj = { 
            nullValue: null, 
            undefinedValue: undefined, 
            nested: { 
                alsoNull: null, 
                normalValue: "test", 
            }, 
        };
        
        expect(omit(obj, ["nullValue"])).toEqual({ 
            undefinedValue: undefined, 
            nested: { alsoNull: null, normalValue: "test" }, 
        });
        
        expect(omit(obj, ["nested.alsoNull"])).toEqual({ 
            nullValue: null, 
            undefinedValue: undefined, 
            nested: { normalValue: "test" }, 
        });
    });

    it("should handle paths that point to non-object values", () => {
        const obj = { 
            str: "hello", 
            num: 42, 
            nested: { value: "test" }, 
        };
        
        // Trying to omit subkeys of primitive values should be safe
        expect(omit(obj, ["str.nonexistent"])).toEqual(obj);
        expect(omit(obj, ["num.also.nonexistent"])).toEqual(obj);
        
        // Should still work for valid paths
        expect(omit(obj, ["nested.value"])).toEqual({ str: "hello", num: 42 });
    });

    it("should handle arrays as values safely", () => {
        const obj = { 
            array: [1, 2, 3], 
            nested: { 
                arr: ["a", "b"], 
                other: "value", 
            }, 
        };
        
        // Should be able to omit array properties
        expect(omit(obj, ["array"])).toEqual({ 
            nested: { arr: ["a", "b"], other: "value" }, 
        });
        
        expect(omit(obj, ["nested.arr"])).toEqual({ 
            array: [1, 2, 3], 
            nested: { other: "value" }, 
        });
    });

    it("should handle complex parent object cleanup scenarios", () => {
        const obj = { 
            level1: { 
                level2a: { 
                    level3: "remove me", 
                },
                level2b: { 
                    keep: "this value", 
                },
            },
            otherTop: "keep this too",
        };
        
        // Removing the only key in level2a should remove level2a entirely
        expect(omit(obj, ["level1.level2a.level3"])).toEqual({ 
            level1: { 
                level2b: { keep: "this value" }, 
            },
            otherTop: "keep this too",
        });
        
        // But level1 should remain because level2b still has content
        expect(Object.keys(omit(obj, ["level1.level2a.level3"]).level1 || {})).toContain("level2b");
    });

    it("should handle undefined paths gracefully", () => {
        const obj = { a: { b: { c: "value" } } };
        
        // Paths that don't exist should not cause errors
        expect(omit(obj, ["x.y.z"])).toEqual(obj);
        expect(omit(obj, ["a.x.y"])).toEqual(obj);
        expect(omit(obj, ["a.b.x"])).toEqual(obj);
    });
});

