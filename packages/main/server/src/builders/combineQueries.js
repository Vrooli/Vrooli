export function combineQueries(queries) {
    const combined = {};
    for (const query of queries) {
        if (!query)
            continue;
        for (const [key, value] of Object.entries(query)) {
            let currValue = value;
            if (["AND", "OR", "NOT"].includes(key)) {
                if (!Array.isArray(value)) {
                    currValue = [value];
                }
                if (key === "AND") {
                    combined[key] = key in combined ? [...combined[key], ...currValue] : currValue;
                }
                else if (!(key in combined)) {
                    combined[key] = currValue;
                }
                else {
                    const temp = combined[key];
                    delete combined[key];
                    combined.AND = [
                        ...(combined.AND || []),
                        { [key]: temp },
                        { [key]: currValue },
                    ];
                }
            }
            else
                combined[key] = value;
        }
    }
    return combined;
}
//# sourceMappingURL=combineQueries.js.map