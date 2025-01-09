/* eslint-disable @typescript-eslint/ban-ts-comment */
import { CountFields, selPad } from "./infoConverter";

describe("CountFields", () => {
    describe("addToData", () => {
        it("processes valid obj and countFields correctly", () => {
            const obj = { commentsCount: 5 };
            const countFields = { commentsCount: true } as const;
            const expected = { _count: { comments: true } };
            expect(CountFields.addToData(obj, countFields)).toEqual(expected);
        });

        it("returns obj unmodified when countFields is undefined", () => {
            const obj = { commentsCount: 5 };
            expect(CountFields.addToData(obj, undefined)).toEqual(obj);
        });

        it("handles empty obj correctly", () => {
            const countFields = { commentsCount: true } as const;
            expect(CountFields.addToData({}, countFields)).toEqual({});
        });

        it("does not modify obj when countFields is empty", () => {
            const obj = { commentsCount: 5 };
            expect(CountFields.addToData(obj, {})).toEqual(obj);
        });

        it("does not modify obj if specified count fields do not exist", () => {
            const obj = { likesCount: 3 };
            const countFields = { commentsCount: true } as const;
            expect(CountFields.addToData(obj, countFields)).toEqual(obj);
        });

        it("adds to existing _count property without overwriting", () => {
            const obj = { _count: { likes: true }, commentsCount: 5 };
            const countFields = { commentsCount: true } as const;
            const expected = { _count: { likes: true, comments: true } };
            expect(CountFields.addToData(obj, countFields)).toEqual(expected);
        });

        it("handles multiple count fields correctly", () => {
            const obj = { commentsCount: 5, reportsCount: 2 };
            const countFields = { commentsCount: true, reportsCount: true } as const;
            const expected = { _count: { comments: true, reports: true } };
            expect(CountFields.addToData(obj, countFields)).toEqual(expected);
        });

        it("ignores count fields not ending in \"Count\"", () => {
            const obj = { commentData: 5 };
            const countFields = { commentData: true } as const;
            expect(CountFields.addToData(obj, countFields)).toEqual(obj);
        });
    });
});

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
                            data: "bloop", // Should convert to true
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
