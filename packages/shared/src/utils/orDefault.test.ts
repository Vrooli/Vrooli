import { describe, it, expect } from "vitest";
import { orDefault } from "./orDefault.js";

describe("orDefault", () => {
    it("should return the default array if the existing array is empty", () => {
        const defaultArray = [1, 2, 3];
        expect(orDefault([], defaultArray)).to.deep.equal(defaultArray);
    });

    it("should return the existing array if it is non-empty", () => {
        const existingArray = [4, 5, 6];
        expect(orDefault(existingArray, [1, 2, 3])).to.deep.equal(existingArray);
    });

    it("should return the default object if the existing object is empty", () => {
        const defaultObj = { a: 1, b: 2 };
        expect(orDefault({}, defaultObj)).to.deep.equal(defaultObj);
    });

    it("should return the existing object if it is non-empty", () => {
        const existingObj = { a: 10, y: 20 };
        expect(orDefault(existingObj, { a: 1, b: 2 })).to.deep.include({ a: 10, b: 2 });
    });

    it("should return the default value if the existing value is null or undefined", () => {
        expect(orDefault(null, [1, 2, 3])).to.deep.equal([1, 2, 3]);
        expect(orDefault(undefined, { a: 1, b: 2 })).to.deep.equal({ a: 1, b: 2 });
    });

    it("should fill in missing nested properties from the default object", () => {
        const existingObj = { a: { x: 10 } };
        const defaultObj = { a: { x: 1, y: 2 }, b: 3 };
        const expectedObj = { a: { x: 10, y: 2 }, b: 3 };
        expect(orDefault(existingObj, defaultObj)).to.deep.equal(expectedObj);
    });

    it("should handle arrays within objects", () => {
        const existingObj = { a: [1, 2], b: { x: 10 } };
        const defaultObj = { a: [3, 4, 5], b: { x: 1, y: 2 } };
        const expectedObj = { a: [1, 2], b: { x: 10, y: 2 } };
        expect(orDefault(existingObj, defaultObj)).to.deep.equal(expectedObj);
    });

    it("should handle objects within arrays", () => {
        const existingArray = [{ a: 1 }, { b: 2 }];
        const defaultArray = [{ a: 0, b: 0 }];
        const expectedArray = [{ a: 1, b: 0 }, { a: 0, b: 2 }];
        expect(orDefault(existingArray, defaultArray)).to.deep.equal(expectedArray);
    });

    it("should handle deeply nested structures", () => {
        const existingObj = { a: { x: { y: 1 } } };
        const defaultObj = { a: { x: { y: 0, z: 2 }, w: 3 }, b: 4 };
        const expectedObj = { a: { x: { y: 1, z: 2 }, w: 3 }, b: 4 };
        expect(orDefault(existingObj, defaultObj)).to.deep.equal(expectedObj);
    });

    it("should return the existing array of primitives if non-empty", () => {
        const existingArray = [1, 2, 3];
        const defaultArray = [4, 5, 6];
        expect(orDefault(existingArray, defaultArray)).to.deep.equal(existingArray);
    });

    it("should return the default array of objects if the existing array is empty", () => {
        const existingArray: Array<{ a: number, b: number }> = [];
        const defaultArray = [{ a: 1, b: 2 }];
        expect(orDefault(existingArray, defaultArray)).to.deep.equal(defaultArray);
    });
});

