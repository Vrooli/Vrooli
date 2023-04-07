/**
 * Uppercases the first letter of a string
 * @param str String to capitalize
 * @returns Uppercased string
 */
export const uppercaseFirstLetter = (str: string): string => {
    return str.charAt(0).toUpperCase() + str.slice(1);
}