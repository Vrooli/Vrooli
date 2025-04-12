/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import { isRelationshipArray, isRelationshipObject } from "./isOfType.js";

describe("isRelationshipObject", () => {
    const validCases = [
        { description: "an object", value: { key: "value" } },
        { description: "an empty object", value: {} },
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        { description: "a function", value() { } },
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        { description: "an arrow function", value: () => { } },
    ];

    const invalidCases = [
        { description: "null", value: null },
        { description: "undefined", value: undefined },
        { description: "an array", value: [1, 2, 3] },
        { description: "an empty array", value: [] },
        { description: "a Date object", value: new Date() },
        { description: "a string", value: "string" },
        { description: "a number", value: 123 },
        { description: "a boolean", value: true },
        { description: "a Symbol", value: Symbol("sym") },
        { description: "BigInt", value: BigInt(123) },
    ];

    validCases.forEach(({ description, value }) => {
        it(`should return true for ${description}`, () => {
            expect(isRelationshipObject(value)).to.equal(true);
        });
    });

    invalidCases.forEach(({ description, value }) => {
        it(`should return false for ${description}`, () => {
            expect(isRelationshipObject(value)).to.equal(false);
        });
    });
});

describe("isRelationshipArray", () => {
    const validCases = [
        { description: "an array of objects", value: [{ key: "value" }, {}] },
        { description: "an array with a single object", value: [{}] },
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        { description: "an array of functions", value: [() => { }, function () { }] },
        { description: "an empty array", value: [] },
    ];

    const invalidCases = [
        { description: "a single object", value: { key: "value" } },
        { description: "null", value: null },
        { description: "undefined", value: undefined },
        { description: "a non-array", value: "string" },
        { description: "an array with mixed types", value: [{}, "string", 123] },
        { description: "an array of arrays", value: [[], [1, 2, 3]] },
    ];

    validCases.forEach(({ description, value }) => {
        it(`should return true for ${description}`, () => {
            expect(isRelationshipArray(value)).to.equal(true);
        });
    });

    invalidCases.forEach(({ description, value }) => {
        it(`should return false for ${description}`, () => {
            expect(isRelationshipArray(value)).to.equal(false);
        });
    });
});
