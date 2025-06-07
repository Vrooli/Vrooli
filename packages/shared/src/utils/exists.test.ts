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
});
