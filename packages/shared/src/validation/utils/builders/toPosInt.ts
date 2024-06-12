/**
 * Strips non-numeric characters from a string, leaving a positive integer
 */
export const toPosInt = (str: string) => {
    return parseInt(str.replace(/[^0-9]/g, ""), 10);
};