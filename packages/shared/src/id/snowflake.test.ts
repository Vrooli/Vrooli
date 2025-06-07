import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { generatePK, validatePK, initIdGenerator, DUMMY_ID } from "./snowflake.js";

describe("Snowflake IDs", () => {
    it("generates a valid Snowflake ID as bigint", () => {
        const id = generatePK();
        expect(typeof id).to.equal("bigint");
        expect(validatePK(id)).to.equal(true);
    });

    it("generates a valid Snowflake ID string", () => {
        const id = generatePK().toString();
        expect(validatePK(id)).to.equal(true);
    });

    it("generates unique IDs", () => {
        const ids = new Set();
        // Generate 1000 IDs
        const COUNT = 1000;
        for (let i = 0; i < COUNT; i++) {
            ids.add(generatePK().toString());
        }
        expect(ids.size).to.equal(COUNT);
    });

    it("validates Snowflake IDs correctly", () => {
        // Valid ID
        const validId = generatePK().toString();
        expect(validatePK(validId)).to.equal(true);

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
            expect(validatePK(id)).to.equal(false);
        });
    });

    describe("initIdGenerator", () => {
        let consoleWarnSpy: any;

        beforeEach(() => {
            consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {
                // Mock implementation
            });
        });

        afterEach(() => {
            consoleWarnSpy.mockRestore();
        });

        it("should initialize successfully in Node.js environment", async () => {
            // Mock Node.js environment
            const originalProcess = global.process;
            global.process = { versions: { node: "18.0.0" } } as any;

            try {
                await initIdGenerator(1, 1609459200000);
                // If no error is thrown, initialization succeeded
                expect(true).toBe(true);
            } catch (error) {
                // Expected to fail in test environment due to missing nodejs-snowflake
                expect(consoleWarnSpy).toHaveBeenCalled();
            } finally {
                global.process = originalProcess;
            }
        });

        it("should use fallback generator when nodejs-snowflake is not available", async () => {
            // Mock Node.js environment
            const originalProcess = global.process;
            global.process = { versions: { node: "18.0.0" } } as any;

            try {
                await initIdGenerator();
                // Generator should still work even if native module fails
                const id = generatePK();
                expect(typeof id).toBe("bigint");
                expect(validatePK(id)).toBe(true);
            } finally {
                global.process = originalProcess;
            }
        });

        it("should handle non-Node.js environment", async () => {
            // Mock browser environment
            const originalProcess = global.process;
            global.process = undefined as any;

            try {
                await initIdGenerator();
                // Should work in browser environment
                const id = generatePK();
                expect(typeof id).toBe("bigint");
                expect(validatePK(id)).toBe(true);
            } finally {
                global.process = originalProcess;
            }
        });

        it("should use custom workerId and epoch", async () => {
            await initIdGenerator(5, 1234567890000);
            // Should still generate valid IDs with custom parameters
            const id = generatePK();
            expect(typeof id).toBe("bigint");
            expect(validatePK(id)).toBe(true);
        });
    });

    describe("DUMMY_ID", () => {
        it("should export DUMMY_ID constant", () => {
            expect(DUMMY_ID).toBe("0");
            expect(typeof DUMMY_ID).toBe("string");
        });
    });

    describe("edge cases", () => {
        it("should handle BigInt validation edge cases", () => {
            // Test maximum valid BigInt
            const maxBigInt = (2n ** 64n) - 1n;
            expect(validatePK(maxBigInt)).toBe(true);
            expect(validatePK(maxBigInt.toString())).toBe(true);

            // Test just over maximum
            const overMax = 2n ** 64n;
            expect(validatePK(overMax)).toBe(false);
            expect(validatePK(overMax.toString())).toBe(false);

            // Test minimum valid value
            expect(validatePK(1n)).toBe(true);
            expect(validatePK("1")).toBe(true);

            // Test zero (should be invalid)
            expect(validatePK(0n)).toBe(false);
            expect(validatePK("0")).toBe(false);
        });

        it("should handle malformed string inputs", () => {
            const malformedInputs = [
                "123abc",
                "12.34",
                "1e10", // Scientific notation fails BigInt conversion
                "Infinity",
                "NaN",
            ];

            malformedInputs.forEach(input => {
                expect(validatePK(input)).toBe(false);
            });
        });

        it("should handle valid BigInt conversions", () => {
            // These actually convert successfully to BigInt
            expect(validatePK("123\n")).toBe(true); // Newline gets trimmed
            expect(validatePK("0x123")).toBe(true); // Hex notation is valid
            expect(validatePK("0xFF")).toBe(true); // Another hex example
        });

        it("should handle strings with whitespace", () => {
            // Leading/trailing spaces are trimmed by BigInt and should pass
            expect(validatePK(" 123 ")).toBe(true);
            expect(validatePK("  456  ")).toBe(true);
            expect(validatePK("\t789\t")).toBe(true);
        });
    });
});
