import { uuid } from "@local/shared";
import { noEmptyString, noNull, validNumber, validUuid } from "./noNull";

describe("noNull", () => {
    it("returns the first valid argument when all are valid", () => {
        expect(noNull(1, 2, 3)).toBe(1);
    });

    it("skips null and undefined to return the first valid value", () => {
        expect(noNull(null, undefined, "valid", "also valid")).toBe("valid");
    });

    it("returns undefined when all arguments are null", () => {
        expect(noNull(null, null)).toBeUndefined();
    });

    it("returns undefined when all arguments are undefined", () => {
        expect(noNull(undefined, undefined)).toBeUndefined();
    });

    it("returns undefined for a mix of null and undefined values", () => {
        expect(noNull(null, undefined, null)).toBeUndefined();
    });

    it("returns undefined when called with no arguments", () => {
        expect(noNull()).toBeUndefined();
    });

    it("works with complex objects", () => {
        const obj = { key: "value" };
        expect(noNull(null, obj)).toBe(obj);
    });

    it("correctly returns a function argument", () => {
        const func = () => "test";
        expect(noNull(undefined, func)).toBe(func);
    });
});

describe("noEmptyString", () => {
    test("returns the first non-empty string", () => {
        expect(noEmptyString("first", "second")).toBe("first");
        expect(noEmptyString("", "second", "third")).toBe("second");
    });

    test("ignores empty strings", () => {
        expect(noEmptyString("", "", "non-empty", "another")).toBe("non-empty");
    });

    test("returns undefined if all strings are empty", () => {
        expect(noEmptyString("", "", "")).toBeUndefined();
    });

    test("ignores null and undefined values", () => {
        expect(noEmptyString(null, undefined, "non-empty")).toBe("non-empty");
        expect(noEmptyString(null, "", undefined)).toBeUndefined();
    });

    test("handles mixed types, but only returns strings", () => {
        expect(noEmptyString(123 as any, "non-empty", {} as any)).toBe("non-empty");
    });

    test("returns undefined with no arguments", () => {
        expect(noEmptyString()).toBeUndefined();
    });

    test("returns undefined with only null and undefined arguments", () => {
        expect(noEmptyString(null, undefined)).toBeUndefined();
    });

    test("handles a single non-empty string argument", () => {
        expect(noEmptyString("only")).toBe("only");
    });

    test("handles a single empty string argument", () => {
        expect(noEmptyString("")).toBeUndefined();
    });

    test("handles a single null or undefined argument", () => {
        expect(noEmptyString(null)).toBeUndefined();
        expect(noEmptyString(undefined)).toBeUndefined();
    });
});

describe("validNumber", () => {
    it("should return the first valid number", () => {
        expect(validNumber(1, 2, 3)).toBe(1);
        expect(validNumber(-1, 0, 1)).toBe(-1);
    });

    it("should ignore NaN and return the first valid number", () => {
        expect(validNumber(NaN, 1, 2)).toBe(1);
    });

    it("should ignore Infinity and return the first valid number", () => {
        expect(validNumber(Infinity, 1, 2)).toBe(1);
    });

    it("should ignore -Infinity and return the first valid number", () => {
        expect(validNumber(-Infinity, 1, 2)).toBe(1);
    });

    it("should return undefined if all values are NaN", () => {
        expect(validNumber(NaN, NaN)).toBeUndefined();
    });

    it("should return undefined if all values are Infinity or -Infinity", () => {
        expect(validNumber(Infinity, -Infinity)).toBeUndefined();
    });

    it("should ignore null and undefined values and return the first valid number", () => {
        expect(validNumber(null, undefined, 1, 2)).toBe(1);
    });

    it("should return undefined if no valid numbers are provided", () => {
        expect(validNumber(null, undefined, NaN, Infinity)).toBeUndefined();
    });

    it("should ignore non-numeric types and return the first valid number", () => {
        expect(validNumber("string", true, {}, [], 1, 2)).toBe(1);
    });

    it("should handle a mix of invalid types and values, returning the first valid number", () => {
        expect(validNumber("text", null, NaN, undefined, -Infinity, 42)).toBe(42);
    });

    it("should return undefined if only non-numeric types and invalid numbers are provided", () => {
        expect(validNumber("string", true, {}, [], NaN, Infinity)).toBeUndefined();
    });
});

describe("validUuid", () => {
    it("should return the first valid UUID", () => {
        const uuid1 = uuid();
        const uuid2 = uuid();
        expect(validUuid(uuid1, uuid2)).toBe(uuid1);
    });

    it("should ignore invalid UUIDs and return the first valid UUID", () => {
        const goodUuid = uuid();
        const badUuid = "invalid-uuid";
        expect(validUuid(badUuid, goodUuid)).toBe(goodUuid);
    });

    it("should ignore null and undefined values and return the first valid UUID", () => {
        const goodUuid = uuid();
        expect(validUuid(null, undefined, goodUuid)).toBe(goodUuid);
    });

    it("should return undefined if no valid UUIDs are provided", () => {
        const badUuid = "invalid-uuid";
        expect(validUuid(null, undefined, badUuid)).toBeUndefined();
    });

    it("should ignore non-string types and return the first valid UUID", () => {
        const goodUuid = uuid();
        expect(validUuid(42, true, {}, [], goodUuid)).toBe(goodUuid);
    });

    it("should handle a mix of invalid types and values, returning the first valid UUID", () => {
        const goodUuid = uuid();
        const badUuid = "invalid-uuid";
        expect(validUuid("text", null, undefined, badUuid, goodUuid)).toBe(goodUuid);
    });

    it("should return undefined if only non-string types and invalid UUIDs are provided", () => {
        const badUuid = uuid() + "1";
        expect(validUuid("string", true, {}, [], badUuid)).toBeUndefined();
    });

    it("should return undefined if no arguments are provided", () => {
        expect(validUuid()).toBeUndefined();
    });

    it("should return undefined if only null and undefined arguments are provided", () => {
        expect(validUuid(null, undefined)).toBeUndefined();
    });
});
