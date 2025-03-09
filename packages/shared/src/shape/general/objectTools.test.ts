/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import { uuid } from "../../id/uuid.js";
import { convertToDot, hasObjectChanged, valueFromDot } from "./objectTools.js";

describe("valueFromDot function tests", () => {
    it("valid path to a primitive value", () => {
        const obj = { parent: { child: { property: "value" } } };
        const result = valueFromDot(obj, "parent.child.property");
        expect(result).to.equal("value");
    });

    it("valid path to an object", () => {
        const obj = { parent: { child: { property: "value" } } };
        const result = valueFromDot(obj, "parent.child");
        expect(result).to.deep.equal({ property: "value" });
    });

    it("valid path to an array element", () => {
        const obj = { array: ["first", "second", "third"] };
        const result = valueFromDot(obj, "array.1");
        expect(result).to.equal("second");
    });

    it("non-existent path", () => {
        const obj = { parent: { child: { property: "value" } } };
        const result = valueFromDot(obj, "parent.unknown.property");
        expect(result).to.be.null;
    });

    it("path leads to undefined", () => {
        const obj = { parent: { child: undefined } };
        const result = valueFromDot(obj, "parent.child");
        expect(result).to.be.null;
    });

    it("empty object with valid path", () => {
        const obj = {};
        const result = valueFromDot(obj, "parent.child");
        expect(result).to.be.null;
    });

    it("valid object with empty path", () => {
        const obj = { parent: { child: { property: "value" } } };
        const result = valueFromDot(obj, "");
        expect(result).to.be.null;
    });

    it("null object and path", () => {
        // @ts-ignore: Testing runtime scenario
        const result = valueFromDot(null, null);
        expect(result).to.be.null;
    });

    it("undefined object and path", () => {
        // @ts-ignore: Testing runtime scenario
        const result = valueFromDot(undefined, undefined);
        expect(result).to.be.null;
    });

    it("nested objects", () => {
        const obj = { a: { b: { c: { d: "value" } } } };
        const result = valueFromDot(obj, "a.b.c.d");
        expect(result).to.equal("value");
    });

    it("nested arrays", () => {
        const obj = { a: [[["value"]]] };
        const result = valueFromDot(obj, "a.0.0.0");
        expect(result).to.equal("value");
    });

    it("combination of nested objects and arrays", () => {
        const obj = { a: { b: [{ c: "value" }] } };
        const result = valueFromDot(obj, "a.b.0.c");
        expect(result).to.equal("value");
    });
});

describe("convertToDot function tests", () => {
    it("simple object without nested objects", () => {
        const obj = { a: 1, b: 2 };
        const result = convertToDot(obj);
        expect(result).to.deep.equal({ "a": 1, "b": 2 });
    });

    it("nested object", () => {
        const obj = { a: { b: 2 } };
        const result = convertToDot(obj);
        expect(result).to.deep.equal({ "a.b": 2 });
    });

    it("multiple levels of nesting", () => {
        const obj = { a: { b: { c: 3 } } };
        const result = convertToDot(obj);
        expect(result).to.deep.equal({ "a.b.c": 3 });
    });

    it("object with array", () => {
        const obj = { a: [1, 2, 3] };
        const result = convertToDot(obj);
        expect(result).to.deep.equal({ "a.0": 1, "a.1": 2, "a.2": 3 });
    });

    it("nested objects within an array", () => {
        const obj = { a: [{ b: 2 }, { c: 3 }] };
        const result = convertToDot(obj);
        expect(result).to.deep.equal({ "a.0.b": 2, "a.1.c": 3 });
    });

    it("empty object", () => {
        const obj = {};
        const result = convertToDot(obj);
        expect(result).to.deep.equal({});
    });

    it("object with null values", () => {
        const obj = { a: null, b: { c: null } };
        const result = convertToDot(obj);
        expect(result).to.deep.equal({ "a": null, "b.c": null });
    });

    it("object with various types of values", () => {
        const obj = { a: "string", b: 123, c: true };
        const result = convertToDot(obj);
        expect(result).to.deep.equal({ "a": "string", "b": 123, "c": true });
    });
});

describe("hasObjectChanged", () => {
    it("should return false for identical objects", () => {
        const original = { a: 1, b: 2 };
        expect(hasObjectChanged(original, { ...original })).to.equal(false);
    });

    it("should return true for different objects", () => {
        const original = { a: 1, b: 2 };
        const updated = { a: 1, b: 3 };
        expect(hasObjectChanged(original, updated)).to.equal(true);
    });

    it("should return false for identical nested objects", () => {
        const original = { a: { c: 3 }, b: 2 };
        expect(hasObjectChanged(original, { ...original })).to.equal(false);
    });

    it("should return true for different nested objects", () => {
        const original = { a: { c: 3 }, b: 2 };
        const updated = { a: { c: 4 }, b: 2 };
        expect(hasObjectChanged(original, updated)).to.equal(true);
    });

    it("should return true when a field is added to the object", () => {
        const original = { a: 1 };
        const updated = { a: 1, b: 2 };
        expect(hasObjectChanged(original, updated)).to.equal(true);
    });

    it("should return true when a field is removed from the object", () => {
        const original = { a: 1, b: 2 };
        const updated = { a: 1 };
        expect(hasObjectChanged(original, updated)).to.equal(true);
    });

    it("should return false for identical objects with arrays", () => {
        const original = { a: [1, 2, 3] };
        const updated = { a: [1, 2, 3] };
        expect(hasObjectChanged(original, updated)).to.equal(false);
    });

    it("should return true for different objects with arrays", () => {
        const original = { a: [1, 2, 3] };
        const updated = { a: [1, 2, 4] };
        expect(hasObjectChanged(original, updated)).to.equal(true);
    });

    it("should return true for objects with arrays of different lengths", () => {
        const original = { a: [1, 2, 3] };
        const updated = { a: [1, 2] };
        expect(hasObjectChanged(original, updated)).to.equal(true);
    });

    it("should return false for unchanged fields specified in dot notation", () => {
        const original = { a: { b: { c: 1, d: "boop" } } };
        const updated = { a: { b: { c: 1, d: "beep" } } };
        expect(hasObjectChanged(original, updated, ["a.b.c"])).to.equal(false);
    });

    it("should return true for changed fields specified in dot notation", () => {
        const original = { a: { b: { c: 1 } } };
        const updated = { a: { b: { c: 2 } } };
        expect(hasObjectChanged(original, updated, ["a.b.c"])).to.equal(true);
    });

    it("should return false for non-existent nested fields specified in dot notation", () => {
        const original = { a: { b: 1 } };
        const updated = { a: { b: 1 } };
        expect(hasObjectChanged(original, updated, ["a.c"])).to.equal(false);
    });

    it("should handle multiple fields in dot notation", () => {
        const original = { a: { b: 1, d: 4 }, c: 3 };
        const updated = { a: { b: 2, d: 4 }, c: 3 };
        expect(hasObjectChanged(original, updated, ["a.b", "c"])).to.equal(true);
        expect(hasObjectChanged(original, updated, ["a.d", "c"])).to.equal(false);
    });

    it("should return true when a nested field is added", () => {
        const original = { a: { b: 1 } };
        const updated = { a: { b: 1, c: 2 } };
        expect(hasObjectChanged(original, updated)).to.equal(true);
    });

    it("should return true when a nested field is removed", () => {
        const original = { a: { b: 1, c: 2 } };
        const updated = { a: { b: 1 } };
        expect(hasObjectChanged(original, updated)).to.equal(true);
    });

    it("should return false for identical objects with nested arrays", () => {
        const original = { a: [{ b: 1 }, { b: 2 }] };
        const updated = { a: [{ b: 1 }, { b: 2 }] };
        expect(hasObjectChanged(original, updated)).to.equal(false);
    });

    it("should return true for objects with different nested arrays", () => {
        const original = { a: [{ b: 1 }, { b: 2 }] };
        const updated = { a: [{ b: 1 }, { b: 3 }] };
        expect(hasObjectChanged(original, updated)).to.equal(true);
    });

    it("should return true for objects with nested arrays of different lengths", () => {
        const original = { a: [{ b: 1 }, { b: 2 }] };
        const updated = { a: [{ b: 1 }] };
        expect(hasObjectChanged(original, updated)).to.equal(true);
    });

    it("should handle empty fields array", () => {
        const original = { a: 1, b: 2 };
        const updated = { a: 1, b: 3 };
        expect(hasObjectChanged(original, updated, [])).to.equal(true);
    });

    it("should return false for unchanged objects with empty fields array", () => {
        const original = { a: 1, b: 2 };
        const updated = { a: 1, b: 2 };
        expect(hasObjectChanged(original, updated, [])).to.equal(false);
    });

    it("should return false for null or undefined updated object", () => {
        const original = { a: 1 };
        // @ts-ignore: Testing runtime scenario
        expect(hasObjectChanged(original, null)).to.equal(false);
        // @ts-ignore: Testing runtime scenario
        expect(hasObjectChanged(original, undefined)).to.equal(false);
    });

    it("should return true for null or undefined original object", () => {
        const updated = { a: 1 };
        // @ts-ignore: Testing runtime scenario
        expect(hasObjectChanged(null, updated)).to.equal(true);
        // @ts-ignore: Testing runtime scenario
        expect(hasObjectChanged(undefined, updated)).to.equal(true);
    });

    it("should return false for deeply identical nested objects", () => {
        const original = {
            owner: {
                id: uuid(),
                name: "Bugs Bunny",
                you: {
                    canBookmark: true,
                    isViewed: true,
                },
            },
        };
        const updated = JSON.parse(JSON.stringify(original)); // Deep clone
        expect(hasObjectChanged(original, updated)).to.equal(false);
    });
});
