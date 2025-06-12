/* eslint-disable @typescript-eslint/ban-ts-comment */
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
        expect(getDotNotationValue(obj, "foo.bar.baz")).to.equal(42);
        expect(getDotNotationValue(obj, "foo.arr[1]")).to.equal(2);
        expect(getDotNotationValue(obj, "foo.arr[2]")).to.equal("3");
        expect(getDotNotationValue(obj, "foo.arr[3]")).to.be.undefined;
        expect(getDotNotationValue(obj, "foo.nonExistent")).to.be.undefined;
        expect(getDotNotationValue(obj, "foo.nullValue")).to.be.null;
        expect(getDotNotationValue(obj, "foo.undefinedValue")).to.be.undefined;
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
        expect(getDotNotationValue(obj, "foo.bar.baz.")).to.equal(42);
        expect(getDotNotationValue(obj, "foo.arr[0].")).to.equal(1);
        expect(getDotNotationValue(obj, "foo.arr[1].")).to.equal(2);
        expect(getDotNotationValue(obj, "foo.arr[2].")).to.equal("3");
        expect(getDotNotationValue(obj, "foo.arr[3].")).to.be.undefined;
        expect(getDotNotationValue(obj, "foo.nullValue.")).to.be.null;
        expect(getDotNotationValue(obj, "foo.undefinedValue.")).to.be.undefined;
    });

    it("should handle edge cases", () => {
        expect(getDotNotationValue(undefined, "foo")).to.be.undefined;
        expect(getDotNotationValue({}, "foo")).to.be.undefined;
    });

    it("should handle invalid dot notation paths", () => {
        const obj = { foo: "bar", baz: 42 };
        // A single dot is not a valid dot notation path and should return undefined
        expect(getDotNotationValue(obj, ".")).to.be.undefined;
        // Empty path should return undefined
        expect(getDotNotationValue(obj, "")).to.be.undefined;
        // Path with only dots should return undefined
        expect(getDotNotationValue(obj, "..")).to.be.undefined;
        expect(getDotNotationValue(obj, "...")).to.be.undefined;
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
        expect(obj.foo.bar.baz).to.equal(100);

        setDotNotationValue(obj, "foo.arr[1]", 10);
        expect(obj.foo.arr[1]).to.equal(10);

        setDotNotationValue(obj, "foo.arr[3]", 4);
        expect(obj.foo.arr[3]).to.equal(4);

        setDotNotationValue(obj, "foo.newProp", "newValue");
        expect(obj.foo.newProp).to.equal("newValue");
    });

    it("should handle edge cases", () => {
        const obj1: any = {};
        setDotNotationValue(obj1, "", "value");
        expect(obj1).to.deep.equal({});

        const obj2 = { foo: 42 };
        setDotNotationValue(obj2, "foo", "updatedValue");
        expect(obj2.foo).to.equal("updatedValue");
    });

    it("should handle numeric keys on non-array objects", () => {
        const obj: Record<string, any> = {
            data: { someField: "value" }
        };
        // Set a numeric key on a non-array object
        setDotNotationValue(obj, "data[0]", "newValue");
        expect(obj.data[0]).to.equal("newValue");
    });

    it("should create nested structures when setting numeric keys", () => {
        const obj: Record<string, any> = {};
        // This will create nested objects when using numeric indices
        setDotNotationValue(obj, "foo[0][1]", "value");
        expect(obj.foo[0][1]).to.equal("value");
    });

    it("should throw error when trying to access properties on non-objects", () => {
        const obj: Record<string, any> = {
            foo: "stringValue"
        };
        // Try to set a property on a string value (should throw)
        expect(() => setDotNotationValue(obj, "foo.bar", "value")).to.throw("Expected object for property access");
    });

    it("should throw error when using numeric key on non-object", () => {
        const obj: Record<string, any> = {
            foo: 42
        };
        // Try to use numeric key on a number (should throw)
        expect(() => setDotNotationValue(obj, "foo[0]", "value")).to.throw("Expected object for property access");
    });
});

describe("splitDotNotation", () => {
    it("splits standard dot notation strings correctly", () => {
        const input = ["first.second.third", "another.example.string"];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).to.deep.equal(["first", "another"]);
        expect(remainders).to.deep.equal(["second.third", "example.string"]);
    });

    it("handles empty strings", () => {
        const input = [""];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).to.deep.equal([""]);
        expect(remainders).to.deep.equal([""]);
    });

    it("handles strings without dots", () => {
        const input = ["first", "second"];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).to.deep.equal(["first", "second"]);
        expect(remainders).to.deep.equal(["", ""]);
    });

    it("handles strings with multiple dots", () => {
        const input = ["first.second.third", "another.one.two.three"];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).to.deep.equal(["first", "another"]);
        expect(remainders).to.deep.equal(["second.third", "one.two.three"]);
    });

    it("handles an empty array", () => {
        const input: string[] = [];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).to.have.lengthOf(0);
        expect(remainders).to.have.lengthOf(0);
    });

    it("handles an array with only empty strings", () => {
        const input = ["", ""];
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).to.deep.equal(["", ""]);
        expect(remainders).to.deep.equal(["", ""]);
    });

    it("should treat null as empty string for safety", () => {
        const input = [null, "valid.string", null];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input);
        // null values should be treated as empty strings to avoid errors
        expect(fields).to.deep.equal(["", "valid", ""]);
        expect(remainders).to.deep.equal(["", "string", ""]);
    });

    it("should treat undefined as empty string for safety", () => {
        const input = [undefined, "test.value", undefined];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input);
        // undefined values should be treated as empty strings to avoid errors
        expect(fields).to.deep.equal(["", "test", ""]);
        expect(remainders).to.deep.equal(["", "value", ""]);
    });

    it("handles mixed valid and invalid inputs", () => {
        const input = [null, "first.second", "", undefined, "another.one"];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).to.deep.equal(["", "first", "", "", "another"]);
        expect(remainders).to.deep.equal(["", "second", "", "", "one"]);
    });

    it("removes empty strings when removeEmpty is true", () => {
        const input = [null, "first.second", "", undefined, "another.one"];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input, true);
        expect(fields).to.deep.equal(["first", "another"]);
        expect(remainders).to.deep.equal(["second", "one"]);
    });

    it("includes empty strings when removeEmpty is false", () => {
        const input = [null, "first.second", "", undefined, "another.one"];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input, false);
        expect(fields).to.deep.equal(["", "first", "", "", "another"]);
        expect(remainders).to.deep.equal(["", "second", "", "", "one"]);
    });

    it("defaults to including empty strings when removeEmpty is not provided", () => {
        const input = ["", "first.second", null, "another.one", undefined];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).to.deep.equal(["", "first", "", "another", ""]);
        expect(remainders).to.deep.equal(["", "second", "", "one", ""]);
    });
});


describe("isObject", () => {
    it("should identify objects correctly", () => {
        expect(isObject({})).to.equal(true);
        expect(isObject([])).to.equal(true);
        expect(isObject(new Date())).to.equal(true);
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        expect(isObject(() => { })).to.equal(true);
        expect(isObject(null)).to.equal(false);
        expect(isObject(undefined)).to.equal(false);
        expect(isObject(42)).to.equal(false);
        expect(isObject("string")).to.equal(false);
        expect(isObject(true)).to.equal(false);
    });
});

describe("isOfType", () => {
    it("should identify types correctly", () => {
        const obj1 = { __typename: "TypeA", property: "value" };
        const obj2 = { __typename: "TypeB", property: "value" };
        const obj3 = { otherProperty: "value" };
        const obj4 = { __typename: "TypeC", property: "value" };

        expect(isOfType(obj1, "TypeA")).to.equal(true);
        expect(isOfType(obj1, "TypeB")).to.equal(false);
        expect(isOfType(obj1, "TypeA", "TypeB")).to.equal(true);
        expect(isOfType(obj2, "TypeA")).to.equal(false);
        expect(isOfType(obj2, "TypeB")).to.equal(true);
        expect(isOfType(obj2, "TypeA", "TypeB")).to.equal(true);
        expect(isOfType(obj3, "TypeA")).to.equal(false);
        expect(isOfType(obj4, "TypeA", "TypeB")).to.equal(false);
    });
});

describe("deepClone", () => {
    it("should deep clone primitives", () => {
        expect(deepClone(42)).to.equal(42);
        expect(deepClone("hello")).to.equal("hello");
        expect(deepClone(true)).to.equal(true);
        expect(deepClone(null)).to.equal(null);
        expect(deepClone(undefined)).to.be.undefined;
    });

    it("should deep clone date objects", () => {
        const date = new Date(2000, 0, 1);
        const clonedDate = deepClone(date);

        expect(clonedDate).to.deep.equal(date);
        expect(clonedDate).not.to.equal(date); // Ensure a new Date object
    });

    it("should deep clone arrays", () => {
        const arr = [1, "hello", true, null, [2, 3, 4], { a: 1, b: 2 }];
        const clonedArr = deepClone(arr);

        expect(clonedArr).to.deep.equal(arr);
        expect(clonedArr).not.to.equal(arr); // Ensure a new array
        expect(clonedArr[4]).not.to.equal(arr[4]); // Ensure nested array is cloned
        expect(clonedArr[5]).not.to.equal(arr[5]); // Ensure nested object is cloned
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

        expect(clonedObj).to.deep.equal(obj);
        expect(clonedObj).not.to.equal(obj); // Ensure a new object
        expect(clonedObj.arr).not.to.equal(obj.arr); // Ensure nested array is cloned
        expect(clonedObj.nested).not.to.equal(obj.nested); // Ensure nested object is cloned
        expect(clonedObj.nested.c).not.to.equal(obj.nested.c); // Ensure nested object's array is cloned
        expect(clonedObj.nested.d).not.to.equal(obj.nested.d); // Ensure nested object's object is cloned
    });
});

describe("mergeDeep", () => {
    // Cases where params are invalid
    it("should return the source object if it is not an object", () => {
        expect(mergeDeep(5, 10)).to.equal(5);
        expect(mergeDeep("hello", "world")).to.equal("hello");
        expect(mergeDeep(true, false)).to.equal(true);
    });
    it("should return the default object if the source is null or undefined", () => {
        expect(mergeDeep(null, { a: 1 })).to.deep.equal({ a: 1 });
        expect(mergeDeep(undefined, { a: 1 })).to.deep.equal({ a: 1 });
    });
    it("should merge properties from the default object into the source object", () => {
        expect(mergeDeep({ a: 1 }, { b: 2 })).to.deep.equal({ a: 1, b: 2 });
        expect(mergeDeep({ a: 1 }, { a: 2, b: 2 })).to.deep.equal({ a: 1, b: 2 });
    });

    // Cases where params are of similar shape
    it("should handle nested objects", () => {
        expect(mergeDeep({ a: { b: 1 } }, { a: { c: 2 }, d: 3 })).to.deep.equal({ a: { b: 1, c: 2 }, d: 3 });
        expect(mergeDeep({ a: { b: 1 } }, { a: { b: 2, c: 2 } })).to.deep.equal({ a: { b: 1, c: 2 } });
    });
    it("should not merge arrays", () => {
        expect(mergeDeep([1, 2, 3], [4, 5, 6])).to.deep.equal([1, 2, 3]);
        expect(mergeDeep({ a: [1, 2, 3] }, { a: [4, 5, 6] })).to.deep.equal({ a: [1, 2, 3] });
    });
    it("should handle complex nested structures", () => {
        const source = { a: { b: 1, d: [1, 2, 3] }, e: "test" };
        const defaults = { a: { b: 2, c: 3, d: [4, 5, 6] }, e: "default", f: false };
        const expected = { a: { b: 1, c: 3, d: [1, 2, 3] }, e: "test", f: false };
        expect(mergeDeep(source, defaults)).to.deep.equal(expected);
    });
    it("should handle empty objects and arrays", () => {
        expect(mergeDeep({}, { a: 1 })).to.deep.equal({ a: 1 });
        expect(mergeDeep([], [1, 2, 3])).to.have.lengthOf(0);
        expect(mergeDeep({ a: {} }, { a: { b: 2 } })).to.deep.equal({ a: { b: 2 } });
    });

    it("should handle circular references without infinite recursion", () => {
        const circular: any = { a: 1 };
        circular.self = circular;
        
        const defaults = { b: 2 };
        
        // mergeDeep should handle circular references without throwing
        const result = mergeDeep(circular, defaults);
        expect(result.a).to.equal(1);
        expect(result.b).to.equal(2);
        expect(result.self).to.equal(circular); // Should maintain the circular reference
        
        // Test that circular reference doesn't cause infinite loop
        const startTime = Date.now();
        mergeDeep(circular, defaults);
        const endTime = Date.now();
        expect(endTime - startTime).to.be.lessThan(100); // Should complete quickly
        
        // Test nested circular reference
        const obj1: any = { name: "obj1" };
        const obj2: any = { name: "obj2", ref: obj1 };
        obj1.ref = obj2; // Create circular reference
        
        const defaults2 = { extra: "value" };
        const result2 = mergeDeep(obj1, defaults2);
        expect(result2.name).to.equal("obj1");
        expect(result2.extra).to.equal("value");
        // Should preserve circular structure without infinite recursion
        expect(result2.ref.name).to.equal("obj2");
        expect(result2.ref.ref).to.equal(result2); // Should point back to result2
        
        // Test circular reference in defaults
        const circularDefaults: any = { defaultValue: 10 };
        circularDefaults.circular = circularDefaults;
        
        const source = { sourceValue: 20 };
        const result3 = mergeDeep(source, circularDefaults);
        expect(result3.sourceValue).to.equal(20);
        expect(result3.defaultValue).to.equal(10);
        // Circular structure in defaults should be handled properly
        expect(result3.circular).to.be.an('object');
        
        // Test merging two objects with circular references
        const circular1: any = { value: 1 };
        circular1.self = circular1;
        const circular2: any = { value: 2, other: 3 };
        circular2.self = circular2;
        
        const result4 = mergeDeep(circular1, circular2);
        expect(result4.value).to.equal(1); // Source value takes precedence
        expect(result4.other).to.equal(3); // Added from defaults
        expect(result4.self).to.equal(circular1); // Maintains source circular reference
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
        expect(result.instance.value).to.equal(5); // From target, not defaults
        expect(result.extra).to.equal("property"); // From defaults
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
        expect(result.visible).to.equal('public');
        // Should include properties from defaults that aren't in source
        expect(result.other).to.equal('default');
        // Non-enumerable properties should not be copied during merge
        expect((result as any).hidden).to.be.undefined;
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
        
        expect(result).to.deep.equal({
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
        
        expect(result.a).to.be.null;
        expect(result.b).to.be.undefined;
        expect(result.c).to.equal("value");
        expect(result.d).to.equal("default");
    });

    it("should handle function properties correctly", () => {
        const fn = () => "test";
        const source = { func: fn, value: "source" };
        const defaults = { func: () => "default", other: "default" };
        
        const result = mergeDeep(source, defaults);
        
        expect(result.func).to.equal(fn);
        expect(result.value).to.equal("source");
        expect(result.other).to.equal("default");
    });
});
