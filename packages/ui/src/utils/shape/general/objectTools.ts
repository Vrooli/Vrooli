/**
 * Functions for manipulating state objects
 */
import { exists } from "@local/shared";

/**
 * Grabs data from an object using dot notation (ex: 'parent.child.property')
 */
export const valueFromDot = (object: any, notation: string): any => {
    // Utility function to index into object using string key
    function index(obj: any, i: string) {
        // Return null if obj is falsy
        return exists(obj) ? obj[i] : null;
    }
    // Return null if either object or notation is falsy
    if (!object || !notation) return null;
    // Use reduce method to traverse the object using dot notation
    // If the final result is falsy, return null
    const value = notation.split(".").reduce(index, object);
    return exists(value) ? value : null;
};

/**
 * Maps the keys of an object to dot notation
 */
export const convertToDot = (obj: Record<string, any>, parent = [], keyValue = {}) => {
    for (const key in obj) {
        const keyPath: any = [...parent, key];
        if (obj[key] !== null && typeof obj[key] === "object") {
            Object.assign(keyValue, convertToDot(obj[key], keyPath, keyValue));
        } else {
            keyValue[keyPath.join(".")] = obj[key];
        }
    }
    return keyValue;
}

/**
 * Checks if any of the specified fields (supports dot notation) in an object have been changed. 
 * If no fields are specified, checks if any fields have been changed.
 * @param original The original object
 * @param updated The updated object
 * @param fields The fields to check for changes, or empty array to check all fields
 */
export const hasObjectChanged = (original: unknown, updated: unknown, fields: string[] = []): boolean => {
    if (updated === null || updated === undefined) return false;
    if (original === null || original === undefined) return true;

    const isObject = (obj: unknown) => obj && typeof obj === "object" && !Array.isArray(obj);

    // Direct comparison for non-objects
    if (!isObject(original) || !isObject(updated)) {
        return original !== updated;
    }

    const checkField = (original: any, updated: any, field: string): boolean => {
        const [topLevelField, ...rest] = field.split(".");

        if (rest.length > 0) {
            if (isObject(original[topLevelField]) && isObject(updated[topLevelField])) {
                return hasObjectChanged(original[topLevelField], updated[topLevelField], [rest.join(".")]);
            }
            return false;
        } else {
            if (Array.isArray(original[topLevelField]) && Array.isArray(updated[topLevelField])) {
                if (original[topLevelField].length !== updated[topLevelField].length) return true;
                for (let i = 0; i < original[topLevelField].length; i++) {
                    if (hasObjectChanged(original[topLevelField][i], updated[topLevelField][i])) return true;
                }
                return false;
            } else if (isObject(original[topLevelField]) && isObject(updated[topLevelField])) {
                return hasObjectChanged(original[topLevelField], updated[topLevelField]);
            }
            return original[topLevelField] !== updated[topLevelField];
        }
    };

    // Combine keys from both objects
    const allKeys = new Set([...Object.keys(original), ...Object.keys(updated)]);

    // Check specified fields, or all fields if none specified
    const fieldsToCheck = fields.length > 0 ? fields : Array.from(allKeys);

    for (const field of fieldsToCheck) {
        if (checkField(original, updated, field)) {
            return true;
        }
    }

    return false;
};
