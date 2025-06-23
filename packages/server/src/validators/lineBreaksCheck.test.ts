import { describe, expect, it } from "vitest";
import { lineBreaksCheck } from "./lineBreaksCheck.js";
import { CustomError } from "../events/error.js";

describe("lineBreaksCheck", () => {
    const mockError = "TooManyLineBreaks" as any;
    
    describe("main object validation", () => {
        it("should pass for valid input with no line breaks", () => {
            const input = {
                title: "Single line title",
                description: "Single line description",
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).not.toThrow();
        });

        it("should pass for valid input with allowed line breaks", () => {
            const input = {
                title: "Line 1\nLine 2",
                description: "Line 1\nLine 2\nLine 3",
            };
            
            // Default k=2, so up to 3 lines (2 line breaks) should be allowed
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).not.toThrow();
        });

        it("should throw error when exceeding default line break limit", () => {
            const input = {
                title: "Line 1\nLine 2\nLine 3\nLine 4", // 3 line breaks, exceeds default k=2
            };
            
            expect(() => lineBreaksCheck(input, ["title"], mockError)).toThrow(CustomError);
        });

        it("should throw error when exceeding custom line break limit", () => {
            const input = {
                description: "Line 1\nLine 2", // 1 line break, should exceed k=0
            };
            
            expect(() => lineBreaksCheck(input, ["description"], mockError, 0)).toThrow(CustomError);
        });

        it("should handle custom k parameter correctly", () => {
            const input = {
                content: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5", // 4 line breaks
            };
            
            // k=4 should allow up to 5 lines (4 line breaks)
            expect(() => lineBreaksCheck(input, ["content"], mockError, 4)).not.toThrow();
            
            // k=3 should not allow 4 line breaks
            expect(() => lineBreaksCheck(input, ["content"], mockError, 3)).toThrow(CustomError);
        });

        it("should handle multiple fields", () => {
            const input = {
                title: "Line 1\nLine 2",
                description: "Line 1\nLine 2\nLine 3",
                notes: "Single line",
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description", "notes"], mockError)).not.toThrow();
        });

        it("should throw error if any field exceeds limit", () => {
            const input = {
                title: "Line 1\nLine 2", // Valid
                description: "Line 1\nLine 2\nLine 3\nLine 4", // Invalid - 3 line breaks exceeds k=2
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).toThrow(CustomError);
        });

        it("should handle missing fields gracefully", () => {
            const input = {
                title: "Existing field",
            };
            
            // Should not throw when checking for non-existent fields
            expect(() => lineBreaksCheck(input, ["title", "nonExistent"], mockError)).not.toThrow();
        });

        it("should handle null/undefined field values", () => {
            const input = {
                title: null,
                description: undefined,
                content: "Valid content",
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description", "content"], mockError)).not.toThrow();
        });

        it("should handle empty strings", () => {
            const input = {
                title: "",
                description: "Normal content",
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).not.toThrow();
        });
    });

    describe("translations validation", () => {
        it("should validate translationsCreate correctly", () => {
            const input = {
                translationsCreate: [
                    { title: "Line 1\nLine 2", description: "Single line" },
                    { title: "Another title", description: "Line 1\nLine 2\nLine 3" },
                ],
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).not.toThrow();
        });

        it("should throw error for invalid translationsCreate", () => {
            const input = {
                translationsCreate: [
                    { title: "Line 1\nLine 2\nLine 3\nLine 4" }, // 3 line breaks, exceeds k=2
                ],
            };
            
            expect(() => lineBreaksCheck(input, ["title"], mockError)).toThrow(CustomError);
        });

        it("should validate translationsUpdate correctly", () => {
            const input = {
                translationsUpdate: [
                    { title: "Line 1\nLine 2", description: "Single line" },
                    { title: "Another title", description: "Line 1\nLine 2\nLine 3" },
                ],
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).not.toThrow();
        });

        it("should throw error for invalid translationsUpdate", () => {
            const input = {
                translationsUpdate: [
                    { title: "Valid title" },
                    { description: "Line 1\nLine 2\nLine 3\nLine 4" }, // 3 line breaks, exceeds k=2
                ],
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).toThrow(CustomError);
        });

        it("should handle both translationsCreate and translationsUpdate", () => {
            const input = {
                translationsCreate: [
                    { title: "Line 1\nLine 2" },
                ],
                translationsUpdate: [
                    { description: "Line 1\nLine 2\nLine 3" },
                ],
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).not.toThrow();
        });

        it("should validate translations with custom k parameter", () => {
            const input = {
                translationsCreate: [
                    { content: "Line 1\nLine 2" }, // 1 line break
                ],
            };
            
            // k=1 should allow 1 line break
            expect(() => lineBreaksCheck(input, ["content"], mockError, 1)).not.toThrow();
            
            // k=0 should not allow any line breaks
            expect(() => lineBreaksCheck(input, ["content"], mockError, 0)).toThrow(CustomError);
        });

        it("should handle missing translation fields", () => {
            const input = {
                translationsCreate: [
                    { title: "Has title" },
                    { description: "Has description" },
                ],
            };
            
            // Should not throw when checking for fields that don't exist in some translations
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).not.toThrow();
        });

        it("should handle empty translations arrays", () => {
            const input = {
                translationsCreate: [],
                translationsUpdate: [],
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).not.toThrow();
        });

        it("should handle null/undefined translation values", () => {
            const input = {
                translationsCreate: [
                    { title: null, description: undefined },
                    { title: "Valid title" },
                ],
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).not.toThrow();
        });
    });

    describe("combined validation", () => {
        it("should validate both main object and translations", () => {
            const input = {
                title: "Main title\nLine 2",
                translationsCreate: [
                    { title: "Translation 1\nLine 2" },
                ],
                translationsUpdate: [
                    { title: "Translation 2\nLine 2\nLine 3" },
                ],
            };
            
            expect(() => lineBreaksCheck(input, ["title"], mockError)).not.toThrow();
        });

        it("should throw error if main object is invalid even if translations are valid", () => {
            const input = {
                title: "Line 1\nLine 2\nLine 3\nLine 4", // Invalid - 3 line breaks
                translationsCreate: [
                    { title: "Valid translation" },
                ],
            };
            
            expect(() => lineBreaksCheck(input, ["title"], mockError)).toThrow(CustomError);
        });

        it("should throw error if translations are invalid even if main object is valid", () => {
            const input = {
                title: "Valid main title",
                translationsCreate: [
                    { title: "Line 1\nLine 2\nLine 3\nLine 4" }, // Invalid - 3 line breaks
                ],
            };
            
            expect(() => lineBreaksCheck(input, ["title"], mockError)).toThrow(CustomError);
        });
    });

    describe("edge cases", () => {
        it("should handle objects without any relevant fields", () => {
            const input = {
                someOtherField: "irrelevant",
            };
            
            expect(() => lineBreaksCheck(input, ["title", "description"], mockError)).not.toThrow();
        });

        it("should handle extremely long content with valid line breaks", () => {
            const longContent = "A".repeat(1000) + "\n" + "B".repeat(1000) + "\n" + "C".repeat(1000);
            const input = {
                content: longContent,
            };
            
            expect(() => lineBreaksCheck(input, ["content"], mockError)).not.toThrow();
        });

        it("should handle different line break styles", () => {
            const input = {
                content1: "Line 1\nLine 2\nLine 3", // Unix style \n
                content2: "Line 1\r\nLine 2\r\nLine 3", // Windows style \r\n (treated as single line breaks)
            };
            
            expect(() => lineBreaksCheck(input, ["content1", "content2"], mockError)).not.toThrow();
        });

        it("should handle zero k parameter", () => {
            const input = {
                title: "Single line only",
            };
            
            expect(() => lineBreaksCheck(input, ["title"], mockError, 0)).not.toThrow();
            
            const inputWithBreak = {
                title: "Line 1\nLine 2",
            };
            
            expect(() => lineBreaksCheck(inputWithBreak, ["title"], mockError, 0)).toThrow(CustomError);
        });

        it("should validate that CustomError is thrown with correct error message", () => {
            const input = {
                title: "Line 1\nLine 2\nLine 3\nLine 4", // Too many line breaks
            };
            
            try {
                lineBreaksCheck(input, ["title"], mockError);
                expect.fail("Should have thrown an error");
            } catch (error) {
                expect(error).toBeInstanceOf(CustomError);
                expect((error as CustomError).code).toBe(mockError);
                expect((error as CustomError).trace).toMatch(/^0117-/); // Main object trace code
            }
        });

        it("should validate that CustomError is thrown with correct error message for translations", () => {
            const input = {
                translationsCreate: [
                    { title: "Line 1\nLine 2\nLine 3\nLine 4" }, // Too many line breaks
                ],
            };
            
            try {
                lineBreaksCheck(input, ["title"], mockError);
                expect.fail("Should have thrown an error");
            } catch (error) {
                expect(error).toBeInstanceOf(CustomError);
                expect((error as CustomError).code).toBe(mockError);
                expect((error as CustomError).trace).toMatch(/^0116-/); // Translation trace code
            }
        });
    });
});
