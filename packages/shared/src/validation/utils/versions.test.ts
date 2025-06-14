/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, it, expect } from "vitest";
import { calculateVersionsFromString, getMinVersion, meetsMinVersion } from "./versions.js";

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
            it(`"${version}" should return ${JSON.stringify(expected)}`, () => {
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
            it(`"${version}" should return ${JSON.stringify(expected)}`, () => {
                expect(calculateVersionsFromString(version)).toEqual(expected);
            });
        });
    });
});

describe("getMinVersion function tests", () => {
    it("versions in random order", () => {
        const versions = ["1.0.0", "2.0.0", "1.5.0"];
        const result = getMinVersion(versions);
        expect(result).toBe("2.0.0");
    });

    it("versions in ascending order", () => {
        const versions = ["1.0.0", "1.5.0", "2.0.0"];
        const result = getMinVersion(versions);
        expect(result).toBe("2.0.0");
    });

    it("versions in descending order", () => {
        const versions = ["2.0.0", "1.5.0", "1.0.0"];
        const result = getMinVersion(versions);
        expect(result).toBe("2.0.0");
    });

    it("empty list of versions", () => {
        const versions = [];
        const result = getMinVersion(versions);
        expect(result).toBe("0.0.1");
    });

    it("list with only one version", () => {
        const versions = ["1.0.0"];
        const result = getMinVersion(versions);
        expect(result).toBe("1.0.0");
    });

    it("list with duplicate versions", () => {
        const versions = ["1.0.0", "1.0.0", "2.0.0"];
        const result = getMinVersion(versions);
        expect(result).toBe("2.0.0");
    });
});

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
            it(`"${version}" compared to "${minimumVersion}" should return ${expected}`, () => {
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
            it(`"${version}" compared to "${minimumVersion}" should return ${expected}`, () => {
                expect(meetsMinVersion(version, minimumVersion)).toEqual(expected);
            });
        });
    });
});


