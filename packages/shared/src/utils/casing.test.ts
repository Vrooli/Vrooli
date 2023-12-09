import { lowercaseFirstLetter, uppercaseFirstLetter } from "./casing";

describe("lowercaseFirstLetter", () => {
    it("should lowercase the first letter of a string", () => {
        expect(lowercaseFirstLetter("Hello")).toBe("hello");
        expect(lowercaseFirstLetter("WORLD")).toBe("wORLD");
    });

    it("should return the same string if the first letter is already lowercase", () => {
        expect(lowercaseFirstLetter("already")).toBe("already");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(lowercaseFirstLetter("")).toBe("");
    });

    it("should handle non-alphabetical characters", () => {
        expect(lowercaseFirstLetter("1hello")).toBe("1hello");
        expect(lowercaseFirstLetter("@World")).toBe("@World");
    });
});

describe("uppercaseFirstLetter", () => {
    it("should uppercase the first letter of a string", () => {
        expect(uppercaseFirstLetter("hello")).toBe("Hello");
        expect(uppercaseFirstLetter("wORLD")).toBe("WORLD");
    });

    it("should return the same string if the first letter is already uppercase", () => {
        expect(uppercaseFirstLetter("Already")).toBe("Already");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(uppercaseFirstLetter("")).toBe("");
    });

    it("should handle non-alphabetical characters", () => {
        expect(uppercaseFirstLetter("1hello")).toBe("1hello");
        expect(uppercaseFirstLetter("@world")).toBe("@world");
    });
});
