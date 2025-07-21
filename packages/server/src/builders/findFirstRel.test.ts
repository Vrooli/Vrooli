import { expect, describe, it } from "vitest";
import { findFirstRel } from "./findFirstRel.js";

describe("findFirstRel", () => {
    describe("finding existing fields", () => {
        it("returns first non-null field when it's the first in list", () => {
            const obj = {
                field1: "value1",
                field2: "value2",
                field3: "value3",
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field1", "value1"]);
        });

        it("returns first non-null field when it's in the middle", () => {
            const obj = {
                field1: null,
                field2: "value2",
                field3: "value3",
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field2", "value2"]);
        });

        it("returns first non-null field when it's the last", () => {
            const obj = {
                field1: null,
                field2: undefined,
                field3: "value3",
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field3", "value3"]);
        });

        it("skips undefined values", () => {
            const obj = {
                field1: undefined,
                field2: "value2",
                field3: "value3",
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field2", "value2"]);
        });

        it("returns first field even if value is falsy but not null/undefined", () => {
            const obj = {
                field1: 0,
                field2: "value2",
                field3: "value3",
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field1", 0]);
        });

        it("handles empty string as valid value", () => {
            const obj = {
                field1: "",
                field2: "value2",
            };
            const fieldsToCheck = ["field1", "field2"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field1", ""]);
        });

        it("handles boolean false as valid value", () => {
            const obj = {
                field1: false,
                field2: true,
            };
            const fieldsToCheck = ["field1", "field2"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field1", false]);
        });
    });

    describe("no matching fields", () => {
        it("returns undefined when no fields exist in object", () => {
            const obj = {
                otherField: "value",
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual([undefined, undefined]);
        });

        it("returns undefined when all fields are null", () => {
            const obj = {
                field1: null,
                field2: null,
                field3: null,
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual([undefined, undefined]);
        });

        it("returns undefined when all fields are undefined", () => {
            const obj = {
                field1: undefined,
                field2: undefined,
                field3: undefined,
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual([undefined, undefined]);
        });

        it("returns undefined when mix of null, undefined, and missing fields", () => {
            const obj = {
                field1: null,
                field2: undefined,
                // field3 doesn't exist
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual([undefined, undefined]);
        });
    });

    describe("edge cases", () => {
        it("handles empty object", () => {
            const obj = {};
            const fieldsToCheck = ["field1", "field2"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual([undefined, undefined]);
        });

        it("handles empty fields array", () => {
            const obj = {
                field1: "value1",
                field2: "value2",
            };
            const fieldsToCheck: string[] = [];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual([undefined, undefined]);
        });

        it("handles fields not in check order", () => {
            const obj = {
                field3: "value3",
                field1: "value1",
                field2: "value2",
            };
            const fieldsToCheck = ["field2", "field1", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field2", "value2"]);
        });

        it("handles complex values", () => {
            const complexValue = { nested: { data: [1, 2, 3] } };
            const obj = {
                field1: null,
                field2: complexValue,
                field3: "simple",
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field2", complexValue]);
        });

        it("handles array values", () => {
            const arrayValue = [1, 2, 3, 4, 5];
            const obj = {
                field1: arrayValue,
                field2: "value2",
            };
            const fieldsToCheck = ["field1", "field2"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field1", arrayValue]);
        });

        it("handles function values", () => {
            const funcValue = () => "test";
            const obj = {
                field1: null,
                field2: funcValue,
                field3: "value3",
            };
            const fieldsToCheck = ["field1", "field2", "field3"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field2", funcValue]);
        });

        it("handles Symbol values", () => {
            const symbolValue = Symbol("test");
            const obj = {
                field1: symbolValue,
                field2: "value2",
            };
            const fieldsToCheck = ["field1", "field2"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["field1", symbolValue]);
        });

        it("preserves exact value reference", () => {
            const objectValue = { test: "data" };
            const obj = {
                field1: objectValue,
            };
            const fieldsToCheck = ["field1"];

            const [, value] = findFirstRel(obj, fieldsToCheck);

            expect(value).toBe(objectValue); // Same reference
        });

        it("checks fields in specified order regardless of object key order", () => {
            const obj = {
                z: "last",
                a: "first",
                m: "middle",
            };
            const fieldsToCheck = ["m", "z", "a"];

            const result = findFirstRel(obj, fieldsToCheck);

            expect(result).toEqual(["m", "middle"]);
        });
    });
});
