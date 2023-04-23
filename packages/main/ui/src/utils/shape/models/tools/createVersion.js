export const createVersion = (root, shape, preShape) => {
    if (!root.versionInfo)
        return {};
    const preShaper = preShape ?? ((x) => x);
    return {
        versionsCreate: [shape.create(preShaper({
                ...root.versionInfo,
                versionLabel: root.versionInfo.versionLabel ?? "0.0.1",
            }))],
    };
};
//# sourceMappingURL=createVersion.js.map