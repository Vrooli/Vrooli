import { meetsMinVersion } from "./meetsMinVersion";
export const minVersionTest = (minVersion) => {
    return [
        "version",
        `Minimum version is ${minVersion}`,
        (value) => {
            if (!value)
                return true;
            return meetsMinVersion(value, minVersion);
        },
    ];
};
//# sourceMappingURL=minVersionTest.js.map