import { calculateVersionsFromString } from "./calculateVersionsFromString";
export const meetsMinVersion = (version, minimumVersion) => {
    const { major: major1, moderate: moderate1, minor: minor1 } = calculateVersionsFromString(version);
    const { major: major2, moderate: moderate2, minor: minor2 } = calculateVersionsFromString(minimumVersion);
    if (major1 < major2)
        return false;
    if (major1 === major2 && moderate1 < moderate2)
        return false;
    if (major1 === major2 && moderate1 === moderate2 && minor1 < minor2)
        return false;
    return true;
};
//# sourceMappingURL=meetsMinVersion.js.map