/* eslint-disable @typescript-eslint/ban-ts-comment */
import { selPad } from "./selectHelper";

describe("selPad", () => {
    test("returns true for primitive fields (required for recursion", () => {
        const invalidInputs = [123, "string", new Date()];
        invalidInputs.forEach(input => {
            // @ts-ignore: Testing runtime scenario
            expect(selPad(input)).toBe(true);
        });
    });

    test("returns empty object for invalid/empty fields (required for recursion", () => {
        const invalidInputs = [null, undefined, [], ["hello"]];
        invalidInputs.forEach(input => {
            // @ts-ignore: Testing runtime scenario
            expect(selPad(input)).toEqual({});
        });
    });

    test("advanced test", () => {
        const inputWithSelect = {
            select: { // Shouldn't add additional "select"
                key: {
                    boop: {}, // Should omit fields with empty objects
                    beep: {
                        select: { // Shouldn't add additional "select"
                            data: "yeet", // Should convert to true
                        },
                    },
                    bop: {
                        yo: new Date(), // Should convert to true
                    },
                },
            },
        };
        const expectedOutput = {
            select: {
                key: {
                    select: {
                        beep: {
                            select: {
                                data: true,
                            },
                        },
                        bop: {
                            select: {
                                yo: true,
                            },
                        },
                    },
                },
            },
        };
        expect(selPad(inputWithSelect)).toEqual(expectedOutput);
    });

    test("transforms a valid object without select into Prisma select format", () => {
        const validInput = { key: "value" };
        const expectedOutput = { select: { key: true } };
        expect(selPad(validInput)).toEqual(expectedOutput);
    });

    test("recursively transforms nested objects into Prisma select format", () => {
        const nestedInput = { level1: { level2: { key: "value" } } };
        const expectedOutput = { select: { level1: { select: { level2: { select: { key: true } } } } } };
        expect(selPad(nestedInput)).toEqual(expectedOutput);
    });

    test("is idempotent for already transformed objects", () => {
        const alreadyTransformed = { select: { key: true } };
        expect(selPad(alreadyTransformed)).toEqual(alreadyTransformed);
        expect(selPad(selPad(alreadyTransformed))).toEqual(alreadyTransformed); // Apply twice to test idempotency
    });

    test("removes arrays (prisma doesn't support arrays in select queries)", () => {
        const arrayInput = {
            key: [{ boop: "value" }, { beep: "hello" }],
            otherKey: [],
        };
        const expectedOutput = {}; // Array fields are omitted, which makes the object empty, which means it shouldn't be padded either
        expect(selPad(arrayInput)).toEqual(expectedOutput);
    });
});
