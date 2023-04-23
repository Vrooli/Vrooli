import { meetsMinVersion } from "@local/validation";
export const getMinimumVersion = (versions) => {
    if (versions.length === 0)
        return "0.0.1";
    versions.sort((a, b) => meetsMinVersion(a, b) ? 1 : -1);
    const highestVersion = versions[versions.length - 1];
    return highestVersion;
};
//# sourceMappingURL=getMinimumVersion.js.map