import { meetsMinVersion } from "@shared/validation";
import { VersionInfo } from "types";

/**
* Determines mimimum version. This is either: 
* 1. The highest version in the list of versions
* 2. A version larger than the highest version in the list of versions
* 3. Any version number greater than 0.0.0, if there are no versions in the list
* @param versions The list of versions
* @returns The minimum version
*/
export const getMinimumVersion = (versions: VersionInfo[]): string => {
    // If no versions, return 0.0.1
    if (versions.length === 0) return '0.0.1';
    // Get all version labels
    const versionLabels = versions.map(v => v.versionLabel);
    // Sort using meetsMinVersion, which determines if a version is greater than or equal to another version
    versionLabels.sort((a, b) => meetsMinVersion(a, b) ? 1 : -1);
    // Get the highest version
    const highestVersion = versionLabels[versionLabels.length - 1];
    return highestVersion;
} 