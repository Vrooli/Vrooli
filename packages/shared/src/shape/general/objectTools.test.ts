/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import { convertToDot, hasObjectChanged, valueFromDot } from "./objectTools.js";

describe("valueFromDot", () => {
    it("should handle empty path", () => {
        const obj = { test: "value" };
        expect(valueFromDot(obj, "")).to.be.null;
    });

    it("should handle single level path", () => {
        const obj = { test: "value" };
        expect(valueFromDot(obj, "test")).to.equal("value");
    });

    it("should handle multi level path", () => {
        const obj = { level1: { level2: { level3: "value" } } };
        expect(valueFromDot(obj, "level1.level2.level3")).to.equal("value");
    });

    it("should handle undefined values", () => {
        const obj = { test: undefined };
        expect(valueFromDot(obj, "test")).to.be.null;
    });

    it("should handle non-existent paths", () => {
        const obj = { test: "value" };
        expect(valueFromDot(obj, "nonexistent")).to.be.null;
    });
});

describe("convertToDot", () => {
    it("should handle empty object", () => {
        expect(convertToDot({})).to.deep.equal({});
    });

    it("should handle single level object", () => {
        const obj = { test: "value" };
        expect(convertToDot(obj)).to.deep.equal({ "test": "value" });
    });

    it("should handle multi level object", () => {
        const obj = { level1: { level2: { level3: "value" } } };
        expect(convertToDot(obj)).to.deep.equal({ "level1.level2.level3": "value" });
    });

    it("should handle multiple paths", () => {
        const obj = { path1: "value1", path2: { nested: "value2" } };
        expect(convertToDot(obj)).to.deep.equal({
            "path1": "value1",
            "path2.nested": "value2",
        });
    });
});

describe("hasObjectChanged", () => {
    it("should detect changes in simple values", () => {
        const original = { a: 1, b: 2 };
        const updated = { a: 1, b: 3 };
        expect(hasObjectChanged(original, updated)).to.be.true;
    });

    it("should detect changes in nested objects", () => {
        const original = { a: { x: 1 }, b: 2 };
        const updated = { a: { x: 2 }, b: 2 };
        expect(hasObjectChanged(original, updated)).to.be.true;
    });

    it("should detect changes in arrays", () => {
        const original = { arr: [1, 2, 3] };
        const updated = { arr: [1, 2, 4] };
        expect(hasObjectChanged(original, updated)).to.be.true;
    });

    it("should detect changes in specific fields", () => {
        const original = { a: 1, b: 2, c: 3 };
        const updated = { a: 1, b: 3, c: 3 };
        expect(hasObjectChanged(original, updated, ["b"])).to.be.true;
        expect(hasObjectChanged(original, updated, ["a"])).to.be.false;
    });

    it("should handle null and undefined", () => {
        expect(hasObjectChanged(null, { a: 1 })).to.be.true;
        expect(hasObjectChanged({ a: 1 }, null)).to.be.false;
        expect(hasObjectChanged(undefined, { a: 1 })).to.be.true;
        expect(hasObjectChanged({ a: 1 }, undefined)).to.be.false;
    });
});
