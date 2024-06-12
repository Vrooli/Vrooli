import { CustomError } from "../events/error";
import { logger } from "../events/logger";

/**
 * Recursively sorts keys of an object alphabetically. 
 * This can be used to simplify object comparison
 * @parm obj The object to sort
 * @returns The sorted object
 */
export const sortObjectKeys = (obj: unknown): any => {
    if (typeof obj !== "object" || obj === null || obj instanceof Date || Array.isArray(obj)) {
        return obj;
    }
    const sorted: { [x: string]: any } = {};
    Object.keys(obj).sort().forEach(key => {
        sorted[key] = sortObjectKeys(obj[key]);
    });
    return sorted;
};

/**
 * Parses JSON string, recursively sorts all keys alphabetically, and restringifies
 * @param stringified The stringified JSON object
 * @param languages Preferred languages for error messages
 * @returns The sortified, stringified object
 */
export const sortify = (stringified: string, languages: string[]): string => {
    try {
        const obj = JSON.parse(stringified);
        return JSON.stringify(sortObjectKeys(obj));
    } catch (error) {
        throw new CustomError("0210", "InvalidArgs", languages);
    }
};

/**
 * Parses a JSON string and returns the result or a default value if parsing fails.
 * @param jsonStr The JSON string to parse
 * @param defaultValue The default value to return if parsing fails
 * @returns The parsed object or the default value
 */
export const parseJsonOrDefault = <T>(jsonStr: string | null, defaultValue: T): T => {
    try {
        return jsonStr ? JSON.parse(jsonStr) : defaultValue;
    } catch (error) {
        logger.error("Failed to parse JSON", { trace: "0431" });
        return defaultValue;
    }
};
