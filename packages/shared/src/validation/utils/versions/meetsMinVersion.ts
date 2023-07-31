import { calculateVersionsFromString } from "./calculateVersionsFromString";

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
