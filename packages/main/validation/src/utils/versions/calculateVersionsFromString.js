export const calculateVersionsFromString = (version) => {
    const [major, moderate, minor] = version.split(".").map(v => parseInt(v));
    return {
        major: major || 0,
        moderate: moderate || 0,
        minor: minor || 0,
    };
};
//# sourceMappingURL=calculateVersionsFromString.js.map