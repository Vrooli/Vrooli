/* eslint-disable func-style */
import { expect } from "chai";
import { noEmptyString, noNull, validNumber } from "./noNull.js";

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

