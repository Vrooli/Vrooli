import { meetsMinVersion } from "./meetsMinVersion";

/**
 * Test for minimum version
 */
export const minVersionTest = (minVersion: string): [string, string, (value: string | undefined) => boolean] => {
    return [
        'version',
        `Minimum version is ${minVersion}`,
        (value: string | undefined) => {
            if (!value) return true;
            return meetsMinVersion(value, minVersion);
        }
    ]
}