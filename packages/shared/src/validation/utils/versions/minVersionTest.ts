import { meetsMinVersion } from "./meetsMinVersion";

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
