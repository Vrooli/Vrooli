/**
 * Strips non-numeric characters from a string, leaving a positive double
 */
export const toPosDouble = (str: string) => {
    return parseFloat(str.replace(/[^0-9.]/g, ""));
}