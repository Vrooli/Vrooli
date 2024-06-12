/**
 * Calculates major, moderate, and minor versions from a version string
 * Ex: 1.2.3 => major = 1, moderate = 2, minor = 3
 * Ex: 1 => major = 1, moderate = 0, minor = 0
 * Ex: 1.2 => major = 1, moderate = 2, minor = 0
 * Ex: asdfasdf (or any other invalid number) => major = 1, moderate = 0, minor = 0
 * @param version Version string
 * @returns Major, moderate, and minor versions
 */
export const calculateVersionsFromString = (version: string): { major: number, moderate: number, minor: number } => {
    const [major, moderate, minor] = version.split(".").map(v => parseInt(v));
    return {
        major: major || 0,
        moderate: moderate || 0,
        minor: minor || 0,
    };
};

/**
* Determines mimimum version. This is either: 
* 1. The highest version in the list of versions
* 2. A version larger than the highest version in the list of versions
* 3. Any version number greater than 0.0.0, if there are no versions in the list
* @param versions The list of versions
* @returns The minimum version
*/
export const getMinVersion = (versions: string[]): string => {
    // If no versions, return 0.0.1
    if (versions.length === 0) return "0.0.1";
    // Sort using meetsMinVersion, which determines if a version is greater than or equal to another version
    versions.sort((a, b) => meetsMinVersion(a, b) ? 1 : -1);
    // Get the highest version
    const highestVersion = versions[versions.length - 1];
    return highestVersion ?? "0.0.1";
};

/**
 * Determines if a version is greater than or equal to the minimum version
 * @param version Version string
 * @param minimumVersion Minimum version string
 * @returns True if version is greater than or equal to minimum version
 */
export const meetsMinVersion = (version: string, minimumVersion: string): boolean => {
    // Parse versions
    const { major: major1, moderate: moderate1, minor: minor1 } = calculateVersionsFromString(version);
    const { major: major2, moderate: moderate2, minor: minor2 } = calculateVersionsFromString(minimumVersion);
    // Return false if major version is less than minimum
    if (major1 < major2) return false;
    // Return false if major version is equal to minimum and moderate version is less than minimum
    if (major1 === major2 && moderate1 < moderate2) return false;
    // Return false if major and moderate versions are equal to minimum and minor version is less than minimum
    if (major1 === major2 && moderate1 === moderate2 && minor1 < minor2) return false;
    // Otherwise, return true
    return true;
};

/**
 * Test for minimum version
 */
export const minVersionTest = (minVersion: string): [string, string, (value: string | undefined) => boolean] => {
    const versionRegex = /^\d+\.\d+\.\d+$/;
    return [
        "version",
        `Minimum version is ${minVersion}`,
        (value: string | undefined) => {
            if (!value) return true;
            return versionRegex.test(value) && meetsMinVersion(value, minVersion);
        },
    ];
};

/**
 * Sorts versions from lowest to highest
 */
export const sortVersions = <T extends { versionLabel: string }>(versions: T[]): T[] => {
    if (!Array.isArray(versions)) return [];
    return versions.sort((a, b) => {
        const { major: majorA, moderate: moderateA, minor: minorA } = calculateVersionsFromString(a.versionLabel);
        const { major: majorB, moderate: moderateB, minor: minorB } = calculateVersionsFromString(b.versionLabel);
        if (majorA > majorB) return 1;
        if (majorA < majorB) return -1;
        if (moderateA > moderateB) return 1;
        if (moderateA < moderateB) return -1;
        if (minorA > minorB) return 1;
        if (minorA < minorB) return -1;
        return 0;
    });
};
