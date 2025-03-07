import { expect } from "chai";
import { addHttps, toPosDouble, toPosInt } from "./convert.js";

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
            it(`"${input}" should be converted to ${expected}`, () => {
                expect(toPosDouble(input)).to.deep.equal(expected);
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
            it(`"${input}" should be converted to ${expected}`, () => {
                const result = toPosDouble(input);
                if (isNaN(expected)) {
                    expect(result).toBeNaN();
                } else {
                    expect(result).to.deep.equal(expected);
                }
            });
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
                expect(toPosInt(input)).to.deep.equal(expected);
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
            { input: "123.456", expected: 123456 }, // decimal number
            { input: "1e2", expected: 12 }, // scientific notation
            { input: "Infinity", expected: NaN }, // Infinity
        ];

        // Test each edge case
        edgeCases.forEach(({ input, expected }) => {
            it(`"${input}" should be converted to ${expected}`, () => {
                const result = toPosInt(input);
                if (isNaN(expected)) {
                    expect(result).toBeNaN();
                } else {
                    expect(result).to.deep.equal(expected);
                }
            });
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
                expect(addHttps(value)).to.deep.equal(value);
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
                expect(addHttps(input)).to.deep.equal(expected);
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
                expect(addHttps(input)).to.deep.equal(expected);
            });
        });
    });
});
