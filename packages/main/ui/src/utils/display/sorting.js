import i18next from "i18next";
export function labelledSortOptions(sortValues) {
    if (!sortValues)
        return [];
    return Object.keys(sortValues).map((key) => ({
        label: (i18next.t(key, key)),
        value: key,
    }));
}
//# sourceMappingURL=sorting.js.map