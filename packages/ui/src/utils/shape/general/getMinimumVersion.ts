import { meetsMinVersion } from "@local/shared";

/**
* Determines mimimum version. This is either: 
* 1. The highest version in the list of versions
* 2. A version larger than the highest version in the list of versions
* 3. Any version number greater than 0.0.0, if there are no versions in the list
* @param versions The list of versions
* @returns The minimum version
*/
export const getMinimumVersion = (versions: string[]): string => {
    // If no versions, return 0.0.1
    if (versions.length === 0) return '0.0.1';
    // Sort using meetsMinVersion, which determines if a version is greater than or equal to another version
    versions.sort((a, b) => meetsMinVersion(a, b) ? 1 : -1);
    // Get the highest version
    const highestVersion = versions[versions.length - 1];
    return highestVersion;
} 