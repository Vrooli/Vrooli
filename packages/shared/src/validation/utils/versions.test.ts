/* eslint-disable @typescript-eslint/ban-ts-comment */
import { calculateVersionsFromString, getMinVersion, meetsMinVersion, minVersionTest, sortVersions } from "./versions";

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

describe("getMinVersion function tests", () => {
    test("versions in random order", () => {
        const versions = ["1.0.0", "2.0.0", "1.5.0"];
        const result = getMinVersion(versions);
        expect(result).toBe("2.0.0");
    });

    test("versions in ascending order", () => {
        const versions = ["1.0.0", "1.5.0", "2.0.0"];
        const result = getMinVersion(versions);
        expect(result).toBe("2.0.0");
    });

    test("versions in descending order", () => {
        const versions = ["2.0.0", "1.5.0", "1.0.0"];
        const result = getMinVersion(versions);
        expect(result).toBe("2.0.0");
    });

    test("empty list of versions", () => {
        const versions = [];
        const result = getMinVersion(versions);
        expect(result).toBe("0.0.1");
    });

    test("list with only one version", () => {
        const versions = ["1.0.0"];
        const result = getMinVersion(versions);
        expect(result).toBe("1.0.0");
    });

    test("list with duplicate versions", () => {
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

describe("minVersionTest function tests", () => {
    const minVersion = "1.0.0";

    test("version meets the minimum version requirement", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn("1.0.1")).toBe(true);
    });

    test("version does not meet the minimum version requirement", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn("0.9.9")).toBe(false);
    });

    test("undefined version", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn(undefined)).toBe(true);
    });

    test("minimum version as input version", () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn(minVersion)).toBe(true);
    });
});

describe("sortVersions", () => {
    it("should correctly sort an array of versions by major, moderate, and minor", () => {
        const versions = [
            { versionLabel: "1.2.3" },
            { versionLabel: "1.2.1" },
            { versionLabel: "2.1.1" },
            { versionLabel: "1.3.1" },
            { versionLabel: "0.9.9" },
        ];
        const sorted = sortVersions(versions);
        expect(sorted.map(v => v.versionLabel)).toEqual(["0.9.9", "1.2.1", "1.2.3", "1.3.1", "2.1.1"]);
    });

    it("should return an empty array when provided with a non-array input", () => {
        const invalidInput = "not an array";
        // @ts-ignore: Testing runtime scenario
        const sorted = sortVersions(invalidInput);
        expect(sorted).toEqual([]);
    });

    it("should return an empty array when provided with an empty array", () => {
        const emptyArray = [];
        const sorted = sortVersions(emptyArray);
        expect(sorted).toEqual([]);
    });

    it("should sort versions that are identical", () => {
        const versions = [
            { versionLabel: "1.1.1" },
            { versionLabel: "1.1.1" },
            { versionLabel: "1.1.1" },
        ];
        const sorted = sortVersions(versions);
        expect(sorted.map(v => v.versionLabel)).toEqual(["1.1.1", "1.1.1", "1.1.1"]);
    });

    it("should maintain the order of elements with the same version", () => {
        // Assuming stable sort
        const versions = [
            { versionLabel: "1.1.1", name: "A" },
            { versionLabel: "1.1.1", name: "B" },
            { versionLabel: "1.1.1", name: "C" },
        ];
        const sorted = sortVersions(versions);
        expect(sorted.map(v => v.name)).toEqual(["A", "B", "C"]);
    });
});

