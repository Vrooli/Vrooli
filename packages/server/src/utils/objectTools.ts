// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-06-17
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";

/**
 * Recursively sorts keys of an object alphabetically. 
 * This can be used to simplify object comparison
 * @parm obj The object to sort
 * @returns The sorted object
 */
export function sortObjectKeys<T>(obj: T): T {
    if (typeof obj !== "object" || obj === null || obj instanceof Date || Array.isArray(obj)) {
        return obj;
    }
    const sorted: Record<string, unknown> = {};
    Object.keys(obj as object).sort().forEach(key => {
        sorted[key] = sortObjectKeys((obj as Record<string, unknown>)[key]);
    });
    return sorted as T;
}

/**
 * Parses JSON string, recursively sorts all keys alphabetically, and restringifies
 * @param stringified The stringified JSON object
 * @returns The sortified, stringified object
 */
export function sortify(stringified: string): string {
    try {
        const obj = JSON.parse(stringified);
        return JSON.stringify(sortObjectKeys(obj));
    } catch (error) {
        throw new CustomError("0210", "InvalidArgs");
    }
}

/**
 * Parses a JSON string and returns the result or a default value if parsing fails.
 * @param jsonStr The JSON string to parse
 * @param defaultValue The default value to return if parsing fails
 * @returns The parsed object or the default value
 */
export function parseJsonOrDefault<T>(jsonStr: string | null, defaultValue: T): T {
    try {
        return jsonStr ? JSON.parse(jsonStr) : defaultValue;
    } catch (error) {
        logger.error("Failed to parse JSON", { trace: "0431" });
        return defaultValue;
    }
}
