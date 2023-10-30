import { deepClone, getDotNotationValue, isObject, isOfType, setDotNotationValue } from "./objects";

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