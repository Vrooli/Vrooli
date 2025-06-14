import { describe, it, expect } from "vitest";
import { exists } from "./exists.js";

describe("exists", () => {
    it("should return true for values that are not null and not undefined", () => {
        expect(exists(0)).toBe(true);
        expect(exists("")).toBe(true);
        expect(exists("Hello")).toBe(true);
        expect(exists([])).toBe(true);
        expect(exists({})).toBe(true);
        expect(exists(false)).toBe(true);
        expect(exists(NaN)).toBe(true);
        expect(exists(Infinity)).toBe(true);
        expect(exists(-Infinity)).toBe(true);
        expect(exists(() => {})).toBe(true);
        expect(exists(Symbol())).toBe(true);
        expect(exists(BigInt(123))).toBe(true);
    });

    it("should return false for null and undefined values", () => {
        expect(exists(null)).toBe(false);
        expect(exists(undefined)).toBe(false);
    });

    it("should work with generic types", () => {
        type SampleType = { name: string, age: number };
        const sample: SampleType | null | undefined = { name: "John", age: 25 };

        if (exists(sample)) {
            // Demonstrating type narrowing. The type of sample is narrowed to SampleType here.
            expect(sample.name).toBe("John");
            expect(sample.age).toBe(25);
        } else {
            fail("Sample should exist");
        }
    });

    it("should work as a filter function for arrays", () => {
        const mixedArray = [1, null, "hello", undefined, false, 0, "", []];
        const filteredArray = mixedArray.filter(exists);

        expect(filteredArray).toEqual([1, "hello", false, 0, "", []]);
        expect(filteredArray).not.toContain(null);
        expect(filteredArray).not.toContain(undefined);
    });

    it("should properly handle nested null/undefined values", () => {
        const nestedObject = { prop: null };
        const arrayWithNulls = [null, undefined, 1, 2];

        expect(exists(nestedObject)).toBe(true); // Object itself exists
        expect(exists(nestedObject.prop)).toBe(false); // But property is null
        expect(exists(arrayWithNulls)).toBe(true); // Array exists
        expect(arrayWithNulls.filter(exists)).toEqual([1, 2]); // But filtered content excludes nulls
    });

    it("should handle function return values correctly", () => {
        function returnsNull() { return null; }
        function returnsUndefined() { return undefined; }
        function returnsValue() { return "value"; }

        expect(exists(returnsNull())).toBe(false);
        expect(exists(returnsUndefined())).toBe(false);
        expect(exists(returnsValue())).toBe(true);
    });
});
