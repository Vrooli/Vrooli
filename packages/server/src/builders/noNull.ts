import { exists } from "@local/shared";

/**
 * Returns the first non-null value from the list of 
 * parameters, or undefined if none are non-null and 
 * non-undefined
 * @param args - List of arguments
 * @returns First non-null value
 */
export const noNull = <T>(...args: (T | undefined | null)[]): T | undefined => {
    for (const arg of args) {
        if (exists(arg)) {
            return arg;
        }
    }
    return undefined;
};

/**
 * Returns the first non-empty string from the list of parameters,
 * or undefined if none are non-empty and non-undefined
 * @param args - List of string arguments
 * @returns First non-empty string value
 */
export const noEmptyString = (...args: unknown[]): string | undefined => {
    for (const arg of args) {
        // Check if arg is a string, not null, not undefined, and not an empty string
        if (typeof arg === "string" && arg !== "") {
            return arg;
        }
    }
    return undefined;
};


/**
 * Returns the first valid number from the list of parameters,
 * or undefined if none are valid numbers.
 * @param args - List of numeric arguments
 * @returns First valid number value
 */
export const validNumber = (...args: unknown[]): number | undefined => {
    for (const arg of args) {
        // Check if arg is a finite number
        if (typeof arg === "number" && isFinite(arg)) {
            return arg;
        }
    }
    return undefined;
};
