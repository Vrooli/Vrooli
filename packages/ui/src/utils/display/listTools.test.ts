import { placeholderColor, placeholderColors, simpleHash } from "./listTools";

describe("simpleHash", () => {
    it("should return the same hash for the same input", () => {
        const input = "testString";
        expect(simpleHash(input)).toBe(simpleHash(input));
    });

    it("should return different hashes for different inputs", () => {
        const input1 = "testString1";
        const input2 = "testString2";
        expect(simpleHash(input1)).not.toBe(simpleHash(input2));
    });

    it("should handle an empty string", () => {
        expect(simpleHash("")).toBe(0);
    });
});

describe("placeholderColor", () => {
    it("should return a valid color pair from the array", () => {
        const result = placeholderColor();
        expect(placeholderColors).toContainEqual(result);
    });

    it("should return consistent results for the same seed", () => {
        const seed = "consistentSeed";
        const result1 = placeholderColor(seed);
        const result2 = placeholderColor(seed);
        expect(result1).toEqual(result2);
    });

    it("should return different results for different seeds", () => {
        const seed1 = "seed1";
        const seed2 = "seed2";
        const result1 = placeholderColor(seed1);
        const result2 = placeholderColor(seed2);
        expect(result1).not.toEqual(result2);
    });

    it("should handle numeric seeds", () => {
        const seed = 12345;
        const result1 = placeholderColor(seed);
        const result2 = placeholderColor(seed);
        expect(result1).toEqual(result2);
    });

    it("should return different results without a seed", () => {
        const result1 = placeholderColor();
        const result2 = placeholderColor();
        expect(result1).not.toEqual(result2); // Note: This test might occasionally fail due to random chance
    });
});
