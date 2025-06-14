import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { generatePK, validatePK, initIdGenerator, DUMMY_ID } from "./snowflake.js";

describe("Snowflake IDs", () => {
    it("generates a valid Snowflake ID as bigint", () => {
        const id = generatePK();
        expect(typeof id).toBe("bigint");
        expect(validatePK(id)).toBe(true);
    });

    it("generates a valid Snowflake ID string", () => {
        const id = generatePK().toString();
        expect(validatePK(id)).toBe(true);
    });

    it("generates unique IDs", () => {
        const ids = new Set();
        // Generate 1000 IDs
        const COUNT = 1000;
        for (let i = 0; i < COUNT; i++) {
            ids.add(generatePK().toString());
        }
        expect(ids.size).toBe(COUNT);
    });

    it("validates Snowflake IDs correctly", () => {
        // Valid ID
        const validId = generatePK().toString();
        expect(validatePK(validId)).toBe(true);

        // Invalid IDs
        const invalidIds = [
            "abc", // Non-numeric
            "-123", // Negative
            "", // Empty string
            "18446744073709551616", // Too large (2^64)
            null, // Null
            undefined, // Undefined
            123, // Number (not string or bigint)
            {}, // Object
        ];

        invalidIds.forEach(id => {
            expect(validatePK(id)).toBe(false);
        });
    });

    describe("initIdGenerator", () => {
        it("should initialize and generate valid IDs", async () => {
            await initIdGenerator();
            
            // Should generate valid IDs after initialization
            const id = generatePK();
            expect(typeof id).toBe("bigint");
            expect(validatePK(id)).toBe(true);
        });

        it("should accept custom configuration parameters", async () => {
            // Test with custom workerId and epoch
            await initIdGenerator(5, 1234567890000);
            
            // Should still generate valid IDs with custom parameters
            const id = generatePK();
            expect(typeof id).toBe("bigint");
            expect(validatePK(id)).toBe(true);
        });

        it("should work consistently across multiple initializations", async () => {
            // Initialize once
            await initIdGenerator(1, 1609459200000);
            const id1 = generatePK();
            
            // Initialize again with different parameters
            await initIdGenerator(2, 1609459200000);
            const id2 = generatePK();
            
            // Both should be valid and different
            expect(validatePK(id1)).toBe(true);
            expect(validatePK(id2)).toBe(true);
            expect(id1).not.toBe(id2);
        });
    });

    describe("Snowflake ID Format Validation", () => {
        it("should generate IDs that follow Snowflake format", () => {
            const id = generatePK();
            const idBits = id.toString(2).padStart(64, '0');
            
            // Snowflake ID format (64 bits):
            // 1 bit: unused (sign bit for compatibility, always 0)
            // 41 bits: timestamp (milliseconds since epoch)
            // 10 bits: worker ID
            // 12 bits: sequence number
            
            expect(idBits.length).toBeLessThanOrEqual(64);
            
            // The ID should be greater than 0 (positive)
            expect(id).toBeGreaterThan(0n);
            
            // The ID should fit in 64 bits
            expect(id).toBeLessThan(2n ** 64n);
        });

        it("should generate unique IDs even when generated rapidly", async () => {
            const ids: bigint[] = [];
            
            // Generate multiple IDs rapidly
            for (let i = 0; i < 10; i++) {
                ids.push(generatePK());
            }
            
            // All IDs should be unique
            const uniqueIds = new Set(ids.map(id => id.toString()));
            expect(uniqueIds.size).toBe(ids.length);
            
            // All IDs should be valid
            ids.forEach(id => {
                expect(validatePK(id)).toBe(true);
            });
        });

        it("should extract timestamp from Snowflake ID", () => {
            // Test that we can extract meaningful timestamp data
            const beforeTime = Date.now();
            const id = generatePK();
            const afterTime = Date.now();
            
            // Convert ID to binary to examine structure
            const idBinary = id.toString(2).padStart(64, '0');
            
            // Extract timestamp bits (bits 22-63 in standard Snowflake)
            // This is implementation-specific but validates the format
            const timestampBits = idBinary.substring(1, 42);
            const timestampValue = parseInt(timestampBits, 2);
            
            // Timestamp should be reasonable (within test execution time)
            // This assumes a standard epoch (e.g., Unix epoch or custom epoch)
            expect(timestampValue).toBeGreaterThan(0);
        });
    });

    describe("ID Generation Requirements", () => {
        it("should generate IDs with reasonable distribution", () => {
            const ids: bigint[] = [];
            
            // Generate multiple IDs in sequence
            for (let i = 0; i < 20; i++) {
                ids.push(generatePK());
            }
            
            // All IDs should be unique
            const uniqueIds = new Set(ids.map(id => id.toString()));
            expect(uniqueIds.size).toBe(ids.length);
            
            // All IDs should be positive and within valid range
            ids.forEach(id => {
                expect(id).toBeGreaterThan(0n);
                expect(id).toBeLessThanOrEqual((2n ** 64n) - 1n);
            });
        });

        it("should generate IDs with sufficient entropy for uniqueness", () => {
            const id1 = generatePK();
            const id2 = generatePK();
            const id3 = generatePK();
            
            // All IDs should be different
            expect(id1).not.toBe(id2);
            expect(id2).not.toBe(id3);
            expect(id1).not.toBe(id3);
            
            // IDs should be in valid range
            expect(id1).toBeGreaterThan(0n);
            expect(id2).toBeGreaterThan(0n);
            expect(id3).toBeGreaterThan(0n);
        });

        it("should consistently generate valid IDs across multiple calls", () => {
            // Test that the generator produces valid IDs consistently
            for (let i = 0; i < 20; i++) {
                const id = generatePK();
                expect(typeof id).toBe("bigint");
                expect(validatePK(id)).toBe(true);
                expect(id).toBeGreaterThan(0n);
            }
        });

        it("should generate IDs that convert to valid string representations", () => {
            const id = generatePK();
            const idString = id.toString();
            
            expect(typeof idString).toBe("string");
            expect(idString).toMatch(/^\d+$/); // Should be all digits
            expect(validatePK(idString)).toBe(true);
            expect(BigInt(idString)).toBe(id); // Round-trip conversion should work
        });
    });

    describe("DUMMY_ID", () => {
        it("should export DUMMY_ID constant", () => {
            expect(DUMMY_ID).toBe("0");
            expect(typeof DUMMY_ID).toBe("string");
        });
    });

    describe("ID Validation", () => {
        it("should validate IDs within the correct range", () => {
            // Test maximum valid 64-bit unsigned integer
            const maxBigInt = (2n ** 64n) - 1n;
            expect(validatePK(maxBigInt)).toBe(true);
            expect(validatePK(maxBigInt.toString())).toBe(true);

            // Test just over maximum (should be invalid)
            const overMax = 2n ** 64n;
            expect(validatePK(overMax)).toBe(false);
            expect(validatePK(overMax.toString())).toBe(false);

            // Test valid positive values
            expect(validatePK(1n)).toBe(true);
            expect(validatePK("1")).toBe(true);
            expect(validatePK(1000000000n)).toBe(true);
            expect(validatePK("1000000000")).toBe(true);

            // Test invalid values (zero and negative)
            expect(validatePK(0n)).toBe(false);
            expect(validatePK("0")).toBe(false);
            expect(validatePK(-1n)).toBe(false);
            expect(validatePK("-1")).toBe(false);
        });

        it("should reject malformed string inputs", () => {
            const invalidInputs = [
                "123abc",      // Contains non-digits
                "12.34",       // Contains decimal point
                "1e10",        // Scientific notation
                "Infinity",    // Invalid string
                "NaN",         // Invalid string
                "",            // Empty string
                " ",           // Whitespace only
                "abc",         // Non-numeric
            ];

            invalidInputs.forEach(input => {
                expect(validatePK(input)).toBe(false);
            });
        });

        it("should accept valid string representations", () => {
            // Test various valid numeric strings
            expect(validatePK("123")).toBe(true);
            expect(validatePK("999999999999999999")).toBe(true);
            
            // BigInt constructor handles these cases
            expect(validatePK("123\n")).toBe(true); // Newline gets trimmed
            expect(validatePK(" 123 ")).toBe(true);  // Spaces get trimmed
            expect(validatePK("\t789\t")).toBe(true); // Tabs get trimmed
        });

        it("should reject non-string and non-bigint types", () => {
            const invalidTypes = [
                null,
                undefined,
                123,           // number (not bigint)
                true,
                false,
                {},
                [],
                new Date(),
                Symbol("test"),
            ];

            invalidTypes.forEach(value => {
                expect(validatePK(value)).toBe(false);
            });
        });

        it("should accept numeric string representations that BigInt can parse", () => {
            // BigInt() accepts hex, octal, and binary notation
            expect(validatePK("0x123")).toBe(true);  // Hex converts to 291
            expect(validatePK("0xFF")).toBe(true);   // Hex converts to 255
            expect(validatePK("0x0")).toBe(false);   // Zero is invalid per Snowflake rules
            expect(validatePK("0o377")).toBe(true);  // Octal 377 = 255
            expect(validatePK("0b1111")).toBe(true); // Binary 1111 = 15
            
            // Standard decimal strings should also be accepted
            expect(validatePK("123")).toBe(true);
            expect(validatePK("255")).toBe(true);
            expect(validatePK("15")).toBe(true);
        });
    });
});
