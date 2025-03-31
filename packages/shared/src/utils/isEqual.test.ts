import { expect } from "chai";
import { isEqual } from "./isEqual.js";

describe("isEqual", () => {
    it("should return true for two identical primitive values", () => {
        expect(isEqual(5, 5)).to.equal(true);
        expect(isEqual("test", "test")).to.equal(true);
        expect(isEqual(true, true)).to.equal(true);
        expect(isEqual(undefined, undefined)).to.equal(true);
    });

    it("should return false for two different primitive values", () => {
        expect(isEqual(5, 6)).to.equal(false);
        expect(isEqual("test", "test1")).to.equal(false);
        expect(isEqual(true, false)).to.equal(false);
    });

    it("should return true for two identical arrays", () => {
        expect(isEqual([1, 2, 3], [1, 2, 3])).to.equal(true);
        expect(isEqual(["a", "b", "c"], ["a", "b", "c"])).to.equal(true);
        expect(isEqual([true, false], [true, false])).to.equal(true);
    });

    it("should return false for two different arrays", () => {
        expect(isEqual([1, 2, 3], [1, 2])).to.equal(false);
        expect(isEqual(["a", "b"], ["a", "b", "c"])).to.equal(false);
        expect(isEqual([true, false], [false, true])).to.equal(false);
    });

    it("should return true for two identical objects", () => {
        expect(isEqual({ a: 1, b: 2 }, { a: 1, b: 2 })).to.equal(true);
        expect(isEqual({ name: "John", age: 25 }, { name: "John", age: 25 })).to.equal(true);
    });

    it("should return false for two different objects", () => {
        expect(isEqual({ a: 1, b: 2 }, { a: 1 })).to.equal(false);
        expect(isEqual({ name: "John", age: 25 }, { name: "Doe", age: 25 })).to.equal(false);
    });

    it("should return true for nested structures that are the same", () => {
        const obj1 = { a: [1, 2, { x: 10, y: 20 }], b: "test" };
        const obj2 = { a: [1, 2, { x: 10, y: 20 }], b: "test" };
        expect(isEqual(obj1, obj2)).to.equal(true);
    });

    it("should return false for nested structures that are different", () => {
        const obj1 = { a: [1, 2, { x: 10, y: 20 }], b: "test" };
        const obj2 = { a: [1, 2, { x: 10, y: 30 }], b: "test" };
        expect(isEqual(obj1, obj2)).to.equal(false);
    });
});
