import { describe, it, expect, vi } from "vitest";
import { addHttps, toDouble, toPosDouble, toPosInt, enumToYup } from "./convert.js";
import "../yupAugmentations.js"; // Import yup augmentations

describe("Positive Double Conversion Tests", () => {
    // Test suite for typical cases
    describe("Typical Cases", () => {
        // Test cases with expected results
        const typicalCases = [
            { input: "123", expected: 123 }, // pure numeric string
            { input: "a1b2c3", expected: 123 }, // alphanumeric string - extracts numeric digits
            { input: "  0045 ", expected: 45 }, // leading and trailing whitespaces
            { input: "1,234", expected: 1234 }, // comma in number
            { input: "123.456", expected: 123.456 }, // decimal number
        ];

        // Test each typical case
        typicalCases.forEach(({ input, expected }) => {
            it(`"${input}" should be converted to ${expected}`, () => {
                expect(toPosDouble(input)).toBe(expected);
            });
        });
    });

    // Test suite for number inputs
    describe("Number Inputs", () => {
        const numberCases = [
            { input: 123, expected: 123 }, // positive number
            { input: -456, expected: 0 }, // negative number becomes 0
            { input: 0, expected: 0 }, // zero
            { input: 123.456, expected: 123.456 }, // decimal
            { input: -123.456, expected: 0 }, // negative decimal becomes 0
        ];

        numberCases.forEach(({ input, expected }) => {
            it(`number ${input} should be converted to ${expected}`, () => {
                expect(toPosDouble(input)).toBe(expected);
            });
        });
    });

    // Test suite for edge cases
    describe("Edge Cases", () => {
        it("should return NaN for empty or non-numeric strings", () => {
            expect(isNaN(toPosDouble(""))).toBe(true);
            expect(isNaN(toPosDouble("abcdef"))).toBe(true);
            expect(isNaN(toPosDouble("Infinity"))).toBe(true);
        });

        it("should convert zero strings correctly", () => {
            expect(toPosDouble("0000")).toBe(0);
            expect(toPosDouble("0.0")).toBe(0);
        });

        it("should handle negative number strings by extracting digits only", () => {
            // Current behavior extracts digits, ignoring negative sign
            // This is questionable for a "positive double" function
            // Should probably return 0 for negative inputs instead
            expect(toPosDouble("-123")).toBe(123); // Current behavior
            expect(toPosDouble("-45.67")).toBe(45.67); // Current behavior
        });
    });

    // Test suite for intended behavior
    describe("Intended Behavior Tests", () => {
        it("should extract numeric values from currency formats", () => {
            expect(toPosDouble("$123.45")).toBe(123.45);
            expect(toPosDouble("€456.78")).toBe(456.78);
            expect(toPosDouble("£789.00")).toBe(789.00);
        });

        it("should extract numeric values from percentage strings", () => {
            expect(toPosDouble("50%")).toBe(50);
            expect(toPosDouble("12.5%")).toBe(12.5);
        });

        it("should clamp negative numbers to 0 for positive double", () => {
            expect(toPosDouble(-123)).toBe(0);
            expect(toPosDouble(-45.67)).toBe(0);
        });

        it("should preserve precision for large numbers", () => {
            expect(toPosDouble("1234567890.12")).toBe(1234567890.12);
            expect(toPosDouble(Number.MAX_SAFE_INTEGER.toString())).toBe(Number.MAX_SAFE_INTEGER);
        });

        it("should properly parse scientific notation", () => {
            expect(toPosDouble("1e2")).toBe(100);
            expect(toPosDouble("2.5e3")).toBe(2500);
            expect(toPosDouble("1.23e-2")).toBe(0.0123);
            expect(toPosDouble("-5e2")).toBe(0); // Negative scientific notation clamped to 0
        });

        it("should handle mixed formats with text", () => {
            expect(toPosDouble("Price: $42.99")).toBe(42.99);
            expect(toPosDouble("Total 123.45 USD")).toBe(123.45);
            expect(toPosDouble("Item #567")).toBe(567);
        });
    });
});

describe("Double Conversion Tests", () => {
    // Test suite for typical cases
    describe("Typical Cases", () => {
        // Test cases with expected results
        const typicalCases = [
            { input: "123", expected: 123 }, // pure numeric string
            { input: "a1b2c3", expected: 123 }, // alphanumeric string
            { input: "  0045 ", expected: 45 }, // leading and trailing whitespaces
            { input: "1,234", expected: 1234 }, // comma in number
            { input: "123.456", expected: 123.456 }, // decimal number
            { input: "0.8", expected: 0.8 }, // decimal less than 1
            { input: "-0.8", expected: -0.8 }, // negative decimal
            { input: 0.8, expected: 0.8 }, // number input
        ];

        // Test each typical case
        typicalCases.forEach(({ input, expected }) => {
            it(`"${input}" should be converted to ${expected}`, () => {
                expect(toDouble(input)).toBe(expected);
            });
        });
    });

    // Test suite for edge cases
    describe("Edge Cases", () => {
        // Cases with expected results for edge cases
        const edgeCases = [
            { input: "", expected: NaN }, // empty string
            { input: "abcdef", expected: NaN }, // no digits
            { input: "0000", expected: 0 }, // all zeros
            { input: "Infinity", expected: NaN }, // Infinity
        ];

        // Test each edge case
        edgeCases.forEach(({ input, expected }) => {
            it(`"${input}" should be converted to ${expected}`, () => {
                const result = toDouble(input);
                if (isNaN(expected)) {
                    expect(isNaN(result)).toBe(true);
                } else {
                    expect(result).toBe(expected);
                }
            });
        });
    });

    // Test suite for expected behavior
    describe("Expected Behavior Tests", () => {
        it("should handle negative numbers correctly", () => {
            expect(toDouble("-123")).toBe(-123);
            expect(toDouble("-123.45")).toBe(-123.45);
            expect(toDouble("  -456  ")).toBe(-456);
        });

        it("should handle scientific notation properly", () => {
            expect(toDouble("1e2")).toBe(100);
            expect(toDouble("-1e2")).toBe(-100);
            expect(toDouble("2.5e3")).toBe(2500);
            expect(toDouble("-2.5e3")).toBe(-2500);
            expect(toDouble("1.23e-2")).toBe(0.0123);
        });

        it("should handle currency formats", () => {
            expect(toDouble("$123.45")).toBe(123.45);
            expect(toDouble("-$123.45")).toBe(-123.45);
            expect(toDouble("€456.78")).toBe(456.78);
        });
    });
});

describe("Positive Integer Conversion Tests", () => {
    // Test suite for typical cases
    describe("Typical Cases", () => {
        // Test cases with expected results
        const typicalCases = [
            { input: "123", expected: 123 }, // pure numeric string
            { input: "a1b2c3", expected: 123 }, // alphanumeric string
            { input: "  0045 ", expected: 45 }, // leading and trailing whitespaces
            { input: "1,234", expected: 1234 }, // comma in number
        ];

        // Test each typical case
        typicalCases.forEach(({ input, expected }) => {
            it(`"${input}" should be converted to ${expected}`, () => {
                expect(toPosInt(input)).toBe(expected);
            });
        });
    });

    // Test suite for edge cases
    describe("Edge Cases", () => {
        // Cases with expected results for edge cases
        const edgeCases = [
            { input: "", expected: NaN }, // empty string
            { input: "abcdef", expected: NaN }, // no digits
            { input: "0000", expected: 0 }, // all zeros
            { input: "-123", expected: 123 }, // negative number - strips negative sign
            { input: "Infinity", expected: NaN }, // Infinity string
        ];

        // Test each edge case
        edgeCases.forEach(({ input, expected }) => {
            it(`"${input}" should be converted to ${expected}`, () => {
                const result = toPosInt(input);
                if (isNaN(expected)) {
                    expect(isNaN(result)).toBe(true);
                } else {
                    expect(result).toBe(expected);
                }
            });
        });
    });

    // Test suite for expected behavior
    describe("Expected Behavior Tests", () => {
        it("should extract integers from mixed content", () => {
            expect(toPosInt("Item #123")).toBe(123);
            expect(toPosInt("Order-456-ABC")).toBe(456);
            expect(toPosInt("ID: 789")).toBe(789);
        });

        it("should handle numbers with thousands separators", () => {
            expect(toPosInt("1,000")).toBe(1000);
            expect(toPosInt("1,234,567")).toBe(1234567);
        });

        it("should handle decimal numbers by truncating (not rounding)", () => {
            expect(toPosInt("123.456")).toBe(123);
            expect(toPosInt("999.999")).toBe(999);
            expect(toPosInt("0.5")).toBe(0);
            expect(toPosInt("0.99")).toBe(0);
            expect(toPosInt("10.99")).toBe(10);
        });

        it("should handle scientific notation properly", () => {
            expect(toPosInt("1e3")).toBe(1000);
            expect(toPosInt("2.5e2")).toBe(250); // 2.5 * 100 = 250, truncated to 250
            expect(toPosInt("1.5e1")).toBe(15); // 1.5 * 10 = 15
            expect(toPosInt("9.99e2")).toBe(999); // 9.99 * 100 = 999
        });
    });
});

describe("HTTPS Addition Tests", () => {
    // Test suite for URLs that should not be modified
    describe("No Modification Required", () => {
        // URLs and strings that should not change
        const noChangeCases = [
            "http://www.example.com",
            "https://secure.example.com",
            "addr1q9pdxnvnezqpj3hph45ltkr2z3fyllprd2j9cl3zv3j8y5ghl4xyz5hfkdf82jlqm4w8sn3kx2tjy0s8fluxz9flkjqquvvyg9", // valid wallet address
            "username", // valid handle
        ];

        // Test each case
        noChangeCases.forEach(value => {
            it(`"${value}" should not be modified`, () => {
                expect(addHttps(value)).toBe(value);
            });
        });
    });

    // Test suite for URLs that should be modified
    describe("Modification Required", () => {
        // Cases where 'https://' should be added
        const changeCases = [
            { input: "www.example.com", expected: "https://www.example.com" },
            { input: "example.com/path?query=string", expected: "https://example.com/path?query=string" },
            { input: "subdomain.example.co.uk", expected: "https://subdomain.example.co.uk" },
        ];

        // Test each case
        changeCases.forEach(({ input, expected }) => {
            it(`"${input}" should be modified to "${expected}"`, () => {
                expect(addHttps(input)).toBe(expected);
            });
        });
    });

    // Test suite for invalid or unusual inputs
    describe("Invalid or Unusual Inputs", () => {
        // Cases with expected results for invalid or unusual inputs
        const invalidCases = [
            { input: undefined, expected: "" }, // undefined input
            { input: "ftp://example.org", expected: "ftp://example.org" }, // unsupported protocol
            { input: "JustSomeRandomString", expected: "JustSomeRandomString" }, // not a valid URL
        ];

        // Test each invalid or unusual case
        invalidCases.forEach(({ input, expected }) => {
            it(`"${input}" should return "${expected}"`, () => {
                expect(addHttps(input)).toBe(expected);
            });
        });
    });
});

describe("Enum to Yup Tests", () => {
    // Test suite for normal enum objects
    describe("Normal Enum Objects", () => {
        it("should convert enum object to yup oneOf", () => {
            const testEnum = {
                VALUE1: "value1",
                VALUE2: "value2", 
                VALUE3: "value3",
            };
            
            const schema = enumToYup(testEnum);
            
            // Test valid values
            expect(() => schema.validateSync("value1")).not.toThrow();
            expect(() => schema.validateSync("value2")).not.toThrow();
            expect(() => schema.validateSync("value3")).not.toThrow();
            
            // Test invalid value
            expect(() => schema.validateSync("invalid")).toThrow();
        });
    });

    // Test suite for function-based enums
    describe("Function-based Enums", () => {
        it("should handle enum functions", () => {
            const enumFunction = () => ({
                OPTION_A: "optionA",
                OPTION_B: "optionB",
            });
            
            const schema = enumToYup(enumFunction);
            
            // Test valid values
            expect(() => schema.validateSync("optionA")).not.toThrow();
            expect(() => schema.validateSync("optionB")).not.toThrow();
            
            // Test invalid value
            expect(() => schema.validateSync("optionC")).toThrow();
        });
    });

    // Test suite for undefined/null enum handling
    describe("Undefined Enum Handling", () => {
        it("should handle undefined enum gracefully", () => {
            const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
            
            const schema = enumToYup(undefined as any);
            
            // Should log a warning
            expect(consoleSpy).toHaveBeenCalledWith(
                "enumToYup called with undefined enum. This may be due to circular dependencies.",
            );
            
            // Should return a basic string schema that accepts any trimmed string
            expect(() => schema.validateSync("anyString")).not.toThrow();
            expect(() => schema.validateSync("  trimmed  ")).not.toThrow();
            
            consoleSpy.mockRestore();
        });

        it("should handle function that returns undefined", () => {
            const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
            
            const enumFunction = () => undefined as any;
            const schema = enumToYup(enumFunction);
            
            // Should log a warning
            expect(consoleSpy).toHaveBeenCalledWith(
                "enumToYup called with undefined enum. This may be due to circular dependencies.",
            );
            
            // Should return a basic string schema
            expect(() => schema.validateSync("anyString")).not.toThrow();
            
            consoleSpy.mockRestore();
        });
    });
});

