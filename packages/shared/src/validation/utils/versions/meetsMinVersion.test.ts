import { meetsMinVersion } from "./meetsMinVersion";

describe("Minimum Version Requirement Tests", () => {
    // Test suite for valid comparisons
    describe("Valid Comparisons", () => {
        // Test cases with expected results
        const comparisonCases = [
            { version: "1.2.3", minimumVersion: "1.2.3", expected: true }, // exactly the same
            { version: "1.2.4", minimumVersion: "1.2.3", expected: true }, // minor version higher
            { version: "1.3.0", minimumVersion: "1.2.9", expected: true }, // moderate version higher
            { version: "2.0.0", minimumVersion: "1.9.9", expected: true }, // major version higher
            { version: "1.2.2", minimumVersion: "1.2.3", expected: false }, // minor version lower
            { version: "1.1.9", minimumVersion: "1.2.0", expected: false }, // moderate version lower
            { version: "0.9.9", minimumVersion: "1.0.0", expected: false }, // major version lower
            // edge cases
            { version: "1.2", minimumVersion: "1.2.0", expected: true }, // implicit minor version
            { version: "1", minimumVersion: "1.0.0", expected: true }, // implicit moderate and minor version
            { version: "1.2.3.4", minimumVersion: "1.2.3", expected: true }, // extra parts in version
        ];

        // Test each case
        comparisonCases.forEach(({ version, minimumVersion, expected }) => {
            test(`"${version}" compared to "${minimumVersion}" should return ${expected}`, () => {
                expect(meetsMinVersion(version, minimumVersion)).toEqual(expected);
            });
        });
    });

    // Test suite for invalid or unusual inputs
    describe("Invalid or Unusual Inputs", () => {
        // Test cases with expected results
        const invalidCases = [
            { version: "asdf", minimumVersion: "1.2.3", expected: false }, // non-numeric version
            { version: "1.2.3", minimumVersion: "asdf", expected: true }, // non-numeric minimum version
            { version: "", minimumVersion: "1.2.3", expected: false }, // empty version string
            { version: "1.2.3", minimumVersion: "", expected: true }, // empty minimum version string
        ];

        // Test each invalid or unusual case
        invalidCases.forEach(({ version, minimumVersion, expected }) => {
            test(`"${version}" compared to "${minimumVersion}" should return ${expected}`, () => {
                expect(meetsMinVersion(version, minimumVersion)).toEqual(expected);
            });
        });
    });
});

