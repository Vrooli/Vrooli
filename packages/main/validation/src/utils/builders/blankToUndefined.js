export const blankToUndefined = (value) => {
    if (!value)
        return undefined;
    const trimmed = value.trim();
    if (trimmed === "")
        return undefined;
    return trimmed;
};
//# sourceMappingURL=blankToUndefined.js.map