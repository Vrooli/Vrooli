import { addHttps } from "./addHttps";

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
            test(`"${value}" should not be modified`, () => {
                expect(addHttps(value)).toEqual(value);
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
            test(`"${input}" should be modified to "${expected}"`, () => {
                expect(addHttps(input)).toEqual(expected);
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
            test(`"${input}" should return "${expected}"`, () => {
                expect(addHttps(input)).toEqual(expected);
            });
        });
    });
});
