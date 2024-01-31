/* eslint-disable @typescript-eslint/ban-ts-comment */
import { deepClone, getDotNotationValue, isObject, isOfType, mergeDeep, setDotNotationValue, splitDotNotation } from "./objects";

describe("getDotNotationValue", () => {
    it("should retrieve values using dot notation", () => {
        const obj = {
            foo: {
                bar: {
                    baz: 42,
                },
                arr: [1, 2, "3"],
            },
        };
        expect(getDotNotationValue(obj, "foo.bar.baz")).toBe(42);
        expect(getDotNotationValue(obj, "foo.arr[1]")).toBe(2);
        expect(getDotNotationValue(obj, "foo.arr[2]")).toBe("3");
        expect(getDotNotationValue(obj, "foo.arr[3]")).toBeUndefined();
        expect(getDotNotationValue(obj, "foo.nonExistent")).toBeUndefined();
    });

    it("should handle edge cases", () => {
        expect(getDotNotationValue(undefined, "foo")).toBeUndefined();
        expect(getDotNotationValue({}, "foo")).toBeUndefined();
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

    it("handles null input", () => {
        const input = [null, "valid.string", null];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input);
        expect(fields).toEqual(["", "valid", ""]);
        expect(remainders).toEqual(["", "string", ""]);
    });

    it("handles undefined input", () => {
        const input = [undefined, "test.value", undefined];
        // @ts-ignore: Testing runtime scenario
        const [fields, remainders] = splitDotNotation(input);
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
});
