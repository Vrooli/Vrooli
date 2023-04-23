export const rootObjectDisplay = (versionModelLogic) => ({
    select: () => ({
        id: true,
        versions: {
            orderBy: { versionIndex: "desc" },
            take: 10,
            select: {
                ...versionModelLogic.display.select(),
                isLatest: true,
                isPrivate: true,
                versionIndex: true,
            },
        },
    }),
    label: (select, languages) => {
        if (select.versions.length === 0)
            return "";
        let latest = select.versions.find(v => v.isLatest);
        if (!latest) {
            latest = select.versions.find(v => !v.isPrivate);
            if (!latest)
                latest = select.versions[0];
        }
        return versionModelLogic.display.label(latest, languages);
    },
});
//# sourceMappingURL=rootObjectDisplay.js.map