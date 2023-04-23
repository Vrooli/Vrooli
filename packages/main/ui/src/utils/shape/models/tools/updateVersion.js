export const updateVersion = (originalRoot, updatedRoot, shape, preShape) => {
    if (!updatedRoot.versionInfo)
        return {};
    const preShaper = preShape ?? ((x) => x);
    const isCreate = !originalRoot.versionInfo || originalRoot.versionInfo.id !== updatedRoot.versionInfo.id;
    if (isCreate) {
        return {
            versionsCreate: [shape.create(preShaper({
                    ...updatedRoot.versionInfo,
                    versionLabel: updatedRoot.versionInfo.versionLabel ?? "0.0.1",
                }, updatedRoot))],
        };
    }
    else {
        return {
            versionsUpdate: [shape.update(preShaper(originalRoot.versionInfo, originalRoot), preShaper(updatedRoot.versionInfo, updatedRoot))],
        };
    }
};
//# sourceMappingURL=updateVersion.js.map