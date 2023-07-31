/**
 * Lowercases the first letter of a string
 * @param str String to lowercase
 * @returns Lowercased string
 */
export const lowercaseFirstLetter = (str: string): string => {
    return str.charAt(0).toLowerCase() + str.slice(1);
};

/**
 * Uppercases the first letter of a string
 * @param str String to capitalize
 * @returns Uppercased string
 */
export const uppercaseFirstLetter = (str: string): string => {
    return str.charAt(0).toUpperCase() + str.slice(1);
};
