import { toPosDouble } from "./toPosDouble";

describe("Positive Double Conversion Tests", () => {
    // Test suite for typical cases
    describe("Typical Cases", () => {
        // Test cases with expected results
        const typicalCases = [
            { input: "123", expected: 123 }, // pure numeric string
            { input: "a1b2c3", expected: 123 }, // alphanumeric string
            { input: "  0045 ", expected: 45 }, // leading and trailing whitespaces
            { input: "1,234", expected: 1234 }, // comma in number
            { input: "123.456", expected: 123.456 }, // decimal number
            { input: "1e2", expected: 12 }, // scientific notation
        ];

        // Test each typical case
        typicalCases.forEach(({ input, expected }) => {
            test(`"${input}" should be converted to ${expected}`, () => {
                expect(toPosDouble(input)).toEqual(expected);
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
            { input: "-123", expected: 123 }, // negative number
            { input: "Infinity", expected: NaN }, // Infinity
        ];

        // Test each edge case
        edgeCases.forEach(({ input, expected }) => {
            test(`"${input}" should be converted to ${expected}`, () => {
                const result = toPosDouble(input);
                if (isNaN(expected)) {
                    expect(result).toBeNaN();
                } else {
                    expect(result).toEqual(expected);
                }
            });
        });
    });
});