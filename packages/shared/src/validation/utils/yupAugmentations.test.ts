import { describe, it, expect } from "vitest";
import * as yup from "yup";
import "./yupAugmentations.js"; // Import augmentations

describe("Yup Augmentations Tests", () => {
    describe("removeEmptyString transform", () => {
        it("should keep non-empty strings", () => {
            const schema = yup.string().removeEmptyString();
            
            expect(schema.validateSync("hello")).toBe("hello");
            expect(schema.validateSync("  world  ")).toBe("  world  ");
            expect(schema.validateSync("123")).toBe("123");
        });

        it("should convert empty strings to undefined", () => {
            const schema = yup.string().removeEmptyString();
            
            expect(schema.validateSync("")).toBeUndefined();
        });

        it("should convert whitespace-only strings to undefined", () => {
            const schema = yup.string().removeEmptyString();
            
            expect(schema.validateSync("   ")).toBeUndefined();
            expect(schema.validateSync("\t")).toBeUndefined();
            expect(schema.validateSync("\n")).toBeUndefined();
            expect(schema.validateSync("  \t  \n  ")).toBeUndefined();
        });

        it("should handle non-string inputs safely", () => {
            const schema = yup.string().removeEmptyString();
            
            // null and undefined should become undefined for consistency
            expect(schema.validateSync(null as any)).toBeUndefined();
            expect(schema.validateSync(undefined as any)).toBeUndefined();
            
            // Numbers should be converted to strings when meaningful
            expect(schema.validateSync(123 as any)).toBe("123");
            expect(schema.validateSync(0 as any)).toBe("0");
            expect(schema.validateSync(-45.67 as any)).toBe("-45.67");
            
            // Booleans should convert to their string representation
            expect(schema.validateSync(true as any)).toBe("true");
            expect(schema.validateSync(false as any)).toBe("false");
            
            // Complex types that convert to empty strings should become undefined
            expect(schema.validateSync([] as any)).toBeUndefined(); // String([]) is ""
            expect(schema.validateSync({} as any)).toBe("[object Object]");
            // Objects with custom toString are converted to their string representation
            const customObj = { toString: () => "CustomObject" };
            expect(schema.validateSync(customObj as any)).toBe("CustomObject");
        });

        it("should work with required validation", () => {
            const schema = yup.string().removeEmptyString().required("Field is required");
            
            // Should pass with valid string
            expect(() => schema.validateSync("hello")).not.toThrow();
            
            // Should fail with empty string (becomes undefined)
            expect(() => schema.validateSync("")).toThrow("Field is required");
            expect(() => schema.validateSync("   ")).toThrow("Field is required");
        });
    });

    describe("toBool transform", () => {
        describe("boolean inputs", () => {
            it("should keep boolean values unchanged", () => {
                const schema = yup.bool().toBool();
                
                expect(schema.validateSync(true)).toBe(true);
                expect(schema.validateSync(false)).toBe(false);
            });
        });

        describe("string inputs", () => {
            it("should convert truthy string values to true", () => {
                const schema = yup.bool().toBool();
                
                expect(schema.validateSync("true")).toBe(true);
                expect(schema.validateSync("yes")).toBe(true);
                expect(schema.validateSync("1")).toBe(true);
            });

            it("should handle whitespace in truthy strings", () => {
                const schema = yup.bool().toBool();
                
                expect(schema.validateSync("  true  ")).toBe(true);
                expect(schema.validateSync("\tyes\t")).toBe(true);
                expect(schema.validateSync("\n1\n")).toBe(true);
            });

            it("should convert falsy strings to false", () => {
                const schema = yup.bool().toBool();
                
                // Common falsy representations
                expect(schema.validateSync("false")).toBe(false);
                expect(schema.validateSync("no")).toBe(false);
                expect(schema.validateSync("0")).toBe(false);
                expect(schema.validateSync("")).toBe(false);
                
                // Any other string should be false
                expect(schema.validateSync("random")).toBe(false);
                expect(schema.validateSync("2")).toBe(false);
                expect(schema.validateSync("off")).toBe(false);
            });

            it("should handle case sensitivity consistently", () => {
                const schema = yup.bool().toBool();
                
                // All variations of "yes" should be true for consistency
                expect(schema.validateSync("YES")).toBe(true);
                expect(schema.validateSync("Yes")).toBe(true);
                expect(schema.validateSync("yes")).toBe(true);
                
                // All variations of "true" should be true
                expect(schema.validateSync("TRUE")).toBe(true);
                expect(schema.validateSync("True")).toBe(true);
                expect(schema.validateSync("TrUe")).toBe(true);
                expect(schema.validateSync("true")).toBe(true);
            });
        });

        describe("number inputs", () => {
            it("should convert number 1 to true", () => {
                const schema = yup.bool().toBool();
                
                expect(schema.validateSync(1)).toBe(true);
            });

            it("should convert other numbers to false", () => {
                const schema = yup.bool().toBool();
                
                expect(schema.validateSync(0)).toBe(false);
                expect(schema.validateSync(-1)).toBe(false);
                expect(schema.validateSync(2)).toBe(false);
                expect(schema.validateSync(0.5)).toBe(false);
                expect(schema.validateSync(NaN)).toBe(false);
                expect(schema.validateSync(Infinity)).toBe(false);
            });
        });

        describe("other inputs", () => {
            it("should provide sensible defaults for non-boolean types", () => {
                const schema = yup.bool().toBool();
                
                // Null should default to false for safety
                expect(schema.validateSync(null as any)).toBe(false);
                
                // Undefined is passed through unchanged by yup
                expect(schema.validateSync(undefined as any)).toBeUndefined();
                
                // Complex types should default to false
                expect(schema.validateSync([] as any)).toBe(false);
                expect(schema.validateSync({} as any)).toBe(false);
                expect(schema.validateSync(new Date() as any)).toBe(false);
                expect(schema.validateSync((() => { }) as any)).toBe(false);
                expect(schema.validateSync(Symbol("test") as any)).toBe(false);
            });

            it("should handle edge cases gracefully", () => {
                const schema = yup.bool().toBool();
                
                // Empty values default to false
                expect(schema.validateSync("")).toBe(false);
                expect(schema.validateSync(null)).toBe(false);
                
                // Special number values default to false
                expect(schema.validateSync(NaN)).toBe(false);
                expect(schema.validateSync(Infinity)).toBe(false);
                expect(schema.validateSync(-Infinity)).toBe(false);
            });

            it("should handle objects with custom conversion methods", () => {
                const schema = yup.bool().toBool();
                
                // Objects with valueOf returning 1 should be true
                const objWithValueOf = { valueOf: () => 1 };
                expect(schema.validateSync(objWithValueOf as any)).toBe(true);
                
                // Objects with toString returning truthy strings should be true
                const objWithToString = { toString: () => "true" };
                expect(schema.validateSync(objWithToString as any)).toBe(true);
                
                // But most objects should default to false
                const plainObject = { foo: "bar" };
                expect(schema.validateSync(plainObject as any)).toBe(false);
            });
        });

        it("should work with validation chains", () => {
            const schema = yup.bool().toBool().required("Boolean is required");
            
            // Should work with various inputs
            expect(() => schema.validateSync("true")).not.toThrow();
            expect(() => schema.validateSync(1)).not.toThrow();
            expect(() => schema.validateSync(false)).not.toThrow();
            expect(() => schema.validateSync("false")).not.toThrow();
            
            // Should fail with undefined/null if required
            expect(() => schema.validateSync(undefined)).toThrow("Boolean is required");
        });
    });

    describe("combined usage", () => {
        it("should work with both augmentations in a schema", () => {
            const schema = yup.object({
                name: yup.string().removeEmptyString().required("Name is required"),
                isActive: yup.bool().toBool().required("Active status is required"),
            });

            // Valid data
            const validData = {
                name: "John Doe",
                isActive: "true",
            };
            expect(() => schema.validateSync(validData)).not.toThrow();
            
            const result = schema.validateSync(validData);
            expect(result.name).toBe("John Doe");
            expect(result.isActive).toBe(true);

            // Invalid data - empty name
            const invalidData = {
                name: "   ",
                isActive: "1",
            };
            expect(() => schema.validateSync(invalidData)).toThrow("Name is required");
        });
    });

    describe("real-world usage patterns", () => {
        it("should handle form data with optional fields", () => {
            const schema = yup.object({
                username: yup.string().removeEmptyString().required(),
                bio: yup.string().removeEmptyString(), // optional
                newsletter: yup.bool().toBool().default(false),
            });

            // User submits form with empty optional field
            const formData = {
                username: "john_doe",
                bio: "   ", // empty whitespace
                newsletter: "yes",
            };

            const result = schema.validateSync(formData);
            expect(result.username).toBe("john_doe");
            expect(result.bio).toBeUndefined(); // empty string removed
            expect(result.newsletter).toBe(true);
        });

        it("should handle API responses with string booleans", () => {
            const apiResponseSchema = yup.object({
                id: yup.number().required(),
                isVerified: yup.bool().toBool(),
                isPremium: yup.bool().toBool(),
                hasNotifications: yup.bool().toBool(),
            });

            // API returns string representations of booleans
            const apiResponse = {
                id: 123,
                isVerified: "1",
                isPremium: "false",
                hasNotifications: "yes",
            };

            const result = apiResponseSchema.validateSync(apiResponse);
            expect(result.isVerified).toBe(true);
            expect(result.isPremium).toBe(false);
            expect(result.hasNotifications).toBe(true);
        });

        it("should clean up user input strings", () => {
            const searchSchema = yup.object({
                query: yup.string().removeEmptyString().required("Search is required").min(2, "Search must be at least 2 characters"),
                includeArchived: yup.bool().toBool().default(false),
            });

            // User types only spaces - should fail validation due to required field
            // removeEmptyString transforms "  " to undefined, then required() fails
            expect(() => searchSchema.validateSync({
                query: "  ",
                includeArchived: "true",
            })).toThrow();

            // Valid search
            const result = searchSchema.validateSync({
                query: "typescript",
                includeArchived: "1",
            });
            expect(result.query).toBe("typescript");
            expect(result.includeArchived).toBe(true);
        });

        it("should handle database boolean representations", () => {
            const dbRecordSchema = yup.object({
                enabled: yup.bool().toBool(),
                deleted: yup.bool().toBool(),
                published: yup.bool().toBool(),
            });

            // Database might return 1/0 for booleans
            const dbRecord = {
                enabled: 1,
                deleted: 0,
                published: 1,
            };

            const result = dbRecordSchema.validateSync(dbRecord);
            expect(result.enabled).toBe(true);
            expect(result.deleted).toBe(false);
            expect(result.published).toBe(true);
        });
    });
});
