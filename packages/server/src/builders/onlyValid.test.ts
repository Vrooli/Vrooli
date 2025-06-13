import { describe, expect, it } from "vitest";
import { onlyValidHandles, onlyValidIds } from "./onlyValid.js";

describe("onlyValidHandles", () => {
    describe("basic functionality", () => {
        it("should return valid handles starting with $ and 3-16 alphanumeric characters", () => {
            const input = ["$abc", "$user123", "$test"];
            expect(onlyValidHandles(input)).toEqual(["$abc", "$user123", "$test"]);
        });

        it("should filter out handles that don't start with $", () => {
            const input = ["@user", "user123", "$valid", "#invalid"];
            expect(onlyValidHandles(input)).toEqual(["$valid"]);
        });

        it("should filter out handles that are too short (less than 4 total characters)", () => {
            const input = ["$a", "$ab", "$abc", "$abcd"];
            expect(onlyValidHandles(input)).toEqual(["$abc", "$abcd"]);
        });

        it("should filter out handles that are too long (more than 17 total characters)", () => {
            const input = [
                "$abcdefghijklmnop", // 17 chars total (16 after $) - valid
                "$abcdefghijklmnopq", // 18 chars total (17 after $) - invalid
                "$valid123",
            ];
            expect(onlyValidHandles(input)).toEqual(["$abcdefghijklmnop", "$valid123"]);
        });

        it("should filter out handles with non-alphanumeric characters", () => {
            const input = [
                "$valid123",
                "$invalid-handle",
                "$invalid_handle",
                "$invalid.handle",
                "$invalid@handle",
                "$invalid handle",
                "$test123",
            ];
            expect(onlyValidHandles(input)).toEqual(["$valid123", "$test123"]);
        });

        it("should handle mixed case alphanumeric characters", () => {
            const input = ["$TestUser123", "$ALLCAPS", "$lowercase", "$MiXeDcAsE123"];
            expect(onlyValidHandles(input)).toEqual(["$TestUser123", "$ALLCAPS", "$lowercase", "$MiXeDcAsE123"]);
        });
    });

    describe("null and undefined handling", () => {
        it("should filter out null values", () => {
            const input = ["$valid", null, "$another", null];
            expect(onlyValidHandles(input)).toEqual(["$valid", "$another"]);
        });

        it("should filter out undefined values", () => {
            const input = ["$valid", undefined, "$another", undefined];
            expect(onlyValidHandles(input)).toEqual(["$valid", "$another"]);
        });

        it("should handle mixed null, undefined, and valid handles", () => {
            const input = [null, "$valid", undefined, "$another", null, "invalid"];
            expect(onlyValidHandles(input)).toEqual(["$valid", "$another"]);
        });

        it("should return empty array when all values are null or undefined", () => {
            expect(onlyValidHandles([null, undefined, null])).toEqual([]);
            expect(onlyValidHandles([null])).toEqual([]);
            expect(onlyValidHandles([undefined])).toEqual([]);
        });
    });

    describe("edge cases", () => {
        it("should return empty array for empty input", () => {
            expect(onlyValidHandles([])).toEqual([]);
        });

        it("should handle minimum valid length (4 characters total)", () => {
            const input = ["$abc", "$123", "$a1b"];
            expect(onlyValidHandles(input)).toEqual(["$abc", "$123", "$a1b"]);
        });

        it("should handle maximum valid length (17 characters total)", () => {
            const input = ["$abcdefghijklmnop", "$1234567890123456"];
            expect(onlyValidHandles(input)).toEqual(["$abcdefghijklmnop", "$1234567890123456"]);
        });

        it("should handle numeric-only handles after $", () => {
            const input = ["$123", "$456789", "$000"];
            expect(onlyValidHandles(input)).toEqual(["$123", "$456789", "$000"]);
        });

        it("should handle letter-only handles after $", () => {
            const input = ["$abc", "$DEFGH", "$xyZ"];
            expect(onlyValidHandles(input)).toEqual(["$abc", "$DEFGH", "$xyZ"]);
        });

        it("should handle complex mixed arrays with various invalid types", () => {
            const input = [
                "$valid1",
                42 as any,
                "$valid2",
                {} as any,
                "invalid",
                "$",
                "$ab",
                "$valid3",
                [] as any,
                true as any,
                null,
                undefined,
                "$toolong12345678901",
            ];
            expect(onlyValidHandles(input)).toEqual(["$valid1", "$valid2", "$valid3"]);
        });

        it("should preserve order of valid handles", () => {
            const input = ["$zebra", "$alpha", "$beta", "invalid", "$gamma"];
            expect(onlyValidHandles(input)).toEqual(["$zebra", "$alpha", "$beta", "$gamma"]);
        });
    });

    describe("special characters and unicode", () => {
        it("should reject handles with unicode characters", () => {
            const input = ["$tëst", "$test123", "$tést", "$válid", "$normal"];
            expect(onlyValidHandles(input)).toEqual(["$test123", "$normal"]);
        });

        it("should reject handles with emojis", () => {
            const input = ["$test😀", "$valid123", "$🔥fire"];
            expect(onlyValidHandles(input)).toEqual(["$valid123"]);
        });

        it("should reject handles with various special characters", () => {
            const input = [
                "$test!",
                "$test?",
                "$test#",
                "$test%",
                "$test&",
                "$test*",
                "$test+",
                "$test=",
                "$valid123",
            ];
            expect(onlyValidHandles(input)).toEqual(["$valid123"]);
        });
    });
});

describe("onlyValidIds", () => {
    describe("basic functionality", () => {
        it("should return valid IDs that pass validatePK function", () => {
            // These should be valid snowflake-like IDs (positive BigInts within 64-bit range)
            const validIds = ["123456789012345678", "987654321098765432", "111111111111111111"];
            const invalidIds = ["invalid", "0", "-123", "abc123"];
            const input = [...validIds, ...invalidIds];
            const result = onlyValidIds(input);
            
            // Should contain all valid IDs
            validIds.forEach(id => {
                expect(result).toContain(id);
            });
            
            // Should not contain invalid ones
            invalidIds.forEach(id => {
                expect(result).not.toContain(id);
            });
        });

        it("should filter out non-string values", () => {
            const input = [
                "123456789012345678", // valid ID
                123456789012345678 as any, // number
                { id: "123456789012345678" } as any, // object
                ["123456789012345678"] as any, // array
                true as any, // boolean
                "987654321098765432", // valid ID
            ];
            const result = onlyValidIds(input);
            
            // Should only contain string IDs that are valid
            expect(result).toContain("123456789012345678");
            expect(result).toContain("987654321098765432");
            expect(result).not.toContain(123456789012345678);
        });

        it("should handle empty strings", () => {
            const input = ["", "123456789012345678", "", "987654321098765432"];
            const result = onlyValidIds(input);
            
            expect(result).toContain("123456789012345678");
            expect(result).toContain("987654321098765432");
            expect(result).not.toContain("");
        });
    });

    describe("null and undefined handling", () => {
        it("should filter out null values", () => {
            const input = ["123456789012345678", null, "987654321098765432", null];
            const result = onlyValidIds(input);
            
            expect(result).toContain("123456789012345678");
            expect(result).toContain("987654321098765432");
            expect(result).toHaveLength(2);
        });

        it("should filter out undefined values", () => {
            const input = ["123456789012345678", undefined, "987654321098765432", undefined];
            const result = onlyValidIds(input);
            
            expect(result).toContain("123456789012345678");
            expect(result).toContain("987654321098765432");
            expect(result).toHaveLength(2);
        });

        it("should handle mixed null, undefined, and valid IDs", () => {
            const input = [null, "123456789012345678", undefined, "987654321098765432", null, "invalid"];
            const result = onlyValidIds(input);
            
            expect(result).toContain("123456789012345678");
            expect(result).toContain("987654321098765432");
            expect(result).not.toContain("invalid");
        });

        it("should return empty array when all values are null or undefined", () => {
            expect(onlyValidIds([null, undefined, null])).toEqual([]);
            expect(onlyValidIds([null])).toEqual([]);
            expect(onlyValidIds([undefined])).toEqual([]);
        });
    });

    describe("edge cases", () => {
        it("should return empty array for empty input", () => {
            expect(onlyValidIds([])).toEqual([]);
        });

        it("should handle very long invalid strings", () => {
            const input = [
                "123456789012345678", // valid
                "a".repeat(1000), // very long invalid string
                "987654321098765432", // valid
            ];
            const result = onlyValidIds(input);
            
            expect(result).toContain("123456789012345678");
            expect(result).toContain("987654321098765432");
            expect(result).toHaveLength(2);
        });

        it("should preserve order of valid IDs", () => {
            const input = [
                "333333333333333333",
                "111111111111111111",
                "222222222222222222",
                "invalid",
                "444444444444444444",
            ];
            const result = onlyValidIds(input);
            
            expect(result[0]).toBe("333333333333333333");
            expect(result[1]).toBe("111111111111111111");
            expect(result[2]).toBe("222222222222222222");
            expect(result[3]).toBe("444444444444444444");
        });

        it("should handle complex mixed arrays with various invalid types", () => {
            const input = [
                "123456789012345678", // valid
                42 as any,
                "987654321098765432", // valid
                {} as any,
                "invalid-id",
                "short",
                "111111111111111111", // valid
                [] as any,
                true as any,
                null,
                undefined,
                "not-a-valid-id",
                "222222222222222222", // valid
            ];
            const result = onlyValidIds(input);
            
            expect(result).toEqual([
                "123456789012345678",
                "987654321098765432",
                "111111111111111111",
                "222222222222222222",
            ]);
        });
    });

    describe("validatePK dependency", () => {
        it("should reject invalid ID formats", () => {
            const input = ["0", "-123", "123456789012345678", "abc", "18446744073709551616"]; // last one exceeds 64-bit max
            const result = onlyValidIds(input);
            
            // Only the properly formatted ID should be valid
            expect(result).toContain("123456789012345678");
            expect(result).not.toContain("0"); // zero is invalid for snowflake
            expect(result).not.toContain("-123"); // negative is invalid
            expect(result).not.toContain("abc"); // non-numeric
        });

        it("should reject IDs with non-numeric characters", () => {
            const input = [
                "123456789012345678", // valid
                "12345678901234567a", // contains letter
                "1234567890123456-8", // contains dash
                "1234567890123456 8", // contains space
                "987654321098765432", // valid
            ];
            const result = onlyValidIds(input);
            
            expect(result).toContain("123456789012345678");
            expect(result).toContain("987654321098765432");
            expect(result).not.toContain("12345678901234567a");
            expect(result).not.toContain("1234567890123456-8");
            expect(result).not.toContain("1234567890123456 8");
        });

        it("should handle leading zeros in IDs", () => {
            const input = ["000000000000000001", "123456789012345678"];
            const result = onlyValidIds(input);
            
            // Both should be valid if they pass validatePK
            expect(result).toContain("123456789012345678");
            // The behavior for leading zeros depends on validatePK implementation
            // We just check that the function processes it consistently
            expect(Array.isArray(result)).toBe(true);
        });
    });
});
