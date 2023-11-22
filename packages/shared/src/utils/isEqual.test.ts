import { isEqual } from "./isEqual";

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
});
