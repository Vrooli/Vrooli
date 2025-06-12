import { describe, it, expect } from "vitest";
import { exists } from "./exists.js";

describe("exists", () => {
    it("should return true for values that are not null and not undefined", () => {
        expect(exists(0)).to.equal(true);
        expect(exists("")).to.equal(true);
        expect(exists("Hello")).to.equal(true);
        expect(exists([])).to.equal(true);
        expect(exists({})).to.equal(true);
        expect(exists(false)).to.equal(true);
        expect(exists(NaN)).to.equal(true);
        expect(exists(Infinity)).to.equal(true);
        expect(exists(-Infinity)).to.equal(true);
        expect(exists(() => {})).to.equal(true);
        expect(exists(Symbol())).to.equal(true);
        expect(exists(BigInt(123))).to.equal(true);
    });

    it("should return false for null and undefined values", () => {
        expect(exists(null)).to.equal(false);
        expect(exists(undefined)).to.equal(false);
    });

    it("should work with generic types", () => {
        type SampleType = { name: string, age: number };
        const sample: SampleType | null | undefined = { name: "John", age: 25 };

        if (exists(sample)) {
            // Demonstrating type narrowing. The type of sample is narrowed to SampleType here.
            expect(sample.name).to.equal("John");
            expect(sample.age).to.equal(25);
        } else {
            fail("Sample should exist");
        }
    });

    it("should work as a filter function for arrays", () => {
        const mixedArray = [1, null, "hello", undefined, false, 0, "", []];
        const filteredArray = mixedArray.filter(exists);

        expect(filteredArray).to.deep.equal([1, "hello", false, 0, "", []]);
        expect(filteredArray).not.to.include(null);
        expect(filteredArray).not.to.include(undefined);
    });

    it("should properly handle nested null/undefined values", () => {
        const nestedObject = { prop: null };
        const arrayWithNulls = [null, undefined, 1, 2];

        expect(exists(nestedObject)).to.equal(true); // Object itself exists
        expect(exists(nestedObject.prop)).to.equal(false); // But property is null
        expect(exists(arrayWithNulls)).to.equal(true); // Array exists
        expect(arrayWithNulls.filter(exists)).to.deep.equal([1, 2]); // But filtered content excludes nulls
    });

    it("should handle function return values correctly", () => {
        function returnsNull() { return null; }
        function returnsUndefined() { return undefined; }
        function returnsValue() { return "value"; }

        expect(exists(returnsNull())).to.equal(false);
        expect(exists(returnsUndefined())).to.equal(false);
        expect(exists(returnsValue())).to.equal(true);
    });
});
