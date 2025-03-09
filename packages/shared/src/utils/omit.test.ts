import { expect } from "chai";
import { omit } from "./omit.js";

describe("omit", () => {
    it("should omit top-level keys from an object", () => {
        const obj = { a: 1, b: 2, c: 3 };
        expect(omit(obj, ["a"])).to.deep.equal({ b: 2, c: 3 });
        expect(omit(obj, ["a", "b"])).to.deep.equal({ c: 3 });
    });

    it("should omit nested keys from an object using dot notation", () => {
        const obj = { a: { b: 2, c: 3 }, d: 4 };
        expect(omit(obj, ["a.b"])).to.deep.equal({ a: { c: 3 }, d: 4 });
        expect(omit(obj, ["a.b", "a.c"])).to.deep.equal({ d: 4 });
    });

    it("should remove parent objects if they are empty after omitting keys", () => {
        const obj = { a: { b: { c: 3, d: 4 } }, e: 5 };
        expect(omit(obj, ["a.b.c", "a.b.d"])).to.deep.equal({ e: 5 });
    });

    it("should handle cases where keys are not present in the object", () => {
        const obj = { a: 1, b: 2, c: 3 };
        expect(omit(obj, ["d"])).to.deep.equal({ a: 1, b: 2, c: 3 });
        expect(omit(obj, ["a.d"])).to.deep.equal({ a: 1, b: 2, c: 3 });
    });

    it("should return a new object and not modify the original object", () => {
        const obj = { a: 1, b: 2, c: 3 };
        const result = omit(obj, ["a"]);
        expect(result).not.to.equal(obj);
        expect(obj).to.deep.equal({ a: 1, b: 2, c: 3 });
    });

    it("should handle empty arrays of keys to omit", () => {
        const obj = { a: 1, b: 2, c: 3 };
        expect(omit(obj, [])).to.deep.equal(obj);
    });

    it("should handle deeply nested objects", () => {
        const obj = { a: { b: { c: { d: 4, e: 5 } } } };
        expect(omit(obj, ["a.b.c.d"])).to.deep.equal({ a: { b: { c: { e: 5 } } } });
        expect(omit(obj, ["a.b.c.d", "a.b.c.e"])).to.deep.equal({});
    });
});

