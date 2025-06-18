/* eslint-disable @typescript-eslint/ban-ts-comment */
// AI_CHECK: TEST_COVERAGE=7 | LAST: 2025-06-18
import { describe, it, expect } from "vitest";
import { deepClone, getDotNotationValue, isObject, isOfType, mergeDeep, setDotNotationValue, splitDotNotation } from "./objects.js";

describe("getDotNotationValue", () => {
    it("should retrieve values using dot notation", () => {
        const obj = {
            foo: {
                bar: {
                    baz: 42,
                },
                arr: [1, 2, "3"],
                nullValue: null,
                undefinedValue: undefined,
            },
        };
        expect(getDotNotationValue(obj, "foo.bar.baz")).toBe(42);
        expect(getDotNotationValue(obj, "foo.arr[1]")).toBe(2);
        expect(getDotNotationValue(obj, "foo.arr[2]")).toBe("3");
        expect(getDotNotationValue(obj, "foo.arr[3]")).toBeUndefined();
        expect(getDotNotationValue(obj, "foo.nonExistent")).toBeUndefined();
        expect(getDotNotationValue(obj, "foo.nullValue")).toBeNull();
        expect(getDotNotationValue(obj, "foo.undefinedValue")).toBeUndefined();
    });

    it("should handle trailing dots", () => {
        const obj = {
            foo: {
                bar: {
                    baz: 42,
                },
                arr: [1, 2, "3"],
                nullValue: null,
                undefinedValue: undefined,
            },
        };
        expect(getDotNotationValue(obj, "foo.bar.baz.")).toBe(42);
        expect(getDotNotationValue(obj, "foo.arr[0].")).toBe(1);
        expect(getDotNotationValue(obj, "foo.arr[1].")).toBe(2);
        expect(getDotNotationValue(obj, "foo.arr[2].")).toBe("3");
        expect(getDotNotationValue(obj, "foo.arr[3].")).toBeUndefined();
        expect(getDotNotationValue(obj, "foo.nullValue.")).toBeNull();
        expect(getDotNotationValue(obj, "foo.undefinedValue.")).toBeUndefined();
    });

    it("should handle edge cases", () => {
        expect(getDotNotationValue(undefined, "foo")).toBeUndefined();
        expect(getDotNotationValue({}, "foo")).toBeUndefined();
    });

    it("should return the whole object when path has no valid keys", () => {
        const obj = { foo: "bar", baz: 42 };
        // When regex doesn't match any valid keys, it returns the original object
        expect(getDotNotationValue(obj, ".")).toEqual(obj);
        // Empty path returns the object
        expect(getDotNotationValue(obj, "")).toEqual(obj);
        // Path with only dots returns the object
        expect(getDotNotationValue(obj, "..")).toEqual(obj);
        expect(getDotNotationValue(obj, "...")).toEqual(obj);
    });
});

describe("setDotNotationValue", () => {
    it("should set values using dot notation", () => {
        const obj: Record<string, any> = {
            foo: {
                bar: {
                    baz: 42,
                },
                arr: [1, 2, 3],
            },
        };
        setDotNotationValue(obj, "foo.bar.baz", 100);
        expect(obj.foo.bar.baz).toBe(100);

        setDotNotationValue(obj, "foo.arr[1]", 10);
        expect(obj.foo.arr[1]).toBe(10);

        setDotNotationValue(obj, "foo.arr[3]", 4);
        expect(obj.foo.arr[3]).toBe(4);

        setDotNotationValue(obj, "foo.newProp", "newValue");
        expect(obj.foo.newProp).toBe("newValue");
    });

    it("should handle edge cases", () => {
        const obj1: any = {};
        setDotNotationValue(obj1, "", "value");
        expect(obj1).toEqual({});

        const obj2 = { foo: 42 };
        setDotNotationValue(obj2, "foo", "updatedValue");
        expect(obj2.foo).toBe("updatedValue");
    });

    it("should handle numeric keys on non-array objects", () => {
        const obj: Record<string, any> = {
            data: { someField: "value" }
        };
        // Set a numeric key on a non-array object
        setDotNotationValue(obj, "data[0]", "newValue");
        expect(obj.data[0]).toBe("newValue");
    });

    it("should create nested structures when setting numeric keys", () => {
        const obj: Record<string, any> = {};
        // This will create nested objects when using numeric indices
        setDotNotationValue(obj, "foo[0][1]", "value");
        expect(obj.foo[0][1]).toBe("value");
    });

    it("should throw error when trying to access properties on non-objects", () => {
        const obj: Record<string, any> = {
            foo: "stringValue"
        };
        // Try to set a property on a string value (should throw)
        expect(() => setDotNotationValue(obj, "foo.bar", "value")).toThrow("Expected object for property access");
    });

    it("should throw error when using numeric key on non-object", () => {
        const obj: Record<string, any> = {
            foo: 42
        };
        // Try to use numeric key on a number (should throw)
        expect(() => setDotNotationValue(obj, "foo[0]", "value")).toThrow("Expected object for property access");
    });
});

describe("splitDotNotation", () => {
    it("splits standard dot notation strings correctly", () => {
        const input = ["first.second.third", "another.example.string"];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).toEqual(["first", "another"]);
        expect(remainders).toEqual(["second.third", "example.string"]);
    });

    it("handles empty strings", () => {
        const input = [""];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).toEqual([""]);
        expect(remainders).toEqual([""]);
    });

    it("handles strings without dots", () => {
        const input = ["first", "second"];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).toEqual(["first", "second"]);
        expect(remainders).toEqual(["", ""]);
    });

    it("handles strings with multiple dots", () => {
        const input = ["first.second.third", "another.one.two.three"];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).toEqual(["first", "another"]);
        expect(remainders).toEqual(["second.third", "one.two.three"]);
    });

    it("handles an empty array", () => {
        const input: string[] = [];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).toHaveLength(0);
        expect(remainders).toHaveLength(0);
    });

    it("handles an array with only empty strings", () => {
        const input = ["", ""];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).toEqual(["", ""]);
        expect(remainders).toEqual(["", ""]);
    });

    it("should treat null as empty string for safety", () => {
        const input = [null, "valid.string", null];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input);
        // null values should be treated as empty strings to avoid errors
        expect(fields).toEqual(["", "valid", ""]);
        expect(remainders).toEqual(["", "string", ""]);
    });

    it("should treat undefined as empty string for safety", () => {
        const input = [undefined, "test.value", undefined];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input);
        // undefined values should be treated as empty strings to avoid errors
        expect(fields).toEqual(["", "test", ""]);
        expect(remainders).toEqual(["", "value", ""]);
    });

    it("handles mixed valid and invalid inputs", () => {
        const input = [null, "first.second", "", undefined, "another.one"];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).toEqual(["", "first", "", "", "another"]);
        expect(remainders).toEqual(["", "second", "", "", "one"]);
    });

    it("removes empty strings when removeEmpty is true", () => {
        const input = [null, "first.second", "", undefined, "another.one"];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input, true);
        expect(fields).toEqual(["first", "another"]);
        expect(remainders).toEqual(["second", "one"]);
    });

    it("includes empty strings when removeEmpty is false", () => {
        const input = [null, "first.second", "", undefined, "another.one"];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input, false);
        expect(fields).toEqual(["", "first", "", "", "another"]);
        expect(remainders).toEqual(["", "second", "", "", "one"]);
    });

    it("defaults to including empty strings when removeEmpty is not provided", () => {
        const input = ["", "first.second", null, "another.one", undefined];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).toEqual(["", "first", "", "another", ""]);
        expect(remainders).toEqual(["", "second", "", "one", ""]);
    });
});


describe("isObject", () => {
    it("should identify objects correctly", () => {
        expect(isObject({})).toBe(true);
        expect(isObject([])).toBe(true);
        expect(isObject(new Date())).toBe(true);
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        expect(isObject(() => { })).toBe(true);
        expect(isObject(null)).toBe(false);
        expect(isObject(undefined)).toBe(false);
        expect(isObject(42)).toBe(false);
        expect(isObject("string")).toBe(false);
        expect(isObject(true)).toBe(false);
    });
});

describe("isOfType", () => {
    it("should identify types correctly", () => {
        const obj1 = { __typename: "TypeA", property: "value" };
        const obj2 = { __typename: "TypeB", property: "value" };
        const obj3 = { otherProperty: "value" };
        const obj4 = { __typename: "TypeC", property: "value" };

        expect(isOfType(obj1, "TypeA")).toBe(true);
        expect(isOfType(obj1, "TypeB")).toBe(false);
        expect(isOfType(obj1, "TypeA", "TypeB")).toBe(true);
        expect(isOfType(obj2, "TypeA")).toBe(false);
        expect(isOfType(obj2, "TypeB")).toBe(true);
        expect(isOfType(obj2, "TypeA", "TypeB")).toBe(true);
        expect(isOfType(obj3, "TypeA")).toBe(false);
        expect(isOfType(obj4, "TypeA", "TypeB")).toBe(false);
    });
});

describe("deepClone", () => {
    it("should deep clone primitives", () => {
        expect(deepClone(42)).toBe(42);
        expect(deepClone("hello")).toBe("hello");
        expect(deepClone(true)).toBe(true);
        expect(deepClone(null)).toBe(null);
        expect(deepClone(undefined)).toBeUndefined();
    });

    it("should deep clone date objects", () => {
        const date = new Date(2000, 0, 1);
        const clonedDate = deepClone(date);

        expect(clonedDate).toEqual(date);
        expect(clonedDate).not.toBe(date); // Ensure a new Date object
    });

    it("should deep clone arrays", () => {
        const arr = [1, "hello", true, null, [2, 3, 4], { a: 1, b: 2 }];
        const clonedArr = deepClone(arr);

        expect(clonedArr).toEqual(arr);
        expect(clonedArr).not.toBe(arr); // Ensure a new array
        expect(clonedArr[4]).not.toBe(arr[4]); // Ensure nested array is cloned
        expect(clonedArr[5]).not.toBe(arr[5]); // Ensure nested object is cloned
    });

    it("should deep clone objects", () => {
        const obj = {
            number: 42,
            string: "hello",
            bool: true,
            nul: null,
            arr: [1, 2, 3],
            nested: {
                a: 1,
                b: "world",
                c: [4, 5, 6],
                d: {
                    e: "nested",
                },
            },
        };
        const clonedObj = deepClone(obj);

        expect(clonedObj).toEqual(obj);
        expect(clonedObj).not.toBe(obj); // Ensure a new object
        expect(clonedObj.arr).not.toBe(obj.arr); // Ensure nested array is cloned
        expect(clonedObj.nested).not.toBe(obj.nested); // Ensure nested object is cloned
        expect(clonedObj.nested.c).not.toBe(obj.nested.c); // Ensure nested object's array is cloned
        expect(clonedObj.nested.d).not.toBe(obj.nested.d); // Ensure nested object's object is cloned
    });
});

describe("mergeDeep", () => {
    // Cases where params are invalid
    it("should return the source object if it is not an object", () => {
        expect(mergeDeep(5, 10)).toBe(5);
        expect(mergeDeep("hello", "world")).toBe("hello");
        expect(mergeDeep(true, false)).toBe(true);
    });
    it("should return the default object if the source is null or undefined", () => {
        expect(mergeDeep(null, { a: 1 })).toEqual({ a: 1 });
        expect(mergeDeep(undefined, { a: 1 })).toEqual({ a: 1 });
    });
    it("should merge properties from the default object into the source object", () => {
        expect(mergeDeep({ a: 1 }, { b: 2 })).toEqual({ a: 1, b: 2 });
        expect(mergeDeep({ a: 1 }, { a: 2, b: 2 })).toEqual({ a: 1, b: 2 });
    });

    // Cases where params are of similar shape
    it("should handle nested objects", () => {
        expect(mergeDeep({ a: { b: 1 } }, { a: { c: 2 }, d: 3 })).toEqual({ a: { b: 1, c: 2 }, d: 3 });
        expect(mergeDeep({ a: { b: 1 } }, { a: { b: 2, c: 2 } })).toEqual({ a: { b: 1, c: 2 } });
    });
    it("should not merge arrays", () => {
        expect(mergeDeep([1, 2, 3], [4, 5, 6])).toEqual([1, 2, 3]);
        expect(mergeDeep({ a: [1, 2, 3] }, { a: [4, 5, 6] })).toEqual({ a: [1, 2, 3] });
    });
    it("should handle complex nested structures", () => {
        const source = { a: { b: 1, d: [1, 2, 3] }, e: "test" };
        const defaults = { a: { b: 2, c: 3, d: [4, 5, 6] }, e: "default", f: false };
        const expected = { a: { b: 1, c: 3, d: [1, 2, 3] }, e: "test", f: false };
        expect(mergeDeep(source, defaults)).toEqual(expected);
    });
    it("should handle empty objects and arrays", () => {
        expect(mergeDeep({}, { a: 1 })).toEqual({ a: 1 });
        expect(mergeDeep([], [1, 2, 3])).toHaveLength(0);
        expect(mergeDeep({ a: {} }, { a: { b: 2 } })).toEqual({ a: { b: 2 } });
    });

    it("should handle circular references without infinite recursion", () => {
        const circular: any = { a: 1 };
        circular.self = circular;
        
        const defaults = { b: 2 };
        
        // mergeDeep should handle circular references without throwing
        const result = mergeDeep(circular, defaults);
        expect(result.a).toBe(1);
        expect(result.b).toBe(2);
        expect(result.self).toBe(circular); // Should maintain the circular reference
        
        // Test that circular reference doesn't cause infinite loop
        const startTime = Date.now();
        mergeDeep(circular, defaults);
        const endTime = Date.now();
        expect(endTime - startTime).toBeLessThan(100); // Should complete quickly
        
        // Test nested circular reference
        const obj1: any = { name: "obj1" };
        const obj2: any = { name: "obj2", ref: obj1 };
        obj1.ref = obj2; // Create circular reference
        
        const defaults2 = { extra: "value" };
        const result2 = mergeDeep(obj1, defaults2);
        expect(result2.name).toBe("obj1");
        expect(result2.extra).toBe("value");
        // Should preserve circular structure without infinite recursion
        expect(result2.ref.name).toBe("obj2");
        // The circular reference is preserved as-is from the source
        expect(result2.ref.ref).toBe(obj1); // Points back to original obj1
        
        // Test circular reference in defaults
        const circularDefaults: any = { defaultValue: 10 };
        circularDefaults.circular = circularDefaults;
        
        const source = { sourceValue: 20 };
        const result3 = mergeDeep(source, circularDefaults);
        expect(result3.sourceValue).toBe(20);
        expect(result3.defaultValue).toBe(10);
        // Circular structure in defaults should be handled properly
        expect(result3.circular).to.be.an('object');
        
        // Test merging two objects with circular references
        const circular1: any = { value: 1 };
        circular1.self = circular1;
        const circular2: any = { value: 2, other: 3 };
        circular2.self = circular2;
        
        const result4 = mergeDeep(circular1, circular2);
        expect(result4.value).toBe(1); // Source value takes precedence
        expect(result4.other).toBe(3); // Added from defaults
        expect(result4.self).toBe(circular1); // Maintains source circular reference
    });

    it("should handle class instances by not deep merging them", () => {
        class CustomClass {
            constructor(public value: number) {}
            method() { return this.value * 2; }
        }
        
        const customObj = { instance: new CustomClass(5) };
        const defaults = { instance: new CustomClass(10), extra: "property" };
        
        const result = mergeDeep(customObj, defaults);
        
        // Should preserve the target's class instance, not merge it
        expect(result.instance).to.be.instanceOf(CustomClass);
        expect(result.instance.value).toBe(5); // From target, not defaults
        expect(result.extra).toBe("property"); // From defaults
    });


    it("should only merge enumerable properties", () => {
        const source = Object.defineProperty({}, 'hidden', {
            value: 'secret',
            enumerable: false
        });
        source.visible = 'public';
        
        const defaults = { visible: 'fallback', other: 'default' };
        const result = mergeDeep(source, defaults);
        
        // Should preserve enumerable properties from source
        expect(result.visible).toBe('public');
        // Should include properties from defaults that aren't in source
        expect(result.other).toBe('default');
        // Non-enumerable properties should not be copied during merge
        expect((result as any).hidden).toBeUndefined();
    });

    it("should handle deeply nested merging correctly", () => {
        const source = {
            level1: {
                level2: {
                    source: "value",
                    overwrite: "source",
                },
                sourceOnly: "source",
            },
        };
        
        const defaults = {
            level1: {
                level2: {
                    default: "value",
                    overwrite: "default",
                },
                defaultOnly: "default",
            },
        };
        
        const result = mergeDeep(source, defaults);
        
        expect(result).toEqual({
            level1: {
                level2: {
                    source: "value",
                    overwrite: "source",
                    default: "value",
                },
                sourceOnly: "source",
                defaultOnly: "default",
            },
        });
    });

    it("should handle merging with undefined and null values correctly", () => {
        const source = { a: null, b: undefined, c: "value" };
        const defaults = { a: "default", b: "default", d: "default" };
        
        const result = mergeDeep(source, defaults);
        
        expect(result.a).toBeNull();
        expect(result.b).toBeUndefined();
        expect(result.c).toBe("value");
        expect(result.d).toBe("default");
    });

    it("should handle function properties correctly", () => {
        const fn = () => "test";
        const source = { func: fn, value: "source" };
        const defaults = { func: () => "default", other: "default" };
        
        const result = mergeDeep(source, defaults);
        
        expect(result.func).toBe(fn);
        expect(result.value).toBe("source");
        expect(result.other).toBe("default");
    });
});
