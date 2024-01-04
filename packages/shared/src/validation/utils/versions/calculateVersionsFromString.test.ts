import { calculateVersionsFromString } from "./calculateVersionsFromString";

describe("Version Calculation Tests", () => {
    // Test suite for valid version strings
    describe("Valid Version Strings", () => {
        // Test cases with expected results
        const validCases = [
            { version: "1.2.3", expected: { major: 1, moderate: 2, minor: 3 } },
            { version: "1", expected: { major: 1, moderate: 0, minor: 0 } },
            { version: "1.2", expected: { major: 1, moderate: 2, minor: 0 } },
            { version: "0.1.1", expected: { major: 0, moderate: 1, minor: 1 } },
            // ... add more as needed
        ];

        // Test each valid case
        validCases.forEach(({ version, expected }) => {
            test(`"${version}" should return ${JSON.stringify(expected)}`, () => {
                expect(calculateVersionsFromString(version)).toEqual(expected);
            });
        });
    });

    // Test suite for invalid version strings
    describe("Invalid Version Strings", () => {
        // Test cases with expected results for invalid version strings
        const invalidCases = [
            { version: "1.a.3", expected: { major: 1, moderate: 0, minor: 3 } }, // 'a' is invalid, so moderate is 0
            { version: "1.2.b", expected: { major: 1, moderate: 2, minor: 0 } }, // 'b' is invalid, so minor is 0
            { version: "1.2.3.4", expected: { major: 1, moderate: 2, minor: 3 } }, // only consider the first three parts
        ];

        // Test each invalid case
        invalidCases.forEach(({ version, expected }) => {
            test(`"${version}" should return ${JSON.stringify(expected)}`, () => {
                expect(calculateVersionsFromString(version)).toEqual(expected);
            });
        });
    });
});

