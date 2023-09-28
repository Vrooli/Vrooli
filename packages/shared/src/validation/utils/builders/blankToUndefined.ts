/**
 * Converts empty/whitespace strings to undefined
 */
export const blankToUndefined = (value: string | undefined) => {
    console.log("in blankToUndefined", typeof value);
    if (typeof value !== "string") return undefined;
    const trimmed = value.trim();
    if (trimmed === "") return undefined;
    return trimmed;
};
