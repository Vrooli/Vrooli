/**
 * Returns the first non-null value from the list of 
 * parameters, or undefined if none are non-null and 
 * non-undefined
 * @param args - List of arguments
 * @returns First non-null value
 */
export const noNull = <T>(...args: (T | undefined | null)[]): T | undefined => {
    for (const arg of args) {
        if (arg !== undefined && arg !== null) {
            return arg;
        }
    }
    return undefined;
}