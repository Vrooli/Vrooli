/**
 * Converts empty/whitespace strings to undefined
 */
export const blankToUndefined = (value: string | undefined) => {
    if (!value) return undefined;
    const trimmed = value.trim();
    if (trimmed === '') return undefined;
    return trimmed;
}