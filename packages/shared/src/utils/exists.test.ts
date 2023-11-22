import { exists } from "./exists";

describe("exists", () => {
    it("should return true for values that are not null and not undefined", () => {
        expect(exists(0)).toBe(true);
        expect(exists("")).toBe(true);
        expect(exists("Hello")).toBe(true);
        expect(exists([])).toBe(true);
        expect(exists({})).toBe(true);
        expect(exists(false)).toBe(true);
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
});
