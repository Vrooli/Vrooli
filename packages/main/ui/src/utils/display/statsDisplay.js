export const statsDisplay = (stats) => {
    const aggregate = {};
    const visual = {};
    for (const stat of stats) {
        for (const key of Object.keys(stat)) {
            if (["__typename", "id", "periodStart", "periodEnd", "periodType"].includes(key))
                continue;
            if (typeof stat[key] !== "number")
                continue;
            if (key.startsWith("active")) {
                if (aggregate[key] === undefined || stat[key] > aggregate[key]) {
                    aggregate[key] = stat[key];
                }
                visual[key] = [...(visual[key] ?? []), stat[key]];
            }
            else {
                aggregate[key] = (aggregate[key] ?? 0) + stat[key];
                visual[key] = [...(visual[key] ?? []), stat[key]];
            }
        }
    }
    for (const key of Object.keys(aggregate)) {
        if (key.endsWith("Average")) {
            aggregate[key] = aggregate[key] / stats.length;
        }
    }
    return { aggregate, visual };
};
//# sourceMappingURL=statsDisplay.js.map