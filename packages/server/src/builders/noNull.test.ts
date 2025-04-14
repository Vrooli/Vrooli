/* eslint-disable func-style */
import { uuid } from "@local/shared";
import { expect } from "chai";
import { noEmptyString, noNull, validNumber, validUuid } from "./noNull.js";

describe("noNull", () => {
    it("returns the first valid argument when all are valid", () => {
        expect(noNull(1, 2, 3)).to.equal(1);
    });

    it("skips null and undefined to return the first valid value", () => {
        expect(noNull(null, undefined, "valid", "also valid")).to.equal("valid");
    });

    it("returns undefined when all arguments are null", () => {
        expect(noNull(null, null)).to.be.undefined;
    });

    it("returns undefined when all arguments are undefined", () => {
        expect(noNull(undefined, undefined)).to.be.undefined;
    });

    it("returns undefined for a mix of null and undefined values", () => {
        expect(noNull(null, undefined, null)).to.be.undefined;
    });

    it("returns undefined when called with no arguments", () => {
        expect(noNull()).to.be.undefined;
    });

    it("works with complex objects", () => {
        const obj = { key: "value" };
        expect(noNull(null, obj)).to.equal(obj);
    });

    it("correctly returns a function argument", () => {
        const func = () => "test";
        expect(noNull(undefined, func)).to.equal(func);
    });
});

describe("noEmptyString", () => {
    it("returns the first non-empty string", () => {
        expect(noEmptyString("first", "second")).to.equal("first");
        expect(noEmptyString("", "second", "third")).to.equal("second");
    });

    it("ignores empty strings", () => {
        expect(noEmptyString("", "", "non-empty", "another")).to.equal("non-empty");
    });

    it("returns undefined if all strings are empty", () => {
        expect(noEmptyString("", "", "")).to.be.undefined;
    });

    it("ignores null and undefined values", () => {
        expect(noEmptyString(null, undefined, "non-empty")).to.equal("non-empty");
        expect(noEmptyString(null, "", undefined)).to.be.undefined;
    });

    it("handles mixed types, but only returns strings", () => {
        expect(noEmptyString(123 as any, "non-empty", {} as any)).to.equal("non-empty");
    });

    it("returns undefined with no arguments", () => {
        expect(noEmptyString()).to.be.undefined;
    });

    it("returns undefined with only null and undefined arguments", () => {
        expect(noEmptyString(null, undefined)).to.be.undefined;
    });

    it("handles a single non-empty string argument", () => {
        expect(noEmptyString("only")).to.equal("only");
    });

    it("handles a single empty string argument", () => {
        expect(noEmptyString("")).to.be.undefined;
    });

    it("handles a single null or undefined argument", () => {
        expect(noEmptyString(null)).to.be.undefined;
        expect(noEmptyString(undefined)).to.be.undefined;
    });
});

describe("validNumber", () => {
    it("should return the first valid number", () => {
        expect(validNumber(1, 2, 3)).to.equal(1);
        expect(validNumber(-1, 0, 1)).to.equal(-1);
    });

    it("should ignore NaN and return the first valid number", () => {
        expect(validNumber(NaN, 1, 2)).to.equal(1);
    });

    it("should ignore Infinity and return the first valid number", () => {
        expect(validNumber(Infinity, 1, 2)).to.equal(1);
    });

    it("should ignore -Infinity and return the first valid number", () => {
        expect(validNumber(-Infinity, 1, 2)).to.equal(1);
    });

    it("should return undefined if all values are NaN", () => {
        expect(validNumber(NaN, NaN)).to.be.undefined;
    });

    it("should return undefined if all values are Infinity or -Infinity", () => {
        expect(validNumber(Infinity, -Infinity)).to.be.undefined;
    });

    it("should ignore null and undefined values and return the first valid number", () => {
        expect(validNumber(null, undefined, 1, 2)).to.equal(1);
    });

    it("should return undefined if no valid numbers are provided", () => {
        expect(validNumber(null, undefined, NaN, Infinity)).to.be.undefined;
    });

    it("should ignore non-numeric types and return the first valid number", () => {
        expect(validNumber("string", true, {}, [], 1, 2)).to.equal(1);
    });

    it("should handle a mix of invalid types and values, returning the first valid number", () => {
        expect(validNumber("text", null, NaN, undefined, -Infinity, 42)).to.equal(42);
    });

    it("should return undefined if only non-numeric types and invalid numbers are provided", () => {
        expect(validNumber("string", true, {}, [], NaN, Infinity)).to.be.undefined;
    });
});

describe("validUuid", () => {
    it("should return the first valid UUID", () => {
        const uuid1 = uuid();
        const uuid2 = uuid();
        expect(validUuid(uuid1, uuid2)).to.equal(uuid1);
    });

    it("should ignore invalid UUIDs and return the first valid UUID", () => {
        const goodUuid = uuid();
        const badUuid = "invalid-uuid";
        expect(validUuid(badUuid, goodUuid)).to.equal(goodUuid);
    });

    it("should ignore null and undefined values and return the first valid UUID", () => {
        const goodUuid = uuid();
        expect(validUuid(null, undefined, goodUuid)).to.equal(goodUuid);
    });

    it("should return undefined if no valid UUIDs are provided", () => {
        const badUuid = "invalid-uuid";
        expect(validUuid(null, undefined, badUuid)).to.be.undefined;
    });

    it("should ignore non-string types and return the first valid UUID", () => {
        const goodUuid = uuid();
        expect(validUuid(42, true, {}, [], goodUuid)).to.equal(goodUuid);
    });

    it("should handle a mix of invalid types and values, returning the first valid UUID", () => {
        const goodUuid = uuid();
        const badUuid = "invalid-uuid";
        expect(validUuid("text", null, undefined, badUuid, goodUuid)).to.equal(goodUuid);
    });

    it("should return undefined if only non-string types and invalid UUIDs are provided", () => {
        const badUuid = uuid() + "1";
        expect(validUuid("string", true, {}, [], badUuid)).to.be.undefined;
    });

    it("should return undefined if no arguments are provided", () => {
        expect(validUuid()).to.be.undefined;
    });

    it("should return undefined if only null and undefined arguments are provided", () => {
        expect(validUuid(null, undefined)).to.be.undefined;
    });
});
