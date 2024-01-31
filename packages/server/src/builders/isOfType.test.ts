/* eslint-disable @typescript-eslint/ban-ts-comment */
import { isRelationshipArray, isRelationshipObject } from "./isOfType";

describe("isRelationshipObject", () => {
    const validCases = [
        { description: "an object", value: { key: "value" } },
        { description: "an empty object", value: {} },
        // @ts-ignore: Testing runtime scenario
        { description: "a function", value() { } },
        // @ts-ignore: Testing runtime scenario
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
        test(`should return true for ${description}`, () => {
            expect(isRelationshipObject(value)).toBe(true);
        });
    });

    invalidCases.forEach(({ description, value }) => {
        test(`should return false for ${description}`, () => {
            expect(isRelationshipObject(value)).toBe(false);
        });
    });
});

describe("isRelationshipArray", () => {
    const validCases = [
        { description: "an array of objects", value: [{ key: "value" }, {}] },
        { description: "an array with a single object", value: [{}] },
        // @ts-ignore: Testing runtime scenario
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
        test(`should return true for ${description}`, () => {
            expect(isRelationshipArray(value)).toBe(true);
        });
    });

    invalidCases.forEach(({ description, value }) => {
        test(`should return false for ${description}`, () => {
            expect(isRelationshipArray(value)).toBe(false);
        });
    });
});
